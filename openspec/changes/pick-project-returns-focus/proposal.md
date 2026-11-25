# Proposal

## Why

In the start dialog (`c`, or `enter` to resume) and the log dialog (`m`, `o`, `O`),
picking a project from the project row's picker leaves focus on the project row. There,
`enter` means "open the picker". So the natural reflex, pick a project then press
`enter` to go, reopens the picker instead of starting the timer or logging the entry.
The `@` dropdown doesn't have this problem: after a pick, focus is still on the
description, where `enter` starts or logs. Toggl's app works the same way: `enter`
only starts from the text field.

## What Changes

- **In the start and log dialogs, picking a project from the project row's picker
  moves focus to the description row.** The next `enter` starts the timer or logs the
  entry, the same as after picking from the `@` dropdown.
- **Closing the picker with `esc` leaves focus on the project row** (nothing was
  picked, so nothing moves).
- **The edit dialog (`e`) is unchanged.** Focus stays on the project row after a pick.
  `enter` there never saves (saving goes through `esc` and the review), so there is
  no reflex to protect, and the edit form is navigated row by row.
- No other key changes. To change the project again, `tab` back to the project row.

### Keybindings

| Context | Key | Action |
|---|---|---|
| Project picker, start and log dialogs | `enter` / `tab` | Pick the project, close the picker, and move focus to the description **(changed: focus used to stay on the project row)** |
| Project picker, edit dialog | `enter` / `tab` | Pick the project and close the picker; focus stays on the project row (unchanged) |
| Project picker, any dialog | `esc` | Close the picker, project and focus unchanged (unchanged) |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `project-picking`: the "Project picker from the project row" requirement says where
  focus goes after a pick: the description in the start and log dialogs, the project
  row in the edit dialog.

## Impact

- `internal/ui/dialogs/start.go`: on a pick from the row's picker, focus the
  description.
- `internal/ui/dialogs/edit.go`: the same, in log mode only.
- Tests: `TestStartPickProjectFromRow` in `internal/ui/project_test.go` currently presses
  `shift+tab` after picking to get back to the description. With this change that would
  move to the project row, so the test is rewritten to press `enter` straight after
  the pick. New cases go in `project_test.go` and `log_test.go`.
- Docs: `docs/REQUIREMENTS.md` (the Projects keymap's picker row).
- API: none. No request is added or removed.
