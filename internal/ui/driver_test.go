package ui

// A synchronous test driver for the app model. Unlike teatest (which runs the
// real event loop against a PTY), the driver calls Update directly and
// executes returned commands inline, feeding back only the app's own message
// types (ticks and cursor blinks are dropped). Combined with fixture entries
// pinned to a fixed clock, every flow is deterministic regardless of when the
// suite runs — no day-boundary or ordering flakes.

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/dialogs"
)

// fixedNow pins day grouping: fixtures are built around this instant, so row
// positions never depend on the wall clock.
var fixedNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)

// Fixture IDs deliberately match entries seeded in the mock so that mutation
// commands (Stop/Update/Delete) resolve against the service. Update and
// Delete only need the ID to resolve; Stop is different — it takes just an
// ID and hands back whatever the mock has stored for it, so entry 1's
// description/project below must match toggl.NewMock()'s own seed for ID 1
// ("Writing project plan", ACME), or a pause/stop round-trip would silently
// overwrite the fixture's description with the mock's.
//
// Row layout: 0 "Today" header · 1 running ID1 · 2 finished ID2 ·
// 3 "Yesterday" header · 4 finished ID3. After load the cursor clamps off
// the header onto the running entry (row 1).
func fixtureEntries() []toggl.TimeEntry {
	return []toggl.TimeEntry{
		{ID: 1, WorkspaceID: 1, ProjectID: 101, Description: "Writing project plan",
			Start: fixedNow.Add(-10 * time.Minute), Duration: -1},
		{ID: 2, WorkspaceID: 1, Description: "Standup",
			Start: fixedNow.Add(-2 * time.Hour), Stop: fixedNow.Add(-105 * time.Minute), Duration: 15 * 60},
		{ID: 3, WorkspaceID: 1, ProjectID: 102, Description: "Yesterday task",
			Start: fixedNow.AddDate(0, 0, -1), Stop: fixedNow.AddDate(0, 0, -1).Add(time.Hour), Duration: 3600},
	}
}

type driver struct {
	t    *testing.T
	m    Model
	mock *toggl.Mock
}

func newDriver(t *testing.T) *driver {
	t.Helper()
	mock := toggl.NewMock()
	return newDriverWith(t, mock, mock)
}

// newDriverWith drives the app over svc; mock is the Mock inside it, for
// call counts.
func newDriverWith(t *testing.T, svc toggl.Service, mock *toggl.Mock) *driver {
	t.Helper()
	d := &driver{t: t, m: New(svc), mock: mock}
	mm, _ := d.m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	d.m = mm.(Model)
	mm, _ = d.m.Update(tickMsg(fixedNow)) // discard the re-arm tick cmd
	d.m = mm.(Model)
	d.feed(dataLoadedMsg{entries: fixtureEntries(), projects: []toggl.Project{
		{ID: 101, Name: "ACME", Color: "#e36a00", Active: true},
		{ID: 102, Name: "Internal", Color: "#2da608", Active: true},
	}})
	return d
}

func (d *driver) press(msg tea.KeyMsg) {
	d.t.Helper()
	hadDialog := d.m.dialog != nil
	mm, cmd := d.m.Update(msg)
	d.m = mm.(Model)
	// Text typed into a dialog input only yields cursor-blink tea.Ticks;
	// executing those would sleep ~530ms per keystroke. Only dialog keys that
	// can emit app messages (submit/close/confirm) are worth executing.
	if hadDialog && d.m.dialog != nil {
		switch msg.String() {
		case "enter", "esc", "y", "n":
		default:
			return
		}
	}
	d.exec(cmd)
}

func (d *driver) exec(cmd tea.Cmd) {
	d.t.Helper()
	if cmd == nil {
		return
	}
	d.feed(cmd())
}

func (d *driver) feed(msg tea.Msg) {
	d.t.Helper()
	switch msg.(type) {
	case dataLoadedMsg, entryStartedMsg, entryStoppedMsg, entryPausedMsg, entryUpdatedMsg,
		entryDeletedMsg, errMsg, projectCreatedMsg,
		dialogs.CloseMsg, dialogs.StartSubmitMsg, dialogs.EditSubmitMsg,
		dialogs.DeleteConfirmMsg:
		mm, cmd := d.m.Update(msg)
		d.m = mm.(Model)
		d.exec(cmd)
	default:
		// ticks, blinks, nil — the driver stays synchronous
	}
}

func (d *driver) key(s string) {
	d.t.Helper()
	switch s {
	case "enter":
		d.press(tea.KeyMsg{Type: tea.KeyEnter})
	case "esc":
		d.press(tea.KeyMsg{Type: tea.KeyEsc})
	case "tab":
		d.press(tea.KeyMsg{Type: tea.KeyTab})
	case "ctrl+u":
		d.press(tea.KeyMsg{Type: tea.KeyCtrlU})
	case "ctrl+a":
		d.press(tea.KeyMsg{Type: tea.KeyCtrlA})
	case "ctrl+x":
		d.press(tea.KeyMsg{Type: tea.KeyCtrlX})
	case "shift+tab":
		d.press(tea.KeyMsg{Type: tea.KeyShiftTab})
	case "up":
		d.press(tea.KeyMsg{Type: tea.KeyUp})
	case "down":
		d.press(tea.KeyMsg{Type: tea.KeyDown})
	default:
		d.press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	}
}

func (d *driver) keys(ks ...string) {
	for _, k := range ks {
		d.key(k)
	}
}

func (d *driver) entryByID(id int64) (toggl.TimeEntry, bool) {
	for _, e := range d.m.entries {
		if e.ID == id {
			return e, true
		}
	}
	return toggl.TimeEntry{}, false
}

func (d *driver) dialogLacks(unwanted string) {
	d.t.Helper()
	if d.m.dialog == nil {
		d.t.Fatalf("no dialog open, wanted one without %q", unwanted)
	}
	if v := d.m.dialog.View(); strings.Contains(v, unwanted) {
		d.t.Fatalf("dialog unexpectedly contains %q:\n%s", unwanted, v)
	}
}

func (d *driver) dialogContains(want string) {
	d.t.Helper()
	if d.m.dialog == nil {
		d.t.Fatalf("no dialog open, wanted one containing %q", want)
	}
	if v := d.m.dialog.View(); !strings.Contains(v, want) {
		d.t.Fatalf("dialog does not contain %q:\n%s", want, v)
	}
}

// Flows.

func TestResumePrefillsAndStarts(t *testing.T) {
	d := newDriver(t)
	d.key("j") // running ID1 -> finished ID2 ("Standup")
	d.key("enter")

	if _, ok := d.m.dialog.(*dialogs.Start); !ok {
		t.Fatalf("dialog = %T, want *dialogs.Start", d.m.dialog)
	}
	d.dialogContains("Resume entry")
	d.dialogContains("Standup")

	d.key("enter") // accept the prefilled description

	run := d.m.runningEntry()
	if run == nil || run.Description != "Standup" {
		t.Fatalf("running entry = %+v, want resumed Standup", run)
	}
	old, _ := d.entryByID(1)
	if old.Running() {
		t.Fatal("previously running entry was not stopped locally")
	}
	if !old.Stop.Equal(run.Start) {
		t.Fatalf("old entry stop = %v, want new entry start %v", old.Stop, run.Start)
	}
	if len(d.m.entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(d.m.entries))
	}
}

func TestEnterOnRunningEntryStops(t *testing.T) {
	d := newDriver(t)
	d.key("enter") // cursor starts on running ID1

	if d.m.dialog != nil {
		t.Fatal("enter on the running entry must stop it, not open a dialog")
	}
	if d.m.runningEntry() != nil {
		t.Fatal("timer still running")
	}
}

func TestCreateBindingOpensStartDialog(t *testing.T) {
	d := newDriver(t)
	d.key("c")
	d.dialogContains("Start entry")
	d.keys("Fresh task", "enter")

	run := d.m.runningEntry()
	if run == nil || run.Description != "Fresh task" {
		t.Fatalf("running entry = %+v, want Fresh task", run)
	}
	if old, _ := d.entryByID(1); old.Running() {
		t.Fatal("old running entry not stopped when the new one started")
	}

	// c works from the Timer panel too.
	d.keys("tab", "c")
	d.dialogContains("Start entry")
	d.key("esc")
}

func TestSpacePausesAndResumes(t *testing.T) {
	d := newDriver(t)

	d.key(" ") // pause: stops the entry via the API but pins it to the Timer panel
	if d.m.runningEntry() != nil {
		t.Fatal("timer still running after space")
	}
	paused := d.m.pausedEntry()
	if paused == nil || paused.Description != "Writing project plan" {
		t.Fatalf("paused entry = %+v, want the stopped 'Writing project plan' entry pinned", paused)
	}
	if view := d.m.View(); !strings.Contains(view, "⏸") || !strings.Contains(view, "Writing project plan") {
		t.Fatalf("Timer panel does not show the paused entry with its indicator:\n%s", view)
	}

	d.key(" ") // resume: continues the paused entry
	run := d.m.runningEntry()
	if run == nil || run.Description != paused.Description || run.ProjectID != paused.ProjectID {
		t.Fatalf("resumed entry = %+v, want copy of %+v", run, paused)
	}
	if d.m.pausedEntry() != nil {
		t.Fatal("paused state not cleared after resuming")
	}
	if len(d.m.entries) != 4 {
		t.Fatalf("entries = %d, want 4 (resume creates a new entry)", len(d.m.entries))
	}
}

// TestSpaceNoOpWhenIdle: space only toggles what the Timer panel is already
// showing. It must never reach into history to resume something that was
// never running or paused in the first place.
func TestSpaceNoOpWhenIdle(t *testing.T) {
	d := newDriver(t)
	d.key("enter") // stop the running entry outright (not pause) — panel goes idle

	if d.m.runningEntry() != nil || d.m.pausedEntry() != nil {
		t.Fatal("setup failed: expected a fully idle Timer panel")
	}
	before := len(d.m.entries)
	beforeStatus := d.m.status

	d.key(" ")

	if d.m.runningEntry() != nil || d.m.pausedEntry() != nil {
		t.Fatal("space started an entry from an idle Timer panel")
	}
	if len(d.m.entries) != before {
		t.Fatalf("entries = %d, want %d unchanged (space must not create an entry)", len(d.m.entries), before)
	}
	if d.m.status != beforeStatus {
		t.Fatalf("status changed to %q on an idle space press, want unchanged %q", d.m.status, beforeStatus)
	}
}

func TestTimerFocusActsOnRunningEntry(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "tab") // select "Standup" in the list, then focus the Timer panel

	// e edits the running entry, not the list selection.
	d.key("e")
	d.dialogContains("Edit entry")
	d.dialogContains("Writing project plan")
	d.key("esc")

	// d deletes the running entry, not the list selection.
	d.key("d")
	d.dialogContains("Delete entry?")
	d.dialogContains("Writing project plan")
	d.key("n")

	// Pausing keeps the entry pinned to the Timer panel, so e/d still target
	// it — not whatever the list has selected — while paused.
	d.key(" ")
	if d.m.runningEntry() != nil {
		t.Fatal("timer still running after pause")
	}
	d.key("e")
	d.dialogContains("Edit entry")
	d.dialogContains("Writing project plan")
	d.key("esc")

	// Deleting the paused entry clears it; e/d on an idle Timer panel is
	// then a no-op with a hint.
	d.keys("d", "y")
	if d.m.dialog != nil {
		t.Fatal("dialog still open after delete")
	}
	d.key("e")
	if d.m.dialog != nil {
		t.Fatal("edit dialog opened with no running or paused entry")
	}
	if d.m.status != "no timer running" {
		t.Fatalf("status = %q, want 'no timer running'", d.m.status)
	}
}

func TestEditSavesDescription(t *testing.T) {
	d := newDriver(t)
	d.key("e") // cursor starts on the running entry
	d.dialogContains("Edit entry")
	d.keys("ctrl+u", "Renamed", "esc", "y")

	if d.m.dialog != nil {
		t.Fatal("dialog still open after save")
	}
	e, _ := d.entryByID(1)
	if e.Description != "Renamed" || !e.Running() {
		t.Fatalf("entry = %+v, want running with description Renamed", e)
	}
}

func TestDeleteCancelAndConfirm(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d")
	d.dialogContains("Delete entry?")
	d.dialogContains("Yesterday task")

	d.key("n")
	if _, ok := d.entryByID(3); !ok {
		t.Fatal("entry deleted despite cancel")
	}

	d.keys("d", "y")
	if _, ok := d.entryByID(3); ok {
		t.Fatal("entry still present after confirmed delete")
	}
	if len(d.m.entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(d.m.entries))
	}
}

func TestDialogSwallowsGlobalKeys(t *testing.T) {
	d := newDriver(t)
	d.key("c")                // open start dialog
	d.keys("qcder", " ", "x") // global keys and space — all just text here

	d.dialogContains("qcder x")
	if len(d.m.entries) != 3 {
		t.Fatalf("entries = %d, want 3 (global keys leaked through the dialog)", len(d.m.entries))
	}
	d.key("esc")
	if d.m.dialog != nil {
		t.Fatal("dialog still open after esc")
	}
	if run := d.m.runningEntry(); run == nil || run.ID != 1 {
		t.Fatal("running entry changed while a dialog was open")
	}
}

func TestRefreshReloadsFromService(t *testing.T) {
	d := newDriver(t)
	if len(d.m.entries) != 3 {
		t.Fatalf("fixtures = %d entries, want 3", len(d.m.entries))
	}
	d.key("r") // reload pulls the mock's own seeded state (8 entries)
	if len(d.m.entries) != 8 {
		t.Fatalf("entries after refresh = %d, want 8 from the service", len(d.m.entries))
	}
	if d.m.loading {
		t.Fatal("still marked loading after refresh completed")
	}

	// Budget: a refresh costs exactly one request. Projects are fetched once
	// per session (at startup), and the session's map is reused — every
	// request counts against Toggl's 30/hour Free-plan quota.
	if got := d.mock.Calls("RecentEntries"); got != 1 {
		t.Fatalf("RecentEntries called %d times, want 1", got)
	}
	if got := d.mock.Calls("Projects"); got != 0 {
		t.Fatalf("Projects called %d times on refresh, want 0 — refresh must reuse the session's projects", got)
	}
	if len(d.m.projects) != 2 {
		t.Fatalf("projects map lost on refresh: %d entries, want the 2 fixtures kept", len(d.m.projects))
	}
}

// The API-error notice: a floating popup that can't be missed or silently
// overwritten the way the one-line status can.

func TestAPIErrorShowsNoticePopup(t *testing.T) {
	d := newDriver(t)
	d.feed(errMsg{errors.New("boom failure")})

	view := d.m.View()
	for _, want := range []string{"API error", "boom failure", "esc to dismiss"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q after an API error:\n%s", want, view)
		}
	}

	d.key("esc")
	if strings.Contains(d.m.View(), "API error") {
		t.Fatal("notice still visible after esc")
	}
}

func TestRateLimitNoticeExplainsQuota(t *testing.T) {
	d := newDriver(t)
	d.feed(errMsg{toggl.ErrRateLimited})
	if !strings.Contains(d.m.View(), "30 requests/hour") {
		t.Fatalf("rate-limit notice does not explain the quota:\n%s", d.m.View())
	}
}

func TestNoticeClearsOnNextSuccess(t *testing.T) {
	d := newDriver(t)
	d.feed(errMsg{errors.New("transient blip")})
	d.key(" ") // successful pause round-trip through the mock
	if strings.Contains(d.m.View(), "API error") {
		t.Fatal("notice still visible after a successful API call")
	}
}

func TestNoticeAutoExpires(t *testing.T) {
	d := newDriver(t)
	d.feed(errMsg{errors.New("stale error")})

	mm, _ := d.m.Update(tickMsg(fixedNow.Add(noticeTTL / 2)))
	d.m = mm.(Model)
	if !strings.Contains(d.m.View(), "API error") {
		t.Fatal("notice expired too early")
	}

	mm, _ = d.m.Update(tickMsg(fixedNow.Add(noticeTTL + time.Second)))
	d.m = mm.(Model)
	if strings.Contains(d.m.View(), "API error") {
		t.Fatal("notice still visible past its TTL")
	}
}

func TestErrorLandsInStatusBar(t *testing.T) {
	d := newDriver(t)
	d.feed(errMsg{errors.New("boom")})
	if d.m.status != "boom" || !d.m.statusErr {
		t.Fatalf("status = %q (err=%v), want boom as error", d.m.status, d.m.statusErr)
	}
}
