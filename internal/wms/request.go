package wms

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GetMapRequest holds the validated parameters of an OGC WMS GetMap request.
type GetMapRequest struct {
	Version string
	Layers  []string
	Styles  []string
	CRS     string
	Bbox    [4]float64 // always [minLon, minLat, maxLon, maxLat] (EPSG:4326) or [minX,minY,maxX,maxY]
	Width   int
	Height  int
	Format  string
}

// GetFeatureInfoRequest holds validated WMS GetFeatureInfo parameters.
type GetFeatureInfoRequest struct {
	Version     string
	Layers      []string
	QueryLayers []string
	Styles      []string
	CRS         string
	Bbox        [4]float64 // normalized to [minLon,minLat,maxLon,maxLat] for EPSG:4326 / CRS:84
	Width       int
	Height      int
	I           int // pixel x (WMS 1.3.0: I, WMS 1.1.1: X)
	J           int // pixel y (WMS 1.3.0: J, WMS 1.1.1: Y)
	InfoFormat  string
	FeatureCount int
}

// ParseGetMapRequest parses and validates OGC WMS 1.1.1 / 1.3.0 GetMap query params.
func ParseGetMapRequest(q url.Values) (GetMapRequest, error) {
	var r GetMapRequest
	get := func(key string) string {
		if v := q.Get(key); v != "" {
			return v
		}
		return q.Get(strings.ToLower(key))
	}

	r.Version = get("VERSION")

	layers := get("LAYERS")
	if layers == "" {
		return r, fmt.Errorf("LAYERS is required")
	}
	r.Layers = strings.Split(layers, ",")

	stylesStr := get("STYLES")
	if stylesStr != "" {
		r.Styles = strings.Split(stylesStr, ",")
	}

	// CRS (1.3.0) or SRS (1.1.1)
	r.CRS = get("CRS")
	if r.CRS == "" {
		r.CRS = get("SRS")
	}
	if r.CRS == "" {
		return r, fmt.Errorf("CRS (or SRS) is required")
	}

	bboxStr := get("BBOX")
	if bboxStr == "" {
		return r, fmt.Errorf("BBOX is required")
	}
	parts := strings.Split(bboxStr, ",")
	if len(parts) != 4 {
		return r, fmt.Errorf("BBOX must have exactly 4 comma-separated values")
	}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return r, fmt.Errorf("BBOX[%d] is not a number: %w", i, err)
		}
		r.Bbox[i] = v
	}

	// WMS 1.3.0 with EPSG:4326 uses lat/lon axis order: BBOX = miny,minx,maxy,maxx
	// Normalize to lon/lat: minx,miny,maxx,maxy
	if r.Version == "1.3.0" && r.CRS == "EPSG:4326" {
		r.Bbox = [4]float64{r.Bbox[1], r.Bbox[0], r.Bbox[3], r.Bbox[2]}
	}

	w, err := strconv.Atoi(get("WIDTH"))
	if err != nil || w <= 0 {
		return r, fmt.Errorf("WIDTH must be a positive integer")
	}
	r.Width = w

	h, err := strconv.Atoi(get("HEIGHT"))
	if err != nil || h <= 0 {
		return r, fmt.Errorf("HEIGHT must be a positive integer")
	}
	r.Height = h

	r.Format = get("FORMAT")
	if r.Format == "" {
		r.Format = "image/png"
	}

	return r, nil
}

// ParseGetFeatureInfoRequest parses and validates WMS GetFeatureInfo query params.
func ParseGetFeatureInfoRequest(q url.Values) (GetFeatureInfoRequest, error) {
	var r GetFeatureInfoRequest
	get := func(key string) string {
		if v := q.Get(key); v != "" {
			return v
		}
		return q.Get(strings.ToLower(key))
	}

	r.Version = get("VERSION")

	layers := get("LAYERS")
	if layers == "" {
		return r, fmt.Errorf("LAYERS is required")
	}
	r.Layers = strings.Split(layers, ",")

	queryLayers := get("QUERY_LAYERS")
	if queryLayers == "" {
		queryLayers = layers
	}
	r.QueryLayers = strings.Split(queryLayers, ",")

	stylesStr := get("STYLES")
	if stylesStr != "" {
		r.Styles = strings.Split(stylesStr, ",")
	}

	r.CRS = get("CRS")
	if r.CRS == "" {
		r.CRS = get("SRS")
	}
	if r.CRS == "" {
		return r, fmt.Errorf("CRS (or SRS) is required")
	}

	bboxStr := get("BBOX")
	if bboxStr == "" {
		return r, fmt.Errorf("BBOX is required")
	}
	parts := strings.Split(bboxStr, ",")
	if len(parts) != 4 {
		return r, fmt.Errorf("BBOX must have exactly 4 comma-separated values")
	}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return r, fmt.Errorf("BBOX[%d] is not a number: %w", i, err)
		}
		r.Bbox[i] = v
	}
	if r.Version == "1.3.0" && r.CRS == "EPSG:4326" {
		r.Bbox = [4]float64{r.Bbox[1], r.Bbox[0], r.Bbox[3], r.Bbox[2]}
	}

	w, err := strconv.Atoi(get("WIDTH"))
	if err != nil || w <= 0 {
		return r, fmt.Errorf("WIDTH must be a positive integer")
	}
	r.Width = w

	h, err := strconv.Atoi(get("HEIGHT"))
	if err != nil || h <= 0 {
		return r, fmt.Errorf("HEIGHT must be a positive integer")
	}
	r.Height = h

	iRaw := get("I")
	if iRaw == "" {
		iRaw = get("X")
	}
	jRaw := get("J")
	if jRaw == "" {
		jRaw = get("Y")
	}
	i, err := strconv.Atoi(iRaw)
	if err != nil || i < 0 || i >= w {
		return r, fmt.Errorf("I/X must be an integer in [0, WIDTH)")
	}
	j, err := strconv.Atoi(jRaw)
	if err != nil || j < 0 || j >= h {
		return r, fmt.Errorf("J/Y must be an integer in [0, HEIGHT)")
	}
	r.I = i
	r.J = j

	r.InfoFormat = get("INFO_FORMAT")
	if r.InfoFormat == "" {
		r.InfoFormat = "application/json"
	}

	if fc := get("FEATURE_COUNT"); fc != "" {
		n, err := strconv.Atoi(fc)
		if err != nil || n <= 0 {
			return r, fmt.Errorf("FEATURE_COUNT must be a positive integer")
		}
		r.FeatureCount = n
	} else {
		r.FeatureCount = 1
	}

	return r, nil
}
