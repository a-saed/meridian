package file_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"

	"meridian/internal/datasource"
	"meridian/internal/datasource/file"
)

func TestOpenGeoJSONSource_LocalPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "x.geojson")
	data := []byte(`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[1,2]}}]}`)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(file.GeoJSONSourceConfig{Path: p})
	src, err := file.OpenGeoJSONSource(context.Background(), raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = src.Close() })
	feats, err := src.Query(context.Background(), datasource.Query{
		Bound: orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{3, 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(feats) != 1 {
		t.Fatalf("want 1 feature, got %d", len(feats))
	}
}

func TestOpenGeoJSONSource_S3RequiresGetter(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(file.GeoJSONSourceConfig{
		S3: &file.S3ObjectRef{Bucket: "b", Key: "k"},
	})
	_, err := file.OpenGeoJSONSource(context.Background(), raw, nil)
	if err == nil {
		t.Fatal("expected error when s3 config set but getter is nil")
	}
}
