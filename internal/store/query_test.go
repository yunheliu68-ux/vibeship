package store

import (
	"fmt"
	"testing"
	"time"
)

func TestRecentEventsQuery(t *testing.T) {
	s, err := Open("/Users/francis/.vibeship/data.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	events, err := s.RecentEvents(5 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\n=== Got %d events in last 5 min ===\n", len(events))
	for i, e := range events {
		if i >= 5 {
			break
		}
		fmt.Printf("  [%d] type=%s name=%s status=%s detail=%s ts=%s\n",
			i, e.EventType, e.Name, e.Status, e.Detail, e.Timestamp.Format(time.RFC3339))
	}
	if len(events) == 0 {
		t.Error("expected events but got 0")
	}
}
