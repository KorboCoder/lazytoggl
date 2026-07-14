package panels

import (
	"testing"
	"time"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

func fin(id int64, start time.Time, dur time.Duration) toggl.TimeEntry {
	return toggl.TimeEntry{ID: id, Start: start, Stop: start.Add(dur), Duration: int64(dur.Seconds())}
}

// testRows: [+New, "Today" header, running, finished, "Yesterday" header,
// finished, "Mon, Jul 9"-style header, finished]
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

	wantKinds := []RowKind{RowNew, RowHeader, RowEntry, RowEntry, RowHeader, RowEntry, RowHeader, RowEntry}
	if len(rows) != len(wantKinds) {
		t.Fatalf("rows = %d, want %d", len(rows), len(wantKinds))
	}
	for i, k := range wantKinds {
		if rows[i].Kind != k {
			t.Fatalf("rows[%d].Kind = %d, want %d", i, rows[i].Kind, k)
		}
	}
	if rows[1].Label != "Today" {
		t.Fatalf("rows[1].Label = %q, want Today", rows[1].Label)
	}
	// Today's total = 10m running (elapsed) + 30m finished.
	if rows[1].Total != 40*time.Minute {
		t.Fatalf("today total = %v, want 40m", rows[1].Total)
	}
	if rows[4].Label != "Yesterday" {
		t.Fatalf("rows[4].Label = %q, want Yesterday", rows[4].Label)
	}
	wantOld := now.AddDate(0, 0, -5).Format("Mon, Jan 2")
	if rows[6].Label != wantOld {
		t.Fatalf("rows[6].Label = %q, want %q", rows[6].Label, wantOld)
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
		{1, 2},  // over "Today" header
		{1, 3},  //
		{1, 5},  // over "Yesterday" header
		{1, 7},  // over the date header
		{1, 7},  // bottom: stays
		{-1, 5}, // back up over the header
		{-1, 3},
		{-1, 2},
		{-1, 0}, // back to "+ New entry"
		{-1, 0}, // top: stays
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
	if p.Cursor != 7 {
		t.Fatalf("clamp beyond end: cursor = %d, want 7", p.Cursor)
	}

	p.Cursor = 4 // "Yesterday" header
	p.Clamp(rows)
	if rows[p.Cursor].Kind == RowHeader {
		t.Fatalf("clamp left cursor on a header (index %d)", p.Cursor)
	}

	p.Cursor = 3
	p.Clamp(BuildRows(nil, now)) // list shrank to just "+ New entry"
	if p.Cursor != 0 {
		t.Fatalf("clamp on shrink: cursor = %d, want 0", p.Cursor)
	}
}
