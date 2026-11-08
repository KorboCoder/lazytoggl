package dialogs

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// DateTimeField edits a start or stop: a date segment beside the HH:MM
// TimeField. It opens on the time, and tab switches segments, so the date is
// always visible but never in the way of editing a time.
type DateTimeField struct {
	day      time.Time // midnight of the field's date
	clock    TimeField
	lo, hi   time.Time // bounds; a zero lo means no lower bound
	dateSeg  bool      // the date segment is active
	clockedV bool      // the time segment was edited (a digit or a step)
	dated    bool      // the date was changed by hand
	stepped  int       // net days stepped on the date segment
	resolve  func(day time.Time, minutes int) time.Time
}

// NewDateTimeField opens a field on t. lo and hi bound the value (a zero lo
// means no lower bound). resolve, if set, places a typed time on a day of
// its choosing until the user changes the date by hand.
func NewDateTimeField(t time.Time, lo, hi time.Time, resolve func(day time.Time, minutes int) time.Time) DateTimeField {
	return DateTimeField{
		day:     midnight(t),
		clock:   NewTimeField(minutesOfDay(t), 0, dayMinutes-1),
		lo:      lo,
		hi:      hi,
		resolve: resolve,
	}
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// Value is the field's timestamp, zero seconds. Once the time is edited, the
// resolver chooses its day, until the date is changed by hand. Typed digits are
// raw until enter, so it is only meaningful once the clock is in range.
func (f DateTimeField) Value() time.Time {
	if f.resolve != nil && f.clockedV && !f.dated && f.clock.valid() {
		return f.resolve(f.day, f.clock.Value())
	}
	return atClock(f.day, f.clock.Value())
}

func (f DateTimeField) DateActive() bool { return f.dateSeg }

// Cursor is the time segment's digit cursor (0–3).
func (f DateTimeField) Cursor() int { return f.clock.Cursor() }

// Clocked reports whether the time segment was edited.
func (f DateTimeField) Clocked() bool { return f.clockedV }

// Stepped is the net number of days stepped on the date segment.
func (f DateTimeField) Stepped() int { return f.stepped }

func (f DateTimeField) Err() string { return f.clock.Err() }

// stepDay moves the date by one day, unless that leaves the bounds.
func (f *DateTimeField) stepDay(sign int) {
	day := f.day.AddDate(0, 0, sign)
	if day.After(midnight(f.hi)) || (!f.lo.IsZero() && day.Before(midnight(f.lo))) {
		return
	}
	f.day = day
	f.stepped += sign
	f.dated = true
}

func (f DateTimeField) Update(k tea.KeyMsg) (DateTimeField, FieldResult) {
	switch k.String() {
	case "tab", "shift+tab":
		f.dateSeg = !f.dateSeg
		return f, FieldEditing
	}
	if f.dateSeg {
		switch k.String() {
		case "ctrl+a":
			f.stepDay(1)
		case "ctrl+x":
			f.stepDay(-1)
		case "enter", "esc":
			return f.updateClock(k)
		}
		return f, FieldEditing
	}
	switch k.String() {
	case "ctrl+a":
		f.stepClock(1)
		return f, FieldEditing
	case "ctrl+x":
		f.stepClock(-1)
		return f, FieldEditing
	}
	return f.updateClock(k)
}

// stepClock adds one unit of the part under the cursor (an hour on the
// hours, a minute on the minutes), carrying across midnight into the date,
// then clamps to the bounds.
func (f *DateTimeField) stepClock(sign int) {
	unit := 1
	if f.clock.Cursor() < 2 {
		unit = 60
	}
	// time.Date normalizes minutes past either end of the day into the
	// neighbouring date, on the wall clock.
	from := f.Value()
	v := time.Date(from.Year(), from.Month(), from.Day(), 0, minutesOfDay(from)+sign*unit, 0, 0, time.Local)
	if v.After(f.hi) {
		v = f.hi
	}
	if !f.lo.IsZero() && v.Before(f.lo) {
		v = f.lo
	}
	// Only a carry changes the date by hand; otherwise the resolver keeps
	// choosing the day, so a step and a typed digit agree.
	if day := midnight(v); !day.Equal(midnight(from)) {
		f.day = day
		f.dated = true
	}
	f.clock.SetValue(minutesOfDay(v))
	f.clockedV = true
}

func (f DateTimeField) updateClock(k tea.KeyMsg) (DateTimeField, FieldResult) {
	before := f.clock.String()
	var res FieldResult
	f.clock, res = f.clock.Update(k)
	if f.clock.String() != before {
		f.clockedV = true
	}
	return f, res
}

func (f DateTimeField) View() string {
	date := f.Value().Format(dateLayout)
	if !f.clock.valid() {
		date = f.day.Format(dateLayout)
	}
	if f.dateSeg {
		date = theme.FieldCursorStyle.Render(date)
	}
	return date + "  " + f.clock.View()
}
