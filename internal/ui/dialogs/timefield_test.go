package dialogs

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// keyMsg builds the KeyMsg a terminal sends for s: named keys by name,
// anything else as typed runes.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+a":
		return tea.KeyMsg{Type: tea.KeyCtrlA}
	case "ctrl+x":
		return tea.KeyMsg{Type: tea.KeyCtrlX}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+p":
		return tea.KeyMsg{Type: tea.KeyCtrlP}
	case "ctrl+n":
		return tea.KeyMsg{Type: tea.KeyCtrlN}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// feedField sends each key to f and returns the field and the last result.
func feedField(f TimeField, keys ...string) (TimeField, FieldResult) {
	res := FieldEditing
	for _, k := range keys {
		f, res = f.Update(keyMsg(k))
	}
	return f, res
}

const dayMax = 23*60 + 59

func TestTimeFieldStartsOnFirstDigit(t *testing.T) {
	f := NewTimeField(14*60+32, 0, dayMax)
	if f.String() != "14:32" || f.Cursor() != 0 {
		t.Fatalf("got %q cursor %d, want 14:32 cursor 0", f.String(), f.Cursor())
	}
}

func TestTimeFieldTypingOverwritesInPlace(t *testing.T) {
	f := NewTimeField(14*60+32, 0, dayMax)
	f, _ = feedField(f, "l", "l") // first minute digit
	f, _ = feedField(f, "1", "5")
	if f.String() != "14:15" {
		t.Fatalf("got %q, want 14:15", f.String())
	}
}

func TestTimeFieldTypingSkipsColonAndStopsAtEnd(t *testing.T) {
	f := NewTimeField(0, 0, dayMax)
	f, _ = feedField(f, "0", "9", "4", "5", "6")
	if f.String() != "09:46" {
		t.Fatalf("got %q, want 09:46 (typing advances across the colon, last digit overwrites in place)", f.String())
	}
	if f.Cursor() != 3 {
		t.Fatalf("cursor = %d, want 3", f.Cursor())
	}
}

func TestTimeFieldLettersDoNotChangeValue(t *testing.T) {
	f := NewTimeField(14*60+32, 0, dayMax)
	f, _ = feedField(f, "l")
	for _, k := range []string{"x", "q", "z", ":", " "} {
		g, res := feedField(f, k)
		if g.String() != "14:32" || g.Cursor() != 1 || res != FieldEditing {
			t.Fatalf("key %q: got %q cursor %d res %v, want unchanged", k, g.String(), g.Cursor(), res)
		}
	}
}

func TestTimeFieldViewShowsTime(t *testing.T) {
	f := NewTimeField(9*60+5, 0, dayMax)
	v := f.View()
	for _, d := range []string{"0", "9", ":", "5"} {
		if !strings.Contains(v, d) {
			t.Fatalf("view %q missing %q", v, d)
		}
	}
}

func TestTimeFieldMotions(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		want int
	}{
		{"l moves right", []string{"l"}, 1},
		{"l stops at the last digit", []string{"l", "l", "l", "l", "l"}, 3},
		{"h moves left", []string{"l", "l", "h"}, 1},
		{"h stops at the first digit", []string{"h"}, 0},
		{"$ jumps to the last digit", []string{"$"}, 3},
		{"^ jumps to the first digit", []string{"$", "^"}, 0},
		{"w jumps to the minutes", []string{"w"}, 2},
		{"w from the minutes stays on them", []string{"$", "w"}, 2},
		{"b jumps to the hours", []string{"$", "b"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := feedField(NewTimeField(14*60+32, 0, dayMax), c.keys...)
			if f.Cursor() != c.want {
				t.Fatalf("cursor = %d, want %d", f.Cursor(), c.want)
			}
			if f.String() != "14:32" {
				t.Fatalf("value changed to %q by a motion", f.String())
			}
		})
	}
}

func TestTimeFieldZeroTypesADigit(t *testing.T) {
	f, _ := feedField(NewTimeField(14*60+32, 0, dayMax), "$", "0")
	if f.String() != "14:30" {
		t.Fatalf("got %q, want 14:30 (0 is content, not a motion)", f.String())
	}
}

func TestTimeFieldCursorSkipsColon(t *testing.T) {
	f, _ := feedField(NewTimeField(14*60+32, 0, dayMax), "l") // last hour digit
	f, _ = feedField(f, "l")
	if f.Cursor() != 2 {
		t.Fatalf("cursor = %d, want 2 (first minute digit)", f.Cursor())
	}
	if !strings.Contains(f.View(), ":") {
		t.Fatal("colon missing from view")
	}
}

func TestTimeFieldIncrement(t *testing.T) {
	cases := []struct {
		name     string
		value    int
		min, max int
		keys     []string
		want     string
	}{
		{"ctrl+a on minutes adds a minute", 14*60 + 32, 0, dayMax, []string{"w", "ctrl+a"}, "14:33"},
		{"ctrl+a on hours adds an hour", 14*60 + 32, 0, dayMax, []string{"ctrl+a"}, "15:32"},
		{"ctrl+x on hours subtracts an hour", 14*60 + 32, 0, dayMax, []string{"ctrl+x"}, "13:32"},
		{"Minutes carry into hours", 14*60 + 59, 0, dayMax, []string{"w", "ctrl+a"}, "15:00"},
		{"Minutes borrow from hours", 15 * 60, 0, dayMax, []string{"w", "ctrl+x"}, "14:59"},
		{"Day boundary holds", dayMax, 0, dayMax, []string{"w", "ctrl+a"}, "23:59"},
		{"Hours clamp at the day boundary", 23*60 + 30, 0, dayMax, []string{"ctrl+a"}, "23:59"},
		{"Start of day holds", 0, 0, dayMax, []string{"w", "ctrl+x"}, "00:00"},
		{"Duration floor of 00:01", 1, 1, dayMax, []string{"w", "ctrl+x"}, "00:01"},
		{"Duration hours clamp to the floor", 30, 1, dayMax, []string{"ctrl+x"}, "00:01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := feedField(NewTimeField(c.value, c.min, c.max), c.keys...)
			if f.String() != c.want {
				t.Fatalf("got %q, want %q", f.String(), c.want)
			}
		})
	}
}

func TestTimeFieldIncrementKeepsCursor(t *testing.T) {
	f, _ := feedField(NewTimeField(14*60+59, 0, dayMax), "$", "ctrl+a")
	if f.Cursor() != 3 {
		t.Fatalf("cursor = %d, want 3", f.Cursor())
	}
}

func TestTimeFieldEnterAcceptsValidValue(t *testing.T) {
	f, res := feedField(NewTimeField(14*60+32, 0, dayMax), "w", "1", "5", "enter")
	if res != FieldAccepted {
		t.Fatalf("result = %v, want FieldAccepted", res)
	}
	if f.Value() != 14*60+15 || f.Err() != "" {
		t.Fatalf("value %d err %q, want 14:15 and no error", f.Value(), f.Err())
	}
}

func TestTimeFieldOutOfRangeTimeIsRejected(t *testing.T) {
	for _, typed := range [][]string{{"2", "7", "1", "0"}, {"1", "4", "6", "0"}} {
		f, res := feedField(NewTimeField(0, 0, dayMax), append(typed, "enter")...)
		if res != FieldEditing {
			t.Fatalf("%v: result = %v, want the field to stay open", typed, res)
		}
		if !strings.Contains(f.Err(), "00–23") || !strings.Contains(f.Err(), "00–59") {
			t.Fatalf("%v: err = %q, want the valid range", typed, f.Err())
		}
	}
}

func TestTimeFieldErrorClearsOnNextEdit(t *testing.T) {
	f, _ := feedField(NewTimeField(0, 0, dayMax), "2", "7", "enter")
	f, _ = feedField(f, "^", "1")
	if f.Err() != "" {
		t.Fatalf("err = %q after editing, want it cleared", f.Err())
	}
}

func TestTimeFieldEscReverts(t *testing.T) {
	_, res := feedField(NewTimeField(14*60+32, 0, dayMax), "0", "9", "esc")
	if res != FieldReverted {
		t.Fatalf("result = %v, want FieldReverted", res)
	}
}

// Bubble Tea delivers a fast burst of typed characters as one KeyMsg with
// several runes; each rune must still count as its own keystroke.
func TestTimeFieldBurstOfRunesTypesEachDigit(t *testing.T) {
	f := NewTimeField(14*60+32, 0, dayMax)
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1145")})
	if f.String() != "11:45" || f.Cursor() != 3 {
		t.Fatalf("got %q cursor %d, want 11:45 cursor 3", f.String(), f.Cursor())
	}
}

func TestTimeFieldBurstMixesDigitsAndMotions(t *testing.T) {
	f := NewTimeField(14*60+32, 0, dayMax)
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w05")})
	if f.String() != "14:05" {
		t.Fatalf("got %q, want 14:05", f.String())
	}
}

// SetValue lets a host (the date+time field's carry) replace the value
// without moving the user's cursor.
func TestTimeFieldSetValueKeepsCursor(t *testing.T) {
	f, _ := feedField(NewTimeField(14*60+32, 0, dayMax), "w", "l") // last digit
	f.SetValue(9*60 + 5)
	if f.String() != "09:05" || f.Cursor() != 3 {
		t.Fatalf("got %q cursor %d, want 09:05 cursor 3", f.String(), f.Cursor())
	}
}
