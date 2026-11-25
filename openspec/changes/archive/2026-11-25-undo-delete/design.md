# Design

## Context

- `d` opens `dialogs.Confirm`; `y` emits `DeleteConfirmMsg`, which runs `deleteCmd`
  (`DELETE /workspaces/{wid}/time_entries/{id}`, 1 request). Its `entryDeletedMsg`
  removes the entry from `m.entries` and clears `pausedID` if it was the paused one.
- `m.entries` is the only source of truth for the list and the timer panel.
  `runningEntry()` and `pausedEntry()` look entries up there, so removing an entry
  from `m.entries` hides it everywhere.
- The app already ticks once a second (`tickMsg` sets `m.now`). That is precise
  enough for a 5-second grace period and needs no extra timer.
- A refresh (`dataLoadedMsg`) replaces `m.entries` with whatever Toggl returns.
- `q` and `ctrl+c` return `tea.Quit` at once; `ctrl+c` does so even with a dialog
  open.

Requirements: `specs/entry-deleting/spec.md`. Motivation: `proposal.md`.

## Goals / Non-Goals

**Goals:**
- Undo costs nothing: no request until the grace period ends.
- A delete the user made is sent, even when they quit right after making it.

**Non-Goals:**
- Undo for anything other than deletes (edits, starts, logs).
- More than one pending delete, or undo after the delete was sent (that would mean
  re-creating the entry).

## Decisions

### 1. A pending delete is a hidden copy plus a deadline

```go
type pendingDelete struct {
	entry    toggl.TimeEntry
	paused   bool      // it was the timer panel's paused entry
	deadline time.Time // m.now at delete + 5s
}
```

`Model.pending *pendingDelete`.

- **`d`** removes the target from `m.entries` (and clears `pausedID` if it was
  paused), stores it in `pending`, and clamps the cursor.
- **`u`** puts it back with `replaceEntry` (which re-sorts), restores `pausedID`,
  moves the cursor to it with `SelectID`, and clears `pending`.

Because the timer panel reads `m.entries`, hiding and restoring the running or paused
entry needs no special code beyond `pausedID`.

- *Alternative:* keep the entry in `m.entries` with a "pending" flag and filter it
  out on every read. Rejected: every reader of `m.entries` (rows, totals, the timer
  panel, gaps, `entryByID`) would have to remember the filter.

### 2. The deadline is checked on the existing tick

On `tickMsg`, if `pending != nil` and `!m.now.Before(pending.deadline)`, the app
sends the delete (`commitDelete`). The countdown shown is
`ceil(deadline − now)` in whole seconds.

- *Alternative:* a dedicated `tea.Tick(5s)` per delete. Rejected: it needs a token to
  ignore stale ticks after an undo or a second delete, and the 1-second tick already
  exists.

### 3. Sending, and getting the entry back on failure

`commitDelete` moves the entry from `pending` into `m.deleting` (a map of IDs to the
hidden copies, with whether each was paused) and returns
`deleteCmd(entry, paused)`.

- **On success,** `entryDeletedMsg` drops it from `m.deleting`.
- **On failure,** a new `deleteFailedMsg{entry, paused, err}` restores the entry as
  for an undo, then shows the error like any `errMsg`: the notice, the status, and
  the rate-limit wording.
- **Refreshes** filter out both the pending ID and the IDs in `m.deleting`, so an
  entry whose `DELETE` is in flight doesn't flash back. The fresh data replaces the
  hidden copy, so an undo restores what Toggl has now.

### 3a. Hidden copies follow what happens to the entry

The pending and in-flight copies (`eachHidden`) get the same updates the entry would
get if it were listed, so an undo or a failed delete brings back the right state:

- **A new start** stops a running hidden copy and clears its paused flag, as it does
  for listed entries. Otherwise an undo would bring back a second running timer, or
  re-pause an entry the new start superseded.
- **A stop, pause, or edit answered after the delete** (`patchHidden`) updates the
  hidden copy instead of `m.entries`. Before this, the reply's `replaceEntry` put the
  entry back on screen while its delete was pending, so it could be deleted twice.
- **A failed delete** restores the entry without moving the cursor (`restore` only
  clamps it). `u` moves the cursor to the entry; a failure arrives while the user may
  be doing something else.

### 4. One pending delete: a second `d` commits the first

`d` while `pending != nil` first runs `commitDelete` on the pending one, then makes
the new one pending. The returned command is that delete request.

### 5. Quitting sends first

`q`/`ctrl+c` with a pending delete (or one in flight) sends it, sets
`quitting`, and shows `deleting before quitting… (q again to quit now)`. When the
last delete is answered, success or failure, the app quits.

- *Changed during implementation:* the plan was `tea.Sequence(delete, tea.Quit)`.
  But a slow network, with the client's retries, could hold that quit for a long
  time with no way out. The flag lets a second `q` quit at once, and it is
  straightforward to test.

While quitting:

- Only `q` and `ctrl+c` act; every other key is ignored, so no new delete or edit
  can start and then be dropped on exit. `ctrl+c` in an open dialog closes it.
- The status line keeps the quitting status, whatever other answers arrive.
- Either way the app quits.
- If a delete failed, or a second `q` quit before Toggl answered, the entry may
  still exist in Toggl. That is the same safe outcome as the app being killed, but it
  is not silent: `Model.QuitError()` collects these, and `main.go` prints them to
  stderr once the screen is restored and exits with status 1.

### 6. The status line shows the hint next to other statuses

`View` renders `m.status`, and while `pending != nil` it appends
`· deleted "<desc>" · u undo (<n>s)`, or shows the hint alone when there's no other
status. Other actions keep calling `setStatus` as they do now, so the hint can't be
overwritten.

When the line is wider than the terminal (`statusTexts`), the description shortens
first (to a few characters and `…`), then the other status, so `u undo (<n>s)`
always stays on screen.

### 7. The confirm dialog goes away

`dialogs/confirm.go`, `NewConfirm`, and `DeleteConfirmMsg` are deleted. The driver
test helper's list of fed messages drops `DeleteConfirmMsg` and adds
`deleteFailedMsg`.

### API request cost

| | Requests |
|---|---|
| `d` | 0 (sent when the grace period ends: 1) |
| `u` | 0 |
| A second `d`, or quitting, while one is pending | sends the pending one now (1) |
| Startup / refresh | unchanged |

Total per delete: 1, as today. Undone deletes cost nothing.

## Risks / Trade-offs

- [The app is killed (not `q`/`ctrl+c`) during the grace period] → The delete is
  never sent and the entry reappears next launch. This is the safe direction, and the
  backlog accepted it.
- [The deadline counts from the last tick, which can be up to a second before the
  key press, so the grace period lasts between 4 and 5 seconds] → Acceptable for an
  undo window, and it needs no timer of its own. The countdown shows the time left
  until the delete is sent.
- [Muscle memory for `d` then `y`] → `y` is unbound in the list, so a reflexive `y`
  does nothing, and the status shows the undo hint.

## Migration Plan

UI only, with no API, data, or config change. Rollback is reverting the commits.
