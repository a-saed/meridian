package observability

import (
	"context"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"time"
)

// LogEntry is one captured log record.
type LogEntry struct {
	Time  time.Time      `json:"time"`
	Level string         `json:"level"`
	Msg   string         `json:"msg"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// RingBuffer is a thread-safe circular buffer of LogEntry values.
type RingBuffer struct {
	mu       sync.Mutex
	buf      []LogEntry
	capacity int
	head     int
	wrapped  bool // true once the buffer has filled and head has looped back to 0
}

// NewRingBuffer creates a ring buffer with the given capacity.
// Panics if capacity < 1.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 1 {
		panic("observability: RingBuffer capacity must be >= 1")
	}
	return &RingBuffer{buf: make([]LogEntry, capacity), capacity: capacity}
}

// Add appends an entry, evicting the oldest when full.
func (r *RingBuffer) Add(e LogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.head] = e
	r.head = (r.head + 1) % r.capacity
	if r.head == 0 {
		r.wrapped = true
	}
}

// Entries returns up to limit entries in insertion order (oldest first).
// If limit <= 0, all stored entries are returned.
func (r *RingBuffer) Entries(limit int) []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	var size int
	if r.wrapped {
		size = r.capacity
	} else {
		size = r.head
	}
	if size == 0 {
		return nil
	}

	start := 0
	if r.wrapped {
		start = r.head
	}

	out := make([]LogEntry, size)
	for i := range size {
		out[i] = r.buf[(start+i)%r.capacity]
	}

	if limit > 0 && limit < size {
		// Return the newest `limit` entries.
		out = out[size-limit:]
	}
	return out
}

// RingBufHandler is a slog.Handler that delegates to an inner handler and
// simultaneously captures records in a RingBuffer.
//
// Attr keys are prefixed with the current group stack (e.g. "http.method")
// so the ring buffer JSON matches the inner handler's structured output.
type RingBufHandler struct {
	inner    slog.Handler
	ring     *RingBuffer
	preAttrs map[string]any // attrs from WithAttrs calls, already group-prefixed
	groups   []string       // current group name stack
}

// NewRingBufHandler wraps inner so every handled record is also stored in ring.
// Panics if ring is nil.
func NewRingBufHandler(inner slog.Handler, ring *RingBuffer) *RingBufHandler {
	if ring == nil {
		panic("observability: NewRingBufHandler: ring must not be nil")
	}
	return &RingBufHandler{inner: inner, ring: ring}
}

func (h *RingBufHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// groupPrefix returns the dot-separated group prefix for the current group stack.
// e.g. groups=["http","request"] → "http.request."
func (h *RingBufHandler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

func (h *RingBufHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}
	attrs := make(map[string]any, len(h.preAttrs)+r.NumAttrs())
	maps.Copy(attrs, h.preAttrs)
	prefix := h.groupPrefix()
	r.Attrs(func(a slog.Attr) bool {
		attrs[prefix+a.Key] = a.Value.Any()
		return true
	})
	e := LogEntry{
		Time:  r.Time.UTC(),
		Level: r.Level.String(),
		Msg:   r.Message,
	}
	if len(attrs) > 0 {
		e.Attrs = attrs
	}
	h.ring.Add(e)
	return nil
}

func (h *RingBufHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	prefix := h.groupPrefix()
	merged := make(map[string]any, len(h.preAttrs)+len(attrs))
	maps.Copy(merged, h.preAttrs)
	for _, a := range attrs {
		merged[prefix+a.Key] = a.Value.Any()
	}
	return &RingBufHandler{
		inner:    h.inner.WithAttrs(attrs),
		ring:     h.ring,
		preAttrs: merged,
		groups:   h.groups,
	}
}

func (h *RingBufHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &RingBufHandler{
		inner:    h.inner.WithGroup(name),
		ring:     h.ring,
		preAttrs: h.preAttrs, // pre-group attrs stay at their original prefix
		groups:   newGroups,
	}
}
