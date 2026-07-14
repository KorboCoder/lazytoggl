package dialogs

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// Confirm is the delete-confirmation dialog.
type Confirm struct {
	entryID int64
	text    string
}

func NewConfirm(entryID int64, text string) *Confirm {
	return &Confirm{entryID: entryID, text: text}
}

func (d *Confirm) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "y", "enter":
			id := d.entryID
			return d, func() tea.Msg { return DeleteConfirmMsg{EntryID: id} }
		case "n", "esc":
			return d, closeCmd()
		}
	}
	return d, nil
}

func (d *Confirm) View() string {
	lines := []string{
		theme.Truncate(d.text, boxWidth-6),
		"",
		theme.MutedStyle.Render("y/enter delete · n/esc cancel"),
	}
	return box("Delete entry?", lines, boxWidth)
}
