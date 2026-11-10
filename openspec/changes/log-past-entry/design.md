# Design

## Context

- `Service.Start` is the only way to create an entry. The client sends
  `start = now, duration = -1`. The mock and the app treat a start as "the running
  entry stops": `entryStartedMsg` (`app.go`) closes any running entry locally.
- The edit dialog (`dialogs/edit.go`) already has everything a finished entry needs:
  the rows; the date+time and duration fields with their rules; the project row,
  picker, and `@` dropdown; `changes()` against an original entry; and the
  create-then-save path for a `(new)` project (`EditSubmitMsg.NewProject` →
  `createProjectCmd` → `projectCreatedMsg`).
- After a save, `replaceEntry` inserts or replaces the entry and re-sorts the list,
  `SelectID` moves the cursor to it, and `savedStatus` warns when the start left the
  fetch window.
- The live check in `pick-entry-project` created a finished entry with
  `POST .../time_entries` and `start`, `stop`, `duration`, which succeeded. Whether
  that leaves a running timer running wasn't observed.

Requirements: `specs/entry-logging/spec.md`. Motivation: `proposal.md`.

## Goals / Non-Goals

**Goals:**
- Reuse the edit form as it is, with the same rows, fields, rules, and picker, so
  logging and editing can't drift apart.
- Never touch the running entry, locally or through the API.

**Non-Goals:**
- `o` / `O` gap-filling from a selected entry (a follow-up).
- Overlap detection.
- Starting a running timer with a past start ("I started 15 minutes ago"). Editing
  the running entry's start already does this.

## Decisions

### 1. A log mode on `Edit`, not a new dialog

`NewLog(now, projects)` returns an `*Edit` built from a synthetic, unsaved finished
entry, `{Start: stop−30m, Stop: now truncated to the minute}` with ID 0, plus
`mode: modeLog`. That synthetic entry is the `orig` the dialog compares against, so
`changes()` reports exactly what the user changed from the defaults. The mode
changes only four things:

| | edit mode | log mode |
|---|---|---|
| Title | `Edit entry` | `Log entry` |
| `enter` on description | next row | submit `LogSubmitMsg` |
| `esc` on the form | review if changed, else close | discard prompt if changed, else close |
| Review | yes | none |

Everything else, including the fields, resolvers, bounds, picker, and `@`, runs the
same code. The `@` dropdown already owns `enter` and `esc` while open, so "pick, don't
log" needs no extra code.

- *Why:* the edit form is the larger part of this feature, and an edit and a log
  should behave the same for the same keys.
- *Alternative:* a separate `Log` dialog that composes shared form pieces. Rejected:
  it would mean pulling the rows, fields, `applyTime`, and the picker out of `Edit`,
  a big refactor for four small differences.

### 2. The discard prompt is a state like the review

`confirmingDiscard` is set when `esc` finds changes. While it is set, the form stays
visible and the help line reads `Discard this entry? y discard · n back`. `y` sends
`CloseMsg`, and `n` or `esc` clears the state. Every other key is ignored, as in the
review, so a stray letter can't edit the hidden form.

### 3. `Service.Log` creates a finished entry

```go
type LogRequest struct {
	Description string
	ProjectID   int64 // 0: no project
	Start, Stop time.Time
}
Log(ctx context.Context, req LogRequest) (TimeEntry, error)
```

- **Client:** `POST /workspaces/{wid}/time_entries` with `workspace_id`,
  `description`, `start`, `stop` (RFC3339), `duration` (seconds),
  `created_with: "lazytoggl"`, and `project_id` only when it isn't 0. The response is
  decoded like any entry.
- **Mock:** appends a finished entry with the next ID and never stops the running
  one.

*Why not extend `Start`:* `Start` means "now, running, stops the current timer" in
the client, the mock, and the app's `entryStartedMsg`. A separate method keeps that
contract and makes "logging doesn't stop the timer" true by construction.

### 4. App flow

```
 m --> NewLog(m.now, m.projectList())
 LogSubmitMsg{Req, NewProject}
   NewProject == "" --> logCmd(req) ------------------------> entryLoggedMsg
   NewProject != "" --> createProjectCmd(name, "logging…", next)
                           --> projectCreatedMsg --> next(id) = logCmd(req with id)
 entryLoggedMsg --> replaceEntry (insert + re-sort), SelectID,
                    status = loggedStatus(entry, now)
```

- `entryLoggedMsg` is new and is **not** `entryStartedMsg`, so nothing stops locally.
- `savedStatus` gains a verb: an edit says "entry saved", a log says "entry logged".
  The outside-window wording is reused. For a log there is no previous start, so
  the warning shows whenever the start is outside the window.
- `m` is a new `keyMap.Log` binding, shown in the help bar as `m log entry`.

### 5. Live fact to confirm (task 1)

The design depends on one API behaviour: a finished `POST` leaves a running timer
running. The check uses throwaway data, needs about 5 requests and the user's OK, and
runs only while the user has no timer of their own running:

1. start a throwaway running entry;
2. log a finished entry;
3. read the running entry back;
4. delete the throwaway running entry;
5. delete the throwaway finished entry.

If the running entry turns out to stop, pause and revisit the spec, because the app
would then need to mirror the stop.

### API request cost

| | Requests |
|---|---|
| Startup / refresh | unchanged |
| Opening, editing, or cancelling the log dialog | 0 |
| Logging | 1 (`POST`) |
| Logging with a new project | 2 (create the project, then `POST`) |

## Risks / Trade-offs

- [`m` from the timer panel while a timer runs could be read as acting on the timer]
  → The dialog is titled `Log entry` and never touches the timer; the help bar
  names it.
- [`esc` with changes needs a second key to discard] → Only when something changed.
  An untouched dialog still closes on the first `esc`, as in Toggl.
- [Log mode adds conditionals to `Edit`] → They are limited to the four rows in the
  table above, each behind one `mode` check, and the shared behaviour is covered by
  the existing edit tests.

## Migration Plan

UI and client changes plus one new `Service` method. No data or config migration.
Rollback is reverting the commits.
