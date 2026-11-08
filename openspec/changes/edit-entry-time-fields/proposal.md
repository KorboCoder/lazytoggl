# Proposal

## Why

Fixing an entry's times means editing a raw `2006-01-02 15:04` string in a plain text
field: backspacing over digits, retyping them, and getting the format exactly right.
The two edits that actually happen most — changing an entry's duration, and moving its
start or stop to fit a timeslot — should take a few keystrokes, not careful string
surgery.

## What Changes

- The edit dialog becomes a form with four rows: description, start, stop, and a new
  **duration** row. Start and stop show their date read-only next to an `HH:MM` time.
- `enter` on a start, stop, or duration row opens an `HH:MM` time field over the form.
  The field is digits-only, so it takes vim-style motions without modes: digits
  overwrite in place, `h`/`l`/`^`/`$`/`w`/`b` move, `ctrl+a`/`ctrl+x` increment and
  decrement the hours or minutes under the cursor, with minutes carrying into hours.
- Start, stop, and duration stay consistent: editing start keeps stop fixed, editing
  stop or duration keeps start fixed, and every entry stays at least 1 minute long —
  a time that would cross the other end falls back to 1 minute from it.
- Running entries: stop and duration are not editable.
- Untouched times keep their exact original timestamp, including seconds. (Today any
  save silently zeroes the seconds of start and stop.)
- **BREAKING (keys):** `enter` no longer saves the edit dialog. `esc` closes the form;
  if anything changed, a review popup shows each changed field before → after, with
  save, back to the form, or discard. If nothing changed, the dialog closes with no API
  call.
- Free-text timestamp entry and its "use format …" error go away.

Out of scope, planned as the next change: editing the date (moving an entry to
another day).

### Keybindings

| Context | Key | Action |
|---|---|---|
| Edit form | `tab` / `shift+tab`, `down` / `up` | Move between rows (unchanged) |
| Edit form | `enter` on start / stop / duration | Open the `HH:MM` time field **(new)** |
| Edit form | `enter` on description | Move to the next row **(changed: used to save)** |
| Edit form | `esc` | Close the form → review if anything changed **(changed: used to cancel)** |
| Time field | `0`–`9` | Overwrite the digit under the cursor and advance **(new)** |
| Time field | `h` / `l`, `^` / `$`, `w` / `b` | Move cursor; first / last digit; minutes / hours **(new)** |
| Time field | `ctrl+a` / `ctrl+x` | +1 / −1 on the hours or minutes under the cursor **(new)** |
| Time field | `enter` / `esc` | Accept / revert the field **(new)** |
| Review | `y` / `enter` | Save **(new)** |
| Review | `esc` | Back to the form, edits intact **(new)** |
| Review | `n` | Discard all edits and close **(new)** |

## Capabilities

### New Capabilities

- `entry-editing`: how a time entry's description, start, stop, and duration are edited
  and saved from the edit dialog.

### Modified Capabilities

None — no specs exist yet.

## Impact

- `internal/ui/dialogs/edit.go` — rewritten as a form with an internal time-field and
  review state; still emits `EditSubmitMsg` / `CloseMsg`.
- `internal/ui/app.go` — only passes the current time when opening the edit dialog.
- New `internal/ui/dialogs/timefield.go` — the reusable `HH:MM` widget.
- `internal/ui/driver_test.go` — `TestEditValidation` and `TestEditSavesDescription`
  are rewritten for the new flow; new cases cover the time field and review.
- `docs/REQUIREMENTS.md` (keymap) and `docs/PLAN.md` (decisions log) are updated.
- API: no change. A save is still one `PUT`; closing with no changes or discarding
  costs nothing.
