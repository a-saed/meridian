package file_test

import (
	"context"
	"testing"

	"github.com/paulmach/orb"

	"meridian/internal/datasource"
	"meridian/internal/datasource/file"
)

func TestGeoJSONQuery_BboxHit(t *testing.T) {
	src, err := file.NewGeoJSONSource("testdata/sample.geojson")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()

	// BBOX covering Amman only
	q := datasource.Query{
		Bound: orb.Bound{Min: orb.Point{35.0, 31.0}, Max: orb.Point{36.5, 32.5}},
	}

	features, err := src.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(features) != 1 {
		t.Errorf("want 1 feature (Amman), got %d", len(features))
	}
}

func TestGeoJSONQuery_BboxMiss(t *testing.T) {
	src, err := file.NewGeoJSONSource("testdata/sample.geojson")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()

	// BBOX in the middle of the Atlantic — no features
	q := datasource.Query{
		Bound: orb.Bound{Min: orb.Point{-40.0, 20.0}, Max: orb.Point{-30.0, 30.0}},
	}

	features, err := src.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(features) != 0 {
		t.Errorf("want 0 features, got %d", len(features))
	}
}

func TestGeoJSONQuery_PolygonIncluded(t *testing.T) {
	src, err := file.NewGeoJSONSource("testdata/sample.geojson")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()

	// BBOX covering Region A (central Europe) + Aleppo
	q := datasource.Query{
		Bound: orb.Bound{Min: orb.Point{9.0, 35.0}, Max: orb.Point{37.0, 51.0}},
	}

	features, err := src.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	// Should include Aleppo (point) + Region A (polygon)
	if len(features) < 2 {
		t.Errorf("want >= 2 features, got %d", len(features))
	}
}
