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

	// Prefer most recent TodoWrite full snapshot (events are new→old)
	for _, e := range events {
		if e.EventType == "todo_snapshot" {
			return renderTodoBar(e.TodoDone, e.TodoTotal, colors, safeWidth)
		}
	}

	// Fallback: incremental TaskCreate/TaskUpdate in window
	created, done := 0, 0
	for _, e := range events {
		if e.EventType == "todo" {
			created += e.TodoTotal
			done += e.TodoDone
		}
	}

	if created == 0 && done == 0 {
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

	return renderTodoBar(done, created, colors, safeWidth)
}

func renderTodoBar(done, total int, colors theme.Colors, width int) string {
	pct := 0
	if total > 0 {
		pct = (done * 100) / total
		if pct > 100 {
			pct = 100
		}
	}

	barWidth := max(0, width-6)
	filled := (pct * barWidth) / 100
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	content := fmt.Sprintf("%s %d%% · %d/%d done", bar, pct, done, total)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📋 Todos") + "\n" + content,
	)
}
