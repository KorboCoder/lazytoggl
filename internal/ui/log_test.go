package ui

// Logging past entries (openspec change log-past-entry, spec entry-logging).
// fixedNow is Tue 14 Jul 2026 12:00, so the log dialog opens on 11:30-12:00.

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/ui/dialogs"
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
	d.dialogContains("Discard this entry?")
	d.dialogLacks("Save changes?") // no review in log mode
	d.key("y")
	if d.m.dialog != nil || d.mock.Calls("Log") != 0 {
		t.Fatalf("after y: dialog open %v, Log calls %d; want closed and none", d.m.dialog != nil, d.mock.Calls("Log"))
	}
}

func TestLogDiscardPromptBackKeepsForm(t *testing.T) {
	for _, back := range []string{"n", "esc"} {
		d := newDriver(t)
		openLog(d)
		d.keys("standup", "esc", back)
		d.dialogLacks("Discard this entry?")
		d.dialogContains("standup")
		d.dialogContains("Log entry")
	}
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
