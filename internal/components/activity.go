package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// RenderActivityCard renders the activity card showing active skills, MCP, and tools.
// Priority: skills > MCP > tools.
func RenderActivityCard(events []store.TranscriptEvent, recommendation string, colors theme.Colors, width int) string {
	// Collect unique active items
	activeSkills := make(map[string]bool)
	activeMCPs := make(map[string]bool)
	activeTools := make(map[string]string) // toolName -> detail
	completedCount := 0

	for _, e := range events {
		switch e.EventType {
		case "skill":
			if e.Status == "active" {
				activeSkills[e.Name] = true
			}
		case "mcp":
			if e.Status == "active" {
				parts := strings.SplitN(e.Name, "__", 3)
				server := e.Name
				if len(parts) >= 2 {
					server = parts[1]
				}
				activeMCPs[server] = true
			}
		case "tool_call":
			if e.Status == "active" {
				activeTools[e.Name] = e.Detail
			} else if e.Status == "done" {
				completedCount++
			}
		}
	}

	var lines []string

	// Skills first (most important — what the user cares about)
	for name := range activeSkills {
		lines = append(lines, fmt.Sprintf("🧩 %s", name))
	}

	// MCP servers second
	for name := range activeMCPs {
		lines = append(lines, fmt.Sprintf("🔌 %s", name))
	}

	// Tools third — show count by type if many, otherwise list
	if len(activeSkills) == 0 && len(activeMCPs) == 0 && len(activeTools) > 0 {
		toolCounts := make(map[string]int)
		for name := range activeTools {
			toolCounts[name]++
		}
		var parts []string
		for name, count := range toolCounts {
			if count > 1 {
				parts = append(parts, fmt.Sprintf("%s×%d", name, count))
			} else {
				parts = append(parts, name)
			}
		}
		lines = append(lines, "◐ "+strings.Join(parts, " · "))
	} else if len(activeSkills) == 0 && len(activeMCPs) == 0 {
		if completedCount > 0 {
			lines = append(lines, fmt.Sprintf("✓ %d tools completed", completedCount))
		} else {
			lines = append(lines, "—")
		}
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width-2).
		Padding(0, 1)

	if recommendation != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(colors.Warning).Render("💡 "+recommendation))
	}

	header := lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("⚡ Activity")
	return style.Render(header + "\n" + strings.Join(lines, "\n"))
}
