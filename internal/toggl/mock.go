package toggl

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// mockLatency makes the mock feel like a network so the UI's async paths and
// in-flight statuses actually get exercised during the prototype phase.
const mockLatency = 150 * time.Millisecond

// Mock is an in-memory Service seeded with realistic data, including one
// running entry. It mimics API behavior the UI depends on: starting a new
// entry stops the running one.
type Mock struct {
	mu       sync.Mutex
	entries  []TimeEntry
	projects []Project
	nextID   int64
	calls    map[string]int
}

// Calls reports how many times the named method ("RecentEntries",
// "Projects", …) has been invoked — lets tests assert the app's request
// budget, which matters under Toggl's 30-requests/hour Free-plan limit.
func (m *Mock) Calls(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[method]
}

func (m *Mock) countCall(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.calls == nil {
		m.calls = map[string]int{}
	}
	m.calls[method]++
}

func NewMock() *Mock {
	now := time.Now()
	at := func(daysAgo, hour, min int) time.Time {
		d := now.AddDate(0, 0, -daysAgo)
		return time.Date(d.Year(), d.Month(), d.Day(), hour, min, 0, 0, time.Local)
	}
	fin := func(id, projectID int64, desc string, start time.Time, dur time.Duration) TimeEntry {
		return TimeEntry{
			ID: id, WorkspaceID: 1, ProjectID: projectID, Description: desc,
			Start: start, Stop: start.Add(dur), Duration: int64(dur.Seconds()),
		}
	}
	return &Mock{
		projects: []Project{
			{ID: 101, Name: "ACME", Color: "#e36a00"},
			{ID: 102, Name: "Internal", Color: "#2da608"},
		},
		entries: []TimeEntry{
			{ID: 1, WorkspaceID: 1, ProjectID: 101, Description: "Writing project plan",
				Start: now.Add(-12 * time.Minute), Duration: -1},
			// Today's entries sit just before the running one rather than at
			// fixed clock times, which would be in the future in the early
			// morning and so outside the RecentEntries window.
			fin(2, 0, "Standup", now.Add(-150*time.Minute), 15*time.Minute),
			fin(3, 101, "Code review", now.Add(-100*time.Minute), 62*time.Minute),
			fin(4, 101, "API integration spike", at(1, 9, 15), 96*time.Minute),
			fin(5, 102, "1:1 with manager", at(1, 13, 0), 30*time.Minute),
			fin(6, 101, "Fix flaky tests", at(1, 14, 0), 110*time.Minute),
			fin(7, 102, "Expense reports", at(2, 11, 0), 25*time.Minute),
			fin(8, 101, "Sprint planning", at(2, 15, 0), 55*time.Minute),
		},
		nextID: 1000,
	}
}

func (m *Mock) Me(ctx context.Context) (Me, error) {
	m.countCall("Me")
	time.Sleep(mockLatency)
	return Me{ID: 42, Fullname: "Mock User", DefaultWorkspaceID: 1}, nil
}

func (m *Mock) RecentEntries(ctx context.Context) ([]TimeEntry, error) {
	m.countCall("RecentEntries")
	time.Sleep(mockLatency)
	m.mu.Lock()
	defer m.mu.Unlock()
	// Mirror the real client's start_date/end_date window, keeping the
	// running entry whatever its start.
	now := time.Now()
	var out []TimeEntry
	for _, e := range m.entries {
		if e.Running() || InRecentWindow(e.Start, now) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.After(out[j].Start) })
	return out, nil
}

func (m *Mock) Projects(ctx context.Context) ([]Project, error) {
	m.countCall("Projects")
	time.Sleep(mockLatency)
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Project(nil), m.projects...), nil
}

func (m *Mock) Start(ctx context.Context, req StartRequest) (TimeEntry, error) {
	m.countCall("Start")
	time.Sleep(mockLatency)
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.stopRunningLocked(now)
	m.nextID++
	e := TimeEntry{
		ID: m.nextID, WorkspaceID: 1, ProjectID: req.ProjectID,
		Description: req.Description, Start: now, Duration: -1,
	}
	m.entries = append(m.entries, e)
	return e, nil
}

func (m *Mock) Stop(ctx context.Context, entryID int64) (TimeEntry, error) {
	m.countCall("Stop")
	time.Sleep(mockLatency)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.entries {
		if m.entries[i].ID == entryID {
			if m.entries[i].Running() {
				now := time.Now()
				m.entries[i].Stop = now
				m.entries[i].Duration = int64(now.Sub(m.entries[i].Start).Seconds())
			}
			return m.entries[i], nil
		}
	}
	return TimeEntry{}, errors.New("time entry not found")
}

func (m *Mock) Update(ctx context.Context, entry TimeEntry) (TimeEntry, error) {
	m.countCall("Update")
	time.Sleep(mockLatency)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.entries {
		if m.entries[i].ID == entry.ID {
			if !entry.Running() {
				entry.Duration = int64(entry.Stop.Sub(entry.Start).Seconds())
			} else {
				entry.Duration = -1
			}
			m.entries[i] = entry
			return entry, nil
		}
	}
	return TimeEntry{}, errors.New("time entry not found")
}

func (m *Mock) Delete(ctx context.Context, entryID int64) error {
	m.countCall("Delete")
	time.Sleep(mockLatency)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.entries {
		if m.entries[i].ID == entryID {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return errors.New("time entry not found")
}

func (m *Mock) stopRunningLocked(at time.Time) {
	for i := range m.entries {
		if m.entries[i].Running() {
			m.entries[i].Stop = at
			m.entries[i].Duration = int64(at.Sub(m.entries[i].Start).Seconds())
		}
	}
}
