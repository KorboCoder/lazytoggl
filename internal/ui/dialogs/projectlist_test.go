package dialogs

import (
	"slices"
	"strings"
	"testing"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

// listProjects is the shared fixture: three active projects in no particular
// order, and one archived.
var listProjects = []toggl.Project{
	{ID: 102, Name: "Internal", Active: true},
	{ID: 101, Name: "ACME", Active: true},
	{ID: 104, Name: "acme website", Active: true},
	{ID: 103, Name: "Old Client", Active: false},
}

// labels renders a list's options as the text a user would read.
func labels(l ProjectList) []string {
	var out []string
	for _, o := range l.Options() {
		switch o.Kind {
		case OptionNone:
			out = append(out, "No project")
		case OptionProject:
			out = append(out, o.Project.Name)
		case OptionCreate:
			out = append(out, `+ Create "`+o.Name+`"`)
		}
	}
	return out
}

func wantLabels(t *testing.T, l ProjectList, want ...string) {
	t.Helper()
	if got := labels(l); !slices.Equal(got, want) {
		t.Fatalf("options = %q, want %q", got, want)
	}
}

func TestProjectListEmptySearchOrder(t *testing.T) {
	l := NewProjectList(listProjects, true)
	wantLabels(t, l, "No project", "ACME", "acme website", "Internal")
}

func TestProjectListNoProjectOnlyWithCurrent(t *testing.T) {
	l := NewProjectList(listProjects, false)
	wantLabels(t, l, "ACME", "acme website", "Internal")
}

func TestProjectListFiltersBySubstring(t *testing.T) {
	l := NewProjectList(listProjects, true)
	l.SetQuery("web")
	wantLabels(t, l, "acme website", `+ Create "web"`)
	l.SetQuery("INT")
	wantLabels(t, l, "Internal", `+ Create "INT"`)
}

func TestProjectListSpacesArePartOfSearch(t *testing.T) {
	l := NewProjectList(append(slices.Clone(listProjects), toggl.Project{ID: 105, Name: "Acme Internal", Active: true}), false)
	l.SetQuery("acme int")
	wantLabels(t, l, "Acme Internal", `+ Create "acme int"`)
}

func TestProjectListCreateWhenNothingMatches(t *testing.T) {
	l := NewProjectList(listProjects, true)
	l.SetQuery("newclient")
	wantLabels(t, l, `+ Create "newclient"`)
}

func TestProjectListCreateAfterPartialMatches(t *testing.T) {
	l := NewProjectList(listProjects, false)
	l.SetQuery("acme w")
	wantLabels(t, l, "acme website", `+ Create "acme w"`)
}

func TestProjectListNoCreateOnExactMatch(t *testing.T) {
	l := NewProjectList(listProjects, false)
	l.SetQuery("Acme") // case ignored
	wantLabels(t, l, "ACME", "acme website")
	// A trailing space (typing on toward "acme website") still counts as an
	// exact match for create, while the space narrows the list.
	l.SetQuery("acme ")
	wantLabels(t, l, "acme website")
}

func TestProjectListNoCreateForArchivedName(t *testing.T) {
	l := NewProjectList(listProjects, false)
	l.SetQuery("old client")
	wantLabels(t, l) // archived: neither listed nor offered for creation
}

func TestProjectListCreateNameIsTrimmed(t *testing.T) {
	l := NewProjectList(listProjects, false)
	l.SetQuery("  newclient ")
	if o := l.Selected(); o.Kind != OptionCreate || o.Name != "newclient" {
		t.Fatalf("selected = %+v, want create newclient", o)
	}
}

func TestProjectListSelectionResetsOnQuery(t *testing.T) {
	l := NewProjectList(listProjects, true)
	l.Move(2)
	l.SetQuery("a")
	if o := l.Selected(); o.Kind != OptionProject || o.Project.Name != "ACME" {
		t.Fatalf("selected = %+v, want the first match after a new query", o)
	}
}

func TestProjectListMoveClamps(t *testing.T) {
	l := NewProjectList(listProjects, false)
	l.Move(-1)
	if o := l.Selected(); o.Project.Name != "ACME" {
		t.Fatalf("selected = %+v, want ACME at the top", o)
	}
	l.Move(10)
	if o := l.Selected(); o.Project.Name != "Internal" {
		t.Fatalf("selected = %+v, want Internal at the bottom", o)
	}
}

func TestProjectListViewScrollsToSelection(t *testing.T) {
	var many []toggl.Project
	for _, n := range []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8"} {
		many = append(many, toggl.Project{ID: int64(len(many) + 1), Name: n, Active: true})
	}
	l := NewProjectList(many, false)
	if v := l.View(20); strings.Count(v, "\n")+1 != listRows || !strings.Contains(v, "a1") || strings.Contains(v, "a7") {
		t.Fatalf("view = %q, want the first %d rows", v, listRows)
	}
	l.Move(7) // a8
	v := l.View(20)
	if !strings.Contains(v, "a8") || strings.Contains(v, "a1") {
		t.Fatalf("view = %q, want it scrolled to a8", v)
	}
	if !strings.Contains(v, "> a8") {
		t.Fatalf("view = %q, want a marker on the selection", v)
	}
}
