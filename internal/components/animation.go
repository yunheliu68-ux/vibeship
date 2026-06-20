package components

import (
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

// RenderAnimation dispatches to the correct animation renderer based on the
// active theme. The DJ theme animation will be implemented in a future task.
func RenderAnimation(thm theme.ThemeName, snap store.Snapshot, tick int, colors theme.Colors, w, h int) string {
	switch thm {
	case theme.DJ:
		return renderDJ(snap, tick, colors, w, h)
	default:
		return renderSpaceship(snap, tick, colors, w, h)
	}
}
