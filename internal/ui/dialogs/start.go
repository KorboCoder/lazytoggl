package dialogs

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// Start is the start/resume dialog: a description input, with the project
// carried over unchanged when resuming (project picking is post-MVP).
type Start struct {
	title       string
	input       textinput.Model
	projectID   int64
	projectName string
}

func NewStart(title, description string, projectID int64, projectName string) *Start {
	ti := textinput.New()
	ti.Placeholder = "what are you working on?"
	ti.CharLimit = 200
	ti.Width = boxWidth - 8
	ti.SetValue(description)
	ti.CursorEnd()
	ti.Focus()
	return &Start{title: title, input: ti, projectID: projectID, projectName: projectName}
}

func (d *Start) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "enter":
			req := toggl.StartRequest{
				Description: strings.TrimSpace(d.input.Value()),
				ProjectID:   d.projectID,
			}
			return d, func() tea.Msg { return StartSubmitMsg{Req: req} }
		case "esc":
			return d, closeCmd()
		}
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

func (d *Start) View() string {
	lines := []string{d.input.View()}
	if d.projectName != "" {
		lines = append(lines, theme.MutedStyle.Render("project: ")+d.projectName)
	}
	lines = append(lines, "", theme.MutedStyle.Render("enter start · esc cancel"))
	return box(d.title, lines, boxWidth)
}
