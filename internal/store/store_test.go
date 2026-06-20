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
