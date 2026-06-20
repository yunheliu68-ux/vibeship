package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/theme"
)

// RenderTopBar renders the top bar with the project name, git branch, and theme icon.
func RenderTopBar(projectName, gitBranch string, thm theme.ThemeName, colors theme.Colors, width int) string {
	left := fmt.Sprintf("Vibeship · %s", projectName)
	if gitBranch != "" {
		left += fmt.Sprintf(" git:(%s)", gitBranch)
	}

	themeIcon := "🚀"
	if thm == theme.DJ {
		themeIcon = "🎵"
	}
	right := fmt.Sprintf("%s 曲速", themeIcon)

	style := lipgloss.NewStyle().
		Foreground(colors.Text).
		Width(width).
		Padding(0, 1)

	return style.Render(lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(colors.Primary).Render(left),
		lipgloss.NewStyle().Foreground(colors.Dim).Render(right),
	))
}
