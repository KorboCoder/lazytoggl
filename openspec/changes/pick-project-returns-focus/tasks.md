# Tasks

Test fixtures: the driver in `internal/ui/driver_test.go`, with projects ACME (101) and
Internal (102). `openLog(d)` in `log_test.go` opens the log dialog; `dialogMsg` runs a
key's command inside the dialog and returns the message.

## 1. Start dialog

- [x] 1.1 A pick from the project row's picker moves focus to the description; `esc` in the picker leaves focus on the project row. Rewrite `TestStartPickProjectFromRow` in `internal/ui/project_test.go` to press `enter` straight after the pick (no `shift+tab`) and expect a timer for `Task` on ACME. Add `TestStartPickerEscKeepsProjectRow`: open the picker, press `esc`, and the dialog still shows `> project`
- [x] 1.2 The same through `tab` in the picker (it picks like `enter`): add `TestStartPickerTabPicksThenEnterStarts`

## 2. Log dialog

- [x] 2.1 In log mode, a pick from the project row's picker moves focus to the description, so the next `enter` logs. Add `TestLogPickFromRowThenEnterLogs` to `internal/ui/log_test.go`: type `standup`, `shift+tab` to the project row (the last row), `enter`, type `int`, `enter`, then `enter` returns a `LogSubmitMsg` for `standup` on Internal (102)
- [x] 2.2 The edit dialog keeps focus on the project row after a pick, and nothing is saved. Add `TestEditPickFromRowKeepsFocus` to `internal/ui/project_test.go` (the dialog shows `> project` after the pick, and no `Update` call was made)

## 3. Docs and verification

- [x] 3.1 Update `docs/REQUIREMENTS.md`: the Projects keymap's `enter`, `tab` picker row says that in the start and log dialogs a pick from the row's picker moves focus to the description. Verify by reading it against `proposal.md`'s keybinding table
- [x] 3.2 Run `go test -race ./...` and verify it passes
- [x] 3.3 Run `go run . -mock` and verify by hand:
  - `c`, type a description, `tab`, `enter`, pick a project, then `enter` starts the timer.
  - `m`, then the same steps through the project row, logs the entry.
  - `e` on an entry: picking a project keeps focus on the project row.

## 4. Review follow-ups

- [x] 4.1 Tighten the tests to their scenarios: `TestLogPickFromRowThenEnterLogs` runs the full round trip (one `Log` call, dialog closed, entry on Internal); `TestStartPickerEscKeepsProjectRow` starts from a set project and moves the selection before `esc`; add `TestLogPickerEscKeepsProjectRow` and `TestStartPickCreateThenEnterStarts`. Note the focus move in the Start and Log-entry keymap tables of `docs/REQUIREMENTS.md`. Verify with `go test -race ./...`

## Workflow follow-up

- Run superpowers:requesting-code-review once all tasks are checked.
- Archive with `/opsx:archive` after `go test -race ./...` passes.
