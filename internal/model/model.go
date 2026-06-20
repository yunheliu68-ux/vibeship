package model

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/ingest"
	"github.com/francis/vibeship/internal/layout"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

type overlay int

const (
	overlayNone overlay = iota
	overlaySkills
	overlayThink
	overlayHelp
)

// Model is the root Bubble Tea model for the Vibeship TUI.
type Model struct {
	store      *store.Store
	registry   *config.Registry
	theme      theme.ThemeName
	overlay    overlay
	width      int
	height     int
	projectDir string
	scope      *config.Scope

	// Latest data
	latestSnap   store.Snapshot
	recentEvents []store.TranscriptEvent

	// Animation tick
	tick int

	// Transcript poller stop channel
	pollStop chan struct{}

	// Decision log file
	decisionLog *os.File

	// Sidebar scroll offset
	sidebarScroll int
}

// New creates a new Model with the given store and registry. It opens the
// decision log, loads the project scope, starts the transcript poller, and
// returns a ready-to-run Model.
func New(st *store.Store, reg *config.Registry) *Model {
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()

	// Open decision log
	decisionPath := filepath.Join(home, ".vibeship", "decisions.jsonl")
	os.MkdirAll(filepath.Dir(decisionPath), 0755)
	dl, _ := os.OpenFile(decisionPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	// Load scope file from current working directory
	scope, _ := config.ParseScopeFile(wd)

	m := &Model{
		store:       st,
		registry:    reg,
		theme:       theme.Spaceship,
		overlay:     overlayNone,
		projectDir:  wd,
		scope:       scope,
		decisionLog: dl,
	}

	// Start transcript poller
	sessionsDir := filepath.Join(home, ".claude", "sessions")
	m.pollStop = ingest.StartTranscriptPoller(st, sessionsDir)

	return m
}

// Init returns the initial commands for the Bubble Tea runtime: a tick timer
// and an immediate data refresh.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		refreshDataCmd(m),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(66*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

type refreshMsg struct {
	snap   store.Snapshot
	events []store.TranscriptEvent
}

func refreshDataCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		snap, _ := m.store.LatestSnapshot()
		events, _ := m.store.RecentEvents(5 * time.Minute)
		return refreshMsg{snap: snap, events: events}
	}
}

// Update handles all Bubble Tea messages and delegates to the appropriate handler.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.tick++
		// Refresh data every 5 ticks (~300ms, matching statusline rate)
		if m.tick%5 == 0 {
			return m, tea.Batch(tickCmd(), refreshDataCmd(m))
		}
		return m, tickCmd()

	case refreshMsg:
		if msg.snap.SessionID != "" {
			m.latestSnap = msg.snap
		}
		if len(msg.events) > 0 {
			m.recentEvents = msg.events
			// Update active status on skills
			for _, e := range msg.events {
				if e.EventType == "skill" {
					for i := range m.registry.Skills {
						if m.registry.Skills[i].Name == e.Name {
							m.registry.Skills[i].Active = (e.Status == "active")
						}
					}
				}
			}
		}
		return m, nil

	case tea.QuitMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.overlay != overlayNone {
			m.overlay = overlayNone
			return m, nil
		}
		return m, tea.Quit
	case "t":
		if m.theme == theme.Spaceship {
			m.theme = theme.DJ
		} else {
			m.theme = theme.Spaceship
		}
	case "s":
		if m.overlay == overlaySkills {
			m.overlay = overlayNone
		} else {
			m.overlay = overlaySkills
		}
	case "d":
		if m.overlay == overlayThink {
			m.overlay = overlayNone
		} else {
			m.overlay = overlayThink
		}
	case "?":
		if m.overlay == overlayHelp {
			m.overlay = overlayNone
		} else {
			m.overlay = overlayHelp
		}
	case "r":
		return m, refreshDataCmd(m)
	case "esc":
		m.overlay = overlayNone
	}
	return m, nil
}

// View renders the current state of the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return "Starting Vibeship…\n"
	}

	colors := theme.ColorsFor(m.theme)

	// Render based on overlay state
	if m.overlay == overlayHelp {
		return m.renderHelp(colors)
	}

	return m.renderDashboard(colors)
}

// renderDashboard composes the main screen layout.
func (m *Model) renderDashboard(colors theme.Colors) string {
	// Top bar
	topBar := m.renderTopBar(colors)

	// Main content dimensions
	animW, animH, infoW, _ := layout.ViewportSize(m.width, m.height)
	infoH := animH

	// Left: animation area
	animView := m.renderAnimation(colors, animW, animH)

	// Right: info cards
	infoView := m.renderInfoPanel(colors, infoW, infoH)

	// If overlay active, render sidebar/overlay on top
	if m.overlay == overlaySkills {
		sidebarView := m.renderSidebar(colors, infoW, infoH)
		return lipgloss.JoinVertical(lipgloss.Top,
			topBar,
			lipgloss.JoinHorizontal(lipgloss.Top, animView, sidebarView),
		)
	}

	if m.overlay == overlayThink {
		thinkView := m.renderThink(colors, infoW, infoH)
		return lipgloss.JoinVertical(lipgloss.Top,
			topBar,
			lipgloss.JoinHorizontal(lipgloss.Top, animView, thinkView),
		)
	}

	// Default view
	main := lipgloss.JoinHorizontal(lipgloss.Top, animView, infoView)
	statusBar := layout.StatusBar("t=Theme  s=Skills  d=Think  ?=Help  q=Quit", m.width)

	return lipgloss.JoinVertical(lipgloss.Top, topBar, main, statusBar)
}

// Close shuts down the transcript poller and closes the decision log.
func (m *Model) Close() {
	if m.pollStop != nil {
		close(m.pollStop)
	}
	if m.decisionLog != nil {
		m.decisionLog.Close()
	}
}

// ---------------------------------------------------------------------------
// Stub renderers — to be filled in Tasks 8-12
// ---------------------------------------------------------------------------

func (m *Model) renderTopBar(colors theme.Colors) string {
	return fmt.Sprintf("Vibeship · %s  🚀", filepath.Base(m.projectDir))
}

func (m *Model) renderAnimation(colors theme.Colors, w, h int) string {
	return lipgloss.NewStyle().Width(w).Height(h).Render("animation area")
}

func (m *Model) renderInfoPanel(colors theme.Colors, w, h int) string {
	return lipgloss.NewStyle().Width(w).Height(h).Render("info panel")
}

func (m *Model) renderSidebar(colors theme.Colors, w, h int) string {
	return lipgloss.NewStyle().Width(w).Height(h).Render("sidebar")
}

func (m *Model) renderThink(colors theme.Colors, w, h int) string {
	return lipgloss.NewStyle().Width(w).Height(h).Render("think overlay")
}

func (m *Model) renderHelp(colors theme.Colors) string {
	return "Help overlay"
}
