# Design

## Context

Today `dialogs.Edit` (`internal/ui/dialogs/edit.go`) is a list of `bubbles/textinput`
fields. Times are pre-filled with `Format("2006-01-02 15:04")` and parsed back with
`time.ParseInLocation` on `enter`, which is also the save key. The dialog talks to
`app.go` only through `EditSubmitMsg` / `CloseMsg`, and `app.go` holds a single
`m.dialog`. `app.go` also owns the clock (`m.now`, ticked every second, pinned to
`fixedNow` in `driver_test.go`). Requirements are in `specs/entry-editing/spec.md`.

## Goals / Non-Goals

**Goals:**
- A reusable `HH:MM` widget that the date-editing change (next) and other dialogs
  (e.g. a start time in the start dialog) can reuse with a different mask or bounds.
- Keep the change inside `internal/ui/dialogs`, with `app.go`'s message contract
  unchanged.

**Non-Goals:**
- Modal vim editing (normal/insert modes) for free text like the description.
- Editing dates, or moving an entry to another day.
- Any change to `toggl.Service`, the HTTP client, or the mock.

## Decisions

### 1. A custom `TimeField` model instead of `textinput`

`TimeField` (`internal/ui/dialogs/timefield.go`) stores its value as minutes since
midnight plus a cursor index 0–3 over the four digits. It renders `HH:MM` with a block
cursor on the current digit.

- *Why:* the `HH:MM` shape and "cursor never on the colon" are invariants of the data,
  not checks after the fact. `textinput` has no overwrite mode and lets the cursor and
  the text drift into any shape.
- *Alternative:* `textinput` plus a validator and cursor fixups. Rejected: it fights the
  widget on every keystroke for a 5-character field.

Typed digits are stored as raw digits, so `27:10` can exist until `enter` checks the
range. `ctrl+a`/`ctrl+x` work on the numeric value instead (±60 or ±1 minutes), which
gives carry and borrow for free, then clamp to the field's `[min, max]` bounds. Start
and stop use `[00:00, 23:59]`, duration `[00:01, 23:59]`. The bounds are constructor
parameters, which is what makes the widget reusable for duration.

### 2. Vim motions without modes

Only digits are content, so `h l ^ $ w b` and `ctrl+a/x` can be bound directly. Any
other key is ignored. This gives vim users the muscle memory without a mode indicator,
an `i`/`esc` dance, or a second keymap.

### 3. `Edit` owns a small state machine; no dialog stack in `app.go`

```
  form --enter on time row--> field --enter (valid) / esc--> form
   |
   | esc
   v
  unchanged? --yes--> CloseMsg
   | no
   v
  review --y/enter--> EditSubmitMsg
         --esc------> form (draft intact)
         --n--------> CloseMsg
```

While the field is open, `Edit.View()` draws it over the form with `theme.Overlay`,
next to the focused row. The review replaces the form's content inside the same box.

- *Why:* `app.go` routes every key to the single open dialog, so nested popups can stay
  private to `Edit`. The review is part of editing, not a general confirm.
- *Alternative:* a dialog stack in `app.go`. Rejected: it widens the root model's
  routing for one caller. Reusing `Confirm` for the review was also rejected, since it
  only knows delete and has two outcomes, not three.

### 4. A draft with "touched" tracking preserves seconds

`Edit` keeps the original entry and a draft (`description`, `start`, `stop`). Duration
is never stored; it is always `stop - start`, shown floored to whole minutes.

Accepting a field writes into the draft:
- **Start or stop:** `time.Date(orig.Year(), orig.Month(), orig.Day(), h, m, 0, 0, time.Local)`.
  The value keeps its own date, has zero seconds, and DST is normalized by `time.Date`.
- **Duration:** `stop = start + d`.

The consistency and 1-minute rules from the spec are then applied in one place,
`normalize(changed field)`.

Unchanged fields are never rewritten, so seconds survive. "Changed" means the draft
differs from the original: trimmed description, `!start.Equal`, `!stop.Equal`. That
comparison drives both the skip-review rule and the review's lines.

### 5. `NewEdit` takes the current time

`NewEdit(entry, now time.Time)` uses `now` for the running-entry rule
(start ≤ now − 1 minute). `app.go` passes `m.now` at the call site in the `Edit` key
handler. That is its only change, and it keeps tests deterministic against `fixedNow`.

### API request cost

Startup: unchanged (2). Refresh: unchanged (1). Saving an edit: 1 `PUT`, as today.
Closing with no changes, going back from the review, or discarding: 0.

## Risks / Trade-offs

- [`ctrl+a` is a common tmux prefix, so tmux users with that binding won't reach the
  app] → Document it in the help line. Adding `+`/`-` aliases later is a one-line
  binding if it turns out to matter.
- [`enter` no longer saves, so muscle memory from the old dialog breaks] → `esc` now
  leads to a review instead of discarding, so the old habit can't silently lose an edit.
- [Entries that cross midnight: editing stop on a next-day stop to a "smaller" hour
  stays on its own day, but the user may expect a different day] → Out of scope until
  date editing. Each time keeping its own date is the predictable behavior meanwhile.
- [Entries of 24h or longer don't fit the duration field's 23:59 bound] → The duration
  row still shows the real length. Opening its field starts at 23:59, and accepting
  shortens the entry. This is rare enough to defer.

## Migration Plan

UI-only, with no data or config migration. Rollback is reverting the commit.
