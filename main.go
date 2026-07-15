package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/config"
	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui"
	"github.com/KorboCoder/lazytoggl/internal/ui/onboard"
)

func main() {
	useMock := flag.Bool("mock", false, "run against in-memory mock data instead of the real Toggl API")
	flag.Parse()

	var svc toggl.Service
	if *useMock {
		svc = toggl.NewMock()
	} else {
		client, err := setupClient()
		if err != nil {
			fmt.Fprintln(os.Stderr, "lazytoggl:", err)
			os.Exit(1)
		}
		if client == nil {
			return // onboarding was cancelled
		}
		svc = client
	}

	p := tea.NewProgram(ui.New(svc), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazytoggl:", err)
		os.Exit(1)
	}
}

// setupClient gets a working, workspace-scoped toggl.Client: onboarding on
// first run, otherwise the cached token and workspace id (see
// docs/ARCHITECTURE.md — this is what lets normal launches skip /me). A nil
// client with a nil error means the user cancelled onboarding.
func setupClient() (*toggl.Client, error) {
	token, err := config.LoadToken()
	if err != nil {
		return nil, fmt.Errorf("reading saved token: %w", err)
	}

	if token == "" {
		token, me, cancelled, err := runOnboarding()
		if err != nil {
			return nil, err
		}
		if cancelled {
			return nil, nil
		}
		if err := config.SaveToken(token); err != nil {
			return nil, fmt.Errorf("saving token: %w", err)
		}
		if err := config.Save(config.Config{WorkspaceID: me.DefaultWorkspaceID}); err != nil {
			return nil, fmt.Errorf("saving config: %w", err)
		}
		client := toggl.NewClient(token)
		client.SetWorkspaceID(me.DefaultWorkspaceID)
		return client, nil
	}

	client := toggl.NewClient(token)
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("reading cached config: %w", err)
	}
	if cfg.WorkspaceID != 0 {
		client.SetWorkspaceID(cfg.WorkspaceID)
		return client, nil
	}

	// A token exists (e.g. via $TOGGL_API_TOKEN) but we've never cached its
	// workspace id — one-time /me call to discover and cache it.
	me, err := client.Me(context.Background())
	if err != nil {
		return nil, fmt.Errorf("validating token: %w", err)
	}
	if err := config.Save(config.Config{WorkspaceID: me.DefaultWorkspaceID}); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}
	return client, nil
}

func runOnboarding() (token string, me toggl.Me, cancelled bool, err error) {
	validate := func(ctx context.Context, token string) (toggl.Me, error) {
		return toggl.NewClient(token).Me(ctx)
	}
	p := tea.NewProgram(onboard.New(validate))
	finalModel, err := p.Run()
	if err != nil {
		return "", toggl.Me{}, false, fmt.Errorf("running onboarding: %w", err)
	}
	m := finalModel.(onboard.Model)
	if m.Cancelled {
		return "", toggl.Me{}, true, nil
	}
	return m.Token, m.Me, false, nil
}
