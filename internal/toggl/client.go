package toggl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.track.toggl.com/api/v9"
	createdWith    = "lazytoggl"
)

// ErrRateLimited is returned when Toggl responds 402, which on the Free plan
// means the 30-requests/hour quota is exhausted (see docs/ARCHITECTURE.md).
var ErrRateLimited = errors.New("toggl: rate limited — try again later")

// ErrInvalidToken is returned on 401: the API token is missing or wrong.
var ErrInvalidToken = errors.New("toggl: invalid API token")

// ErrForbidden is returned on 403. Toggl uses it both for a rejected login
// and for an action the user may not take (e.g. creating a project in a
// workspace where only admins may), so only onboarding reads it as a bad
// token.
var ErrForbidden = errors.New("toggl: request refused (403)")

// APIError wraps any other non-2xx response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("toggl: request failed (%d): %s", e.StatusCode, e.Body)
}

// Client is the real Toggl API v9 implementation of Service. It authenticates
// with HTTP Basic auth using the API token as the username and the literal
// string "api_token" as the password — Toggl's documented convention.
type Client struct {
	apiToken     string
	workspaceID  int64
	httpClient   *http.Client
	baseURL      string
	retryBackoff time.Duration // first retry delay; doubles per attempt
}

func NewClient(apiToken string) *Client {
	return &Client{
		apiToken:     apiToken,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		baseURL:      defaultBaseURL,
		retryBackoff: 500 * time.Millisecond,
	}
}

// SetWorkspaceID restores a workspace id cached from a previous session
// (see docs/ARCHITECTURE.md), letting normal launches skip the /me call
// that Me would otherwise need to discover it.
func (c *Client) SetWorkspaceID(id int64) { c.workspaceID = id }

func (c *Client) WorkspaceID() int64 { return c.workspaceID }

var _ Service = (*Client)(nil)

func (c *Client) Me(ctx context.Context) (Me, error) {
	raw, err := c.do(ctx, http.MethodGet, "/me", nil)
	if err != nil {
		return Me{}, err
	}
	var resp struct {
		ID                 int64  `json:"id"`
		Fullname           string `json:"fullname"`
		DefaultWorkspaceID int64  `json:"default_workspace_id"`
		DefaultWID         int64  `json:"default_wid"` // legacy alias, kept as a fallback
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return Me{}, fmt.Errorf("toggl: decoding /me: %w", err)
	}
	wid := resp.DefaultWorkspaceID
	if wid == 0 {
		wid = resp.DefaultWID
	}
	c.workspaceID = wid
	return Me{ID: resp.ID, Fullname: resp.Fullname, DefaultWorkspaceID: wid}, nil
}

func (c *Client) RecentEntries(ctx context.Context) ([]TimeEntry, error) {
	now := time.Now()
	q := url.Values{}
	// The API rejects start_date without end_date (and vice versa) with a
	// 400 — they're required together, confirmed live against the real API.
	q.Set("start_date", RecentEntriesSince(now).Format(time.RFC3339))
	q.Set("end_date", now.Format(time.RFC3339))
	raw, err := c.do(ctx, http.MethodGet, "/me/time_entries?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	wires, err := decodeTimeEntryList(raw)
	if err != nil {
		return nil, err
	}
	entries := make([]TimeEntry, 0, len(wires))
	for _, w := range wires {
		e, err := w.toEntry()
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Start.After(entries[j].Start) })
	return entries, nil
}

type projectWire struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
	Active bool   `json:"active"`
}

func (w projectWire) toProject() Project {
	return Project{ID: w.ID, Name: w.Name, Color: w.Color, Active: w.Active}
}

// projectsPageSize is the page size of GET .../projects (X-Page-Size,
// confirmed live): a full page means there may be another.
const projectsPageSize = 151

func (c *Client) Projects(ctx context.Context) ([]Project, error) {
	base := fmt.Sprintf("/workspaces/%d/projects", c.workspaceID)
	// Deliberately not filtered by "active": a past entry may reference an
	// archived project, and dropping it here would make projectLabel show
	// "—" instead of the real name for that history.
	var projects []Project
	for page := 1; ; page++ {
		path := base
		if page > 1 {
			path = fmt.Sprintf("%s?page=%d", base, page)
		}
		raw, err := c.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		var wires []projectWire
		if err := json.Unmarshal(raw, &wires); err != nil {
			return nil, fmt.Errorf("toggl: decoding projects: %w", err)
		}
		for _, w := range wires {
			projects = append(projects, w.toProject())
		}
		// Each further page costs a request, so only a full page asks.
		if len(wires) < projectsPageSize {
			return projects, nil
		}
	}
}

func (c *Client) CreateProject(ctx context.Context, name string) (Project, error) {
	path := fmt.Sprintf("/workspaces/%d/projects", c.workspaceID)
	raw, err := c.do(ctx, http.MethodPost, path, map[string]any{"name": name, "active": true})
	if err != nil {
		return Project{}, err
	}
	var w projectWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return Project{}, fmt.Errorf("toggl: decoding created project: %w", err)
	}
	return w.toProject(), nil
}

func (c *Client) Start(ctx context.Context, req StartRequest) (TimeEntry, error) {
	body := map[string]any{
		"workspace_id": c.workspaceID,
		"description":  req.Description,
		"start":        time.Now().Format(time.RFC3339),
		"duration":     -1,
		"created_with": createdWith,
	}
	if req.ProjectID != 0 {
		body["project_id"] = req.ProjectID
	}
	path := fmt.Sprintf("/workspaces/%d/time_entries", c.workspaceID)
	raw, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return TimeEntry{}, err
	}
	return decodeTimeEntry(raw)
}

func (c *Client) Stop(ctx context.Context, entryID int64) (TimeEntry, error) {
	path := fmt.Sprintf("/workspaces/%d/time_entries/%d/stop", c.workspaceID, entryID)
	raw, err := c.do(ctx, http.MethodPatch, path, nil)
	if err != nil {
		return TimeEntry{}, err
	}
	return decodeTimeEntry(raw)
}

func (c *Client) Update(ctx context.Context, entry TimeEntry) (TimeEntry, error) {
	body := map[string]any{
		"workspace_id": c.workspaceID,
		"description":  entry.Description,
		"start":        entry.Start.Format(time.RFC3339),
	}
	// Always sent: a PUT that leaves project_id out keeps the old project,
	// and null is what removes it (confirmed live, docs/ARCHITECTURE.md).
	if entry.ProjectID != 0 {
		body["project_id"] = entry.ProjectID
	} else {
		body["project_id"] = nil
	}
	if entry.Running() {
		body["duration"] = -1
	} else {
		body["stop"] = entry.Stop.Format(time.RFC3339)
		body["duration"] = int64(entry.Stop.Sub(entry.Start).Seconds())
	}
	path := fmt.Sprintf("/workspaces/%d/time_entries/%d", c.workspaceID, entry.ID)
	raw, err := c.do(ctx, http.MethodPut, path, body)
	if err != nil {
		return TimeEntry{}, err
	}
	return decodeTimeEntry(raw)
}

func (c *Client) Delete(ctx context.Context, entryID int64) error {
	path := fmt.Sprintf("/workspaces/%d/time_entries/%d", c.workspaceID, entryID)
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// Retry policy: transient failures only (network errors, 5xx, 408/429) are
// retried, with exponential backoff and a hard attempt cap so retries can
// never run indefinitely. Deterministic failures (400s) are never retried,
// and neither is 402 rate limiting — automatic retries against an exhausted
// hourly quota would just burn more of it.
const maxAttempts = 3

// do sends a request and returns the raw response body on 2xx, or a typed
// error (ErrRateLimited, ErrInvalidToken, ErrForbidden, *APIError) otherwise.
func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("toggl: encoding request: %w", err)
		}
		payload = b
	}

	backoff := c.retryBackoff
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
		respBody, retryable, err := c.attempt(ctx, method, path, payload, body != nil)
		if err == nil {
			return respBody, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
	}
	return nil, lastErr
}

// attempt performs one HTTP round trip. retryable reports whether the
// failure is transient and worth another attempt.
func (c *Client) attempt(ctx context.Context, method, path string, payload []byte, hasBody bool) (respBody []byte, retryable bool, err error) {
	var reqBody io.Reader
	if hasBody {
		reqBody = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, false, fmt.Errorf("toggl: building request: %w", err)
	}
	req.SetBasicAuth(c.apiToken, "api_token")
	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("toggl: request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("toggl: reading response: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusCreated ||
		resp.StatusCode == http.StatusNoContent:
		return respBody, false, nil
	case resp.StatusCode == http.StatusPaymentRequired:
		return nil, false, ErrRateLimited
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, false, ErrInvalidToken
	case resp.StatusCode == http.StatusForbidden:
		return nil, false, ErrForbidden
	case resp.StatusCode == http.StatusRequestTimeout ||
		resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode >= http.StatusInternalServerError:
		return nil, true, &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody))}
	default:
		return nil, false, &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody))}
	}
}

// timeEntryWire mirrors the v9 time entry JSON shape. A running entry is
// identified by a null/absent "stop" — the exact negative "duration"
// convention for running entries is undocumented and not relied on.
type timeEntryWire struct {
	ID          int64   `json:"id"`
	WorkspaceID int64   `json:"workspace_id"`
	ProjectID   int64   `json:"project_id"`
	Description string  `json:"description"`
	Start       string  `json:"start"`
	Stop        *string `json:"stop"`
	Duration    int64   `json:"duration"`
}

func (w timeEntryWire) toEntry() (TimeEntry, error) {
	start, err := time.Parse(time.RFC3339, w.Start)
	if err != nil {
		return TimeEntry{}, fmt.Errorf("toggl: parsing start time %q: %w", w.Start, err)
	}
	e := TimeEntry{
		ID: w.ID, WorkspaceID: w.WorkspaceID, ProjectID: w.ProjectID,
		Description: w.Description, Start: start,
	}
	if w.Stop == nil || *w.Stop == "" {
		e.Duration = -1
		return e, nil
	}
	stop, err := time.Parse(time.RFC3339, *w.Stop)
	if err != nil {
		return TimeEntry{}, fmt.Errorf("toggl: parsing stop time %q: %w", *w.Stop, err)
	}
	e.Stop = stop
	e.Duration = w.Duration
	return e, nil
}

func decodeTimeEntry(raw []byte) (TimeEntry, error) {
	var w timeEntryWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return TimeEntry{}, fmt.Errorf("toggl: decoding time entry: %w", err)
	}
	return w.toEntry()
}

// decodeTimeEntryList tolerates either a bare JSON array or an
// {"items": [...]} envelope — the two shapes seen across Toggl's own
// documentation for this endpoint — since it could not be pinned down with
// certainty ahead of integration (see docs/ARCHITECTURE.md).
func decodeTimeEntryList(raw []byte) ([]timeEntryWire, error) {
	var arr []timeEntryWire
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var wrapped struct {
		Items []timeEntryWire `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("toggl: decoding time entries list: %w", err)
	}
	return wrapped.Items, nil
}
