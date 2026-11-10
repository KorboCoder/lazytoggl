# lazytoggl — Architecture

## Package layout

```
lazytoggl/
├── main.go                 # -mock flag, onboarding gate, wires service + tea.Program
├── go.mod                  # module github.com/KorboCoder/lazytoggl
├── docs/                   # PLAN.md, REQUIREMENTS.md, ARCHITECTURE.md
└── internal/
    ├── toggl/
    │   ├── service.go      # Service interface: the seam between UI and data
    │   ├── types.go        # TimeEntry, Project, Workspace, Me
    │   ├── client.go       # real HTTP client (Phase 2) — net/http only
    │   └── mock.go         # in-memory fake with seeded data (Phase 1 prototype)
    ├── config/             # config.json (workspace id) + keychain token storage
    └── ui/
        ├── app.go          # root model: panel focus, dialog routing, global keys
        ├── keys.go         # key.Binding definitions (drives the help view)
        ├── panels/         # timer.go, entries.go
        ├── dialogs/        # start.go, edit.go, confirm.go
        ├── theme/          # theme.go: lipgloss styles, Panel(), adaptive light/dark
        └── onboard/         # first-run token entry, run as its own tea.Program
```

**Dependencies:** `charmbracelet/bubbletea`, `charmbracelet/bubbles`,
`charmbracelet/lipgloss`, `zalando/go-keyring`. Standard library for HTTP and JSON.

## The service seam

The single most important design decision: the UI depends only on this interface.

```go
// internal/toggl/service.go
type Service interface {
    Me(ctx context.Context) (Me, error)
    RecentEntries(ctx context.Context) ([]TimeEntry, error) // includes the running entry
    Projects(ctx context.Context) ([]Project, error)         // archived included, Active marks them
    CreateProject(ctx context.Context, name string) (Project, error)
    Start(ctx context.Context, req StartRequest) (TimeEntry, error)
    Stop(ctx context.Context, entryID int64) (TimeEntry, error)
    Update(ctx context.Context, entry TimeEntry) (TimeEntry, error)
    Delete(ctx context.Context, entryID int64) error
}
```

- **Phase 1** runs the whole TUI against `mock.go` — an in-memory implementation seeded
  with realistic fake entries, including a "running" one. The prototype is not throwaway;
  `main.go -mock` still runs it, unchanged, for UX iteration without a real account.
- **Phase 2** swaps in `client.go` (real HTTP) behind the same interface. `ui.Model`
  imports only `toggl.Service` — zero code in `internal/ui/...` changed to support it.
- Tests exercise the client against `net/http/httptest` fixtures (`internal/toggl/client_test.go`)
  and drive UI logic against the mock (`internal/ui/driver_test.go`).

## UI data flow (Elm architecture)

- The **root model** (`app.go`) owns panel focus and any open dialog, and delegates
  `Update`/`View` to child models (panels, dialogs).
- **All service calls are async `tea.Cmd`s** returning typed messages
  (`entriesLoadedMsg`, `entryStartedMsg`, `apiErrorMsg`, …). The UI never blocks on the
  network; in-flight actions show a spinner/status.
- A **1-second `tea.Tick`** drives the elapsed-time display. Elapsed time is computed
  locally as `now - entry.Start`; ticking costs zero API requests.
- **Optimistic local state:** the entries slice held by the root model is the source of
  truth for rendering. Mutation responses from the API replace/patch items in that slice
  directly — never followed by a refetch.
- Errors land in a status bar; a failed optimistic update rolls back to the previous
  state and shows the error.

## Toggl API v9

Base URL `https://api.track.toggl.com/api/v9`. HTTP Basic auth: **username is the
API token, password is the literal string `"api_token"`** — not the token in the
password field, a common integration mistake. Confirmed against Toggl's engineering
docs and community API reference during Phase 2 implementation (`internal/toggl/client.go`).

| Purpose | Endpoint |
|---|---|
| Validate token, default workspace | `GET /me` → `default_workspace_id` (falls back to legacy `default_wid` if absent) |
| Recent entries | `GET /me/time_entries?start_date=...&end_date=...` (the window is `[toggl.RecentEntriesSince(now), now]`, i.e. the last 192h set by the exported `toggl.RecentEntriesLookback`; `toggl.InRecentWindow(start, now)` tests membership. The mock filters its entries with it, plus the running entry, so `-mock` behaves like the API on refresh, and the UI uses it to warn when an edit moves an entry's start out of the window, including past now. That the API leaves out entries starting after `end_date` is assumed from the parameter, not confirmed live). Confirmed live: the API 400s ("start_date and end_date are both required") if either is sent without the other |
| Projects (for name/color lookup and the picker) | `GET /workspaces/{workspace_id}/projects` — **not** filtered by `active`; a past entry can reference an archived project, and filtering it out would break its display. The picker hides projects whose `active` is false. Paged at 151: the client asks for `?page=2…` only while a page comes back full of projects it hasn't seen, and at most 5 pages per call |
| Create project | `POST /workspaces/{workspace_id}/projects` with `{"name", "active": true}`; only when an entry is saved with a `(new)` project, right before the entry's own request. A taken name is a `400` |
| Start | `POST /workspaces/{workspace_id}/time_entries` with `description`, `start` (RFC3339), `duration: -1`, `created_with: "lazytoggl"`, optional `project_id` |
| Stop | `PATCH /workspaces/{workspace_id}/time_entries/{id}/stop` (no body) |
| Edit | `PUT /workspaces/{workspace_id}/time_entries/{id}`; `stop`/`duration` only sent when the entry is finished, omitted while running; `project_id` always sent, `null` for no project (leaving it out would keep the old project) |
| Delete | `DELETE /workspaces/{workspace_id}/time_entries/{id}` |

A running entry is identified by a null/absent `"stop"` field. Toggl's own docs
describe the running entry's `"duration"` as a negative number encoding the start
time, but the exact convention is undocumented and **not relied on** — `stop == null`
is the only signal the client trusts; a running entry's `Duration` is always stored
internally as the sentinel `-1` regardless of what the API sent, matching `mock.go`.

`GET /me/time_entries`'s exact response shape (bare JSON array vs. an
`{"items": [...]}` envelope) could not be pinned down with certainty from
documentation alone — different Toggl doc sources show each. `decodeTimeEntryList`
in `client.go` tries the bare array first, then falls back to the envelope, so
either shape works.

**Confirmed live (2026-11-10, change `pick-entry-project`):**

- **Auth:** HTTP Basic, the API token as the username and the literal `api_token`
  as the password. A token sent as a Bearer header gets `401`; a wrong Basic login
  gets `403` with the body `Incorrect username and/or password`. Failed logins did
  not reduce the quota.
- **Quota headers:** every response carries `X-Toggl-Quota-Remaining` and
  `X-Toggl-Quota-Resets-In` (seconds).
- **Projects are paged:** `GET /workspaces/{wid}/projects` returns a bare array with
  `X-Page: 1` and `X-Page-Size: 151`. A workspace with 151 or more projects needs
  further pages; the client asks for them with `?page=2…`. Each project carries `active`, `color`, `is_private`, `client_id`,
  `client_name`, and more.
- **Creating a project:** `POST /workspaces/{wid}/projects` with only
  `{"name", "active": true}` succeeds (`200`). The response is the project, with a
  colour Toggl picked (`#0b83d9` here) and `is_private: true` by default.
- **Duplicate name:** the same `POST` with an existing name is a `400` with the body
  `"a project with the name '<name>' already exists"`.
- **Clearing a project:** `PUT /workspaces/{wid}/time_entries/{id}` with
  `"project_id": null` (and the usual `workspace_id`, `description`, `start`,
  `stop`, `duration`) removes the project; the response has `project_id: null`.
- **Deleting:** `DELETE` on a time entry or a project returns `200`.
- **Not checked:** whether `GET .../projects` without parameters includes archived
  projects (this workspace had none), and that `?page=N` is the paging parameter
  (this workspace has 3 projects). The client stops as soon as a page brings no new
  projects, so a server that ignored `page` costs one extra request, not the quota.

## Rate limits

Per https://support.toggl.com/en-us/article/api-webhook-limits-85auc8/ — and Daniel is on
the **Free plan**:

- Workspace/organization endpoints: Free = **30 requests/hour** per user per org
  (Starter 240, Premium 600).
- User endpoints (`/me`, `/me/time_entries`): 30 requests/hour on all non-Enterprise plans.
- Exceeding the quota returns **HTTP 402**; sliding 60-minute window.

Design consequences (these are architectural constraints, not optimizations):

1. **Zero polling.** One `GET /me/time_entries` at startup populates the list *and* the
   running entry — no separate `/current` call. Elapsed time ticks locally.
2. **Optimistic local state.** Start/stop/edit/delete each cost exactly 1 request; the
   response updates local state; never refetch after a mutation. Starting or saving
   with a new project costs 2: the project is created first and added to the
   session's project map from the response, then the entry is saved with its ID.
   Browsing, filtering, or picking projects costs nothing.
3. `workspace_id` (from onboarding's `GET /me`) is cached in the config file so later
   launches skip `/me` entirely.
4. **Projects are fetched once per session** (at startup). A manual refresh (`r`)
   refetches entries only and reuses the session's project map — so startup costs 2
   requests and each refresh costs 1 (2 only while projects have never loaded, e.g.
   after a startup that failed, until they do). (New projects created mid-session on the web app
   appear after a restart. Workspaces with 151 or more projects pay one request per
   extra page at startup, at most 4 more.)
5. On HTTP 402: never retried automatically (retrying an exhausted hourly quota just
   burns more of it), surfaced prominently via the error notice (below).

**Retry policy** (`client.go`): transient failures — network errors, 5xx, 408, 429 —
retry with exponential backoff (500ms → 1s → 2s), hard-capped at 3 total attempts so
retries can never run indefinitely. Deterministic failures (400s, 402 rate limits,
401/403 auth) never retry. A retried POST resends the identical body.

**Auth errors:** 401 is `ErrInvalidToken`; 403 is `ErrForbidden`. Toggl answers a wrong
Basic login with 403 and also uses 403 for an action the user may not take (e.g. a
workspace where only admins may create projects). Onboarding treats both as "token
rejected"; the main app reports a 403 as a refused request, not a bad token.

**Error notice UI** (`app.go`): any failed API call raises a floating red-bordered
popup in the top-right corner (composited over the panels via `theme.Overlay`,
ANSI-aware splicing) in addition to the status-bar line, so errors can't be missed or
silently overwritten by the next status change. Rate-limit errors get a message
explaining the Free-plan quota. Dismissed with `esc`, cleared by the next successful
API call, or auto-expires after 10s.

Typical day budget: 2 startup requests + ~10–20 start/stop/edit actions ≈ well under
limits, as long as nothing polls. Note when testing: repeated launches add up fast —
each launch costs 2 requests against the 30/hour window.

## Config & secrets

- **API token:** OS keychain via `zalando/go-keyring` (service `lazytoggl`), written by
  `internal/config.SaveToken`. Never written to the config file. `$TOGGL_API_TOKEN`
  overrides the keychain when set (checked first in `config.LoadToken`), for
  headless/dev use.
- **Config file:** `os.UserConfigDir()/lazytoggl/config.json` (e.g.
  `~/Library/Application Support/lazytoggl/config.json` on macOS) — non-secret settings
  only (`workspace_id`, future preferences). Created with 0600.
- **Onboarding** (`internal/ui/onboard`): a standalone `tea.Program`, run once from
  `main.go` before the main app, when no token is found. It collects the token,
  validates it via `Client.Me`, and returns the token plus the authenticated `Me` —
  `main.go` then saves both and launches the real app. Kept separate from `ui.Model` so
  onboarding state never leaks into the main app's state machine.
- **`-mock` flag:** bypasses config/keychain/onboarding entirely and runs directly
  against `toggl.NewMock()`, for UX work and demos without a real account.
