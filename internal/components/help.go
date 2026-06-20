package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/theme"
)

func RenderHelp(colors theme.Colors, w, h int) string {
	helpLines := []string{
		"Vibeship — Vibe Coding Cockpit",
		"",
		"Keys:",
		"  t      切换主题 (Spaceship ↔ DJ)",
		"  s      全部 Skills & 插件 + 建议",
		"  d      思路副驾 (Scope检查 / 头脑风暴)",
		"  r      强制刷新",
		"  ?      此帮助",
		"  q      退出",
		"",
		"Setup:",
		"  将 vibeship collect 配置为 Claude Code statusline pipe:",
		"  \"statusLine\": { \"command\": \"vibeship collect | ...\" }",
		"",
		"Scope (可选):",
		"  在项目根目录创建 SCOPE.md，Vibeship 自动读取并做跑偏检查",
	}

	content := lipgloss.JoinVertical(lipgloss.Left, helpLines...)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colors.Primary).
		Width(w - 4).
		Padding(1, 2).
		Render(content)
}
