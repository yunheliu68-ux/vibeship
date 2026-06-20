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
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
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

// IngestStatusline reads newline-delimited JSON statusline payloads from r,
// echos each to stdout, and inserts a store.Snapshot into the database.
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

		sessionID := os.Getenv("CLAUDE_SESSION_ID")
		if sessionID == "" {
			sessionID = "default"
		}
		snap := store.Snapshot{
			Timestamp:         time.Now(),
			SessionID:         sessionID,
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
