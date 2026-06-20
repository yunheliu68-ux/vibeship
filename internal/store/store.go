package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Snapshot holds a point-in-time usage snapshot from the Claude Code session.
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
	SevenDayResetsAt  string
}

// TranscriptEvent represents a single event from the Claude Code transcript.
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

// Store wraps a SQLite database and provides access to stored data.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) a SQLite database at the given path and runs
// migrations to ensure the schema is current. WAL journal mode is enabled.
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

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates tables and indexes if they do not yet exist.
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

// InsertSnapshot persists a usage snapshot.
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

// LatestSnapshot returns the most recently inserted usage snapshot.
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

// InsertEvent persists a transcript event.
func (s *Store) InsertEvent(e TranscriptEvent) error {
	_, err := s.db.Exec(
		`INSERT INTO transcript_events (timestamp, session_id, event_type, name, status, detail, duration_ms, todo_total, todo_done)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp.Format(time.RFC3339), e.SessionID, e.EventType,
		e.Name, e.Status, e.Detail, e.DurationMs, e.TodoTotal, e.TodoDone,
	)
	return err
}

// RecentEvents returns up to 50 transcript events newer than the given duration.
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
