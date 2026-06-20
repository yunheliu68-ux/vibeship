package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/francis/vibeship/internal/store"
)

// transcriptRecord represents one line in the Claude Code session JSONL.
type transcriptRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Tool      struct {
		Name  string                 `json:"name"`
		Input map[string]interface{} `json:"input"`
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

		// Resolve symlinks to prevent path traversal
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		cleanSessionsDir := filepath.Clean(sessionsDir)
		if !strings.HasPrefix(resolvedPath, cleanSessionsDir+"/") && resolvedPath != cleanSessionsDir {
			continue
		}

		// Ensure it's a regular file, not a symlink
		info, err := os.Lstat(resolvedPath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			continue
		}

		lastPos := lastPositions[resolvedPath]

		fi, err := os.Stat(resolvedPath)
		if err != nil {
			continue
		}

		if fi.Size() <= lastPos {
			continue // no new data
		}

		f, err := os.Open(resolvedPath)
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
		lastPositions[resolvedPath] = pos
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
