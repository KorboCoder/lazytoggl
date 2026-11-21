# entry-logging Specification

## Purpose

Defines how a finished time entry is created for time that already passed: the
log-entry dialog, its Toggl-like defaults, how it is logged or cancelled, and where the
new entry lands.

## Requirements

### Requirement: Opening the log-entry dialog
`m` SHALL open a log-entry dialog from the entries list or the timer panel. The dialog SHALL show the rows description, start, stop, duration, and project, with the description focused and empty and the project `(none)`.

#### Scenario: Open from the list
- **WHEN** the entries list is focused and the user presses `m`
- **THEN** a `Log entry` dialog opens with description, start, stop, duration, and project rows, the description focused and empty

#### Scenario: Open from the timer panel
- **WHEN** the timer panel is focused and the user presses `m`
- **THEN** the `Log entry` dialog opens

### Requirement: Default times
When opened with `m`, the dialog SHALL open with stop set to the current time rounded down to the minute and start set to 30 minutes before stop.

#### Scenario: Defaults
- **WHEN** the time is Tue 10 Nov 11:40:27 and the user presses `m`
- **THEN** start is `Tue 10 Nov 11:10`, stop is `Tue 10 Nov 11:40`, and duration is `00:30`

#### Scenario: Shortly after midnight
- **WHEN** the time is Tue 10 Nov 00:10 and the user presses `m`
- **THEN** start is `Mon 09 Nov 23:40` and stop is `Tue 10 Nov 00:10`

### Requirement: Logging into the gap below with o
`o` on a selected entry in the entries list SHALL open the log-entry dialog with stop at the selected entry's start and start at the latest stop among the other entries on the same day that started before it. With no such entry, start SHALL be 30 minutes before the selected entry's start. Start and stop SHALL keep their exact times, seconds included.

#### Scenario: Gap between two entries
- **WHEN** today has Standup 08:30–09:00 and Code 10:00–10:30, and the user selects Code and presses `o`
- **THEN** the log dialog opens on 09:00–10:00

#### Scenario: First entry of the day
- **WHEN** Standup 08:30–09:00 is the earliest entry of its day and the user selects it and presses `o`
- **THEN** the log dialog opens on 08:00–08:30

#### Scenario: Exact seconds
- **WHEN** the previous entry ended at 09:00:17 and the selected one starts at 10:00:05
- **THEN** the dialog's start and stop are 09:00:17 and 10:00:05, and logging it leaves no gap or overlap

### Requirement: Logging into the gap above with O
`O` on a selected finished entry SHALL open the log-entry dialog with start at the selected entry's stop and stop at the earliest start among the other entries on the same day that start after it, the running entry included. With no such entry, stop SHALL be now rounded down to the minute when the day is today, and 30 minutes after the selected entry's stop otherwise.

#### Scenario: Gap between two entries
- **WHEN** today has Code 10:00–10:30 and Review 11:00–12:00, and the user selects Code and presses `O`
- **THEN** the log dialog opens on 10:30–11:00

#### Scenario: Up to the running timer
- **WHEN** Review ended at 11:00 and a timer has been running since 11:40
- **THEN** `O` on Review opens the log dialog on 11:00–11:40

#### Scenario: Newest entry today
- **WHEN** it is 12:40:27, no timer is running, and the user presses `O` on the entry that ended at 12:00
- **THEN** the log dialog opens on 12:00–12:40

#### Scenario: Newest entry of a past day
- **WHEN** the user presses `O` on yesterday's last entry, which ended at 18:00
- **THEN** the log dialog opens on yesterday 18:00–18:30

### Requirement: A gap never overlaps an entry
The gap `o` or `O` opens SHALL never overlap another entry on any day, the running entry included (it runs until now). The 30-minute fallbacks SHALL stop short of entries on neighbouring days. When the gap would still overlap an entry, such as one that contains the selected entry, the dialog SHALL NOT open, as for no gap.

#### Scenario: Fallback across midnight
- **WHEN** the selected entry starts today at 00:10, it is today's first, and yesterday's last entry ended at 23:55
- **THEN** `o` opens the log dialog on yesterday 23:55 to today 00:10

#### Scenario: Past day's last entry before a timer started after midnight
- **WHEN** yesterday's last entry ended at 23:50 and a timer has been running since 00:05 today
- **THEN** `O` on that entry opens the log dialog on 23:50–00:05

#### Scenario: Inside a longer entry
- **WHEN** the selected entry 10:00–10:30 lies inside an entry that ran 09:00–11:00
- **THEN** `O` opens no dialog and says there is no gap above

### Requirement: No gap to fill
When the gap `o` or `O` would open is shorter than one minute (the entries touch or overlap), the dialog SHALL NOT open, and the status line SHALL say `no gap below "<description>"` or `no gap above "<description>"`.

#### Scenario: Touching entries
- **WHEN** Standup ends at 09:00 and Code starts at 09:00, and the user presses `o` on Code
- **THEN** no dialog opens and the status says `no gap below "Code"`

#### Scenario: Overlap
- **WHEN** an earlier entry ran 09:30–10:15 and the selected entry starts at 10:00
- **THEN** `o` opens no dialog and says there is no gap below

### Requirement: o and O on the running entry
`o` on the running entry SHALL open the log dialog on the gap before it, as for any entry. `O` on the running entry SHALL NOT open a dialog, and the status line SHALL say `nothing after the running entry`.

#### Scenario: Before the running timer
- **WHEN** a timer has been running since 11:50 and the previous entry ended at 10:15
- **THEN** `o` on the running entry opens the log dialog on 10:15–11:50

#### Scenario: After the running timer
- **WHEN** the user presses `O` on the running entry
- **THEN** no dialog opens and the status says `nothing after the running entry`

### Requirement: Where o and O work
`o` and `O` SHALL act on the selected entry of the entries list only. With the timer panel focused they SHALL do nothing.

#### Scenario: Timer panel
- **WHEN** the timer panel is focused and the user presses `o`
- **THEN** no dialog opens

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
`esc` on the form SHALL close the dialog without any request when nothing differs from the defaults the dialog opened with. Otherwise it SHALL ask `Discard this entry?`, where `d` SHALL discard and close, `esc` SHALL return to the form with its values intact, and every other key SHALL do nothing. While the `@` dropdown is open, `esc` SHALL only close the dropdown.

#### Scenario: Nothing changed
- **WHEN** the user opens the dialog and presses `esc`
- **THEN** the dialog closes and no request is made

#### Scenario: Discard after typing
- **WHEN** the user types `standup` and presses `esc`, then `d`
- **THEN** the dialog closes and no request is made

#### Scenario: y does not discard
- **WHEN** the discard prompt is showing and the user presses `y` or `n`
- **THEN** the prompt stays and nothing is discarded or logged

#### Scenario: Esc with the dropdown open
- **WHEN** the `@` dropdown is open and the user presses `esc`
- **THEN** only the dropdown closes, and a second `esc` shows the discard prompt

#### Scenario: Keep editing
- **WHEN** the user types `standup`, presses `esc`, then `esc` again
- **THEN** the form returns with `standup` still in the description

### Requirement: Cancel keys don't clash with the edit review
Because `y` saves in the edit dialog's review, `y` and `n` SHALL do nothing on the discard prompt, and the form's help line SHALL describe `esc` as cancelling on every row, never as "done".

#### Scenario: Help on a time row
- **WHEN** the log dialog's start row is focused
- **THEN** the help line says `esc cancel`

### Requirement: The logged entry in the list
A logged entry SHALL be listed under the day of its start, in start order, with the list cursor on it. Logging SHALL NOT stop or change a running entry. When the entry's start is outside the list's fetch window (earlier than its start, or later than now), the status line SHALL say it is outside the list and will drop off on the next refresh.

#### Scenario: Logged while a timer runs
- **WHEN** a timer has been running since 11:20 and the user logs 10:00–10:45
- **THEN** the timer is still running, and the new entry is listed under today with the cursor on it

#### Scenario: Logged before the window
- **WHEN** today is Tue 10 Nov and the user logs an entry starting Mon 26 Oct
- **THEN** the status line says it is outside the list and will drop off on refresh, and the entry is listed under Mon 26 Oct until then
