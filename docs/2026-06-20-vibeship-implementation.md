# Vibeship Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone TUI dashboard (Go + Bubble Tea) that pairs with Claude Code to visualize token consumption, active tools/skills/agents, task progress, and provide an on-demand thinking co-pilot.

**Architecture:** Single Go binary with two modes — `vibeship collect` (transparent statusline pipe → SQLite) and `vibeship` (Bubble Tea TUI reading SQLite + transcript JSONL). Left 70% animation area, right 30% info cards. Two visual themes (Spaceship/DJ), toggleable. On-demand overlays for skills sidebar and thinking co-pilot.

**Tech Stack:** Go 1.22+, Bubble Tea, Lip Gloss, mattn/go-sqlite3, gopkg.in/yaml.v3 (for settings.json parsing)

## Global Constraints

- Go 1.22+
- Dependencies: Bubble Tea, Lip Gloss, mattn/go-sqlite3, yaml.v3 — no others without justification
- Binary name: `vibeship`
- Data directory: `~/.vibeship/`
- SQLite file: `~/.vibeship/data.db`
- Decision log: `~/.vibeship/decisions.jsonl`
- Coding language: English for code, Chinese for TUI labels
- No LLM calls in MVP — all intelligence is rules-based

---

## File Structure

```
vibeship/
├── main.go                          # Entry point: mode dispatch
├── go.mod                           # Module: github.com/francis/vibeship
├── internal/
│   ├── store/
│   │   └── store.go                 # SQLite: open, migrate, insert, query
│   ├── ingest/
│   │   ├── statusline.go            # Parse Claude Code statusline stdin JSON
│   │   └── transcript.go            # Poll & parse ~/.claude/sessions/*.jsonl
│   ├── model/
│   │   └── model.go                 # Root Bubble Tea Model + Update/View
│   ├── layout/
│   │   └── layout.go               # Flex layout: left 70% / right 30%
│   ├── components/
│   │   ├── topbar.go               # Top bar: project name, git branch, theme
│   │   ├── animation.go            # Animation dispatcher (theme switch)
│   │   ├── spaceship.go            # Spaceship: particles + speedometer + warp line
│   │   ├── dj.go                   # DJ: spectrum bars + breathing halo
│   │   ├── metrics.go              # Metrics card: cost, ctx%, rate limit
│   │   ├── activity.go             # Activity card: active tool/skill/MCP
│   │   ├── agents.go               # Agents card: running agents
│   │   ├── todos.go                # Todos card: progress bar + items
│   │   ├── sidebar.go              # Skills/plugins sidebar (`s` key)
│   │   ├── think.go                # Thinking co-pilot overlay (`d` key)
│   │   └── help.go                 # Help overlay (`?` key)
│   ├── theme/
│   │   └── theme.go                # Two color palettes, theme type
│   ├── rules/
│   │   └── rules.go                # Recommendations + scope check questions
│   └── config/
│       └── config.go               # Read ~/.claude/settings.json + plugins dir
```

---

### Task 1: Project Scaffold & Go Module

**Files:**
- Create: `vibeship/go.mod`
- Create: `vibeship/main.go`
- Create: `vibeship/Makefile`

**Interfaces:**
- Consumes: nothing (first task)
- Produces: module `github.com/francis/vibeship`, `main()` entry point with mode dispatch skeleton

- [ ] **Step 1: Initialize Go module**

```bash
mkdir -p vibeship
cd vibeship
go mod init github.com/francis/vibeship
```

Expected: `go.mod` created.

- [ ] **Step 2: Write main.go skeleton**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "collect" {
		runCollect()
		return
	}
	runTUI()
}

func runCollect() {
	fmt.Fprintln(os.Stderr, "vibeship collect: not yet implemented")
	os.Exit(0)
}

func runTUI() {
	fmt.Fprintln(os.Stderr, "vibeship tui: not yet implemented")
	os.Exit(0)
}
```

- [ ] **Step 3: Write Makefile**

```makefile
.PHONY: build run collect test clean

build:
	go build -o vibeship .

run: build
	./vibeship

collect: build
	./vibeship collect

test:
	go test ./...

clean:
	rm -f vibeship
```

- [ ] **Step 4: Build and verify**

```bash
cd vibeship && make build
./vibeship
./vibeship collect
```

Expected: "not yet implemented" messages for both modes. Exit code 0.

- [ ] **Step 5: Commit**

```bash
cd vibeship && git init && git add -A && git commit -m "feat: project scaffold with mode dispatch"
```

---

### Task 2: SQLite Store

**Files:**
- Create: `vibeship/internal/store/store.go`

**Interfaces:**
- Consumes: `go.mod` requires `github.com/mattn/go-sqlite3`
- Produces:
  - `func Open(path string) (*Store, error)`
  - `func (s *Store) Close() error`
  - `func (s *Store) InsertSnapshot(s Snapshot) error`
  - `func (s *Store) LatestSnapshot() (Snapshot, error)`
  - `func (s *Store) InsertEvent(e TranscriptEvent) error`
  - `func (s *Store) RecentEvents(since time.Duration) ([]TranscriptEvent, error)`
  - `type Snapshot struct { ... }`
  - `type TranscriptEvent struct { ... }`

- [ ] **Step 1: Add sqlite3 dependency**

```bash
cd vibeship && go get github.com/mattn/go-sqlite3
```

- [ ] **Step 2: Write store.go with schema and operations**

```go
package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Snapshot struct {
	Timestamp         time.Time
	SessionID         string
	ModelDisplayName  string
	ContextUsedPct    float64
	InputTokens       int64
	OutputTokens      int64
	CacheCreateTokens int64
	CacheReadTokens   int64
	TotalCostUSD      float64
	FiveHourUsedPct   float64
	FiveHourResetsAt  string
	SevenDayUsedPct   float64
	SevenDayResetsAt   string
}

type TranscriptEvent struct {
	Timestamp  time.Time
	SessionID  string
	EventType  string // tool_call, skill, agent, todo
	Name       string
	Status     string // active, done, queued
	Detail     string
	DurationMs int64
	TodoTotal  int
	TodoDone   int
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS usage_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			session_id TEXT NOT NULL,
			model_display_name TEXT,
			context_used_pct REAL,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cache_create_tokens INTEGER,
			cache_read_tokens INTEGER,
			total_cost_usd REAL,
			five_hour_used_pct REAL,
			five_hour_resets_at TEXT,
			seven_day_used_pct REAL,
			seven_day_resets_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS transcript_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			session_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			name TEXT,
			status TEXT,
			detail TEXT,
			duration_ms INTEGER,
			todo_total INTEGER,
			todo_done INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_session ON usage_snapshots(session_id, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_events_session ON transcript_events(session_id, timestamp)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) InsertSnapshot(snap Snapshot) error {
	_, err := s.db.Exec(
		`INSERT INTO usage_snapshots (timestamp, session_id, model_display_name,
		 context_used_pct, input_tokens, output_tokens, cache_create_tokens,
		 cache_read_tokens, total_cost_usd, five_hour_used_pct,
		 five_hour_resets_at, seven_day_used_pct, seven_day_resets_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.Timestamp.Format(time.RFC3339), snap.SessionID, snap.ModelDisplayName,
		snap.ContextUsedPct, snap.InputTokens, snap.OutputTokens,
		snap.CacheCreateTokens, snap.CacheReadTokens, snap.TotalCostUSD,
		snap.FiveHourUsedPct, snap.FiveHourResetsAt,
		snap.SevenDayUsedPct, snap.SevenDayResetsAt,
	)
	return err
}

func (s *Store) LatestSnapshot() (Snapshot, error) {
	var snap Snapshot
	var ts string
	err := s.db.QueryRow(
		`SELECT timestamp, session_id, model_display_name, context_used_pct,
		 input_tokens, output_tokens, cache_create_tokens, cache_read_tokens,
		 total_cost_usd, five_hour_used_pct, five_hour_resets_at,
		 seven_day_used_pct, seven_day_resets_at
		 FROM usage_snapshots ORDER BY id DESC LIMIT 1`,
	).Scan(&ts, &snap.SessionID, &snap.ModelDisplayName, &snap.ContextUsedPct,
		&snap.InputTokens, &snap.OutputTokens, &snap.CacheCreateTokens,
		&snap.CacheReadTokens, &snap.TotalCostUSD, &snap.FiveHourUsedPct,
		&snap.FiveHourResetsAt, &snap.SevenDayUsedPct, &snap.SevenDayResetsAt)
	if err != nil {
		return snap, err
	}
	snap.Timestamp, _ = time.Parse(time.RFC3339, ts)
	return snap, nil
}

func (s *Store) InsertEvent(e TranscriptEvent) error {
	_, err := s.db.Exec(
		`INSERT INTO transcript_events (timestamp, session_id, event_type, name, status, detail, duration_ms, todo_total, todo_done)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp.Format(time.RFC3339), e.SessionID, e.EventType,
		e.Name, e.Status, e.Detail, e.DurationMs, e.TodoTotal, e.TodoDone,
	)
	return err
}

func (s *Store) RecentEvents(since time.Duration) ([]TranscriptEvent, error) {
	cutoff := time.Now().Add(-since).Format(time.RFC3339)
	rows, err := s.db.Query(
		`SELECT timestamp, session_id, event_type, name, status, detail, duration_ms, todo_total, todo_done
		 FROM transcript_events WHERE timestamp >= ? ORDER BY id DESC LIMIT 50`,
		cutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TranscriptEvent
	for rows.Next() {
		var e TranscriptEvent
		var ts string
		if err := rows.Scan(&ts, &e.SessionID, &e.EventType, &e.Name,
			&e.Status, &e.Detail, &e.DurationMs, &e.TodoTotal, &e.TodoDone); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		events = append(events, e)
	}
	return events, nil
}
```

- [ ] **Step 3: Write and run basic test**

Create `vibeship/internal/store/store_test.go`:

```go
package store

import (
	"os"
	"testing"
	"time"
)

func TestOpenAndInsert(t *testing.T) {
	path := "/tmp/vibeship_test.db"
	os.Remove(path)

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	defer os.Remove(path)

	snap := Snapshot{
		Timestamp:        time.Now(),
		SessionID:        "test-session",
		ModelDisplayName: "test-model",
		ContextUsedPct:   45.0,
		InputTokens:      1000,
		OutputTokens:     500,
		TotalCostUSD:     0.01,
	}
	if err := s.InsertSnapshot(snap); err != nil {
		t.Fatalf("InsertSnapshot: %v", err)
	}

	got, err := s.LatestSnapshot()
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if got.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "test-session")
	}
	if got.ContextUsedPct != 45.0 {
		t.Errorf("ContextUsedPct = %f, want 45.0", got.ContextUsedPct)
	}
}
```

Run: `cd vibeship && go test ./internal/store/ -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: SQLite store with snapshots and transcript events"
```

---

### Task 3: Statusline Ingest (vibeship collect)

**Files:**
- Create: `vibeship/internal/ingest/statusline.go`
- Modify: `vibeship/main.go` (implement `runCollect`)

**Interfaces:**
- Consumes: `store.Store`, `store.Snapshot`
- Produces: `func IngestStatusline(r io.Reader, s *store.Store) error` — reads stdin JSON, inserts into store, echoes to stdout

- [ ] **Step 1: Write statusline.go parser**

```go
package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/francis/vibeship/internal/store"
)

// StatuslinePayload matches the JSON Claude Code sends to its statusline command.
type StatuslinePayload struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage float64 `json:"used_percentage"`
		CurrentUsage   struct {
			InputTokens            int64 `json:"input_tokens"`
			OutputTokens           int64 `json:"output_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens    int64 `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	Cost struct {
		TotalCostUSD float64 `json:"total_cost_usd"`
	} `json:"cost"`
	RateLimits struct {
		FiveHour struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       string  `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       string  `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
}

func IngestStatusline(r io.Reader, st *store.Store) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Echo to stdout (transparent pipe)
		fmt.Println(string(line))

		var payload StatuslinePayload
		if err := json.Unmarshal(line, &payload); err != nil {
			fmt.Fprintf(os.Stderr, "vibeship: failed to parse statusline JSON: %v\n", err)
			continue
		}

		snap := store.Snapshot{
			Timestamp:         time.Now(),
			ModelDisplayName:  payload.Model.DisplayName,
			ContextUsedPct:    payload.ContextWindow.UsedPercentage,
			InputTokens:       payload.ContextWindow.CurrentUsage.InputTokens,
			OutputTokens:      payload.ContextWindow.CurrentUsage.OutputTokens,
			CacheCreateTokens: payload.ContextWindow.CurrentUsage.CacheCreationInputTokens,
			CacheReadTokens:   payload.ContextWindow.CurrentUsage.CacheReadInputTokens,
			TotalCostUSD:      payload.Cost.TotalCostUSD,
			FiveHourUsedPct:   payload.RateLimits.FiveHour.UsedPercentage,
			FiveHourResetsAt:  payload.RateLimits.FiveHour.ResetsAt,
			SevenDayUsedPct:   payload.RateLimits.SevenDay.UsedPercentage,
			SevenDayResetsAt:   payload.RateLimits.SevenDay.ResetsAt,
		}

		if err := st.InsertSnapshot(snap); err != nil {
			fmt.Fprintf(os.Stderr, "vibeship: failed to store snapshot: %v\n", err)
		}
	}
	return scanner.Err()
}
```

- [ ] **Step 2: Update main.go runCollect**

```go
func runCollect() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot find home dir:", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".vibeship", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot create data dir:", err)
		os.Exit(1)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot open store:", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := ingest.IngestStatusline(os.Stdin, st); err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: ingest error:", err)
		os.Exit(1)
	}
}
```

Add imports: `"os"`, `"path/filepath"`, `"github.com/francis/vibeship/internal/ingest"`, `"github.com/francis/vibeship/internal/store"`.

- [ ] **Step 3: Test with sample JSON**

```bash
cd vibeship && echo '{"model":{"display_name":"test"},"context_window":{"used_percentage":45,"current_usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":200}},"cost":{"total_cost_usd":0.01},"rate_limits":{"five_hour":{"used_percentage":25,"resets_at":"2026-06-20T10:00:00Z"},"seven_day":{"used_percentage":80,"resets_at":"2026-06-27T00:00:00Z"}}}' | ./vibeship collect
```

Expected: JSON echoed to stdout. Check `~/.vibeship/data.db` exists with data.

```bash
sqlite3 ~/.vibeship/data.db "SELECT * FROM usage_snapshots;"
```

Expected: one row with the sample data.

- [ ] **Step 4: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: statusline ingest via vibeship collect"
```

---

### Task 4: Transcript Polling

**Files:**
- Create: `vibeship/internal/ingest/transcript.go`

**Interfaces:**
- Consumes: `store.Store`, `store.TranscriptEvent`
- Produces:
  - `func StartTranscriptPoller(st *store.Store, sessionsDir string) chan struct{}` — starts background goroutine, returns stop channel

- [ ] **Step 1: Write transcript.go poller**

```go
package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/francis/vibeship/internal/store"
)

// transcriptRecord represents one line in the Claude Code session JSONL.
type transcriptRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Tool      struct {
		Name   string `json:"name"`
		Input  map[string]interface{} `json:"input"`
	} `json:"tool"`
	Message struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	TodoWrite struct {
		NewTodos []struct {
			Content    string `json:"content"`
			Status     string `json:"status"`
			ActiveForm string `json:"active_form"`
		} `json:"newTodos"`
	} `json:"todo_write"`
	Subagent struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Model string `json:"model"`
	} `json:"subagent"`
}

func StartTranscriptPoller(st *store.Store, sessionsDir string) chan struct{} {
	stop := make(chan struct{})
	lastPositions := make(map[string]int64) // file path -> last read offset

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				processTranscripts(sessionsDir, st, lastPositions)
			}
		}
	}()
	return stop
}

func processTranscripts(sessionsDir string, st *store.Store, lastPositions map[string]int64) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(sessionsDir, entry.Name())
		lastPos := lastPositions[path]

		fi, err := os.Stat(path)
		if err != nil {
			continue
		}

		if fi.Size() <= lastPos {
			continue // no new data
		}

		f, err := os.Open(path)
		if err != nil {
			continue
		}

		if lastPos > 0 {
			f.Seek(lastPos, 0)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var rec transcriptRecord
			if err := json.Unmarshal(line, &rec); err != nil {
				continue
			}

			events := extractEvents(rec, entry.Name())
			for _, e := range events {
				if err := st.InsertEvent(e); err != nil {
					fmt.Fprintf(os.Stderr, "vibeship: insert event: %v\n", err)
				}
			}
		}

		// Update position
		pos, _ := f.Seek(0, 1) // current position
		lastPositions[path] = pos
		f.Close()
	}
}

func extractEvents(rec transcriptRecord, sessionFile string) []store.TranscriptEvent {
	var events []store.TranscriptEvent
	ts, _ := time.Parse(time.RFC3339Nano, rec.Timestamp)

	// Tool call events
	if rec.Tool.Name != "" {
		detail := ""
		if filePath, ok := rec.Tool.Input["file_path"].(string); ok {
			detail = filePath
		} else if path, ok := rec.Tool.Input["path"].(string); ok {
			detail = path
		}
		events = append(events, store.TranscriptEvent{
			Timestamp: ts,
			SessionID: sessionFile,
			EventType: "tool_call",
			Name:      rec.Tool.Name,
			Status:    "active",
			Detail:    detail,
		})
	}

	// Skill invocations (Skill tool)
	if rec.Tool.Name == "Skill" {
		if skillName, ok := rec.Tool.Input["skill"].(string); ok {
			events = append(events, store.TranscriptEvent{
				Timestamp: ts,
				SessionID: sessionFile,
				EventType: "skill",
				Name:      skillName,
				Status:    "active",
			})
		}
	}

	// MCP tool calls
	if rec.Tool.Name != "" {
		// MCP tools have pattern: mcp__server__tool
		if len(rec.Tool.Name) > 5 && rec.Tool.Name[:5] == "mcp__" {
			events = append(events, store.TranscriptEvent{
				Timestamp: ts,
				SessionID: sessionFile,
				EventType: "mcp",
				Name:      rec.Tool.Name,
				Status:    "active",
			})
		}
	}

	// Todo events
	if len(rec.TodoWrite.NewTodos) > 0 {
		total := len(rec.TodoWrite.NewTodos)
		done := 0
		for _, t := range rec.TodoWrite.NewTodos {
			if t.Status == "completed" {
				done++
			}
		}
		events = append(events, store.TranscriptEvent{
			Timestamp: ts,
			SessionID: sessionFile,
			EventType: "todo",
			TodoTotal: total,
			TodoDone:  done,
		})
	}

	// Subagent events
	if rec.Subagent.ID != "" {
		events = append(events, store.TranscriptEvent{
			Timestamp: ts,
			SessionID: sessionFile,
			EventType: "agent",
			Name:      rec.Subagent.Name,
			Detail:    rec.Subagent.Model,
			Status:    "active",
		})
	}

	return events
}
```

- [ ] **Step 2: Build and verify it compiles**

```bash
cd vibeship && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: transcript JSONL polling with event extraction"
```

---

### Task 5: Config Loader & Theme Definitions

**Files:**
- Create: `vibeship/internal/config/config.go`
- Create: `vibeship/internal/theme/theme.go`

**Interfaces:**
- Consumes: `gopkg.in/yaml.v3`
- Produces:
  - `config.LoadSkillsAndPlugins(homeDir string) (*Registry, error)`
  - `type Registry struct { Skills []SkillItem; Plugins []PluginItem }`
  - `theme.Theme` type with `Spaceship` and `DJ` instances
  - `theme.Colors` struct with palette fields

- [ ] **Step 1: Add yaml dependency**

```bash
cd vibeship && go get gopkg.in/yaml.v3
```

- [ ] **Step 2: Write theme.go**

```go
package theme

import "github.com/charmbracelet/lipgloss"

type ThemeName string

const (
	Spaceship ThemeName = "spaceship"
	DJ        ThemeName = "dj"
)

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

func ColorsFor(t ThemeName) Colors {
	switch t {
	case DJ:
		return DJColors
	default:
		return SpaceshipColors
	}
}
```

- [ ] **Step 3: Write config.go**

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type SkillItem struct {
	Name     string
	Active   bool
	Category string // e.g. "superpowers", "lark", "figma"
}

type PluginItem struct {
	Name   string
	Active bool
}

type Registry struct {
	Skills  []SkillItem
	Plugins []PluginItem
}

func LoadSkillsAndPlugins(homeDir string) (*Registry, error) {
	r := &Registry{}

	// Read settings.json for enabled plugins
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return r, nil // no settings, return empty
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return r, nil
	}

	// Track which plugins are enabled
	enabledPlugins := make(map[string]bool)
	if ep, ok := settings["enabledPlugins"].(map[string]interface{}); ok {
		for name, enabled := range ep {
			if v, ok := enabled.(bool); ok && v {
				enabledPlugins[name] = true
			}
		}
	}

	// Scan skills directory
	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	skillEntries, _ := os.ReadDir(skillsDir)
	for _, entry := range skillEntries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check subdirectories for plugin-namespaced skills
		subEntries, err := os.ReadDir(filepath.Join(skillsDir, name))
		if err != nil {
			continue
		}
		for _, sub := range subEntries {
			if sub.IsDir() {
				fullName := name + ":" + sub.Name()
				r.Skills = append(r.Skills, SkillItem{
					Name:     fullName,
					Category: name,
					Active:   false, // updated by transcript events
				})
			}
		}
	}

	// Scan plugins directory for marketplace names
	pluginsDir := filepath.Join(homeDir, ".claude", "plugins")
	pluginEntries, _ := os.ReadDir(pluginsDir)
	for _, entry := range pluginEntries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Try to read plugin metadata
		manifestPath := filepath.Join(pluginsDir, name, "manifest.json")
		var pluginName string
		if manifestData, err := os.ReadFile(manifestPath); err == nil {
			var manifest struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(manifestData, &manifest) == nil && manifest.Name != "" {
				pluginName = manifest.Name
			}
		}
		if pluginName == "" {
			pluginName = name
		}

		r.Plugins = append(r.Plugins, PluginItem{
			Name:   pluginName,
			Active: enabledPlugins[pluginName],
		})
	}

	// Sort skills by category then name
	sort.Slice(r.Skills, func(i, j int) bool {
		if r.Skills[i].Category != r.Skills[j].Category {
			return r.Skills[i].Category < r.Skills[j].Category
		}
		return r.Skills[i].Name < r.Skills[j].Name
	})

	return r, nil
}

// ParseScopeFile reads SCOPE.md or PRD.md from a project directory
// and returns extracted sections.
func ParseScopeFile(projectDir string) (*Scope, error) {
	for _, name := range []string{"SCOPE.md", "PRD.md"} {
		data, err := os.ReadFile(filepath.Join(projectDir, name))
		if err != nil {
			continue
		}
		return parseScopeMarkdown(string(data)), nil
	}
	return nil, nil // no scope file found, that's OK
}

type Scope struct {
	Goals       []string
	Files       []string
	OutOfScope  []string
	DevelopOrder []string
}

func parseScopeMarkdown(content string) *Scope {
	s := &Scope{}
	currentSection := ""
	for _, line := range splitLines(content) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			switch strings.ToLower(strings.TrimPrefix(line, "## ")) {
			case "goals":
				currentSection = "goals"
			case "files":
				currentSection = "files"
			case "out of scope":
				currentSection = "out"
			case "develop order":
				currentSection = "order"
			default:
				currentSection = ""
			}
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			item := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			switch currentSection {
			case "goals":
				s.Goals = append(s.Goals, item)
			case "files":
				s.Files = append(s.Files, item)
			case "out":
				s.OutOfScope = append(s.OutOfScope, item)
			case "order":
				s.DevelopOrder = append(s.DevelopOrder, item)
			}
		}
	}
	return s
}

func splitLines(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' })
}
```

Add imports: `"strings"`.

- [ ] **Step 4: Build and verify**

```bash
cd vibeship && go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: config loader, theme palettes, scope file parser"
```

---

### Task 6: Rules Engine

**Files:**
- Create: `vibeship/internal/rules/rules.go`

**Interfaces:**
- Consumes: `config.Scope`, `store.TranscriptEvent`
- Produces:
  - `func RecommendSkill(recentEvents []store.TranscriptEvent) string`
  - `func GenerateCheckQuestions(scope *config.Scope, recentEvents []store.TranscriptEvent) []string`

- [ ] **Step 1: Write rules.go**

```go
package rules

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/store"
)

// RecommendSkill returns a skill name to suggest based on recent activity.
func RecommendSkill(events []store.TranscriptEvent) string {
	// Look at last 5 minutes of events
	cutoff := time.Now().Add(-5 * time.Minute)
	hasEdit := false
	hasBashError := false
	hasNewBranch := false

	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		switch e.EventType {
		case "tool_call":
			if e.Name == "Write" || e.Name == "Edit" {
				hasEdit = true
			}
			if e.Name == "Bash" && strings.Contains(strings.ToLower(e.Detail), "error") {
				hasBashError = true
			}
		}
	}

	if hasEdit {
		return "code-review"
	}
	if hasBashError {
		return "systematic-debugging"
	}
	if hasNewBranch {
		return "writing-plans"
	}
	return ""
}

// GenerateCheckQuestions produces scope/thinking questions based on recent activity.
func GenerateCheckQuestions(scope *config.Scope, events []store.TranscriptEvent) []string {
	var questions []string
	cutoff := time.Now().Add(-10 * time.Minute)

	if scope != nil {
		// Check if recent file edits are within scope
		var outOfScopeFiles []string
		var inScopeFiles []string
		for _, e := range events {
			if e.Timestamp.Before(cutoff) {
				continue
			}
			if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
				if e.Detail == "" {
					continue
				}
				if isInScope(e.Detail, scope.Files) {
					inScopeFiles = append(inScopeFiles, e.Detail)
				} else {
					outOfScopeFiles = append(outOfScopeFiles, e.Detail)
				}
			}
		}

		if len(outOfScopeFiles) > 0 {
			questions = append(questions,
				"⚠️ 这些文件不在 SCOPE.md 里：\n   "+strings.Join(outOfScopeFiles, ", ")+"\n   确定要继续改吗？")
		}

		// Check if writing code without schema changes
		hasWrite := false
		hasSchema := false
		for _, e := range events {
			if e.Timestamp.Before(cutoff) {
				continue
			}
			if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
				hasWrite = true
			}
			if e.EventType == "tool_call" && e.Name == "Write" {
				if strings.Contains(e.Detail, "migration") ||
					strings.Contains(e.Detail, "schema") ||
					strings.Contains(e.Detail, ".sql") {
					hasSchema = true
				}
			}
		}
		if hasWrite && !hasSchema && len(scope.DevelopOrder) > 0 {
			questions = append(questions,
				"💡 先定义数据结构再写代码。Scope 里的开发顺序：\n   "+strings.Join(scope.DevelopOrder, " → "))
		}

		// Check frontend/backend coordination
		hasFrontend := false
		hasBackend := false
		for _, e := range events {
			if e.Timestamp.Before(cutoff) {
				continue
			}
			if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
				ext := strings.ToLower(filepath.Ext(e.Detail))
				dir := filepath.Dir(e.Detail)
				if ext == ".tsx" || ext == ".jsx" || ext == ".vue" || strings.Contains(dir, "frontend") || strings.Contains(dir, "components") {
					hasFrontend = true
				}
				if ext == ".go" || ext == ".py" || ext == ".rs" || strings.Contains(dir, "backend") || strings.Contains(dir, "api") {
					hasBackend = true
				}
			}
		}
		if hasFrontend && !hasBackend {
			questions = append(questions,
				"💡 只改了前端，后端接口对齐了吗？建议先确认 API contract。")
		}
		if hasBackend && !hasFrontend {
			questions = append(questions,
				"💡 只改了后端，前端对接准备好了吗？")
		}
	}

	// Check if stuck (no todo completions recently)
	hasTodoCompletion := false
	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		if e.EventType == "todo" && e.TodoDone > 0 {
			hasTodoCompletion = true
			break
		}
	}
	if !hasTodoCompletion && len(events) > 0 {
		questions = append(questions,
			"💡 最近 10 分钟没有完成待办——是不是卡住了？换个角度聊聊？")
	}

	// Check new dependencies
	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
			base := filepath.Base(e.Detail)
			if base == "go.mod" || base == "package.json" || base == "Cargo.toml" {
				questions = append(questions,
					"💡 新增了依赖。确认过选型吗？有无更轻量的替代？")
				break
			}
		}
	}

	// Always show scope goals as reference
	if scope != nil && len(scope.Goals) > 0 {
		questions = append([]string{"📋 当前目标：\n   " + strings.Join(scope.Goals, "\n   ")}, questions...)
	}

	return questions
}

func isInScope(filePath string, scopeFiles []string) bool {
	for _, pattern := range scopeFiles {
		// Simple glob: prefix or suffix match
		if strings.HasPrefix(pattern, "*.") {
			if strings.HasSuffix(filePath, strings.TrimPrefix(pattern, "*")) {
				return true
			}
		} else if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(filePath, dir) {
				return true
			}
		} else if strings.HasSuffix(pattern, "/**") {
			dir := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(filePath, dir) {
				return true
			}
		} else if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Build and verify**

```bash
cd vibeship && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: rules engine for skill recommendations and scope check questions"
```

---

### Task 7: Bubble Tea Model & Layout

**Files:**
- Create: `vibeship/internal/model/model.go`
- Create: `vibeship/internal/layout/layout.go`

**Interfaces:**
- Consumes: `store.Store`, `config.Registry`, `theme.Theme`
- Produces: Root Bubble Tea `Model` with full Update/View lifecycle

- [ ] **Step 1: Add Bubble Tea and Lip Gloss deps**

```bash
cd vibeship && go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss
```

- [ ] **Step 2: Write layout.go**

```go
package layout

import (
	"github.com/charmbracelet/lipgloss"
)

// Viewport constants
const (
	AnimationWidthPct  = 0.70
	InfoPanelWidthPct  = 0.30
)

func ViewportSize(termWidth, termHeight int) (animW, animH, infoW, infoH int) {
	// Subtract 2 for top bar + bottom status bar
	contentHeight := termHeight - 2
	animW = int(float64(termWidth) * AnimationWidthPct)
	infoW = termWidth - animW
	animH = contentHeight
	infoH = contentHeight
	return
}

func Card(title string, content string, width int, active bool) string {
	borderColor := lipgloss.Color("#444444")
	if active {
		borderColor = lipgloss.Color("#00ff88")
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Padding(0, 1)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		content,
	))
}

func StatusBar(keys string, width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#576574")).
		Width(width).
		Render(keys)
}
```

- [ ] **Step 3: Write model.go**

```go
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
	latestSnap  store.Snapshot
	recentEvents []store.TranscriptEvent

	// Animation tick
	tick int

	// Transcript poller stop channel
	pollStop chan struct{}

	// Decision log file
	decisionLog *os.File

	// Sidebar scroll
	sidebarScroll int
}

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

func (m *Model) renderDashboard(colors theme.Colors) string {
	// Top bar
	topBar := m.renderTopBar(colors)

	// Main content
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
	statusBar := layout.StatusBar("t=主题  s=skills  d=思路  ?=帮助  q=退出", m.width)

	return lipgloss.JoinVertical(lipgloss.Top, topBar, main, statusBar)
}

func (m *Model) Close() {
	if m.pollStop != nil {
		close(m.pollStop)
	}
	if m.decisionLog != nil {
		m.decisionLog.Close()
	}
}
```

Add imports for all component packages (to be created in subsequent tasks). For now, stub the render methods:

```go
func (m *Model) renderTopBar(colors theme.Colors) string {
	return fmt.Sprintf("Vibeship · %s  %s", filepath.Base(m.projectDir), "🚀")
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
```

- [ ] **Step 4: Update main.go runTUI**

In `main.go`, replace the stub `runTUI`:

```go
func runTUI() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot find home dir:", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".vibeship", "data.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vibeship: failed to open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	reg, _ := config.LoadSkillsAndPlugins(home)

	m := model.New(st, reg)
	defer m.Close()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vibeship: error: %v\n", err)
		os.Exit(1)
	}
}
```

Add imports: `"github.com/charmbracelet/bubbletea"`, `"github.com/francis/vibeship/internal/model"`, `"github.com/francis/vibeship/internal/config"`.

- [ ] **Step 5: Build and test basic render**

```bash
cd vibeship && go build ./...
```

Expected: compiles. Don't test TUI in non-interactive shell — just verify it builds.

- [ ] **Step 6: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: Bubble Tea model with layout, key handling, overlay dispatch"
```

---

### Task 8: Top Bar & Info Cards

**Files:**
- Create: `vibeship/internal/components/topbar.go`
- Create: `vibeship/internal/components/metrics.go`
- Create: `vibeship/internal/components/activity.go`
- Create: `vibeship/internal/components/agents.go`
- Create: `vibeship/internal/components/todos.go`
- Modify: `vibeship/internal/model/model.go` (wire real renderers)

**Interfaces:**
- Each component exposes a `Render` function taking data + colors + width, returning a string

- [ ] **Step 1: Write topbar.go**

```go
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/theme"
)

func RenderTopBar(projectName, gitBranch string, thm theme.ThemeName, colors theme.Colors, width int) string {
	left := fmt.Sprintf("Vibeship · %s", projectName)
	if gitBranch != "" {
		left += fmt.Sprintf(" git:(%s)", gitBranch)
	}

	themeIcon := "🚀"
	if thm == theme.DJ {
		themeIcon = "🎵"
	}
	right := fmt.Sprintf("%s 曲速", themeIcon)

	style := lipgloss.NewStyle().
		Foreground(colors.Text).
		Width(width).
		Padding(0, 1)

	return style.Render(lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(colors.Primary).Render(left),
		lipgloss.NewStyle().Foreground(colors.Dim).Render(right),
	))
}
```

- [ ] **Step 2: Write metrics.go**

```go
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func RenderMetricsCard(snap store.Snapshot, colors theme.Colors, width int) string {
	costStr := fmt.Sprintf("$%.2f", snap.TotalCostUSD)
	ctxStr := fmt.Sprintf("%.0f%% ctx", snap.ContextUsedPct)

	paceEmoji := "🟢"
	paceLabel := "on pace"
	if snap.FiveHourUsedPct > 80 {
		paceEmoji = "🔴"
		paceLabel = "hot"
	} else if snap.FiveHourUsedPct > 50 {
		paceEmoji = "🟡"
		paceLabel = "warming"
	}

	limitStr := fmt.Sprintf("%.0f%% limit", snap.FiveHourUsedPct)

	content := fmt.Sprintf("%s  %s  %s %s",
		lipgloss.NewStyle().Foreground(colors.Primary).Bold(true).Render(costStr),
		lipgloss.NewStyle().Foreground(colors.Text).Render(ctxStr),
		lipgloss.NewStyle().Foreground(colors.Text).Render(limitStr),
		lipgloss.NewStyle().Render(paceEmoji+" "+paceLabel),
	)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width - 2).
		Padding(0, 1)

	return style.Render(content)
}
```

- [ ] **Step 3: Write activity.go**

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func RenderActivityCard(events []store.TranscriptEvent, colors theme.Colors, width int) string {
	var activeTool, activeSkill, activeMCP string

	for _, e := range events {
		switch e.EventType {
		case "tool_call":
			if e.Status == "active" {
				if e.Detail != "" {
					activeTool = fmt.Sprintf("◐ %s: %s", e.Name, e.Detail)
				} else {
					activeTool = fmt.Sprintf("◐ %s", e.Name)
				}
			}
		case "skill":
			if e.Status == "active" {
				activeSkill = fmt.Sprintf("🧩 %s", e.Name)
			}
		case "mcp":
			if e.Status == "active" {
				// Extract server name from mcp__server__tool
				parts := strings.SplitN(e.Name, "__", 3)
				if len(parts) >= 2 {
					activeMCP = fmt.Sprintf("🔌 %s", parts[1])
				}
			}
		}
	}

	// Count completed tools
	doneCount := 0
	for _, e := range events {
		if e.EventType == "tool_call" && e.Status == "done" {
			doneCount++
		}
	}

	var lines []string
	if activeTool != "" {
		lines = append(lines, activeTool)
	}
	if activeSkill != "" {
		lines = append(lines, activeSkill)
	}
	if activeMCP != "" {
		lines = append(lines, activeMCP)
	}
	if doneCount > 0 {
		lines = append(lines, fmt.Sprintf("✓ %d completed", doneCount))
	}
	if len(lines) == 0 {
		lines = append(lines, "—")
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width - 2).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("⚡ Activity") + "\n" +
			strings.Join(lines, "\n"),
	)
}
```

- [ ] **Step 4: Write agents.go**

```go
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func RenderAgentsCard(events []store.TranscriptEvent, colors theme.Colors, width int) string {
	var lines []string
	for _, e := range events {
		if e.EventType != "agent" {
			continue
		}
		icon := "◐"
		if e.Status == "done" {
			icon = "✓"
		} else if e.Status == "queued" {
			icon = "○"
		}
		modelStr := ""
		if e.Detail != "" {
			modelStr = fmt.Sprintf("[%s]", e.Detail)
		}
		durStr := ""
		if e.DurationMs > 0 {
			d := time.Duration(e.DurationMs) * time.Millisecond
			durStr = fmt.Sprintf("⏱ %s", d.Round(time.Second))
		}
		lines = append(lines, fmt.Sprintf("%s %s %s  %s",
			icon, e.Name, modelStr, durStr))
	}
	if len(lines) == 0 {
		lines = append(lines, "—")
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width - 2).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("🤖 Agents") + "\n" +
			strings.Join(lines, "\n"),
	)
}
```

- [ ] **Step 5: Write todos.go**

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func RenderTodosCard(events []store.TranscriptEvent, colors theme.Colors, width int) string {
	// Find the most recent todo event
	var latestTodo store.TranscriptEvent
	for _, e := range events {
		if e.EventType == "todo" {
			latestTodo = e
			break
		}
	}

	if latestTodo.TodoTotal == 0 {
		content := "—"
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Width(width - 2).
			Padding(0, 1)
		return style.Render(
			lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📋 Todos") + "\n" + content,
		)
	}

	pct := 0
	if latestTodo.TodoTotal > 0 {
		pct = (latestTodo.TodoDone * 100) / latestTodo.TodoTotal
	}

	barWidth := width - 6
	filled := (pct * barWidth) / 100
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	content := fmt.Sprintf("%s %d%% · %d/%d done",
		bar, pct, latestTodo.TodoDone, latestTodo.TodoTotal)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(width - 2).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("📋 Todos") + "\n" + content,
	)
}
```

- [ ] **Step 6: Wire real renderers in model.go**

Replace the stub `renderTopBar`, `renderInfoPanel` methods:

```go
func (m *Model) renderTopBar(colors theme.Colors) string {
	return components.RenderTopBar(
		filepath.Base(m.projectDir), "", m.theme, colors, m.width,
	)
}

func (m *Model) renderInfoPanel(colors theme.Colors, w, h int) string {
	cardW := w
	metricsView := components.RenderMetricsCard(m.latestSnap, colors, cardW)
	activityView := components.RenderActivityCard(m.recentEvents, colors, cardW)
	agentsView := components.RenderAgentsCard(m.recentEvents, colors, cardW)
	todosView := components.RenderTodosCard(m.recentEvents, colors, cardW)
	return lipgloss.JoinVertical(lipgloss.Top,
		metricsView,
		activityView,
		agentsView,
		todosView,
	)
}
```

- [ ] **Step 7: Build and verify**

```bash
cd vibeship && go build ./...
```

Expected: compiles.

- [ ] **Step 8: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: top bar, metrics, activity, agents, todos cards"
```

---

### Task 9: Spaceship Theme Animation

**Files:**
- Create: `vibeship/internal/components/animation.go`
- Create: `vibeship/internal/components/spaceship.go`

**Interfaces:**
- Consumes: `store.Snapshot`, tick counter
- Produces: `RenderSpaceship(snap store.Snapshot, tick int, colors theme.Colors, w, h int) string`

- [ ] **Step 1: Write animation.go dispatcher**

```go
package components

import (
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func RenderAnimation(thm theme.ThemeName, snap store.Snapshot, tick int, colors theme.Colors, w, h int) string {
	switch thm {
	case theme.DJ:
		return renderDJ(snap, tick, colors, w, h)
	default:
		return renderSpaceship(snap, tick, colors, w, h)
	}
}
```

- [ ] **Step 2: Write spaceship.go**

```go
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
	// Particle field (stars) in background
	particles := renderParticles(tick, w, h, snap)

	// Speedometer gauge in center
	gauge := renderGauge(snap, colors)

	// Warp speed line at bottom
	warpLine := renderWarpLine(snap, tick, w, colors)

	// Compose: particles as background, gauge centered, warp line at bottom
	// Center the gauge
	centerY := (h - lipgloss.Height(gauge)) / 2
	centerX := (w - lipgloss.Width(gauge)) / 2

	// Place gauge on top of particles
	result := particles
	result = placeString(result, gauge, centerX, centerY)

	// Place warp line at bottom
	result = placeString(result, warpLine, 0, h-1)

	return result
}

func renderParticles(tick int, w, h int, snap store.Snapshot) string {
	// Generate pseudo-random star positions based on tick
	// Density scales with output token rate
	density := 10 // base stars
	if snap.OutputTokens > 0 {
		// More tokens = more stars, up to 80 stars
		density = int(math.Min(float64(snap.OutputTokens/2), 80))
		if density < 10 {
			density = 10
		}
	}

	// Create a grid of spaces
	grid := make([][]rune, h)
	for y := range grid {
		grid[y] = []rune(strings.Repeat(" ", w))
	}

	// Place stars using a simple LCG seeded by tick and position
	for i := 0; i < density; i++ {
		seed := (tick + i*7) % 1000
		x := (seed * 13) % w
		y := (seed * 17) % h

		// Star brightness based on position phase
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
	// Simple speedometer: a dial with rate value in center
	rate := snap.OutputTokens // approximate rate per tick window
	rateStr := fmt.Sprintf("%d t/s", rate)
	if rate == 0 {
		rateStr = "— t/s"
	}

	// Determine gauge color based on rate
	gaugeColor := colors.Primary
	if snap.FiveHourUsedPct > 80 {
		gaugeColor = colors.Danger
	} else if snap.FiveHourUsedPct > 50 {
		gaugeColor = colors.Warning
	}

	// Build gauge frame
	top := "╭─────────────────────╮"
	bottom := "╰─────────────────────╯"
	pointer := "│        ╱╲            │"
	rateLine := fmt.Sprintf("│  %s  │", lipgloss.NewStyle().Foreground(gaugeColor).Bold(true).Render(centerStr(rateStr, 19)))
	pointer2 := "│       ╲╱           │"

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
	// Horizontal line of dashes, speed proportional to token rate
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
```

- [ ] **Step 3: Wire animation into model.go**

Replace stub `renderAnimation`:

```go
func (m *Model) renderAnimation(colors theme.Colors, w, h int) string {
	return components.RenderAnimation(m.theme, m.latestSnap, m.tick, colors, w, h)
}
```

- [ ] **Step 4: Build and verify**

```bash
cd vibeship && go build ./...
```

Expected: compiles.

- [ ] **Step 5: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: spaceship theme with particles, speedometer, warp line"
```

---

### Task 10: DJ Theme Animation

**Files:**
- Create: `vibeship/internal/components/dj.go`

**Interfaces:**
- Consumes: `store.Snapshot`, tick
- Produces: `renderDJ(snap store.Snapshot, tick int, colors theme.Colors, w, h int) string`

- [ ] **Step 1: Write dj.go**

```go
package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/francis/vibeship/internal/store"
	"github.com/francis/vibeship/internal/theme"
)

func renderDJ(snap store.Snapshot, tick int, colors theme.Colors, w, h int) string {
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
```

- [ ] **Step 2: Build and verify**

```bash
cd vibeship && go build ./...
```

Expected: compiles.

- [ ] **Step 3: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: DJ theme with spectrum bars and breathing halo"
```

---

### Task 11: Skills Sidebar & Help Overlay

**Files:**
- Create: `vibeship/internal/components/sidebar.go`
- Create: `vibeship/internal/components/help.go`
- Modify: `vibeship/internal/model/model.go` (wire sidebar and help renderers)

- [ ] **Step 1: Write sidebar.go**

```go
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

func RenderSidebar(reg *config.Registry, events []store.TranscriptEvent, colors theme.Colors, w, h int) string {
	var lines []string

	// Skills section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("🧩 Skills"))
	lines = append(lines, strings.Repeat("─", w-4))

	currentCategory := ""
	for _, sk := range reg.Skills {
		if sk.Category != currentCategory {
			currentCategory = sk.Category
			lines = append(lines, lipgloss.NewStyle().Foreground(colors.Dim).Render(currentCategory+":"))
		}
		marker := "  —"
		if sk.Active {
			marker = "  ✓"
		}
		lines = append(lines, fmt.Sprintf("%s %s", marker, sk.Name))
	}

	// Plugins section
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colors.Primary).Render("🔌 Plugins (enabled)"))
	lines = append(lines, strings.Repeat("─", w-4))

	for _, p := range reg.Plugins {
		if p.Active {
			lines = append(lines, fmt.Sprintf("  ✓ %s", p.Name))
		}
	}

	// Separator
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", w-4))

	// Recommendation
	rec := rules.RecommendSkill(events)
	if rec != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colors.Warning).Render("💡 Recommended"))
		lines = append(lines, fmt.Sprintf("  → %s", rec))
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(w - 2).
		Height(h).
		Padding(0, 1)

	// Truncate if too long
	content := strings.Join(lines, "\n")
	return style.Render(content)
}
```

- [ ] **Step 2: Write help.go**

```go
package components

import (
	"strings"

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
```

- [ ] **Step 3: Wire in model.go**

Replace stub `renderSidebar` and `renderHelp`:

```go
func (m *Model) renderSidebar(colors theme.Colors, w, h int) string {
	return components.RenderSidebar(m.registry, m.recentEvents, colors, w, h)
}

func (m *Model) renderHelp(colors theme.Colors) string {
	return components.RenderHelp(colors, m.width-10, m.height-4)
}
```

- [ ] **Step 4: Build and verify**

```bash
cd vibeship && go build ./...
```

Expected: compiles.

- [ ] **Step 5: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: skills sidebar with recommendations, help overlay"
```

---

### Task 12: Thinking Co-pilot Overlay

**Files:**
- Create: `vibeship/internal/components/think.go`
- Modify: `vibeship/internal/model/model.go` (wire think renderer, add text input mode)

- [ ] **Step 1: Write think.go**

```go
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

func RenderThinkPanel(scope *config.Scope, events []store.TranscriptEvent, colors theme.Colors, w, h int) string {
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
	sections = append(sections, strings.Repeat("─", w-4))

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
	sections = append(sections, strings.Repeat("─", w-4))

	// Input prompt
	sections = append(sections, lipgloss.NewStyle().Foreground(colors.Dim).Render("💬 输入你的问题，Enter 后切到 Claude Code 去问"))
	sections = append(sections, lipgloss.NewStyle().Foreground(colors.Primary).Render("▸ _"))

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Width(w - 2).
		Height(h).
		Padding(0, 1)

	return style.Render(strings.Join(sections, "\n"))
}
```

- [ ] **Step 2: Wire in model.go**

Replace stub `renderThink`:

```go
func (m *Model) renderThink(colors theme.Colors, w, h int) string {
	return components.RenderThinkPanel(m.scope, m.recentEvents, colors, w, h)
}
```

- [ ] **Step 3: Build and verify**

```bash
cd vibeship && go build ./...
```

Expected: compiles.

- [ ] **Step 4: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: thinking co-pilot overlay with scope check and questions"
```

---

### Task 13: Integration & Polish

**Files:**
- Modify: `vibeship/main.go` (final imports, error handling)
- Modify: `vibeship/internal/model/model.go` (edge cases, empty states)

- [ ] **Step 1: Add edge case handling to model**

Update `refreshDataCmd` to not crash on empty DB:

```go
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
```

Add a "waiting for Claude Code" state to `renderAnimation`:

```go
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
	return components.RenderAnimation(m.theme, m.latestSnap, m.tick, colors, w, h)
}
```

- [ ] **Step 2: Add stale data detection to metrics card**

In metrics.go, detect stale snapshots (>30s):

```go
func RenderMetricsCard(snap store.Snapshot, colors theme.Colors, width int) string {
	stale := time.Since(snap.Timestamp) > 30*time.Second

	// ... existing content building ...

	if stale && snap.SessionID != "" {
		content = lipgloss.NewStyle().Foreground(colors.Dim).Render(content)
	}

	// ... rest of rendering ...
}
```

Add import for `"time"`.

- [ ] **Step 3: Verify final build**

```bash
cd vibeship && go build ./... && go vet ./...
```

Expected: clean build, no vet warnings.

- [ ] **Step 4: Run tests**

```bash
cd vibeship && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd vibeship && git add -A && git commit -m "feat: edge case handling, stale detection, empty states"
```

---

### Task 14: Packaging & README

**Files:**
- Create: `vibeship/README.md`

- [ ] **Step 1: Write README.md**

Create a concise README with:
- What Vibeship is (one sentence)
- Install: `go install github.com/francis/vibeship@latest`
- Setup: statusline configuration snippet
- Usage: `vibeship` in one terminal, Claude Code in another
- Screenshot placeholder (to be added after first run)
- Keybindings reference

- [ ] **Step 2: Final commit**

```bash
cd vibeship && git add -A && git commit -m "docs: README with install and setup instructions"
```

---

## Appendix: Complete main.go After All Tasks

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/ingest"
	"github.com/francis/vibeship/internal/model"
	"github.com/francis/vibeship/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "collect" {
		runCollect()
		return
	}
	runTUI()
}

func runCollect() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot find home dir:", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".vibeship", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot create data dir:", err)
		os.Exit(1)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot open store:", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := ingest.IngestStatusline(os.Stdin, st); err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: ingest error:", err)
		os.Exit(1)
	}
}

func runTUI() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot find home dir:", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".vibeship", "data.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vibeship: failed to open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	reg, _ := config.LoadSkillsAndPlugins(home)

	m := model.New(st, reg)
	defer m.Close()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vibeship: error: %v\n", err)
		os.Exit(1)
	}
}
```
