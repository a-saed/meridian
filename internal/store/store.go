package store

import "context"

// SourceRecord represents a registered data source.
type SourceRecord struct {
	ID     string
	Name   string
	Type   string // "postgis" | "geojson"
	Config []byte // JSON-encoded type-specific config
}

// LayerRecord represents a WMS-published layer.
type LayerRecord struct {
	ID          string
	Name        string  // used in WMS LAYERS= param
	Title       string  // human-readable label
	SourceID    string
	SourceLayer string  // table name for PostGIS, ignored for file sources
	StyleID     string
	SRS         string  // JSON array e.g. `["EPSG:4326","EPSG:3857"]`
	MinX, MinY, MaxX, MaxY float64
}

// StyleRecord holds rendering parameters for a layer.
type StyleRecord struct {
	ID          string
	Name        string
	FillColor   string
	StrokeColor string
	StrokeWidth float64
	Opacity     float64
}

// Store is the persistence interface for all config entities.
type Store interface {
	CreateSource(ctx context.Context, s SourceRecord) error
	ListSources(ctx context.Context) ([]SourceRecord, error)
	GetSource(ctx context.Context, id string) (SourceRecord, error)
	DeleteSource(ctx context.Context, id string) error

	CreateLayer(ctx context.Context, l LayerRecord) error
	ListLayers(ctx context.Context) ([]LayerRecord, error)
	GetLayerByName(ctx context.Context, name string) (LayerRecord, error)
	UpdateLayer(ctx context.Context, l LayerRecord) error
	DeleteLayer(ctx context.Context, id string) error

	CreateStyle(ctx context.Context, s StyleRecord) error
	ListStyles(ctx context.Context) ([]StyleRecord, error)
	GetStyle(ctx context.Context, id string) (StyleRecord, error)
	DeleteStyle(ctx context.Context, id string) error

	Close() error
}
