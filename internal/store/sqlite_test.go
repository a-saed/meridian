package store_test

import (
	"context"
	"testing"

	"meridian/internal/store"
)

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSourceCRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	rec := store.SourceRecord{
		ID:     "src-1",
		Name:   "my_postgis",
		Type:   "postgis",
		Config: []byte(`{"conn_string":"postgres://localhost/gis"}`),
	}

	if err := s.CreateSource(ctx, rec); err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	list, err := s.ListSources(ctx)
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(list) != 1 || list[0].Name != "my_postgis" {
		t.Errorf("want 1 source named my_postgis, got %+v", list)
	}

	if err := s.DeleteSource(ctx, "src-1"); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}

	list, _ = s.ListSources(ctx)
	if len(list) != 0 {
		t.Errorf("want 0 sources after delete, got %d", len(list))
	}
}

func TestLayerCRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	layer := store.LayerRecord{
		ID:          "layer-1",
		Name:        "roads",
		Title:       "Road Network",
		SourceID:    "src-1",
		SourceLayer: "public.roads",
		StyleID:     "style-1",
		SRS:         `["EPSG:4326","EPSG:3857"]`,
		MinX:        -180, MinY: -90, MaxX: 180, MaxY: 90,
	}

	if err := s.CreateLayer(ctx, layer); err != nil {
		t.Fatalf("CreateLayer: %v", err)
	}

	got, err := s.GetLayerByName(ctx, "roads")
	if err != nil {
		t.Fatalf("GetLayerByName: %v", err)
	}
	if got.Title != "Road Network" {
		t.Errorf("want title 'Road Network', got %s", got.Title)
	}

	got.Title = "Roads"
	if err := s.UpdateLayer(ctx, got); err != nil {
		t.Fatalf("UpdateLayer: %v", err)
	}

	got2, _ := s.GetLayerByName(ctx, "roads")
	if got2.Title != "Roads" {
		t.Errorf("want updated title 'Roads', got %s", got2.Title)
	}
}

func TestStyleCRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	style := store.StyleRecord{
		ID:          "style-1",
		Name:        "blue-fill",
		FillColor:   "#3388ff",
		StrokeColor: "#ffffff",
		StrokeWidth: 1.0,
		Opacity:     0.8,
	}

	if err := s.CreateStyle(ctx, style); err != nil {
		t.Fatalf("CreateStyle: %v", err)
	}

	got, err := s.GetStyle(ctx, "style-1")
	if err != nil {
		t.Fatalf("GetStyle: %v", err)
	}
	if got.FillColor != "#3388ff" {
		t.Errorf("want FillColor #3388ff, got %s", got.FillColor)
	}
}
