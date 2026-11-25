# Spec Delta

## MODIFIED Requirements

### Requirement: Project picker from the project row
`enter` on a project row (start, log, or edit dialog) SHALL open a picker popup with an empty search line and the project list. Typing SHALL edit the search, `up`/`down` and `ctrl+p`/`ctrl+n` SHALL move the selection, `enter` or `tab` SHALL set the project row to the selected option and close the picker, and `esc` SHALL close it with the project row unchanged. After a pick, focus SHALL move to the description row in the start and log dialogs, so the next `enter` starts the timer or logs the entry; in the edit dialog focus SHALL stay on the project row. After `esc`, focus SHALL stay on the project row in every dialog.

#### Scenario: Picking from the row
- **WHEN** the project row shows `(none)`, the user presses `enter`, types `int`, and presses `enter`
- **THEN** the project row shows `Internal`

#### Scenario: Esc leaves the project unchanged
- **WHEN** the project row shows `ACME`, the user opens the picker, moves to `Internal`, and presses `esc`
- **THEN** the project row still shows `ACME`, and focus is still on the project row

#### Scenario: Pick, then start
- **WHEN** in the start dialog the description is `Task`, the user moves to the project row, picks `ACME` from the picker, and presses `enter`
- **THEN** a timer starts for `Task` on ACME

#### Scenario: Pick, then log
- **WHEN** in the log dialog the description is `standup`, the user moves to the project row, picks `Internal` from the picker, and presses `enter`
- **THEN** a finished entry `standup` on Internal is logged with one request and the dialog closes

#### Scenario: Pick in the edit dialog
- **WHEN** in the edit dialog the user picks a project from the project row's picker
- **THEN** focus stays on the project row, and nothing is saved
