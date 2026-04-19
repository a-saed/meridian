package file

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/tidwall/rtree"

	"meridian/internal/datasource"
)

// GeoJSONSource loads a GeoJSON FeatureCollection into memory and indexes
// it with an R-tree for fast bbox queries.
type GeoJSONSource struct {
	features []datasource.Feature
	index    rtree.RTreeG[int]
	bounds   [2][2]float64 // [min, max] indexed as [0]=min [1]=max, each [lon,lat]
}

// NewGeoJSONSource reads a GeoJSON file at path, parses it, and builds the R-tree index.
func NewGeoJSONSource(path string) (*GeoJSONSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("geojson: read %s: %w", path, err)
	}
	return NewGeoJSONSourceFromBytes(data)
}

// NewGeoJSONSourceFromBytes parses GeoJSON bytes (FeatureCollection) and builds the R-tree index.
func NewGeoJSONSourceFromBytes(data []byte) (*GeoJSONSource, error) {
	fc, err := geojson.UnmarshalFeatureCollection(data)
	if err != nil {
		return nil, fmt.Errorf("geojson: unmarshal: %w", err)
	}

	s := &GeoJSONSource{
		bounds: [2][2]float64{
			{math.MaxFloat64, math.MaxFloat64},
			{-math.MaxFloat64, -math.MaxFloat64},
		},
	}
	for i, f := range fc.Features {
		if f.Geometry == nil {
			continue
		}
		feat := datasource.Feature{
			Geometry:   f.Geometry,
			Properties: f.Properties,
		}
		s.features = append(s.features, feat)

		b := f.Geometry.Bound()
		s.index.Insert(
			[2]float64{b.Min[0], b.Min[1]},
			[2]float64{b.Max[0], b.Max[1]},
			i,
		)
		if b.Min[0] < s.bounds[0][0] { s.bounds[0][0] = b.Min[0] }
		if b.Min[1] < s.bounds[0][1] { s.bounds[0][1] = b.Min[1] }
		if b.Max[0] > s.bounds[1][0] { s.bounds[1][0] = b.Max[0] }
		if b.Max[1] > s.bounds[1][1] { s.bounds[1][1] = b.Max[1] }
	}

	return s, nil
}

// Bounds returns the geographic extent of all features (EPSG:4326).
func (s *GeoJSONSource) Bounds() orb.Bound {
	return orb.Bound{
		Min: orb.Point{s.bounds[0][0], s.bounds[0][1]},
		Max: orb.Point{s.bounds[1][0], s.bounds[1][1]},
	}
}

// GeomType returns the dominant geometry type of the features in this source.
func (s *GeoJSONSource) GeomType() string {
	if len(s.features) == 0 {
		return "unknown"
	}
	switch s.features[0].Geometry.GeoJSONType() {
	case "Point", "MultiPoint":
		return "point"
	case "LineString", "MultiLineString":
		return "line"
	case "Polygon", "MultiPolygon":
		return "polygon"
	default:
		return "mixed"
	}
}

// Query returns all features whose bounding box intersects q.Bound.
func (s *GeoJSONSource) Query(_ context.Context, q datasource.Query) ([]datasource.Feature, error) {
	var result []datasource.Feature
	s.index.Search(
		[2]float64{q.Bound.Min[0], q.Bound.Min[1]},
		[2]float64{q.Bound.Max[0], q.Bound.Max[1]},
		func(_, _ [2]float64, idx int) bool {
			result = append(result, s.features[idx])
			return true
		},
	)
	return result, nil
}

func (s *GeoJSONSource) Close() error { return nil }

// Ensure GeoJSONSource satisfies the DataSource interface at compile time.
var _ datasource.DataSource = (*GeoJSONSource)(nil)
