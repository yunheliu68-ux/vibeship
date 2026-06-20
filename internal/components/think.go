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

// RenderThinkPanel renders the Thinking Co-pilot overlay showing the current
// scope, recent file changes, auto-generated check questions, and a text input
// prompt at the bottom. thinkInput is the currently typed question text;
// thinkSubmitted is the last submitted question shown prominently.
func RenderThinkPanel(scope *config.Scope, events []store.TranscriptEvent, colors theme.Colors, w, h int, thinkInput, thinkSubmitted string) string {
	// Guard against panics on very small terminals
	if w < 20 || h < 10 {
		return "Terminal too small — please resize to at least 20x10"
	}

	safeW := max(0, w-4)

	var sections []string

	// Scope section
	sections = append(sections, lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📋 当前 Scope"))
	if scope == nil {
		sections = append(sections, lipgloss.NewStyle().Foreground(colors.Dim).Render("  (未找到 SCOPE.md)"))
	} else {
		for _, g := range scope.Goals {
			sections = append(sections, "  • "+g)
		}
		if len(scope.Files) > 0 {
			sections = append(sections, lipgloss.NewStyle().Foreground(colors.Dim).Render("  范围文件:"))
			for _, f := range scope.Files {
				sections = append(sections, "    "+f)
			}
		}
	}

	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📊 最近改动"))
	// Show recent Write/Edit events
	shown := 0
	for _, e := range events {
		if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
			if e.Detail != "" {
				sections = append(sections, fmt.Sprintf("  ✏️ %s", e.Detail))
				shown++
				if shown >= 5 {
					break
				}
			}
		}
	}
	if shown == 0 {
		sections = append(sections, "  —")
	}

	sections = append(sections, "")
	sections = append(sections, strings.Repeat("─", safeW))

	// Check questions
	sections = append(sections, lipgloss.NewStyle().Bold(true).Foreground(colors.Warning).Render("⚡ 快速检查"))
	questions := rules.GenerateCheckQuestions(scope, events)
	if len(questions) == 0 {
		sections = append(sections, "  一切正常 ✓")
	} else {
		for i, q := range questions {
			sections = append(sections, fmt.Sprintf("  %d. %s", i+1, q))
		}
	}

	sections = append(sections, "")
	sections = append(sections, strings.Repeat("─", safeW))

	// Display last submitted question prominently
	if thinkSubmitted != "" {
		sections = append(sections, lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📤 已发送"))
		sections = append(sections, lipgloss.NewStyle().Foreground(colors.Text).Render("  \""+thinkSubmitted+"\""))
		sections = append(sections, "")
	}

	// Text input area
	sections = append(sections, lipgloss.NewStyle().Foreground(colors.Dim).Render("💬 输入你的问题，Enter 发送，Esc 退出"))
	cursor := "▸"
	if thinkInput == "" {
		sections = append(sections, lipgloss.NewStyle().Foreground(colors.Primary).Render(cursor+" _"))
	} else {
		sections = append(sections, lipgloss.NewStyle().Foreground(colors.Primary).Render(cursor+" "+thinkInput+"█"))
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(w - 2).
		Height(h).
		Padding(0, 1)

	return style.Render(strings.Join(sections, "\n"))
}
