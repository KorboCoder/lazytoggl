package ui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// waitFor waits until want shows up in newly produced output. Note that the
// output stream is consumed and Bubble Tea repaints only changed lines, so
// each waitFor must target text that appears *after* the previous one.
func waitFor(t *testing.T, tm *teatest.TestModel, want string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(want))
	}, teatest.WithDuration(5*time.Second))
}

// TestSmoke drives the core loop against the mock: entries load, the running
// timer pauses via space, a new entry starts through the create dialog, and
// the app quits cleanly.
func TestSmoke(t *testing.T) {
	tm := teatest.NewTestModel(t, New(toggl.NewMock()), teatest.WithInitialTermSize(90, 30))

	// Seeded data appears (the "Yesterday" header renders in the same frame
	// as the running entry and day totals).
	waitFor(t, tm, "Yesterday")

	// Pause the running timer from anywhere.
	tm.Send(keyRune(' '))
	waitFor(t, tm, "timer stopped")

	// Create a new entry through the start dialog.
	tm.Send(keyRune('c'))
	waitFor(t, tm, "what are you working on?")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Prototype smoke test")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, tm, "timer started")

	tm.Send(keyRune('q'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
