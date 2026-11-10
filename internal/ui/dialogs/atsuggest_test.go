package dialogs

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// atHarness is a description input with an @ suggester, driven the way a
// dialog drives it: every key goes to the suggester, which forwards text
// keys to the input.
type atHarness struct {
	t   *testing.T
	ti  textinput.Model
	s   atSuggest
	res PickResult
}

func newAtHarness(t *testing.T, value string) *atHarness {
	ti := textinput.New()
	ti.SetValue(value)
	ti.CursorEnd()
	ti.Focus()
	return &atHarness{t: t, ti: ti, s: newAtSuggest(listProjects)}
}

func (h *atHarness) keys(ks ...string) {
	for _, k := range ks {
		h.res, _ = h.s.Update(&h.ti, keyMsg(k), false)
	}
}

func (h *atHarness) wantOpen(open bool) {
	h.t.Helper()
	if h.s.Open() != open {
		h.t.Fatalf("open = %v, want %v (value %q)", h.s.Open(), open, h.ti.Value())
	}
}

func (h *atHarness) wantValue(v string) {
	h.t.Helper()
	if h.ti.Value() != v {
		h.t.Fatalf("value = %q, want %q", h.ti.Value(), v)
	}
}

func TestAtSuggestOpensAtWordStart(t *testing.T) {
	h := newAtHarness(t, "")
	h.keys("@")
	h.wantOpen(true)
	g := newAtHarness(t, "fix ")
	g.keys("@")
	g.wantOpen(true)
}

func TestAtSuggestIgnoresAtInsideWord(t *testing.T) {
	h := newAtHarness(t, "mail me")
	h.keys("@")
	h.wantOpen(false)
	h.wantValue("mail me@")
}

func TestAtSuggestIgnoresPastedAt(t *testing.T) {
	h := newAtHarness(t, "")
	h.res, _ = h.s.Update(&h.ti, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" @acme")}, false)
	h.wantOpen(false)
	h.wantValue(" @acme")
}

func TestAtSuggestQueryFollowsTyping(t *testing.T) {
	h := newAtHarness(t, "fix ")
	h.keys("@", "i", "n")
	if o := h.s.list.Selected(); o.Project.Name != "Internal" {
		t.Fatalf("selected = %+v, want Internal for @in", o)
	}
	h.keys("space", "x") // spaces are part of the search
	h.wantOpen(true)
	if o := h.s.list.Selected(); o.Kind != OptionCreate || o.Name != "in x" {
		t.Fatalf("selected = %+v, want create \"in x\"", o)
	}
}

func TestAtSuggestClosesOnBackspaceOverAt(t *testing.T) {
	h := newAtHarness(t, "fix ")
	h.keys("@", "a", "backspace")
	h.wantOpen(true)
	h.keys("backspace")
	h.wantOpen(false)
	h.wantValue("fix ")
}

func TestAtSuggestClosesWhenCursorLeaves(t *testing.T) {
	h := newAtHarness(t, "x ")
	h.keys("@", "a", "c", "left", "left")
	h.wantOpen(true) // cursor right after the @
	h.keys("left")
	h.wantOpen(false)
}

func TestAtSuggestPickSplicesText(t *testing.T) {
	h := newAtHarness(t, "fix login bug ")
	h.keys("@", "i", "n", "t", "enter")
	if h.res != PickPicked || h.s.Picked().Project.Name != "Internal" {
		t.Fatalf("res %v picked %+v, want Internal", h.res, h.s.Picked())
	}
	h.wantOpen(false)
	h.wantValue("fix login bug")
}

func TestAtSuggestPickInTheMiddle(t *testing.T) {
	h := newAtHarness(t, "fix  bug")
	h.ti.SetCursor(4) // between the two spaces
	h.keys("@", "a", "c", "m", "e", "tab")
	h.wantValue("fix bug")
	if h.ti.Position() != 4 {
		t.Fatalf("cursor = %d, want 4 (where the @ was)", h.ti.Position())
	}
}

func TestAtSuggestPickAtStartOfText(t *testing.T) {
	h := newAtHarness(t, "")
	h.keys("@", "a", "c", "m", "e", "enter")
	h.wantValue("")
	if h.s.Picked().Project.Name != "ACME" {
		t.Fatalf("picked = %+v, want ACME", h.s.Picked())
	}
}

func TestAtSuggestEscKeepsText(t *testing.T) {
	h := newAtHarness(t, "")
	h.keys("@", "a", "c", "m", "esc")
	if h.res != PickCancelled {
		t.Fatalf("res = %v, want cancelled", h.res)
	}
	h.wantOpen(false)
	h.wantValue("@acm")
}

func TestAtSuggestConsumesEnterAndEsc(t *testing.T) {
	h := newAtHarness(t, "x ")
	h.keys("@", "down", "up")
	h.wantValue("x @") // selection keys don't reach the input
	h.keys("enter")
	if h.res != PickPicked {
		t.Fatalf("enter while open = %v, want picked", h.res)
	}
	// Closed again: enter belongs to the dialog, so the suggester ignores it.
	h.keys("enter")
	if h.res != PickEditing || h.s.Open() {
		t.Fatalf("enter while closed = %v open %v, want ignored", h.res, h.s.Open())
	}
}
