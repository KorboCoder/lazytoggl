package panels

import (
	"strings"
	"testing"
	"time"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

func TestTimerStates(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)
	running := &toggl.TimeEntry{ID: 1, Description: "Deep work", Start: now.Add(-10 * time.Minute), Duration: -1}
	paused := &toggl.TimeEntry{ID: 2, Description: "Old task", Start: now.Add(-30 * time.Minute),
		Stop: now.Add(-5 * time.Minute), Duration: 25 * 60}

	cases := []struct {
		name             string
		running, paused  *toggl.TimeEntry
		want, wantAbsent []string
	}{
		{"idle", nil, nil, []string{"no timer running"}, []string{"▶", "⏸"}},
		{"running", running, paused, []string{"▶", "Deep work", "00:10:00"}, []string{"⏸", "Old task"}},
		{"paused", nil, paused, []string{"⏸", "Old task", "00:25:00"}, []string{"▶", "Deep work"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			view := Timer(c.running, c.paused, nil, now, 60, false)
			for _, want := range c.want {
				if !strings.Contains(view, want) {
					t.Errorf("view missing %q:\n%s", want, view)
				}
			}
			for _, absent := range c.wantAbsent {
				if strings.Contains(view, absent) {
					t.Errorf("view unexpectedly contains %q:\n%s", absent, view)
				}
			}
		})
	}
}

// A running entry with a stale paused ID must win: the panel never shows
// both states at once.
func TestTimerRunningTakesPriorityOverPaused(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)
	running := &toggl.TimeEntry{ID: 1, Description: "Active", Start: now, Duration: -1}
	paused := &toggl.TimeEntry{ID: 2, Description: "Stale paused", Start: now, Stop: now, Duration: 60}

	view := Timer(running, paused, nil, now, 60, false)
	if strings.Contains(view, "⏸") || strings.Contains(view, "Stale paused") {
		t.Errorf("paused state leaked through while running:\n%s", view)
	}
}
