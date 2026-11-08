# Proposal

## Why

The edit dialog can only change an entry's times; its dates are read-only. That leaves
two gaps: an entry logged on the wrong day can't be moved, and an entry that runs past
midnight can't be fixed at all — a start or stop can't cross the day boundary, and
`ctrl+a` stops dead at 23:59. Dates are edited rarely, so this should be complete
without adding anything to the common path of editing a time.

## What Changes

- The start and stop time fields show their **date** next to the time, always visible.
  `tab` / `shift+tab` inside the field switch between the date and time segments. The
  form, its rows, and its tab order are unchanged; the duration field stays time-only.
- On the date segment, `ctrl+a` / `ctrl+x` move ±1 day. Dates can't go past today.
- On the time segment, `ctrl+a` / `ctrl+x` now **carry across midnight** into the date
  (`23:59` → `00:00` the next day) instead of stopping at the day boundary. **BREAKING
  (behavior):** the "day boundary holds" rule is removed for start and stop.
- Changing **start's date moves the whole entry** (stop moves by the same days, so the
  duration is kept). Changing **stop's date** moves only stop, which is how an overnight
  entry is made.
- **Typed times cross midnight when it makes sense:** a typed stop lands on the first
  occurrence of that time after start, and a typed start on the last occurrence before
  stop (or before now, for a running entry), when that gives an entry of 12h or less.
  Otherwise the time stays on the field's date and the existing 1-minute fallback
  applies. The field's date updates live as you type, so the step is never hidden. A
  date the user changed in the field is not re-resolved.
- A **running entry's** start date can't move earlier than 7 days before today, so the
  next refresh still finds it.
- A **finished entry can move before the list's fetch window**. The review marks it
  `(outside list)`, and after saving the status line says it will drop off on the next
  refresh. Until then it stays in the list, so the move can be fixed.
- The review shows dates on start and stop lines when a date changed.
- After a save, the entries list **re-sorts by start** so the entry sits under its
  (possibly new) day, and the cursor stays on the edited entry. This also fixes a
  pre-existing bug: `replaceEntry` swapped entries in place, so an edited start could
  leave the list out of order or split a day into two groups.
- A start or stop moved only by its date keeps its seconds.

Out of scope: typing a date with digits (the date segment only steps), editing a
duration of 24h or more, and widening the fetch window.

### Keybindings

| Context | Key | Action |
|---|---|---|
| Start / stop field | `tab` / `shift+tab` | Switch between the date and time segments **(new)** |
| Date segment | `ctrl+a` / `ctrl+x` | +1 / −1 day **(new)** |
| Time segment | `ctrl+a` / `ctrl+x` | +1 / −1 hour or minute, now carrying across midnight into the date **(changed: used to stop at 00:00 / 23:59)** |
| Date segment | `0`–`9`, `h l ^ $ w b` | Ignored **(new; reserved)** |
| Duration field | `tab` / `shift+tab` | Ignored (no date segment) |

All other edit-dialog keys are unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `entry-editing`: start and stop fields gain a date segment; increments carry across
  midnight; date changes move the whole entry (start) or only stop; typed times resolve
  across midnight within 12h; untouched-seconds rule covers date-only moves; the review
  shows dates and flags moves outside the list; a saved entry is placed under its day
  with the cursor following it, and a move outside the list is announced.

## Impact

- `internal/ui/dialogs/timefield.go` — unchanged as the `HH:MM` widget; a new
  `internal/ui/dialogs/datetimefield.go` wraps it with a date segment, `tab`, carry, and
  date bounds.
- `internal/ui/dialogs/edit.go` — start/stop rows open the date+time field; new date and
  resolution rules in `applyField`; review shows dates and the outside-list flag.
- `internal/ui/app.go` — `replaceEntry` re-sorts and keeps the cursor on the edited
  entry; the save status mentions a move outside the list.
- `internal/toggl` — the fetch lookback is exported so the UI can tell where the list
  window starts; the mock's `RecentEntries` filters to that window like the real API.
- Tests: `internal/ui/dialogs/datetimefield_test.go` (new), `internal/ui/edit_test.go`,
  `internal/ui/driver_test.go`, `internal/toggl/mock_test.go` or equivalent.
- Docs: `docs/REQUIREMENTS.md` (feature list, keymap, editing rules), `docs/PLAN.md`
  (decisions log), `docs/ARCHITECTURE.md` (exported lookback).
- API: no change. A save is still one `PUT`; startup and refresh costs are unchanged.
