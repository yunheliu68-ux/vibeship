package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/rules"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func RenderSidebar(reg *config.Registry, events []store.TranscriptEvent, colors theme.Colors, w, h int) string {
	// Guard against panics on very small terminals
	if w < 20 || h < 10 {
		return "Terminal too small"
	}

	safeW := max(0, w-4)

	var lines []string

	// Skills section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("🧩 Skills"))
	lines = append(lines, strings.Repeat("─", safeW))

	currentCategory := ""
	for _, sk := range reg.Skills {
		if sk.Category != currentCategory {
			currentCategory = sk.Category
			lines = append(lines, lipgloss.NewStyle().Foreground(colors.Dim).Render(currentCategory+":"))
		}
		marker := "  —"
		if sk.Active {
			marker = "  ✓"
		}
		lines = append(lines, fmt.Sprintf("%s %s", marker, sk.Name))
	}

	// Plugins section
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("🔌 Plugins (enabled)"))
	lines = append(lines, strings.Repeat("─", safeW))

	for _, p := range reg.Plugins {
		if p.Active {
			lines = append(lines, fmt.Sprintf("  ✓ %s", p.Name))
		}
	}

	// Separator
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", safeW))

	// Recommendation
	rec := rules.RecommendSkill(events)
	if rec != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colors.Warning).Render("💡 Recommended"))
		lines = append(lines, fmt.Sprintf("  → %s", rec))
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(w-2).
		Height(h).
		Padding(0, 1)

	// Truncate if too long
	content := strings.Join(lines, "\n")
	return style.Render(content)
}
