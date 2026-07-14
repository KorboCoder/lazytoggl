package panels

import (
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// Timer renders the active-timer panel: the running entry with a live clock,
// a paused entry with a frozen clock (Toggl has no native pause — pausing
// stops the entry, but the panel keeps showing it until resumed), or an
// idle hint when neither is set.
func Timer(running, paused *toggl.TimeEntry, projects map[int64]toggl.Project, now time.Time, width int, focused bool) string {
	iw := theme.PanelInnerWidth(width)
	var line string
	switch {
	case running != nil:
		line = renderTimerLine(*running, "▶ ", theme.RunningStyle, projects, now, iw)
	case paused != nil:
		line = renderTimerLine(*paused, "⏸ ", theme.PausedStyle, projects, now, iw)
	default:
		line = theme.MutedStyle.Render(theme.Truncate("no timer running — press c to start", iw))
	}
	return theme.Panel("Timer", line, width, 3, focused)
}

func renderTimerLine(e toggl.TimeEntry, icon string, accent lipgloss.Style, projects map[int64]toggl.Project, now time.Time, iw int) string {
	clock := theme.FormatClock(e.Elapsed(now))
	projName, projColor := projectLabel(e.ProjectID, projects)

	right := projName + "  " + clock
	desc := e.Description
	if desc == "" {
		desc = "(no description)"
	}
	descW := iw - 2 - len([]rune(right)) - 2
	desc = theme.Truncate(desc, descW)

	gap := iw - 2 - len([]rune(desc)) - len([]rune(right))
	var projStyled string
	if projColor != "" {
		projStyled = lipgloss.NewStyle().Foreground(lipgloss.Color(projColor)).Render(projName)
	} else {
		projStyled = theme.MutedStyle.Render(projName)
	}
	return accent.Render(icon) + desc + theme.Pad(gap) + projStyled + "  " + accent.Render(clock)
}

// projectLabel returns the display name and color for a project ID; "—" with
// no color when the entry has no project.
func projectLabel(projectID int64, projects map[int64]toggl.Project) (string, string) {
	if p, ok := projects[projectID]; ok && projectID != 0 {
		return p.Name, p.Color
	}
	return "—", ""
}
