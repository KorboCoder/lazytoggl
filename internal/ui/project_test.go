package ui

// Project picking (openspec change pick-entry-project, spec project-picking).
// Fixtures: projects ACME (101) and Internal (102); ID1 (running) is on
// ACME, ID2 has no project, ID3 (yesterday) is on Internal.

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

func TestStartDialogShowsResumedProject(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "enter") // resume ID3
	d.dialogContains("Resume entry")
	d.dialogContains("project")
	d.dialogContains("Internal")
}

func TestStartDialogTabToProjectRow(t *testing.T) {
	d := newDriver(t)
	d.key("c")
	d.dialogContains("> description")
	d.dialogContains("(none)")
	d.key("tab")
	d.dialogContains("> project")
	d.key("shift+tab")
	d.dialogContains("> description")
}

func TestStartPickProjectFromRow(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "Task", "tab", "enter") // open the project picker
	d.dialogContains("search or create")
	d.keys("acm", "enter")
	d.dialogContains("ACME")
	d.dialogLacks("search or create") // the picker closed
	d.keys("shift+tab", "enter")      // back on the description: enter starts
	run := d.m.runningEntry()
	if run == nil || run.Description != "Task" || run.ProjectID != 101 {
		t.Fatalf("running = %+v, want Task on ACME (101)", run)
	}
}

// While the @ dropdown is open, enter picks the project and doesn't start.
func TestStartAtPicksProjectWithoutStarting(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "fix", " ", "@", "int")
	d.dialogContains("Internal") // the dropdown lists it
	d.key("enter")
	if d.m.dialog == nil || d.mock.Calls("Start") != 0 {
		t.Fatalf("enter with the dropdown open started a timer (dialog %v, Start calls %d)", d.m.dialog != nil, d.mock.Calls("Start"))
	}
	d.dialogContains("Internal") // now on the project row
	d.dialogLacks("@int")
}

func TestStartAtThenEnterStarts(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "fix", " ", "@", "int", "enter", "enter")
	run := d.m.runningEntry()
	if run == nil || run.Description != "fix" || run.ProjectID != 102 {
		t.Fatalf("running = %+v, want fix on Internal (102)", run)
	}
}

func TestStartAtEscKeepsTextAndDialog(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "@", "ac", "esc")
	if d.m.dialog == nil {
		t.Fatal("esc with the dropdown open closed the dialog")
	}
	d.dialogContains("@ac")
	d.dialogLacks("↑/↓ choose") // the dropdown closed
	d.key("esc")                // a second esc closes the dialog
	if d.m.dialog != nil {
		t.Fatal("second esc did not close the dialog")
	}
}

// A (new) project is created on start, before the entry, then used for it.
func TestStartWithNewProjectCreatesThenStarts(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "x", " ", "@", "newclient", "enter")
	d.dialogContains("newclient (new)")
	if d.mock.Calls("CreateProject") != 0 {
		t.Fatal("picking the create option already created the project")
	}
	d.key("enter")
	if c, s := d.mock.Calls("CreateProject"), d.mock.Calls("Start"); c != 1 || s != 1 {
		t.Fatalf("CreateProject %d, Start %d calls; want 1 and 1", c, s)
	}
	run := d.m.runningEntry()
	if run == nil || run.Description != "x" || run.ProjectID == 0 {
		t.Fatalf("running = %+v, want x on the new project", run)
	}
	if p, ok := d.m.projects[run.ProjectID]; !ok || p.Name != "newclient" || !p.Active {
		t.Fatalf("project %d in the session = %+v (found %v), want active newclient", run.ProjectID, p, ok)
	}
}

func TestStartCancelWithNewProjectSendsNothing(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "@", "newclient", "enter", "esc")
	if d.m.dialog != nil {
		t.Fatal("esc did not close the dialog")
	}
	if n := d.mock.Calls("CreateProject") + d.mock.Calls("Start"); n != 0 {
		t.Fatalf("%d requests sent, want 0", n)
	}
}

// Edit dialog: the project row comes last, after duration (after start for
// a running entry).
func TestEditFormShowsProjectRow(t *testing.T) {
	d := openEditID3(t)
	d.dialogContains("project")
	d.dialogContains("Internal")
	d.keys("tab", "tab", "tab", "tab")
	d.dialogContains("> project")
}

func TestEditRunningFormShowsProjectRow(t *testing.T) {
	d := newDriver(t)
	d.key("e") // the running entry, on ACME
	d.dialogContains("ACME")
	d.keys("tab", "tab")
	d.dialogContains("> project")
	d.dialogLacks("stop")
}

func TestEditEnterOnProjectRowOpensPicker(t *testing.T) {
	d := openEditID3(t)
	d.keys("shift+tab", "enter") // wrap back from description to project
	d.dialogContains("search or create")
	d.dialogContains("No project") // the entry has a project to remove
}

func TestEditAtPicksProject(t *testing.T) {
	d := openEditID3(t) // on Internal
	d.keys(" ", "@", "acm", "enter")
	d.dialogContains("ACME") // the project row
	d.dialogLacks("@acm")
	d.dialogContains("Yesterday task") // the description, trimmed back
}

func TestEditAtEnterDoesNotMoveRow(t *testing.T) {
	d := openEditID3(t)
	d.keys(" ", "@", "acm", "enter")
	d.dialogContains("> description") // enter picked; it didn't move to start
}

func TestEditAtEscDoesNotOpenReview(t *testing.T) {
	d := openEditID3(t)
	d.keys(" ", "@", "ac", "esc")
	d.dialogContains("Edit entry")
	d.dialogLacks("Save changes?")
	d.dialogContains("@ac")
}

// openEditID2 opens the finished entry with no project ("Standup").
func openEditID2(t *testing.T) *driver {
	t.Helper()
	d := newDriver(t)
	d.keys("j", "e")
	d.dialogContains("Standup")
	return d
}

func TestEditReviewShowsProjectChange(t *testing.T) {
	d := openEditID3(t)
	d.keys(" ", "@", "acm", "enter", "esc")
	d.dialogContains("Save changes?")
	d.dialogContains("Internal -> ACME")
	d.dialogLacks("description") // " @acm" was removed again: no description change
}

func TestEditReviewShowsNewProject(t *testing.T) {
	d := openEditID2(t)
	d.keys(" ", "@", "newclient", "enter", "esc")
	d.dialogContains("(none) -> newclient (new)")
}

// pickNoProject opens ID3's project picker and picks "No project".
func pickNoProject(d *driver) {
	d.keys("shift+tab", "enter", "enter")
}

func TestEditReviewShowsProjectRemoved(t *testing.T) {
	d := openEditID3(t)
	pickNoProject(d)
	d.dialogContains("(none)")
	d.key("esc")
	d.dialogContains("Internal -> (none)")
}

func TestEditClearProjectSaves(t *testing.T) {
	d := openEditID3(t)
	pickNoProject(d)
	d.keys("esc", "y")
	if e, _ := d.entryByID(3); e.ProjectID != 0 {
		t.Fatalf("saved ProjectID = %d, want 0", e.ProjectID)
	}
}

func TestEditSaveWithNewProjectCreatesThenUpdates(t *testing.T) {
	d := openEditID2(t)
	d.keys(" ", "@", "newclient", "enter", "esc", "y")
	if c, u := d.mock.Calls("CreateProject"), d.mock.Calls("Update"); c != 1 || u != 1 {
		t.Fatalf("CreateProject %d, Update %d calls; want 1 and 1", c, u)
	}
	e, _ := d.entryByID(2)
	if p, ok := d.m.projects[e.ProjectID]; e.ProjectID == 0 || !ok || p.Name != "newclient" {
		t.Fatalf("saved ProjectID = %d (%+v), want the new newclient project", e.ProjectID, p)
	}
}

func TestEditDiscardWithNewProjectSendsNothing(t *testing.T) {
	d := openEditID2(t)
	d.keys(" ", "@", "newclient", "enter", "esc", "n")
	if n := d.mock.Calls("CreateProject") + d.mock.Calls("Update"); n != 0 {
		t.Fatalf("%d requests sent, want 0", n)
	}
}

// forbiddingService is the mock, except that creating a project is refused
// the way a workspace that lets only admins create projects refuses it.
type forbiddingService struct{ *toggl.Mock }

func (forbiddingService) CreateProject(ctx context.Context, name string) (toggl.Project, error) {
	return toggl.Project{}, toggl.ErrForbidden
}

func TestCreateProjectForbiddenShowsPermissionError(t *testing.T) {
	mock := toggl.NewMock()
	d := newDriverWith(t, forbiddingService{mock}, mock)
	d.keys("c", "x", " ", "@", "newclient", "enter", "enter")
	if !strings.Contains(d.m.notice, "admins") || !strings.Contains(d.m.notice, "refused") {
		t.Fatalf("notice = %q, want a refused/admins permission message", d.m.notice)
	}
	if strings.Contains(strings.ToLower(d.m.notice+d.m.status), "token") {
		t.Fatalf("notice %q / status %q mention the token", d.m.notice, d.m.status)
	}
	if s := mock.Calls("Start"); s != 0 {
		t.Fatalf("Start called %d times after a failed create, want 0", s)
	}
}

// failingStartService creates projects normally but every start fails.
type failingStartService struct{ *toggl.Mock }

func (f failingStartService) Start(ctx context.Context, req toggl.StartRequest) (toggl.TimeEntry, error) {
	return toggl.TimeEntry{}, toggl.ErrRateLimited
}

// The project made before a failed save is in the list from then on, so
// the retry picks it by name and nothing is created twice.
func TestCreatedProjectReusedAfterFailedSave(t *testing.T) {
	mock := toggl.NewMock()
	d := newDriverWith(t, failingStartService{mock}, mock)
	d.keys("c", "x", " ", "@", "newclient", "enter", "enter")
	if mock.Calls("CreateProject") != 1 || d.m.notice == "" || mock.Calls("Start") != 0 {
		t.Fatalf("CreateProject %d, Start %d, notice %q: want one create and a failed start",
			mock.Calls("CreateProject"), mock.Calls("Start"), d.m.notice)
	}
	d.keys("c", "@", "newclient")
	d.dialogContains("newclient")
	d.dialogLacks("+ Create")
}

// An open dropdown or picker grows the dialog rather than covering its help.
func TestProjectPopupsKeepHelpVisible(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "@")
	d.dialogContains("↑/↓ choose · enter pick · esc close")
	d.keys("esc", "esc", "c", "tab", "enter") // the start dialog's row picker
	d.dialogContains("↑/↓ choose · enter pick · esc close")
	d.keys("esc", "esc")
	e := openEditID3(t) // the edit dialog's dropdown and picker
	e.keys(" ", "@")
	e.dialogContains("↑/↓ choose · enter pick · esc close")
	e.keys("esc", "shift+tab", "enter")
	e.dialogContains("↑/↓ choose · enter pick · esc close")
}

// newUnloadedDriver is the app after its startup load failed (e.g. launched
// while rate-limited): no entries, no projects yet.
func newUnloadedDriver(t *testing.T) *driver {
	t.Helper()
	mock := toggl.NewMock()
	d := &driver{t: t, m: New(mock), mock: mock}
	mm, _ := d.m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	d.m = mm.(Model)
	mm, _ = d.m.Update(tickMsg(fixedNow))
	d.m = mm.(Model)
	d.feed(errMsg{toggl.ErrRateLimited})
	return d
}

// Refresh fetches projects while they have never loaded, so @ACME matches
// instead of offering to create a duplicate.
func TestRefreshLoadsProjectsWhenNeverLoaded(t *testing.T) {
	d := newUnloadedDriver(t)
	d.key("r")
	if n := d.mock.Calls("Projects"); n != 1 {
		t.Fatalf("Projects calls on refresh = %d, want 1 while never loaded", n)
	}
	d.keys("c", "@", "acm")
	d.dialogContains("ACME")
	d.key("r") // dialog open: r types; close and refresh again
	d.keys("esc", "esc", "esc", "r")
	if n := d.mock.Calls("Projects"); n != 1 {
		t.Fatalf("Projects calls after a second refresh = %d, want still 1", n)
	}
}

// Before projects load, the list says so and offers no create option.
func TestUnloadedProjectsOfferNoCreate(t *testing.T) {
	d := newUnloadedDriver(t)
	d.keys("c", "@", "acme")
	d.dialogContains("projects not loaded")
	d.dialogLacks("+ Create")
}

// A load that lands after a project was created keeps it.
func TestLateLoadKeepsCreatedProject(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "@", "newclient", "enter", "enter")
	run := d.m.runningEntry()
	d.feed(dataLoadedMsg{entries: fixtureEntries(), projects: []toggl.Project{
		{ID: 101, Name: "ACME", Active: true}, {ID: 102, Name: "Internal", Active: true},
	}})
	if _, ok := d.m.projects[run.ProjectID]; !ok {
		t.Fatalf("created project %d dropped by a later load", run.ProjectID)
	}
}

// While the entry starts after its project was created, the status says
// "starting…", not "saving…".
func TestStartWithNewProjectStatusSaysStarting(t *testing.T) {
	d := newDriver(t)
	d.keys("c", "@", "newclient", "enter")
	mm, cmd := d.m.Update(keyEnter()) // the dialog submits
	d.m = mm.(Model)
	mm, cmd = d.m.Update(cmd()) // StartSubmitMsg -> the create command
	d.m = mm.(Model)
	mm, _ = d.m.Update(cmd()) // projectCreatedMsg; next is not run
	d.m = mm.(Model)
	if d.m.status != "starting…" {
		t.Fatalf("status = %q, want starting…", d.m.status)
	}
}

func keyEnter() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }
