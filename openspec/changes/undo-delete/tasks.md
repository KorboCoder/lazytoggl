# Tasks

Test fixtures: `driver_test.go` pins `fixedNow` to Tue 14 Jul 2026 12:00. ID1 is running
on ACME, ID2 is "Standup", and ID3 is "Yesterday task" (list rows: ID1, ID2, then ID3
under Yesterday). The driver advances the clock by feeding `tickMsg(fixedNow + n·s)`.

## 1. Delete with a grace period, and undo

- [x] 1.1 Replace the confirm dialog with a pending delete: `d` hides the target (list selection, or the timer panel's running or paused entry), stores it with a 5-second deadline, sends nothing, and shows `deleted "<desc>" · u undo (5s)`. Delete `dialogs/confirm.go` and `DeleteConfirmMsg`. Verify in a new `internal/ui/delete_test.go`: `TestDeleteHidesAtOnce` (no `Delete` call; status text), `TestDeleteRunningEntryClearsTimer`, `TestDeleteEmptyDescription`; rewrite `TestDeleteCancelAndConfirm` (→ `TestDeleteThenUndo`) and the `d`/`y` steps in `TestTimerFocusActsOnRunningEntry`
- [x] 1.2 Add `u` (`keyMap.Undo`): restore the pending entry as it was (running, or paused via `pausedID`), with the cursor on it and no request; `u` with nothing pending does nothing. Verify with `TestUndoRestoresEntry`, `TestUndoRestoresRunning`, `TestUndoRestoresPaused`, `TestUndoWithNothingPending`

## 2. Sending the delete

- [x] 2.1 On `tickMsg`, send the pending delete when its deadline passes, counting the hint down each second. Verify with `TestDeleteCountsDown` (`(3s)` after 2s) and `TestDeleteSentAfterGrace` (one `Delete` call at 5s; hint gone; `u` then does nothing)
- [x] 2.2 A second `d` sends the pending delete at once and makes the new one pending. Verify with `TestSecondDeleteCommitsFirst`
- [x] 2.3 A failed delete restores the entry and shows the error (`deleteFailedMsg`). Verify with `TestDeleteFailureRestores`, using a fake `Service` whose `Delete` fails
- [x] 2.4 Refreshes keep the pending entry, and entries whose delete is in flight, hidden. Verify with `TestRefreshKeepsPendingHidden` (then `u` still restores) and `TestRefreshKeepsInFlightDeleteHidden`

## 3. Quitting and the status line

- [x] 3.1 `q` / `ctrl+c` with a pending delete send it, then quit (`tea.Sequence`). Verify with `TestQuitSendsPendingDelete` (running the returned command: `Delete` called, then a quit message) and the same through `ctrl+c` with a dialog open
- [x] 3.2 The undo hint stays visible alongside any other status until the delete is sent or undone. Verify with `TestUndoHintSurvivesOtherStatus` (start a timer within the grace period: both statuses rendered)

## 4. Docs and integration

- [x] 4.1 Update `docs/REQUIREMENTS.md` (feature list: delete with undo; keymap: `d`, `u`, `q`; remove the "Deletion undo" backlog item), `README.md` (keys table), and `docs/PLAN.md` (a decisions row citing this change). Verify by reading each key against `proposal.md`'s table
- [x] 4.2 Run `go test -race ./...` and verify it passes
- [x] 4.3 Run `go run . -mock` and verify by hand: `d` hides an entry and counts down; `u` brings it back; letting it run out removes it; deleting the running timer and pressing `u` resumes the ticking timer; `d` then `q` quits cleanly

## Workflow follow-up

- Run superpowers:requesting-code-review once all tasks are checked.
- Archive with `/opsx:archive` after `go test -race ./...` passes.
