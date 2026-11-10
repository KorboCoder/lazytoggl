# Proposal

## Why

The only way to create an entry is to start a timer now (`c`). Work that has already
happened, such as a meeting where nobody started the timer, can't be logged without
starting a timer and then editing its times. Toggl's own apps solve this with a manual
("log time") mode, which defaults to the last 30 minutes ending now. lazytoggl already
has every piece that mode needs: the edit form's date+time fields and rules, and the
project row with `@`.

## What Changes

- **`m` opens a "Log entry" dialog** from the list or the timer panel. It mirrors
  Toggl's manual-mode `M` key. `c` is unchanged.
- **It is the edit form for a new, finished entry.** It has the same rows
  (description, start, stop, duration, project) and uses the same date+time fields
  and rules: crossing midnight, the 1-minute minimum, the 12h wrap of typed times, no
  date past today, `@` and the project picker.
- **Defaults like Toggl:** stop is now, rounded down to the minute, and start is
  30 minutes before stop.
- **`enter` on the description logs the entry**, as in Toggl and the start dialog,
  with one request, or two when it creates a `(new)` project first. On the time rows
  and the project row, `enter` opens that row's field or picker, as in the edit form.
  While the `@` dropdown is open, `enter` picks a project and doesn't log.
- **`esc` cancels.** When nothing was changed from the defaults, it closes
  immediately. Otherwise it asks `Discard this entry?`: `y` discards, and `n` or `esc`
  goes back to the form.
- **There is no review step**, because every field is new.
- **After logging,** the entry is listed under its day with the cursor on it, the
  same as after an edit. If its start is outside the fetch window (older than 8 days,
  or later than now), the status line warns that it drops off on refresh, as edits
  already do.
- **Logging doesn't touch a running timer.** Unlike starting, it doesn't stop the
  running entry, locally or in Toggl.
- **No overlap warning,** as in Toggl: a logged entry may overlap others.

Out of scope, planned as a follow-up: `o` / `O` on a selected entry, which would log
into the gap below or above it (vim's "open line"). Also out of scope: tags and the
billable flag.

### Keybindings

| Context | Key | Action |
|---|---|---|
| List / timer panel | `m` | Open the log-entry dialog **(new)** |
| Log entry, description | `enter` | Log the entry (only while the `@` dropdown is closed) **(new)** |
| Log entry, start / stop / duration / project row | `enter` | Open the row's field or picker (as in the edit form) |
| Log entry | `tab` / `shift+tab`, `down` / `up` | Move between rows (as in the edit form) |
| Log entry | `esc` | Cancel; asks `Discard this entry?` only if something changed **(new)** |
| Discard prompt | `y` | Discard and close **(new)** |
| Discard prompt | `n` / `esc` | Back to the form **(new)** |

All field, picker, and dropdown keys are the same as in the edit dialog.

## Capabilities

### New Capabilities

- `entry-logging`: creating a finished entry for time that already passed, from the
  log-entry dialog: its defaults, how it is logged or cancelled, and how it lands in
  the list.

### Modified Capabilities

None. The edit form's field and project behaviour is reused unchanged and
referenced, not respecified.

## Impact

- `internal/toggl`: a new `Service.Log(ctx, LogRequest)` that creates a finished
  entry (`POST .../time_entries` with `start`, `stop`, `duration`), implemented in
  the client, the mock, and `httptest` fixtures.
- `internal/ui/dialogs/edit.go`: a log mode on the edit dialog, with defaults,
  `enter` to log, `esc` with a discard prompt, and no review; a new `LogSubmitMsg`.
- `internal/ui/app.go`, `keys.go`: the `m` binding; on success, insert, re-sort and
  select the entry without stopping the running one; reuse create-then-save for a
  `(new)` project and the outside-list status.
- Tests: `edit_test.go` / a new `log_test.go` (driver), `client_test.go`,
  `mock_test.go`.
- Docs: `docs/REQUIREMENTS.md` (feature list, keymap), `docs/ARCHITECTURE.md`
  (`Service` listing, endpoint, costs), `docs/PLAN.md` (decision).
- API: startup and refresh unchanged. Logging costs 1 request, or 2 with a new
  project. Opening, editing, or cancelling the dialog costs 0.
