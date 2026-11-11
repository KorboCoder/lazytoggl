package ui

// Logging past entries (openspec change log-past-entry, spec entry-logging).
// fixedNow is Tue 14 Jul 2026 12:00, so the log dialog opens on 11:30-12:00.

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/dialogs"
	"github.com/KorboCoder/lazytoggl/internal/ui/panels"
)

// openLog opens the log-entry dialog directly (the m key is wired in app.go).
func openLog(d *driver) {
	d.m.dialog = dialogs.NewLog(d.m.now, d.m.projectList())
	d.dialogContains("Log entry")
}

// dialogMsg sends key k to the open dialog and returns the message its
// command produces (nil if none).
func dialogMsg(d *driver, k tea.KeyMsg) tea.Msg {
	d.t.Helper()
	dlg, cmd := d.m.dialog.Update(k)
	d.m.dialog = dlg
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestLogDialogDefaults(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	for _, want := range []string{"> description", "Tue 14 Jul 11:30", "Tue 14 Jul 12:00", "00:30", "(none)"} {
		d.dialogContains(want)
	}
}

func TestLogDialogDefaultsAcrossMidnight(t *testing.T) {
	d := newDriver(t)
	mm, _ := d.m.Update(tickMsg(time.Date(2026, 7, 14, 0, 10, 27, 0, time.Local)))
	d.m = mm.(Model)
	openLog(d)
	d.dialogContains("Mon 13 Jul 23:40")
	d.dialogContains("Tue 14 Jul 00:10")
}

func keyEnterMsg() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }

func TestLogEnterOnDescriptionLogs(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.key("standup")
	msg, ok := dialogMsg(d, keyEnterMsg()).(dialogs.LogSubmitMsg)
	if !ok {
		t.Fatalf("enter on the description did not submit a LogSubmitMsg")
	}
	want := time.Date(2026, 7, 14, 11, 30, 0, 0, time.Local)
	if r := msg.Req; r.Description != "standup" || !r.Start.Equal(want) || !r.Stop.Equal(want.Add(30*time.Minute)) || r.ProjectID != 0 {
		t.Fatalf("request = %+v, want standup 11:30-12:00 with no project", r)
	}
}

func TestLogEnterOnTimeRowOpensField(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.keys("tab", "tab")
	if msg := dialogMsg(d, keyEnterMsg()); msg != nil {
		if _, logged := msg.(dialogs.LogSubmitMsg); logged {
			t.Fatal("enter on the stop row logged the entry")
		}
	}
	d.dialogContains("─ stop ")
}

func TestLogAtEnterPicksAndDoesNotLog(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.keys("standup", " ", "@", "int")
	if _, logged := dialogMsg(d, keyEnterMsg()).(dialogs.LogSubmitMsg); logged {
		t.Fatal("enter with the @ dropdown open logged the entry")
	}
	d.dialogContains("Internal")
	msg, ok := dialogMsg(d, keyEnterMsg()).(dialogs.LogSubmitMsg)
	if !ok || msg.Req.Description != "standup" || msg.Req.ProjectID != 102 {
		t.Fatalf("second enter = %+v (ok %v), want standup on Internal (102)", msg, ok)
	}
}

func TestLogEscUnchangedCloses(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatal("esc on an untouched log dialog did not close it")
	}
}

func TestLogEscAfterTypingAsksThenDiscards(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.keys("standup", "esc")
	d.dialogContains("Discard this entry? d discard · esc back")
	d.dialogLacks("Save changes?") // no review in log mode
	d.keys("y", "n")               // the edit review's keys do nothing here
	d.dialogContains("Discard this entry?")
	d.key("d")
	if d.m.dialog != nil || d.mock.Calls("Log") != 0 {
		t.Fatalf("after y: dialog open %v, Log calls %d; want closed and none", d.m.dialog != nil, d.mock.Calls("Log"))
	}
}

func TestLogDiscardPromptBackKeepsForm(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.keys("standup", "esc", "esc")
	d.dialogLacks("Discard this entry?")
	d.dialogContains("standup")
	d.dialogContains("Log entry")
}

func TestLogDiscardPromptIgnoresOtherKeys(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.keys("standup", "esc", "x", "enter")
	d.dialogContains("Discard this entry?")
	d.dialogLacks("standupx")
	if d.mock.Calls("Log") != 0 {
		t.Fatal("a key on the discard prompt logged the entry")
	}
}

func TestLogKeyOpensDialogFromList(t *testing.T) {
	d := newDriver(t)
	d.key("m")
	d.dialogContains("Log entry")
	d.dialogContains("Tue 14 Jul 11:30")
}

func TestLogKeyOpensDialogFromTimer(t *testing.T) {
	d := newDriver(t)
	d.keys("tab", "m") // focus the timer panel
	d.dialogContains("Log entry")
}

func TestLogKeyShownInHelp(t *testing.T) {
	d := newDriver(t)
	if v := d.m.View(); !strings.Contains(v, "m log") || !strings.Contains(v, "? help") {
		t.Fatalf("help bar lacks \"m log\" or \"? help\" at width 100:\n%s", v)
	}
}

// loggedEntry finds the entry logged as desc.
func loggedEntry(t *testing.T, d *driver, desc string) toggl.TimeEntry {
	t.Helper()
	for _, e := range d.m.entries {
		if e.Description == desc && e.ID > 3 {
			return e
		}
	}
	t.Fatalf("no logged entry %q in %+v", desc, d.m.entries)
	return toggl.TimeEntry{}
}

func TestLogCreatesFinishedEntry(t *testing.T) {
	d := newDriver(t)
	d.keys("m", "standup", "enter")
	if d.m.dialog != nil || d.mock.Calls("Log") != 1 {
		t.Fatalf("dialog open %v, Log calls %d; want closed and 1", d.m.dialog != nil, d.mock.Calls("Log"))
	}
	e := loggedEntry(t, d, "standup")
	if e.Running() || !e.Start.Equal(time.Date(2026, 7, 14, 11, 30, 0, 0, time.Local)) || e.Duration != 1800 {
		t.Fatalf("logged = %+v, want finished 11:30-12:00", e)
	}
	if h := headerAbove(t, d, e.ID); h != "Today" {
		t.Fatalf("logged entry listed under %q, want Today", h)
	}
	wantSelected(t, d, e.ID)
	if d.m.status != "entry logged" {
		t.Fatalf("status = %q, want entry logged", d.m.status)
	}
}

func TestLogKeepsTimerRunning(t *testing.T) {
	d := newDriver(t)
	d.keys("m", "standup", "enter")
	if run := d.m.runningEntry(); run == nil || run.ID != 1 {
		t.Fatalf("running entry = %+v, want ID1 still running", run)
	}
	entries, _ := d.mock.RecentEntries(context.Background())
	for _, e := range entries {
		if e.ID == 1 && !e.Running() {
			t.Fatal("the mock's running entry was stopped by a log")
		}
	}
}

// headerAbove is the day header the entry with id is listed under.
func headerAbove(t *testing.T, d *driver, id int64) string {
	t.Helper()
	header := ""
	for _, r := range panels.BuildRows(d.m.entries, d.m.now) {
		if r.Kind == panels.RowHeader {
			header = r.Label
		} else if r.Entry.ID == id {
			return header
		}
	}
	t.Fatalf("entry %d not listed", id)
	return ""
}

// Every row's help says esc cancels, since in log mode esc on a changed
// draft leads to the discard prompt, never to "done".
func TestLogHelpSaysCancelOnEveryRow(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	for i := 0; i < 5; i++ {
		d.dialogContains("esc cancel")
		d.dialogLacks("esc done")
		d.key("tab")
	}
}

// With the @ dropdown open, esc only closes it; a second esc asks.
func TestLogEscWithDropdownOnlyClosesIt(t *testing.T) {
	d := newDriver(t)
	openLog(d)
	d.keys("x", " ", "@", "ac", "esc")
	d.dialogContains("Log entry")
	d.dialogLacks("Discard this entry?")
	d.key("esc")
	d.dialogContains("Discard this entry?")
}

func TestLogWithNewProjectCreatesThenLogs(t *testing.T) {
	d := newDriver(t)
	d.keys("m", "x", " ", "@", "newclient", "enter", "enter")
	if c, l := d.mock.Calls("CreateProject"), d.mock.Calls("Log"); c != 1 || l != 1 {
		t.Fatalf("CreateProject %d, Log %d calls; want 1 and 1", c, l)
	}
	e := loggedEntry(t, d, "x")
	if p, ok := d.m.projects[e.ProjectID]; !ok || p.Name != "newclient" {
		t.Fatalf("logged on project %d (%+v), want the new newclient", e.ProjectID, p)
	}
}

// The fetch window starts Mon 06 Jul 12:00 (fixedNow - 192h).
func TestLogOutsideWindowShowsStatus(t *testing.T) {
	d := newDriver(t)
	d.keys("m", "old", "tab", "enter", "tab") // start field, date segment
	for i := 0; i < 9; i++ {
		d.key("ctrl+x") // Sun 05 Jul; the stop moves with it
	}
	d.keys("enter", "shift+tab", "enter")
	for _, want := range []string{"entry logged", "Sun, Jul 5", "outside the 8-day list", "drops off on refresh"} {
		if !strings.Contains(d.m.status, want) {
			t.Fatalf("status = %q, want it to contain %q", d.m.status, want)
		}
	}
	loggedEntry(t, d, "old") // still listed until a refresh
}

func TestLogAfterNowShowsStatus(t *testing.T) {
	d := newDriver(t)
	d.keys("m", "later")
	d.keys("tab", "tab", "enter", "1", "5", "0", "0", "enter")             // stop 15:00
	d.keys("shift+tab", "enter", "1", "3", "0", "0", "enter", "shift+tab") // start 13:00
	d.key("enter")
	if !strings.Contains(d.m.status, "entry logged — starts after now") {
		t.Fatalf("status = %q, want the after-now warning", d.m.status)
	}
}
