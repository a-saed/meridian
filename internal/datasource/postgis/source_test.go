//go:build integration

package postgis_test

import (
	"context"
	"os"
	"testing"

	"github.com/paulmach/orb"

	"meridian/internal/datasource"
	"meridian/internal/datasource/postgis"
)

func TestPostGISQuery(t *testing.T) {
	connStr := os.Getenv("TEST_POSTGIS_CONN")
	if connStr == "" {
		t.Skip("TEST_POSTGIS_CONN not set")
	}

	ctx := context.Background()
	cfg := postgis.Config{
		ConnString: connStr,
		Table:      "public.test_features",
		GeomColumn: "geom",
	}

	src, err := postgis.New(ctx, cfg)
	if err != nil {
		t.Fatalf("new source: %v", err)
	}
	defer src.Close()

	q := datasource.Query{
		Bound: orb.Bound{Min: orb.Point{-180, -90}, Max: orb.Point{180, 90}},
	}

	features, err := src.Query(ctx, q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	t.Logf("got %d features", len(features))
}
