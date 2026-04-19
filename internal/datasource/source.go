package datasource

import (
	"context"

	"github.com/paulmach/orb"
)

// Feature is a single geographic entity returned from a DataSource.
type Feature struct {
	Geometry   orb.Geometry
	Properties map[string]any
}

// Query specifies the spatial and attribute filters for a DataSource query.
type Query struct {
	Bound orb.Bound // bounding box in EPSG:4326 (lon/lat)
}

// DataSource is the interface all spatial backends must implement.
type DataSource interface {
	Query(ctx context.Context, q Query) ([]Feature, error)
	Close() error
}

// Bounder is an optional interface a DataSource may implement when it knows
// the geographic extent of its data without a full scan.
type Bounder interface {
	Bounds() orb.Bound
}

// GeomTyper is an optional interface a DataSource may implement to report
// its dominant geometry type ("point", "line", "polygon", "mixed").
type GeomTyper interface {
	GeomType() string
}
