package dialogs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// Start is the start/resume dialog: a description and a project row. The
// project is picked from the row's picker, or by typing "@" in the
// description.
type Start struct {
	title    string
	input    textinput.Model
	at       atSuggest
	projects []toggl.Project
	project  projectChoice
	focus    int // 0 description, 1 project
	// pickerOpen is set while the project row's picker is open; it then
	// takes every key.
	pickerOpen bool
	picker     ProjectPicker
}

func NewStart(title, description string, projectID int64, projects []toggl.Project) *Start {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "what are you working on?"
	ti.CharLimit = 200
	ti.Width = boxWidth - 8
	ti.SetValue(description)
	ti.CursorEnd()
	ti.Focus()
	return &Start{
		title: title, input: ti, at: newAtSuggest(projects),
		projects: projects, project: projectChoice{ID: projectID},
	}
}

func (d *Start) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	k, isKey := msg.(tea.KeyMsg)
	if !isKey {
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return d, cmd
	}
	if d.pickerOpen {
		var res PickResult
		d.picker, res = d.picker.Update(k)
		if res == PickPicked {
			d.project = chose(d.picker.Picked())
		}
		if res != PickEditing {
			d.pickerOpen = false
		}
		return d, nil
	}
	if d.focus == 0 && d.at.Open() {
		return d, d.updateAt(k)
	}
	switch k.String() {
	case "esc":
		return d, closeCmd()
	case "tab", "shift+tab", "up", "down":
		return d, d.setFocus(1 - d.focus)
	case "enter":
		if d.focus == 1 {
			d.picker = NewProjectPicker(d.projects, d.project.set())
			d.pickerOpen = true
			return d, nil
		}
		return d, d.submitCmd()
	}
	if d.focus == 1 {
		return d, nil
	}
	return d, d.updateAt(k)
}

// updateAt sends a description key through the @ suggester.
func (d *Start) updateAt(k tea.KeyMsg) tea.Cmd {
	res, cmd := d.at.Update(&d.input, k, d.project.set())
	if res == PickPicked {
		d.project = chose(d.at.Picked())
	}
	return cmd
}

func (d *Start) setFocus(i int) tea.Cmd {
	d.focus = i
	if i == 0 {
		return d.input.Focus()
	}
	d.input.Blur()
	return nil
}

func (d *Start) submitCmd() tea.Cmd {
	req := toggl.StartRequest{Description: strings.TrimSpace(d.input.Value()), ProjectID: d.project.ID}
	newProject := d.project.NewName
	return func() tea.Msg { return StartSubmitMsg{Req: req, NewProject: newProject} }
}

func (d *Start) rowLine(i int, label, value string) string {
	label = fmt.Sprintf("%-11s", label)
	if i == d.focus {
		return theme.CursorStyle.Render("> "+label) + value
	}
	return theme.MutedStyle.Render("  "+label) + value
}

func (d *Start) View() string {
	var help string
	switch {
	case d.pickerOpen, d.at.Open():
		help = "↑/↓ choose · enter pick · esc close"
	case d.focus == 1:
		help = "enter pick project · tab description · esc cancel"
	default:
		help = "enter start · @ project · tab next · esc cancel"
	}
	lines := []string{
		d.rowLine(0, "description", ""),
		"  " + d.input.View(),
		"",
		d.rowLine(1, "project", projectLabel(d.project, d.projects)),
		"",
		theme.MutedStyle.Render(help),
	}
	switch {
	case d.at.Open():
		return overlayInBox(d.title, lines, projectPopup("project", d.at.View(pickerWidth)), 2)
	case d.pickerOpen:
		return overlayInBox(d.title, lines, projectPopup("project", d.picker.View()), 4)
	}
	return box(d.title, lines, boxWidth)
}
