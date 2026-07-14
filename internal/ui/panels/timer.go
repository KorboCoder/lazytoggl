package panels

import (
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
	"github.com/KorboCoder/lazytoggl/internal/ui/theme"
)

// Timer renders the active-timer panel: the running entry with a live clock,
// or an idle hint when nothing is tracking.
func Timer(running *toggl.TimeEntry, projects map[int64]toggl.Project, now time.Time, width int, focused bool) string {
	iw := theme.PanelInnerWidth(width)
	var line string
	if running == nil {
		line = theme.MutedStyle.Render(theme.Truncate("no timer running — press enter to start", iw))
	} else {
		clock := theme.FormatClock(running.Elapsed(now))
		projName, projColor := projectLabel(running.ProjectID, projects)

		right := projName + "  " + clock
		desc := running.Description
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
		line = theme.RunningStyle.Render("▶ ") + desc + theme.Pad(gap) +
			projStyled + "  " + theme.RunningStyle.Render(clock)
	}
	return theme.Panel("Timer", line, width, 3, focused)
}

// projectLabel returns the display name and color for a project ID; "—" with
// no color when the entry has no project.
func projectLabel(projectID int64, projects map[int64]toggl.Project) (string, string) {
	if p, ok := projects[projectID]; ok && projectID != 0 {
		return p.Name, p.Color
	}
	return "—", ""
}
