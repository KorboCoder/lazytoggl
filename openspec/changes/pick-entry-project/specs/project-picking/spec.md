# Spec Delta

## Purpose

Defines how an entry's project is chosen, cleared, or created from the start and edit
dialogs, through a project row or by typing `@` in the description, and how a new
project is created when the entry is saved.

## ADDED Requirements

### Requirement: Project row in the start dialog
The start dialog SHALL show two rows: description, then project. `tab`/`down` and `shift+tab`/`up` SHALL move between them. The project row SHALL show the chosen project's name, or `(none)`. Resuming an entry SHALL pre-fill its project. `enter` on the description row SHALL start the timer while no project dropdown is open.

#### Scenario: Resume shows the project
- **WHEN** the user resumes an entry whose project is ACME
- **THEN** the start dialog's project row shows `ACME`

#### Scenario: Enter on the description starts
- **WHEN** the start dialog's description holds `fix login bug`, no dropdown is open, and the user presses `enter`
- **THEN** a timer starts with that description and the dialog's project

### Requirement: Project picker from the project row
`enter` on a project row (start or edit dialog) SHALL open a picker popup with an empty search line and the project list. Typing SHALL edit the search, `up`/`down` and `ctrl+p`/`ctrl+n` SHALL move the selection, `enter` or `tab` SHALL set the project row to the selected option and close the picker, and `esc` SHALL close it with the project row unchanged.

#### Scenario: Picking from the row
- **WHEN** the project row shows `(none)`, the user presses `enter`, types `int`, and presses `enter`
- **THEN** the project row shows `Internal`

#### Scenario: Esc leaves the project unchanged
- **WHEN** the project row shows `ACME`, the user opens the picker, moves to `Internal`, and presses `esc`
- **THEN** the project row still shows `ACME`

### Requirement: Project list order
With an empty search, the list SHALL start with "No project" when the entry currently has a project, followed by all active projects sorted A–Z ignoring case. Archived projects SHALL NOT be listed, but SHALL still name the entries that use them.

#### Scenario: Entry with a project
- **WHEN** the active projects are `Internal`, `ACME`, and `acme website`, an archived project is `Old Client`, and the picker opens on an entry whose project is ACME
- **THEN** the list reads `No project`, `ACME`, `acme website`, `Internal`

#### Scenario: Entry without a project
- **WHEN** the picker opens on an entry with no project
- **THEN** the list has no `No project` option

### Requirement: Filtering by search
A non-empty search SHALL keep only the active projects whose name contains the search, ignoring case, in the same A–Z order. Spaces SHALL be part of the search. "No project" SHALL be shown only while the search is empty.

#### Scenario: Substring with a space
- **WHEN** the active projects are `ACME`, `Acme Internal`, and `Internal` and the search is `acme int`
- **THEN** only `Acme Internal` is listed, followed by `+ Create "acme int"`

#### Scenario: Middle of a name
- **WHEN** the active projects include `Acme Website` and the search is `web`
- **THEN** `Acme Website` is listed

### Requirement: Create option
While the search is non-empty and no project name, active or archived, equals the search ignoring case and surrounding spaces, the list SHALL end with a `+ Create "<search>"` option. When no project matches, it SHALL be the only option. Picking it SHALL set the project row to `<search> (new)`, with surrounding spaces trimmed. Nothing SHALL be sent to Toggl when it is picked.

#### Scenario: Nothing matches
- **WHEN** the search is `newclient` and no project name contains it
- **THEN** the only option is `+ Create "newclient"`

#### Scenario: Partial matches plus create
- **WHEN** the search is `acme` and the projects are `ACME Website` and `Acme Internal`
- **THEN** both are listed, followed by `+ Create "acme"`

#### Scenario: Exact match offers no create
- **WHEN** the search is `acme` and a project is named `ACME`
- **THEN** no create option is listed

#### Scenario: Archived name offers no create
- **WHEN** the search is `old client` and an archived project is named `Old Client`
- **THEN** no create option is listed and the archived project is not listed either

### Requirement: Inline @ opens a dropdown
Typing `@` in a description, at the very beginning or right after a space, SHALL open a project dropdown below the description. The text after the `@`, up to the cursor, SHALL be the search. Typing `@` anywhere else SHALL insert it as text with no dropdown.

#### Scenario: At a word start
- **WHEN** the description is `fix login bug ` and the user types `@`
- **THEN** a dropdown opens listing the project list for an empty search

#### Scenario: Inside a word
- **WHEN** the description is `mail me` and the user types `@`
- **THEN** the description is `mail me@` and no dropdown opens

### Requirement: Dropdown keys
While the dropdown is open, typed characters and `backspace` SHALL edit the description and refilter. `up`/`down` and `ctrl+p`/`ctrl+n` SHALL move the selection. `enter` or `tab` SHALL pick the selected option, remove the `@` and search from the description (collapsing the spaces around it), and close the dropdown. `esc` SHALL close only the dropdown, keeping the text. Deleting the `@`, or moving the cursor before it, SHALL close the dropdown.

#### Scenario: Pick removes the search text
- **WHEN** the description is `fix login bug @acme int` with the dropdown on `Acme Internal`, and the user presses `enter`
- **THEN** the description is `fix login bug`, the project row shows `Acme Internal`, the dropdown is closed, and no timer starts

#### Scenario: Esc keeps the text
- **WHEN** the dropdown is open for `@acm` and the user presses `esc`
- **THEN** the dropdown closes, the description still ends with `@acm`, and the dialog stays open

#### Scenario: Backspace over the @
- **WHEN** the description ends with `@` and the dropdown is open, and the user presses `backspace`
- **THEN** the `@` is deleted and the dropdown closes

### Requirement: Enter after the dropdown closes
Once the dropdown is closed, `enter` on the description SHALL do what it does without a dropdown: start the timer in the start dialog, and move to the next row in the edit dialog.

#### Scenario: Pick, then start
- **WHEN** the user picks a project from the dropdown in the start dialog and then presses `enter`
- **THEN** a timer starts with the trimmed description and the picked project

### Requirement: A new project is created on save
When an entry is saved with a `(new)` project, the system SHALL first create the project in Toggl (1 request), add it to the session's project list, and then save the entry with it (1 request). If creating fails, the entry SHALL NOT be saved and the error SHALL be shown. Closing the start dialog or discarding the edit SHALL send nothing.

#### Scenario: Start with a new project
- **WHEN** the start dialog's project row shows `newclient (new)` and the user starts the timer
- **THEN** the project `newclient` is created, then the timer starts with it, in 2 requests

#### Scenario: Discard sends nothing
- **WHEN** the edit dialog's project row shows `newclient (new)` and the user discards from the review
- **THEN** no request is made and no project is created

#### Scenario: Created project is reused
- **WHEN** a project was created but saving the entry then failed, and the user types the same name again
- **THEN** the list shows the created project and no create option

#### Scenario: Creating fails
- **WHEN** creating the project fails
- **THEN** the entry is not saved and the error is shown

### Requirement: Clearing a project
Picking "No project" SHALL set the project row to `(none)`, and saving SHALL leave the entry with no project in Toggl.

#### Scenario: Remove a project
- **WHEN** an entry's project is ACME, the user picks `No project`, and saves
- **THEN** the saved entry has no project

### Requirement: Permission errors are reported as such
A request that Toggl refuses with HTTP 403 after onboarding SHALL be reported as a permission problem, naming that the workspace may only let admins create projects when the request was creating a project. It SHALL NOT be reported as an invalid API token. During onboarding, a 403 SHALL still mean the token was rejected.

#### Scenario: Create refused
- **WHEN** creating a project returns HTTP 403
- **THEN** the error says Toggl refused the request and that the workspace may only let admins create projects, and does not mention the API token
