package main

import (
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// testBin is the path to the compiled vibeship binary, built once in TestMain.
var testBin string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "vibeship-test")
	if err != nil {
		os.Exit(1)
	}
	testBin = filepath.Join(tmpDir, "vibeship")
	build := exec.Command("go", "build", "-o", testBin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(tmpDir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// runCollectBin runs `vibeship collect` with the given HOME and stdin input.
func runCollectBin(t *testing.T, home, input string) (stdout, stderr string) {
	t.Helper()
	cmd := exec.Command(testBin, "collect")
	cmd.Env = append(os.Environ(), "HOME="+home)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("collect failed: %v\nstderr: %s", err, errBuf.String())
	}
	return outBuf.String(), errBuf.String()
}

// TestBuild ensures the package compiles (belt-and-suspenders; TestMain already
// built the binary successfully).
func TestBuild(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
}

// TestCollectModeEmptyStdin verifies collect mode exits cleanly when stdin
// is empty.
func TestCollectModeEmptyStdin(t *testing.T) {
	tmpHome := t.TempDir()
	stdout, _ := runCollectBin(t, tmpHome, "")
	if stdout != "" {
		t.Logf("stdout with empty stdin: %q", stdout)
	}
}

// TestCollectModeWithData verifies collect mode parses valid JSON, stores to
// SQLite, and outputs formatted status lines with rate-limit indicators.
func TestCollectModeWithData(t *testing.T) {
	tmpHome := t.TempDir()
	os.MkdirAll(filepath.Join(tmpHome, ".vibeship"), 0755)

	input := `{"session_id":"s1","model":{"display_name":"Claude Opus 4.8"},"context_window":{"used_percentage":42.5,"current_usage":{"input_tokens":15000,"output_tokens":3000,"cache_creation_input_tokens":0,"cache_read_input_tokens":5000}},"cost":{"total_cost_usd":2.35},"rate_limits":{"five_hour":{"used_percentage":30.0,"resets_at":0},"seven_day":{"used_percentage":15.0,"resets_at":0}}}
{"session_id":"s2","model":{"display_name":"Claude Sonnet"},"context_window":{"used_percentage":78.0,"current_usage":{"input_tokens":20000,"output_tokens":5000,"cache_creation_input_tokens":100,"cache_read_input_tokens":200}},"cost":{"total_cost_usd":3.50},"rate_limits":{"five_hour":{"used_percentage":85.0,"resets_at":0},"seven_day":{"used_percentage":45.0,"resets_at":0}}}
`

	stdout, _ := runCollectBin(t, tmpHome, input)

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 output lines, got %d: %q", len(lines), stdout)
	}

	// Line 1: 🚀 Claude Opus 4.8 · $2.35 · ctx 42% · 5h 30%
	line0 := lines[0]
	if !strings.Contains(line0, "🚀") {
		t.Errorf("line 0 should start with 🚀, got: %q", line0)
	}
	if !strings.Contains(line0, "Claude Opus 4.8") {
		t.Errorf("line 0 should contain model name, got: %q", line0)
	}
	if !strings.Contains(line0, "ctx 42%") {
		t.Errorf("line 0 should show 'ctx 42%%', got: %q", line0)
	}
	if !strings.Contains(line0, "$2.35") {
		t.Errorf("line 0 should show $2.35, got: %q", line0)
	}
	if !strings.Contains(line0, "5h 30%") {
		t.Errorf("line 0 should show '5h 30%%', got: %q", line0)
	}

	// Line 2: 🚀 Claude Sonnet · $3.50 · ctx 78% · 5h 85%
	line1 := lines[1]
	if !strings.Contains(line1, "Claude Sonnet") {
		t.Errorf("line 1 should contain 'Claude Sonnet', got: %q", line1)
	}
	if !strings.Contains(line1, "ctx 78%") {
		t.Errorf("line 1 should show 'ctx 78%%', got: %q", line1)
	}
	if !strings.Contains(line1, "$3.50") {
		t.Errorf("line 1 should show $3.50, got: %q", line1)
	}
	if !strings.Contains(line1, "5h 85%") {
		t.Errorf("line 1 should show '5h 85%%', got: %q", line1)
	}

	// Verify DB contents
	dbPath := filepath.Join(tmpHome, ".vibeship", "data.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open DB: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM usage_snapshots").Scan(&count); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 snapshots, got %d", count)
	}
}

// TestCollectModeInvalidJSON verifies collect mode handles bad JSON gracefully:
// echoes the original line, logs parse error to stderr, continues processing.
func TestCollectModeInvalidJSON(t *testing.T) {
	tmpHome := t.TempDir()
	os.MkdirAll(filepath.Join(tmpHome, ".vibeship"), 0755)

	input := `not-valid-json
{"session_id":"ok","model":{"display_name":"Claude"},"context_window":{"used_percentage":10.0,"current_usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"cost":{"total_cost_usd":0.01},"rate_limits":{"five_hour":{"used_percentage":5.0,"resets_at":0},"seven_day":{"used_percentage":2.0,"resets_at":0}}}
`

	stdout, stderr := runCollectBin(t, tmpHome, input)

	if !strings.Contains(stdout, "not-valid-json") {
		t.Errorf("bad JSON line should be echoed; stdout: %q", stdout)
	}
	if !strings.Contains(stdout, "ctx 10%") {
		t.Errorf("valid JSON should produce formatted line with 'ctx 10%%'; stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "failed to parse") {
		t.Errorf("stderr should mention parse failure: %q", stderr)
	}

	// Only 1 good record in DB
	dbPath := filepath.Join(tmpHome, ".vibeship", "data.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open DB: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM usage_snapshots").Scan(&count); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 snapshot (bad JSON skipped), got %d", count)
	}
}

// TestCollectModeHighFrequency verifies collect handles rapid-fire input
// without data loss (simulating ~15s of 300ms tick statusline payloads).
func TestCollectModeHighFrequency(t *testing.T) {
	tmpHome := t.TempDir()
	os.MkdirAll(filepath.Join(tmpHome, ".vibeship"), 0755)

	var buf bytes.Buffer
	for i := 0; i < 50; i++ {
		buf.WriteString(`{"session_id":"burst","model":{"display_name":"Claude"},"context_window":{"used_percentage":10.0,"current_usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"cost":{"total_cost_usd":0.01},"rate_limits":{"five_hour":{"used_percentage":5.0,"resets_at":0},"seven_day":{"used_percentage":2.0,"resets_at":0}}}` + "\n")
	}

	stdout, _ := runCollectBin(t, tmpHome, buf.String())

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 50 {
		t.Errorf("expected 50 output lines, got %d", len(lines))
	}

	dbPath := filepath.Join(tmpHome, ".vibeship", "data.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open DB: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM usage_snapshots").Scan(&count); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if count != 50 {
		t.Errorf("expected 50 snapshots, got %d", count)
	}
}
