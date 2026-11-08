// Package theme holds the shared colors, styles, and rendering helpers used
// by the app, panels, and dialogs. It sits below all other ui packages so
// none of them need to import each other for styling.
package theme

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	Accent = lipgloss.AdaptiveColor{Light: "#B3186B", Dark: "#E57CD8"} // toggl pink
	Green  = lipgloss.AdaptiveColor{Light: "#0A7D33", Dark: "#43C465"}
	Amber  = lipgloss.AdaptiveColor{Light: "#B8860B", Dark: "#E0A030"}
	Red    = lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#E06C60"}
	Muted  = lipgloss.AdaptiveColor{Light: "#9E9E9E", Dark: "#6B6B6B"}
)

var (
	MutedStyle   = lipgloss.NewStyle().Foreground(Muted)
	HeaderStyle  = lipgloss.NewStyle().Foreground(Muted).Bold(true)
	RunningStyle = lipgloss.NewStyle().Foreground(Green).Bold(true)
	PausedStyle  = lipgloss.NewStyle().Foreground(Amber).Bold(true)
	CursorStyle  = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	ErrStyle     = lipgloss.NewStyle().Foreground(Red)
	// FieldCursorStyle is the block cursor on the digit a TimeField edits.
	FieldCursorStyle = lipgloss.NewStyle().Reverse(true).Bold(true)
)

// PanelInnerWidth is the content width available inside a Panel of the given
// total width (borders plus one space of padding per side).
func PanelInnerWidth(width int) int { return width - 4 }

// Panel draws a rounded-border box with the title embedded in the top border,
// lazygit-style. Content lines are padded (ANSI-aware) to fill the box; the
// box is exactly width x height cells.
func Panel(title, content string, width, height int, focused bool) string {
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}
	borderColor := Muted
	if focused {
		borderColor = Accent
	}
	bs := lipgloss.NewStyle().Foreground(borderColor)
	ts := lipgloss.NewStyle().Foreground(borderColor).Bold(true)

	inner := width - 2
	var top string
	if title != "" {
		fill := inner - lipgloss.Width(title) - 3
		if fill < 0 {
			fill = 0
		}
		top = bs.Render("╭─ ") + ts.Render(title) + bs.Render(" "+strings.Repeat("─", fill)+"╮")
	} else {
		top = bs.Render("╭" + strings.Repeat("─", inner) + "╮")
	}

	lines := strings.Split(content, "\n")
	out := []string{top}
	for i := 0; i < height-2; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		pad := inner - 2 - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		out = append(out, bs.Render("│ ")+line+strings.Repeat(" ", pad)+bs.Render(" │"))
	}
	out = append(out, bs.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return strings.Join(out, "\n")
}

// FormatClock renders a duration as HH:MM:SS.
func FormatClock(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// FormatTotal renders a duration compactly for day totals, e.g. "3h 45m".
func FormatTotal(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %02dm", h, m)
}

// Truncate limits s to w runes, ending with an ellipsis when cut.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

// Pad returns n spaces (never negative).
func Pad(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// Overlay composites fg on top of bg with fg's top-left corner at cell
// column x, row y — used for the floating API-error notice. Splicing is
// ANSI-aware (styled cells count as their display width), and a style reset
// is inserted at the left seam so background styling can't bleed into the
// overlaid box.
func Overlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	for i, fgLine := range strings.Split(fg, "\n") {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		line := bgLines[row]
		left := ansi.Truncate(line, x, "")
		pad := Pad(x - lipgloss.Width(left))
		right := ansi.TruncateLeft(line, x+lipgloss.Width(fgLine), "")
		bgLines[row] = left + "\x1b[0m" + pad + fgLine + right
	}
	return strings.Join(bgLines, "\n")
}
