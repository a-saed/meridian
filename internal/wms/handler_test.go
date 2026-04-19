package wms_test

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulmach/orb"

	"meridian/internal/cache"
	"meridian/internal/datasource"
	"meridian/internal/metrics"
	"meridian/internal/renderer"
	"meridian/internal/store"
	"meridian/internal/wms"
)

// --- mocks ---

type mockStore struct{ layers []store.LayerRecord }

func (m *mockStore) CreateSource(_ context.Context, _ store.SourceRecord) error { return nil }
func (m *mockStore) ListSources(_ context.Context) ([]store.SourceRecord, error) {
	return nil, nil
}
func (m *mockStore) GetSource(_ context.Context, _ string) (store.SourceRecord, error) {
	return store.SourceRecord{}, nil
}
func (m *mockStore) DeleteSource(_ context.Context, _ string) error { return nil }
func (m *mockStore) CreateLayer(_ context.Context, _ store.LayerRecord) error { return nil }
func (m *mockStore) ListLayers(_ context.Context) ([]store.LayerRecord, error) {
	return m.layers, nil
}
func (m *mockStore) GetLayerByName(_ context.Context, name string) (store.LayerRecord, error) {
	for _, l := range m.layers {
		if l.Name == name {
			return l, nil
		}
	}
	return store.LayerRecord{}, fmt.Errorf("not found")
}
func (m *mockStore) UpdateLayer(_ context.Context, _ store.LayerRecord) error  { return nil }
func (m *mockStore) DeleteLayer(_ context.Context, _ string) error             { return nil }
func (m *mockStore) CreateStyle(_ context.Context, _ store.StyleRecord) error  { return nil }
func (m *mockStore) ListStyles(_ context.Context) ([]store.StyleRecord, error) { return nil, nil }
func (m *mockStore) GetStyle(_ context.Context, _ string) (store.StyleRecord, error) {
	return store.StyleRecord{
		FillColor:   "#3388ff",
		StrokeColor: "#ffffff",
		StrokeWidth: 1.0,
		Opacity:     1.0,
	}, nil
}
func (m *mockStore) DeleteStyle(_ context.Context, _ string) error { return nil }
func (m *mockStore) Close() error                                   { return nil }

type mockDataSource struct{}

func (m *mockDataSource) Query(_ context.Context, _ datasource.Query) ([]datasource.Feature, error) {
	return []datasource.Feature{{Geometry: orb.Point{35.9, 31.9}}}, nil
}
func (m *mockDataSource) Close() error { return nil }

type recordingDataSource struct {
	lastQuery datasource.Query
}

func (r *recordingDataSource) Query(_ context.Context, q datasource.Query) ([]datasource.Feature, error) {
	r.lastQuery = q
	return []datasource.Feature{{Geometry: orb.Point{35.9, 31.9}}}, nil
}
func (r *recordingDataSource) Close() error { return nil }

type mockRenderer struct{}

func (m *mockRenderer) Render(_ context.Context, _ renderer.Request) (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, 256, 256)), nil
}

// --- tests ---

func makeHandler(t *testing.T) http.Handler {
	t.Helper()

	layers := []store.LayerRecord{
		{
			ID:      "l1",
			Name:    "roads",
			Title:   "Roads",
			StyleID: "s1",
			SRS:     `["EPSG:4326","EPSG:3857"]`,
			MinX:    -180, MinY: -90, MaxX: 180, MaxY: 90,
		},
	}

	st := &mockStore{layers: layers}
	reg := datasource.NewRegistry()
	reg.Register("roads", &mockDataSource{})
	rend := &mockRenderer{}
	c, _ := cache.New(10)

	return wms.NewHandler(st, reg, rend, c, &metrics.Counters{})
}

func TestGetCapabilitiesHTTP(t *testing.T) {
	h := makeHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetCapabilities", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/xml; charset=utf-8" {
		t.Errorf("want text/xml content-type, got %s", ct)
	}
}

func TestGetMapHTTP(t *testing.T) {
	h := makeHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap"+
			"&LAYERS=roads&STYLES=&CRS=EPSG:4326"+
			"&BBOX=31,34,33,36&WIDTH=256&HEIGHT=256&FORMAT=image/png",
		nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — body: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "image/png" {
		t.Errorf("want image/png, got %s", ct)
	}
}

func TestGetMapHTTPLowercaseParams(t *testing.T) {
	h := makeHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/wms?service=WMS&version=1.3.0&request=GetMap"+
			"&layers=roads&styles=&crs=EPSG:4326"+
			"&bbox=31,34,33,36&width=256&height=256&format=image/png",
		nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d — body: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "image/png" {
		t.Errorf("want image/png, got %s", ct)
	}
}

func TestGetMapCacheHit(t *testing.T) {
	h := makeHandler(t)
	url := "/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap" +
		"&LAYERS=roads&STYLES=&CRS=EPSG:4326" +
		"&BBOX=31,34,33,36&WIDTH=256&HEIGHT=256&FORMAT=image/png"

	// First request — cache miss
	r1 := httptest.NewRequest(http.MethodGet, url, nil)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)

	// Second request — should hit cache
	r2 := httptest.NewRequest(http.MethodGet, url, nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("cache hit: want 200, got %d", w2.Code)
	}
}

func TestUnknownRequest(t *testing.T) {
	h := makeHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/wms?REQUEST=GetFeatureInfo", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for unsupported REQUEST, got %d", w.Code)
	}
}

func TestGetFeatureInfoHTTP(t *testing.T) {
	h := makeHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetFeatureInfo"+
			"&LAYERS=roads&QUERY_LAYERS=roads&STYLES=&CRS=EPSG:4326"+
			"&BBOX=31.5,35.5,32.5,36.5&WIDTH=1000&HEIGHT=1000&I=400&J=600"+
			"&INFO_FORMAT=application/json&FEATURE_COUNT=1",
		nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("want application/json, got %s", ct)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if gotLayer, _ := resp["layer"].(string); gotLayer != "roads" {
		t.Fatalf("want layer roads, got %q", gotLayer)
	}
}

func TestGetMapExpandsQueryBBoxForSymbolPadding(t *testing.T) {
	layers := []store.LayerRecord{
		{
			ID:      "l1",
			Name:    "roads",
			Title:   "Roads",
			StyleID: "s1",
			SRS:     `["EPSG:4326","EPSG:3857"]`,
			MinX:    -180, MinY: -90, MaxX: 180, MaxY: 90,
		},
	}
	st := &mockStore{layers: layers}
	reg := datasource.NewRegistry()
	rec := &recordingDataSource{}
	reg.Register("roads", rec)
	rend := &mockRenderer{}
	c, _ := cache.New(10)
	h := wms.NewHandler(st, reg, rend, c, &metrics.Counters{})

	// EPSG:4326 in WMS 1.3.0 comes as lat/lon axis order and is normalized by parser.
	req := httptest.NewRequest(http.MethodGet,
		"/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap"+
			"&LAYERS=roads&STYLES=&CRS=EPSG:4326"+
			"&BBOX=31,34,33,36&WIDTH=256&HEIGHT=256&FORMAT=image/png",
		nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Normalized bbox without padding would be [34,31,36,33].
	// We expect query bounds to be expanded to avoid dropping point symbols near tile edges.
	if rec.lastQuery.Bound.Min[0] >= 34 || rec.lastQuery.Bound.Min[1] >= 31 ||
		rec.lastQuery.Bound.Max[0] <= 36 || rec.lastQuery.Bound.Max[1] <= 33 {
		t.Fatalf("expected padded query bbox, got min=%v max=%v", rec.lastQuery.Bound.Min, rec.lastQuery.Bound.Max)
	}
}
