package dialogs

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Fixture clock for the date+time field: today is Tue 14 Jul 2026.
var dtToday = time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)

func at(day, hour, min int) time.Time {
	return time.Date(2026, 7, day, hour, min, 0, 0, time.Local)
}

// todayEnd is the usual upper bound: no date past today.
var todayEnd = at(14, 23, 59)

func feedDT(f DateTimeField, keys ...string) (DateTimeField, FieldResult) {
	res := FieldEditing
	for _, k := range keys {
		f, res = f.Update(keyMsg(k))
	}
	return f, res
}

func wantValue(t *testing.T, f DateTimeField, want time.Time) {
	t.Helper()
	if !f.Value().Equal(want) {
		t.Fatalf("value = %s, want %s", f.Value().Format("Mon 02 Jan 15:04"), want.Format("Mon 02 Jan 15:04"))
	}
}

func TestDateTimeFieldOpensOnTime(t *testing.T) {
	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	v := f.View()
	if !strings.Contains(v, "Mon 13 Jul") || !strings.Contains(v, "14:32") {
		t.Fatalf("view = %q, want date and time", v)
	}
	if f.DateActive() || f.Cursor() != 0 {
		t.Fatalf("date active %v cursor %d, want time segment, cursor 0", f.DateActive(), f.Cursor())
	}
}

func TestDateTimeFieldTabSwitchesSegments(t *testing.T) {
	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "l", "tab")
	if !f.DateActive() {
		t.Fatal("tab did not activate the date segment")
	}
	f, _ = feedDT(f, "tab")
	if f.DateActive() || f.Cursor() != 1 {
		t.Fatalf("date active %v cursor %d, want time segment with cursor kept at 1", f.DateActive(), f.Cursor())
	}
	f, _ = feedDT(f, "shift+tab")
	if !f.DateActive() {
		t.Fatal("shift+tab did not switch segments")
	}
}

func TestDateTimeFieldTimeSegmentDelegates(t *testing.T) {
	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "0", "9", "w", "1", "5")
	wantValue(t, f, at(13, 9, 15))
	if !f.Clocked() {
		t.Fatal("typing digits did not mark the clock edited")
	}
	if _, res := feedDT(f, "enter"); res != FieldAccepted {
		t.Fatalf("enter = %v, want accepted", res)
	}
	if _, res := feedDT(f, "esc"); res != FieldReverted {
		t.Fatalf("esc = %v, want reverted", res)
	}
	// A fast burst of runes types each digit.
	g, _ := feedDT(NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil), "0915")
	wantValue(t, g, at(13, 9, 15))
}

func TestDateTimeFieldMotionsAreNotEdits(t *testing.T) {
	f, _ := feedDT(NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil), "l", "w", "h", "$", "^")
	if f.Clocked() {
		t.Fatal("motions marked the clock edited")
	}
}

func TestDateTimeFieldRejectsOutOfRangeTime(t *testing.T) {
	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	f, res := feedDT(f, "2", "7", "enter")
	if res != FieldEditing || f.Err() == "" {
		t.Fatalf("27:32 enter = %v err %q, want still editing with an error", res, f.Err())
	}
	if !strings.Contains(f.View(), "27:32") {
		t.Fatalf("view = %q, want the raw digits kept", f.View())
	}
}

func TestDateTimeFieldStepsDate(t *testing.T) {
	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "tab", "ctrl+x", "ctrl+x", "ctrl+x")
	wantValue(t, f, at(10, 14, 32))
	if !strings.Contains(f.View(), "Fri 10 Jul") {
		t.Fatalf("view = %q, want Fri 10 Jul", f.View())
	}
	f, _ = feedDT(f, "ctrl+a")
	wantValue(t, f, at(11, 14, 32))
	if f.Stepped() != -2 {
		t.Fatalf("stepped = %d, want -2", f.Stepped())
	}
	if f.Clocked() {
		t.Fatal("date steps marked the clock edited")
	}
}

func TestDateTimeFieldDateIgnoresDigitsAndMotions(t *testing.T) {
	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "tab", "5", "h", "l", "w", "b", "$", "^", "0")
	wantValue(t, f, at(13, 14, 32))
	if f.Cursor() != 0 || f.Clocked() {
		t.Fatalf("cursor %d clocked %v, want untouched time segment", f.Cursor(), f.Clocked())
	}
}

func TestDateTimeFieldDateStopsAtToday(t *testing.T) {
	f := NewDateTimeField(at(14, 9, 0), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "tab", "ctrl+a")
	wantValue(t, f, at(14, 9, 0))
	if f.Stepped() != 0 {
		t.Fatalf("stepped = %d, want 0", f.Stepped())
	}
}

func TestDateTimeFieldDateStopsAtLowerBound(t *testing.T) {
	lo := at(7, 0, 0) // a running entry's start: no earlier than 7 days back
	f := NewDateTimeField(at(7, 9, 0), lo, todayEnd, nil)
	f, _ = feedDT(f, "tab", "ctrl+x")
	wantValue(t, f, at(7, 9, 0))
}

func TestDateTimeFieldNoLowerBound(t *testing.T) {
	f := NewDateTimeField(at(1, 9, 0), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "tab", "ctrl+x")
	wantValue(t, f, time.Date(2026, 6, 30, 9, 0, 0, 0, time.Local))
}

func TestDateTimeFieldEnterAndEscWorkOnDateSegment(t *testing.T) {
	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	if _, res := feedDT(f, "tab", "ctrl+x", "enter"); res != FieldAccepted {
		t.Fatalf("enter on date = %v, want accepted", res)
	}
	if _, res := feedDT(f, "tab", "esc"); res != FieldReverted {
		t.Fatalf("esc on date = %v, want reverted", res)
	}
}

func TestDateTimeFieldCarriesForwardAcrossMidnight(t *testing.T) {
	f := NewDateTimeField(at(13, 23, 59), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "w", "ctrl+a")
	wantValue(t, f, at(14, 0, 0))
	if !strings.Contains(f.View(), "Tue 14 Jul") {
		t.Fatalf("view = %q, want the carried date", f.View())
	}
	if !f.Clocked() || f.Stepped() != 0 {
		t.Fatalf("clocked %v stepped %d: a carry is a time edit, not a date step", f.Clocked(), f.Stepped())
	}
	if f.Cursor() != 2 {
		t.Fatalf("cursor = %d, want it kept on the minutes", f.Cursor())
	}
	// Hours carry too.
	g, _ := feedDT(NewDateTimeField(at(13, 23, 30), time.Time{}, todayEnd, nil), "ctrl+a")
	wantValue(t, g, at(14, 0, 30))
}

func TestDateTimeFieldBorrowsBackAcrossMidnight(t *testing.T) {
	f := NewDateTimeField(at(14, 0, 0), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "w", "ctrl+x")
	wantValue(t, f, at(13, 23, 59))
}

func TestDateTimeFieldHoldsAtUpperBound(t *testing.T) {
	f := NewDateTimeField(at(14, 23, 59), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "w", "ctrl+a")
	wantValue(t, f, at(14, 23, 59))
	g, _ := feedDT(NewDateTimeField(at(14, 23, 30), time.Time{}, todayEnd, nil), "ctrl+a")
	wantValue(t, g, at(14, 23, 59)) // an hour step clamps to the bound
}

func TestDateTimeFieldHoldsAtLowerBound(t *testing.T) {
	f := NewDateTimeField(at(7, 0, 0), at(7, 0, 0), todayEnd, nil)
	f, _ = feedDT(f, "w", "ctrl+x")
	wantValue(t, f, at(7, 0, 0))
}

// nextDayBeforeNoon stands in for Edit's stop resolver: a morning time
// lands on the day after the field's date.
func nextDayBeforeNoon(day time.Time, minutes int) time.Time {
	if minutes < 12*60 {
		day = day.AddDate(0, 0, 1)
	}
	return atClock(day, minutes)
}

func TestDateTimeFieldViewShowsResolvedDate(t *testing.T) {
	f := NewDateTimeField(at(13, 23, 50), time.Time{}, todayEnd, nextDayBeforeNoon)
	if !strings.Contains(f.View(), "Mon 13 Jul") {
		t.Fatalf("view = %q, want the opening date before any typing", f.View())
	}
	f, _ = feedDT(f, "0")
	if !strings.Contains(f.View(), "Tue 14 Jul") {
		t.Fatalf("view = %q, want the date to follow the typed time", f.View())
	}
	wantValue(t, f, at(14, 3, 50))
	f, _ = feedDT(f, "^", "2")
	wantValue(t, f, at(13, 23, 50))
}

func TestDateTimeFieldHandSetDateIsNotResolved(t *testing.T) {
	f := NewDateTimeField(at(13, 23, 50), time.Time{}, todayEnd, nextDayBeforeNoon)
	f, _ = feedDT(f, "tab", "ctrl+x", "tab", "0")
	wantValue(t, f, at(12, 3, 50))
}

func TestDateTimeFieldCarryPinsTheDate(t *testing.T) {
	f := NewDateTimeField(at(13, 23, 59), time.Time{}, todayEnd, nextDayBeforeNoon)
	f, _ = feedDT(f, "w", "ctrl+a") // carries to Tue 00:00
	f, _ = feedDT(f, "^", "1")      // 10:00 must stay on Tue, not resolve again
	wantValue(t, f, at(14, 10, 0))
}

func TestDateTimeFieldStepsFromTheResolvedValue(t *testing.T) {
	f := NewDateTimeField(at(13, 23, 50), time.Time{}, todayEnd, nextDayBeforeNoon)
	f, _ = feedDT(f, "0", "0", "1", "0", "ctrl+a") // 00:10 resolves to Tue
	wantValue(t, f, at(14, 0, 11))
}

// Until the time is edited the field holds exactly what it opened with; the
// resolver only places typed times.
func TestDateTimeFieldResolvesOnlyEditedTimes(t *testing.T) {
	f := NewDateTimeField(at(13, 9, 0), time.Time{}, todayEnd, nextDayBeforeNoon)
	wantValue(t, f, at(13, 9, 0))
	f, _ = feedDT(f, "l", "w") // motions are not edits
	wantValue(t, f, at(13, 9, 0))
}

// Review fixes: the field's stored date must always be the date it shows,
// so date and time steps start from what the user sees.

func TestDateTimeFieldDateStepStartsFromShownDate(t *testing.T) {
	f := NewDateTimeField(at(12, 23, 50), time.Time{}, todayEnd, nextDayBeforeNoon)
	f, _ = feedDT(f, "0", "0", "1", "5") // resolves to Mon 13 00:15
	f, _ = feedDT(f, "tab", "ctrl+x")
	wantValue(t, f, at(12, 0, 15))
	f, _ = feedDT(f, "ctrl+a", "ctrl+a")
	wantValue(t, f, at(14, 0, 15))
}

func TestDateTimeFieldStepKeepsShownDate(t *testing.T) {
	f := NewDateTimeField(at(12, 23, 50), time.Time{}, todayEnd, nextDayBeforeNoon)
	f, _ = feedDT(f, "1", "1", "0", "0") // resolves to Mon 13 11:00
	f, _ = feedDT(f, "^", "ctrl+a")      // +1h must stay on Mon, not re-resolve
	wantValue(t, f, at(13, 12, 0))
	f, _ = feedDT(f, "^", "0") // typing again resolves again: 02:00 → next day of base
	wantValue(t, f, at(14, 2, 0))
}

func TestDateTimeFieldOutOfRangeDigitsThenStepCarries(t *testing.T) {
	f := NewDateTimeField(at(13, 9, 0), time.Time{}, todayEnd, nil)
	f, _ = feedDT(f, "2", "7", "1", "0", "ctrl+a") // 27:10 is Tue 03:10
	wantValue(t, f, at(14, 3, 11))
}

// Steps move by real minutes, so a spring-forward gap is stepped over
// rather than looping inside the hour before it.
func TestDateTimeFieldStepsAcrossDSTGap(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no tz database: %v", err)
	}
	saved := time.Local
	time.Local = ny
	t.Cleanup(func() { time.Local = saved })
	day := func(h, m int) time.Time { return time.Date(2026, 3, 8, h, m, 0, 0, time.Local) }
	hi := day(23, 59)

	f, _ := feedDT(NewDateTimeField(day(1, 58), time.Time{}, hi, nil), "w", "ctrl+a", "ctrl+a")
	wantValue(t, f, day(3, 0))
	g, _ := feedDT(NewDateTimeField(day(1, 30), time.Time{}, hi, nil), "ctrl+a")
	wantValue(t, g, day(3, 30))
}

// Only the active segment carries a cursor block: the date while it is
// active, otherwise the time digit under the cursor.
func TestDateTimeFieldOneCursorAtATime(t *testing.T) {
	saved := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(saved) })
	blocks := func(v string) int {
		return strings.Count(v, "\x1b[1;7m") + strings.Count(v, "\x1b[7;1m") + strings.Count(v, "\x1b[7m")
	}

	f := NewDateTimeField(at(13, 14, 32), time.Time{}, todayEnd, nil)
	if n := blocks(f.View()); n != 1 {
		t.Fatalf("time segment: %d cursor blocks in %q, want 1", n, f.View())
	}
	f, _ = feedDT(f, "tab")
	if n := blocks(f.View()); n != 1 {
		t.Fatalf("date segment: %d cursor blocks in %q, want 1", n, f.View())
	}
	if !strings.Contains(f.View(), "14:32") {
		t.Fatalf("view = %q, want the time shown plainly while the date is active", f.View())
	}
}
