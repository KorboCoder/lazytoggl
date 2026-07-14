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

// waitFor waits until want shows up in newly produced output. teatest fully
// drains the output stream on each call, so a single frame's text can only
// be asserted once — checking two substrings from the *same* frame back to
// back will hang the second call forever (nothing new ever arrives). Use
// waitForAll for that case.
func waitFor(t *testing.T, tm *teatest.TestModel, want string) {
	t.Helper()
	waitForAll(t, tm, want)
}

// waitForAll waits until every want has appeared in the same accumulated
// read, so multiple substrings from one rendered frame can be asserted
// together without racing teatest's drain-on-each-call behavior.
func waitForAll(t *testing.T, tm *teatest.TestModel, wants ...string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		for _, w := range wants {
			if !bytes.Contains(b, []byte(w)) {
				return false
			}
		}
		return true
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

	// Pause the running timer from anywhere; it stays pinned to the Timer
	// panel with a paused indicator instead of clearing to idle.
	tm.Send(keyRune(' '))
	waitForAll(t, tm, "⏸", "paused")

	// Create a new entry through the start dialog.
	tm.Send(keyRune('c'))
	waitFor(t, tm, "what are you working on?")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Prototype smoke test")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, tm, "timer started")

	tm.Send(keyRune('q'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
