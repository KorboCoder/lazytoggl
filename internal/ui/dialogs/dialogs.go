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

// StartSubmitMsg asks the app to start (or resume) an entry.
type StartSubmitMsg struct{ Req toggl.StartRequest }

// EditSubmitMsg asks the app to persist an edited entry.
type EditSubmitMsg struct{ Entry toggl.TimeEntry }

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
