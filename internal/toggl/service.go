package toggl

import "context"

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
	// Start begins a new entry. Toggl stops any running entry automatically.
	Start(ctx context.Context, req StartRequest) (TimeEntry, error)
	Stop(ctx context.Context, entryID int64) (TimeEntry, error)
	Update(ctx context.Context, entry TimeEntry) (TimeEntry, error)
	Delete(ctx context.Context, entryID int64) error
}
