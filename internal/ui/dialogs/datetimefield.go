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
	// typed is set by a typed digit and cleared by a step: only a typed
	// time is placed by the resolver; a step's result is where it lands.
	typed   bool
	resolve func(day time.Time, minutes int) time.Time
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
	if f.resolve != nil && f.typed && !f.dated && f.clock.valid() {
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

// stepDay moves the shown date by one day, unless that leaves the bounds.
func (f *DateTimeField) stepDay(sign int) {
	base := f.day
	if f.clock.valid() {
		base = midnight(f.Value()) // the date shown, which a typed time may have moved
	}
	day := base.AddDate(0, 0, sign)
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
	// Step from the value shown, by real minutes, so a DST gap is stepped
	// over and raw out-of-range digits (27:10) carry like any other time.
	from := f.Value()
	v := from.Add(time.Duration(sign*unit) * time.Minute)
	if v.After(f.hi) {
		v = f.hi
	}
	if !f.lo.IsZero() && v.Before(f.lo) {
		v = f.lo
	}
	// The step's result is pinned; crossing midnight counts as changing
	// the date by hand.
	if !midnight(v).Equal(midnight(from)) {
		f.dated = true
	}
	f.day = midnight(v)
	f.clock.SetValue(minutesOfDay(v))
	f.clockedV = true
	f.typed = false
}

func (f DateTimeField) updateClock(k tea.KeyMsg) (DateTimeField, FieldResult) {
	before := f.clock.String()
	var res FieldResult
	f.clock, res = f.clock.Update(k)
	if f.clock.String() != before {
		f.clockedV = true
		f.typed = true
	}
	return f, res
}

func (f DateTimeField) View() string {
	date := f.Value().Format(dateLayout)
	if !f.clock.valid() {
		date = f.day.Format(dateLayout)
	}
	// One cursor at a time, on the active segment.
	if f.dateSeg {
		return theme.FieldCursorStyle.Render(date) + "  " + f.clock.String()
	}
	return date + "  " + f.clock.View()
}
