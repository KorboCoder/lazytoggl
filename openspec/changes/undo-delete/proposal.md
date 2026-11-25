# Proposal

## Why

Deleting an entry is the only action in lazytoggl that can't be taken back, and the
confirm dialog only guards against pressing `d` by accident, not against confirming
the wrong entry. Toggl's own apps delete at once and offer "Undo" instead. The
backlog already plans this as a grace period before the API call
(`docs/REQUIREMENTS.md`, "Deletion undo"), which makes undo free under the
30-requests/hour limit: nothing is sent until the grace period ends.

## What Changes

- **`d` deletes at once, with no confirm dialog.** The entry disappears from the list,
  or from the timer panel if it's the running or paused one. The status line shows
  `deleted "<description>" · u undo (5s)`, counting down.
- **`u` undoes** within the 5 seconds, from the list or the timer panel. The entry
  comes back exactly as it was: still running, or still paused, with the cursor on
  it. Nothing is sent to Toggl.
- **After 5 seconds the `DELETE` is sent** (1 request, as today). If it fails, the
  entry comes back and the error notice shows, as for any failed request.
- **One pending delete at a time.** Pressing `d` again while one is pending sends the
  first delete right away, and the new one becomes the one `u` undoes.
- **Refreshing** (`r`) keeps a pending entry hidden, even though Toggl still returns
  it.
- **Quitting** (`q`, `ctrl+c`) while a delete is pending sends it first, then quits.
  If the app is killed outright, the delete is never sent and the entry survives,
  which is the safe way to fail.
- **The undo hint stays visible** next to any other status until the grace period
  ends, so `u` never works invisibly.
- **BREAKING (keys):** the "Delete entry?" dialog and its `y`/`n` keys go away.

### Keybindings

| Context | Key | Action |
|---|---|---|
| List / timer panel | `d` | Delete at once, undoable for 5 seconds **(changed: used to open a confirm dialog)** |
| List / timer panel | `u` | Undo the pending delete **(new)** |
| Delete-confirm dialog | `y` / `enter`, `n` / `esc` | **Removed** with the dialog |
| Anywhere | `q` / `ctrl+c` | Quit; sends a pending delete first **(changed)** |

## Capabilities

### New Capabilities

- `entry-deleting`: deleting an entry with an undo grace period, including when the
  delete is sent, undone, kept through a refresh, or sent on quit.

### Modified Capabilities

None. No existing spec covers deleting.

## Impact

- `internal/ui/app.go`: a pending-delete state, hiding and restoring the entry
  (including running and paused), sending on tick when the grace period ends, on a
  second `d`, and on quit; filtering it out of refreshes; the status hint; restoring
  on failure.
- `internal/ui/keys.go`: a `u` binding; the `d` help text.
- `internal/ui/dialogs/confirm.go` and `DeleteConfirmMsg`: removed.
- Tests: driver tests replace `TestDeleteCancelAndConfirm` and the timer-focus delete
  check; a new `delete_test.go`.
- Docs: `docs/REQUIREMENTS.md` (feature list, keymap, removes the backlog item),
  `README.md` (keys table), `docs/PLAN.md` (decision).
- API: unchanged cost. A delete is 1 request, sent after the grace period, and an undo
  is 0. Startup and refresh are unchanged.
