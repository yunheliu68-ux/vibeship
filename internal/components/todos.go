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

	// Aggregate all todo events
	totalCreated := 0
	totalDone := 0
	for _, e := range events {
		if e.EventType == "todo" {
			totalCreated += e.TodoTotal
			totalDone += e.TodoDone
		}
	}

	if totalCreated == 0 && totalDone == 0 {
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

	totalCreated += totalDone // compensate: TaskUpdate completed also counts toward total
	pct := 0
	if totalCreated > 0 {
		pct = (totalDone * 100) / totalCreated
		if pct > 100 {
			pct = 100
		}
	}

	barWidth := max(0, width-6)
	filled := (pct * barWidth) / 100
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	content := fmt.Sprintf("%s %d%% · %d/%d done",
		bar, pct, totalDone, totalCreated)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(safeWidth).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📋 Todos") + "\n" + content,
	)
}
