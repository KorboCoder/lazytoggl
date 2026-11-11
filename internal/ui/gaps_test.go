package ui

import (
	"testing"
	"time"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

// at is a wall-clock time on Tue 14 Jul 2026 (day 14) or another July day.
func at(day, hour, min, sec int) time.Time {
	return time.Date(2026, 7, day, hour, min, sec, 0, time.Local)
}

func fin(id int64, start, stop time.Time) toggl.TimeEntry {
	return toggl.TimeEntry{ID: id, Start: start, Stop: stop, Duration: int64(stop.Sub(start).Seconds())}
}

func running(id int64, start time.Time) toggl.TimeEntry {
	return toggl.TimeEntry{ID: id, Start: start, Duration: -1}
}

type gapCase struct {
	name        string
	entries     []toggl.TimeEntry
	sel         int64
	start, stop time.Time
	ok          bool
}

func runGapCases(t *testing.T, f func([]toggl.TimeEntry, toggl.TimeEntry, time.Time) (time.Time, time.Time, bool), cases []gapCase) {
	t.Helper()
	now := at(14, 12, 40, 27)
	for _, c := range cases {
		var sel toggl.TimeEntry
		for _, e := range c.entries {
			if e.ID == c.sel {
				sel = e
			}
		}
		start, stop, ok := f(c.entries, sel, now)
		if ok != c.ok {
			t.Errorf("%s: ok = %v, want %v (got %s-%s)", c.name, ok, c.ok, start.Format("02 15:04:05"), stop.Format("02 15:04:05"))
			continue
		}
		if ok && (!start.Equal(c.start) || !stop.Equal(c.stop)) {
			t.Errorf("%s: gap = %s-%s, want %s-%s", c.name,
				start.Format("02 15:04:05"), stop.Format("02 15:04:05"),
				c.start.Format("02 15:04:05"), c.stop.Format("02 15:04:05"))
		}
	}
}

func TestGapBelow(t *testing.T) {
	standup := fin(1, at(14, 8, 30, 0), at(14, 9, 0, 0))
	code := fin(2, at(14, 10, 0, 0), at(14, 10, 30, 0))
	runGapCases(t, gapBelow, []gapCase{
		{name: "between two entries", entries: []toggl.TimeEntry{code, standup}, sel: 2,
			start: at(14, 9, 0, 0), stop: at(14, 10, 0, 0), ok: true},
		{name: "first of the day", entries: []toggl.TimeEntry{code, standup}, sel: 1,
			start: at(14, 8, 0, 0), stop: at(14, 8, 30, 0), ok: true},
		{name: "exact seconds kept",
			entries: []toggl.TimeEntry{fin(2, at(14, 10, 0, 5), at(14, 10, 30, 0)), fin(1, at(14, 8, 30, 0), at(14, 9, 0, 17))}, sel: 2,
			start: at(14, 9, 0, 17), stop: at(14, 10, 0, 5), ok: true},
		{name: "latest end wins over the row below",
			entries: []toggl.TimeEntry{code, fin(3, at(14, 9, 15, 0), at(14, 9, 30, 0)), fin(4, at(14, 8, 0, 0), at(14, 9, 45, 0))}, sel: 2,
			start: at(14, 9, 45, 0), stop: at(14, 10, 0, 0), ok: true},
		{name: "overlap is no gap",
			entries: []toggl.TimeEntry{code, fin(3, at(14, 9, 30, 0), at(14, 10, 15, 0))}, sel: 2, ok: false},
		{name: "touching is no gap",
			entries: []toggl.TimeEntry{code, fin(3, at(14, 9, 0, 0), at(14, 10, 0, 0))}, sel: 2, ok: false},
		{name: "under a minute is no gap",
			entries: []toggl.TimeEntry{code, fin(3, at(14, 9, 0, 0), at(14, 9, 59, 30))}, sel: 2, ok: false},
		{name: "running selection fills before it",
			entries: []toggl.TimeEntry{running(5, at(14, 11, 50, 0)), fin(3, at(14, 10, 0, 0), at(14, 10, 15, 0))}, sel: 5,
			start: at(14, 10, 15, 0), stop: at(14, 11, 50, 0), ok: true},
		{name: "other days ignored",
			entries: []toggl.TimeEntry{standup, fin(6, at(13, 16, 0, 0), at(13, 18, 0, 0))}, sel: 1,
			start: at(14, 8, 0, 0), stop: at(14, 8, 30, 0), ok: true},
	})
}

func TestGapAbove(t *testing.T) {
	code := fin(2, at(14, 10, 0, 0), at(14, 10, 30, 0))
	review := fin(3, at(14, 11, 0, 0), at(14, 12, 0, 0))
	runGapCases(t, gapAbove, []gapCase{
		{name: "between two entries", entries: []toggl.TimeEntry{review, code}, sel: 2,
			start: at(14, 10, 30, 0), stop: at(14, 11, 0, 0), ok: true},
		{name: "up to the running timer", entries: []toggl.TimeEntry{running(9, at(14, 12, 20, 0)), review}, sel: 3,
			start: at(14, 12, 0, 0), stop: at(14, 12, 20, 0), ok: true},
		{name: "newest today stops at now, to the minute", entries: []toggl.TimeEntry{review, code}, sel: 3,
			start: at(14, 12, 0, 0), stop: at(14, 12, 40, 0), ok: true},
		{name: "newest of a past day: 30 minutes after",
			entries: []toggl.TimeEntry{fin(4, at(13, 16, 0, 0), at(13, 18, 0, 0)), review}, sel: 4,
			start: at(13, 18, 0, 0), stop: at(13, 18, 30, 0), ok: true},
		{name: "earliest later start wins",
			entries: []toggl.TimeEntry{review, fin(5, at(14, 10, 45, 0), at(14, 10, 50, 0)), code}, sel: 2,
			start: at(14, 10, 30, 0), stop: at(14, 10, 45, 0), ok: true},
		{name: "overlap is no gap",
			entries: []toggl.TimeEntry{fin(5, at(14, 10, 15, 0), at(14, 11, 0, 0)), code}, sel: 2, ok: false},
		{name: "stop later than now is no gap",
			entries: []toggl.TimeEntry{fin(6, at(14, 12, 0, 0), at(14, 13, 0, 0))}, sel: 6, ok: false},
		{name: "running selection has nothing after it",
			entries: []toggl.TimeEntry{running(9, at(14, 12, 20, 0)), review}, sel: 9, ok: false},
	})
}
