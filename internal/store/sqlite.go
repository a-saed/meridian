package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

// NewSQLite opens (or creates) a SQLite database at path and runs migrations.
func NewSQLite(path string) (Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}
	return &sqliteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sources (
			id     TEXT PRIMARY KEY,
			name   TEXT NOT NULL UNIQUE,
			type   TEXT NOT NULL,
			config BLOB NOT NULL
		);
		CREATE TABLE IF NOT EXISTS layers (
			id           TEXT PRIMARY KEY,
			name         TEXT NOT NULL UNIQUE,
			title        TEXT NOT NULL,
			source_id    TEXT NOT NULL,
			source_layer TEXT NOT NULL DEFAULT '',
			style_id     TEXT NOT NULL DEFAULT '',
			srs          TEXT NOT NULL DEFAULT '["EPSG:4326"]',
			min_x        REAL NOT NULL DEFAULT -180,
			min_y        REAL NOT NULL DEFAULT -90,
			max_x        REAL NOT NULL DEFAULT 180,
			max_y        REAL NOT NULL DEFAULT 90
		);
		CREATE TABLE IF NOT EXISTS styles (
			id           TEXT PRIMARY KEY,
			name         TEXT NOT NULL UNIQUE,
			fill_color   TEXT NOT NULL DEFAULT '#3388ff',
			stroke_color TEXT NOT NULL DEFAULT '#ffffff',
			stroke_width REAL NOT NULL DEFAULT 1.0,
			opacity      REAL NOT NULL DEFAULT 1.0
		);
	`)
	return err
}

func (s *sqliteStore) Close() error { return s.db.Close() }

// --- Sources ---

func (s *sqliteStore) CreateSource(ctx context.Context, r SourceRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sources (id, name, type, config) VALUES (?, ?, ?, ?)`,
		r.ID, r.Name, r.Type, r.Config,
	)
	return err
}

func (s *sqliteStore) ListSources(ctx context.Context) ([]SourceRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, type, config FROM sources`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SourceRecord
	for rows.Next() {
		var r SourceRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.Config); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetSource(ctx context.Context, id string) (SourceRecord, error) {
	var r SourceRecord
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, type, config FROM sources WHERE id = ?`, id,
	).Scan(&r.ID, &r.Name, &r.Type, &r.Config)
	if err == sql.ErrNoRows {
		return r, fmt.Errorf("source %q not found", id)
	}
	return r, err
}

func (s *sqliteStore) DeleteSource(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sources WHERE id = ?`, id)
	return err
}

// --- Layers ---

func (s *sqliteStore) CreateLayer(ctx context.Context, l LayerRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO layers (id, name, title, source_id, source_layer, style_id, srs, min_x, min_y, max_x, max_y)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.Name, l.Title, l.SourceID, l.SourceLayer, l.StyleID, l.SRS,
		l.MinX, l.MinY, l.MaxX, l.MaxY,
	)
	return err
}

func (s *sqliteStore) ListLayers(ctx context.Context) ([]LayerRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, title, source_id, source_layer, style_id, srs, min_x, min_y, max_x, max_y FROM layers`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LayerRecord
	for rows.Next() {
		var l LayerRecord
		if err := rows.Scan(&l.ID, &l.Name, &l.Title, &l.SourceID, &l.SourceLayer,
			&l.StyleID, &l.SRS, &l.MinX, &l.MinY, &l.MaxX, &l.MaxY); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetLayerByName(ctx context.Context, name string) (LayerRecord, error) {
	var l LayerRecord
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, title, source_id, source_layer, style_id, srs, min_x, min_y, max_x, max_y
		 FROM layers WHERE name = ?`, name,
	).Scan(&l.ID, &l.Name, &l.Title, &l.SourceID, &l.SourceLayer,
		&l.StyleID, &l.SRS, &l.MinX, &l.MinY, &l.MaxX, &l.MaxY)
	if err == sql.ErrNoRows {
		return l, fmt.Errorf("layer %q not found", name)
	}
	return l, err
}

func (s *sqliteStore) UpdateLayer(ctx context.Context, l LayerRecord) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE layers SET title=?, source_id=?, source_layer=?, style_id=?, srs=?,
		 min_x=?, min_y=?, max_x=?, max_y=? WHERE id=?`,
		l.Title, l.SourceID, l.SourceLayer, l.StyleID, l.SRS,
		l.MinX, l.MinY, l.MaxX, l.MaxY, l.ID,
	)
	return err
}

func (s *sqliteStore) DeleteLayer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM layers WHERE id = ?`, id)
	return err
}

// --- Styles ---

func (s *sqliteStore) CreateStyle(ctx context.Context, r StyleRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO styles (id, name, fill_color, stroke_color, stroke_width, opacity)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.FillColor, r.StrokeColor, r.StrokeWidth, r.Opacity,
	)
	return err
}

func (s *sqliteStore) ListStyles(ctx context.Context) ([]StyleRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, fill_color, stroke_color, stroke_width, opacity FROM styles`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StyleRecord
	for rows.Next() {
		var r StyleRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.FillColor, &r.StrokeColor, &r.StrokeWidth, &r.Opacity); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetStyle(ctx context.Context, id string) (StyleRecord, error) {
	var r StyleRecord
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, fill_color, stroke_color, stroke_width, opacity FROM styles WHERE id = ?`, id,
	).Scan(&r.ID, &r.Name, &r.FillColor, &r.StrokeColor, &r.StrokeWidth, &r.Opacity)
	if err == sql.ErrNoRows {
		return r, fmt.Errorf("style %q not found", id)
	}
	return r, err
}

func (s *sqliteStore) DeleteStyle(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM styles WHERE id = ?`, id)
	return err
}
