package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"meridian/internal/datasource"
	"meridian/internal/store"
)

const (
	demoEgyptLayerName  = "test_layer"
	demoEgyptSourceName = "test_layer_geojson"
	demoEgyptStyleName  = "test_layer_style"
	demoEgyptGeoJSON    = `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {
        "name": "Cairo"
      },
      "geometry": {
        "type": "Point",
        "coordinates": [31.2357, 30.0444]
      }
    },
    {
      "type": "Feature",
      "properties": {
        "name": "Alexandria"
      },
      "geometry": {
        "type": "Point",
        "coordinates": [29.9187, 31.2001]
      }
    },
    {
      "type": "Feature",
      "properties": {
        "name": "Giza Plateau Area"
      },
      "geometry": {
        "type": "Polygon",
        "coordinates": [
          [
            [30.95, 29.95],
            [31.55, 29.95],
            [31.55, 30.35],
            [30.95, 30.35],
            [30.95, 29.95]
          ]
        ]
      }
    }
  ]
}`
)

func ensureDemoEgyptLayer(
	ctx context.Context,
	st store.Store,
	reg *datasource.Registry,
	opener func(context.Context, string, string) (datasource.DataSource, error),
	dataDir string,
) error {
	sourceID, err := ensureDemoSource(ctx, st, dataDir)
	if err != nil {
		return err
	}
	styleID, err := ensureDemoStyle(ctx, st)
	if err != nil {
		return err
	}

	srsJSON, err := json.Marshal([]string{"EPSG:4326", "EPSG:3857"})
	if err != nil {
		return fmt.Errorf("marshal demo srs: %w", err)
	}

	layer, err := st.GetLayerByName(ctx, demoEgyptLayerName)
	if err == nil {
		layer.Title = "Egypt Test Layer"
		layer.SourceID = sourceID
		layer.SourceLayer = ""
		layer.StyleID = styleID
		layer.SRS = string(srsJSON)
		layer.MinX = 24.7
		layer.MinY = 22.0
		layer.MaxX = 36.9
		layer.MaxY = 31.9
		if err := st.UpdateLayer(ctx, layer); err != nil {
			return fmt.Errorf("update test layer: %w", err)
		}
		return registerLayerSource(ctx, reg, opener, layer)
	}
	if !isNotFoundErr(err) {
		return fmt.Errorf("check test layer: %w", err)
	}

	layer = store.LayerRecord{
		ID:       uuid.NewString(),
		Name:     demoEgyptLayerName,
		Title:    "Egypt Test Layer",
		SourceID: sourceID,
		StyleID:  styleID,
		SRS:      string(srsJSON),
		MinX:     24.7,
		MinY:     22.0,
		MaxX:     36.9,
		MaxY:     31.9,
	}
	if err := st.CreateLayer(ctx, layer); err != nil {
		return fmt.Errorf("create test layer: %w", err)
	}
	return registerLayerSource(ctx, reg, opener, layer)
}

func ensureDemoSource(ctx context.Context, st store.Store, dataDir string) (string, error) {
	sources, err := st.ListSources(ctx)
	if err != nil {
		return "", fmt.Errorf("list sources: %w", err)
	}
	for _, s := range sources {
		if s.Name == demoEgyptSourceName {
			return s.ID, nil
		}
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir demo data dir: %w", err)
	}
	geojsonPath := filepath.Join(dataDir, "test_layer_egypt.geojson")
	if err := os.WriteFile(geojsonPath, []byte(demoEgyptGeoJSON), 0o644); err != nil {
		return "", fmt.Errorf("write demo geojson: %w", err)
	}

	cfg, err := json.Marshal(struct {
		Path string `json:"path"`
	}{Path: geojsonPath})
	if err != nil {
		return "", fmt.Errorf("marshal demo source config: %w", err)
	}

	rec := store.SourceRecord{
		ID:     uuid.NewString(),
		Name:   demoEgyptSourceName,
		Type:   "geojson",
		Config: cfg,
	}
	if err := st.CreateSource(ctx, rec); err != nil {
		return "", fmt.Errorf("create demo source: %w", err)
	}
	return rec.ID, nil
}

func ensureDemoStyle(ctx context.Context, st store.Store) (string, error) {
	styles, err := st.ListStyles(ctx)
	if err != nil {
		return "", fmt.Errorf("list styles: %w", err)
	}
	for _, s := range styles {
		if s.Name == demoEgyptStyleName {
			return s.ID, nil
		}
	}

	rec := store.StyleRecord{
		ID:          uuid.NewString(),
		Name:        demoEgyptStyleName,
		FillColor:   "#ffcc00",
		StrokeColor: "#8a3b12",
		StrokeWidth: 1.4,
		Opacity:     0.85,
	}
	if err := st.CreateStyle(ctx, rec); err != nil {
		return "", fmt.Errorf("create demo style: %w", err)
	}
	return rec.ID, nil
}

func registerLayerSource(
	ctx context.Context,
	reg *datasource.Registry,
	opener func(context.Context, string, string) (datasource.DataSource, error),
	layer store.LayerRecord,
) error {
	src, err := opener(ctx, layer.SourceID, layer.SourceLayer)
	if err != nil {
		return fmt.Errorf("open demo source: %w", err)
	}
	if src == nil {
		return errors.New("demo source returned nil datasource")
	}
	if old := reg.Swap(layer.Name, src); old != nil {
		_ = old.Close()
	}
	return nil
}

func isNotFoundErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}
