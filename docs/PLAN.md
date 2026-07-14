# lazytoggl — Project Plan

A lazygit-style TUI for [Toggl Track](https://toggl.com/track/), written in Go with
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

**North star:** near feature-parity with the Toggl desktop/web apps, but the core loop
comes first and must be excellent: **start a task, stop the active task.**

See also: [REQUIREMENTS.md](REQUIREMENTS.md) · [ARCHITECTURE.md](ARCHITECTURE.md)

## Process

The project moves through phases, each ending in a checkpoint where Daniel reviews and
approves before the next phase begins.

### Phase 0 — Requirements & architecture (this doc set)

- Scaffold the repo (`git init`, `go mod init github.com/KorboCoder/lazytoggl`)
- Write REQUIREMENTS.md (feature spec, keymap, UI sketch, non-functional requirements)
- Write ARCHITECTURE.md (package layout, service seam, data flow, Toggl API + rate-limit notes)

**Checkpoint:** docs reviewed and approved. ✅ once approved, Phase 1 starts.

### Phase 1 — Interactive mock prototype (no API)

Build the *full* TUI running against an in-memory mock of the Toggl service, seeded with
realistic fake data. Every MVP flow works end-to-end against fake state:

- Start a new entry (dialog with description input)
- Stop the running entry
- Resume a past entry (pre-filled dialog)
- Edit and delete entries (with confirmation)
- Live-ticking elapsed time, day grouping with totals, help overlay

The prototype is **not throwaway**: the UI talks only to a `toggl.Service` interface, and
Phase 2 swaps the mock for the real HTTP client behind the same interface.

**Checkpoint:** Daniel runs `go run .`, exercises the UX, and either approves or requests
changes. Iterate here until the UX feels right — changing layouts/keybindings is cheapest
in this phase.

### Phase 2 — MVP (real API)

- Real Toggl API v9 client (`net/http`, no heavy dependencies) with `httptest` unit tests
- First-run onboarding: prompt for the API token, validate with `GET /me`, store in the
  OS keychain (macOS Keychain via `zalando/go-keyring`)
- Swap the mock for the real client; add error/offline states (status-bar errors,
  rate-limit messaging)
- Strict request budget: no polling, optimistic local state (see ARCHITECTURE.md —
  required by the Free-plan limit of 30 requests/hour)

**Checkpoint:** Daniel tracks real time with it; entries verified in the Toggl web app.

### Phase 3 — Daily use & feedback loop

Daniel uses lazytoggl as his actual tracker and files feedback. Iterate on rough edges
first, then pull from the backlog (REQUIREMENTS.md) toward feature parity: project/tag
pickers, reports, multi-workspace, and beyond.

## Decisions log

| Decision | Choice | Why |
|---|---|---|
| Stack | Go + Bubble Tea (+ bubbles, lipgloss) | User requirement |
| MVP scope | start/stop, resume recent, edit/delete | User priority; pickers & reports deferred |
| Token storage | OS keychain (`zalando/go-keyring`) | Most secure; no plaintext token on disk |
| UI style | lazygit-style panels, vim keys, popup dialogs | User preference; fits the project name |
| Refresh model | Manual only, zero polling, optimistic state | Free plan allows only 30 req/hour |
| Start verbs | `c` create · `space` pause/resume · `enter` resume/stop selection | Prototype feedback (2026-07-15): `+ New entry` row and `x` removed as redundant |
| Timer-panel `e`/`d` | Act on the running entry, not the list selection | Prototype feedback (2026-07-15) |
