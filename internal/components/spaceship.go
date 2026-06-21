package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func renderSpaceship(snap store.Snapshot, rate int64, tick int, colors theme.Colors, w, h int) string {
	if w < 20 || h < 10 {
		return "Terminal too small"
	}
	_ = rate // output token delta is not a valid rate source — drive by context usage instead

	pct := snap.ContextUsedPct // 0..100, real and grows with session

	particles := renderParticles(pct, tick, w, h)
	gauge := renderGauge(snap, colors)
	warpLine := renderWarpLine(pct, tick, w, colors)

	centerY := (h - lipgloss.Height(gauge)) / 2
	centerX := (w - lipgloss.Width(gauge)) / 2
	if centerY < 0 {
		centerY = 0
	}
	if centerX < 0 {
		centerX = 0
	}
	result := particles
	result = placeString(result, gauge, centerX, centerY)
	result = placeString(result, warpLine, 0, h-1)
	return result
}

func renderParticles(pct float64, tick int, w, h int) string {
	density := 10 + int(pct*0.7) // 10..~80, denser as context fills
	if density > 80 {
		density = 80
	}
	grid := make([][]rune, h)
	for y := range grid {
		grid[y] = []rune(strings.Repeat(" ", w))
	}
	for i := 0; i < density; i++ {
		seed := (tick + i*7) % 1000
		x := (seed * 13) % w
		y := (seed * 17) % h
		brightness := (tick + i) % 3
		var star rune
		switch brightness {
		case 0:
			star = '·'
		case 1:
			star = '✦'
		case 2:
			star = '✧'
		}
		if x >= 0 && x < w && y >= 0 && y < h {
			grid[y][x] = star
		}
	}
	var lines []string
	for _, row := range grid {
		lines = append(lines, string(row))
	}
	return strings.Join(lines, "\n")
}

func renderGauge(snap store.Snapshot, colors theme.Colors) string {
	valStr := fmt.Sprintf("%.0f%% ctx", snap.ContextUsedPct)

	gaugeColor := colors.Primary
	if snap.ContextUsedPct > 85 || snap.FiveHourUsedPct > 80 {
		gaugeColor = colors.Danger
	} else if snap.ContextUsedPct > 60 || snap.FiveHourUsedPct > 50 {
		gaugeColor = colors.Warning
	}

	top := "╭─────────────────────╮"
	bottom := "╰─────────────────────╯"
	pointer := "│        ╱╲            │"
	rateLine := fmt.Sprintf("│  %s  │", lipgloss.NewStyle().Foreground(gaugeColor).Bold(true).Render(centerStr(valStr, 19)))
	pointer2 := "│       ╲╱           │"

	return lipgloss.NewStyle().Foreground(gaugeColor).Render(
		lipgloss.JoinVertical(lipgloss.Center, top, pointer, rateLine, pointer2, bottom),
	)
}

func renderWarpLine(pct float64, tick int, w int, colors theme.Colors) string {
	if w < 1 {
		return ""
	}
	speed := 1
	if pct > 60 {
		speed = 3
	} else if pct > 30 {
		speed = 2
	}
	offset := (tick * speed) % 8
	var chars []rune
	for i := 0; i < w; i++ {
		phase := (i + offset) % 8
		switch phase {
		case 0, 1:
			chars = append(chars, '═')
		case 2:
			chars = append(chars, '—')
		default:
			chars = append(chars, ' ')
		}
	}
	return lipgloss.NewStyle().Foreground(colors.Primary).Render(string(chars))
}

func centerStr(s string, w int) string {
	pad := (w - len(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s + strings.Repeat(" ", pad)
}

func placeString(bg string, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fgLine := range fgLines {
		targetY := y + i
		if targetY < 0 || targetY >= len(bgLines) {
			continue
		}
		bgLine := []rune(bgLines[targetY])
		fgRunes := []rune(fgLine)
		for j, r := range fgRunes {
			targetX := x + j
			if targetX < 0 || targetX >= len(bgLine) {
				continue
			}
			if r != ' ' {
				bgLine[targetX] = r
			}
		}
		bgLines[targetY] = string(bgLine)
	}
	return strings.Join(bgLines, "\n")
}
