package ui

// Edit-dialog flows (openspec change edit-entry-time-fields, spec
// entry-editing). Fixture ID3 is the finished entry yesterday,
// Mon 13 Jul 12:00–13:00; ID1 is running since Tue 14 Jul 11:50.

import (
	"testing"
	"time"
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
