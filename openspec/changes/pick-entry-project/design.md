# Design

## Context

- **Projects** are fetched once per session (`GET /workspaces/{wid}/projects`, one of
  the two startup requests) into `Model.projects map[int64]toggl.Project`, and are used
  only to show names. The client keeps archived projects on purpose so old entries
  keep their names (`TestProjectsDoesNotFilterInactive`), but `Project` has no
  `Active` field, so nothing can tell them apart.
- **The start dialog** (`dialogs/start.go`) is one `textinput`, plus a project that's
  carried along when resuming. `enter` submits.
- **The edit dialog** (`dialogs/edit.go`) is a row form with popups that own all keys
  while open (the date+time field). Its description is a `textinput`.
- **Saving:** both dialogs emit a submit message, and `app.go` closes the dialog
  *before* the request runs. Errors land in the notice popup and status line.
  Mutations patch local state from the response, with no refetch.
- **`Client.Update`** sends `project_id` only when it isn't 0, so a project can't be
  removed today.
- **`Client.do`** maps both 401 and 403 to `ErrInvalidToken`. Onboarding compares the
  error with `==` to show "That token was rejected".
- **`theme.Overlay`** clips anything below the dialog's last line, so a popup must fit
  inside the dialog box.

Requirements: `specs/project-picking/spec.md`, `specs/entry-editing/spec.md`.

## Goals / Non-Goals

**Goals:**
- One project list (ordering, filtering, the create option, rendering) shared by the
  row picker and the inline `@` dropdown, so they can't drift apart.
- Exactly one owner for each key at any time, as with the date+time field. While a
  list is open, it gets `enter`, `tab`, `esc`, and the selection keys.
- No new request unless a project is being created.

**Non-Goals:**
- Fuzzy matching, project colours or clients in the list, tags.
- Keeping the dialog open after a failed save. Today's behaviour (the dialog closes,
  the error is shown) stays as it is for every save.
- Refreshing the project list after startup. A project created in another Toggl app
  appears on the next launch.

## Decisions

### 1. `ProjectList`: a model with no input of its own

New `internal/ui/dialogs/projectlist.go`:

```
ProjectList
  active   []toggl.Project   // active only, sorted A-Z by strings.ToLower(Name)
  names    map[string]bool   // lower-cased trimmed names of ALL projects (archived too)
  hasCurrent bool            // entry has a project -> offer "No project" on empty search
  query    string
  sel      int
  Options() []Option         // {Kind: None|Project|Create, Project, Name}
  SetQuery(q), Move(+-1), Selected() Option, View(width, rows) string
```

- **Filter:** `strings.Contains(strings.ToLower(name), strings.ToLower(q))`. The
  query is used as typed, so spaces count.
- **Create:** offered when `strings.TrimSpace(q) != ""` and
  `!names[lower(TrimSpace(q))]`. Archived names are in `names`, so a duplicate of an
  archived project is never offered.
- **Selection:** resets to the first option on every query change. `Move` clamps (no
  wrap).
- **Rendering:** at most 6 rows, with a scroll window that follows the selection.

*Why a model without an input:* the row picker and the dropdown get their query from
different places. The picker has its own `textinput`; the dropdown uses the
description. Each host owns its input and calls `SetQuery`.

### 2. Row picker (A): `ProjectPicker` popup

A small struct: its own search `textinput` plus a `ProjectList`, and a result like
`FieldResult`: editing, picked(Option), or cancelled. The host dialog opens it on
`enter` on the project row and gives it every key while it's open. It is drawn with
`theme.Panel` over the dialog, the same way the date+time field is.

### 3. Inline `@` (B): `atSuggest` around a description `textinput`

New `internal/ui/dialogs/atsuggest.go`, embedded in both dialogs next to their
description input:

```
 key --> open? --yes--> up/down/ctrl+p/n: move     (consumed)
              |         enter/tab: pick -> splice  (consumed)
              |         esc: close                 (consumed)
              |         else: textinput.Update, then re-check:
              |               value[at] != '@' or cursor <= at -> close
              |               otherwise query = value[at+1 : cursor]
              no
              +-------> textinput.Update; if the key was the single rune '@'
                        and (at == 0 or value[at-1] == ' ') -> open at cursor-1
```

- **What triggers it:** only a key that is exactly one rune, `@`. A pasted burst
  containing `@` never opens the dropdown. Indexes are in runes, matching
  `textinput.Position()`.
- **Splice on pick:** remove `value[at:cursor]`, then turn the spaces left at the join
  into one space (or none at either end), trim the result, and place the cursor where
  the `@` was.
- **Ownership:** while the dropdown is open, the host dialog passes keys to
  `atSuggest` first. Its own `enter` (start the timer, or move to the next row) and
  `esc` (close, or open the review) never run, which is the rule the spec sets.

### 4. Where the popups go

- **Dropdown:** directly below the description input, left-aligned with it, at most
  6 rows plus a border. The edit dialog has room below the description. The start
  dialog is too short, so its `View` pads blank lines while the dropdown is open, and
  the box grows only for that time.
- **Row picker:** anchored to the project row and opened upward when the space below
  is too small. In the edit dialog the project row is last, so it always opens upward.

### 5. Row order: project last in the edit dialog

The edit rows become description, start, stop, duration, project (running:
description, start, project).

- *Why:* the current `tab` path to start, stop, and duration stays as it is, and `@`
  is the fast path for the project anyway. Existing tests keep their row positions.
- *Alternative:* project second, as in the Toggl app. Rejected: it moves every time
  row by one, for both muscle memory and every `setRow` test, for a field `@` already
  reaches from the first row.

The start dialog has two rows, description then project. `tab` wraps.

### 6. The draft carries a project choice, and saving runs in two steps

Both dialogs hold `project projectChoice{ID int64; NewName string}`. `NewName != ""`
means "create on save".

- `StartSubmitMsg{Req, NewProject string}` and
  `EditSubmitMsg{Entry, NewProject string}` gain a `NewProject` field.
- `app.go`: when `NewProject != ""`, it runs `createProjectCmd(name, then)`.
  `then(id)` builds the original start or update with `ProjectID = id`. The result is
  a `projectCreatedMsg{project, next tea.Cmd}`.
  - The app adds the project to `m.projects`, which is how the spec's "created
    project is reused" works.
  - Then it returns `next`.
  - A failed create is an ordinary `errMsg`, and `next` never runs.
- The edit review shows the project line from the choice: `(none)`, a name, or
  `name (new)`.

*Alternative:* one command that makes both calls and returns one message. Rejected:
the project would only reach `m.projects` if both calls succeed, which breaks reuse
after a failed entry save.

### 7. `toggl` changes

- `Project.Active bool`, decoded from `active`. The mock seed adds an archived
  `Old Client` (ID 103).
- `Service.CreateProject(ctx, name string) (Project, error)`:
  `POST /workspaces/{wid}/projects` with `{"name": name, "active": true}`. The mock
  appends the project with the next ID and active set.
- `Update` always sends `project_id`, with `null` when it is 0, so removing a project
  takes effect. `Start` keeps leaving it out when it is 0.
- New `ErrForbidden` for 403, with 401 staying `ErrInvalidToken`.
  - `onboard.friendlyError` treats both as "token rejected", using `errors.Is`.
  - `app.go`'s `errMsg` text for `ErrForbidden` is "Toggl refused the request — you may
    not have permission".
  - `createProjectCmd` wraps the error so the text adds "(this workspace may only let
    admins create projects)".

### 8. Facts to confirm live before building on them (task 1)

These cost about 2–3 requests, and only with the user's OK:
1. Does `PUT` with `"project_id": null` clear the project?
2. Is `POST .../projects` with only `name` and `active` accepted, and which colour
   does the response carry?
3. Is `GET .../projects` paged at the user's project count?
4. What does a duplicate name return?

If 1 fails, the fallback is to try `0`. If the list is paged, the client follows the
pages at startup, costing one extra request per page, and `ARCHITECTURE.md` says so.

**Results (confirmed live, recorded in `docs/ARCHITECTURE.md`):**
1. Yes: `"project_id": null` clears the project.
2. Yes: `name` + `active` is enough. Toggl picks the colour, and the new project is
   private (`is_private: true`); lazytoggl keeps that default.
3. Paged: `X-Page-Size: 151`. The client asks for the next page only while a page
   comes back full, so startup stays at 2 requests below 152 projects.
4. A duplicate is a `400` with the body
   `"a project with the name '<name>' already exists"`, shown like any API error.

Still unknown: whether the projects list includes archived projects (the test
workspace had none). The design doesn't depend on it, since a duplicate of a
missing archived name fails cleanly with the `400` above.

### API request cost

| | Requests |
|---|---|
| Startup | 2, unchanged (more only if task 1 finds paging) |
| Refresh | 1, unchanged |
| Opening, filtering, or picking from the picker or dropdown | 0 |
| Starting or saving with an existing project or none | 1, as today |
| Starting or saving with a new project | 2 (create, then the entry) |
| Cancelling the start dialog, or discarding the edit | 0 |

## Risks / Trade-offs

- [The two-step save fails halfway (the project is created, the entry isn't), and the
  dialog has already closed] → The created project is in the list from then on, so
  the user re-picks it by name. Nothing is created twice. The error is shown, as for
  any failed save.
- [A project created in another Toggl app or the web isn't in the list until the next
  launch, so `@` offers to create a duplicate] → Toggl rejects the duplicate (task 1
  checks how), and the error is shown. Refreshing projects on `r` would cost a request
  per refresh, which the rate-limit budget argues against.
- [`ctrl+p` / `ctrl+n` may be bound by the terminal or tmux] → `up` / `down` always
  work.
- [`@` at a word start can't be typed as a literal character without closing the
  dropdown] → `esc` keeps the text. This is a rare case.
- [The start dialog's box grows while the dropdown is open] → Only for that time, and
  it shrinks back when the dropdown closes.

## Migration Plan

UI and client changes only, plus one new endpoint. No data or config migration.
Rollback is reverting the commits.
