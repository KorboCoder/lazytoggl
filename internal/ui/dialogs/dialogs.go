// Package dialogs holds the popup dialogs (start/resume, edit, confirm).
// Dialogs communicate with the app purely through messages: they emit
// CloseMsg / *SubmitMsg commands, and the app owns opening and closing them.
package dialogs

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

type Dialog interface {
	Update(tea.Msg) (Dialog, tea.Cmd)
	View() string
}

// CloseMsg asks the app to close the open dialog without acting.
type CloseMsg struct{}

// StartSubmitMsg asks the app to start (or resume) an entry. A non-empty
// NewProject is created first and the entry started on it.
type StartSubmitMsg struct {
	Req        toggl.StartRequest
	NewProject string
}

// EditSubmitMsg asks the app to persist an edited entry. A non-empty
// NewProject is created first and the entry saved on it.
type EditSubmitMsg struct {
	Entry      toggl.TimeEntry
	NewProject string
}

// LogSubmitMsg asks the app to create a finished entry. A non-empty
// NewProject is created first and the entry logged on it.
type LogSubmitMsg struct {
	Req        toggl.LogRequest
	NewProject string
}

// DeleteConfirmMsg asks the app to delete an entry.
type DeleteConfirmMsg struct{ EntryID int64 }

func closeCmd() tea.Cmd { return func() tea.Msg { return CloseMsg{} } }

const boxWidth = 56

// box renders dialog content in a focused-style panel.
func box(title string, lines []string, width int) string {
	content := ""
	for i, l := range lines {
		if i > 0 {
			content += "\n"
		}
		content += l
	}
	return theme.Panel(title, content, width, len(lines)+2, true)
}
