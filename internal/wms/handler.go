package wms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"
	"strings"

	"github.com/paulmach/orb"

	"meridian/internal/cache"
	"meridian/internal/datasource"
	"meridian/internal/metrics"
	"meridian/internal/proj"
	"meridian/internal/renderer"
	"meridian/internal/store"
	"meridian/internal/style"
)

const symbolQueryPaddingPx = 8.0

// Handler serves OGC WMS requests.
type Handler struct {
	store    store.Store
	registry *datasource.Registry
	renderer renderer.Renderer
	cache    *cache.Cache
	metrics  *metrics.Counters
}

// NewHandler constructs a Handler with all dependencies injected.
func NewHandler(s store.Store, reg *datasource.Registry, rend renderer.Renderer, c *cache.Cache, m *metrics.Counters) *Handler {
	return &Handler{store: s, registry: reg, renderer: rend, cache: c, metrics: m}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.metrics.WMSRequests.Add(1)
	q := r.URL.Query()
	request := q.Get("REQUEST")
	if request == "" {
		request = q.Get("request")
	}
	switch strings.ToUpper(request) {
	case "GETMAP":
		h.metrics.GetMapTotal.Add(1)
		h.handleGetMap(w, r)
	case "GETFEATUREINFO":
		h.handleGetFeatureInfo(w, r)
	case "GETCAPABILITIES":
		h.metrics.GetCapTotal.Add(1)
		h.handleGetCapabilities(w, r)
	default:
		h.metrics.WMSErrors.Add(1)
		http.Error(w, "unsupported REQUEST: "+request, http.StatusBadRequest)
	}
}

func (h *Handler) handleGetCapabilities(w http.ResponseWriter, r *http.Request) {
	layers, err := h.store.ListLayers(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		slog.Error("ListLayers", "err", err)
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	serviceURL := fmt.Sprintf("%s://%s/wms", scheme, r.Host)

	xmlBytes, err := BuildCapabilities(serviceURL, layers)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write(xmlBytes)
}

func (h *Handler) handleGetMap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	// Normalize all param keys to upper for consistent cache keying
	params := make(map[string]string, len(q))
	for k, v := range q {
		if len(v) > 0 {
			params[strings.ToUpper(k)] = v[0]
		}
	}

	cacheKey := cache.Key(params)
	if cached, ok := h.cache.Get(cacheKey); ok {
		h.metrics.CacheHits.Add(1)
		writeImageResponse(w, params["FORMAT"], cached)
		return
	}

	req, err := ParseGetMapRequest(q)
	if err != nil {
		h.metrics.WMSErrors.Add(1)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !proj.IsSupportedCRS(req.CRS) {
		h.metrics.WMSErrors.Add(1)
		http.Error(w, "unsupported CRS: "+req.CRS, http.StatusBadRequest)
		return
	}

	if len(req.Layers) == 0 {
		h.metrics.WMSErrors.Add(1)
		http.Error(w, "no layers specified", http.StatusBadRequest)
		return
	}
	layerName := req.Layers[0] // phase 1: single layer per request

	layerRec, err := h.store.GetLayerByName(ctx, layerName)
	if err != nil {
		http.Error(w, "layer not found: "+layerName, http.StatusNotFound)
		return
	}

	src, ok := h.registry.Resolve(layerName)
	if !ok {
		http.Error(w, "layer source not loaded: "+layerName, http.StatusInternalServerError)
		return
	}

	// Normalize BBOX to EPSG:4326 for querying
	bboxIn4326 := proj.NormalizeBound(req.Bbox, req.CRS)
	queryBbox := expandBBoxByPixels(bboxIn4326, req.Width, req.Height, symbolQueryPaddingPx)
	bound := orb.Bound{
		Min: orb.Point{queryBbox[0], queryBbox[1]},
		Max: orb.Point{queryBbox[2], queryBbox[3]},
	}

	features, err := src.Query(ctx, datasource.Query{Bound: bound})
	if err != nil {
		http.Error(w, "data source error", http.StatusInternalServerError)
		slog.Error("datasource query", "layer", layerName, "err", err)
		return
	}

	styleRec, err := h.store.GetStyle(ctx, layerRec.StyleID)
	if err != nil {
		styleRec = store.StyleRecord{
			FillColor: "#3388ff", StrokeColor: "#ffffff", StrokeWidth: 1.0, Opacity: 1.0,
		}
	}

	s := style.Style{
		FillColor:   styleRec.FillColor,
		StrokeColor: styleRec.StrokeColor,
		StrokeWidth: styleRec.StrokeWidth,
		Opacity:     styleRec.Opacity,
	}

	img, err := h.renderer.Render(ctx, renderer.Request{
		Features: features,
		Style:    s,
		Width:    req.Width,
		Height:   req.Height,
		Bbox:     bboxIn4326, // features are in EPSG:4326; use the normalized bbox
	})
	if err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
		slog.Error("render", "layer", layerName, "err", err)
		return
	}

	var buf bytes.Buffer
	switch req.Format {
	case "image/jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}
	default: // image/png
		if err := png.Encode(&buf, img); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}
	}

	imgBytes := buf.Bytes()
	h.cache.Set(cacheKey, imgBytes)
	writeImageResponse(w, req.Format, imgBytes)
}

func writeImageResponse(w http.ResponseWriter, format string, data []byte) {
	ct := "image/png"
	if format == "image/jpeg" {
		ct = "image/jpeg"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write(data)
}

func (h *Handler) handleGetFeatureInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req, err := ParseGetFeatureInfoRequest(r.URL.Query())
	if err != nil {
		h.metrics.WMSErrors.Add(1)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !proj.IsSupportedCRS(req.CRS) {
		h.metrics.WMSErrors.Add(1)
		http.Error(w, "unsupported CRS: "+req.CRS, http.StatusBadRequest)
		return
	}
	if len(req.QueryLayers) == 0 {
		h.metrics.WMSErrors.Add(1)
		http.Error(w, "QUERY_LAYERS is required", http.StatusBadRequest)
		return
	}

	layerName := req.QueryLayers[0] // phase 1: identify on first query layer
	if _, err := h.store.GetLayerByName(ctx, layerName); err != nil {
		http.Error(w, "layer not found: "+layerName, http.StatusNotFound)
		return
	}
	src, ok := h.registry.Resolve(layerName)
	if !ok {
		http.Error(w, "layer source not loaded: "+layerName, http.StatusInternalServerError)
		return
	}

	bboxIn4326 := proj.NormalizeBound(req.Bbox, req.CRS)
	hitBbox := featureInfoHitBBox(bboxIn4326, req.Width, req.Height, req.I, req.J, symbolQueryPaddingPx)
	features, err := src.Query(ctx, datasource.Query{
		Bound: orb.Bound{
			Min: orb.Point{hitBbox[0], hitBbox[1]},
			Max: orb.Point{hitBbox[2], hitBbox[3]},
		},
	})
	if err != nil {
		h.metrics.WMSErrors.Add(1)
		http.Error(w, "data source error", http.StatusInternalServerError)
		return
	}
	if req.FeatureCount > 0 && len(features) > req.FeatureCount {
		features = features[:req.FeatureCount]
	}

	type infoFeature struct {
		GeometryType string         `json:"geometry_type"`
		Properties   map[string]any `json:"properties,omitempty"`
	}
	respFeatures := make([]infoFeature, 0, len(features))
	for _, f := range features {
		props := f.Properties
		if props == nil {
			props = map[string]any{}
		}
		respFeatures = append(respFeatures, infoFeature{
			GeometryType: f.Geometry.GeoJSONType(),
			Properties:   props,
		})
	}

	resp := map[string]any{
		"layer":       layerName,
		"crs":         req.CRS,
		"featureCount": len(respFeatures),
		"features":    respFeatures,
	}

	switch req.InfoFormat {
	case "text/plain":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if len(respFeatures) == 0 {
			_, _ = w.Write([]byte("No features found"))
			return
		}
		_, _ = w.Write([]byte(fmt.Sprintf("Layer: %s\nFeatures: %d", layerName, len(respFeatures))))
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func featureInfoHitBBox(bbox [4]float64, width, height, i, j int, tolerancePx float64) [4]float64 {
	if width <= 0 || height <= 0 {
		return bbox
	}
	dx := bbox[2] - bbox[0]
	dy := bbox[3] - bbox[1]
	if dx <= 0 || dy <= 0 {
		return bbox
	}

	x := bbox[0] + (float64(i)/float64(width))*dx
	y := bbox[3] - (float64(j)/float64(height))*dy
	tolX := dx * (tolerancePx / float64(width))
	tolY := dy * (tolerancePx / float64(height))
	return [4]float64{x - tolX, y - tolY, x + tolX, y + tolY}
}

func expandBBoxByPixels(bbox [4]float64, width, height int, padPx float64) [4]float64 {
	if width <= 0 || height <= 0 || padPx <= 0 {
		return bbox
	}
	dx := bbox[2] - bbox[0]
	dy := bbox[3] - bbox[1]
	if dx <= 0 || dy <= 0 {
		return bbox
	}

	padX := dx * (padPx / float64(width))
	padY := dy * (padPx / float64(height))
	return [4]float64{
		bbox[0] - padX,
		bbox[1] - padY,
		bbox[2] + padX,
		bbox[3] + padY,
	}
}
