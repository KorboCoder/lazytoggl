package toggl

import (
	"context"
	"testing"
	"time"
)

// These tests pin the service semantics the UI relies on. When the real HTTP
// client lands in Phase 2, its httptest-backed tests must assert the same
// behaviors (start stops the running entry, durations are recomputed, the
// list is newest-first).

func TestStartStopsRunningEntry(t *testing.T) {
	m := NewMock()
	ctx := context.Background()

	started, err := m.Start(ctx, StartRequest{Description: "new work", ProjectID: 101})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !started.Running() {
		t.Fatal("started entry should be running")
	}

	entries, err := m.RecentEntries(ctx)
	if err != nil {
		t.Fatalf("RecentEntries: %v", err)
	}
	var running []TimeEntry
	for _, e := range entries {
		if e.Running() {
			running = append(running, e)
		}
	}
	if len(running) != 1 || running[0].ID != started.ID {
		t.Fatalf("want exactly the new entry running, got %d running", len(running))
	}

	// The previously running seeded entry (ID 1) must be closed out with a
	// duration matching its elapsed time (~12 minutes).
	for _, e := range entries {
		if e.ID == 1 {
			if e.Running() {
				t.Fatal("old running entry was not stopped")
			}
			if e.Duration < 11*60 || e.Duration > 13*60 {
				t.Fatalf("old entry duration = %ds, want ~12m", e.Duration)
			}
		}
	}
}

func TestStopComputesDuration(t *testing.T) {
	m := NewMock()
	stopped, err := m.Stop(context.Background(), 1)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stopped.Running() {
		t.Fatal("entry still running after Stop")
	}
	if stopped.Duration < 11*60 || stopped.Duration > 13*60 {
		t.Fatalf("duration = %ds, want ~12m", stopped.Duration)
	}
	// Stopping an already-stopped entry is a no-op, not an error.
	again, err := m.Stop(context.Background(), 1)
	if err != nil || again.Duration != stopped.Duration {
		t.Fatalf("second Stop changed the entry: %v, %d", err, again.Duration)
	}
}

func TestUpdateRecomputesDuration(t *testing.T) {
	m := NewMock()
	ctx := context.Background()
	entries, _ := m.RecentEntries(ctx)
	var standup TimeEntry
	for _, e := range entries {
		if e.ID == 2 {
			standup = e
		}
	}
	standup.Stop = standup.Start.Add(30 * time.Minute)

	updated, err := m.Update(ctx, standup)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Duration != 30*60 {
		t.Fatalf("duration = %ds, want 1800", updated.Duration)
	}
}

func TestDeleteRemovesEntry(t *testing.T) {
	m := NewMock()
	ctx := context.Background()
	before, _ := m.RecentEntries(ctx)

	if err := m.Delete(ctx, 2); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	after, _ := m.RecentEntries(ctx)
	if len(after) != len(before)-1 {
		t.Fatalf("entries after delete = %d, want %d", len(after), len(before)-1)
	}
	for _, e := range after {
		if e.ID == 2 {
			t.Fatal("deleted entry still present")
		}
	}
	if err := m.Delete(ctx, 2); err == nil {
		t.Fatal("deleting a missing entry should error")
	}
}

func TestRecentEntriesNewestFirst(t *testing.T) {
	m := NewMock()
	entries, err := m.RecentEntries(context.Background())
	if err != nil {
		t.Fatalf("RecentEntries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no seeded entries")
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Start.After(entries[i-1].Start) {
			t.Fatalf("entries not sorted newest-first at index %d", i)
		}
	}
}

// The mock returns the same window the real client asks for, so a moved
// entry drops off on refresh under -mock just as it does live.
func TestMockRecentEntriesFiltersToWindow(t *testing.T) {
	m := NewMock()
	now := time.Now()
	old := now.Add(-RecentEntriesLookback - time.Hour)
	future := now.Add(2 * time.Hour)
	m.entries = append(m.entries,
		TimeEntry{ID: 900, Description: "too old", Start: old, Stop: old.Add(time.Hour), Duration: 3600},
		TimeEntry{ID: 901, Description: "future", Start: future, Stop: future.Add(time.Hour), Duration: 3600},
		TimeEntry{ID: 902, Description: "edge of window", Start: now.Add(-RecentEntriesLookback + time.Hour),
			Stop: now.Add(-RecentEntriesLookback + 2*time.Hour), Duration: 3600},
	)
	entries, err := m.RecentEntries(context.Background())
	if err != nil {
		t.Fatalf("RecentEntries: %v", err)
	}
	got := map[int64]bool{}
	for _, e := range entries {
		got[e.ID] = true
	}
	if got[900] || got[901] {
		t.Fatalf("entries outside the window returned: %v", got)
	}
	if !got[902] {
		t.Fatal("entry inside the window missing")
	}
	if !got[1] {
		t.Fatal("running entry missing")
	}
}

func TestInRecentWindowBounds(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	since := RecentEntriesSince(now)
	for _, c := range []struct {
		start time.Time
		want  bool
	}{
		{since, true},
		{now, true},
		{now.Add(-time.Hour), true},
		{since.Add(-time.Second), false},
		{now.Add(time.Second), false},
	} {
		if got := InRecentWindow(c.start, now); got != c.want {
			t.Errorf("InRecentWindow(%s) = %v, want %v", c.start, got, c.want)
		}
	}
}

// The seed must sit inside the window at any time of day: a "today" entry
// seeded at a fixed clock time starts in the future in the early morning,
// and the window filter (like the real API) leaves it out.
func TestMockSeedIsInWindowAtAnyTimeOfDay(t *testing.T) {
	m := NewMock()
	now := time.Now()
	for _, e := range m.entries {
		if !e.Running() && !InRecentWindow(e.Start, now) {
			t.Errorf("seed entry %d %q starts %s, outside the window at %s", e.ID, e.Description, e.Start, now)
		}
		if !e.Running() && e.Stop.After(now) {
			t.Errorf("seed entry %d %q stops %s, after now", e.ID, e.Description, e.Stop)
		}
	}
}

// The mock seeds an archived project so -mock shows the picker hiding it.
func TestMockSeedsAnArchivedProject(t *testing.T) {
	projects, err := NewMock().Projects(context.Background())
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	var active, archived int
	for _, p := range projects {
		if p.Active {
			active++
		} else {
			archived++
		}
	}
	if active != 2 || archived != 1 {
		t.Fatalf("active %d archived %d, want 2 and 1", active, archived)
	}
}

func TestMockCreateProject(t *testing.T) {
	m := NewMock()
	ctx := context.Background()
	p, err := m.CreateProject(ctx, "newclient")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID == 0 || p.Name != "newclient" || !p.Active {
		t.Fatalf("project = %+v, want an active newclient with an ID", p)
	}
	projects, _ := m.Projects(ctx)
	found := false
	for _, q := range projects {
		found = found || q.ID == p.ID
	}
	if !found {
		t.Fatal("created project missing from Projects")
	}
	// Like the API, a duplicate name is refused.
	if _, err := m.CreateProject(ctx, "newclient"); err == nil {
		t.Fatal("duplicate name accepted")
	}
}
