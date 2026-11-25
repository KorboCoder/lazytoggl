# entry-deleting Specification

## Purpose

Defines how a time entry is deleted with an undo grace period: what `d` does at once,
how `u` takes it back, and when the delete is actually sent to Toggl, including on a
second delete, a refresh, a failure, and quitting.

## Requirements

### Requirement: Deleting with d
`d` on the selected list entry, or on the entry the timer panel shows, SHALL remove it from view at once, with no confirmation. The status line SHALL show `deleted "<description>" · u undo (<n>s)`, counting down the grace period of 5 seconds, using `(no description)` for an empty one. Nothing SHALL be sent to Toggl yet.

#### Scenario: Delete from the list
- **WHEN** the user selects "Yesterday task" and presses `d`
- **THEN** the entry is no longer listed, the status shows `deleted "Yesterday task" · u undo (5s)`, and no request is made

#### Scenario: Delete the running entry
- **WHEN** the timer panel shows the running entry and the user presses `d`
- **THEN** the timer panel shows no timer and the entry is no longer listed

### Requirement: Undo with u
`u` during the grace period SHALL bring the deleted entry back exactly as it was, still running or still paused if it was, with the list cursor on it, and SHALL send nothing to Toggl. With no pending delete, `u` SHALL do nothing.

#### Scenario: Undo
- **WHEN** the user deletes "Yesterday task" and presses `u` two seconds later
- **THEN** the entry is listed again with the cursor on it, and no request has been made

#### Scenario: Undo a running entry
- **WHEN** the user deletes the running entry and presses `u`
- **THEN** the timer panel shows it running again

#### Scenario: Undo after starting another timer
- **WHEN** the user deletes the running entry, starts a new timer, and presses `u`
- **THEN** the deleted entry comes back stopped, as Toggl now has it, and only the new timer runs

### Requirement: The delete is sent after the grace period
When the grace period ends without an undo, the system SHALL send the delete to Toggl with one request. If it fails, the entry SHALL come back as it was and the error SHALL be shown as for any failed request.

#### Scenario: Grace period ends
- **WHEN** the user deletes an entry and presses nothing for 5 seconds
- **THEN** one delete request is sent and the undo hint disappears

#### Scenario: Delete fails
- **WHEN** the delete request fails
- **THEN** the entry is listed again and the error notice is shown

### Requirement: One pending delete at a time
Deleting another entry while a delete is pending SHALL send the pending one at once, and the new delete SHALL become the one `u` undoes, with a fresh grace period.

#### Scenario: Two deletes in a row
- **WHEN** the user deletes "Standup", then deletes "Yesterday task" before 5 seconds pass, then presses `u`
- **THEN** "Standup" was deleted with one request, and "Yesterday task" is back

### Requirement: Refresh keeps a pending delete hidden
A refresh during the grace period SHALL keep the pending entry hidden, even though Toggl still returns it.

#### Scenario: Refresh while pending
- **WHEN** the user deletes an entry and presses `r` within 5 seconds
- **THEN** the refreshed list doesn't show the entry, and `u` still brings it back

### Requirement: Quitting sends a pending delete
Quitting with `q` or `ctrl+c` during the grace period, or while a delete is in flight, SHALL send the pending delete, wait for Toggl's answers, and then quit. While waiting, the status line SHALL show `deleting before quitting… (q again to quit now)`, a second `q` or `ctrl+c` SHALL quit at once, and other keys SHALL do nothing. A delete that failed or went unanswered SHALL be reported after the app exits.

#### Scenario: Quit right after deleting
- **WHEN** the user deletes an entry and presses `q` at once
- **THEN** one delete request is sent and then the app quits

#### Scenario: Delete fails while quitting
- **WHEN** the user quits with a pending delete and the delete request fails
- **THEN** the app quits and then prints that the entry couldn't be deleted and is still in Toggl

### Requirement: The undo hint stays visible
While a delete is pending, the undo hint SHALL stay visible in the status line alongside any other status, until the grace period ends or the delete is undone. When the line doesn't fit the terminal, the description and then the other status SHALL be shortened so that `u undo (<n>s)` stays visible.

#### Scenario: Narrow terminal
- **WHEN** the terminal is 80 columns wide and the user deletes an entry with a long description and then starts a timer
- **THEN** the status line fits in 80 columns and ends with `u undo (5s)`

#### Scenario: Another action during the grace period
- **WHEN** the user deletes an entry and then starts a timer within 5 seconds
- **THEN** the status line shows the start's status and the undo hint together
