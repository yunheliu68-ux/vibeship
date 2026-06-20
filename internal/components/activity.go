package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// RenderActivityCard renders the activity card showing currently active tools, skills, and MCP connections.
func RenderActivityCard(events []store.TranscriptEvent, colors theme.Colors, width int) string {
	var activeTool, activeSkill, activeMCP string

	for _, e := range events {
		switch e.EventType {
		case "tool_call":
			if e.Status == "active" {
				if e.Detail != "" {
					activeTool = fmt.Sprintf("◐ %s: %s", e.Name, e.Detail)
				} else {
					activeTool = fmt.Sprintf("◐ %s", e.Name)
				}
			}
		case "skill":
			if e.Status == "active" {
				activeSkill = fmt.Sprintf("🧩 %s", e.Name)
			}
		case "mcp":
			if e.Status == "active" {
				// Extract server name from mcp__server__tool
				parts := strings.SplitN(e.Name, "__", 3)
				if len(parts) >= 2 {
					activeMCP = fmt.Sprintf("🔌 %s", parts[1])
				}
			}
		}
	}

	// Count completed tools
	doneCount := 0
	for _, e := range events {
		if e.EventType == "tool_call" && e.Status == "done" {
			doneCount++
		}
	}

	var lines []string
	if activeTool != "" {
		lines = append(lines, activeTool)
	}
	if activeSkill != "" {
		lines = append(lines, activeSkill)
	}
	if activeMCP != "" {
		lines = append(lines, activeMCP)
	}
	if doneCount > 0 {
		lines = append(lines, fmt.Sprintf("✓ %d completed", doneCount))
	}
	if len(lines) == 0 {
		lines = append(lines, "—")
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width - 2).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("⚡ Activity") + "\n" +
			strings.Join(lines, "\n"),
	)
}
