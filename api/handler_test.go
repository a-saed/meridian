package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulmach/orb"

	"meridian/api"
	"meridian/internal/datasource"
	"meridian/internal/observability"
	"meridian/internal/store"
)

type mockStore struct {
	sources []store.SourceRecord
	layers  []store.LayerRecord
	styles  []store.StyleRecord
}

func (m *mockStore) CreateSource(_ context.Context, s store.SourceRecord) error {
	m.sources = append(m.sources, s)
	return nil
}
func (m *mockStore) ListSources(_ context.Context) ([]store.SourceRecord, error) {
	return m.sources, nil
}
func (m *mockStore) GetSource(_ context.Context, id string) (store.SourceRecord, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return s, nil
		}
	}
	return store.SourceRecord{}, fmt.Errorf("not found")
}
func (m *mockStore) DeleteSource(_ context.Context, id string) error {
	for i, s := range m.sources {
		if s.ID == id {
			m.sources = append(m.sources[:i], m.sources[i+1:]...)
			return nil
		}
	}
	return nil
}
func (m *mockStore) CreateLayer(_ context.Context, l store.LayerRecord) error {
	m.layers = append(m.layers, l)
	return nil
}
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
func (m *mockStore) UpdateLayer(_ context.Context, l store.LayerRecord) error {
	for i, existing := range m.layers {
		if existing.ID == l.ID {
			m.layers[i] = l
			return nil
		}
	}
	return nil
}
func (m *mockStore) DeleteLayer(_ context.Context, id string) error {
	for i, l := range m.layers {
		if l.ID == id {
			m.layers = append(m.layers[:i], m.layers[i+1:]...)
		}
	}
	return nil
}
func (m *mockStore) CreateStyle(_ context.Context, s store.StyleRecord) error {
	m.styles = append(m.styles, s)
	return nil
}
func (m *mockStore) ListStyles(_ context.Context) ([]store.StyleRecord, error) {
	return m.styles, nil
}
func (m *mockStore) GetStyle(_ context.Context, id string) (store.StyleRecord, error) {
	for _, s := range m.styles {
		if s.ID == id {
			return s, nil
		}
	}
	return store.StyleRecord{}, fmt.Errorf("not found")
}
func (m *mockStore) DeleteStyle(_ context.Context, id string) error {
	for i, s := range m.styles {
		if s.ID == id {
			m.styles = append(m.styles[:i], m.styles[i+1:]...)
		}
	}
	return nil
}
func (m *mockStore) Close() error { return nil }

type mockDataSource struct{}

func (m *mockDataSource) Query(_ context.Context, _ datasource.Query) ([]datasource.Feature, error) {
	return []datasource.Feature{{Geometry: orb.Point{31.2, 30.0}}}, nil
}
func (m *mockDataSource) Close() error { return nil }

func noopOpener(_ context.Context, _, _ string) (datasource.DataSource, error) {
	return &mockDataSource{}, nil
}
func errOpener(_ context.Context, _, _ string) (datasource.DataSource, error) {
	return nil, fmt.Errorf("open failed")
}

func TestCreateAndListSources(t *testing.T) {
	st := &mockStore{}
	h := api.NewHandler(st, datasource.NewRegistry(), noopOpener, t.TempDir(), nil, nil)

	body, _ := json.Marshal(map[string]any{
		"name":   "my_db",
		"type":   "postgis",
		"config": map[string]string{"conn_string": "postgres://localhost/gis"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d — %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)

	var sources []map[string]any
	json.NewDecoder(w2.Body).Decode(&sources)
	if len(sources) != 1 {
		t.Errorf("want 1 source, got %d", len(sources))
	}
}

func TestCreateAndListLayers(t *testing.T) {
	st := &mockStore{}
	h := api.NewHandler(st, datasource.NewRegistry(), noopOpener, t.TempDir(), nil, nil)

	body, _ := json.Marshal(map[string]any{
		"name":         "roads",
		"title":        "Road Network",
		"source_id":    "src-1",
		"source_layer": "public.roads",
		"style_id":     "style-1",
		"srs":          []string{"EPSG:4326"},
		"bbox":         []float64{-180, -90, 180, 90},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/layers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d — %s", w.Code, w.Body.String())
	}
}

func TestCreateLayerFailsWhenSourceCannotBeOpened(t *testing.T) {
	st := &mockStore{}
	h := api.NewHandler(st, datasource.NewRegistry(), errOpener, t.TempDir(), nil, nil)

	body, _ := json.Marshal(map[string]any{
		"name":         "broken",
		"title":        "Broken Layer",
		"source_id":    "src-1",
		"source_layer": "",
		"style_id":     "",
		"srs":          []string{"EPSG:4326"},
		"bbox":         []float64{-180, -90, 180, 90},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/layers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d — %s", w.Code, w.Body.String())
	}
	if got := len(st.layers); got != 0 {
		t.Fatalf("expected no layer persisted on opener failure, got %d", got)
	}
}

func TestUpload(t *testing.T) {
	dir := t.TempDir()
	h := api.NewHandler(&mockStore{}, datasource.NewRegistry(), noopOpener, dir, nil, nil)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "roads.geojson")
	fw.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	path, _ := resp["path"].(string)
	if path == "" {
		t.Fatal("expected non-empty path in response")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("uploaded file not found at %s: %v", path, err)
	}
}

func TestUploadRejectsBadExtension(t *testing.T) {
	dir := t.TempDir()
	h := api.NewHandler(&mockStore{}, datasource.NewRegistry(), noopOpener, dir, nil, nil)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "malware.exe")
	fw.Write([]byte("not geojson"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("want 415, got %d", w.Code)
	}
}

func TestUploadMissingFile(t *testing.T) {
	dir := t.TempDir()
	h := api.NewHandler(&mockStore{}, datasource.NewRegistry(), noopOpener, dir, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", bytes.NewReader([]byte("--boundary--\r\n")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestUpload_InvalidGeoJSON(t *testing.T) {
	dir := t.TempDir()
	h := api.NewHandler(&mockStore{}, datasource.NewRegistry(), noopOpener, dir, nil, nil)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "bad.geojson")
	fw.Write([]byte(`{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]}}`))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	return &mockStore{}
}

func TestLogsEndpoint_Basic(t *testing.T) {
	ring := observability.NewRingBuffer(10)
	ring.Add(observability.LogEntry{
		Time:  time.Now().UTC(),
		Level: "INFO",
		Msg:   "test-message",
		Attrs: map[string]any{"key": "value"},
	})

	st := newTestStore(t)
	h := api.NewHandler(st, datasource.NewRegistry(), noopOpener, t.TempDir(), nil, ring)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Entries []struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		} `json:"entries"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Msg != "test-message" {
		t.Errorf("want test-message, got %s", resp.Entries[0].Msg)
	}
	if resp.Total != 1 {
		t.Errorf("want total=1, got %d", resp.Total)
	}
}

func TestLogsEndpoint_LevelFilter(t *testing.T) {
	ring := observability.NewRingBuffer(10)
	ring.Add(observability.LogEntry{Level: "INFO", Msg: "info-line"})
	ring.Add(observability.LogEntry{Level: "WARN", Msg: "warn-line"})
	ring.Add(observability.LogEntry{Level: "ERROR", Msg: "error-line"})

	st := newTestStore(t)
	h := api.NewHandler(st, datasource.NewRegistry(), noopOpener, t.TempDir(), nil, ring)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?level=WARN", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp struct {
		Entries []struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		} `json:"entries"`
		Total int `json:"total"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Total != 3 {
		t.Errorf("total should count all entries, got %d", resp.Total)
	}
	for _, e := range resp.Entries {
		if e.Level == "INFO" {
			t.Errorf("INFO entry should be filtered out, entries: %+v", resp.Entries)
		}
	}
	if len(resp.Entries) != 2 {
		t.Errorf("want 2 entries (WARN+ERROR), got %d", len(resp.Entries))
	}
}
