//go:build windows

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseHistoryKeepsLastThirtyWithinSevenDays(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local)
	var content strings.Builder
	fmt.Fprintf(&content, "[%s] expired\r\nold\r\n\r\n", now.Add(-7*24*time.Hour-time.Second).Format("2006-01-02 15:04:05"))
	for i := 0; i < 31; i++ {
		fmt.Fprintf(&content, "[%s] message %d\r\nbody %d\r\n\r\n", now.Format("2006-01-02 15:04:05"), i, i)
	}
	entries := parseHistory(content.String(), now)
	if len(entries) != 30 {
		t.Fatalf("got %d entries, want 30", len(entries))
	}
	if entries[0].title != "message 1" || entries[29].body != "body 30" {
		t.Fatalf("unexpected retained range: first=%q last=%q", entries[0].title, entries[29].body)
	}
}

func TestParseHistoryKeepsSevenDayBoundaryAndMultilineBody(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local)
	content := fmt.Sprintf("[%s] boundary\r\nline one\r\nline two\r\n\r\n", now.Add(-7*24*time.Hour).Format("2006-01-02 15:04:05"))
	entries := parseHistory(content, now)
	if len(entries) != 1 || entries[0].body != "line one\r\nline two" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}
