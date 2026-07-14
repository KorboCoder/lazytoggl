package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui"
)

func main() {
	// Phase 1: the UI runs entirely on the in-memory mock — no API calls.
	// Phase 2 swaps in the real client behind the same toggl.Service.
	svc := toggl.NewMock()

	p := tea.NewProgram(ui.New(svc), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazytoggl:", err)
		os.Exit(1)
	}
}
