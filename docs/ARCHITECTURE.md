# lazytoggl — Architecture

## Package layout

```
lazytoggl/
├── main.go                 # wires config + service + tea.Program
├── go.mod                  # module github.com/KorboCoder/lazytoggl
├── docs/                   # PLAN.md, REQUIREMENTS.md, ARCHITECTURE.md
└── internal/
    ├── toggl/
    │   ├── service.go      # Service interface: the seam between UI and data
    │   ├── types.go        # TimeEntry, Project, Workspace, Me
    │   ├── client.go       # real HTTP client (Phase 2) — net/http only
    │   └── mock.go         # in-memory fake with seeded data (Phase 1 prototype)
    ├── config/             # config file (~/.config/lazytoggl/) + keyring token storage
    └── ui/
        ├── app.go          # root model: panel focus, dialog routing, global keys
        ├── panels/         # timer.go, entries.go
        ├── dialogs/        # start.go, edit.go, confirm.go
        ├── keys.go         # key.Binding definitions (drives the help view)
        └── styles.go       # lipgloss styles, adaptive light/dark
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
  with realistic fake entries, including a "running" one. The prototype is not throwaway.
- **Phase 2** swaps in `client.go` (real HTTP) behind the same interface. UI code does
  not change.
- Tests exercise the client against `net/http/httptest` fixtures and can drive UI logic
  against the mock.

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

Base URL `https://api.track.toggl.com/api/v9`, HTTP Basic auth with `api_token:api_token`.
Exact request/response shapes to be verified against https://engineering.toggl.com/docs/
during Phase 2 (the docs site is JS-rendered; shapes below are from prior knowledge).

| Purpose | Endpoint |
|---|---|
| Validate token, default workspace | `GET /me` |
| Recent entries (running one has negative duration) | `GET /me/time_entries` |
| Start | `POST /workspaces/{wid}/time_entries` with `duration: -1`, `start`, `created_with` |
| Stop | `PATCH /workspaces/{wid}/time_entries/{id}/stop` |
| Edit | `PUT /workspaces/{wid}/time_entries/{id}` |
| Delete | `DELETE /workspaces/{wid}/time_entries/{id}` |

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

- **API token:** OS keychain via `zalando/go-keyring` (service `lazytoggl`). Never
  written to the config file. `TOGGL_API_TOKEN` env var respected as an override for
  keychain-less environments.
- **Config file:** `~/.config/lazytoggl/config.json` — non-secret settings only
  (`workspace_id`, future preferences). Created with 0600.
