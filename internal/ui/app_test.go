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
// timer stops via x, a new entry starts through the start dialog, and the
// app quits cleanly.
func TestSmoke(t *testing.T) {
	tm := teatest.NewTestModel(t, New(toggl.NewMock()), teatest.WithInitialTermSize(90, 30))

	// Seeded data appears (the "Yesterday" header renders in the same frame
	// as the running entry and day totals).
	waitFor(t, tm, "Yesterday")

	// Stop the running timer from anywhere.
	tm.Send(keyRune('x'))
	waitFor(t, tm, "timer stopped")

	// Open the start dialog from the "+ New entry" row and start an entry.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, tm, "what are you working on?")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Prototype smoke test")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, tm, "timer started")

	tm.Send(keyRune('q'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestDeleteFlow exercises the confirm dialog.
func TestDeleteFlow(t *testing.T) {
	tm := teatest.NewTestModel(t, New(toggl.NewMock()), teatest.WithInitialTermSize(90, 30))
	waitFor(t, tm, "Standup")

	// Move from "+ New entry" down past the header to the first two entries.
	tm.Send(keyRune('j')) // running entry
	tm.Send(keyRune('j')) // "Standup"
	tm.Send(keyRune('d'))
	waitFor(t, tm, "Delete entry?")
	tm.Send(keyRune('y'))
	waitFor(t, tm, "entry deleted")

	tm.Send(keyRune('q'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
