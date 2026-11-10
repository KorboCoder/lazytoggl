package dialogs

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// projectChoice is a dialog's project: an existing one (ID), none (both
// zero), or one to create on save (NewName).
type projectChoice struct {
	ID      int64
	NewName string
}

func (c projectChoice) set() bool { return c.ID != 0 || c.NewName != "" }

// chose applies a picked option.
func chose(o Option) projectChoice {
	switch o.Kind {
	case OptionProject:
		return projectChoice{ID: o.Project.ID}
	case OptionCreate:
		return projectChoice{NewName: o.Name}
	}
	return projectChoice{}
}

// projectLabel is how a choice reads on a project row and in the review.
func projectLabel(c projectChoice, projects []toggl.Project) string {
	if c.NewName != "" {
		return c.NewName + " (new)"
	}
	if c.ID == 0 {
		return "(none)"
	}
	for _, p := range projects {
		if p.ID == c.ID {
			return p.Name
		}
	}
	return "(unknown project)"
}

// popupWidth is the outer width of the project dropdown and picker.
const popupWidth = pickerWidth + 4

// projectPopup boxes a project list (or picker) view for overlaying.
func projectPopup(title, content string) string {
	return theme.Panel(title, content, popupWidth, strings.Count(content, "\n")+3, true)
}

// overlayInBox draws pop over a dialog's content lines, its top border on
// content line top, growing the dialog (blank lines before its last line,
// the help) when the popup would run past the bottom. It returns the boxed
// dialog with the popup over it.
func overlayInBox(title string, lines []string, pop string, top int) string {
	need := top + lipgloss.Height(pop)
	for len(lines) < need {
		lines = append(lines[:len(lines)-1], "", lines[len(lines)-1])
	}
	view := box(title, lines, boxWidth)
	// Content line i is view line i+1 (the top border), starting at column 2.
	return theme.Overlay(view, pop, 2, top+1)
}
