package proj_test

import (
	"math"
	"testing"

	"meridian/internal/proj"
)

const eps = 0.01 // tolerance in meters / degrees

func TestTo3857RoundTrip(t *testing.T) {
	lon, lat := 35.9106, 31.9539 // Amman, Jordan
	x, y := proj.To3857(lon, lat)
	gotLon, gotLat := proj.To4326(x, y)
	if math.Abs(gotLon-lon) > eps {
		t.Errorf("lon round-trip: want %f, got %f", lon, gotLon)
	}
	if math.Abs(gotLat-lat) > eps {
		t.Errorf("lat round-trip: want %f, got %f", lat, gotLat)
	}
}

func TestTo3857KnownValues(t *testing.T) {
	// Origin: lon=0, lat=0 → x=0, y=0
	x, y := proj.To3857(0, 0)
	if math.Abs(x) > eps || math.Abs(y) > eps {
		t.Errorf("origin: want (0,0), got (%f,%f)", x, y)
	}
}

func TestBoundTo3857(t *testing.T) {
	b := proj.BoundTo3857([4]float64{-180, -85.051129, 180, 85.051129})
	// Web Mercator world extent ≈ ±20037508
	if math.Abs(math.Abs(b[0])-20037508.34) > 1 {
		t.Errorf("expected ~±20037508, got %f", b[0])
	}
}
