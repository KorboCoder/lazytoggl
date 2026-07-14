package toggl

import "time"

// TimeEntry mirrors a Toggl Track time entry. A running entry has a zero
// Stop time and a negative Duration, matching the API convention.
type TimeEntry struct {
	ID          int64
	WorkspaceID int64
	ProjectID   int64 // 0 means no project
	Description string
	Start       time.Time
	Stop        time.Time
	Duration    int64 // seconds; -1 while running
}

func (e TimeEntry) Running() bool { return e.Stop.IsZero() }

// Elapsed is the entry's tracked time as of now. Running entries are computed
// locally from Start so the UI can tick without any API traffic.
func (e TimeEntry) Elapsed(now time.Time) time.Duration {
	if e.Running() {
		d := now.Sub(e.Start)
		if d < 0 {
			d = 0
		}
		return d
	}
	return time.Duration(e.Duration) * time.Second
}

type Project struct {
	ID    int64
	Name  string
	Color string // hex, e.g. "#e36a00"
}

type Workspace struct {
	ID   int64
	Name string
}

type Me struct {
	ID                 int64
	Fullname           string
	DefaultWorkspaceID int64
}
