package observability_test

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"meridian/internal/observability"
)

func TestRingBuffer_CapacityEviction(t *testing.T) {
	ring := observability.NewRingBuffer(3)
	for i := range 5 {
		ring.Add(observability.LogEntry{Msg: fmt.Sprintf("msg%d", i)})
	}
	entries := ring.Entries(10)
	if len(entries) != 3 {
		t.Fatalf("want 3 entries (cap), got %d", len(entries))
	}
	if entries[0].Msg != "msg2" {
		t.Errorf("want msg2 oldest, got %s", entries[0].Msg)
	}
	if entries[2].Msg != "msg4" {
		t.Errorf("want msg4 newest, got %s", entries[2].Msg)
	}
}

func TestRingBuffer_LimitEntries(t *testing.T) {
	ring := observability.NewRingBuffer(100)
	for i := range 10 {
		ring.Add(observability.LogEntry{Msg: fmt.Sprintf("m%d", i)})
	}
	if got := ring.Entries(3); len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	if got := ring.Entries(0); len(got) != 10 {
		t.Fatalf("want 10 for limit=0, got %d", len(got))
	}
}

func TestRingBufHandler_CapturesRecords(t *testing.T) {
	ring := observability.NewRingBuffer(50)
	inner := slog.NewTextHandler(&strings.Builder{}, nil)
	h := observability.NewRingBufHandler(inner, ring)
	logger := slog.New(h)

	logger.Info("hello", "key", "val")
	logger.Warn("world")

	entries := ring.Entries(10)
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Msg != "hello" {
		t.Errorf("want hello, got %s", entries[0].Msg)
	}
	if entries[0].Attrs["key"] != "val" {
		t.Errorf("want key=val, got %v", entries[0].Attrs["key"])
	}
	if entries[1].Level != "WARN" {
		t.Errorf("want WARN, got %s", entries[1].Level)
	}
}

func TestRingBufHandler_LevelFiltering(t *testing.T) {
	ring := observability.NewRingBuffer(50)
	inner := slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := observability.NewRingBufHandler(inner, ring)
	logger := slog.New(h)

	logger.Debug("debug-msg")
	logger.Info("info-msg")
	logger.Warn("warn-msg")

	entries := ring.Entries(10)
	if len(entries) != 1 || entries[0].Msg != "warn-msg" {
		t.Errorf("want only warn-msg, got %d entries: %+v", len(entries), entries)
	}
}
