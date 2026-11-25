package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func resize(d *driver, w int) {
	mm, _ := d.m.Update(tea.WindowSizeMsg{Width: w, Height: 30})
	d.m = mm.(Model)
}

// At 80 columns the short help used to come out 89 wide: bubbles' help
// adds an item that doesn't fit when there's no room for its ellipsis.
// It now ends in "…" within the width instead.
func TestHelpBarFitsAt80(t *testing.T) {
	d := newDriver(t)
	resize(d, 80)
	lines := strings.Split(d.m.View(), "\n")
	bar := strings.TrimRight(lines[len(lines)-1], " ") // JoinVertical pads to the widest block
	if w := lipgloss.Width(bar); w > 80 || !strings.HasSuffix(bar, "…") || !strings.Contains(bar, "e edit") {
		t.Fatalf("help bar (%d cols) = %q, want it within 80, through \"e edit\", ending in …", w, bar)
	}
}

// Every width: the help (short and full) fits, and shows everything when
// there is room.
func TestHelpFitsEveryWidth(t *testing.T) {
	for _, all := range []bool{false, true} {
		for w := 10; w <= 120; w++ {
			d := newDriver(t)
			resize(d, w)
			d.m.help.ShowAll = all
			hv := d.m.helpView()
			for _, l := range strings.Split(hv, "\n") {
				if lipgloss.Width(l) > w {
					t.Fatalf("full=%v width %d: help line %d wide: %q", all, w, lipgloss.Width(l), l)
				}
			}
			if w >= 120 && strings.Contains(hv, "…") {
				t.Fatalf("full=%v width %d: help truncated with room to spare: %q", all, w, hv)
			}
		}
	}
}
