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
| Recent entries | `GET /me/time_entries?start_date=...&end_date=...` (client bounds the window to the last 8 days). Confirmed live: the API 400s ("start_date and end_date are both required") if either is sent without the other |
| Projects (for name/color lookup) | `GET /workspaces/{workspace_id}/projects` — **not** filtered by `active`; a past entry can reference an archived project, and filtering it out would break its display |
| Start | `POST /workspaces/{workspace_id}/time_entries` with `description`, `start` (RFC3339), `duration: -1`, `created_with: "lazytoggl"`, optional `project_id` |
| Stop | `PATCH /workspaces/{workspace_id}/time_entries/{id}/stop` (no body) |
| Edit | `PUT /workspaces/{workspace_id}/time_entries/{id}`; `stop`/`duration` only sent when the entry is finished, omitted while running |
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
   response updates local state; never refetch after a mutation.
3. `workspace_id` (from onboarding's `GET /me`) is cached in the config file so later
   launches skip `/me` entirely.
4. On HTTP 402: status-bar "rate limited — try again later" message; no automatic retries.

Typical day budget: 1 startup fetch + ~10–20 start/stop/edit actions ≈ well under limits,
as long as nothing polls.

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
