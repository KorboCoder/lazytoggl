# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Default times
When opened with `m`, the dialog SHALL open with stop set to the current time rounded down to the minute and start set to 30 minutes before stop.

#### Scenario: Defaults
- **WHEN** the time is Tue 10 Nov 11:40:27 and the user presses `m`
- **THEN** start is `Tue 10 Nov 11:10`, stop is `Tue 10 Nov 11:40`, and duration is `00:30`

#### Scenario: Shortly after midnight
- **WHEN** the time is Tue 10 Nov 00:10 and the user presses `m`
- **THEN** start is `Mon 09 Nov 23:40` and stop is `Tue 10 Nov 00:10`
