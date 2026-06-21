package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/francis/vibeship/internal/components"
	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/ingest"
	"github.com/francis/vibeship/internal/layout"
	"github.com/francis/vibeship/internal/rules"
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

	// Think panel text input state
	thinkInput     string // current typed question text
	thinkSubmitted string // last submitted question (shown in panel)

	// Rate tracking
	prevOutputTokens int64
	prevSnapshotTime time.Time
	outputRate       int64 // tokens per second
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
	projectsDir := filepath.Join(home, ".claude", "projects")
	m.pollStop = ingest.StartTranscriptPoller(st, projectsDir)

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
		snap, err := m.store.LatestSnapshot()
		if err != nil {
			snap = store.Snapshot{} // empty snapshot is fine
		}
		events, err := m.store.RecentEvents(5 * time.Minute)
		if err != nil {
			events = nil
		}
		return refreshMsg{snap: snap, events: events}
	}
}

// Update handles all Bubble Tea messages and delegates to the appropriate handler.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When think overlay is active, handle text input
		if m.overlay == overlayThink {
			return m.handleThinkInput(msg)
		}
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
			// Calculate output token rate
			if m.prevOutputTokens > 0 && msg.snap.OutputTokens > m.prevOutputTokens {
				elapsed := msg.snap.Timestamp.Sub(m.prevSnapshotTime).Seconds()
				if elapsed > 0.1 {
					m.outputRate = int64(float64(msg.snap.OutputTokens-m.prevOutputTokens) / elapsed)
				}
			}
			m.prevOutputTokens = msg.snap.OutputTokens
			m.prevSnapshotTime = msg.snap.Timestamp
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

// handleThinkInput processes key events when the think overlay is active,
// allowing the user to type and submit a question.
func (m *Model) handleThinkInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear input and dismiss overlay
		m.thinkInput = ""
		m.overlay = overlayNone
		return m, nil

	case "d":
		// Toggle overlay off
		m.thinkInput = ""
		m.overlay = overlayNone
		return m, nil

	case "enter":
		if strings.TrimSpace(m.thinkInput) == "" {
			return m, nil
		}
		// Write to decision log
		m.writeDecision(m.thinkInput)
		// Show as submitted
		m.thinkSubmitted = m.thinkInput
		m.thinkInput = ""
		return m, nil

	case "backspace":
		if len(m.thinkInput) > 0 {
			// Handle multi-byte runes correctly
			runes := []rune(m.thinkInput)
			m.thinkInput = string(runes[:len(runes)-1])
		}
		return m, nil

	case " ":
		m.thinkInput += " "
		return m, nil
	}

	// For all other single-character keys, append to input
	// tea.KeyMsg.String() returns the character for printable keys
	if len(msg.String()) == 1 {
		m.thinkInput += msg.String()
	} else if len(msg.Runes) == 1 {
		m.thinkInput += string(msg.Runes[0])
	}

	return m, nil
}

// writeDecision writes a decision log entry to decisions.jsonl.
func (m *Model) writeDecision(question string) {
	if m.decisionLog == nil {
		return
	}
	sessionID := ""
	if m.latestSnap.SessionID != "" {
		sessionID = m.latestSnap.SessionID
	}
	entry := decisionEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Question:  question,
		SessionID: sessionID,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	m.decisionLog.Write(append(data, '\n'))
}

// decisionEntry is the JSON structure written to the decision log.
type decisionEntry struct {
	Timestamp string `json:"timestamp"`
	Question  string `json:"question"`
	SessionID string `json:"session_id"`
}

// View renders the current state of the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return "Starting Vibeship…\n"
	}
	if m.width < 20 || m.height < 10 {
		return "Terminal too small — please resize to at least 20x10\n"
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
		// Bottom-right floating popup (not full overlay)
		popupW := infoW
		if popupW > 40 {
			popupW = 40
		}
		if popupW < 28 {
			popupW = 28
		}
		popupH := 16
		thinkView := m.renderThink(colors, popupW, popupH)

		// Render normal dashboard with think popup placed bottom-right
		main := lipgloss.JoinHorizontal(lipgloss.Top, animView, infoView)
		mainWithPopup := lipgloss.Place(m.width, animH,
			lipgloss.Right, lipgloss.Bottom,
			thinkView,
			lipgloss.WithWhitespaceChars("│"),
		)
		_ = main
		_ = mainWithPopup

		// Manual composition: draw popup over the bottom-right of the main content
		mainLines := strings.Split(main, "\n")
		popupLines := strings.Split(thinkView, "\n")
		result := make([]string, len(mainLines))
		copy(result, mainLines)
		for i := 0; i < len(popupLines) && i < len(result); i++ {
			targetRow := len(result) - len(popupLines) + i
			if targetRow < 0 {
				continue
			}
			if targetRow >= len(result) {
				break
			}
			row := []rune(result[targetRow])
			popupRow := []rune(popupLines[i])
			padding := m.width - len(popupRow)
			if padding < 0 {
				padding = 0
			}
			startCol := padding
			for j := 0; j < len(popupRow); j++ {
				col := startCol + j
				if col < len(row) && popupRow[j] != ' ' {
					row[col] = popupRow[j]
				}
			}
			result[targetRow] = string(row)
		}

		statusBar := layout.StatusBar("d=close  Enter=send  Esc=dismiss", m.width)
		return lipgloss.JoinVertical(lipgloss.Top, topBar, strings.Join(result, "\n"), statusBar)
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
	return components.RenderTopBar(
		filepath.Base(m.projectDir), "", m.theme, colors, m.width,
	)
}

func (m *Model) renderAnimation(colors theme.Colors, w, h int) string {
	if m.latestSnap.SessionID == "" {
		msg := "Waiting for Claude Code…\n\nStart Claude Code to see data."
		return lipgloss.NewStyle().
			Foreground(colors.Dim).
			Width(w).
			Height(h).
			Align(lipgloss.Center, lipgloss.Center).
			Render(msg)
	}
	return components.RenderAnimation(m.theme, m.latestSnap, m.outputRate, m.tick, colors, w, h)
}

func (m *Model) renderInfoPanel(colors theme.Colors, w, h int) string {
	cardW := w
	metricsView := components.RenderMetricsCard(m.latestSnap, colors, cardW)
	rec := rules.RecommendSkill(m.recentEvents)
	activityView := components.RenderActivityCard(m.recentEvents, rec, colors, cardW)
	agentsView := components.RenderAgentsCard(m.recentEvents, colors, cardW)
	todosView := components.RenderTodosCard(m.recentEvents, colors, cardW)
	return lipgloss.JoinVertical(lipgloss.Top,
		metricsView,
		activityView,
		agentsView,
		todosView,
	)
}

func (m *Model) renderSidebar(colors theme.Colors, w, h int) string {
	return components.RenderSidebar(m.registry, m.recentEvents, colors, w, h)
}

func (m *Model) renderThink(colors theme.Colors, w, h int) string {
	return components.RenderThinkPanel(m.scope, m.recentEvents, colors, w, h, m.thinkInput, m.thinkSubmitted)
}

func (m *Model) renderHelp(colors theme.Colors) string {
	return components.RenderHelp(colors, m.width-10, m.height-4)
}
