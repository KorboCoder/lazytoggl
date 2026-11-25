package ui

// Deleting with undo (openspec change undo-delete, spec entry-deleting).
// Fixtures: ID1 running "Writing project plan" since 11:50, ID2 "Standup"
// today, ID3 "Yesterday task" (rows: ID1, ID2, then ID3 under Yesterday).

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
