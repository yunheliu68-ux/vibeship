package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// renderDJ composes the DJ visualizer: top spectrum (input tokens),
// breathing halo with rate number in the center, and bottom spectrum
// (output tokens).
func renderDJ(snap store.Snapshot, tick int, colors theme.Colors, w, h int) string {
	if w < 1 || h < 1 {
		return "Terminal too small"
	}

	specH := (h - 5) / 2 // split remaining height between top and bottom spectrum
	if specH < 3 {
		specH = 3
	}

	// Top spectrum: input tokens
	inputSpec := renderSpectrum(snap.InputTokens, tick, w, specH, colors, true)

	// Breathing halo + rate number in center
	halo := renderHalo(snap, tick, colors)

	// Bottom spectrum: output tokens
	outputSpec := renderSpectrum(snap.OutputTokens, tick, w, specH, colors, false)

	return lipgloss.JoinVertical(lipgloss.Center, inputSpec, halo, outputSpec)
}

var columnChars = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// renderSpectrum renders a column-based spectrum visualization of token
// activity. Column height is proportional to token volume, animated with a
// sine-based tick wave. Characters cycle through unicode block chars.
func renderSpectrum(tokens int64, tick int, w, h int, colors theme.Colors, isInput bool) string {
	columns := w / 2
	if columns < 8 {
		columns = 8
	}

	// Base activity level from tokens
	level := 0
	if tokens > 200 {
		level = 7
	} else if tokens > 100 {
		level = 5
	} else if tokens > 50 {
		level = 3
	} else if tokens > 10 {
		level = 2
	}

	var rows []string
	for y := h - 1; y >= 0; y-- {
		var line string
		for x := 0; x < columns; x++ {
			// Animate: columns alternate with tick
			colLevel := level
			// Add variation based on position and tick
			wave := int(math.Sin(float64(x+tick/3)*0.5)*2) + 2
			colLevel = (colLevel + wave) / 2
			if colLevel < 0 {
				colLevel = 0
			}
			if colLevel > 7 {
				colLevel = 7
			}

			barHeight := (colLevel * h) / 8
			if y < barHeight {
				// Colored gradient: hotter at top
				char := "█"
				if colLevel < 3 {
					char = "▄"
				}
				line += char
			} else {
				line += " "
			}
		}
		rows = append(rows, line)
	}

	label := "output"
	if isInput {
		label = "input"
	}
	labelLine := lipgloss.NewStyle().Foreground(colors.Dim).Render(label + " ▔▔▔▔▔ " + label)

	return lipgloss.JoinVertical(lipgloss.Center, labelLine, strings.Join(rows, "\n"))
}

// renderHalo renders a breathing halo around the center rate number.
// The breath period scales with budget consumption percentage, using
// a sine-based radius oscillation. Color shifts with state:
//   - green (idle): normal breathing, color is Primary
//   - yellow (active): faster breathing, color is Warning
//   - red (peak): rapid breathing, color is Danger
func renderHalo(snap store.Snapshot, tick int, colors theme.Colors) string {
	// Breathing halo: expands and contracts
	// Breath rate scales with budget consumption %
	breathPeriod := 60 // ticks per breath cycle at normal pace
	if snap.FiveHourUsedPct > 80 {
		breathPeriod = 20 // rapid breathing
	} else if snap.FiveHourUsedPct > 50 {
		breathPeriod = 40
	}

	phase := float64(tick%breathPeriod) / float64(breathPeriod)
	// Use sine to create smooth breathing
	scale := math.Sin(phase * 2 * math.Pi)
	radius := int(2 + scale*2) // radius varies from 2 to 6

	haloColor := colors.Primary
	if snap.FiveHourUsedPct > 80 {
		haloColor = colors.Danger
	} else if snap.FiveHourUsedPct > 50 {
		haloColor = colors.Warning
	}

	// Build concentric halo rings
	rate := snap.OutputTokens
	rateStr := fmt.Sprintf("%d t/s", rate)
	if rate == 0 {
		rateStr = "— t/s"
	}

	// Center the rate number
	centerContent := lipgloss.NewStyle().
		Foreground(haloColor).
		Bold(true).
		Render(fmt.Sprintf("╭─────────╮\n│ %s │\n╰─────────╯", centerStr(rateStr, 7)))

	// Simple halo: just the center with breathing effect
	// The "breathing" is expressed through the rate number color brightness
	return lipgloss.NewStyle().
		Padding(radius/2, radius/2).
		Render(centerContent)
}
