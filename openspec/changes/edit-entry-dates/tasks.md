# Tasks

Test fixtures: `fixedNow` is Tue 14 Jul 2026 12:00, so the fetch window starts Mon 06 Jul
12:00. ID3 is finished, Mon 13 Jul 12:00–13:00. ID1 is running since Tue 14 Jul 11:50.
Overnight cases shift ID3 with a helper like `openEditID3Shifted`.

## 1. Fetch window in `internal/toggl`

- [ ] 1.1 Export `RecentEntriesLookback` and add `RecentEntriesSince(now)`; have `Client.RecentEntries` build `start_date` from it. Verify by extending `TestRecentEntriesSortsNewestFirstAndSendsLookbackWindow` (`client_test.go`) to pin `start_date` to exactly `now − 192h`
- [ ] 1.2 Make `Mock.RecentEntries` return only entries starting within `[RecentEntriesSince(now), now]`, plus the running entry. Verify with a new `TestMockRecentEntriesFiltersToWindow` in `mock_test.go`; existing `TestRecentEntriesNewestFirst` and `TestRefreshReloadsFromService` still pass
- [ ] 1.3 Update `docs/ARCHITECTURE.md` (Recent entries row) to name the exported lookback and the mock's matching filter. Verify by reading the row against the code

## 2. `DateTimeField` widget

- [ ] 2.1 Add `TimeField.SetValue(minutes)` that keeps the cursor. Verify with `TestTimeFieldSetValueKeepsCursor` in `timefield_test.go`; all existing `TimeField` tests still pass
- [ ] 2.2 Create `internal/ui/dialogs/datetimefield.go` with date and time segments: it opens on the time segment, `tab`/`shift+tab` switch segments and keep the clock cursor, and keys other than `ctrl+a`/`ctrl+x` on the time segment go to `TimeField` (typing, motions, range error, `enter`, `esc`, rune bursts). Verify with `TestDateTimeFieldOpensOnTime`, `TestDateTimeFieldTabSwitchesSegments`, `TestDateTimeFieldTimeSegmentDelegates`, `TestDateTimeFieldRejectsOutOfRangeTime` in a new `datetimefield_test.go`
- [ ] 2.3 Date segment: `ctrl+a`/`ctrl+x` step ±1 day within `[lo, hi]`, digits and motions are ignored, and `Stepped()` reports the net days. Verify with `TestDateTimeFieldStepsDate`, `TestDateTimeFieldDateIgnoresDigitsAndMotions`, `TestDateTimeFieldDateStopsAtToday`, `TestDateTimeFieldDateStopsAtLowerBound`, `TestDateTimeFieldNoLowerBound`
- [ ] 2.4 Time-segment carry: `ctrl+a`/`ctrl+x` step the full timestamp with carry into the date, clamp to `[lo, hi]`, and set `Clocked()` (and `dated` when the day changes). Verify with `TestDateTimeFieldCarriesForwardAcrossMidnight` (`Mon 13 Jul 23:59 → Tue 14 Jul 00:00`), `TestDateTimeFieldBorrowsBackAcrossMidnight`, `TestDateTimeFieldHoldsAtUpperBound`
- [ ] 2.5 Resolver: `Value()` and `View()` use the resolver until the date is changed by hand. Verify with `TestDateTimeFieldViewShowsResolvedDate` (the shown date changes as digits are typed) and `TestDateTimeFieldHandSetDateIsNotResolved`

## 3. Edit dialog uses the date+time field

- [ ] 3.1 Open a `DateTimeField` for start and stop (bounds: today 23:59 at the top, and for a running entry's start, today − 7 days at 00:00 at the bottom), and keep `TimeField` for duration. Show the date in the popup and add `tab date/time` to the field help. Verify in `edit_test.go`: `TestEditStartFieldShowsDate`, `TestEditDurationFieldIgnoresTab`, extend `TestEditHelpLinesMatchEachState`; existing `TestEditEnterOnTimeRowOpensField` and `TestEditFieldErrorKeepsRowVisible` still pass
- [ ] 3.2 Supply the stop and start resolvers (first occurrence after start, last before stop or `now`, within 12h and the bounds, else the field's date), then run the existing 1-minute and running-entry rules. Verify in `edit_test.go`: `TestEditTypedStopCrossesMidnight`, `TestEditTypedStartCrossesMidnight`, `TestEditTypedStopBackToOneDay`, `TestEditRunningTypedStartCrossesMidnight`, `TestEditTypedTimeOver12hStaysOnItsDay`, `TestEditHandSetDateIsKept`; all existing fallback tests (`TestEditStartPastStopFallsBack`, `TestEditStopBeforeStartFallsBack`, `TestEditRunningStartInFutureFallsBack`, …) still pass unchanged
- [ ] 3.3 `applyField`: shift stop by `Stepped()` days for a finished entry's start, use `AddDate` when only the date was stepped (keeps seconds), and use the field's timestamp otherwise. Verify in `edit_test.go`: `TestEditStartDateMovesWholeEntry`, `TestEditStartDateAndTimeTogether`, `TestEditStopDateMovesOnlyStop` (duration `25:00`), `TestEditCarryAcrossMidnightKeepsStop`, `TestEditMovingDateKeepsSeconds`, `TestEditRunningStartDateBoundedToAWeek`
- [ ] 3.4 Update `docs/REQUIREMENTS.md`: feature item 7 (dates are editable), the edit-dialog keymap (`tab`/`shift+tab` in the field, date-segment `ctrl+a`/`ctrl+x`, carry across midnight), and the editing-rules paragraph (start date moves the entry, the 12h midnight resolution, date bounds). Verify by reading each new key against `proposal.md`'s keybinding table

## 4. Review shows dates and the window flag

- [ ] 4.1 In `changes()`, show `Mon 02 Jan HH:MM` on both sides of a start or stop line when its date changed, and `HH:MM` otherwise. Verify with `TestEditReviewShowsDatesWhenMoved` in `edit_test.go`; `TestEditReviewListsOnlyChangedFields` still passes
- [ ] 4.2 Add a muted `(outside list)` line under the start change when the new start is before `RecentEntriesSince(now)`. Verify with `TestEditReviewFlagsOutsideList` (ID3 stepped back to Sun 05 Jul) and `TestEditReviewNoFlagInsideList` (stepped to Tue 07 Jul)

## 5. List placement and save status

- [ ] 5.1 Make `replaceEntry` re-sort `m.entries` by start, newest first (stable). Verify with `TestEditMovedEntryListedUnderItsNewDay` and `TestEditReorderWithinDay` in `edit_test.go`, asserting the order of `panels.BuildRows` output and a single header per day
- [ ] 5.2 Add `panels.Entries.SelectID(rows, id)` and call it on `entryUpdatedMsg`, so the cursor follows the edited entry. Verify by asserting the selected row's entry ID in both 5.1 tests
- [ ] 5.3 On `entryUpdatedMsg`, when the saved start is before `RecentEntriesSince(m.now)`, set the status to the outside-list message and keep the entry listed. Verify with `TestEditSaveOutsideListShowsStatus` (status names the day and "drops off on refresh", entry still in `m.entries`) and `TestEditSaveInsideListPlainStatus`
- [ ] 5.4 Add a `docs/PLAN.md` decisions-log row for date editing (date segment inside the field via `tab`, start date moves the entry, 12h midnight resolution, moving outside the window allowed with a warning, re-sort fix), citing this change. Update the "Editing times" row's "dates not editable yet". Verify by reading the log

## 6. Integration check

- [ ] 6.1 Run `go test -race ./...` and verify that it passes
- [ ] 6.2 Run `go run . -mock` and verify by hand: an overnight stop typed as `00:15` shows the next day live; `tab` → `ctrl+x` moves an entry to yesterday and it regroups with the cursor on it; stepping back 9 days shows `(outside list)` and the status message, and the entry drops off after `r`

## Workflow follow-up

- Run superpowers:requesting-code-review once all tasks are checked.
- Archive with `/opsx:archive` after `go test -race ./...` passes.
