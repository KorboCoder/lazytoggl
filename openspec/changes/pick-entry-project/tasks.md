# Tasks

Test fixtures: `driver_test.go` loads projects ACME (101) and Internal (102). ID1 (running)
is on ACME, ID2 has no project, and ID3 is on Internal. Tests that need an archived project
add `Old Client` (103, inactive) to the loaded projects.

## 1. Live API facts (needs the user's OK: about 2–3 requests)

- [x] 1.1 With the user's OK, confirm against the live API, on a throwaway project and entry, the four facts in design.md §8: a `PUT` with `"project_id": null` clears the project; a `POST .../projects` with `name` and `active` is accepted (note the colour returned); whether `GET .../projects` pages at the user's project count; and what a duplicate name returns. Record the results in `docs/ARCHITECTURE.md`'s API notes. Verify by reading the recorded responses. If a fact differs from the design, pause and update design.md before group 2

## 2. `internal/toggl`

- [x] 2.1 Add `Project.Active`, decoded from `active`; extend `TestProjectsDoesNotFilterInactive` (`client_test.go`) to assert both values. Add the archived `Old Client` to the mock seed and check it in `mock_test.go`
- [x] 2.2 Add `Service.CreateProject(ctx, name)`. Client: `POST /workspaces/{wid}/projects`, pinned by `TestCreateProjectSendsNameAndActive` (`client_test.go`: method, path, body, decoded response). Mock: append with the next ID, pinned by `TestMockCreateProject`
- [x] 2.3 Make `Update` always send `project_id`, as `null` when 0 (or the form task 1.1 found). Verify with `TestUpdateClearsProjectWithNull` and extend `TestUpdateSendsStopOnlyWhenFinished` to assert `project_id` is present. `Start` still leaves it out: extend `TestStartSendsExpectedBody`
- [x] 2.4 Add `ErrForbidden` for 403 (401 stays `ErrInvalidToken`). Verify with `TestForbidden403ReturnsErrForbidden` and update `TestInvalidTokenReturnsErrInvalidToken` to cover 401 only. Make `onboard.friendlyError` use `errors.Is` and treat both as a rejected token, verified with a new case in the onboarding tests
- [x] 2.6 Follow project pages: `Client.Projects` requests `?page=N` while a page holds 151 projects (the page size task 1.1 found), and stops at the first short page. Verify with `TestProjectsFollowsFullPages` (two pages, 151 + 2: two requests, 153 projects) and `TestProjectsStopsAfterShortPage` (one request) in `client_test.go`
- [x] 2.5 Update `docs/ARCHITECTURE.md`: the `Service` interface listing, the endpoint table (create project, clearing a project), the 403 note, and request costs (a new project adds 1). Verify by reading it against design.md

## 3. `ProjectList`

- [x] 3.1 Create `internal/ui/dialogs/projectlist.go` with the empty-search order: "No project" only when the entry has a project, then active projects A–Z ignoring case, with archived ones hidden. Verify with `TestProjectListEmptySearchOrder` and `TestProjectListNoProjectOnlyWithCurrent` in a new `projectlist_test.go`
- [x] 3.2 Substring filtering (spaces count, A–Z kept, "No project" dropped for a non-empty search) and the create option (last; only option when nothing matches; hidden on an exact match ignoring case and spaces, archived names included). Verify with `TestProjectListFiltersBySubstring`, `TestProjectListSpacesArePartOfSearch`, `TestProjectListCreateWhenNothingMatches`, `TestProjectListCreateAfterPartialMatches`, `TestProjectListNoCreateOnExactMatch`, `TestProjectListNoCreateForArchivedName`
- [x] 3.3 Selection and view: the selection resets on a query change, `Move` clamps, and at most 6 rows show with a scroll window that follows the selection. Verify with `TestProjectListSelectionResetsOnQuery`, `TestProjectListMoveClamps`, `TestProjectListViewScrollsToSelection`

## 4. Row picker and inline `@`

- [x] 4.1 Create `ProjectPicker` (its own search input plus a `ProjectList`): typing filters, `up`/`down`/`ctrl+p`/`ctrl+n` move, `enter`/`tab` pick, and `esc` cancels. Verify with `TestProjectPickerPicks`, `TestProjectPickerEscCancels`, `TestProjectPickerTabPicks`
- [x] 4.2 Create `atSuggest` around a `textinput`: it opens only on a single `@` rune at a word start, the query is the text from `@` to the cursor, and it closes when the `@` is deleted or the cursor moves before it. Verify with `TestAtSuggestOpensAtWordStart`, `TestAtSuggestIgnoresAtInsideWord`, `TestAtSuggestIgnoresPastedAt`, `TestAtSuggestQueryFollowsTyping`, `TestAtSuggestClosesOnBackspaceOverAt`, `TestAtSuggestClosesWhenCursorLeaves`
- [x] 4.3 `atSuggest` keys while open: `enter`/`tab` pick and splice (remove `@query`, collapse spaces, trim), `esc` closes and keeps the text, and the selection keys are consumed. Verify with `TestAtSuggestPickSplicesText`, `TestAtSuggestPickAtStartOfText`, `TestAtSuggestEscKeepsText`, `TestAtSuggestConsumesEnterAndEsc`

## 5. Start dialog

- [x] 5.1 Make the start dialog a two-row form (description, project) with `tab`/arrow navigation. `enter` on the description starts the timer when no dropdown is open; `enter` on the project row opens the picker. Resuming pre-fills the project. Verify in `driver_test.go`: `TestStartDialogShowsResumedProject`, `TestStartDialogTabToProjectRow`, `TestStartPickProjectFromRow`; existing `TestResumePrefillsAndStarts` and `TestCreateBindingOpensStartDialog` still pass
- [x] 5.2 Wire `atSuggest` into the start description, with the box growing while the dropdown is open. Verify with `TestStartAtPicksProjectWithoutStarting` (enter picks, no `Start` call, description trimmed), then `TestStartAtThenEnterStarts` (a second `enter` starts with the picked project), and `TestStartAtEscKeepsTextAndDialog`
- [x] 5.3 Start with a new project: the row shows `<name> (new)`; on `enter`, the app creates the project, adds it to `m.projects`, then starts with its ID. Verify with `TestStartWithNewProjectCreatesThenStarts` (mock calls: `CreateProject` 1, `Start` 1; the entry's `ProjectID` is the new ID) and `TestStartCancelWithNewProjectSendsNothing`

## 6. Edit dialog

- [x] 6.1 Add the project row after duration (after start for a running entry), showing the name, `(none)`, or `<name> (new)`; `enter` opens the picker. Verify in `edit_test.go`: `TestEditFormShowsProjectRow`, `TestEditRunningFormShowsProjectRow`, `TestEditEnterOnProjectRowOpensPicker`; extend `TestEditTabCyclesRowsAndWraps` for the new last row; every existing `edit_test.go` case still passes unchanged
- [x] 6.2 Wire `atSuggest` into the edit description: while the dropdown is open, `enter` picks instead of moving rows and `esc` closes only the dropdown, with no review. Verify with `TestEditAtPicksProject`, `TestEditAtEnterDoesNotMoveRow`, `TestEditAtEscDoesNotOpenReview`
- [x] 6.3 Review and save: a `project  <before> -> <after>` line using `(none)` and `(new)`; picking "No project" saves with no project; saving a `(new)` project creates it first. Verify with `TestEditReviewShowsProjectChange`, `TestEditReviewShowsNewProject`, `TestEditReviewShowsProjectRemoved`, `TestEditClearProjectSaves` (saved `ProjectID` 0), `TestEditSaveWithNewProjectCreatesThenUpdates` (mock calls: `CreateProject` 1, `Update` 1), `TestEditDiscardWithNewProjectSendsNothing`

## 7. Errors and reuse

- [x] 7.1 A failed create shows the permission-aware error and starts or updates nothing. Verify with `TestCreateProjectForbiddenShowsPermissionError` (a fake `Service` returning `ErrForbidden`: the notice mentions admins and not the token; `Start`/`Update` not called)
- [x] 7.2 A project created before a failed entry save is reused: the list then matches it exactly and offers no create. Verify with `TestCreatedProjectReusedAfterFailedSave` (create succeeds, start fails; reopen, `@<name>` lists the project and no create option)

## 8. Docs and integration

- [x] 8.1 Update `docs/REQUIREMENTS.md`: feature list (pick, create, and clear a project), the keymap for the start dialog, project row, dropdown, and picker, and remove the "Project / tag / client pickers" backlog item's project part. Add a `docs/PLAN.md` decisions row citing this change. Verify by reading each new key against `proposal.md`'s keybinding table
- [x] 8.2 Run `go test -race ./...` and verify it passes
- [x] 8.3 Run `go run . -mock` and verify by hand: `c`, type `fix @int`, `enter` picks Internal, a second `enter` starts it; `@newclient` offers create, and starting adds a `newclient` project; in the edit dialog, the project row's picker changes the project and the review shows it; `Old Client` never appears in a list

## Workflow follow-up

- Run superpowers:requesting-code-review once all tasks are checked.
- Archive with `/opsx:archive` after `go test -race ./...` passes.
