package theme

import "github.com/charmbracelet/lipgloss"

// ThemeName identifies which visual theme to use.
type ThemeName string

const (
	Spaceship ThemeName = "spaceship"
	DJ        ThemeName = "dj"
)

// Colors holds the complete colour palette for a theme.
type Colors struct {
	Background lipgloss.Color
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Text       lipgloss.Color
	Dim        lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Danger     lipgloss.Color
}

// SpaceshipColors is the default theme inspired by a dark space bridge.
var SpaceshipColors = Colors{
	Background: lipgloss.Color("#0a0e1a"),
	Primary:    lipgloss.Color("#00d4ff"),
	Secondary:  lipgloss.Color("#ff6600"),
	Text:       lipgloss.Color("#c8d6e5"),
	Dim:        lipgloss.Color("#576574"),
	Success:    lipgloss.Color("#00ff88"),
	Warning:    lipgloss.Color("#ffaa00"),
	Danger:     lipgloss.Color("#ff4444"),
}

// DJColors is an alternative theme with a purple base and neon accents.
var DJColors = Colors{
	Background: lipgloss.Color("#1a0a2e"),
	Primary:    lipgloss.Color("#00ff88"),
	Secondary:  lipgloss.Color("#ff00ff"),
	Text:       lipgloss.Color("#e8d5f5"),
	Dim:        lipgloss.Color("#6b5b7a"),
	Success:    lipgloss.Color("#00ff88"),
	Warning:    lipgloss.Color("#ffaa00"),
	Danger:     lipgloss.Color("#ff4444"),
}

// ColorsFor returns the Colors for the given ThemeName. Falls back to
// SpaceshipColors when the name is unknown.
func ColorsFor(t ThemeName) Colors {
	switch t {
	case DJ:
		return DJColors
	default:
		return SpaceshipColors
	}
}
