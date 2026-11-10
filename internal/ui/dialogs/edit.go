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

	// fieldOpen is set while a time row's field is open over the form:
	// dt (date + HH:MM) for start and stop, field (HH:MM) for duration.
	fieldOpen bool
	dtOpen    bool
	dt        DateTimeField
	field     TimeField
	// fieldFrom / dtFrom are the values the field opened with; accepting
	// them unchanged writes nothing, so the original timestamp (seconds
	// included) survives.
	fieldFrom int
	dtFrom    time.Time
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

// openField opens the focused time row's field: date + HH:MM for start
// and stop, HH:MM for duration.
func (d *Edit) openField() {
	d.fieldOpen = true
	d.dtOpen = true
	switch d.rows[d.focus] {
	case rowStart:
		d.dt = NewDateTimeField(d.start, d.startLowerBound(), d.lastMinuteToday(), d.resolveStart)
	case rowStop:
		d.dt = NewDateTimeField(d.stop, time.Time{}, d.lastMinuteToday(), d.resolveStop)
	case rowDuration:
		d.dtOpen = false
		d.field = NewTimeField(min(d.durationMinutes(), dayMinutes-1), 1, dayMinutes-1)
		d.fieldFrom = d.field.Value()
		return
	}
	d.dtFrom = d.dt.Value()
}

// maxWrap is how long an entry may become when a typed time is placed on
// the neighbouring day. Real overnight entries are short; a typo that would
// wrap into a ~23h entry stays on its day instead.
const maxWrap = 12 * time.Hour

// resolveStop places a typed stop time on its first occurrence after start
// when that makes an entry of at most maxWrap, crossing midnight if needed;
// otherwise the time stays on the field's date.
func (d *Edit) resolveStop(day time.Time, minutes int) time.Time {
	c := atClock(d.start, minutes)
	if !c.After(d.start) {
		c = c.AddDate(0, 0, 1)
	}
	if c.Sub(d.start) <= maxWrap && !c.After(d.lastMinuteToday()) {
		return c
	}
	return atClock(day, minutes)
}

// resolveStart places a typed start time on its last occurrence before the
// other end (stop, or now for a running entry) when that is within maxWrap.
func (d *Edit) resolveStart(day time.Time, minutes int) time.Time {
	other := d.stop
	if d.orig.Running() {
		other = d.now
	}
	c := atClock(other, minutes)
	if !c.Before(other) {
		c = c.AddDate(0, 0, -1)
	}
	lo := d.startLowerBound()
	if other.Sub(c) <= maxWrap && (lo.IsZero() || !c.Before(lo)) {
		return c
	}
	return atClock(day, minutes)
}

// lastMinuteToday bounds every date: nothing is set past today.
func (d *Edit) lastMinuteToday() time.Time { return atClock(d.now, dayMinutes-1) }

// startLowerBound keeps a running entry's start within a week, so the next
// refresh (which fetches the last 8 days) still finds it. A finished
// entry's start has no lower bound.
func (d *Edit) startLowerBound() time.Time {
	if !d.orig.Running() {
		return time.Time{}
	}
	return midnight(d.now).AddDate(0, 0, -7)
}

func (d *Edit) updateField(k tea.KeyMsg) {
	var res FieldResult
	if d.dtOpen {
		d.dt, res = d.dt.Update(k)
		switch res {
		case FieldAccepted:
			if v := d.dt.Value(); !v.Equal(d.dtFrom) {
				d.applyTime(d.rows[d.focus], v)
			}
			d.fieldOpen = false
		case FieldReverted:
			d.fieldOpen = false
		}
		return
	}
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

// applyTime writes an accepted start or stop into the draft. Start keeps
// stop, stop keeps start, and an entry is never shorter than a minute: a
// time that would cross the other end lands a minute from it. Stepping
// start's date moves the whole entry; a time moved only by date steps
// keeps its seconds.
func (d *Edit) applyTime(r editRow, v time.Time) {
	days := d.dt.Stepped()
	switch r {
	case rowStart:
		if days != 0 && !d.orig.Running() {
			d.stop = d.stop.AddDate(0, 0, days)
		}
		if d.dt.Clocked() {
			d.start = restoreIfSameMinute(v, d.orig.Start)
		} else {
			d.start = d.start.AddDate(0, 0, days)
		}
		if d.orig.Running() {
			latest := d.now.Truncate(time.Minute).Add(-time.Minute)
			if d.start.After(latest) && !d.start.Equal(d.orig.Start) {
				d.start = latest
			}
		} else if d.stop.Sub(d.start) < time.Minute {
			d.start = d.stop.Add(-time.Minute)
		}
	case rowStop:
		if d.dt.Clocked() {
			d.stop = restoreIfSameMinute(v, d.orig.Stop)
		} else {
			d.stop = d.stop.AddDate(0, 0, days)
		}
		if d.stop.Sub(d.start) < time.Minute {
			d.stop = d.start.Add(time.Minute)
		}
	}
}

// applyField writes an accepted duration into the draft: start stays and
// stop becomes start plus the duration.
func (d *Edit) applyField(r editRow, minutes int) {
	switch r {
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

func (d *Edit) fieldErr() string {
	if d.dtOpen {
		return d.dt.Err()
	}
	return d.field.Err()
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
		before, after := whenPair(o.Start.Local(), d.start)
		out = append(out, change{"start", before, after})
	}
	if o.Running() {
		return out
	}
	if !d.stop.Equal(o.Stop) {
		before, after := whenPair(o.Stop.Local(), d.stop)
		out = append(out, change{"stop", before, after})
	}
	before := int(o.Stop.Sub(o.Start) / time.Minute)
	if after := d.durationMinutes(); after != before {
		out = append(out, change{"duration", clock(before), clock(after)})
	}
	return out
}

// whenPair formats a changed time for the review: HH:MM on both sides, or
// date and time on both sides when the date changed.
func whenPair(before, after time.Time) (string, string) {
	if midnight(before).Equal(midnight(after)) {
		return clock(minutesOfDay(before)), clock(minutesOfDay(after))
	}
	return before.Format(dateLayout) + " " + clock(minutesOfDay(before)),
		after.Format(dateLayout) + " " + clock(minutesOfDay(after))
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
		if c.label == "start" && !toggl.InRecentWindow(d.start, d.now) {
			// The next refresh won't fetch it; it's still saved in Toggl.
			lines = append(lines, strings.Repeat(" ", 12)+theme.MutedStyle.Render("(outside list)"))
		}
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
		if err := d.fieldErr(); err != "" {
			lines = append(lines, theme.ErrStyle.Render(err))
		}
		switch {
		case d.dtOpen && d.dt.DateActive():
			lines = append(lines,
				theme.MutedStyle.Render("ctrl+a/x +/-1 day"),
				theme.MutedStyle.Render("tab time · enter ok · esc revert"))
		case d.dtOpen:
			lines = append(lines,
				theme.MutedStyle.Render("0-9 type · h l ^ $ w b move · ctrl+a/x +/-1"),
				theme.MutedStyle.Render("tab date/time · enter ok · esc revert"))
		default:
			lines = append(lines,
				theme.MutedStyle.Render("0-9 type · h l ^ $ w b move · ctrl+a/x +/-1"),
				theme.MutedStyle.Render("enter ok · esc revert"))
		}
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
	if d.dtOpen {
		content = d.dt.View()
	}
	w := lipgloss.Width(title) + 6
	for _, l := range strings.Split(content, "\n") {
		w = max(w, lipgloss.Width(l)+4)
	}
	h := strings.Count(content, "\n") + 3
	pop := theme.Panel(title, content, w, h, true)
	return theme.Overlay(view, pop, boxWidth-w-2, rowY(d.focus))
}
