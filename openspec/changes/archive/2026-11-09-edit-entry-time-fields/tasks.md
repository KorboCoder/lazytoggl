# Tasks

## 1. TimeField widget

- [x] 1.1 Add `TimeField` in `internal/ui/dialogs/timefield.go` (minutes value, cursor 0–3, `[min, max]` bounds, `HH:MM` view with a block cursor), with digit overwrite-and-advance that skips the colon and ignores non-motion letters; verify with table tests in `timefield_test.go` ("Typing overwrites in place", "Letters do not change the value")
- [x] 1.2 Add motions `h`/`l`, `^`/`$`, `w`/`b`; verify with `timefield_test.go` cases for each motion and "Cursor skips the colon"
- [x] 1.3 Add `ctrl+a`/`ctrl+x` on the part under the cursor with carry, borrow and clamping to bounds; verify with `timefield_test.go` cases "Minutes carry into hours", "Minutes borrow from hours", "Day boundary holds", and the duration floor of 00:01
- [x] 1.4 Add accept/revert: `enter` range-checks (hours 00–23, minutes 00–59) and reports accepted or an error, `esc` reports reverted; verify with `timefield_test.go` cases "Out-of-range time is rejected" and an accepted value

## 2. Edit form and consistency rules

- [x] 2.1 Rework `Edit` into a draft-backed form (rows description/start/stop/duration, start/stop showing read-only date + `HH:MM`, running entries without stop and with read-only duration) and change `NewEdit` to take `now`, passing `m.now` from `app.go`; verify with `driver_test.go` cases "Finished entry" and "Running entry"
- [x] 2.2 Implement form navigation (`tab`/`down`, `shift+tab`/`up` with wrap; `enter` on description moves on; `enter` on a time row opens its `TimeField` drawn with `theme.Overlay`); verify with `driver_test.go` cases "Enter on description moves on" and "Enter on a time row opens its field"
- [x] 2.3 Implement `normalize` for accepted fields: start keeps stop, stop and duration keep start, 1-minute minimum fallbacks, running start ≤ now − 1m, each time keeps its own date with zero seconds; verify with `driver_test.go` cases "Moving start keeps stop", "Editing duration moves stop", "Start past stop falls back", "Stop before start falls back", "Running start in the future falls back", and "Esc reverts the field"
- [x] 2.4 Preserve untouched timestamps (original start/stop, seconds included, for unchanged fields); verify with `driver_test.go` case "Editing only the description" using a fixture whose start has non-zero seconds

## 3. Review and save

- [x] 3.1 Add the review state: `esc` on the form closes with no request when nothing changed, otherwise shows changed fields before → after; `y`/`enter` emits `EditSubmitMsg`, `esc` returns to the form with the draft intact, `n` emits `CloseMsg`; verify with `driver_test.go` cases "Nothing changed", "Review lists only changed fields", "Save", "Back keeps edits", and "Discard", asserting `Update` call counts on the mock
- [x] 3.2 Replace the old `TestEditValidation` and update `TestEditSavesDescription` (and any other edit test that types a full timestamp or relies on `enter` to save) for the new flow; verify `go test -race ./...` passes
- [x] 3.3 Update the dialog help lines (form, time field, review) to list their keys; verify with a `driver_test.go` `dialogContains` check on each state's help text

## 4. Docs

- [x] 4.1 Add the edit-form, time-field and review keys to the keymap in `docs/REQUIREMENTS.md`, and add a decisions-log row in `docs/PLAN.md` (HH:MM field with modeless vim motions; review instead of `enter`-to-save; 1-minute minimum); verify the keymap matches the proposal's Keybindings table

## 5. Integration check

- [x] 5.1 Run `go test -race ./...` and walk the whole flow in `go run . -mock`: edit a finished entry's duration, move a start past its stop, edit a running entry's start, then save, go back, and discard from the review; verify each matches the spec scenarios

## Workflow follow-up

- Request code review once all tasks are checked off, then archive with `/opsx:archive`.
- Next change: date editing (moving an entry to another day, entries that cross midnight).
