package dialogs

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

const timeLayout = "2006-01-02 15:04"

// Edit lets the user change an entry's description and start/stop times.
// A running entry has no stop field; its duration stays open.
type Edit struct {
	entry  toggl.TimeEntry
	inputs []textinput.Model
	labels []string
	focus  int
	errStr string
}

func NewEdit(entry toggl.TimeEntry) *Edit {
	mk := func(value string) textinput.Model {
		ti := textinput.New()
		ti.CharLimit = 200
		ti.Width = boxWidth - 8
		ti.SetValue(value)
		ti.CursorEnd()
		return ti
	}
	d := &Edit{
		entry:  entry,
		inputs: []textinput.Model{mk(entry.Description), mk(entry.Start.Local().Format(timeLayout))},
		labels: []string{"description", "start"},
	}
	if !entry.Running() {
		d.inputs = append(d.inputs, mk(entry.Stop.Local().Format(timeLayout)))
		d.labels = append(d.labels, "stop")
	}
	d.inputs[0].Focus()
	return d
}

func (d *Edit) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			return d, closeCmd()
		case "enter":
			return d.submit()
		case "tab", "down":
			return d, d.setFocus(d.focus + 1)
		case "shift+tab", "up":
			return d, d.setFocus(d.focus - 1)
		}
	}
	var cmd tea.Cmd
	d.inputs[d.focus], cmd = d.inputs[d.focus].Update(msg)
	return d, cmd
}

func (d *Edit) setFocus(i int) tea.Cmd {
	n := len(d.inputs)
	d.focus = ((i % n) + n) % n
	var cmd tea.Cmd
	for j := range d.inputs {
		if j == d.focus {
			cmd = d.inputs[j].Focus()
		} else {
			d.inputs[j].Blur()
		}
	}
	return cmd
}

func (d *Edit) submit() (Dialog, tea.Cmd) {
	e := d.entry
	e.Description = strings.TrimSpace(d.inputs[0].Value())

	start, err := time.ParseInLocation(timeLayout, strings.TrimSpace(d.inputs[1].Value()), time.Local)
	if err != nil {
		d.errStr = "start: use format " + timeLayout
		return d, nil
	}
	e.Start = start

	if !d.entry.Running() {
		stop, err := time.ParseInLocation(timeLayout, strings.TrimSpace(d.inputs[2].Value()), time.Local)
		if err != nil {
			d.errStr = "stop: use format " + timeLayout
			return d, nil
		}
		if !stop.After(start) {
			d.errStr = "stop must be after start"
			return d, nil
		}
		e.Stop = stop
		e.Duration = int64(stop.Sub(start).Seconds())
	}
	return d, func() tea.Msg { return EditSubmitMsg{Entry: e} }
}

func (d *Edit) View() string {
	var lines []string
	for i, in := range d.inputs {
		label := d.labels[i]
		if i == d.focus {
			lines = append(lines, theme.CursorStyle.Render(label))
		} else {
			lines = append(lines, theme.MutedStyle.Render(label))
		}
		lines = append(lines, in.View())
	}
	if d.errStr != "" {
		lines = append(lines, theme.ErrStyle.Render(d.errStr))
	}
	lines = append(lines, "", theme.MutedStyle.Render("tab next field · enter save · esc cancel"))
	return box("Edit entry", lines, boxWidth)
}
