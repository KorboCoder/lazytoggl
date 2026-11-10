package toggl

import (
	"context"
	"time"
)

// RecentEntriesLookback bounds the RecentEntries window so a fresh account
// with years of history doesn't pull it all down; it covers "today" plus
// enough trailing days for the entries list. Every Service implementation
// returns entries starting within [RecentEntriesSince(now), now].
const RecentEntriesLookback = 8 * 24 * time.Hour

// RecentEntriesSince is the start of the RecentEntries window at now. An
// entry that starts earlier drops out of the list on the next refresh.
func RecentEntriesSince(now time.Time) time.Time { return now.Add(-RecentEntriesLookback) }

// InRecentWindow reports whether an entry starting at start is in the
// RecentEntries window at now. One that isn't (too old, or starting after
// now: the API's end_date) drops out of the list on the next refresh.
func InRecentWindow(start, now time.Time) bool {
	return !start.Before(RecentEntriesSince(now)) && !start.After(now)
}

// LogRequest describes a finished time entry to create after the fact.
type LogRequest struct {
	Description string
	ProjectID   int64 // 0 means no project
	Start, Stop time.Time
}

// StartRequest describes a new time entry to start.
type StartRequest struct {
	Description string
	ProjectID   int64
}

// Service is the seam between the UI and Toggl data. The Phase 1 prototype
// runs on the in-memory Mock; Phase 2 swaps in the real HTTP client behind
// the same interface.
type Service interface {
	Me(ctx context.Context) (Me, error)
	// RecentEntries returns entries newest-first; the running entry, if any,
	// is included (zero Stop, negative Duration).
	RecentEntries(ctx context.Context) ([]TimeEntry, error)
	Projects(ctx context.Context) ([]Project, error)
	// CreateProject creates an active project in the workspace. A name that
	// is already taken is refused.
	CreateProject(ctx context.Context, name string) (Project, error)
	// Start begins a new entry. Toggl stops any running entry automatically.
	Start(ctx context.Context, req StartRequest) (TimeEntry, error)
	Stop(ctx context.Context, entryID int64) (TimeEntry, error)
	// Log creates a finished entry. Unlike Start, it leaves a running entry
	// running (confirmed live).
	Log(ctx context.Context, req LogRequest) (TimeEntry, error)
	Update(ctx context.Context, entry TimeEntry) (TimeEntry, error)
	Delete(ctx context.Context, entryID int64) error
}
