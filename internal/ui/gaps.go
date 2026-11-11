package ui

import (
	"time"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

// gapFallback is how long o/O open on when there is no neighbour that day,
// the same default as m.
const gapFallback = 30 * time.Minute

// sameDay reports whether e starts on the local day of t.
func sameDay(e toggl.TimeEntry, t time.Time) bool {
	a, b := e.Start.Local(), t.Local()
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// end is when e ends; a running entry runs until now.
func end(e toggl.TimeEntry, now time.Time) time.Time {
	if e.Running() {
		return now
	}
	return e.Stop
}

// gapBelow is the gap o fills: from the latest end among sel's day's entries
// that started before it (30 minutes before sel when there are none) up to
// sel's start, with exact times. ok is false when it's under a minute, which
// covers entries that touch or overlap.
func gapBelow(entries []toggl.TimeEntry, sel toggl.TimeEntry, now time.Time) (start, stop time.Time, ok bool) {
	stop = sel.Start
	start = stop.Add(-gapFallback)
	found := false
	for _, e := range entries {
		if e.ID == sel.ID || !sameDay(e, sel.Start) || !e.Start.Before(sel.Start) {
			continue
		}
		if t := end(e, now); !found || t.After(start) {
			start, found = t, true
		}
	}
	return start, stop, stop.Sub(start) >= time.Minute
}

// gapAbove is the gap O fills: from sel's stop up to the earliest start
// among sel's day's entries that start after it, the running entry included.
// With none, it stops at now (to the minute) when sel's day is today, else
// 30 minutes after sel. A running sel has nothing after it (ok is false), as
// does a gap under a minute.
func gapAbove(entries []toggl.TimeEntry, sel toggl.TimeEntry, now time.Time) (start, stop time.Time, ok bool) {
	if sel.Running() {
		return time.Time{}, time.Time{}, false
	}
	start = sel.Stop
	found := false
	for _, e := range entries {
		if e.ID == sel.ID || !sameDay(e, sel.Start) || !e.Start.After(sel.Start) {
			continue
		}
		if !found || e.Start.Before(stop) {
			stop, found = e.Start, true
		}
	}
	if !found {
		if sameDay(sel, now) {
			n := now.Local()
			stop = time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), 0, 0, time.Local)
		} else {
			stop = start.Add(gapFallback)
		}
	}
	return start, stop, stop.Sub(start) >= time.Minute
}
