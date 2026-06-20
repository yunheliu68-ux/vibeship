package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func renderSpaceship(snap store.Snapshot, tick int, colors theme.Colors, w, h int) string {
	if w < 1 || h < 1 {
		return "Terminal too small"
	}

	// Particle field (stars) in background
	particles := renderParticles(tick, w, h, snap)

	// Speedometer gauge in center
	gauge := renderGauge(snap, colors)

	// Warp speed line at bottom
	warpLine := renderWarpLine(snap, tick, w, colors)

	// Compose: particles as background, gauge centered, warp line at bottom
	gaugeW := lipgloss.Width(gauge)
	gaugeH := lipgloss.Height(gauge)
	centerX := (w - gaugeW) / 2
	centerY := (h - gaugeH) / 2

	result := particles
	result = placeString(result, gauge, centerX, centerY)
	result = placeString(result, warpLine, 0, h-1)

	return result
}

func renderParticles(tick int, w int, h int, snap store.Snapshot) string {
	// Guard against panics on very small terminals
	if w < 1 || h < 1 {
		return ""
	}

	// Generate pseudo-random star positions based on tick.
	// Density scales with output token count.
	density := 10 // base stars
	if snap.OutputTokens > 0 {
		density = int(math.Min(float64(snap.OutputTokens/2), 80))
		if density < 10 {
			density = 10
		}
	}

	// Create a grid of spaces.
	grid := make([][]rune, h)
	for y := range grid {
		grid[y] = []rune(strings.Repeat(" ", w))
	}

	// Place stars using a simple LCG seeded by tick and position.
	for i := 0; i < density; i++ {
		seed := (tick + i*7) % 1000
		x := (seed * 13) % w
		y := (seed * 17) % h

		// Star brightness cycles with tick.
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
	// Simple speedometer: a dial with rate value in the center.
	rate := snap.OutputTokens
	rateStr := fmt.Sprintf("%d t/s", rate)
	if rate == 0 {
		rateStr = "— t/s"
	}

	// Determine gauge colour based on five-hour budget usage.
	gaugeColor := colors.Primary
	if snap.FiveHourUsedPct > 80 {
		gaugeColor = colors.Danger
	} else if snap.FiveHourUsedPct > 50 {
		gaugeColor = colors.Warning
	}

	top := "╭─────────────────────╮"
	bottom := "╰─────────────────────╯"
	pointer := "│        ╱╲            │"
	pointer2 := "│       ╲╱           │"
	rateLine := fmt.Sprintf("│  %s  │",
		lipgloss.NewStyle().Foreground(gaugeColor).Bold(true).Render(centerStr(rateStr, 17)),
	)

	return lipgloss.NewStyle().Foreground(gaugeColor).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			top,
			pointer,
			rateLine,
			pointer2,
			bottom,
		),
	)
}

func renderWarpLine(snap store.Snapshot, tick int, w int, colors theme.Colors) string {
	if w < 1 {
		return ""
	}

	// Horizontal line of dashes scrolling left. Speed is proportional to token rate.
	speed := 1
	if snap.OutputTokens > 100 {
		speed = 3
	} else if snap.OutputTokens > 50 {
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
