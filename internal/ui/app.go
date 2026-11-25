// Package ui is the Bubble Tea application: the root model owns the entry
// list (source of truth for rendering), panel focus, and the open dialog.
// All service calls run as async tea.Cmds; mutations patch local state from
// the response and never trigger a refetch (see docs/ARCHITECTURE.md).
package ui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/dialogs"
	"github.com/KorboCoder/lazytoggl/internal/ui/panels"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

const (
	focusTimer = iota
	focusEntries
)

type Model struct {
	svc  toggl.Service
	keys keyMap
	help help.Model

	entries  []toggl.TimeEntry
	projects map[int64]toggl.Project
	// projectsLoaded is set once Projects succeeds. Until then a refresh
	// fetches them too, and the pickers offer no create option.
	projectsLoaded bool

	entriesPanel panels.Entries
	focus        int
	dialog       dialogs.Dialog

	// pausedID is the entry shown "paused" in the Timer panel: stopped via
	// space, but kept visible (frozen clock, ⏸) until resumed or superseded
	// by a new entry. Toggl has no native pause, so this is UI-only state
	// layered over m.entries, the same source of truth used elsewhere.
	pausedID int64

	// pending is the delete u can undo until its deadline; deleting holds
	// deletes sent but not yet answered. Both stay hidden from refreshes.
	pending  *pendingDelete
	deleting map[int64]deletedEntry
	// quitting is set by q/ctrl+c while deletes are pending or in flight:
	// the app quits once they're answered (a second q quits at once).
	quitting bool
	// quitErrs are deletes that didn't reach Toggl while quitting, reported
	// after the program exits (QuitError).
	quitErrs []string

	width, height int
	now           time.Time
	loading       bool
	status        string
	statusErr     bool

	// notice is the floating corner popup shown when an API call fails —
	// unlike the one-line status it can't be missed or silently overwritten
	// by the next status change. Cleared by esc, by any successful API
	// response, or when noticeExpiry passes.
	notice       string
	noticeExpiry time.Time
}

// noticeTTL is how long the API-error popup stays up if not dismissed.
const noticeTTL = 10 * time.Second

func New(svc toggl.Service) Model {
	return Model{
		svc:      svc,
		keys:     defaultKeys(),
		help:     help.New(),
		projects: map[int64]toggl.Project{},
		focus:    focusEntries,
		now:      time.Now(),
		loading:  true,
		status:   "loading…",
	}
}

// Messages.

type tickMsg time.Time

type dataLoadedMsg struct {
	entries  []toggl.TimeEntry
	projects []toggl.Project
}

type entryStartedMsg struct{ entry toggl.TimeEntry }
type entryStoppedMsg struct{ entry toggl.TimeEntry }
type entryPausedMsg struct{ entry toggl.TimeEntry }
type entryUpdatedMsg struct{ entry toggl.TimeEntry }

// entryLoggedMsg reports a finished entry created after the fact. Unlike
// entryStartedMsg it never stops the running entry (nor does Toggl).
type entryLoggedMsg struct{ entry toggl.TimeEntry }

// projectCreatedMsg reports a project created for an entry being saved;
// next saves the entry with the project's ID.
type projectCreatedMsg struct {
	project toggl.Project
	status  string // shown while next runs: "starting…" or "saving…"
	next    func(projectID int64) tea.Cmd
}
type entryDeletedMsg struct{ id int64 }

// deleteFailedMsg reports a delete Toggl refused; the entry comes back.
type deleteFailedMsg struct {
	gone deletedEntry
	err  error
}

// undoGrace is how long a delete can be undone before it is sent: undoing
// costs no request because nothing was sent yet.
const undoGrace = 5 * time.Second

// deletedEntry is an entry hidden by d, with whether it was the timer
// panel's paused entry, so an undo (or a failed delete) restores it as it was.
type deletedEntry struct {
	entry  toggl.TimeEntry
	paused bool
}

// pendingDelete is the one delete that u can still undo.
type pendingDelete struct {
	deletedEntry
	deadline time.Time
}
type errMsg struct{ err error }

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(true), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		if m.notice != "" && m.now.After(m.noticeExpiry) {
			m.notice = ""
		}
		// The grace period is over: send the delete (1 request).
		if p := m.pending; p != nil && !m.now.Before(p.deadline) {
			return m, tea.Batch(tickCmd(), m.commitPending())
		}
		return m, tickCmd()

	case dataLoadedMsg:
		m.loading = false
		m.notice = ""
		m.entries = m.withoutDeleted(msg.entries)
		// A refresh reuses the session's project map (projects rarely change
		// and every request counts against the hourly quota); msg.projects
		// is non-nil only when they were fetched. Merged, not replaced, so a
		// project created before a late load survives it.
		if msg.projects != nil {
			for _, p := range msg.projects {
				m.projects[p.ID] = p
			}
			m.projectsLoaded = true
		}
		m.setStatus("", false)
		m.clampCursor()
		return m, nil

	case entryStartedMsg:
		// Toggl stops the running entry when a new one starts; mirror that
		// locally instead of refetching.
		for i := range m.entries {
			stopAt(&m.entries[i], msg.entry)
		}
		// Hidden copies too, so an undo doesn't bring back a second timer
		// or re-pause an entry the new start superseded.
		m.eachHidden(func(g *deletedEntry) {
			stopAt(&g.entry, msg.entry)
			g.paused = false
		})
		m.entries = append([]toggl.TimeEntry{msg.entry}, m.entries...)
		m.pausedID = 0
		m.notice = ""
		m.setStatus("timer started", false)
		m.clampCursor()
		return m, nil

	case entryStoppedMsg:
		if !m.patchHidden(msg.entry, false) {
			m.replaceEntry(msg.entry)
		}
		m.notice = ""
		m.setStatus("timer stopped", false)
		return m, nil

	case entryPausedMsg:
		if !m.patchHidden(msg.entry, true) {
			m.replaceEntry(msg.entry)
			m.pausedID = msg.entry.ID
		}
		m.notice = ""
		m.setStatus("paused", false)
		return m, nil

	case entryUpdatedMsg:
		if m.patchHidden(msg.entry, false) {
			m.setStatus("entry saved", false)
			return m, nil
		}
		var prevStart time.Time
		if prev, ok := m.entryByID(msg.entry.ID); ok {
			prevStart = prev.Start
		}
		m.replaceEntry(msg.entry)
		// An edit can move the entry to another day; keep it selected.
		m.entriesPanel.SelectID(panels.BuildRows(m.entries, m.now), msg.entry.ID)
		m.notice = ""
		m.setStatus(savedStatus(msg.entry, prevStart, m.now), false)
		return m, nil

	case deleteFailedMsg:
		delete(m.deleting, msg.gone.entry.ID)
		if m.quitting {
			// The entry survives in Toggl; say so once the screen is gone.
			m.quitErrs = append(m.quitErrs, fmt.Sprintf("couldn't delete %q (%v): it's still in Toggl",
				label(msg.gone.entry), msg.err))
			return m, m.quitWhenDone()
		}
		// Back where it was, without moving the cursor off what the user
		// has moved on to.
		sel, hasSel := m.entriesPanel.Selected(panels.BuildRows(m.entries, m.now))
		m.restore(msg.gone)
		if hasSel && sel.Kind == panels.RowEntry {
			m.entriesPanel.SelectID(panels.BuildRows(m.entries, m.now), sel.Entry.ID)
		}
		return m.Update(errMsg{msg.err})

	case entryDeletedMsg:
		delete(m.deleting, msg.id)
		if m.quitting {
			return m, m.quitWhenDone()
		}
		for i := range m.entries {
			if m.entries[i].ID == msg.id {
				m.entries = append(m.entries[:i], m.entries[i+1:]...)
				break
			}
		}
		if msg.id == m.pausedID {
			m.pausedID = 0
		}
		m.notice = ""
		m.setStatus("entry deleted", false)
		m.clampCursor()
		return m, nil

	case errMsg:
		m.loading = false
		text := msg.err.Error()
		var create createProjectError
		switch {
		case errors.Is(msg.err, toggl.ErrRateLimited):
			text = "Rate limited by Toggl — the Free plan allows 30 requests/hour. Wait a bit before retrying."
		case errors.Is(msg.err, toggl.ErrForbidden) && errors.As(msg.err, &create):
			text = "Toggl refused to create the project — this workspace may only let admins create projects. Nothing was saved."
		case errors.Is(msg.err, toggl.ErrForbidden):
			text = "Toggl refused the request — you may not have permission."
		}
		m.notice = text
		m.noticeExpiry = m.now.Add(noticeTTL)
		m.setStatus(msg.err.Error(), true)
		return m, nil

	case dialogs.CloseMsg:
		m.dialog = nil
		return m, nil

	case dialogs.StartSubmitMsg:
		m.dialog = nil
		if msg.NewProject != "" {
			req := msg.Req
			m.setStatus("creating project…", false)
			return m, m.createProjectCmd(msg.NewProject, "starting…", func(id int64) tea.Cmd {
				req.ProjectID = id
				return m.startCmd(req)
			})
		}
		m.setStatus("starting…", false)
		return m, m.startCmd(msg.Req)

	case projectCreatedMsg:
		// Added before the entry is saved, so a failed save can reuse it.
		m.projects[msg.project.ID] = msg.project
		m.setStatus(msg.status, false)
		return m, msg.next(msg.project.ID)

	case dialogs.LogSubmitMsg:
		m.dialog = nil
		if msg.NewProject != "" {
			req := msg.Req
			m.setStatus("creating project…", false)
			return m, m.createProjectCmd(msg.NewProject, "logging…", func(id int64) tea.Cmd {
				req.ProjectID = id
				return m.logCmd(req)
			})
		}
		m.setStatus("logging…", false)
		return m, m.logCmd(msg.Req)

	case entryLoggedMsg:
		m.replaceEntry(msg.entry)
		m.entriesPanel.SelectID(panels.BuildRows(m.entries, m.now), msg.entry.ID)
		m.notice = ""
		m.setStatus(loggedStatus(msg.entry, m.now), false)
		return m, nil

	case dialogs.EditSubmitMsg:
		m.dialog = nil
		if msg.NewProject != "" {
			entry := msg.Entry
			m.setStatus("creating project…", false)
			return m, m.createProjectCmd(msg.NewProject, "saving…", func(id int64) tea.Cmd {
				entry.ProjectID = id
				return m.updateCmd(entry)
			})
		}
		m.setStatus("saving…", false)
		return m, m.updateCmd(msg.Entry)

	case tea.KeyMsg:
		// While quitting, only a second q/ctrl+c acts (quitting at once):
		// anything else could start work that would be dropped on exit.
		if m.quitting {
			if msg.String() == "ctrl+c" || (m.dialog == nil && key.Matches(msg, m.keys.Quit)) {
				return m.quit()
			}
			return m, nil
		}
		if m.dialog != nil {
			if msg.String() == "ctrl+c" {
				return m.quit()
			}
			var cmd tea.Cmd
			m.dialog, cmd = m.dialog.Update(msg)
			return m, cmd
		}
		return m.handleKey(msg)

	default:
		// Everything else (e.g. cursor blink) belongs to the open dialog.
		if m.dialog != nil {
			var cmd tea.Cmd
			m.dialog, cmd = m.dialog.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := panels.BuildRows(m.entries, m.now)

	switch {
	case msg.String() == "esc":
		m.notice = ""
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		return m.quit()

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		if m.focus == focusTimer {
			m.focus = focusEntries
		} else {
			m.focus = focusTimer
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.setStatus("refreshing…", false)
		return m, m.loadCmd(!m.projectsLoaded)

	case key.Matches(msg, m.keys.Up):
		m.entriesPanel.Move(rows, -1)
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.entriesPanel.Move(rows, 1)
		return m, nil

	case key.Matches(msg, m.keys.Space):
		// Pause/resume toggles whatever the Timer panel is already showing.
		// Toggl has no real pause, so pausing stops the entry via the API
		// but keeps it pinned in the Timer panel (⏸, frozen clock) until
		// resumed, instead of clearing back to idle. It's a no-op when the
		// panel is idle — space never reaches into history to resume
		// something that wasn't already active or paused.
		if run := m.runningEntry(); run != nil {
			m.setStatus("pausing…", false)
			return m, m.pauseCmd(run.ID)
		}
		if p := m.pausedEntry(); p != nil {
			m.setStatus("resuming…", false)
			return m, m.startCmd(toggl.StartRequest{Description: p.Description, ProjectID: p.ProjectID})
		}
		return m, nil

	case key.Matches(msg, m.keys.Create):
		m.dialog = dialogs.NewStart("Start entry", "", 0, m.projectList())
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Log):
		m.dialog = dialogs.NewLog(m.now, m.projectList())
		return m, textinput.Blink

	case key.Matches(msg, m.keys.GapBelow), key.Matches(msg, m.keys.GapAbove):
		return m.openGap(rows, key.Matches(msg, m.keys.GapAbove))

	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter(rows)

	case key.Matches(msg, m.keys.Edit):
		if e := m.actionTarget(rows); e != nil {
			m.dialog = dialogs.NewEdit(*e, m.now, m.projectList())
			return m, textinput.Blink
		}
		return m, nil

	case key.Matches(msg, m.keys.Delete):
		if e := m.actionTarget(rows); e != nil {
			return m.deleteWithUndo(*e)
		}
		return m, nil

	case key.Matches(msg, m.keys.Undo):
		if p := m.pending; p != nil {
			m.pending = nil
			m.restore(p.deletedEntry)
			m.entriesPanel.SelectID(panels.BuildRows(m.entries, m.now), p.entry.ID)
		}
		return m, nil
	}
	return m, nil
}

// openGap opens the log dialog on the gap below (o) or above (O) the
// selected list entry. It acts on the list only: the timer panel has no
// selection.
func (m Model) openGap(rows []panels.Row, above bool) (tea.Model, tea.Cmd) {
	if m.focus == focusTimer {
		return m, nil
	}
	row, ok := m.entriesPanel.Selected(rows)
	if !ok || row.Kind != panels.RowEntry {
		return m, nil
	}
	gap, side := gapBelow, "below"
	if above {
		if row.Entry.Running() {
			m.setStatus("nothing after the running entry", false)
			return m, nil
		}
		gap, side = gapAbove, "above"
	}
	start, stop, ok := gap(m.entries, row.Entry, m.now)
	if !ok {
		desc := row.Entry.Description
		if desc == "" {
			desc = "(no description)"
		}
		m.setStatus(fmt.Sprintf("no gap %s \"%s\"", side, desc), false)
		return m, nil
	}
	m.dialog = dialogs.NewLogAt(start, stop, m.now, m.projectList())
	return m, textinput.Blink
}

// actionTarget is the entry e/d act on: whatever the Timer panel is showing
// (running, else paused) when it's focused, otherwise the list selection.
func (m *Model) actionTarget(rows []panels.Row) *toggl.TimeEntry {
	if m.focus == focusTimer {
		if run := m.runningEntry(); run != nil {
			return run
		}
		if p := m.pausedEntry(); p != nil {
			return p
		}
		m.setStatus("no timer running", false)
		return nil
	}
	if row, ok := m.entriesPanel.Selected(rows); ok && row.Kind == panels.RowEntry {
		e := row.Entry
		return &e
	}
	return nil
}

// handleEnter resumes a past entry (pre-filled dialog) or stops the running
// one; on an idle Timer panel it opens the create dialog like c does.
func (m Model) handleEnter(rows []panels.Row) (tea.Model, tea.Cmd) {
	if m.focus == focusTimer {
		if run := m.runningEntry(); run != nil {
			m.setStatus("stopping…", false)
			return m, m.stopCmd(run.ID)
		}
		m.dialog = dialogs.NewStart("Start entry", "", 0, m.projectList())
		return m, textinput.Blink
	}

	row, ok := m.entriesPanel.Selected(rows)
	if !ok || row.Kind != panels.RowEntry {
		return m, nil
	}
	if row.Entry.Running() {
		m.setStatus("stopping…", false)
		return m, m.stopCmd(row.Entry.ID)
	}
	m.dialog = dialogs.NewStart("Resume entry", row.Entry.Description,
		row.Entry.ProjectID, m.projectList())
	return m, textinput.Blink
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	status, hint := m.statusTexts(m.width - 1)
	statusLine := " "
	if status != "" {
		if m.statusErr && !m.quitting {
			statusLine = " " + theme.ErrStyle.Render(status)
		} else {
			statusLine = " " + theme.MutedStyle.Render(status)
		}
	}
	if hint != "" {
		if status == "" {
			statusLine = " " + theme.MutedStyle.Render(hint)
		} else {
			statusLine += theme.MutedStyle.Render(" · " + hint)
		}
	}
	helpView := m.help.View(m.keys)
	bottomH := lipgloss.Height(statusLine) + lipgloss.Height(helpView)

	const timerH = 3
	entriesH := m.height - timerH - bottomH
	if entriesH < 5 {
		entriesH = 5
	}

	timer := panels.Timer(m.runningEntry(), m.pausedEntry(), m.projects, m.now, m.width, m.focus == focusTimer)
	rows := panels.BuildRows(m.entries, m.now)
	entriesView := m.entriesPanel.View(rows, m.projects, m.now, m.width, entriesH,
		m.loading, m.focus == focusEntries)

	view := lipgloss.JoinVertical(lipgloss.Left, timer, entriesView, statusLine, helpView)

	if m.dialog != nil {
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.dialog.View())
	}
	if m.notice != "" {
		box := m.noticeView()
		x := m.width - lipgloss.Width(box) - 1
		if x > 0 {
			view = theme.Overlay(view, box, x, 1)
		}
	}
	return view
}

// noticeView renders the floating API-error popup for the top-right corner.
func (m Model) noticeView() string {
	w := 44
	if max := m.width - 8; w > max {
		w = max
	}
	content := theme.ErrStyle.Render("⚠ API error") + "\n" +
		m.notice + "\n" +
		theme.MutedStyle.Render("esc to dismiss")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Red).
		Padding(0, 1).
		Width(w).
		Render(content)
}

// Helpers.

func (m *Model) setStatus(s string, isErr bool) {
	m.status = s
	m.statusErr = isErr
}

func (m *Model) clampCursor() {
	m.entriesPanel.Clamp(panels.BuildRows(m.entries, m.now))
}

// savedStatus is the status after an edit. An entry whose start was moved
// outside the fetch window (too old, or after now) stays listed for now but
// won't come back on refresh, so say so rather than let it vanish
// unexplained.
func savedStatus(e toggl.TimeEntry, prevStart, now time.Time) string {
	if e.Start.Equal(prevStart) {
		return "entry saved"
	}
	return windowStatus("entry saved", "moved to", e, now)
}

// loggedStatus is the status after logging a new entry: there is no earlier
// start, so it warns whenever the entry lands outside the window.
func loggedStatus(e toggl.TimeEntry, now time.Time) string {
	return windowStatus("entry logged", "on", e, now)
}

// windowStatus is done, plus a warning when e's start is outside the fetch
// window: it stays listed now but won't come back on refresh.
func windowStatus(done, dayPrep string, e toggl.TimeEntry, now time.Time) string {
	if toggl.InRecentWindow(e.Start, now) {
		return done
	}
	if e.Start.After(now) {
		return done + " — starts after now, outside the list; it drops off on refresh"
	}
	days := int(toggl.RecentEntriesLookback / (24 * time.Hour))
	return fmt.Sprintf("%s — %s %s, outside the %d-day list; it drops off on refresh",
		done, dayPrep, e.Start.Local().Format("Mon, Jan 2"), days)
}

// replaceEntry patches e into m.entries and keeps them newest-first, which
// panels.BuildRows relies on to group days: an edit can move an entry's
// start anywhere, even to another day.
func (m *Model) replaceEntry(e toggl.TimeEntry) {
	found := false
	for i := range m.entries {
		if m.entries[i].ID == e.ID {
			m.entries[i] = e
			found = true
			break
		}
	}
	if !found {
		m.entries = append([]toggl.TimeEntry{e}, m.entries...)
	}
	sort.SliceStable(m.entries, func(i, j int) bool { return m.entries[i].Start.After(m.entries[j].Start) })
}

func (m Model) entryByID(id int64) (toggl.TimeEntry, bool) {
	for _, e := range m.entries {
		if e.ID == id {
			return e, true
		}
	}
	return toggl.TimeEntry{}, false
}

func (m Model) runningEntry() *toggl.TimeEntry {
	for i := range m.entries {
		if m.entries[i].Running() {
			return &m.entries[i]
		}
	}
	return nil
}

// pausedEntry looks up pausedID in m.entries (the source of truth), so a
// paused entry that gets edited or deleted elsewhere stays consistent (or
// disappears) without any separate bookkeeping.
func (m Model) pausedEntry() *toggl.TimeEntry {
	if m.pausedID == 0 {
		return nil
	}
	for i := range m.entries {
		if m.entries[i].ID == m.pausedID {
			return &m.entries[i]
		}
	}
	return nil
}

// projectList is the session's projects, archived included, for the
// dialogs' pickers (which order and filter them). It is nil until projects
// have loaded, which the pickers show instead of offering to create them.
func (m Model) projectList() []toggl.Project {
	if !m.projectsLoaded {
		return nil
	}
	out := make([]toggl.Project, 0, len(m.projects))
	for _, p := range m.projects {
		out = append(out, p)
	}
	return out
}

func (m Model) projectName(id int64) string {
	if p, ok := m.projects[id]; ok && id != 0 {
		return p.Name
	}
	return ""
}

// Commands (async service calls).

// loadCmd fetches entries, plus projects on the initial load only — a
// refresh reuses the session's project map so it costs one request instead
// of two against the hourly quota.
func (m Model) loadCmd(withProjects bool) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		ctx := context.Background()
		entries, err := svc.RecentEntries(ctx)
		if err != nil {
			return errMsg{err}
		}
		msg := dataLoadedMsg{entries: entries}
		if withProjects {
			projects, err := svc.Projects(ctx)
			if err != nil {
				return errMsg{err}
			}
			// Non-nil marks them fetched, even for a workspace with none.
			msg.projects = append([]toggl.Project{}, projects...)
		}
		return msg
	}
}

// createProjectError marks a failure to create a project, so the notice can
// say what was refused and that the entry wasn't saved either.
type createProjectError struct {
	name string
	err  error
}

func (e createProjectError) Error() string {
	return fmt.Sprintf("creating project %q: %v", e.name, e.err)
}

func (e createProjectError) Unwrap() error { return e.err }

// createProjectCmd creates a project for an entry being saved (1 request);
// next then saves the entry with it.
func (m Model) createProjectCmd(name, status string, next func(int64) tea.Cmd) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		p, err := svc.CreateProject(context.Background(), name)
		if err != nil {
			return errMsg{createProjectError{name: name, err: err}}
		}
		return projectCreatedMsg{project: p, status: status, next: next}
	}
}

func (m Model) logCmd(req toggl.LogRequest) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		e, err := svc.Log(context.Background(), req)
		if err != nil {
			return errMsg{err}
		}
		return entryLoggedMsg{entry: e}
	}
}

func (m Model) startCmd(req toggl.StartRequest) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		e, err := svc.Start(context.Background(), req)
		if err != nil {
			return errMsg{err}
		}
		return entryStartedMsg{entry: e}
	}
}

func (m Model) stopCmd(id int64) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		e, err := svc.Stop(context.Background(), id)
		if err != nil {
			return errMsg{err}
		}
		return entryStoppedMsg{entry: e}
	}
}

// pauseCmd stops the entry via the same API call as stopCmd, but tags the
// result as a pause so the Timer panel keeps it pinned instead of going idle.
func (m Model) pauseCmd(id int64) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		e, err := svc.Stop(context.Background(), id)
		if err != nil {
			return errMsg{err}
		}
		return entryPausedMsg{entry: e}
	}
}

func (m Model) updateCmd(entry toggl.TimeEntry) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		e, err := svc.Update(context.Background(), entry)
		if err != nil {
			return errMsg{err}
		}
		return entryUpdatedMsg{entry: e}
	}
}

// quit quits, but first sends a pending delete and waits for deletes in
// flight, so a delete the user made is never silently dropped. A second
// q/ctrl+c while waiting quits at once.
func (m Model) quit() (tea.Model, tea.Cmd) {
	if m.quitting {
		for _, g := range m.deleting {
			m.quitErrs = append(m.quitErrs, fmt.Sprintf("quit before Toggl answered: %q may still be in Toggl", label(g.entry)))
		}
		return m, tea.Quit
	}
	if m.pending == nil && len(m.deleting) == 0 {
		return m, tea.Quit
	}
	m.quitting = true
	m.dialog = nil
	return m, m.commitPending()
}

const quittingStatus = "deleting before quitting… (q again to quit now)"

// quitWhenDone quits once no delete is in flight.
func (m Model) quitWhenDone() tea.Cmd {
	if len(m.deleting) == 0 {
		return tea.Quit
	}
	return nil
}

// QuitError describes deletes that didn't reach Toggl while quitting, for
// printing after the program exits; "" when there were none.
func (m Model) QuitError() string {
	return strings.Join(m.quitErrs, "\n")
}

// deleteWithUndo hides e at once and makes it the pending delete; a delete
// already pending is sent now, since only one can be undone.
func (m Model) deleteWithUndo(e toggl.TimeEntry) (tea.Model, tea.Cmd) {
	cmd := m.commitPending()
	gone := deletedEntry{entry: e, paused: e.ID == m.pausedID}
	m.hide(e.ID)
	m.pending = &pendingDelete{deletedEntry: gone, deadline: m.now.Add(undoGrace)}
	return m, cmd
}

// withoutDeleted drops the pending delete and deletes still in flight from
// freshly loaded entries: Toggl returns them until the DELETE lands. Their
// hidden copies take the fresh data, so an undo restores what Toggl has.
func (m *Model) withoutDeleted(entries []toggl.TimeEntry) []toggl.TimeEntry {
	out := entries[:0:0]
	for _, e := range entries {
		if !m.patchHidden(e, false) {
			out = append(out, e)
		}
	}
	return out
}

// eachHidden applies f to the pending delete and the deletes in flight.
func (m *Model) eachHidden(f func(*deletedEntry)) {
	if m.pending != nil {
		p := *m.pending // copy: older Models share the pointer
		f(&p.deletedEntry)
		m.pending = &p
	}
	for id, g := range m.deleting {
		f(&g)
		m.deleting[id] = g
	}
}

// patchHidden stores e in its hidden copy, if e is pending or in flight,
// and reports whether it was. paused marks it paused (a pause reply); a
// running copy is never paused.
func (m *Model) patchHidden(e toggl.TimeEntry, paused bool) bool {
	found := false
	m.eachHidden(func(g *deletedEntry) {
		if g.entry.ID == e.ID {
			g.entry = e
			g.paused = (g.paused || paused) && !e.Running()
			found = true
		}
	})
	return found
}

// stopAt mirrors Toggl stopping a running entry when started begins.
func stopAt(e *toggl.TimeEntry, started toggl.TimeEntry) {
	if e.Running() && e.ID != started.ID {
		e.Stop = started.Start
		e.Duration = int64(started.Start.Sub(e.Start).Seconds())
	}
}

// label is an entry's description for messages.
func label(e toggl.TimeEntry) string {
	if e.Description == "" {
		return "(no description)"
	}
	return e.Description
}

// hide removes the entry from m.entries, and so from the list and the
// timer panel, which both read it.
func (m *Model) hide(id int64) {
	for i := range m.entries {
		if m.entries[i].ID == id {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			break
		}
	}
	if id == m.pausedID {
		m.pausedID = 0
	}
	m.clampCursor()
}

// restore puts a hidden entry back as it was.
func (m *Model) restore(gone deletedEntry) {
	m.replaceEntry(gone.entry)
	if gone.paused {
		m.pausedID = gone.entry.ID
	}
	m.clampCursor()
}

// commitPending sends the pending delete, if any, and returns its request.
func (m *Model) commitPending() tea.Cmd {
	p := m.pending
	if p == nil {
		return nil
	}
	m.pending = nil
	if m.deleting == nil {
		m.deleting = map[int64]deletedEntry{}
	}
	m.deleting[p.entry.ID] = p.deletedEntry
	return m.deleteCmd(p.deletedEntry)
}

func (m Model) deleteCmd(gone deletedEntry) tea.Cmd {
	svc := m.svc
	id := gone.entry.ID
	return func() tea.Msg {
		if err := svc.Delete(context.Background(), id); err != nil {
			return deleteFailedMsg{gone: gone, err: err}
		}
		return entryDeletedMsg{id: id}
	}
}

// statusTexts is the status and the undo hint for a status line width
// columns wide. While a delete can be undone, the hint rides along with any
// other status, so u is never available without saying so; when they don't
// fit, the description shortens first, then the other status, so "u undo
// (Ns)" stays on screen.
func (m Model) statusTexts(width int) (status, hint string) {
	status = m.status
	if m.quitting {
		status = quittingStatus
	}
	p := m.pending
	if p == nil {
		return ansi.Truncate(status, max(width, 0), "…"), ""
	}
	left := p.deadline.Sub(m.now)
	secs := max(int((left+time.Second-1)/time.Second), 0) // round up
	head, tail := `deleted "`, fmt.Sprintf(`" · u undo (%ds)`, secs)
	desc := label(p.entry)
	if status != "" {
		width -= ansi.StringWidth(status) + len(" · ")
	}
	room := width - ansi.StringWidth(head+tail)
	if room < ansi.StringWidth(desc) {
		desc = ansi.Truncate(desc, max(room, 4), "…")
	}
	hint = head + desc + tail
	if width < ansi.StringWidth(hint) && status != "" {
		over := ansi.StringWidth(hint) - width
		status = ansi.Truncate(status, max(ansi.StringWidth(status)-over, 1), "…")
	}
	return status, hint
}
