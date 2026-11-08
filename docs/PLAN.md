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
in this phase. ✅ Approved, with two rounds of keymap/behavior feedback folded in
(see decisions log) — Phase 2 started 2026-07-15.

### Phase 2 — MVP (real API)

- ✅ Real Toggl API v9 client (`internal/toggl/client.go`, `net/http` only), 18
  `httptest`-backed unit tests pinning every request/response shape
- ✅ First-run onboarding (`internal/ui/onboard`): a standalone Bubble Tea screen that
  prompts for the API token, validates it with `GET /me`, and hands back the
  authenticated user
- ✅ Token in the OS keychain via `zalando/go-keyring`; workspace id cached in
  `config.json` so normal launches skip `/me` entirely; `$TOGGL_API_TOKEN` env override
- ✅ `main.go` wires it all together; `-mock` flag keeps the Phase 1 prototype runnable
  for UX work without a real account
- ✅ Typed errors for rate-limiting (`ErrRateLimited`, HTTP 402) and bad tokens
  (`ErrInvalidToken`, 401/403), surfaced as status-bar/onboarding messages

**Checkpoint:** Daniel tracks real time with it; entries verified in the Toggl web app.
Not yet done — this needs Daniel's own API token and a live run, which I can't do myself.

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
| Timer-panel `e`/`d` | Act on the running entry, or the paused one, not the list selection | Prototype feedback (2026-07-15) |
| Pause via `space` | Stops the entry via the API but keeps it pinned in the Timer panel (⏸, frozen clock) instead of clearing to idle | Prototype feedback (2026-07-15): "shouldn't clear the Time bar." Toggl has no native pause, so this is UI-only state (`pausedID`) layered over the entries list — resuming starts a new entry copying the paused one |
| `space` on an idle Timer panel | No-op — never falls back to resuming something from history | Prototype feedback (2026-07-15): "I don't want it to do anything if the Timer panel doesn't have an entry active" |
| Running-entry detection | `stop == null` only, never the API's negative `duration` | The exact negative-duration convention for running entries is undocumented; `stop` nullability is the one signal confirmed across Toggl's own docs |
| `GET /me/time_entries` response shape | Decode defensively: try a bare array, fall back to `{"items": [...]}` | Toggl's own docs disagree on which shape this endpoint returns; couldn't be confirmed ahead of integration |
| Projects list | Never filtered by `active` | A past entry can reference an archived project; filtering it out of the lookup map would show "—" instead of its real name |
| Retry policy | Exponential backoff (500ms→1s→2s), max 3 attempts, transient failures only; 402/4xx never retry | Live rate-limit hit (2026-07-15): bounded retries for flaky networks without ever burning quota on an exhausted rate limit |
| Projects per session | Fetched once at startup; `r` refetches entries only | Live rate-limit hit (2026-07-15): halves the refresh cost against the 30/hour Free-plan quota |
| API errors | Floating top-right popup (esc dismiss, 10s TTL) in addition to the status bar | Live rate-limit hit (2026-07-15): status-bar-only errors were too easy to miss |
| Editing times | `HH:MM` time field popped over the edit form, with modeless vim motions (`h l ^ $ w b`, `ctrl+a`/`ctrl+x`); a duration row; dates not editable yet | Feedback (2026-11-08): retyping `2006-01-02 15:04` strings was tedious. The field only holds digits, so letters can be motions without vim modes; `0` types a digit, so `^` jumps to the first digit. OpenSpec change `edit-entry-time-fields` |
| Saving an edit | `esc` opens a before → after review (save / back / discard) instead of `enter` saving; no review and no request when nothing changed | Feedback (2026-11-08): see what's about to be saved before spending a request |
| Minimum entry length | 1 minute; a start/stop that crosses the other end lands 1 minute from it | Toggl rejects zero-length entries |
