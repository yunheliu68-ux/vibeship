package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// RenderMetricsCard renders the metrics card showing cost, context usage, and rate limit pace.
func RenderMetricsCard(snap store.Snapshot, colors theme.Colors, width int) string {
	stale := time.Since(snap.Timestamp) > 30*time.Second

	costStr := fmt.Sprintf("$%.2f", snap.TotalCostUSD)
	ctxStr := fmt.Sprintf("%.0f%% ctx", snap.ContextUsedPct)

	paceEmoji := "🟢"
	paceLabel := "on pace"
	if snap.FiveHourUsedPct > 80 {
		paceEmoji = "🔴"
		paceLabel = "hot"
	} else if snap.FiveHourUsedPct > 50 {
		paceEmoji = "🟡"
		paceLabel = "warming"
	}

	limitStr := fmt.Sprintf("%.0f%% limit", snap.FiveHourUsedPct)

	content := fmt.Sprintf("%s  %s  %s %s",
		lipgloss.NewStyle().Foreground(colors.Primary).Bold(true).Render(costStr),
		lipgloss.NewStyle().Foreground(colors.Text).Render(ctxStr),
		lipgloss.NewStyle().Foreground(colors.Text).Render(limitStr),
		lipgloss.NewStyle().Render(paceEmoji+" "+paceLabel),
	)

	if stale && snap.SessionID != "" {
		content = lipgloss.NewStyle().Foreground(colors.Dim).Render(content)
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width - 2).
		Padding(0, 1)

	return style.Render(content)
}
