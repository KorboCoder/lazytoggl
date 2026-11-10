# entry-editing Specification

## Purpose

Defines how a time entry's description, start, stop, and duration are edited from the
edit dialog, kept consistent with one another, and reviewed before being saved.

## Requirements

### Requirement: Edit form rows
The edit dialog SHALL show the entry as rows: description, start, stop, duration, and project. Start and stop SHALL show their date read-only beside an `HH:MM` time. Duration SHALL show as `HH:MM`. The project row SHALL show the project's name, `(none)`, or `<name> (new)`. For a running entry the dialog SHALL show only description, start, and project as editable; stop SHALL be absent and duration SHALL NOT be editable.

#### Scenario: Finished entry
- **WHEN** the user opens the edit dialog on a finished entry from 14:32 to 15:47 on 2026-11-08 with project ACME
- **THEN** the form shows description, start `Sun 08 Nov 14:32`, stop `Sun 08 Nov 15:47`, duration `01:15`, and project `ACME`

#### Scenario: Running entry
- **WHEN** the user opens the edit dialog on a running entry
- **THEN** the form shows description, start, and project as editable rows, and no stop or editable duration row

### Requirement: Form navigation
`tab`/`down` SHALL move to the next row and `shift+tab`/`up` to the previous one, wrapping at the ends. On the description row, typing SHALL edit the text and `enter` SHALL move to the next row, unless the project dropdown is open. On a start, stop, or duration row, `enter` SHALL open that row's time field. On the project row, `enter` SHALL open the project picker.

#### Scenario: Enter on description moves on
- **WHEN** the description row is focused and the user presses `enter`
- **THEN** the start row is focused and nothing is saved

#### Scenario: Enter on a time row opens its field
- **WHEN** the stop row is focused and the user presses `enter`
- **THEN** an `HH:MM` time field opens holding the stop time, with the cursor on the first digit

#### Scenario: Enter on the project row opens the picker
- **WHEN** the project row is focused and the user presses `enter`
- **THEN** the project picker opens with an empty search

### Requirement: Time field editing
The time field SHALL hold exactly four digits shaped `HH:MM`, with the cursor always on a digit. A digit key SHALL overwrite the digit under the cursor and advance one digit, skipping the colon. `h`/`l` SHALL move one digit, `^`/`$` to the first/last digit (`0` is a digit, so it types rather than moves), and `w`/`b` to the minutes/hours.

#### Scenario: Typing overwrites in place
- **WHEN** the field holds `14:32`, the cursor is on the first minute digit, and the user types `1` then `5`
- **THEN** the field holds `14:15`

#### Scenario: Cursor skips the colon
- **WHEN** the cursor is on the last hour digit and the user presses `l`
- **THEN** the cursor is on the first minute digit

#### Scenario: Letters do not change the value
- **WHEN** the user presses a letter that is not a motion
- **THEN** the field's value and cursor are unchanged

### Requirement: Date segment in start and stop fields
The start and stop fields SHALL show the time's date (`Sun 08 Nov`) beside its `HH:MM` time, with the time segment active when the field opens. `tab` and `shift+tab` SHALL switch between the date and time segments. On the date segment, `ctrl+a`/`ctrl+x` SHALL add/subtract one day, and digits and motions SHALL do nothing. The duration field SHALL have no date segment, and `tab` there SHALL do nothing.

#### Scenario: Field opens on the time with the date visible
- **WHEN** the user opens the start field of an entry starting Sat 07 Nov 14:32
- **THEN** the field shows `Sat 07 Nov` and `14:32`, with the cursor on the first hour digit

#### Scenario: Tab switches segments
- **WHEN** the start field is open on the time segment and the user presses `tab`
- **THEN** the date segment is active, and `tab` again returns to the time segment with its cursor where it was

#### Scenario: Stepping the date
- **WHEN** the date segment shows `Sat 07 Nov` and the user presses `ctrl+x`
- **THEN** the date segment shows `Fri 06 Nov` and the time is unchanged

#### Scenario: Digits do nothing on the date segment
- **WHEN** the date segment is active and the user types `5`
- **THEN** the field's date and time are unchanged

#### Scenario: Duration has no date
- **WHEN** the duration field is open and the user presses `tab`
- **THEN** the field is unchanged

### Requirement: Date bounds
No start or stop date SHALL go past today. A running entry's start date SHALL NOT go earlier than 7 days before today. A finished entry's dates have no lower bound. A step that would cross a bound SHALL leave the field unchanged.

#### Scenario: Not past today
- **WHEN** today is Sun 08 Nov, the date segment shows `Sun 08 Nov`, and the user presses `ctrl+a`
- **THEN** the date segment still shows `Sun 08 Nov`

#### Scenario: Running entry stays within a week
- **WHEN** today is Sun 08 Nov and a running entry's start field shows `Sun 01 Nov` on the date segment and the user presses `ctrl+x`
- **THEN** the date segment still shows `Sun 01 Nov`

#### Scenario: Finished entry can go further back
- **WHEN** today is Sun 08 Nov and a finished entry's start field shows `Sun 01 Nov` on the date segment and the user presses `ctrl+x`
- **THEN** the date segment shows `Sat 31 Oct`

### Requirement: Accepting and reverting a time field
`enter` SHALL accept the field when its hours are 00–23 and minutes 00–59, closing it and applying the value to the form. Out-of-range values SHALL show an error and keep the field open. `esc` SHALL close the field and leave the form's value as it was before the field opened.

#### Scenario: Out-of-range time is rejected
- **WHEN** the user types `27:10` and presses `enter`
- **THEN** an error states the valid range and the field stays open

#### Scenario: Esc reverts the field
- **WHEN** the user changes the start field to `09:00` and presses `esc`
- **THEN** the form's start is unchanged

### Requirement: Increment and decrement
`ctrl+a`/`ctrl+x` SHALL add/subtract one unit of the part under the cursor: one hour on the hours, one minute on the minutes. Minutes SHALL carry into and borrow from hours. On start and stop, hours SHALL carry into and borrow from the date, within the date bounds; duration SHALL stay within 00:01–23:59.

#### Scenario: Minutes carry into hours
- **WHEN** the field holds `14:59`, the cursor is on the minutes, and the user presses `ctrl+a`
- **THEN** the field holds `15:00`

#### Scenario: Minutes borrow from hours
- **WHEN** the field holds `15:00`, the cursor is on the minutes, and the user presses `ctrl+x`
- **THEN** the field holds `14:59`

#### Scenario: Crossing midnight forward
- **WHEN** a stop field holds `Sat 07 Nov 23:59`, the cursor is on the minutes, and the user presses `ctrl+a`
- **THEN** the field holds `Sun 08 Nov 00:00`

#### Scenario: Crossing midnight backward
- **WHEN** a start field holds `Sun 08 Nov 00:00`, the cursor is on the minutes, and the user presses `ctrl+x`
- **THEN** the field holds `Sat 07 Nov 23:59`

#### Scenario: Day boundary holds
- **WHEN** today is Sun 08 Nov, a start field holds `Sun 08 Nov 23:59`, the cursor is on the minutes, and the user presses `ctrl+a`
- **THEN** the field still holds `Sun 08 Nov 23:59`

#### Scenario: Duration bound holds
- **WHEN** a duration field holds `23:59`, the cursor is on the minutes, and the user presses `ctrl+a`
- **THEN** the field still holds `23:59`

### Requirement: Start, stop, and duration stay consistent
Accepting a start SHALL keep stop and recompute duration; when start's date was stepped on the date segment by some number of days, stop SHALL first move by the same number of days, so a date-only change keeps the duration. A start that crosses midnight by a time edit (carry or a typed time) SHALL keep stop. Accepting a stop SHALL keep start and recompute duration. Accepting a duration SHALL keep start and set stop to start plus duration.

#### Scenario: Moving start keeps stop
- **WHEN** an entry runs 14:32–15:47 and the user sets start to 14:00
- **THEN** stop stays 15:47 and duration becomes `01:47`

#### Scenario: Editing duration moves stop
- **WHEN** an entry starts at 14:32 and the user sets duration to `00:45`
- **THEN** start stays 14:32 and stop becomes 15:17

#### Scenario: Moving the start date moves the entry
- **WHEN** an entry runs Sat 07 Nov 14:32–15:47 and the user steps start's date to Fri 06 Nov and accepts
- **THEN** start is `Fri 06 Nov 14:32`, stop is `Fri 06 Nov 15:47`, and duration stays `01:15`

#### Scenario: Start date and time together
- **WHEN** an entry runs Sat 07 Nov 14:32–15:47 and the user sets start to `Fri 06 Nov 14:00`
- **THEN** stop is `Fri 06 Nov 15:47` and duration is `01:47`

#### Scenario: Crossing midnight by time keeps stop
- **WHEN** an entry runs Sun 08 Nov 00:00–01:00 and in the start field the user presses `ctrl+x` on the minutes and accepts
- **THEN** start is `Sat 07 Nov 23:59`, stop stays `Sun 08 Nov 01:00`, and duration is `01:01`

#### Scenario: Moving the stop date moves only stop
- **WHEN** an entry runs Sat 07 Nov 20:00–21:00 and the user steps stop's date to Sun 08 Nov and accepts
- **THEN** start stays `Sat 07 Nov 20:00`, stop is `Sun 08 Nov 21:00`, and duration is `25:00`

### Requirement: Minimum entry length
A saved entry SHALL be at least 1 minute long. A start at or after stop SHALL become stop minus 1 minute. A stop at or before start SHALL become start plus 1 minute. A running entry's start SHALL NOT be later than 1 minute before now.

#### Scenario: Start past stop falls back
- **WHEN** an entry runs 14:32–15:47 and the user sets start to 16:00
- **THEN** start becomes 15:46 and duration `00:01`

#### Scenario: Stop before start falls back
- **WHEN** an entry runs 14:32–15:47 and the user sets stop to 14:00
- **THEN** stop becomes 14:33 and duration `00:01`

#### Scenario: Running start in the future falls back
- **WHEN** the time is 10:00, a running entry started at 09:30, and the user sets start to 11:00
- **THEN** start becomes 09:59

### Requirement: Typed times resolve across midnight
When the user types a time in a start or stop field whose date they have not changed, the time SHALL land on the day that makes the entry 1 minute to 12 hours long: a stop on the first occurrence after start, a start on the last occurrence before stop (before now, for a running entry). If no such day exists within the date bounds, the time SHALL stay on the field's date. The field's date SHALL update as the user types.

#### Scenario: Overnight stop
- **WHEN** an entry runs Sat 07 Nov 23:30–23:50 and the user types `00:15` in the stop field
- **THEN** the field shows `Sun 08 Nov 00:15` before it is accepted, and after accepting the duration is `00:45`

#### Scenario: Overnight start
- **WHEN** an entry runs Sun 08 Nov 00:15–01:00 and the user types `23:30` in the start field
- **THEN** start becomes `Sat 07 Nov 23:30`, stop stays `Sun 08 Nov 01:00`, and duration is `01:30`

#### Scenario: Back to one day
- **WHEN** an entry runs Sat 07 Nov 23:30 to Sun 08 Nov 00:45 and the user types `23:50` in the stop field
- **THEN** stop becomes `Sat 07 Nov 23:50` and duration is `00:20`

#### Scenario: Running entry started before midnight
- **WHEN** the time is Sun 08 Nov 00:10, a running entry started at 00:05, and the user types `23:50` in the start field
- **THEN** start becomes `Sat 07 Nov 23:50`

#### Scenario: More than 12 hours stays on its day
- **WHEN** an entry runs Sun 08 Nov 14:32–15:47 and the user types `14:00` in the stop field
- **THEN** stop stays on `Sun 08 Nov` and the minimum-length rule makes it 14:33

#### Scenario: A date changed by hand is kept
- **WHEN** an entry runs Sat 07 Nov 20:00–21:00, and in the stop field the user steps the date to `Sun 08 Nov`, returns to the time segment, and types `21:30`
- **THEN** stop becomes `Sun 08 Nov 21:30`

### Requirement: Untouched times are preserved
A start or stop the user did not change SHALL be saved with its original timestamp, seconds included. A time moved only by stepping its date (a start whose date alone was stepped, the stop that moves with it, or a stop whose date alone was stepped) SHALL keep its seconds. Any other changed time SHALL have zero seconds, except a stop recomputed from a duration (start plus that duration) and a minimum-length fallback (exactly 1 minute from the other end), which keep the other end's seconds.

#### Scenario: Editing only the description
- **WHEN** an entry started at 14:32:17 and the user changes only its description and saves
- **THEN** the saved start is 14:32:17

#### Scenario: Moving the date keeps seconds
- **WHEN** an entry runs Sat 07 Nov 14:32:17–15:47:05 and the user changes only start's date to Fri 06 Nov and saves
- **THEN** the saved start is Fri 06 Nov 14:32:17 and the saved stop is Fri 06 Nov 15:47:05

### Requirement: Review before saving
`esc` on the form SHALL close the dialog with no API request when nothing changed; while the project dropdown is open, `esc` SHALL only close the dropdown. Otherwise it SHALL show a review listing each changed field as before → after. In the review, `y`/`enter` SHALL save with one update request, preceded by one request to create the project when it is marked `(new)`; `esc` SHALL return to the form with edits intact, and `n` SHALL discard all edits and close.

#### Scenario: Nothing changed
- **WHEN** the user opens the edit dialog and presses `esc` without changing anything
- **THEN** the dialog closes and no API request is made

#### Scenario: Review lists only changed fields
- **WHEN** the user changes only start from 14:32 to 14:15 and presses `esc`
- **THEN** the review shows start `14:32 -> 14:15` and duration `01:15 -> 01:32`, and no description or stop line

#### Scenario: Save
- **WHEN** the review is showing and the user presses `y`
- **THEN** the entry is saved with one update request and the dialog closes

#### Scenario: Save with a new project
- **WHEN** the review shows `(none) -> newclient (new)` and the user presses `y`
- **THEN** the project is created with one request, then the entry is saved with one update request, and the dialog closes

#### Scenario: Esc with the dropdown open
- **WHEN** the description's project dropdown is open and the user presses `esc`
- **THEN** only the dropdown closes, and no review is shown

#### Scenario: Back keeps edits
- **WHEN** the review is showing and the user presses `esc`
- **THEN** the form returns with the edited values intact

#### Scenario: Discard
- **WHEN** the review is showing and the user presses `n`
- **THEN** the dialog closes, the entry is unchanged, and no API request is made

### Requirement: Review shows date changes
In the review, a start or stop line SHALL show its date with the time when that time's date changed, and only the time otherwise.

#### Scenario: Entry moved to another day
- **WHEN** the user moves an entry from Sat 07 Nov 14:32–15:47 to Fri 06 Nov and presses `esc`
- **THEN** the review shows start `Sat 07 Nov 14:32 -> Fri 06 Nov 14:32` and stop `Sat 07 Nov 15:47 -> Fri 06 Nov 15:47`, and no duration line

### Requirement: Review shows project changes
When the project changed, the review SHALL list it as `project  <before> -> <after>`, using `(none)` for no project and `<name> (new)` for a project that saving will create.

#### Scenario: Project changed
- **WHEN** the user changes an entry's project from ACME to Internal and presses `esc`
- **THEN** the review shows `project` with `ACME -> Internal`

#### Scenario: Project to be created
- **WHEN** the user picks `+ Create "newclient"` for an entry with no project and presses `esc`
- **THEN** the review shows `project` with `(none) -> newclient (new)`

#### Scenario: Project removed
- **WHEN** the user picks `No project` for an entry whose project is ACME and presses `esc`
- **THEN** the review shows `project` with `ACME -> (none)`

### Requirement: Review flags moves outside the list
When the edited start is outside the list's fetch window (earlier than its start, or later than now), the review SHALL show `(outside list)` on the line below the start change.

#### Scenario: Moved before the window
- **WHEN** today is Sun 08 Nov and the user moves a finished entry's start to Mon 12 Oct and presses `esc`
- **THEN** the review shows `(outside list)` below the start change

#### Scenario: Moved after now
- **WHEN** it is Sun 08 Nov 12:00 and the user steps a finished entry's start from Sat 07 Nov 15:00 to Sun 08 Nov and presses `esc`
- **THEN** the review shows `(outside list)` below the start change

#### Scenario: Within the window
- **WHEN** today is Sun 08 Nov and the user moves a finished entry's start to Fri 06 Nov and presses `esc`
- **THEN** the review shows no `(outside list)` flag

### Requirement: Saved entry placement
After an edit is saved, the entries list SHALL show the entry under the day of its saved start, in start order, with the list cursor on that entry.

#### Scenario: Moved to another day
- **WHEN** the user moves an entry from today to Fri 06 Nov and saves
- **THEN** the entry is listed once, under Fri 06 Nov, today's group no longer contains it, and the cursor is on it

#### Scenario: Reordered within a day
- **WHEN** today has entries at 09:00 and 11:00 and the user moves the 09:00 entry's start to 12:00 and saves
- **THEN** the moved entry is listed above the 11:00 entry, and the cursor is on it

### Requirement: Saving outside the list
When an edit moves an entry's start outside the list's fetch window (earlier than its start, or later than now), the status line SHALL say the entry is now outside the list and will drop off on the next refresh. The entry SHALL stay in the list until the next refresh. An edit that leaves the start unchanged SHALL NOT show this warning.

#### Scenario: Status after moving out of the window
- **WHEN** today is Sun 08 Nov and the user moves a finished entry to Mon 12 Oct and saves
- **THEN** the status line says it moved to Mon 12 Oct, outside the list, and will drop off on refresh, and the entry is still listed under Mon 12 Oct

#### Scenario: Status after moving past now
- **WHEN** it is Sun 08 Nov 12:00 and the user moves a finished entry's start to Sun 08 Nov 15:00 and saves
- **THEN** the status line says it starts after now, outside the list, and will drop off on refresh

#### Scenario: Start unchanged
- **WHEN** an entry's start was inside the window when the dialog opened but the window has since moved past it, and the user edits only its description and saves
- **THEN** the status line says only that the entry was saved
