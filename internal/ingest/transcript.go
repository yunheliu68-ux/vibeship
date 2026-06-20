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
// Claude Code v2.1+ stores tool calls inside message.content[] blocks.
type transcriptRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Message   struct {
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text"`
			Name  string                 `json:"name"`
			ID    string                 `json:"id"`
			Input map[string]interface{} `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
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

// StartTranscriptPoller starts a background goroutine that scans
// ~/.claude/projects/*/ for .jsonl transcript files every 2 seconds.
func StartTranscriptPoller(st *store.Store, projectsDir string) chan struct{} {
	stop := make(chan struct{})
	lastPositions := make(map[string]int64)

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				scanProjects(projectsDir, st, lastPositions)
			}
		}
	}()
	return stop
}

func scanProjects(projectsDir string, st *store.Store, lastPositions map[string]int64) {
	projectDirs, err := os.ReadDir(projectsDir)
	if err != nil {
		return
	}

	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}
		projPath := filepath.Join(projectsDir, projectDir.Name())
		entries, err := os.ReadDir(projPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) != ".jsonl" {
				continue
			}
			processFile(filepath.Join(projPath, entry.Name()), projPath, st, lastPositions)
		}
	}
}

func processFile(path, projPath string, st *store.Store, lastPositions map[string]int64) {
	// Resolve symlinks to prevent path traversal
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return
	}
	cleanProjPath := filepath.Clean(projPath)
	if !strings.HasPrefix(resolvedPath, cleanProjPath+"/") && resolvedPath != cleanProjPath {
		return
	}

	// Ensure it's a regular file, not a symlink
	info, err := os.Lstat(resolvedPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return
	}

	lastPos := lastPositions[resolvedPath]

	fi, err := os.Stat(resolvedPath)
	if err != nil {
		return
	}

	if fi.Size() <= lastPos {
		return // no new data
	}

	f, err := os.Open(resolvedPath)
	if err != nil {
		return
	}
	defer f.Close()

	if lastPos > 0 {
		f.Seek(lastPos, 0)
	}

scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 8*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var rec transcriptRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}

		events := extractEvents(rec, filepath.Base(path))
		for _, e := range events {
			if err := st.InsertEvent(e); err != nil {
				fmt.Fprintf(os.Stderr, "vibeship: insert event: %v\n", err)
			}
		}
	}

	// Update position
	pos, _ := f.Seek(0, 1)
	lastPositions[resolvedPath] = pos
}

func extractEvents(rec transcriptRecord, sessionFile string) []store.TranscriptEvent {
	var events []store.TranscriptEvent
	ts, _ := time.Parse(time.RFC3339Nano, rec.Timestamp)
	if ts.IsZero() {
		ts = time.Now()
	}

	// Parse tool calls from message.content[] blocks (v2.1+ format)
	for _, c := range rec.Message.Content {
		if c.Type != "tool_use" || c.Name == "" {
			continue
		}

		// Extract file path detail
		detail := ""
		if filePath, ok := c.Input["file_path"].(string); ok {
			detail = filePath
		} else if path, ok := c.Input["path"].(string); ok {
			detail = path
		}

		// Tool call event
		events = append(events, store.TranscriptEvent{
			Timestamp: ts,
			SessionID: sessionFile,
			EventType: "tool_call",
			Name:      c.Name,
			Status:    "active",
			Detail:    detail,
		})

		// Skill invocation
		if c.Name == "Skill" {
			if skillName, ok := c.Input["skill"].(string); ok {
				events = append(events, store.TranscriptEvent{
					Timestamp: ts,
					SessionID: sessionFile,
					EventType: "skill",
					Name:      skillName,
					Status:    "active",
				})
			}
		}

		// MCP tool call
		if len(c.Name) > 5 && c.Name[:5] == "mcp__" {
			events = append(events, store.TranscriptEvent{
				Timestamp: ts,
				SessionID: sessionFile,
				EventType: "mcp",
				Name:      c.Name,
				Status:    "active",
			})
		}

		// Subagent dispatch via Agent or Task tool
		if c.Name == "Agent" || c.Name == "Task" {
			subName := firstNonEmpty(str(c.Input["description"]), str(c.Input["subject"]),
				str(c.Input["subagent_type"]), str(c.Input["name"]))
			if subName != "" {
				events = append(events, store.TranscriptEvent{
					Timestamp: ts, SessionID: sessionFile,
					EventType: "agent", Name: subName, Detail: str(c.Input["model"]), Status: "active",
				})
			}
		}

		// Todo events from TaskCreate / TaskUpdate (v2.1+)
		if c.Name == "TaskCreate" {
			events = append(events, store.TranscriptEvent{
				Timestamp: ts,
				SessionID: sessionFile,
				EventType: "todo",
				TodoTotal: 1,
				TodoDone:  0,
			})
		}
		if c.Name == "TaskUpdate" {
			if st, ok := c.Input["status"].(string); ok && st == "completed" {
				events = append(events, store.TranscriptEvent{
					Timestamp: ts,
					SessionID: sessionFile,
					EventType: "todo",
					TodoTotal: 0,
					TodoDone:  1,
				})
			}
		}

		// TodoWrite snapshot (v2.1+ full list)
		if c.Name == "TodoWrite" {
			if raw, ok := c.Input["todos"].([]interface{}); ok {
				done := 0
				for _, it := range raw {
					if m, ok := it.(map[string]interface{}); ok && str(m["status"]) == "completed" {
						done++
					}
				}
				events = append(events, store.TranscriptEvent{
					Timestamp: ts, SessionID: sessionFile,
					EventType: "todo_snapshot", TodoTotal: len(raw), TodoDone: done,
				})
			}
		}
	}

	return events
}

func str(v interface{}) string { s, _ := v.(string); return s }
func firstNonEmpty(ss ...string) string { for _, s := range ss { if s != "" { return s } }; return "" }
