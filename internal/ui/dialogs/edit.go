package dialogs

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// dateLayout is the read-only date shown beside a start/stop time.
const dateLayout = "Mon 02 Jan"

type editRow int

const (
	rowDescription editRow = iota
	rowStart
	rowStop
	rowDuration
)

var rowLabels = map[editRow]string{
	rowDescription: "description",
	rowStart:       "start",
	rowStop:        "stop",
	rowDuration:    "duration",
}

// Edit is the edit dialog: a form over a draft of the entry. The original
// entry is kept untouched so unchanged fields are saved exactly as they were.
// A running entry has no stop, so it shows only description and start.
type Edit struct {
	orig  toggl.TimeEntry
	now   time.Time
	desc  textinput.Model
	start time.Time
	stop  time.Time
	rows  []editRow
	focus int

	// fieldOpen is set while a time row's HH:MM field is open over the form.
	fieldOpen bool
	field     TimeField
	// fieldFrom is the value the field opened with; accepting it unchanged
	// writes nothing, so the original timestamp (seconds included) survives.
	fieldFrom int
	// reviewing is set while the before/after review is shown.
	reviewing bool
}

const dayMinutes = 24 * 60

func NewEdit(entry toggl.TimeEntry, now time.Time) *Edit {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 200
	ti.Width = boxWidth - 8
	ti.SetValue(entry.Description)
	ti.CursorEnd()
	ti.Focus()
	d := &Edit{
		orig:  entry,
		now:   now,
		desc:  ti,
		start: entry.Start.Local(),
		rows:  []editRow{rowDescription, rowStart},
	}
	if !entry.Running() {
		d.stop = entry.Stop.Local()
		d.rows = append(d.rows, rowStop, rowDuration)
	}
	return d
}

func (d *Edit) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	k, isKey := msg.(tea.KeyMsg)
	if d.reviewing {
		if isKey {
			return d, d.updateReview(k)
		}
		return d, nil
	}
	if d.fieldOpen {
		if isKey {
			d.updateField(k)
		}
		return d, nil
	}
	if isKey {
		switch k.String() {
		case "esc":
			if len(d.changes()) == 0 {
				return d, closeCmd()
			}
			d.reviewing = true
			return d, nil
		case "tab", "down":
			return d, d.setFocus(d.focus + 1)
		case "shift+tab", "up":
			return d, d.setFocus(d.focus - 1)
		case "enter":
			if d.rows[d.focus] == rowDescription {
				return d, d.setFocus(d.focus + 1)
			}
			d.openField()
			return d, nil
		}
	}
	if d.rows[d.focus] != rowDescription {
		return d, nil
	}
	var cmd tea.Cmd
	d.desc, cmd = d.desc.Update(msg)
	return d, cmd
}

func (d *Edit) setFocus(i int) tea.Cmd {
	n := len(d.rows)
	d.focus = ((i % n) + n) % n
	if d.rows[d.focus] == rowDescription {
		return d.desc.Focus()
	}
	d.desc.Blur()
	return nil
}

// openField opens the HH:MM field on the focused time row.
func (d *Edit) openField() {
	switch d.rows[d.focus] {
	case rowStart:
		d.field = NewTimeField(minutesOfDay(d.start), 0, dayMinutes-1)
	case rowStop:
		d.field = NewTimeField(minutesOfDay(d.stop), 0, dayMinutes-1)
	case rowDuration:
		d.field = NewTimeField(min(d.durationMinutes(), dayMinutes-1), 1, dayMinutes-1)
	}
	d.fieldFrom = d.field.Value()
	d.fieldOpen = true
}

func (d *Edit) updateField(k tea.KeyMsg) {
	var res FieldResult
	d.field, res = d.field.Update(k)
	switch res {
	case FieldAccepted:
		if v := d.field.Value(); v != d.fieldFrom {
			d.applyField(d.rows[d.focus], v)
		}
		d.fieldOpen = false
	case FieldReverted:
		d.fieldOpen = false
	}
}

// applyField writes an accepted HH:MM value into the draft. Start keeps
// stop, stop and duration keep start, and an entry is never shorter than a
// minute: a time that would cross the other end lands a minute from it.
func (d *Edit) applyField(r editRow, minutes int) {
	switch r {
	case rowStart:
		d.start = restoreIfSameMinute(atClock(d.start, minutes), d.orig.Start)
		if d.orig.Running() {
			latest := d.now.Truncate(time.Minute).Add(-time.Minute)
			if d.start.After(latest) && !d.start.Equal(d.orig.Start) {
				d.start = latest
			}
		} else if d.stop.Sub(d.start) < time.Minute {
			d.start = d.stop.Add(-time.Minute)
		}
	case rowStop:
		d.stop = restoreIfSameMinute(atClock(d.stop, minutes), d.orig.Stop)
		if d.stop.Sub(d.start) < time.Minute {
			d.stop = d.start.Add(time.Minute)
		}
	case rowDuration:
		stop := d.start.Add(time.Duration(max(1, minutes)) * time.Minute)
		d.stop = restoreIfSameMinute(stop, d.orig.Stop)
		if d.stop.Sub(d.start) < time.Minute {
			d.stop = stop
		}
	}
}

// restoreIfSameMinute returns the original timestamp when t names the same
// minute, so retyping a time you'd changed gives back its exact seconds.
func restoreIfSameMinute(t, orig time.Time) time.Time {
	if t.Truncate(time.Minute).Equal(orig.Truncate(time.Minute)) {
		return orig.Local()
	}
	return t
}

// atClock is t's date at the given minutes since midnight, zero seconds.
func atClock(t time.Time, minutes int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), minutes/60, minutes%60, 0, 0, time.Local)
}

func (d *Edit) updateReview(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "y", "enter":
		return d.submitCmd()
	case "esc":
		d.reviewing = false
		// The review swallowed the input's blink ticks; restart them.
		return d.setFocus(d.focus)
	case "n":
		return closeCmd()
	}
	return nil
}

// change is one field that differs from the original entry.
type change struct{ label, before, after string }

// changes lists the fields the draft changed, in row order. Duration is
// listed whenever it differs, since moving either end changes it.
func (d *Edit) changes() []change {
	o := d.orig
	var out []change
	if desc := d.description(); desc != o.Description {
		out = append(out, change{"description", o.Description, desc})
	}
	if !d.start.Equal(o.Start) {
		out = append(out, change{"start", clock(minutesOfDay(o.Start.Local())), clock(minutesOfDay(d.start))})
	}
	if o.Running() {
		return out
	}
	if !d.stop.Equal(o.Stop) {
		out = append(out, change{"stop", clock(minutesOfDay(o.Stop.Local())), clock(minutesOfDay(d.stop))})
	}
	before := int(o.Stop.Sub(o.Start) / time.Minute)
	if after := d.durationMinutes(); after != before {
		out = append(out, change{"duration", clock(before), clock(after)})
	}
	return out
}

func (d *Edit) reviewView() string {
	var lines []string
	for _, c := range d.changes() {
		label := theme.MutedStyle.Render(fmt.Sprintf("%-12s", c.label))
		if c.label == "description" {
			w := (boxWidth - 6 - 12 - 4) / 2
			lines = append(lines, label+theme.Truncate(c.before, w)+" -> "+theme.Truncate(c.after, w))
			continue
		}
		lines = append(lines, label+c.before+" -> "+c.after)
	}
	lines = append(lines, "", theme.MutedStyle.Render("y/enter save · esc back · n discard"))
	return box("Save changes?", lines, boxWidth)
}

// description is the description to save: the trimmed input, or the
// original verbatim when they differ only by surrounding whitespace.
func (d *Edit) description() string {
	desc := strings.TrimSpace(d.desc.Value())
	if desc == strings.TrimSpace(d.orig.Description) {
		return d.orig.Description
	}
	return desc
}

// draft is the entry as the form currently describes it.
func (d *Edit) draft() toggl.TimeEntry {
	e := d.orig
	e.Description = d.description()
	e.Start = d.start
	if !d.orig.Running() {
		e.Stop = d.stop
		e.Duration = int64(d.stop.Sub(d.start).Seconds())
	}
	return e
}

func (d *Edit) submitCmd() tea.Cmd {
	e := d.draft()
	return func() tea.Msg { return EditSubmitMsg{Entry: e} }
}

// clock formats a time of day, or a duration, as HH:MM (minutes floored).
func clock(minutes int) string { return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60) }

func minutesOfDay(t time.Time) int { return t.Hour()*60 + t.Minute() }

func (d *Edit) durationMinutes() int { return int(d.stop.Sub(d.start) / time.Minute) }

// rowValue renders a time row's current value.
func (d *Edit) rowValue(r editRow) string {
	switch r {
	case rowStart:
		return d.start.Format(dateLayout) + " " + clock(minutesOfDay(d.start))
	case rowStop:
		return d.stop.Format(dateLayout) + " " + clock(minutesOfDay(d.stop))
	case rowDuration:
		return clock(d.durationMinutes())
	}
	return ""
}

// rowLine renders one row: a focus marker, a padded label, and its value.
func (d *Edit) rowLine(i int) string {
	r := d.rows[i]
	label := fmt.Sprintf("%-11s", rowLabels[r])
	if i == d.focus {
		return theme.CursorStyle.Render("> "+label) + d.rowValue(r)
	}
	return theme.MutedStyle.Render("  "+label) + d.rowValue(r)
}

// rowY is the content line a row is drawn on (description's input takes a
// line of its own, then a blank line before the time rows).
func rowY(i int) int {
	if i == 0 {
		return 0
	}
	return 2 + i
}

func (d *Edit) View() string {
	if d.reviewing {
		return d.reviewView()
	}
	lines := []string{d.rowLine(0), "  " + d.desc.View(), ""}
	for i := 1; i < len(d.rows); i++ {
		lines = append(lines, d.rowLine(i))
	}
	lines = append(lines, "")
	if d.fieldOpen {
		if err := d.field.Err(); err != "" {
			lines = append(lines, theme.ErrStyle.Render(err))
		}
		lines = append(lines,
			theme.MutedStyle.Render("0-9 type · h l ^ $ w b move · ctrl+a/x +/-1"),
			theme.MutedStyle.Render("enter ok · esc revert"))
	} else {
		lines = append(lines, theme.MutedStyle.Render("tab next · enter edit time · esc done"))
	}
	view := box("Edit entry", lines, boxWidth)
	if !d.fieldOpen {
		return view
	}
	// The field pops over the right side of the form, its middle line level
	// with the row it edits (+1 for the dialog's top border, -1 for its own).
	title := rowLabels[d.rows[d.focus]]
	content := d.field.View()
	w := lipgloss.Width(title) + 6
	for _, l := range strings.Split(content, "\n") {
		w = max(w, lipgloss.Width(l)+4)
	}
	h := strings.Count(content, "\n") + 3
	pop := theme.Panel(title, content, w, h, true)
	return theme.Overlay(view, pop, boxWidth-w-2, rowY(d.focus))
}
