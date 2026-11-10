package dialogs

import (
	"sort"
	"strings"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// OptionKind is what a project list option does when picked.
type OptionKind int

const (
	OptionProject OptionKind = iota // set this project
	OptionNone                      // remove the entry's project
	OptionCreate                    // create a project named Name on save
)

// Option is one row of a project list.
type Option struct {
	Kind    OptionKind
	Project toggl.Project // OptionProject
	Name    string        // OptionCreate: the trimmed search
}

// listRows is how many options a project list shows at once.
const listRows = 6

// ProjectList is the project list shared by the row picker and the inline @
// dropdown: ordering, filtering, the create option, selection, and its view.
// It has no input of its own; each host feeds it a query.
type ProjectList struct {
	active     []toggl.Project // active projects, A-Z ignoring case
	names      map[string]bool // lower-cased names of all projects, archived too
	hasCurrent bool            // the entry has a project: offer "No project"
	loaded     bool            // projects are known; without them, create is unsafe
	query      string
	options    []Option
	sel        int
	offset     int
}

// NewProjectList lists projects for an entry; hasCurrent offers "No project"
// on an empty search. A nil projects means they haven't loaded: nothing is
// offered for creation, since an existing project could not be told apart.
func NewProjectList(projects []toggl.Project, hasCurrent bool) ProjectList {
	l := ProjectList{names: map[string]bool{}, hasCurrent: hasCurrent, loaded: projects != nil}
	for _, p := range projects {
		l.names[normName(p.Name)] = true
		if p.Active {
			l.active = append(l.active, p)
		}
	}
	// A-Z ignoring case; ties broken by name, then ID, so the order never
	// depends on the order projects came in (a map, in the app).
	sort.Slice(l.active, func(i, j int) bool {
		a, b := l.active[i], l.active[j]
		if la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name); la != lb {
			return la < lb
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	l.rebuild()
	return l
}

func normName(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// SetQuery filters the list and puts the selection back on the first option.
func (l *ProjectList) SetQuery(q string) {
	l.query = q
	l.sel, l.offset = 0, 0
	l.rebuild()
}

func (l *ProjectList) rebuild() {
	l.options = l.options[:0]
	if l.query == "" {
		if l.hasCurrent {
			l.options = append(l.options, Option{Kind: OptionNone})
		}
		for _, p := range l.active {
			l.options = append(l.options, Option{Kind: OptionProject, Project: p})
		}
		return
	}
	q := strings.ToLower(l.query)
	for _, p := range l.active {
		if strings.Contains(strings.ToLower(p.Name), q) {
			l.options = append(l.options, Option{Kind: OptionProject, Project: p})
		}
	}
	if name := strings.TrimSpace(l.query); l.loaded && name != "" && !l.names[normName(name)] {
		l.options = append(l.options, Option{Kind: OptionCreate, Name: name})
	}
}

func (l ProjectList) Options() []Option { return l.options }

// Move steps the selection, stopping at either end.
func (l *ProjectList) Move(by int) {
	if len(l.options) == 0 {
		return
	}
	l.sel = max(0, min(len(l.options)-1, l.sel+by))
	if l.sel < l.offset {
		l.offset = l.sel
	}
	if l.sel >= l.offset+listRows {
		l.offset = l.sel - listRows + 1
	}
}

// Selected is the highlighted option, or an Option of no valid Kind when
// the list is empty (an exact archived name, or projects not loaded).
func (l ProjectList) Selected() Option {
	if len(l.options) == 0 {
		return Option{Kind: -1}
	}
	return l.options[l.sel]
}

// Empty reports whether there is nothing to pick.
func (l ProjectList) Empty() bool { return len(l.options) == 0 }

func optionLabel(o Option) string {
	switch o.Kind {
	case OptionNone:
		return "No project"
	case OptionCreate:
		return `+ Create "` + o.Name + `"`
	}
	return o.Project.Name
}

// View renders up to listRows options, width columns wide, with a marker on
// the selection.
func (l ProjectList) View(width int) string {
	if len(l.options) == 0 {
		msg := "no matching project"
		switch {
		case !l.loaded:
			msg = "projects not loaded — press r"
		case l.names[normName(l.query)]:
			msg = "archived project — not selectable"
		}
		return theme.MutedStyle.Render(theme.Truncate(msg, width))
	}
	end := min(len(l.options), l.offset+listRows)
	lines := make([]string, 0, end-l.offset)
	for i := l.offset; i < end; i++ {
		label := theme.Truncate(optionLabel(l.options[i]), width-2)
		if i == l.sel {
			lines = append(lines, theme.CursorStyle.Render("> "+label))
			continue
		}
		lines = append(lines, "  "+label)
	}
	return strings.Join(lines, "\n")
}
