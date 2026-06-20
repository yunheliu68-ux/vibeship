package components

import (
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// RenderAnimation dispatches to the correct animation renderer based on the active theme.
func RenderAnimation(thm theme.ThemeName, snap store.Snapshot, rate int64, tick int, colors theme.Colors, w, h int) string {
	switch thm {
	case theme.DJ:
		return renderDJ(snap, rate, tick, colors, w, h)
	default:
		return renderSpaceship(snap, rate, tick, colors, w, h)
	}
}
