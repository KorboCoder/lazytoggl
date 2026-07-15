package onboard

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

// press runs a key through Update and, if it returned a command, executes it
// inline and feeds the result(s) back — mirroring the synchronous driver
// used for the main app's tests (see internal/ui/driver_test.go). It expands
// one level of tea.Batch (Update returns Batch(spinner.Tick, validateCmd) on
// submit) but deliberately does not chase whatever cmd each of *those*
// produces — the spinner's own tick reschedules itself forever, and
// recursing into that would hang the test for no test-relevant benefit; the
// messages this suite cares about (validatedMsg/errMsg) are terminal.
//
// Only "enter" ever needs its cmd executed (it's what drives validation).
// Every other key's cmd is textinput's cursor-blink tea.Tick, a real timer —
// executing it would burn ~530ms of wall-clock time per keystroke for a
// purely cosmetic effect, so it's left unexecuted.
func press(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	mm, cmd := m.Update(msg)
	m = mm.(Model)
	if k, ok := msg.(tea.KeyMsg); ok && k.Type != tea.KeyEnter {
		return m
	}
	if cmd == nil {
		return m
	}
	result := cmd()
	if result == nil {
		return m
	}
	if batch, ok := result.(tea.BatchMsg); ok {
		for _, sub := range batch {
			if sub == nil {
				continue
			}
			if subMsg := sub(); subMsg != nil {
				mm, _ := m.Update(subMsg)
				m = mm.(Model)
			}
		}
		return m
	}
	mm, _ = m.Update(result)
	return mm.(Model)
}

func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = press(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestSuccessfulValidation(t *testing.T) {
	validate := func(ctx context.Context, token string) (toggl.Me, error) {
		if token != "good-token" {
			t.Fatalf("validate called with %q, want good-token", token)
		}
		return toggl.Me{ID: 1, Fullname: "Ada", DefaultWorkspaceID: 999}, nil
	}
	m := New(validate)
	m = typeText(t, m, "good-token")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !m.Done || m.Cancelled {
		t.Fatalf("Done=%v Cancelled=%v, want Done and not cancelled", m.Done, m.Cancelled)
	}
	if m.Token != "good-token" || m.Me.DefaultWorkspaceID != 999 {
		t.Fatalf("Token=%q Me=%+v, want good-token / workspace 999", m.Token, m.Me)
	}
}

func TestInvalidTokenShowsErrorAndAllowsRetry(t *testing.T) {
	calls := 0
	validate := func(ctx context.Context, token string) (toggl.Me, error) {
		calls++
		if token == "bad" {
			return toggl.Me{}, toggl.ErrInvalidToken
		}
		return toggl.Me{ID: 2, DefaultWorkspaceID: 5}, nil
	}
	m := New(validate)
	m = typeText(t, m, "bad")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.Done {
		t.Fatal("an invalid token must not finish onboarding")
	}
	if !strings.Contains(m.View(), "rejected") {
		t.Fatalf("view does not show the invalid-token message:\n%s", m.View())
	}

	// Clear the input and retry with a good token.
	for range "bad" {
		m = press(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m = typeText(t, m, "good")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !m.Done || m.Me.DefaultWorkspaceID != 5 {
		t.Fatalf("retry did not succeed: Done=%v Me=%+v", m.Done, m.Me)
	}
	if calls != 2 {
		t.Fatalf("validate called %d times, want 2", calls)
	}
}

func TestEscCancels(t *testing.T) {
	m := New(func(ctx context.Context, token string) (toggl.Me, error) {
		t.Fatal("validate must not be called when the user cancels")
		return toggl.Me{}, nil
	})
	m = typeText(t, m, "something")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	if !m.Done || !m.Cancelled {
		t.Fatalf("Done=%v Cancelled=%v, want both true after esc", m.Done, m.Cancelled)
	}
}

func TestEmptyInputDoesNotSubmit(t *testing.T) {
	m := New(func(ctx context.Context, token string) (toggl.Me, error) {
		t.Fatal("validate must not be called with an empty token")
		return toggl.Me{}, nil
	})
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Done {
		t.Fatal("submitting an empty token must not finish onboarding")
	}
}

func TestRateLimitedShowsFriendlyMessage(t *testing.T) {
	m := New(func(ctx context.Context, token string) (toggl.Me, error) {
		return toggl.Me{}, toggl.ErrRateLimited
	})
	m = typeText(t, m, "token")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !strings.Contains(m.View(), "Rate limited") {
		t.Fatalf("view does not show the rate-limit message:\n%s", m.View())
	}
}

func TestNetworkErrorShowsRawMessage(t *testing.T) {
	m := New(func(ctx context.Context, token string) (toggl.Me, error) {
		return toggl.Me{}, errors.New("connection refused")
	})
	m = typeText(t, m, "token")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !strings.Contains(m.View(), "connection refused") {
		t.Fatalf("view does not surface the underlying error:\n%s", m.View())
	}
}
