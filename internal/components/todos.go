package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// RenderTodosCard renders the todos card with a progress bar showing completion status.
func RenderTodosCard(events []store.TranscriptEvent, colors theme.Colors, width int) string {
	// Guard against panics on very small terminals
	if width < 4 {
		return "Terminal too small"
	}

	safeWidth := max(0, width-2)

	// Find the most recent todo event
	var latestTodo store.TranscriptEvent
	for _, e := range events {
		if e.EventType == "todo" {
			latestTodo = e
			break
		}
	}

	if latestTodo.TodoTotal == 0 {
		content := "—"
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Width(safeWidth).
			Padding(0, 1)
		return style.Render(
			lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📋 Todos") + "\n" + content,
		)
	}

	pct := 0
	if latestTodo.TodoTotal > 0 {
		pct = (latestTodo.TodoDone * 100) / latestTodo.TodoTotal
	}

	barWidth := max(0, width-6)
	filled := (pct * barWidth) / 100
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	content := fmt.Sprintf("%s %d%% · %d/%d done",
		bar, pct, latestTodo.TodoDone, latestTodo.TodoTotal)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(safeWidth).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📋 Todos") + "\n" + content,
	)
}
