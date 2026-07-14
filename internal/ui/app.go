// Package ui is the Bubble Tea application: the root model owns the entry
// list (source of truth for rendering), panel focus, and the open dialog.
// All service calls run as async tea.Cmds; mutations patch local state from
// the response and never trigger a refetch (see docs/ARCHITECTURE.md).
package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	entriesPanel panels.Entries
	focus        int
	dialog       dialogs.Dialog

	// pausedID is the entry shown "paused" in the Timer panel: stopped via
	// space, but kept visible (frozen clock, ⏸) until resumed or superseded
	// by a new entry. Toggl has no native pause, so this is UI-only state
	// layered over m.entries, the same source of truth used elsewhere.
	pausedID int64

	width, height int
	now           time.Time
	loading       bool
	status        string
	statusErr     bool
}

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
type entryDeletedMsg struct{ id int64 }
type errMsg struct{ err error }

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		return m, tickCmd()

	case dataLoadedMsg:
		m.loading = false
		m.entries = msg.entries
		m.projects = map[int64]toggl.Project{}
		for _, p := range msg.projects {
			m.projects[p.ID] = p
		}
		m.setStatus("", false)
		m.clampCursor()
		return m, nil

	case entryStartedMsg:
		// Toggl stops the running entry when a new one starts; mirror that
		// locally instead of refetching.
		for i := range m.entries {
			if m.entries[i].Running() && m.entries[i].ID != msg.entry.ID {
				m.entries[i].Stop = msg.entry.Start
				m.entries[i].Duration = int64(msg.entry.Start.Sub(m.entries[i].Start).Seconds())
			}
		}
		m.entries = append([]toggl.TimeEntry{msg.entry}, m.entries...)
		m.pausedID = 0
		m.setStatus("timer started", false)
		m.clampCursor()
		return m, nil

	case entryStoppedMsg:
		m.replaceEntry(msg.entry)
		m.setStatus("timer stopped", false)
		return m, nil

	case entryPausedMsg:
		m.replaceEntry(msg.entry)
		m.pausedID = msg.entry.ID
		m.setStatus("paused", false)
		return m, nil

	case entryUpdatedMsg:
		m.replaceEntry(msg.entry)
		m.setStatus("entry saved", false)
		return m, nil

	case entryDeletedMsg:
		for i := range m.entries {
			if m.entries[i].ID == msg.id {
				m.entries = append(m.entries[:i], m.entries[i+1:]...)
				break
			}
		}
		if msg.id == m.pausedID {
			m.pausedID = 0
		}
		m.setStatus("entry deleted", false)
		m.clampCursor()
		return m, nil

	case errMsg:
		m.loading = false
		m.setStatus(msg.err.Error(), true)
		return m, nil

	case dialogs.CloseMsg:
		m.dialog = nil
		return m, nil

	case dialogs.StartSubmitMsg:
		m.dialog = nil
		m.setStatus("starting…", false)
		return m, m.startCmd(msg.Req)

	case dialogs.EditSubmitMsg:
		m.dialog = nil
		m.setStatus("saving…", false)
		return m, m.updateCmd(msg.Entry)

	case dialogs.DeleteConfirmMsg:
		m.dialog = nil
		m.setStatus("deleting…", false)
		return m, m.deleteCmd(msg.EntryID)

	case tea.KeyMsg:
		if m.dialog != nil {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
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
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

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
		return m, m.loadCmd()

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
		m.dialog = dialogs.NewStart("Start entry", "", 0, "")
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter(rows)

	case key.Matches(msg, m.keys.Edit):
		if e := m.actionTarget(rows); e != nil {
			m.dialog = dialogs.NewEdit(*e)
			return m, textinput.Blink
		}
		return m, nil

	case key.Matches(msg, m.keys.Delete):
		if e := m.actionTarget(rows); e != nil {
			desc := e.Description
			if desc == "" {
				desc = "(no description)"
			}
			m.dialog = dialogs.NewConfirm(e.ID, desc)
		}
		return m, nil
	}
	return m, nil
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
		m.dialog = dialogs.NewStart("Start entry", "", 0, "")
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
		row.Entry.ProjectID, m.projectName(row.Entry.ProjectID))
	return m, textinput.Blink
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	statusLine := " "
	if m.status != "" {
		if m.statusErr {
			statusLine = " " + theme.ErrStyle.Render(m.status)
		} else {
			statusLine = " " + theme.MutedStyle.Render(m.status)
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

	main := lipgloss.JoinVertical(lipgloss.Left, timer, entriesView, statusLine, helpView)

	if m.dialog != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.dialog.View())
	}
	return main
}

// Helpers.

func (m *Model) setStatus(s string, isErr bool) {
	m.status = s
	m.statusErr = isErr
}

func (m *Model) clampCursor() {
	m.entriesPanel.Clamp(panels.BuildRows(m.entries, m.now))
}

func (m *Model) replaceEntry(e toggl.TimeEntry) {
	for i := range m.entries {
		if m.entries[i].ID == e.ID {
			m.entries[i] = e
			return
		}
	}
	m.entries = append([]toggl.TimeEntry{e}, m.entries...)
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

func (m Model) projectName(id int64) string {
	if p, ok := m.projects[id]; ok && id != 0 {
		return p.Name
	}
	return ""
}

// Commands (async service calls).

func (m Model) loadCmd() tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		ctx := context.Background()
		entries, err := svc.RecentEntries(ctx)
		if err != nil {
			return errMsg{err}
		}
		projects, err := svc.Projects(ctx)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{entries: entries, projects: projects}
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

func (m Model) deleteCmd(id int64) tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		if err := svc.Delete(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return entryDeletedMsg{id: id}
	}
}
