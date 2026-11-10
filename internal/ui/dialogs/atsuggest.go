package dialogs

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

// atSuggest is the inline project dropdown of a description input: "@" at a
// word start opens it, the text after the "@" is its search, and picking
// removes that text. While open it owns enter, tab, esc, and the selection
// keys, so the dialog's own enter/esc never run; other keys still edit the
// description.
type atSuggest struct {
	projects []toggl.Project
	open     bool
	at       int // rune index of the "@"
	list     ProjectList
	picked   Option
}

func newAtSuggest(projects []toggl.Project) atSuggest {
	return atSuggest{projects: projects}
}

func (s atSuggest) Open() bool { return s.open }

// Picked is the option chosen by the last PickPicked.
func (s atSuggest) Picked() Option { return s.picked }

// Update handles a key for the description input ti. While the dropdown is
// open it takes the list keys; any other key edits ti, then the dropdown
// opens or closes to match the text. hasCurrent offers "No project" when it
// opens. While closed, enter (and the dialog's other keys) are ignored here.
func (s *atSuggest) Update(ti *textinput.Model, k tea.KeyMsg, hasCurrent bool) (PickResult, tea.Cmd) {
	if s.open {
		if res, ok := listKey(&s.list, k); ok {
			switch res {
			case PickPicked:
				s.picked = s.list.Selected()
				s.splice(ti)
				s.open = false
			case PickCancelled:
				s.open = false
			}
			return res, nil
		}
	} else if k.String() == "enter" {
		return PickEditing, nil
	}
	var cmd tea.Cmd
	*ti, cmd = ti.Update(k)
	runes := []rune(ti.Value())
	pos := ti.Position()
	if s.open {
		if s.at >= len(runes) || runes[s.at] != '@' || pos <= s.at {
			s.open = false
		} else {
			s.list.SetQuery(string(runes[s.at+1 : pos]))
		}
		return PickEditing, cmd
	}
	// Only a single typed "@" opens it: a paste containing one doesn't.
	if k.Type == tea.KeyRunes && len(k.Runes) == 1 && k.Runes[0] == '@' {
		at := pos - 1
		if at >= 0 && runes[at] == '@' && (at == 0 || runes[at-1] == ' ') {
			s.open, s.at = true, at
			s.list = NewProjectList(s.projects, hasCurrent)
		}
	}
	return PickEditing, cmd
}

// splice removes "@search" from ti, joining what was around it with a
// single space (or none at either end), and leaves the cursor at the join.
func (s *atSuggest) splice(ti *textinput.Model) {
	runes := []rune(ti.Value())
	end := min(max(ti.Position(), s.at), len(runes))
	before := strings.TrimRight(string(runes[:s.at]), " ")
	after := strings.TrimLeft(string(runes[end:]), " ")
	value, cursor := before+after, len([]rune(before))
	if before != "" && after != "" {
		value, cursor = before+" "+after, cursor+1
	}
	ti.SetValue(value)
	ti.SetCursor(cursor)
}

// View renders the dropdown's list, width columns wide.
func (s atSuggest) View(width int) string { return s.list.View(width) }
