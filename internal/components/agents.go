package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// RenderAgentsCard renders the agents card showing agent status with icons, model info, and duration.
func RenderAgentsCard(events []store.TranscriptEvent, colors theme.Colors, width int) string {
	var lines []string
	for _, e := range events {
		if e.EventType != "agent" {
			continue
		}
		icon := "◐"
		if e.Status == "done" {
			icon = "✓"
		} else if e.Status == "queued" {
			icon = "○"
		}
		modelStr := ""
		if e.Detail != "" {
			modelStr = fmt.Sprintf("[%s]", e.Detail)
		}
		durStr := ""
		if e.DurationMs > 0 {
			d := time.Duration(e.DurationMs) * time.Millisecond
			durStr = fmt.Sprintf("⏱ %s", d.Round(time.Second))
		}
		lines = append(lines, fmt.Sprintf("%s %s %s  %s",
			icon, e.Name, modelStr, durStr))
	}
	if len(lines) == 0 {
		lines = append(lines, "—")
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width-2).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("🤖 Agents") + "\n" +
			strings.Join(lines, "\n"),
	)
}
