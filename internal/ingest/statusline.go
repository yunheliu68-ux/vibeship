package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/francis/vibeship/internal/store"
)

// StatuslinePayload matches the JSON Claude Code sends to its statusline command.
type StatuslinePayload struct {
	SessionID string `json:"session_id"`
	Model     struct {
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
			ResetsAt       int64   `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       int64   `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
}

// IngestStatusline reads newline-delimited JSON statusline payloads from r,
// stores each as a Snapshot in the database, and outputs a compact
// human-readable status line to stdout for Claude Code's prompt bar.
func IngestStatusline(r io.Reader, st *store.Store) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()

		var payload StatuslinePayload
		if err := json.Unmarshal(line, &payload); err != nil {
			// Parse failed: echo original so statusline doesn't go blank
			fmt.Println(string(line))
			fmt.Fprintf(os.Stderr, "vibeship: failed to parse statusline JSON: %v\n", err)
			continue
		}

		sessionID := payload.SessionID
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
			FiveHourResetsAt:  isoTime(payload.RateLimits.FiveHour.ResetsAt),
			SevenDayUsedPct:   payload.RateLimits.SevenDay.UsedPercentage,
			SevenDayResetsAt:  isoTime(payload.RateLimits.SevenDay.ResetsAt),
		}
		if err := st.InsertSnapshot(snap); err != nil {
			fmt.Fprintf(os.Stderr, "vibeship: failed to store snapshot: %v\n", err)
		}

		// Output a clean status line instead of raw JSON
		fmt.Println(formatStatusLine(payload))
	}
	return scanner.Err()
}

// formatStatusLine builds a compact status line from a statusline payload
// so that vibeship collect can serve as the statusLine command without
// dumping raw JSON onto the prompt bar.
func formatStatusLine(p StatuslinePayload) string {
	parts := make([]string, 0, 4)
	if p.Model.DisplayName != "" {
		parts = append(parts, p.Model.DisplayName)
	}
	if p.Cost.TotalCostUSD > 0 {
		parts = append(parts, fmt.Sprintf("$%.2f", p.Cost.TotalCostUSD))
	}
	parts = append(parts, fmt.Sprintf("ctx %.0f%%", p.ContextWindow.UsedPercentage))
	if p.RateLimits.FiveHour.UsedPercentage > 0 {
		parts = append(parts, fmt.Sprintf("5h %.0f%%", p.RateLimits.FiveHour.UsedPercentage))
	}
	return "🚀 " + strings.Join(parts, " · ")
}

func isoTime(u int64) string {
	if u <= 0 {
		return ""
	}
	return time.Unix(u, 0).UTC().Format(time.RFC3339)
}
