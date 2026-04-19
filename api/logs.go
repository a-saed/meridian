package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"meridian/internal/observability"
)

type logsHandler struct {
	ring *observability.RingBuffer
}

// list handles GET /api/v1/logs
//
// Query params:
//   - limit  int    (default 200, max 500) — number of newest entries to return
//   - level  string (optional) — filter to this level and above: DEBUG, INFO, WARN, ERROR
func (h *logsHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	filterLevel := strings.ToUpper(strings.TrimSpace(q.Get("level")))

	all := h.ring.Entries(0)
	total := len(all)

	filtered := make([]observability.LogEntry, 0, len(all))
	for _, e := range all {
		if filterLevel == "" || levelGTE(e.Level, filterLevel) {
			filtered = append(filtered, e)
		}
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"entries": filtered,
		"total":   total,
	})
}

var levelOrder = map[string]int{
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
}

// levelGTE returns true if entryLevel >= filterLevel.
func levelGTE(entryLevel, filterLevel string) bool {
	ev, eok := levelOrder[strings.ToUpper(entryLevel)]
	fv, fok := levelOrder[strings.ToUpper(filterLevel)]
	if !eok || !fok {
		return true
	}
	return ev >= fv
}
