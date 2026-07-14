package panels

import (
	"testing"
	"time"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

func fin(id int64, start time.Time, dur time.Duration) toggl.TimeEntry {
	return toggl.TimeEntry{ID: id, Start: start, Stop: start.Add(dur), Duration: int64(dur.Seconds())}
}

// testRows: ["Today" header, running, finished, "Yesterday" header, finished,
// "Mon, Jul 9"-style header, finished]
func testRows(now time.Time) []Row {
	entries := []toggl.TimeEntry{
		{ID: 1, Start: now.Add(-10 * time.Minute), Duration: -1},
		fin(2, now.Add(-2*time.Hour), 30*time.Minute),
		fin(3, now.AddDate(0, 0, -1), time.Hour),
		fin(4, now.AddDate(0, 0, -5), 45*time.Minute),
	}
	return BuildRows(entries, now)
}

func TestBuildRows(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)
	rows := testRows(now)

	wantKinds := []RowKind{RowHeader, RowEntry, RowEntry, RowHeader, RowEntry, RowHeader, RowEntry}
	if len(rows) != len(wantKinds) {
		t.Fatalf("rows = %d, want %d", len(rows), len(wantKinds))
	}
	for i, k := range wantKinds {
		if rows[i].Kind != k {
			t.Fatalf("rows[%d].Kind = %d, want %d", i, rows[i].Kind, k)
		}
	}
	if rows[0].Label != "Today" {
		t.Fatalf("rows[0].Label = %q, want Today", rows[0].Label)
	}
	// Today's total = 10m running (elapsed) + 30m finished.
	if rows[0].Total != 40*time.Minute {
		t.Fatalf("today total = %v, want 40m", rows[0].Total)
	}
	if rows[3].Label != "Yesterday" {
		t.Fatalf("rows[3].Label = %q, want Yesterday", rows[3].Label)
	}
	wantOld := now.AddDate(0, 0, -5).Format("Mon, Jan 2")
	if rows[5].Label != wantOld {
		t.Fatalf("rows[5].Label = %q, want %q", rows[5].Label, wantOld)
	}
}

func TestMoveSkipsHeaders(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)
	rows := testRows(now)
	var p Entries

	steps := []struct {
		delta int
		want  int
	}{
		{1, 1},  // off the "Today" header onto the running entry
		{1, 2},  //
		{1, 4},  // over the "Yesterday" header
		{1, 6},  // over the date header
		{1, 6},  // bottom: stays
		{-1, 4}, // back up over the header
		{-1, 2},
		{-1, 1},
		{-1, 1}, // top: stays (row 0 is a header)
	}
	for i, s := range steps {
		p.Move(rows, s.delta)
		if p.Cursor != s.want {
			t.Fatalf("step %d: cursor = %d, want %d", i, p.Cursor, s.want)
		}
	}
}

func TestClampSnapsToSelectable(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)
	rows := testRows(now)
	var p Entries

	p.Cursor = 99
	p.Clamp(rows)
	if p.Cursor != 6 {
		t.Fatalf("clamp beyond end: cursor = %d, want 6", p.Cursor)
	}

	p.Cursor = 3 // "Yesterday" header
	p.Clamp(rows)
	if rows[p.Cursor].Kind == RowHeader {
		t.Fatalf("clamp left cursor on a header (index %d)", p.Cursor)
	}

	p.Cursor = 3
	p.Clamp(BuildRows(nil, now)) // list shrank to nothing
	if p.Cursor != 0 {
		t.Fatalf("clamp on shrink: cursor = %d, want 0", p.Cursor)
	}
}
