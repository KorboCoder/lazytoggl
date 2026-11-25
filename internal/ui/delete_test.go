package ui

// Deleting with undo (openspec change undo-delete, spec entry-deleting).
// Fixtures: ID1 running "Writing project plan" since 11:50, ID2 "Standup"
// today, ID3 "Yesterday task" (rows: ID1, ID2, then ID3 under Yesterday).

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/panels"
)

// view is the whole screen, status line included.
func view(d *driver) string { return d.m.View() }

func wantView(t *testing.T, d *driver, want string) {
	t.Helper()
	if v := view(d); !strings.Contains(v, want) {
		t.Fatalf("screen lacks %q:\n%s", want, v)
	}
}

func wantNoView(t *testing.T, d *driver, unwanted string) {
	t.Helper()
	if v := view(d); strings.Contains(v, unwanted) {
		t.Fatalf("screen unexpectedly has %q:\n%s", unwanted, v)
	}
}

// tickAt advances the app clock to fixedNow + secs and runs the commands
// the tick returns, except timers.
func tickAt(d *driver, secs int) {
	mm, cmd := d.m.Update(tickMsg(fixedNow.Add(time.Duration(secs) * time.Second)))
	d.m = mm.(Model)
	runSkippingTicks(d, cmd)
}

// tickPtr is the code pointer every tea.Tick closure shares.
var tickPtr = reflect.ValueOf(tea.Tick(time.Hour, func(time.Time) tea.Msg { return nil })).Pointer()

// runSkippingTicks runs cmd (and a batch's parts) without ever calling a
// tea.Tick, which would start a real timer that outlives the test.
func runSkippingTicks(d *driver, cmd tea.Cmd) {
	if cmd == nil || reflect.ValueOf(cmd).Pointer() == tickPtr {
		return
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			runSkippingTicks(d, c)
		}
	case nil:
	default:
		d.feed(msg)
	}
}

func TestDeleteHidesAtOnce(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d") // ID3
	if d.m.dialog != nil {
		t.Fatal("d opened a dialog; it should delete at once")
	}
	if _, ok := d.entryByID(3); ok {
		t.Fatal("deleted entry still listed")
	}
	if n := d.mock.Calls("Delete"); n != 0 {
		t.Fatalf("Delete calls = %d, want 0 during the grace period", n)
	}
	wantView(t, d, `deleted "Yesterday task" · u undo (5s)`)
}

func TestDeleteRunningEntryClearsTimer(t *testing.T) {
	d := newDriver(t)
	d.keys("tab", "d") // the timer panel's running entry
	if d.m.runningEntry() != nil {
		t.Fatal("running entry still shown after d")
	}
	if _, ok := d.entryByID(2); !ok {
		t.Fatal("d deleted the list selection, not the timer's entry")
	}
}

func TestDeleteEmptyDescription(t *testing.T) {
	d := newDriver(t)
	for i := range d.m.entries {
		if d.m.entries[i].ID == 2 {
			d.m.entries[i].Description = ""
		}
	}
	d.keys("j", "d")
	wantView(t, d, `deleted "(no description)" · u undo`)
}

func TestDeleteThenUndo(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d")
	tickAt(d, 2)
	d.key("u")
	if _, ok := d.entryByID(3); !ok {
		t.Fatal("u did not bring the entry back")
	}
	wantSelected(t, d, 3)
	if n := d.mock.Calls("Delete"); n != 0 {
		t.Fatalf("Delete calls = %d, want 0 after an undo", n)
	}
	wantNoView(t, d, "u undo")
}

func TestUndoRestoresRunning(t *testing.T) {
	d := newDriver(t)
	d.keys("d", "u") // the cursor starts on the running ID1
	if run := d.m.runningEntry(); run == nil || run.ID != 1 {
		t.Fatalf("running = %+v, want ID1 running again", run)
	}
}

func TestUndoRestoresPaused(t *testing.T) {
	d := newDriver(t)
	d.keys("tab", " ") // pause the timer: ID1 stays pinned, paused
	if d.m.pausedEntry() == nil {
		t.Fatal("setup: no paused entry")
	}
	d.keys("d", "u")
	if p := d.m.pausedEntry(); p == nil || p.ID != 1 {
		t.Fatalf("paused = %+v, want ID1 paused again", p)
	}
}

func TestUndoWithNothingPending(t *testing.T) {
	d := newDriver(t)
	d.key("u")
	if len(d.m.entries) != 3 || d.mock.Calls("Delete") != 0 {
		t.Fatalf("u with nothing pending changed something: %d entries, %d deletes", len(d.m.entries), d.mock.Calls("Delete"))
	}
}

func TestDeleteCountsDown(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d")
	tickAt(d, 2)
	wantView(t, d, "u undo (3s)")
}

func TestDeleteSentAfterGrace(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d")
	tickAt(d, 4)
	if n := d.mock.Calls("Delete"); n != 0 {
		t.Fatalf("Delete calls at 4s = %d, want 0", n)
	}
	tickAt(d, 5)
	if n := d.mock.Calls("Delete"); n != 1 {
		t.Fatalf("Delete calls at 5s = %d, want 1", n)
	}
	wantNoView(t, d, "u undo")
	d.key("u")
	if _, ok := d.entryByID(3); ok {
		t.Fatal("u after the grace period brought the entry back")
	}
}

// selectEntry puts the list cursor on entry id.
func selectEntry(d *driver, id int64) {
	d.m.entriesPanel.SelectID(panels.BuildRows(d.m.entries, d.m.now), id)
}

func TestSecondDeleteCommitsFirst(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "d") // Standup pending
	selectEntry(d, 3)
	d.key("d") // Yesterday task: Standup is sent now
	if n := d.mock.Calls("Delete"); n != 1 {
		t.Fatalf("Delete calls = %d, want 1 (the first delete, sent at once)", n)
	}
	d.key("u")
	if _, ok := d.entryByID(3); !ok {
		t.Fatal("u did not bring back the second delete")
	}
	if _, ok := d.entryByID(2); ok {
		t.Fatal("the first delete came back")
	}
}

// failingDeleteService is the mock, except that deletes fail.
type failingDeleteService struct{ *toggl.Mock }

func (failingDeleteService) Delete(ctx context.Context, id int64) error {
	return toggl.ErrRateLimited
}

func TestDeleteFailureRestores(t *testing.T) {
	mock := toggl.NewMock()
	d := newDriverWith(t, failingDeleteService{mock}, mock)
	d.keys("j", "j", "d")
	tickAt(d, 5)
	if _, ok := d.entryByID(3); !ok {
		t.Fatal("a failed delete did not bring the entry back")
	}
	if !strings.Contains(d.m.notice, "Rate limited") {
		t.Fatalf("notice = %q, want the rate-limit error", d.m.notice)
	}
}

func TestRefreshKeepsPendingHidden(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d")
	d.feed(dataLoadedMsg{entries: fixtureEntries()}) // Toggl still has it
	if _, ok := d.entryByID(3); ok {
		t.Fatal("refresh brought back a pending delete")
	}
	d.key("u")
	if _, ok := d.entryByID(3); !ok {
		t.Fatal("u after a refresh did not bring the entry back")
	}
}

func TestRefreshKeepsInFlightDeleteHidden(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d")
	mm, _ := d.m.Update(tickMsg(fixedNow.Add(5 * time.Second))) // sent, not yet answered
	d.m = mm.(Model)
	d.feed(dataLoadedMsg{entries: fixtureEntries()})
	if _, ok := d.entryByID(3); ok {
		t.Fatal("refresh brought back an entry whose delete is in flight")
	}
	d.feed(entryDeletedMsg{id: 3})
	d.feed(dataLoadedMsg{entries: fixtureEntries()})
	if _, ok := d.entryByID(3); !ok {
		t.Fatal("after the delete was answered, refresh no longer shows what Toggl returns")
	}
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// q with a pending delete sends it, then quits once it's answered.
func TestQuitSendsPendingDelete(t *testing.T) {
	for _, quit := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
	} {
		d := newDriver(t)
		d.keys("j", "j", "d")
		mm, cmd := d.m.Update(quit)
		d.m = mm.(Model)
		if cmd == nil {
			t.Fatalf("%s did nothing", quit)
		}
		answer := cmd() // runs the delete, once
		if _, quitNow := answer.(tea.QuitMsg); quitNow {
			t.Fatalf("%s quit before sending the pending delete", quit)
		}
		mm, cmd = d.m.Update(answer)
		d.m = mm.(Model)
		if d.mock.Calls("Delete") != 1 || !isQuit(cmd) {
			t.Fatalf("%s: Delete calls %d, quit %v; want 1 then quit", quit, d.mock.Calls("Delete"), isQuit(cmd))
		}
	}
}

// ctrl+c with a dialog open also sends the pending delete first.
func TestCtrlCInDialogSendsPendingDelete(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d", "c") // a start dialog is open
	mm, cmd := d.m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	d.m = mm.(Model)
	if cmd == nil {
		t.Fatal("ctrl+c in a dialog did nothing")
	}
	if _, quitNow := cmd().(tea.QuitMsg); quitNow || d.mock.Calls("Delete") != 1 {
		t.Fatalf("ctrl+c in a dialog: quit at once %v, Delete calls %d; want the delete sent first", quitNow, d.mock.Calls("Delete"))
	}
}

// A failed delete on the way out still quits; the entry survives in Toggl.
func TestQuitAfterFailedDeleteStillQuits(t *testing.T) {
	mock := toggl.NewMock()
	d := newDriverWith(t, failingDeleteService{mock}, mock)
	d.keys("j", "j", "d")
	mm, cmd := d.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	d.m = mm.(Model)
	mm, cmd = d.m.Update(cmd())
	d.m = mm.(Model)
	if !isQuit(cmd) {
		t.Fatal("a failed delete on quit did not quit")
	}
}

// Without a pending delete, q quits at once; a second q while a delete is
// being sent quits without waiting.
func TestQuitWithoutPendingIsImmediate(t *testing.T) {
	d := newDriver(t)
	if _, cmd := d.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); !isQuit(cmd) {
		t.Fatal("q with nothing pending did not quit at once")
	}
	d.keys("j", "j", "d")
	mm, _ := d.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	d.m = mm.(Model)
	if _, cmd := d.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); !isQuit(cmd) {
		t.Fatal("a second q while the delete is sent did not quit")
	}
}

// Another action during the grace period doesn't hide the undo hint.
func TestUndoHintSurvivesOtherStatus(t *testing.T) {
	d := newDriver(t)
	d.keys("j", "j", "d", "c", "fresh", "enter") // start a timer within the grace period
	if d.m.status != "timer started" {
		t.Fatalf("status = %q, want the start's status", d.m.status)
	}
	wantView(t, d, "timer started")
	wantView(t, d, `deleted "Yesterday task" · u undo`)
}
