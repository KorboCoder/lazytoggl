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

// The gap rules: the neighbour that sets each end is chosen from sel's own
// day, but the result is then kept clear of every entry on any day (the
// fallbacks stop short of entries across midnight), and a gap that would
// still overlap an entry, such as one containing sel, is refused. ok is
// false when the gap is under a minute or refused.

// gapBelow is the gap o fills: from the latest end among sel's day's entries
// that started before it (30 minutes before sel when there are none) up to
// sel's start, with exact times.
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
	// Never start inside an earlier entry on another day (across midnight).
	for _, e := range entries {
		if e.ID != sel.ID && e.Start.Before(sel.Start) {
			if t := end(e, now); t.After(start) {
				start = t
			}
		}
	}
	return start, stop, fits(entries, sel, start, stop, now)
}

// gapAbove is the gap O fills: from sel's stop up to the earliest start
// among sel's day's entries that start after it, the running entry included.
// With none, it stops at now (to the minute) when sel's day is today, else
// 30 minutes after sel, but never after now. A running sel has nothing
// after it. (The app reports that case itself, before calling.)
func gapAbove(entries []toggl.TimeEntry, sel toggl.TimeEntry, now time.Time) (start, stop time.Time, ok bool) {
	if sel.Running() {
		return time.Time{}, time.Time{}, false
	}
	start = sel.Stop
	nowMin := now.Truncate(time.Minute) // absolute time: safe in a repeated DST hour
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
		stop = start.Add(gapFallback)
		if sameDay(sel, now) || stop.After(nowMin) {
			stop = nowMin
		}
	}
	// Never stop inside a later entry on another day (across midnight).
	for _, e := range entries {
		if e.ID != sel.ID && e.Start.After(sel.Start) && e.Start.Before(stop) {
			stop = e.Start
		}
	}
	return start, stop, fits(entries, sel, start, stop, now)
}

// fits reports whether start-stop is at least a minute long and overlaps no
// entry other than sel, the running entry included.
func fits(entries []toggl.TimeEntry, sel toggl.TimeEntry, start, stop, now time.Time) bool {
	if stop.Sub(start) < time.Minute {
		return false
	}
	for _, e := range entries {
		if e.ID != sel.ID && e.Start.Before(stop) && end(e, now).After(start) {
			return false
		}
	}
	return true
}
