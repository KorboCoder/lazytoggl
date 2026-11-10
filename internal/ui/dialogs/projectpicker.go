package dialogs

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

// PickResult reports what a key did to a ProjectPicker or atSuggest.
type PickResult int

const (
	PickEditing   PickResult = iota // still open
	PickPicked                      // enter/tab on an option: see Picked
	PickCancelled                   // esc
)

// pickerWidth is the picker popup's inner width.
const pickerWidth = 30

// ProjectPicker is the popup a project row opens: its own search line over
// a ProjectList. It takes every key while open, like the date+time field.
type ProjectPicker struct {
	search textinput.Model
	list   ProjectList
	picked Option
}

func NewProjectPicker(projects []toggl.Project, hasCurrent bool) ProjectPicker {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "search or create"
	ti.CharLimit = 100
	ti.Width = pickerWidth - 3
	ti.Focus()
	return ProjectPicker{search: ti, list: NewProjectList(projects, hasCurrent)}
}

// Picked is the option chosen by the last PickPicked.
func (p ProjectPicker) Picked() Option { return p.picked }

func (p ProjectPicker) Update(k tea.KeyMsg) (ProjectPicker, PickResult) {
	if res, ok := listKey(&p.list, k); ok {
		if res == PickPicked {
			p.picked = p.list.Selected()
		}
		return p, res
	}
	before := p.search.Value()
	p.search, _ = p.search.Update(k)
	if v := p.search.Value(); v != before {
		p.list.SetQuery(v)
	}
	return p, PickEditing
}

// listKey handles the keys a project list owns while open: moving the
// selection, picking, and cancelling. ok is false for any other key.
func listKey(l *ProjectList, k tea.KeyMsg) (res PickResult, ok bool) {
	switch k.String() {
	case "up", "ctrl+p":
		l.Move(-1)
		return PickEditing, true
	case "down", "ctrl+n":
		l.Move(1)
		return PickEditing, true
	case "enter", "tab":
		if l.Empty() {
			return PickEditing, true
		}
		return PickPicked, true
	case "esc":
		return PickCancelled, true
	}
	return PickEditing, false
}

func (p ProjectPicker) View() string {
	return p.search.View() + "\n" + p.list.View(pickerWidth)
}
