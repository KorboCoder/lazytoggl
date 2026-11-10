package toggl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient points a Client at an httptest server and captures every
// request the client makes for inspection.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *[]*http.Request) {
	t.Helper()
	var requests []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("test-token")
	c.baseURL = srv.URL
	c.workspaceID = 555
	c.retryBackoff = time.Millisecond // keep retry tests fast
	return c, &requests
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func TestMeUsesBasicAuthAndCachesWorkspaceID(t *testing.T) {
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/me" {
			t.Errorf("request = %s %s, want GET /me", r.Method, r.URL.Path)
		}
		writeJSON(w, 200, map[string]any{
			"id": 42, "fullname": "Ada Lovelace", "default_workspace_id": 999,
		})
	})
	c.workspaceID = 0 // Me is what's supposed to discover it

	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.ID != 42 || me.Fullname != "Ada Lovelace" || me.DefaultWorkspaceID != 999 {
		t.Fatalf("Me() = %+v, want id 42 / Ada Lovelace / workspace 999", me)
	}
	if c.WorkspaceID() != 999 {
		t.Fatalf("client workspaceID = %d, want 999 (cached from Me)", c.WorkspaceID())
	}

	req := (*requests)[0]
	user, pass, ok := req.BasicAuth()
	if !ok || user != "test-token" || pass != "api_token" {
		t.Fatalf("basic auth = %q:%q (ok=%v), want token as username, literal 'api_token' as password", user, pass, ok)
	}
}

func TestMeFallsBackToLegacyDefaultWID(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"id": 1, "default_wid": 777})
	})
	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.DefaultWorkspaceID != 777 {
		t.Fatalf("DefaultWorkspaceID = %d, want 777 from legacy default_wid", me.DefaultWorkspaceID)
	}
}

func TestStartSendsExpectedBody(t *testing.T) {
	var body map[string]any
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{
			"id": 1, "workspace_id": 555, "project_id": 101,
			"description": "Writing docs", "start": "2026-07-15T10:00:00Z",
			"stop": nil, "duration": -1,
		})
	})

	e, err := c.Start(context.Background(), StartRequest{Description: "Writing docs", ProjectID: 101})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !e.Running() || e.Description != "Writing docs" || e.ProjectID != 101 {
		t.Fatalf("Start() entry = %+v, want a running 'Writing docs' entry with project 101", e)
	}

	req := (*requests)[0]
	if req.Method != http.MethodPost || req.URL.Path != "/workspaces/555/time_entries" {
		t.Fatalf("request = %s %s, want POST /workspaces/555/time_entries", req.Method, req.URL.Path)
	}
	if body["description"] != "Writing docs" || body["created_with"] != "lazytoggl" {
		t.Fatalf("request body = %+v, missing description/created_with", body)
	}
	if wid, ok := body["workspace_id"].(float64); !ok || int64(wid) != 555 {
		t.Fatalf("request body workspace_id = %v, want 555", body["workspace_id"])
	}
	if pid, ok := body["project_id"].(float64); !ok || pid != 101 {
		t.Fatalf("project_id = %v, want 101", body["project_id"])
	}
}

// A new entry with no project leaves project_id out entirely.
func TestStartOmitsProjectWhenNone(t *testing.T) {
	var body map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{"id": 1, "workspace_id": 555, "description": "x",
			"start": "2026-07-15T10:00:00Z", "stop": nil, "duration": -1})
	})
	if _, err := c.Start(context.Background(), StartRequest{Description: "x"}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, has := body["project_id"]; has {
		t.Fatalf("project_id sent for an entry with no project: %v", body)
	}
}

func TestStopUsesPatchOnStopPath(t *testing.T) {
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"id": 7, "workspace_id": 555, "description": "Standup",
			"start": "2026-07-15T09:00:00Z", "stop": "2026-07-15T09:15:00Z", "duration": 900,
		})
	})
	e, err := c.Stop(context.Background(), 7)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if e.Running() || e.Duration != 900 {
		t.Fatalf("Stop() entry = %+v, want a finished 900s entry", e)
	}
	req := (*requests)[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/workspaces/555/time_entries/7/stop" {
		t.Fatalf("request = %s %s, want PATCH /workspaces/555/time_entries/7/stop", req.Method, req.URL.Path)
	}
}

func TestUpdateSendsStopOnlyWhenFinished(t *testing.T) {
	var body map[string]any
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{
			"id": 3, "workspace_id": 555, "description": "Edited",
			"start": "2026-07-15T09:00:00Z", "stop": "2026-07-15T10:00:00Z", "duration": 3600,
		})
	})
	start := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	stop := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

	_, err := c.Update(context.Background(), TimeEntry{ID: 3, Description: "Edited", Start: start, Stop: stop, Duration: 3600})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if body["stop"] == nil {
		t.Fatal("finished entry update must send a stop time")
	}
	if _, hasDuration := body["duration"]; !hasDuration {
		t.Fatal("update body missing duration")
	}
	// Always sent: a PUT that omits project_id keeps the old project.
	if v, has := body["project_id"]; !has || v != nil {
		t.Fatalf("project_id = %v (present %v), want an explicit null for no project", v, has)
	}

	req := (*requests)[0]
	if req.Method != http.MethodPut || req.URL.Path != "/workspaces/555/time_entries/3" {
		t.Fatalf("request = %s %s, want PUT /workspaces/555/time_entries/3", req.Method, req.URL.Path)
	}
}

// Confirmed live: "project_id": null removes the project (docs/ARCHITECTURE.md).
func TestUpdateClearsProjectWithNull(t *testing.T) {
	var raw map[string]json.RawMessage
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&raw)
		writeJSON(w, 200, map[string]any{
			"id": 3, "workspace_id": 555, "project_id": nil, "description": "x",
			"start": "2026-07-15T09:00:00Z", "stop": "2026-07-15T10:00:00Z", "duration": 3600,
		})
	})
	start := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	e, err := c.Update(context.Background(), TimeEntry{ID: 3, Description: "x", Start: start, Stop: start.Add(time.Hour)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if string(raw["project_id"]) != "null" {
		t.Fatalf("project_id sent as %q, want null", raw["project_id"])
	}
	if e.ProjectID != 0 {
		t.Fatalf("decoded ProjectID = %d, want 0", e.ProjectID)
	}
}

func TestUpdateSendsProjectID(t *testing.T) {
	var body map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{
			"id": 3, "workspace_id": 555, "project_id": 101, "description": "x",
			"start": "2026-07-15T09:00:00Z", "stop": nil, "duration": -1,
		})
	})
	_, err := c.Update(context.Background(), TimeEntry{ID: 3, ProjectID: 101, Description: "x",
		Start: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC), Duration: -1})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if pid, ok := body["project_id"].(float64); !ok || pid != 101 {
		t.Fatalf("project_id = %v, want 101", body["project_id"])
	}
}

func TestUpdateOmitsStopWhenStillRunning(t *testing.T) {
	var body map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{
			"id": 3, "workspace_id": 555, "description": "Still going",
			"start": "2026-07-15T09:00:00Z", "stop": nil, "duration": -1,
		})
	})
	_, err := c.Update(context.Background(), TimeEntry{
		ID: 3, Description: "Still going", Start: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC), Duration: -1,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, present := body["stop"]; present {
		t.Fatalf("running entry update must not send a stop field, got body = %+v", body)
	}
}

func TestDeleteUsesDeleteMethod(t *testing.T) {
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := c.Delete(context.Background(), 9); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	req := (*requests)[0]
	if req.Method != http.MethodDelete || req.URL.Path != "/workspaces/555/time_entries/9" {
		t.Fatalf("request = %s %s, want DELETE /workspaces/555/time_entries/9", req.Method, req.URL.Path)
	}
}

func TestRecentEntriesSortsNewestFirstAndSendsLookbackWindow(t *testing.T) {
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, []map[string]any{
			{"id": 1, "description": "older", "start": "2026-07-14T09:00:00Z", "stop": "2026-07-14T10:00:00Z", "duration": 3600},
			{"id": 2, "description": "newer", "start": "2026-07-15T09:00:00Z", "stop": nil, "duration": -1},
		})
	})
	entries, err := c.RecentEntries(context.Background())
	if err != nil {
		t.Fatalf("RecentEntries: %v", err)
	}
	if len(entries) != 2 || entries[0].Description != "newer" || entries[1].Description != "older" {
		t.Fatalf("entries = %+v, want newest-first [newer, older]", entries)
	}

	req := (*requests)[0]
	if req.URL.Path != "/me/time_entries" {
		t.Fatalf("path = %q, want /me/time_entries", req.URL.Path)
	}
	// The API 400s if either is sent without the other — confirmed live.
	if req.URL.Query().Get("start_date") == "" {
		t.Fatal("request missing start_date lookback bound")
	}
	if req.URL.Query().Get("end_date") == "" {
		t.Fatal("request missing end_date — the API requires it alongside start_date")
	}
	// The window is exactly the exported lookback, so the UI's idea of
	// "outside the list" (RecentEntriesSince) matches what refresh returns.
	start, err := time.Parse(time.RFC3339, req.URL.Query().Get("start_date"))
	if err != nil {
		t.Fatalf("start_date: %v", err)
	}
	end, err := time.Parse(time.RFC3339, req.URL.Query().Get("end_date"))
	if err != nil {
		t.Fatalf("end_date: %v", err)
	}
	if RecentEntriesLookback != 8*24*time.Hour {
		t.Fatalf("RecentEntriesLookback = %s, want 192h", RecentEntriesLookback)
	}
	if !RecentEntriesSince(end).Equal(start) {
		t.Fatalf("start_date = %s, want RecentEntriesSince(end_date) = %s", start, RecentEntriesSince(end))
	}
}

// TestRecentEntriesToleratesItemsEnvelope covers the case where the API
// wraps the array as {"items": [...]}, which some Toggl docs show and others
// don't — the exact shape couldn't be confirmed ahead of integration.
func TestRecentEntriesToleratesItemsEnvelope(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"items": []map[string]any{
				{"id": 1, "description": "wrapped", "start": "2026-07-15T09:00:00Z", "stop": nil, "duration": -1},
			},
		})
	})
	entries, err := c.RecentEntries(context.Background())
	if err != nil {
		t.Fatalf("RecentEntries: %v", err)
	}
	if len(entries) != 1 || entries[0].Description != "wrapped" {
		t.Fatalf("entries = %+v, want the one wrapped entry", entries)
	}
}

func TestProjectsDoesNotFilterInactive(t *testing.T) {
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, []map[string]any{
			{"id": 101, "name": "ACME", "color": "#e36a00", "active": true},
			{"id": 102, "name": "Old Client", "color": "#999999", "active": false},
		})
	})
	projects, err := c.Projects(context.Background())
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("projects = %d, want 2 (archived projects must still resolve names for old entries)", len(projects))
	}
	// Active lets the picker hide archived projects while names still resolve.
	if !projects[0].Active || projects[1].Active {
		t.Fatalf("active = %v, %v; want true, false", projects[0].Active, projects[1].Active)
	}
	req := (*requests)[0]
	if req.URL.Path != "/workspaces/555/projects" {
		t.Fatalf("path = %q, want /workspaces/555/projects", req.URL.Path)
	}
}

func TestRateLimitedReturnsErrRateLimited(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	})
	_, err := c.Me(context.Background())
	if err != ErrRateLimited {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
}

func TestInvalidTokenReturnsErrInvalidToken(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	if _, err := c.Me(context.Background()); err != ErrInvalidToken {
		t.Fatalf("401: err = %v, want ErrInvalidToken", err)
	}
}

// 403 is its own error: Toggl uses it for a rejected Basic login and for a
// refused action (e.g. a workspace where only admins may create projects).
func TestForbidden403ReturnsErrForbidden(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Incorrect username and/or password"))
	})
	_, err := c.CreateProject(context.Background(), "x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("403: err = %v, want ErrForbidden", err)
	}
	if errors.Is(err, ErrInvalidToken) {
		t.Fatal("403 must not read as an invalid token outside onboarding")
	}
}

func TestOtherErrorReturnsAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	})
	_, err := c.Me(context.Background())
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err = %v (%T), want *APIError", err, err)
	}
	if apiErr.StatusCode != 500 || apiErr.Body != "boom" {
		t.Fatalf("APIError = %+v, want status 500 body 'boom'", apiErr)
	}
}

// Retry policy tests. The invariants: transient failures retry with backoff,
// retries are hard-capped at maxAttempts, and deterministic failures — most
// importantly 402 rate limiting — never retry at all.

func TestRetriesTransient5xxThenSucceeds(t *testing.T) {
	calls := 0
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		writeJSON(w, 200, map[string]any{"id": 1, "default_workspace_id": 9})
	})
	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatalf("Me after transient failures: %v", err)
	}
	if me.DefaultWorkspaceID != 9 {
		t.Fatalf("Me = %+v, want workspace 9 from the eventual success", me)
	}
	if calls != 3 {
		t.Fatalf("server saw %d requests, want 3 (two failures + success)", calls)
	}
}

func TestRetriesAreCappedAtMaxAttempts(t *testing.T) {
	calls := 0
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.Me(context.Background())
	if err == nil {
		t.Fatal("want an error after exhausting retries")
	}
	if calls != maxAttempts {
		t.Fatalf("server saw %d requests, want exactly %d — retries must not run indefinitely", calls, maxAttempts)
	}
}

func TestRateLimit402NeverRetries(t *testing.T) {
	calls := 0
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusPaymentRequired)
	})
	_, err := c.Me(context.Background())
	if err != ErrRateLimited {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
	if calls != 1 {
		t.Fatalf("server saw %d requests, want 1 — retrying an exhausted quota burns more of it", calls)
	}
}

func TestBadRequest400NeverRetries(t *testing.T) {
	calls := 0
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("start_date and end_date are both required"))
	})
	_, err := c.RecentEntries(context.Background())
	if err == nil {
		t.Fatal("want an error for a 400")
	}
	if calls != 1 {
		t.Fatalf("server saw %d requests, want 1 — 400s are deterministic and must not retry", calls)
	}
}

func TestRetriedPostResendsFullBody(t *testing.T) {
	var bodies []string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeJSON(w, 200, map[string]any{
			"id": 1, "workspace_id": 555, "description": "retry me",
			"start": "2026-07-15T10:00:00Z", "stop": nil, "duration": -1,
		})
	})
	_, err := c.Start(context.Background(), StartRequest{Description: "retry me"})
	if err != nil {
		t.Fatalf("Start after transient failure: %v", err)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] || bodies[1] == "" {
		t.Fatalf("retried request must resend the identical body; got %q then %q", bodies[0], bodies[1])
	}
}

// Pinned against the live API (docs/ARCHITECTURE.md): name and active are
// enough, and Toggl picks the colour.
func TestCreateProjectSendsNameAndActive(t *testing.T) {
	var body map[string]any
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{
			"id": 223149804, "workspace_id": 555, "name": "newclient",
			"active": true, "color": "#0b83d9", "is_private": true,
		})
	})
	p, err := c.CreateProject(context.Background(), "newclient")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID != 223149804 || p.Name != "newclient" || !p.Active || p.Color != "#0b83d9" {
		t.Fatalf("project = %+v, want the decoded response", p)
	}
	req := (*requests)[0]
	if req.Method != http.MethodPost || req.URL.Path != "/workspaces/555/projects" {
		t.Fatalf("request = %s %s, want POST /workspaces/555/projects", req.Method, req.URL.Path)
	}
	if len(body) != 2 || body["name"] != "newclient" || body["active"] != true {
		t.Fatalf("body = %v, want exactly name and active", body)
	}
}

// The projects list is paged at 151 (confirmed live): ask for the next page
// only while a page comes back full.
func projectsPage(n, from int) []map[string]any {
	var out []map[string]any
	for i := 0; i < n; i++ {
		out = append(out, map[string]any{"id": from + i, "name": fmt.Sprintf("p%d", from+i), "active": true})
	}
	return out
}

func TestProjectsFollowsFullPages(t *testing.T) {
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "", "1":
			writeJSON(w, 200, projectsPage(151, 1))
		case "2":
			writeJSON(w, 200, projectsPage(2, 152))
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("page"))
			writeJSON(w, 200, []any{})
		}
	})
	projects, err := c.Projects(context.Background())
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(projects) != 153 || len(*requests) != 2 {
		t.Fatalf("got %d projects in %d requests, want 153 in 2", len(projects), len(*requests))
	}
}

func TestProjectsStopsAfterShortPage(t *testing.T) {
	c, requests := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, projectsPage(3, 1))
	})
	if _, err := c.Projects(context.Background()); err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(*requests) != 1 {
		t.Fatalf("requests = %d, want 1 for a short first page", len(*requests))
	}
	if q := (*requests)[0].URL.RawQuery; q != "" {
		t.Fatalf("first request query = %q, want none (same request as before paging)", q)
	}
}
