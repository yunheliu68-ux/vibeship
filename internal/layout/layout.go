package layout

import (
	"github.com/charmbracelet/lipgloss"
)

// Viewport constants
const (
	AnimationWidthPct = 0.70
	InfoPanelWidthPct = 0.30
)

// ViewportSize calculates the dimensions for the animation area and info panel
// based on the terminal size. It subtracts 2 rows for the top bar and bottom
// status bar. All dimensions are clamped to a minimum of 0.
func ViewportSize(termWidth, termHeight int) (animW, animH, infoW, infoH int) {
	contentHeight := max(0, termHeight-2)
	animW = int(float64(termWidth) * AnimationWidthPct)
	infoW = max(0, termWidth-animW)
	animH = contentHeight
	infoH = contentHeight
	return
}

// Card renders a bordered card with a title and content.
// When active is true, the border glows green; otherwise it is grey.
func Card(title string, content string, width int, active bool) string {
	borderColor := lipgloss.Color("#444444")
	if active {
		borderColor = lipgloss.Color("#00ff88")
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width-2).
		Padding(0, 1)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		content,
	))
}

// StatusBar renders a dimmed keybinding hint bar at the bottom of the screen.
func StatusBar(keys string, width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#576574")).
		Width(width).
		Render(keys)
}
