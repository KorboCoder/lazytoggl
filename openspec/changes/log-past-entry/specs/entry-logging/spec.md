# Spec Delta

## Purpose

Defines how a finished time entry is created for time that already passed: the
log-entry dialog, its Toggl-like defaults, how it is logged or cancelled, and where the
new entry lands.

## ADDED Requirements

### Requirement: Opening the log-entry dialog
`m` SHALL open a log-entry dialog from the entries list or the timer panel. The dialog SHALL show the rows description, start, stop, duration, and project, with the description focused and empty and the project `(none)`.

#### Scenario: Open from the list
- **WHEN** the entries list is focused and the user presses `m`
- **THEN** a `Log entry` dialog opens with description, start, stop, duration, and project rows, the description focused and empty

#### Scenario: Open from the timer panel
- **WHEN** the timer panel is focused and the user presses `m`
- **THEN** the `Log entry` dialog opens

### Requirement: Default times
The dialog SHALL open with stop set to the current time rounded down to the minute and start set to 30 minutes before stop.

#### Scenario: Defaults
- **WHEN** the time is Tue 10 Nov 11:40:27 and the user presses `m`
- **THEN** start is `Tue 10 Nov 11:10`, stop is `Tue 10 Nov 11:40`, and duration is `00:30`

#### Scenario: Shortly after midnight
- **WHEN** the time is Tue 10 Nov 00:10 and the user presses `m`
- **THEN** start is `Mon 09 Nov 23:40` and stop is `Tue 10 Nov 00:10`

### Requirement: Editing the new entry
The dialog's rows SHALL behave as the edit dialog's rows do for a finished entry: the same row navigation, date+time and duration fields, start/stop/duration consistency, 1-minute minimum, midnight resolution of typed times, date bounds, and project picker and `@` dropdown.

#### Scenario: Changing the duration
- **WHEN** the dialog shows start 11:10 and the user sets duration to `01:00`
- **THEN** start stays 11:10 and stop becomes 12:10

#### Scenario: Picking a project with @
- **WHEN** the user types `standup @int` in the description and presses `enter` with the dropdown on `Internal`
- **THEN** the description is `standup`, the project row shows `Internal`, and nothing is logged

### Requirement: Logging with enter
`enter` on the description row, while no project dropdown is open, SHALL create the finished entry in Toggl with one request (preceded by one to create a project marked `(new)`), close the dialog, and add the entry to the list. An empty description SHALL be allowed. On any other row, `enter` SHALL do what it does in the edit dialog.

#### Scenario: Log the defaults
- **WHEN** the dialog shows 11:10–11:40, the user types `standup`, and presses `enter`
- **THEN** a finished entry `standup` from 11:10 to 11:40 is created with one request and the dialog closes

#### Scenario: Log with a new project
- **WHEN** the project row shows `newclient (new)` and the user presses `enter` on the description
- **THEN** the project is created with one request, then the entry is created on it with one request

#### Scenario: Enter on a time row
- **WHEN** the stop row is focused and the user presses `enter`
- **THEN** the stop field opens and nothing is logged

### Requirement: Cancelling
`esc` on the form SHALL close the dialog without any request when nothing differs from the defaults the dialog opened with. Otherwise it SHALL ask `Discard this entry?`, where `y` SHALL discard and close, and `n` or `esc` SHALL return to the form with its values intact. While the `@` dropdown is open, `esc` SHALL only close the dropdown.

#### Scenario: Nothing changed
- **WHEN** the user opens the dialog and presses `esc`
- **THEN** the dialog closes and no request is made

#### Scenario: Discard after typing
- **WHEN** the user types `standup` and presses `esc`, then `y`
- **THEN** the dialog closes and no request is made

#### Scenario: Keep editing
- **WHEN** the user types `standup`, presses `esc`, then `n`
- **THEN** the form returns with `standup` still in the description

### Requirement: The logged entry in the list
A logged entry SHALL be listed under the day of its start, in start order, with the list cursor on it. Logging SHALL NOT stop or change a running entry. When the entry's start is outside the list's fetch window (earlier than its start, or later than now), the status line SHALL say it is outside the list and will drop off on the next refresh.

#### Scenario: Logged while a timer runs
- **WHEN** a timer has been running since 11:20 and the user logs 10:00–10:45
- **THEN** the timer is still running, and the new entry is listed under today with the cursor on it

#### Scenario: Logged before the window
- **WHEN** today is Tue 10 Nov and the user logs an entry starting Mon 26 Oct
- **THEN** the status line says it is outside the list and will drop off on refresh, and the entry is listed under Mon 26 Oct until then
