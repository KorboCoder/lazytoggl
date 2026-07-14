package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

type RowKind int

const (
	RowHeader RowKind = iota
	RowEntry
)

// Row is one line of the entries list. The list is heterogeneous — day
// headers and entries — so the cursor skips non-selectable rows.
type Row struct {
	Kind  RowKind
	Entry toggl.TimeEntry
	Label string        // header text, e.g. "Today"
	Total time.Duration // header day total
}

func (r Row) selectable() bool { return r.Kind != RowHeader }

// BuildRows flattens entries (newest-first) into list rows: a header per day
// with that day's total, then the day's entries.
func BuildRows(entries []toggl.TimeEntry, now time.Time) []Row {
	var rows []Row
	totals := map[string]time.Duration{}
	for _, e := range entries {
		totals[dayKey(e.Start)] += e.Elapsed(now)
	}
	var curDay string
	for _, e := range entries {
		k := dayKey(e.Start)
		if k != curDay {
			curDay = k
			rows = append(rows, Row{Kind: RowHeader, Label: dayLabel(e.Start, now), Total: totals[k]})
		}
		rows = append(rows, Row{Kind: RowEntry, Entry: e})
	}
	return rows
}

func dayKey(t time.Time) string { return t.Local().Format("2006-01-02") }

func dayLabel(t, now time.Time) string {
	switch dayKey(t) {
	case dayKey(now):
		return "Today"
	case dayKey(now.AddDate(0, 0, -1)):
		return "Yesterday"
	}
	return t.Local().Format("Mon, Jan 2")
}

// Entries is the list panel: it owns only cursor and scroll state; rows are
// rebuilt from the app's entry slice every frame.
type Entries struct {
	Cursor int
	offset int
}

// Move steps the cursor by delta (-1/+1), skipping header rows.
func (m *Entries) Move(rows []Row, delta int) {
	i := m.Cursor + delta
	for i >= 0 && i < len(rows) {
		if rows[i].selectable() {
			m.Cursor = i
			return
		}
		i += delta
	}
}

// Clamp snaps the cursor onto the nearest selectable row after the row set
// changes (loads, deletes, refreshes).
func (m *Entries) Clamp(rows []Row) {
	if len(rows) == 0 {
		m.Cursor = 0
		return
	}
	if m.Cursor >= len(rows) {
		m.Cursor = len(rows) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if rows[m.Cursor].selectable() {
		return
	}
	for d := 1; d < len(rows); d++ {
		if i := m.Cursor + d; i < len(rows) && rows[i].selectable() {
			m.Cursor = i
			return
		}
		if i := m.Cursor - d; i >= 0 && rows[i].selectable() {
			m.Cursor = i
			return
		}
	}
	m.Cursor = 0
}

// Selected returns the row under the cursor.
func (m *Entries) Selected(rows []Row) (Row, bool) {
	if m.Cursor < 0 || m.Cursor >= len(rows) {
		return Row{}, false
	}
	return rows[m.Cursor], true
}

// View renders the panel at the given size, keeping the cursor scrolled into
// view. The panel title carries today's total.
func (m *Entries) View(rows []Row, projects map[int64]toggl.Project, now time.Time, width, height int, loading, focused bool) string {
	iw := theme.PanelInnerWidth(width)
	visible := height - 2
	if visible < 1 {
		visible = 1
	}
	if m.Cursor < m.offset {
		m.offset = m.Cursor
	}
	if m.Cursor >= m.offset+visible {
		m.offset = m.Cursor - visible + 1
	}
	if max := len(rows) - visible; m.offset > max {
		m.offset = max
	}
	if m.offset < 0 {
		m.offset = 0
	}

	var lines []string
	for i := m.offset; i < len(rows) && i < m.offset+visible; i++ {
		lines = append(lines, m.renderRow(rows[i], i == m.Cursor, iw, projects, now))
	}
	if len(rows) == 0 {
		hint := "no entries yet — press c to start one"
		if loading {
			hint = "loading…"
		}
		lines = append(lines, "  "+theme.MutedStyle.Render(hint))
	}
	return theme.Panel("Entries", strings.Join(lines, "\n"), width, height, focused)
}

func (m *Entries) renderRow(r Row, selected bool, iw int, projects map[int64]toggl.Project, now time.Time) string {
	cursor := "  "
	if selected {
		cursor = theme.CursorStyle.Render("> ")
	}
	switch r.Kind {
	case RowHeader:
		label := fmt.Sprintf("%s · %s", r.Label, theme.FormatTotal(r.Total))
		return "  " + theme.HeaderStyle.Render(theme.Truncate(label, iw-2))
	}

	e := r.Entry
	const durW = 8
	projW := 12
	if iw < 50 {
		projW = 0
	}

	run := "  "
	if e.Running() {
		run = theme.RunningStyle.Render("▶ ")
	}

	descW := iw - 2 - 2 - durW - 2
	var projStr string
	if projW > 0 {
		descW -= projW + 2
		name, color := projectLabel(e.ProjectID, projects)
		name = theme.Truncate(name, projW)
		pad := theme.Pad(projW - len([]rune(name)))
		if color != "" {
			projStr = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(name) + pad
		} else {
			projStr = theme.MutedStyle.Render(name) + pad
		}
		projStr += "  "
	}

	desc := e.Description
	if desc == "" {
		desc = "(no description)"
	}
	desc = theme.Truncate(desc, descW)
	descPad := theme.Pad(descW - len([]rune(desc)))

	clock := fmt.Sprintf("%*s", durW, theme.FormatClock(e.Elapsed(now)))
	var clockStr string
	if e.Running() {
		clockStr = theme.RunningStyle.Render(clock)
		desc = theme.RunningStyle.Render(desc)
	} else {
		clockStr = clock
	}

	return cursor + run + desc + descPad + projStr + clockStr
}
