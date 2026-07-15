// Package onboard is the first-run screen: it collects a Toggl API token,
// validates it, and hands back the authenticated user. It's a separate,
// small Bubble Tea program run once before the main app starts (see
// main.go) rather than a mode of ui.Model, so onboarding concerns never leak
// into the main app's state machine.
package onboard

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// Validate exchanges a candidate token for the authenticated user, or an
// error if it's invalid. main.go wires this to toggl.Client.Me; tests inject
// a fake so they never touch the network.
type Validate func(ctx context.Context, token string) (toggl.Me, error)

type Model struct {
	input      textinput.Model
	spinner    spinner.Model
	validate   Validate
	validating bool
	errStr     string

	Token     string
	Me        toggl.Me
	Done      bool
	Cancelled bool
}

func New(validate Validate) Model {
	ti := textinput.New()
	ti.Placeholder = "paste your Toggl API token"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.CharLimit = 128
	ti.Width = 50
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.CursorStyle

	return Model{input: ti, spinner: sp, validate: validate}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

type validatedMsg struct {
	token string
	me    toggl.Me
}

type errMsg struct{ err error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Cancelled, m.Done = true, true
			return m, tea.Quit
		case "enter":
			if m.validating {
				return m, nil
			}
			token := strings.TrimSpace(m.input.Value())
			if token == "" {
				return m, nil
			}
			m.validating = true
			m.errStr = ""
			return m, tea.Batch(m.spinner.Tick, validateCmd(m.validate, token))
		}

	case validatedMsg:
		m.Token, m.Me, m.Done = msg.token, msg.me, true
		return m, tea.Quit

	case errMsg:
		m.validating = false
		m.errStr = friendlyError(msg.err)
		return m, nil

	case spinner.TickMsg:
		if !m.validating {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if m.validating {
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	lines := []string{
		"Welcome to lazytoggl — paste your Toggl API token to continue.",
		theme.MutedStyle.Render("Find it at track.toggl.com → Profile settings → API Token."),
		"",
		m.input.View(),
	}
	switch {
	case m.validating:
		lines = append(lines, "", m.spinner.View()+" validating…")
	case m.errStr != "":
		lines = append(lines, "", theme.ErrStyle.Render(m.errStr))
	}
	lines = append(lines, "", theme.MutedStyle.Render("enter continue · esc quit"))
	return theme.Panel("lazytoggl setup", joinLines(lines), 64, len(lines)+2, true)
}

func validateCmd(validate Validate, token string) tea.Cmd {
	return func() tea.Msg {
		me, err := validate(context.Background(), token)
		if err != nil {
			return errMsg{err}
		}
		return validatedMsg{token: token, me: me}
	}
}

func friendlyError(err error) string {
	switch err {
	case toggl.ErrInvalidToken:
		return "That token was rejected — check it and try again."
	case toggl.ErrRateLimited:
		return "Rate limited — wait a moment and try again."
	default:
		return "Couldn't reach Toggl: " + err.Error()
	}
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}
