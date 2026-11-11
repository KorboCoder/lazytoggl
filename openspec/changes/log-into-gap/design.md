# Design

## Context

- `m` opens `dialogs.NewLog(now, projects)`, the edit form in log mode, whose orig is
  a synthetic finished entry `{Start: stop−30m, Stop: now truncated to the minute}`.
  Logging, `@`, the discard prompt, and how the entry lands in the list are already
  specified (`entry-logging`) and tested.
- `m.entries` is kept newest-first by start (`replaceEntry` re-sorts), and
  `panels.BuildRows` groups it by the local day of each start. The selection is
  `entriesPanel.Selected(rows)`.
- `actionTarget` already distinguishes the list from the timer panel for `e` / `d`.

Requirements: `specs/entry-logging/spec.md` (delta). Motivation: `proposal.md`.

## Goals / Non-Goals

**Goals:**
- Keep the gap rules in one pure, table-tested place, independent of the dialog and
  the app.
- Change the log dialog only to accept its start and stop.

**Non-Goals:**
- Gaps across days, or several gaps at once.
- Changing what `m` does.

## Decisions

### 1. `gaps.go`: two pure functions

```go
// gapBelow / gapAbove return the gap next to sel among entries (sel included
// or not), and ok=false when it is shorter than a minute. now is the app clock.
func gapBelow(entries []toggl.TimeEntry, sel toggl.TimeEntry, now time.Time) (start, stop time.Time, ok bool)
func gapAbove(entries []toggl.TimeEntry, sel toggl.TimeEntry, now time.Time) (start, stop time.Time, ok bool)
```

**Same day:** another entry counts when its start is on the same local day as `sel`'s
start, and it isn't `sel` (compared by ID).

**Below:**
- `stop = sel.Start`.
- `start` is the latest end among same-day entries with `Start < sel.Start`. A
  running entry's end counts as `now`, so a gap can't open over a running timer.
- With none, `start = sel.Start − 30m`.

**Above:**
- A running `sel` returns `ok=false` with a reason (see 3).
- `start = sel.Stop`.
- `stop` is the earliest `Start` among same-day entries with `Start > sel.Start`,
  the running entry included.
- With none: when `sel`'s day is today (local date of `now`), `stop = now` rounded
  down to the minute. Otherwise `stop = sel.Stop + 30m`.

**Both:** `ok = stop.Sub(start) >= time.Minute`. A gap that would overlap an
existing entry comes out negative or under a minute, so it is refused rather than
clipped.

- *Why pure functions on the slice:* every edge case (overlaps, the running entry,
  day edges, seconds) is a row in a table test with no UI involved.
- *Why the latest stop and not just the neighbour row's stop:* with overlapping
  entries, the row directly below may end before an earlier, longer entry does.
  Using the maximum keeps the gap clear of every entry.

### 2. `NewLogAt(start, stop, now, projects)`

`NewLogAt` builds the same log-mode `Edit` with an orig of `{Start: start, Stop: stop}`,
seconds as given. `NewLog` becomes
`NewLogAt(stop−30m, stop, now, projects)` with `stop` = `now` rounded down to the
minute, so `m`'s behaviour and tests are unchanged.

Since the defaults are the orig, `changes()` is empty until the user edits. An
untouched `o` then `esc` closes at once, and an untouched `enter` logs the exact gap.
The edit form's rule that a changed time gets zero seconds applies only to rows the
user actually changes.

### 3. App wiring

```
 o / O (list focus, entry row selected)
   gapBelow / gapAbove(m.entries, row.Entry, m.now)
     ok   --> m.dialog = NewLogAt(start, stop, m.now, m.projectList())
     !ok  --> status: no gap below "<desc>" / no gap above "<desc>"
              / nothing after the running entry
 o / O (timer focus, or a header row)  --> nothing
```

- A new `keyMap.GapBelow` (`o`) and `keyMap.GapAbove` (`O`), in `FullHelp` only.
- An empty description reads as `(no description)` in the status, so the message
  never shows `""`.
- From there on, the existing `LogSubmitMsg` / `entryLoggedMsg` path is used
  unchanged.

### API request cost

| | Requests |
|---|---|
| Pressing `o` / `O` (dialog or status) | 0 |
| Logging the gap | 1, or 2 with a new project (as `m`) |
| Startup / refresh | unchanged |

## Risks / Trade-offs

- [`O` on today's newest entry when its stop is later than now (it was edited to
  end in the future)] → `stop = now` is before `start`, so there's no gap and the
  status says so.
- [Logging can leave the list unsorted for a moment if a refresh is in flight] → The
  same as for any mutation today; out of scope.
- [`o` on an entry at 00:10 falls back to 23:40 the previous day, so the 30-minute
  fallback crosses midnight] → This matches `m` shortly after midnight, and the
  dialog shows both dates.

## Migration Plan

UI only, with no API, data, or config change. Rollback is reverting the commits.
