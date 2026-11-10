# Proposal

## Why

An entry's project can't be set or changed anywhere. A new entry gets no project, a
resumed entry silently keeps its old one, and the edit dialog has no project field.
Projects are already loaded once per session, so picking one costs no extra requests.
Users of the Toggl app also expect to type `@` in the description to pick or create a
project, and to see the same A–Z project list.

## What Changes

- **A project row** in the start dialog (below the description) and in the edit
  dialog (after duration, or after start for a running entry). `enter` on the row
  opens a project picker popup. The popup has its own search line.
- **Inline `@`** in the description of both dialogs. `@` typed at the start of a word
  (at the very beginning or after a space) opens a dropdown under the description.
  Typing after it keeps editing the description and filters the dropdown; spaces are
  part of the search. `@` inside a word (`me@acme.com`) types normally.
- **While the dropdown is open, `enter` (or `tab`) picks** the selected project: the
  `@query` text is removed from the description and the project row is set. **`esc`
  closes only the dropdown** and keeps the text as typed. Once the dropdown is
  closed, `enter` does what it normally does: start the timer in the start dialog,
  or move to the next row in the edit dialog.
- **One list for both ways in.** "No project" first (only when the entry has a
  project), then active projects A–Z, ignoring case. Archived projects are hidden
  from the list but still name old entries. Typing filters by case-insensitive
  substring, keeping A–Z.
- **Creating a project.** While the search doesn't exactly match an existing project
  name (archived ones included), the last option is `+ Create "<search>"`. It is the
  only option when nothing matches. Picking it marks the project row
  `<name> (new)`. **The project is created only when the entry is saved** (start, or
  `y` in the edit review), then the entry is saved with it. Cancelling or discarding
  sends nothing.
- The edit review lists a project change as `project  ACME -> Client X` (or
  `-> newclient (new)`, or `-> (none)`).
- Removing a project ("No project") actually clears it. Today the client leaves
  `project_id` out of the update, so the old project would silently stay.
- Permission errors get their own message. A `403` while creating a project (some
  workspaces let only admins create projects) no longer shows "invalid API token".

Assumption: matching is substring, not fuzzy (recommended during exploration; the
user moved to the proposal before confirming). Fuzzy matching could replace it later
without changing any keys.

Out of scope: tags, clients, the billable flag, project colours in the picker, and
fuzzy matching.

### Keybindings

| Context | Key | Action |
|---|---|---|
| Start dialog | `tab` / `shift+tab`, `down` / `up` | Move between description and project **(new)** |
| Start dialog, description | `enter` | Start the timer (unchanged; only while no dropdown is open) |
| Start / edit, project row | `enter` | Open the project picker **(new)** |
| Description (start / edit) | `@` at a word start | Open the project dropdown **(new)** |
| Project dropdown / picker | letters, `backspace` | Edit the search (in the dropdown, the description text itself) **(new)** |
| Project dropdown / picker | `up` / `down`, `ctrl+p` / `ctrl+n` | Move the selection **(new)** |
| Project dropdown / picker | `enter`, `tab` | Pick the selected option **(new)** |
| Project dropdown / picker | `esc` | Close it; the dropdown keeps the typed text, the picker leaves the project unchanged **(new)** |
| Edit form | `esc` | Unchanged, except that while the dropdown is open it only closes the dropdown |

## Capabilities

### New Capabilities

- `project-picking`: choosing, clearing, and creating an entry's project from the start
  and edit dialogs, through the project row or inline `@`, and how a new project is
  created on save.

### Modified Capabilities

- `entry-editing`: the edit form gains a project row, `enter` on it opens the picker,
  and the review lists project changes.

## Impact

- `internal/toggl`: `Project` gains `Active`; a new `Service.CreateProject` (client,
  mock, `httptest` fixtures); `Update` can clear a project; a new forbidden error kept
  apart from the invalid-token error, still treated as a bad token by onboarding.
- `internal/ui/dialogs`: new shared project list and picker popup, an inline
  `@` suggester wrapped around the description inputs; `start.go` becomes a two-row
  form; `edit.go` gains the project row and review line.
- `internal/ui/app.go`: passes the project list into both dialogs; runs
  create-then-save; adds a created project to the session's project map.
- Tests: new dialog tests, `edit_test.go` and `driver_test.go` cases, `client_test.go`
  fixtures for create, clear, and 403.
- Docs: `docs/REQUIREMENTS.md` (feature list, keymap; removes the backlog item),
  `docs/ARCHITECTURE.md` (new endpoint, request costs), `docs/PLAN.md` (decisions).
- API: startup and refresh unchanged. Showing the picker costs 0. Starting or saving
  costs 1 request as today, or 2 when it creates a project.
