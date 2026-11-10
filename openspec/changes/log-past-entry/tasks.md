# Tasks

Test fixtures: `driver_test.go` pins `fixedNow` to Tue 14 Jul 2026 12:00. ID1 is running
on ACME since 11:50, ID2 is a finished "Standup" with no project today, and ID3 is
"Yesterday task" on Internal. With these fixtures the log dialog's defaults are
11:30–12:00.

## 1. Live API fact (needs the user's OK: about 5 requests)

- [ ] 1.1 With the user's OK, and only while they have no timer of their own running, confirm design.md §5: logging a finished entry with `POST .../time_entries` (`start`, `stop`, `duration`) leaves a running entry running. Use a throwaway running entry and a throwaway finished one, and delete both. Record the result in `docs/ARCHITECTURE.md`'s confirmed-live notes. If the running entry stops, pause and revisit the spec before group 3

## 2. `Service.Log`

- [ ] 2.1 Add `toggl.LogRequest` and `Service.Log`. The client sends `POST /workspaces/{wid}/time_entries` with `workspace_id`, `description`, `start`, `stop`, `duration` in seconds, `created_with`, and `project_id` only when it isn't 0. Verify with `TestLogSendsFinishedEntry` and `TestLogOmitsProjectWhenNone` (`client_test.go`: method, path, exact body keys, decoded finished entry)
- [ ] 2.2 Mock `Log`: append a finished entry with the next ID and leave the running entry running. Verify with `TestMockLogKeepsTimerRunning` (`mock_test.go`)
- [ ] 2.3 Update `docs/ARCHITECTURE.md`: the `Service` listing, a "Log entry" row in the endpoint table, and the request cost (1, or 2 with a new project). Verify by reading it against design.md

## 3. Log mode on the edit dialog

- [ ] 3.1 Add `NewLog(now, projects)`: a log-mode `Edit` titled `Log entry`, with stop = `now` truncated to the minute, start = stop − 30m, an empty description, and project `(none)`. Verify in a new `internal/ui/log_test.go`: `TestLogDialogDefaults` (11:30–12:00, `00:30`) and `TestLogDialogDefaultsAcrossMidnight` (a tick at 00:10 gives a start on the previous day at 23:40)
- [ ] 3.2 `enter` on the description emits `LogSubmitMsg` (with `NewProject` for a `(new)` project), and other rows keep their edit behaviour. Verify with `TestLogEnterOnDescriptionLogs`, `TestLogEnterOnTimeRowOpensField`, `TestLogAtEnterPicksAndDoesNotLog`; every existing `edit_test.go` and `project_test.go` case still passes unchanged
- [ ] 3.3 `esc`: close at once when `changes()` is empty, otherwise show the discard prompt (`y` closes, `n` or `esc` goes back, other keys are ignored); no review in log mode. Verify with `TestLogEscUnchangedCloses`, `TestLogEscAfterTypingAsksThenDiscards`, `TestLogDiscardPromptBackKeepsForm`, `TestLogDiscardPromptIgnoresOtherKeys`

## 4. App wiring

- [ ] 4.1 Add a `keyMap.Log` binding on `m`, shown in the help bar, that opens `NewLog` from the list and from the timer panel. Verify with `TestLogKeyOpensDialogFromList` and `TestLogKeyOpensDialogFromTimer`
- [ ] 4.2 Handle `LogSubmitMsg`: `logCmd` → `entryLoggedMsg` → `replaceEntry`, `SelectID`, and an `entry logged` status; for a `(new)` project, `createProjectCmd` with a `logging…` status first. Verify with `TestLogCreatesFinishedEntry` (mock `Log` 1 call; entry listed under today; cursor on it), `TestLogKeepsTimerRunning` (ID1 still running locally and in the mock), `TestLogWithNewProjectCreatesThenLogs` (`CreateProject` 1, `Log` 1)
- [ ] 4.3 Generalise `savedStatus` with a verb, and warn when a logged entry's start is outside the window. Verify with `TestLogOutsideWindowShowsStatus` (logged on Sun 05 Jul: outside the list) and `TestLogAfterNowShowsStatus`; the edit status tests still pass

## 5. Docs and integration

- [ ] 5.1 Update `docs/REQUIREMENTS.md`: a feature-list item for logging past entries, the `m` key in the keymap, and a "Log entry" keymap section (`enter` logs, `esc` with the discard prompt). Add a `docs/PLAN.md` decisions row citing this change, which records Toggl's manual-mode defaults and `o`/`O` as the follow-up. Verify by reading each key against `proposal.md`'s keybinding table
- [ ] 5.2 Run `go test -race ./...` and verify it passes
- [ ] 5.3 Run `go run . -mock` and verify by hand: `m` shows the last 30 minutes; typing `standup @int` and pressing `enter` twice logs it on Internal while the running timer keeps ticking; `esc` after typing asks before discarding; an untouched `m` then `esc` closes at once

## Workflow follow-up

- Run superpowers:requesting-code-review once all tasks are checked.
- Archive with `/opsx:archive` after `go test -race ./...` passes.
