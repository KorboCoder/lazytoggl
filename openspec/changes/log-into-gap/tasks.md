# Tasks

Test fixtures: `driver_test.go` pins `fixedNow` to Tue 14 Jul 2026 12:00. ID1 is running
since 11:50, ID2 "Standup" ran 10:00–10:15 today, and ID3 "Yesterday task" ran
12:00–13:00 yesterday. The list reads Today: ID1, ID2; Yesterday: ID3.

## 1. Gap rules

- [x] 1.1 Create `internal/ui/gaps.go` with `gapBelow` (the latest end before the selected entry on its day; a running entry ends at now; 30 minutes before as the fallback). Verify with table cases in a new `gaps_test.go`: `TestGapBelow` (between two entries, first of the day, exact seconds kept, overlap → no gap, touching → no gap, a running selection, an earlier entry on another day ignored)
- [x] 1.2 Add `gapAbove` (the earliest start after the selected entry on its day, the running entry included; otherwise now rounded down to the minute for today, or 30 minutes after for a past day; a running selection → no gap with the "nothing after" reason). Verify with `TestGapAbove` (between two entries, up to the running timer, newest entry today → now, newest entry of a past day → +30m, overlap → no gap, stop later than now → no gap, a running selection)

## 2. Dialog

- [x] 2.1 Add `dialogs.NewLogAt(start, stop, now, projects)` and make `NewLog` delegate to it. Verify with `TestLogAtOpensOnGivenTimes` in `log_test.go` (exact times shown; an untouched `esc` closes at once; an untouched `enter` logs those exact times, seconds included); every existing `TestLog*` case still passes unchanged

## 3. Keys and app wiring

- [x] 3.1 Add `keyMap.GapBelow` (`o`) and `keyMap.GapAbove` (`O`), in FullHelp only, opening `NewLogAt` on the computed gap for the selected list entry. Verify in `log_test.go`: `TestGapBelowKeyOpensOnGap` (`o` on ID2 → 09:30–10:00), `TestGapAboveKeyOpensUpToTimer` (`O` on ID2 → 10:15–11:50), `TestGapBelowRunningEntry` (`o` on ID1 → 10:15–11:50), `TestGapAbovePastDay` (`O` on ID3 → yesterday 13:00–13:30), and that logging one adds the entry with the running timer untouched
- [x] 3.2 Status lines instead of a dialog: `no gap below "<desc>"` / `no gap above "<desc>"` (`(no description)` when it's empty) and `nothing after the running entry`; nothing at all on the timer panel. Verify with `TestGapNoneShowsStatus` (an entry made to touch its neighbour), `TestGapAboveRunningEntryShowsStatus`, `TestGapKeysIgnoredOnTimerPanel`

## 4. Docs and integration

- [ ] 4.1 Update `docs/REQUIREMENTS.md` (keymap: `o` / `O` in the list; the log-entry section mentions opening on a gap), `README.md` (keys table), and `docs/PLAN.md` (the "Logging past entries" row: `o`/`O` done, citing this change). Verify by reading each key against `proposal.md`'s table
- [ ] 4.2 Run `go test -race ./...` and verify it passes
- [ ] 4.3 Run `go run . -mock` and verify by hand: `o` and `O` on an entry between two others open on the exact gap; logging it fills the gap with the timer still ticking; `o` on touching entries shows the status line; `?` lists `o` / `O`

## Workflow follow-up

- Run superpowers:requesting-code-review once all tasks are checked.
- Archive with `/opsx:archive` after `go test -race ./...` passes.
