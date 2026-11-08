package ui

// Edit-dialog flows (openspec change edit-entry-time-fields, spec
// entry-editing). Fixture ID3 is the finished entry yesterday,
// Mon 13 Jul 12:00–13:00; ID1 is running since Tue 14 Jul 11:50.

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KorboCoder/lazytoggl/internal/ui/panels"
)

// openEditID3 selects the finished entry from yesterday and opens its editor.
func openEditID3(t *testing.T) *driver {
	t.Helper()
	d := newDriver(t)
	d.keys("j", "j", "e")
	d.dialogContains("Edit entry")
	return d
}

func TestEditFormFinishedEntry(t *testing.T) {
	d := openEditID3(t)
	for _, want := range []string{"description", "Yesterday task",
		"start", "Mon 13 Jul 12:00", "stop", "Mon 13 Jul 13:00", "duration", "01:00"} {
		d.dialogContains(want)
	}
}

func TestEditFormRunningEntry(t *testing.T) {
	d := newDriver(t)
	d.key("e") // cursor starts on the running entry
	d.dialogContains("Writing project plan")
	d.dialogContains("Tue 14 Jul 11:50")
	d.dialogLacks("stop")
	d.dialogLacks("duration")
}

func TestEditTabCyclesRowsAndWraps(t *testing.T) {
	d := openEditID3(t)
	d.dialogContains("> description")
	d.key("tab")
	d.dialogContains("> start")
	d.key("down")
	d.dialogContains("> stop")
	d.keys("tab", "tab") // duration, then wrap
	d.dialogContains("> description")
	d.key("shift+tab") // wrap backwards
	d.dialogContains("> duration")
	d.key("up")
	d.dialogContains("> stop")
}

func TestEditEnterOnDescriptionMovesOn(t *testing.T) {
	d := openEditID3(t)
	d.key("enter")
	if d.m.dialog == nil {
		t.Fatal("enter on description closed the dialog")
	}
	d.dialogContains("> start")
	if d.mock.Calls("Update") != 0 {
		t.Fatal("enter on description saved the entry")
	}
}

func TestEditEnterOnTimeRowOpensField(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "tab", "enter") // stop row
	d.dialogContains("─ stop ")   // the field's popover title
	d.dialogContains("ctrl+a")    // field help replaces the form help
}

func TestEditTypingInFieldDoesNotTouchDescription(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "enter", "h", "l", "w", "b")
	d.key("esc") // revert the field
	d.dialogContains("Yesterday task")
	d.dialogLacks("Yesterday taskhlwb")
}

// moveID3StartTo1145 edits ID3's start from 12:00 to 11:45 and accepts it.
func moveID3StartTo1145(d *driver) {
	d.keys("tab", "enter", "1", "1", "4", "5", "enter")
	d.dialogContains("Mon 13 Jul 11:45")
}

func TestEditNothingChangedClosesWithoutRequest(t *testing.T) {
	d := openEditID3(t)
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatal("dialog still open after esc with no changes")
	}
	if n := d.mock.Calls("Update"); n != 0 {
		t.Fatalf("Update calls = %d, want 0", n)
	}
}

func TestEditReviewListsOnlyChangedFields(t *testing.T) {
	d := openEditID3(t)
	moveID3StartTo1145(d)
	d.key("esc")
	d.dialogContains("Save changes?")
	d.dialogContains("12:00 -> 11:45")
	d.dialogContains("01:00 -> 01:15")
	d.dialogLacks("description")
	d.dialogLacks("Yesterday task")
	d.dialogLacks("13:00")
}

func TestEditReviewSave(t *testing.T) {
	d := openEditID3(t)
	moveID3StartTo1145(d)
	d.keys("esc", "y")
	if d.m.dialog != nil {
		t.Fatal("dialog still open after save")
	}
	if n := d.mock.Calls("Update"); n != 1 {
		t.Fatalf("Update calls = %d, want 1", n)
	}
	e, _ := d.entryByID(3)
	if got := e.Start.Format("15:04"); got != "11:45" {
		t.Fatalf("start = %s, want 11:45", got)
	}
	if e.Duration != 75*60 {
		t.Fatalf("duration = %ds, want %d", e.Duration, 75*60)
	}
}

func TestEditReviewEnterAlsoSaves(t *testing.T) {
	d := openEditID3(t)
	moveID3StartTo1145(d)
	d.keys("esc", "enter")
	if n := d.mock.Calls("Update"); n != 1 {
		t.Fatalf("Update calls = %d, want 1", n)
	}
}

func TestEditReviewBackKeepsEdits(t *testing.T) {
	d := openEditID3(t)
	moveID3StartTo1145(d)
	d.keys("esc", "esc")
	d.dialogContains("Edit entry")
	d.dialogContains("Mon 13 Jul 11:45")
	if n := d.mock.Calls("Update"); n != 0 {
		t.Fatalf("Update calls = %d, want 0", n)
	}
}

func TestEditReviewDiscard(t *testing.T) {
	d := openEditID3(t)
	moveID3StartTo1145(d)
	d.keys("esc", "n")
	if d.m.dialog != nil {
		t.Fatal("dialog still open after discard")
	}
	if n := d.mock.Calls("Update"); n != 0 {
		t.Fatalf("Update calls = %d, want 0", n)
	}
	e, _ := d.entryByID(3)
	if got := e.Start.Format("15:04"); got != "12:00" {
		t.Fatalf("start = %s, want 12:00 unchanged", got)
	}
}

// setRow focuses form row n (0 = description) and types hhmm into its field.
func setRow(d *driver, n int, hhmm string) {
	for i := 0; i < n; i++ {
		d.key("tab")
	}
	d.key("enter")
	for _, r := range hhmm {
		d.key(string(r))
	}
	d.key("enter")
}

func TestEditMovingStartKeepsStop(t *testing.T) {
	d := openEditID3(t)
	setRow(d, 1, "1130")
	d.dialogContains("Mon 13 Jul 11:30")
	d.dialogContains("Mon 13 Jul 13:00")
	d.dialogContains("01:30")
}

func TestEditMovingStopKeepsStart(t *testing.T) {
	d := openEditID3(t)
	setRow(d, 2, "1310")
	d.dialogContains("Mon 13 Jul 12:00")
	d.dialogContains("Mon 13 Jul 13:10")
	d.dialogContains("01:10")
}

func TestEditEditingDurationMovesStop(t *testing.T) {
	d := openEditID3(t)
	setRow(d, 3, "0045")
	d.dialogContains("Mon 13 Jul 12:00")
	d.dialogContains("Mon 13 Jul 12:45")
	d.dialogContains("00:45")
}

func TestEditStartPastStopFallsBack(t *testing.T) {
	d := openEditID3(t)
	setRow(d, 1, "1400")
	d.dialogContains("Mon 13 Jul 12:59")
	d.dialogContains("00:01")
}

func TestEditStopBeforeStartFallsBack(t *testing.T) {
	d := openEditID3(t)
	setRow(d, 2, "1100")
	d.dialogContains("Mon 13 Jul 12:01")
	d.dialogContains("00:01")
}

func TestEditZeroDurationFallsBackToOneMinute(t *testing.T) {
	d := openEditID3(t)
	setRow(d, 3, "0000")
	d.dialogContains("Mon 13 Jul 12:01")
	d.dialogContains("00:01")
}

func TestEditRunningStartInFutureFallsBack(t *testing.T) {
	d := newDriver(t) // fixedNow is 12:00; ID1 is running since 11:50
	d.key("e")
	setRow(d, 1, "1300")
	d.dialogContains("Tue 14 Jul 11:59")
}

func TestEditEscRevertsField(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "enter", "0", "9", "esc")
	d.dialogContains("Mon 13 Jul 12:00")
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatal("reverted field still counted as a change")
	}
}

// openEditID3WithSeconds gives ID3 a start of 12:00:17 before opening it.
func openEditID3WithSeconds(t *testing.T) *driver {
	t.Helper()
	d := newDriver(t)
	for i := range d.m.entries {
		if d.m.entries[i].ID == 3 {
			d.m.entries[i].Start = d.m.entries[i].Start.Add(17 * time.Second)
		}
	}
	d.keys("j", "j", "e")
	d.dialogContains("Edit entry")
	return d
}

func TestEditOnlyDescriptionKeepsSeconds(t *testing.T) {
	d := openEditID3WithSeconds(t)
	d.keys("ctrl+u", "Renamed", "esc", "y")
	e, _ := d.entryByID(3)
	if e.Description != "Renamed" {
		t.Fatalf("description = %q, want Renamed", e.Description)
	}
	if e.Start.Second() != 17 {
		t.Fatalf("start = %s, want seconds kept at :17", e.Start.Format("15:04:05"))
	}
}

func TestEditAcceptingAnUnchangedFieldIsNotAChange(t *testing.T) {
	d := openEditID3WithSeconds(t)
	d.keys("tab", "enter", "enter") // open start, accept as-is
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatalf("accepting an unchanged time counted as a change:\n%s", d.m.dialog.View())
	}
}

func TestEditHelpLinesMatchEachState(t *testing.T) {
	d := openEditID3(t)
	d.dialogContains("enter edit time")
	d.dialogContains("esc done")
	d.dialogLacks("enter save")

	d.keys("tab", "enter")
	d.dialogContains("ctrl+a/x")
	d.dialogContains("esc revert")
	d.dialogContains("tab date/time")

	d.keys("1", "1", "enter", "esc")
	d.dialogContains("y/enter save")
	d.dialogContains("esc back")
	d.dialogContains("n discard")
}

// The focus marker is the only "> " in the form; the description input
// must not carry a prompt that looks like one.
func TestEditOnlyFocusedRowShowsMarker(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "tab", "tab") // duration
	d.dialogContains("> duration")
	d.dialogLacks("> Yesterday task")
}

// openEditID3Shifted opens ID3 (12:00–13:00) after nudging its start and
// stop by the given seconds, like a timer-recorded entry.
func openEditID3Shifted(t *testing.T, startSec, stopSec int) *driver {
	t.Helper()
	d := newDriver(t)
	for i := range d.m.entries {
		if d.m.entries[i].ID == 3 {
			d.m.entries[i].Start = d.m.entries[i].Start.Add(time.Duration(startSec) * time.Second)
			d.m.entries[i].Stop = d.m.entries[i].Stop.Add(time.Duration(stopSec) * time.Second)
		}
	}
	d.keys("j", "j", "e")
	d.dialogContains("Edit entry")
	return d
}

func savedID3(t *testing.T, d *driver) (start, stop time.Time) {
	t.Helper()
	d.keys("esc", "y")
	e, _ := d.entryByID(3)
	return e.Start, e.Stop
}

func TestEditStopJustPastSecondsStartKeepsOneMinute(t *testing.T) {
	d := openEditID3Shifted(t, 17, 0) // 12:00:17–13:00:00
	setRow(d, 2, "1201")              // 43s after start
	start, stop := savedID3(t, d)
	if got := stop.Sub(start); got != time.Minute {
		t.Fatalf("length = %s, want exactly 1m (stop %s)", got, stop.Format("15:04:05"))
	}
}

func TestEditStartJustBeforeSecondsStopKeepsOneMinute(t *testing.T) {
	d := openEditID3Shifted(t, 0, 40) // 12:00:00–13:00:40
	setRow(d, 1, "1300")              // 40s before stop
	start, stop := savedID3(t, d)
	if got := stop.Sub(start); got != time.Minute {
		t.Fatalf("length = %s, want exactly 1m (start %s)", got, start.Format("15:04:05"))
	}
}

func TestEditReviewSeparatesDescriptionLabel(t *testing.T) {
	d := openEditID3(t)
	d.keys("ctrl+u", "Renamed", "esc")
	d.dialogContains("Yesterday task -> Renamed")
	d.dialogLacks("descriptionYesterday")
}

func TestEditSurroundingWhitespaceIsNotAChange(t *testing.T) {
	d := newDriver(t)
	for i := range d.m.entries {
		if d.m.entries[i].ID == 3 {
			d.m.entries[i].Description = "Yesterday task "
		}
	}
	d.keys("j", "j", "e", "esc")
	if d.m.dialog != nil {
		t.Fatalf("untouched description with trailing space counted as a change:\n%s", d.m.dialog.View())
	}
}

func TestEditSavingOtherFieldsKeepsDescriptionVerbatim(t *testing.T) {
	d := newDriver(t)
	for i := range d.m.entries {
		if d.m.entries[i].ID == 3 {
			d.m.entries[i].Description = "Yesterday task "
		}
	}
	d.keys("j", "j", "e")
	setRow(d, 1, "1130")
	d.keys("esc")
	d.dialogLacks("description")
	d.key("y")
	e, _ := d.entryByID(3)
	if e.Description != "Yesterday task " {
		t.Fatalf("description = %q, want it saved verbatim", e.Description)
	}
}

func TestEditRetypingOriginalStartRestoresIt(t *testing.T) {
	d := openEditID3Shifted(t, 17, 0) // start 12:00:17
	setRow(d, 1, "1100")
	setRow(d, 0, "1200") // focus is still on start
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatalf("retyped original start counted as a change:\n%s", d.m.dialog.View())
	}
}

func TestEditRetypingOriginalDurationRestoresStop(t *testing.T) {
	d := openEditID3Shifted(t, 17, 40) // 12:00:17–13:00:40, shows 01:00
	setRow(d, 3, "0030")
	setRow(d, 0, "0100") // focus is still on duration
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatalf("retyped original duration counted as a change:\n%s", d.m.dialog.View())
	}
}

func TestEditRetypingRecentRunningStartKeepsIt(t *testing.T) {
	d := newDriver(t) // fixedNow 12:00; move ID1's start to 11:59:40
	for i := range d.m.entries {
		if d.m.entries[i].ID == 1 {
			d.m.entries[i].Start = fixedNow.Add(-20 * time.Second)
		}
	}
	d.key("e")
	setRow(d, 1, "1130")
	setRow(d, 0, "1159")
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatalf("retyped original running start counted as a change:\n%s", d.m.dialog.View())
	}
}

// An out-of-range error shows in the form's help area, not in the popover,
// so the popover stays narrow and the row it edits stays readable.
func TestEditFieldErrorKeepsRowVisible(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "enter", "2", "7", "enter")
	d.dialogContains("00–23")
	d.dialogContains("Mon 13 Jul 12:00")
}

// Date editing (openspec change edit-entry-dates). The start and stop
// fields carry the date beside the time; tab switches between them.

func TestEditStartFieldShowsDate(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "enter")
	d.dialogContains("─ start ")
	d.dialogContains("Mon 13 Jul  12:00") // the popover: date, two spaces, time
	d.dialogContains("Mon 13 Jul 12:00")  // the row it edits stays readable
	d.dialogContains("tab date/time")
	d.key("tab")
	d.dialogContains("ctrl+a/x +/-1 day")
}

func TestEditDurationFieldIgnoresTab(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "tab", "tab", "enter") // duration
	d.dialogLacks("tab date/time")
	d.keys("tab", "0", "0", "4", "5", "enter")
	d.dialogContains("> duration")
	d.dialogContains("00:45")
	d.dialogContains("Mon 13 Jul 12:45")
}

// jul is a wall-clock time in the fixture's July 2026.
func jul(day, hour, min int) time.Time {
	return time.Date(2026, 7, day, hour, min, 0, 0, time.Local)
}

// openEditID3At moves ID3 to start–stop and opens its editor.
func openEditID3At(t *testing.T, start, stop time.Time) *driver {
	t.Helper()
	d := newDriver(t)
	for i := range d.m.entries {
		if d.m.entries[i].ID == 3 {
			d.m.entries[i].Start, d.m.entries[i].Stop = start, stop
			d.m.entries[i].Duration = int64(stop.Sub(start).Seconds())
		}
	}
	d.keys("j", "j", "e")
	d.dialogContains("Edit entry")
	return d
}

// typeInto opens form row n's field, types the digits, and leaves it open.
func typeInto(d *driver, n int, digits string) {
	for i := 0; i < n; i++ {
		d.key("tab")
	}
	d.key("enter")
	for _, r := range digits {
		d.key(string(r))
	}
}

func TestEditTypedStopCrossesMidnight(t *testing.T) {
	d := openEditID3At(t, jul(13, 23, 30), jul(13, 23, 50))
	typeInto(d, 2, "0015")
	d.dialogContains("Tue 14 Jul  00:15") // the field shows the next day live
	d.key("enter")
	d.dialogContains("Tue 14 Jul 00:15")
	d.dialogContains("00:45")
}

func TestEditTypedStartCrossesMidnight(t *testing.T) {
	d := openEditID3At(t, jul(13, 0, 15), jul(13, 1, 0))
	setRow(d, 1, "2330")
	d.dialogContains("Sun 12 Jul 23:30")
	d.dialogContains("Mon 13 Jul 01:00")
	d.dialogContains("01:30")
}

func TestEditTypedStopBackToOneDay(t *testing.T) {
	d := openEditID3At(t, jul(13, 23, 30), jul(14, 0, 45))
	setRow(d, 2, "2350")
	d.dialogContains("Mon 13 Jul 23:50")
	d.dialogContains("00:20")
}

func TestEditRunningTypedStartCrossesMidnight(t *testing.T) {
	d := newDriver(t)
	mm, _ := d.m.Update(tickMsg(jul(14, 0, 10)))
	d.m = mm.(Model)
	for i := range d.m.entries {
		if d.m.entries[i].ID == 1 {
			d.m.entries[i].Start = jul(14, 0, 5)
		}
	}
	d.key("e") // the running entry
	setRow(d, 1, "2350")
	d.dialogContains("Mon 13 Jul 23:50")
}

func TestEditTypedTimeOver12hStaysOnItsDay(t *testing.T) {
	d := openEditID3At(t, jul(13, 14, 32), jul(13, 15, 47))
	setRow(d, 2, "1400")
	d.dialogContains("Mon 13 Jul 14:33") // same day, then the 1-minute rule
	d.dialogContains("00:01")
}

func TestEditHandSetDateIsKept(t *testing.T) {
	d := openEditID3At(t, jul(13, 20, 0), jul(13, 21, 0))
	d.keys("tab", "tab", "enter", "tab", "ctrl+a", "tab", "2", "1", "3", "0", "enter")
	d.dialogContains("Tue 14 Jul 21:30")
}

// Opening and accepting the start of a long (30h) entry changes nothing,
// even though its clock time also occurs within 12h of stop.
func TestEditAcceptingLongEntryStartUnchanged(t *testing.T) {
	d := openEditID3At(t, jul(12, 8, 0), jul(13, 14, 0))
	d.keys("tab", "enter")
	d.dialogContains("Sun 12 Jul  08:00") // the field opens on the real date
	d.keys("enter", "esc")
	if d.m.dialog != nil {
		t.Fatalf("accepting an untouched start counted as a change:\n%s", d.m.dialog.View())
	}
}

func TestEditStartDateMovesWholeEntry(t *testing.T) {
	d := openEditID3(t) // Mon 13 Jul 12:00–13:00
	d.keys("tab", "enter", "tab", "ctrl+x", "enter")
	d.dialogContains("Sun 12 Jul 12:00")
	d.dialogContains("Sun 12 Jul 13:00")
	d.dialogContains("01:00")
}

func TestEditStartDateAndTimeTogether(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "enter", "tab", "ctrl+x", "tab", "1", "1", "0", "0", "enter")
	d.dialogContains("Sun 12 Jul 11:00")
	d.dialogContains("Sun 12 Jul 13:00")
	d.dialogContains("02:00")
}

func TestEditStopDateMovesOnlyStop(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "tab", "enter", "tab", "ctrl+a", "enter")
	d.dialogContains("Mon 13 Jul 12:00")
	d.dialogContains("Tue 14 Jul 13:00")
	d.dialogContains("25:00")
}

func TestEditCarryAcrossMidnightKeepsStop(t *testing.T) {
	d := openEditID3At(t, jul(13, 0, 0), jul(13, 1, 0))
	d.keys("tab", "enter", "w", "ctrl+x", "enter")
	d.dialogContains("Sun 12 Jul 23:59")
	d.dialogContains("Mon 13 Jul 01:00")
	d.dialogContains("01:01")
}

func TestEditMovingDateKeepsSeconds(t *testing.T) {
	d := openEditID3Shifted(t, 17, 5) // 12:00:17–13:00:05
	d.keys("tab", "enter", "tab", "ctrl+x", "enter")
	start, stop := savedID3(t, d)
	if want := time.Date(2026, 7, 12, 12, 0, 17, 0, time.Local); !start.Equal(want) {
		t.Fatalf("start = %s, want %s", start, want)
	}
	if want := time.Date(2026, 7, 12, 13, 0, 5, 0, time.Local); !stop.Equal(want) {
		t.Fatalf("stop = %s, want %s", stop, want)
	}
}

func TestEditStopDateKeepsSeconds(t *testing.T) {
	d := openEditID3Shifted(t, 0, 5) // 12:00:00–13:00:05
	d.keys("tab", "tab", "enter", "tab", "ctrl+a", "enter")
	_, stop := savedID3(t, d)
	if want := time.Date(2026, 7, 14, 13, 0, 5, 0, time.Local); !stop.Equal(want) {
		t.Fatalf("stop = %s, want %s", stop, want)
	}
}

func TestEditRunningStartDateBoundedToAWeek(t *testing.T) {
	d := newDriver(t) // ID1 running since Tue 14 Jul 11:50
	d.keys("e", "tab", "enter", "tab")
	for i := 0; i < 8; i++ {
		d.key("ctrl+x")
	}
	d.key("enter")
	d.dialogContains("Tue 07 Jul 11:50")
}

func TestEditReviewShowsDatesWhenMoved(t *testing.T) {
	d := openEditID3(t)
	d.keys("tab", "enter", "tab", "ctrl+x", "enter", "esc")
	d.dialogContains("Save changes?")
	d.dialogContains("Mon 13 Jul 12:00 -> Sun 12 Jul 12:00")
	d.dialogContains("Mon 13 Jul 13:00 -> Sun 12 Jul 13:00")
	d.dialogLacks("duration")
}

// stepStartDate opens ID3's start field and steps its date by n days.
func stepStartDate(d *driver, n int) {
	d.keys("tab", "enter", "tab")
	k := "ctrl+a"
	if n < 0 {
		k, n = "ctrl+x", -n
	}
	for i := 0; i < n; i++ {
		d.key(k)
	}
	d.key("enter")
}

// The fetch window starts Mon 06 Jul 12:00 (fixedNow − 192h).
func TestEditReviewFlagsOutsideList(t *testing.T) {
	d := openEditID3(t)
	stepStartDate(d, -8) // Sun 05 Jul
	d.key("esc")
	d.dialogContains("Sun 05 Jul 12:00")
	d.dialogContains("(outside list)")
}

func TestEditReviewNoFlagInsideList(t *testing.T) {
	d := openEditID3(t)
	stepStartDate(d, -6) // Tue 07 Jul
	d.key("esc")
	d.dialogContains("Tue 07 Jul 12:00")
	d.dialogLacks("outside list")
}

// listed returns the entries list as day labels and entry IDs, in order.
func listed(d *driver) (headers []string, ids []int64) {
	for _, r := range panels.BuildRows(d.m.entries, d.m.now) {
		if r.Kind == panels.RowHeader {
			headers = append(headers, r.Label)
		} else {
			ids = append(ids, r.Entry.ID)
		}
	}
	return headers, ids
}

func TestEditMovedEntryListedUnderItsNewDay(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "e") // ID2, today 10:00
	stepStartDate(d, -2)
	d.keys("esc", "y")
	headers, ids := listed(d)
	if want := []string{"Today", "Yesterday", "Sun, Jul 12"}; !slices.Equal(headers, want) {
		t.Fatalf("headers = %v, want %v", headers, want)
	}
	if want := []int64{1, 3, 2}; !slices.Equal(ids, want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	wantSelected(t, d, 2)
}

func TestEditReorderWithinDay(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "e") // ID2, today 10:00–10:15; ID1 runs since 11:50
	d.keys("tab", "tab", "enter", "1", "1", "5", "8", "enter")
	d.keys("shift+tab", "enter", "1", "1", "5", "5", "enter")
	d.keys("esc", "y")
	headers, ids := listed(d)
	if want := []string{"Today", "Yesterday"}; !slices.Equal(headers, want) {
		t.Fatalf("headers = %v, want %v", headers, want)
	}
	if want := []int64{2, 1, 3}; !slices.Equal(ids, want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	wantSelected(t, d, 2)
}

// wantSelected asserts the list cursor is on entry id.
func wantSelected(t *testing.T, d *driver, id int64) {
	t.Helper()
	r, ok := d.m.entriesPanel.Selected(panels.BuildRows(d.m.entries, d.m.now))
	if !ok || r.Kind != panels.RowEntry || r.Entry.ID != id {
		t.Fatalf("selected = %+v, want entry %d", r, id)
	}
}

func TestEditSaveOutsideListShowsStatus(t *testing.T) {
	d := openEditID3(t)
	stepStartDate(d, -8) // Sun 05 Jul, before the window
	d.keys("esc", "y")
	for _, want := range []string{"Sun, Jul 5", "outside the 8-day list", "drops off on refresh"} {
		if !strings.Contains(d.m.status, want) {
			t.Fatalf("status = %q, want it to contain %q", d.m.status, want)
		}
	}
	if _, ok := d.entryByID(3); !ok {
		t.Fatal("entry left the list before a refresh")
	}
	headers, _ := listed(d)
	if !slices.Contains(headers, "Sun, Jul 5") {
		t.Fatalf("headers = %v, want the entry listed under Sun, Jul 5", headers)
	}
}

func TestEditSaveInsideListPlainStatus(t *testing.T) {
	d := openEditID3(t)
	stepStartDate(d, -6) // Tue 07 Jul, inside the window
	d.keys("esc", "y")
	if d.m.status != "entry saved" {
		t.Fatalf("status = %q, want %q", d.m.status, "entry saved")
	}
}

// Review fixes: steps work from the date the field shows.

func TestEditTypedStartThenStepDate(t *testing.T) {
	d := openEditID3At(t, jul(13, 0, 15), jul(13, 1, 0))
	d.keys("tab", "enter", "2", "3", "3", "0") // Sun 12 Jul 23:30
	d.keys("tab", "ctrl+x")
	d.dialogContains("Sat 11 Jul  23:30")
	d.key("enter")
	d.dialogContains("Sat 11 Jul 23:30")
	d.dialogContains("Sun 12 Jul 01:00") // a date step moves the whole entry
	d.dialogContains("01:30")
}

func TestEditHourStepsPastResolvedLimitKeepDate(t *testing.T) {
	d := openEditID3At(t, jul(12, 23, 30), jul(12, 23, 50))
	d.keys("tab", "tab", "enter", "0", "0", "1", "5", "^") // Mon 13 Jul 00:15
	for i := 0; i < 12; i++ {
		d.key("ctrl+a")
	}
	d.dialogContains("Mon 13 Jul  12:15")
}

func TestEditMinuteStepOnLongStopKeepsDate(t *testing.T) {
	d := openEditID3At(t, jul(12, 20, 0), jul(13, 21, 0))
	d.keys("tab", "tab", "enter", "w", "ctrl+a")
	d.dialogContains("Mon 13 Jul  21:01")
}

// An entry the window has slid past, but whose start wasn't edited, wasn't
// "moved": the status stays plain.
func TestEditSaveUnmovedEntryPlainStatus(t *testing.T) {
	d := openEditID3At(t, jul(6, 12, 30), jul(6, 13, 0)) // inside the window at 12:00
	mm, _ := d.m.Update(tickMsg(jul(14, 13, 0)))         // window now starts Jul 6 13:00
	d.m = mm.(Model)
	d.keys("ctrl+u", "Renamed", "esc", "y")
	if d.m.status != "entry saved" {
		t.Fatalf("status = %q, want %q", d.m.status, "entry saved")
	}
}
