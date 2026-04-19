package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// Counters holds all Meridian metrics as atomic integers.
type Counters struct {
	WMSRequests   atomic.Int64 // total WMS requests
	CacheHits     atomic.Int64 // GetMap cache hits
	WMSErrors     atomic.Int64 // WMS requests that returned an error
	GetMapTotal   atomic.Int64 // GetMap requests
	GetCapTotal   atomic.Int64 // GetCapabilities requests
}

// Global is the single shared counter set, wired into the WMS handler.
var Global = &Counters{}

// Handler returns an http.HandlerFunc that renders Prometheus-format metrics.
func Handler(c *Counters) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP meridian_wms_requests_total Total WMS requests received\n")
		fmt.Fprintf(w, "# TYPE meridian_wms_requests_total counter\n")
		fmt.Fprintf(w, "meridian_wms_requests_total %d\n\n", c.WMSRequests.Load())

		fmt.Fprintf(w, "# HELP meridian_wms_getmap_total Total GetMap requests\n")
		fmt.Fprintf(w, "# TYPE meridian_wms_getmap_total counter\n")
		fmt.Fprintf(w, "meridian_wms_getmap_total %d\n\n", c.GetMapTotal.Load())

		fmt.Fprintf(w, "# HELP meridian_wms_getcapabilities_total Total GetCapabilities requests\n")
		fmt.Fprintf(w, "# TYPE meridian_wms_getcapabilities_total counter\n")
		fmt.Fprintf(w, "meridian_wms_getcapabilities_total %d\n\n", c.GetCapTotal.Load())

		fmt.Fprintf(w, "# HELP meridian_cache_hits_total GetMap responses served from LRU cache\n")
		fmt.Fprintf(w, "# TYPE meridian_cache_hits_total counter\n")
		fmt.Fprintf(w, "meridian_cache_hits_total %d\n\n", c.CacheHits.Load())

		fmt.Fprintf(w, "# HELP meridian_wms_errors_total WMS requests that resulted in an error response\n")
		fmt.Fprintf(w, "# TYPE meridian_wms_errors_total counter\n")
		fmt.Fprintf(w, "meridian_wms_errors_total %d\n\n", c.WMSErrors.Load())
	}
}
