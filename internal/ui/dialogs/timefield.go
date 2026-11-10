package dialogs

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// FieldResult reports what a key did to a TimeField.
type FieldResult int

const (
	FieldEditing  FieldResult = iota // still open
	FieldAccepted                    // enter on a valid value
	FieldReverted                    // esc: discard the field's edits
)

// TimeField edits an HH:MM value. It holds exactly four digits and the
// cursor always sits on one of them, so the shape can never be malformed.
// Only digits are content, which frees every letter for vim-style motions
// without needing modes.
type TimeField struct {
	digits   [4]int
	cursor   int // 0–3: H H : M M
	min, max int // bounds for ctrl+a / ctrl+x, in minutes
	err      string
}

const rangeErr = "use hours 00–23, minutes 00–59"

func (f TimeField) Err() string { return f.err }

func (f TimeField) valid() bool {
	return f.digits[0]*10+f.digits[1] <= 23 && f.digits[2]*10+f.digits[3] <= 59
}

// NewTimeField opens a field on minutes-since-midnight value, with the
// cursor on the first digit. lo and hi bound ctrl+a / ctrl+x.
func NewTimeField(value, lo, hi int) TimeField {
	f := TimeField{min: lo, max: hi}
	f.set(value)
	return f
}

// step adds one unit of the part under the cursor (an hour on the hours, a
// minute on the minutes) to the whole value, so minutes carry and borrow,
// then clamps to the field's bounds.
func (f *TimeField) step(sign int) {
	unit := 1
	if f.cursor < 2 {
		unit = 60
	}
	f.set(max(f.min, min(f.max, f.Value()+sign*unit)))
}

// SetValue replaces the value (minutes since midnight), keeping the cursor
// and clearing any range error.
func (f *TimeField) SetValue(minutes int) {
	f.set(minutes)
	f.err = ""
}

func (f *TimeField) set(minutes int) {
	h, m := minutes/60, minutes%60
	f.digits = [4]int{h / 10, h % 10, m / 10, m % 10}
}

// Value is the field's minutes since midnight. Typed digits are stored raw,
// so it can be out of range until enter checks it.
func (f TimeField) Value() int {
	return (f.digits[0]*10+f.digits[1])*60 + f.digits[2]*10 + f.digits[3]
}

func (f TimeField) Cursor() int { return f.cursor }

func (f TimeField) String() string {
	return fmt.Sprintf("%d%d:%d%d", f.digits[0], f.digits[1], f.digits[2], f.digits[3])
}

func (f TimeField) Update(k tea.KeyMsg) (TimeField, FieldResult) {
	// A fast burst of typing arrives as one KeyMsg with several runes;
	// treat each rune as its own keystroke.
	if k.Type == tea.KeyRunes && len(k.Runes) > 1 {
		res := FieldEditing
		for _, r := range k.Runes {
			f, res = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		return f, res
	}
	s := k.String()
	switch s {
	case "enter":
		if !f.valid() {
			f.err = rangeErr
			return f, FieldEditing
		}
		f.err = ""
		return f, FieldAccepted
	case "esc":
		return f, FieldReverted
	}
	f.err = ""
	switch {
	case len(s) == 1 && s[0] >= '0' && s[0] <= '9':
		f.digits[f.cursor] = int(s[0] - '0')
		f.move(1)
	case s == "l":
		f.move(1)
	case s == "h":
		f.move(-1)
	case s == "^":
		f.cursor = 0
	case s == "$":
		f.cursor = 3
	case s == "w":
		f.cursor = 2
	case s == "b":
		f.cursor = 0
	case s == "ctrl+a":
		f.step(1)
	case s == "ctrl+x":
		f.step(-1)
	}
	return f, FieldEditing
}

func (f *TimeField) move(by int) {
	f.cursor = max(0, min(3, f.cursor+by))
}

// View renders HH:MM with a block cursor on the current digit. The range
// error (Err) is left to the host to place, so the field stays narrow.
func (f TimeField) View() string {
	out := ""
	for i, d := range f.digits {
		if i == 2 {
			out += ":"
		}
		cell := fmt.Sprint(d)
		if i == f.cursor {
			cell = theme.FieldCursorStyle.Render(cell)
		}
		out += cell
	}
	return out
}
