# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

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
