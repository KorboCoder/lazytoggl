# entry-editing Specification

## Purpose

Defines how a time entry's description, start, stop, and duration are edited from the
edit dialog, kept consistent with one another, and reviewed before being saved.

## Requirements

### Requirement: Edit form rows
The edit dialog SHALL show the entry as rows: description, start, stop, and duration. Start and stop SHALL show their date read-only beside an `HH:MM` time. Duration SHALL show as `HH:MM`. For a running entry the dialog SHALL show only description and start as editable; stop SHALL be absent and duration SHALL NOT be editable.

#### Scenario: Finished entry
- **WHEN** the user opens the edit dialog on a finished entry from 14:32 to 15:47 on 2026-11-08
- **THEN** the form shows description, start `Sun 08 Nov 14:32`, stop `Sun 08 Nov 15:47`, and duration `01:15`

#### Scenario: Running entry
- **WHEN** the user opens the edit dialog on a running entry
- **THEN** the form shows description and start as editable rows, and no stop or editable duration row

### Requirement: Form navigation
`tab`/`down` SHALL move to the next row and `shift+tab`/`up` to the previous one, wrapping at the ends. On the description row, typing SHALL edit the text and `enter` SHALL move to the next row. On a start, stop, or duration row, `enter` SHALL open that row's time field.

#### Scenario: Enter on description moves on
- **WHEN** the description row is focused and the user presses `enter`
- **THEN** the start row is focused and nothing is saved

#### Scenario: Enter on a time row opens its field
- **WHEN** the stop row is focused and the user presses `enter`
- **THEN** an `HH:MM` time field opens holding the stop time, with the cursor on the first digit

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

### Requirement: Accepting and reverting a time field
`enter` SHALL accept the field when its hours are 00–23 and minutes 00–59, closing it and applying the value to the form. Out-of-range values SHALL show an error and keep the field open. `esc` SHALL close the field and leave the form's value as it was before the field opened.

#### Scenario: Out-of-range time is rejected
- **WHEN** the user types `27:10` and presses `enter`
- **THEN** an error states the valid range and the field stays open

#### Scenario: Esc reverts the field
- **WHEN** the user changes the start field to `09:00` and presses `esc`
- **THEN** the form's start is unchanged

### Requirement: Increment and decrement
`ctrl+a`/`ctrl+x` SHALL add/subtract one unit of the part under the cursor: one hour on the hours, one minute on the minutes. Minutes SHALL carry into and borrow from hours. Start and stop SHALL stay within 00:00–23:59; duration within 00:01–23:59.

#### Scenario: Minutes carry into hours
- **WHEN** the field holds `14:59`, the cursor is on the minutes, and the user presses `ctrl+a`
- **THEN** the field holds `15:00`

#### Scenario: Minutes borrow from hours
- **WHEN** the field holds `15:00`, the cursor is on the minutes, and the user presses `ctrl+x`
- **THEN** the field holds `14:59`

#### Scenario: Day boundary holds
- **WHEN** a start field holds `23:59`, the cursor is on the minutes, and the user presses `ctrl+a`
- **THEN** the field still holds `23:59`

### Requirement: Start, stop, and duration stay consistent
Accepting a start SHALL keep stop and recompute duration. Accepting a stop SHALL keep start and recompute duration. Accepting a duration SHALL keep start and set stop to start plus duration. Each time keeps its own date.

#### Scenario: Moving start keeps stop
- **WHEN** an entry runs 14:32–15:47 and the user sets start to 14:00
- **THEN** stop stays 15:47 and duration becomes `01:47`

#### Scenario: Editing duration moves stop
- **WHEN** an entry starts at 14:32 and the user sets duration to `00:45`
- **THEN** start stays 14:32 and stop becomes 15:17

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

### Requirement: Untouched times are preserved
A start or stop the user did not change SHALL be saved with its original timestamp, seconds included. A changed time SHALL have zero seconds, except a stop recomputed from a duration (start plus that duration) and a minimum-length fallback (exactly 1 minute from the other end), which keep the other end's seconds.

#### Scenario: Editing only the description
- **WHEN** an entry started at 14:32:17 and the user changes only its description and saves
- **THEN** the saved start is 14:32:17

### Requirement: Review before saving
`esc` on the form SHALL close the dialog with no API request when nothing changed. Otherwise it SHALL show a review listing each changed field as before → after. In the review, `y`/`enter` SHALL save with one update request, `esc` SHALL return to the form with edits intact, and `n` SHALL discard all edits and close.

#### Scenario: Nothing changed
- **WHEN** the user opens the edit dialog and presses `esc` without changing anything
- **THEN** the dialog closes and no API request is made

#### Scenario: Review lists only changed fields
- **WHEN** the user changes only start from 14:32 to 14:15 and presses `esc`
- **THEN** the review shows start `14:32 -> 14:15` and duration `01:15 -> 01:32`, and no description or stop line

#### Scenario: Save
- **WHEN** the review is showing and the user presses `y`
- **THEN** the entry is saved with one update request and the dialog closes

#### Scenario: Back keeps edits
- **WHEN** the review is showing and the user presses `esc`
- **THEN** the form returns with the edited values intact

#### Scenario: Discard
- **WHEN** the review is showing and the user presses `n`
- **THEN** the dialog closes, the entry is unchanged, and no API request is made
