package postgis

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb/encoding/wkb"

	"meridian/internal/datasource"
)

// Config holds PostGIS connection parameters.
type Config struct {
	ConnString string `json:"conn_string"`
	Table      string `json:"table"`
	GeomColumn string `json:"geom_column"` // defaults to "geom"
}

// Source queries features from a PostGIS table.
type Source struct {
	pool   *pgxpool.Pool
	config Config
}

// New creates a Source and opens the connection pool.
func New(ctx context.Context, cfg Config) (*Source, error) {
	if cfg.GeomColumn == "" {
		cfg.GeomColumn = "geom"
	}
	pool, err := pgxpool.New(ctx, cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("postgis: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgis: ping: %w", err)
	}
	return &Source{pool: pool, config: cfg}, nil
}

// Query returns features from the configured table that intersect q.Bound (EPSG:4326).
func (s *Source) Query(ctx context.Context, q datasource.Query) ([]datasource.Feature, error) {
	col := s.config.GeomColumn
	sql := fmt.Sprintf(`
		SELECT ST_AsBinary(ST_Transform(ST_Force2D(%s), 4326))
		FROM %s
		WHERE %s && ST_MakeEnvelope($1, $2, $3, $4, 4326)
		  AND ST_Intersects(%s, ST_MakeEnvelope($1, $2, $3, $4, 4326))
	`, col, s.config.Table, col, col)

	rows, err := s.pool.Query(ctx, sql,
		q.Bound.Min[0], q.Bound.Min[1],
		q.Bound.Max[0], q.Bound.Max[1],
	)
	if err != nil {
		return nil, fmt.Errorf("postgis: query: %w", err)
	}
	defer rows.Close()

	var features []datasource.Feature
	for rows.Next() {
		var wkbBytes []byte
		if err := rows.Scan(&wkbBytes); err != nil {
			return nil, fmt.Errorf("postgis: scan: %w", err)
		}
		geom, err := wkb.Unmarshal(wkbBytes)
		if err != nil {
			return nil, fmt.Errorf("postgis: decode wkb: %w", err)
		}
		features = append(features, datasource.Feature{Geometry: geom})
	}
	return features, rows.Err()
}

// Close releases the connection pool.
func (s *Source) Close() error {
	s.pool.Close()
	return nil
}

var _ datasource.DataSource = (*Source)(nil)
