# Design

## Context

The edit dialog (`internal/ui/dialogs/edit.go`) holds a draft (`start`, `stop`) and
opens a `TimeField` (`timefield.go`) over the form for start, stop, or duration.
`TimeField` stores four raw digits and a cursor; `ctrl+a`/`ctrl+x` step its value and
clamp to `[min, max]` minutes, which is why start and stop can't cross midnight today.
`applyField` writes an accepted value back with `atClock(t, minutes)` (same date, zero
seconds) and applies the 1-minute rule. `restoreIfSameMinute` gives back the original
timestamp when a time is retyped.

On save, `app.go` handles `entryUpdatedMsg` with `replaceEntry`, which swaps the entry
in place. `panels.BuildRows` starts a new day header whenever the day changes between
neighbours, so it relies on `m.entries` being sorted by start, newest first.
`RecentEntries` fetches `now − 192h → now` (`recentEntriesLookback`, unexported, in
`internal/toggl/client.go`); the mock returns all its entries regardless.

Requirements are in `specs/entry-editing/spec.md`; motivation in `proposal.md`.
Examples in the spec assume today is Sun 08 Nov 2026.

## Goals / Non-Goals

**Goals:**
- Keep `TimeField` as the `HH:MM` widget, untouched for duration, and build the date on
  top of it instead of widening its mask.
- Keep the midnight rules (carry, resolution, bounds) in one place per concern: the
  field owns carry and bounds, `Edit` owns everything that depends on the other end.
- No change to `toggl.Service` or the HTTP client's requests.

**Non-Goals:**
- Typed dates. Digits and motions on the date segment are ignored and reserved.
- Durations of 24h or more in the duration field (rows still show the real length).
- Changing the fetch window.

## Decisions

### 1. `DateTimeField` wraps `TimeField`

New `internal/ui/dialogs/datetimefield.go`:

```
DateTimeField
  day       time.Time      // midnight of the field's explicit date
  clock     TimeField      // the HH:MM segment (cursor, raw digits, range error)
  seg       segDate|segTime
  lo, hi    time.Time      // bounds; lo may be zero (no lower bound)
  stepped   int            // net days stepped on the date segment
  dated     bool           // date changed by hand (date step or carry)
  clocked   bool           // time segment edited (a digit typed or a step)
  resolve   func(day time.Time, minutes int) time.Time   // nil: no resolution
```

- `tab`/`shift+tab` flip `seg`. `TimeField` keeps its cursor while the date is active.
- On the time segment, every key except `ctrl+a`/`ctrl+x` goes to `TimeField.Update`
  unchanged, so typing, motions, range errors, `enter`, and `esc` behave exactly as
  today. `TimeField` gains a `SetValue(minutes)` that keeps the cursor.
- `ctrl+a`/`ctrl+x` on the time segment are handled by the wrapper: take `Value()`,
  add ±1 or ±60 minutes as a `time.Duration`, clamp to `[lo, hi]`, then split the
  result back into `day` and the clock. If the day changed, set `dated`.
- `ctrl+a`/`ctrl+x` on the date segment: `day.AddDate(0, 0, ±1)`, ignored if it would
  leave `[lo.day, hi.day]`. Adjust `stepped` and set `dated`.
- `Value()` is `resolve(day, minutes)` while `!dated && resolve != nil`, else
  `atClock(day, minutes)`. `View()` renders `Value()`'s date, so the shown date
  updates with every keystroke.

*Why a wrapper:* the date segment and the clock have different keys and units. A mask
that included a date would have to teach `TimeField` about month lengths and weekdays,
and duration would carry dead code. The wrapper also leaves every existing
`timefield_test.go` case valid.

*Alternative:* one generalized masked field. Rejected for the reasons above.
*Alternative:* store a single `time.Time` in the wrapper. Rejected: raw digits must be
able to hold `27:10` until `enter`, which `TimeField` already models.

### 2. `Edit` supplies the resolver and the bounds

`Edit` constructs the field with closures over its draft, because resolution depends
on the *other* end, which the field doesn't know:

- **Stop:** `c := atClock(start's day, m)`; if `!c.After(start)`, `c = c.AddDate(0,0,1)`.
  Use `c` if `c − start ≤ 12h` and `c` is within bounds, else `atClock(day, m)`.
- **Start:** `other` is stop, or `now` for a running entry. `c := atClock(other's day, m)`;
  if `!c.Before(other)`, `c = c.AddDate(0,0,-1)`. Use `c` if `other − c ≤ 12h` and in
  bounds, else `atClock(day, m)`.

The existing 1-minute rule and the running-entry `now − 1min` cap still run in
`applyField` after resolution, unchanged, so every current scenario keeps passing. The
spec's "more than 12 hours stays on its day" case falls straight through to them.

Bounds: `hi` is today 23:59 for both fields. `lo` is today − 7 days at 00:00 for a
running entry's start (always inside the 192h fetch window, which starts at the
same clock time 8 days back) and zero otherwise.

### 3. `applyField` takes a timestamp and a step count

For start:

1. `n := field.Stepped()`. If `n != 0` and the entry is finished, `stop = stop.AddDate(0,0,n)`.
2. If the time segment was not edited (`!field.Clocked()`),
   `start = start.AddDate(0,0,n)`, which keeps seconds. Otherwise `start = field.Value()`,
   with zero seconds.
3. `restoreIfSameMinute` against the original, then the existing 1-minute and running
   rules.

Stop is the same without step 1. Duration is unchanged. Moving by the date-segment
step count, not by "the date differs", is what keeps a midnight carry or an overnight
resolution from dragging stop along (spec: "Crossing midnight by time keeps stop").

`AddDate` keeps wall-clock time across DST; `time.Date` normalizes a non-existent local
time, as today.

### 4. Review lines

`changes()` formats a start or stop line with `dateLayout + " " + HH:MM` on both sides
when the before and after dates differ, and as `HH:MM` otherwise, so existing review
scenarios are unchanged. When the new start is outside the window (before its start
or after now), a muted `(outside list)` line follows the start line, indented under the values. A dated line
is 12 + 36 columns, which fits the 56-wide box; the flag would not fit on the same line.

### 5. The window is a `toggl` package value, not a `Service` method

Export the lookback from `internal/toggl` (`RecentEntriesLookback`) with helpers
`RecentEntriesSince(now time.Time) time.Time` and `InRecentWindow(start, now) bool`
(start within `[RecentEntriesSince(now), now]`). The client uses `RecentEntriesSince`
to build `start_date`. `Edit` (review flag), `app.go` (status line), and the mock's
`RecentEntries` filter (plus the running entry) all use `InRecentWindow`, so the
warning and `-mock`'s drop-off on `r` agree with each other and with the real API's
`start_date`/`end_date`. That the API leaves out entries starting after `end_date` is
assumed from the parameter, not confirmed live.

*Alternative:* a `Service.Window()` method. Rejected: it isn't an API call, every
implementation would return the same constant, and the interface is meant to list
requests.

### 6. Re-sort in `replaceEntry`, cursor follows on update

`replaceEntry` sorts `m.entries` by start, newest first, with `sort.SliceStable` after
replacing. This applies to stop and pause too, where it's harmless. On
`entryUpdatedMsg` only, the app rebuilds rows and moves `entriesPanel.Cursor` to the
row holding the edited entry's ID (a small `panels.Entries.SelectID(rows, id)`). The
panel's `View` already scrolls the cursor into view.

### 7. Save status

On `entryUpdatedMsg`, when the edit changed the start (compared with the entry before
the save) and the saved start is outside the window, the status reads
`entry saved — moved to Mon, Oct 12, outside the 8-day list; it drops off on refresh`
for a start before the window, or `entry saved — starts after now, outside the list;
it drops off on refresh` for one after now. The day count comes from the lookback; the
date uses the list's day-label format. Otherwise it stays `entry saved`, including when
the window slid past an entry whose start the edit didn't touch.

### 8. Field help and popup

The field's help gains `tab date/time` on start and stop; with the date segment active,
the first help line reads `ctrl+a/x +/-1 day · tab time`. The popup grows to
`Sat 07 Nov  14:32` (17 columns + border). It still opens at the right of the box,
which starts after the row's date and time, so the row stays readable.

### API request cost

Startup: unchanged (2). Refresh: unchanged (1). Saving an edit: 1 `PUT`, as today,
whether or not a date changed. Stepping dates, the review flag, and the status message
cost 0.

## Risks / Trade-offs

- [Retyping a stop on an entry longer than 12h with an untouched date can land it on
  the day after start] → The date updates live in the field and the review shows dated
  lines; a date set by hand is never re-resolved. Entries this long are rare.
- [Moving a far-back date takes one `ctrl+x` per day] → Rare by design; digits on the
  date segment are reserved for typed dates later.
- [An entry moved before the window vanishes on the next refresh] → Flagged in the
  review, announced in the status, kept in the list until refresh. The data is safe in
  Toggl.
- [Moving an overnight entry's start date to today can put its stop on tomorrow] → The
  bound applies to what the user sets in a field; a stop pushed along by a whole-entry
  move is allowed. Rare, and visible in the review.
- [The mock now filters by window using the wall clock] → Driver tests feed fixtures
  through `dataLoadedMsg` and don't depend on it. `TestRefreshReloadsFromService` uses
  the mock's seed, which is within 2 days.

## Migration Plan

UI-only, plus an exported constant. No data or config migration. Rollback is reverting
the commit.
