package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"meridian/internal/datasource"
	"meridian/internal/store"
)

func TestEnsureDemoEgyptLayerSeedsDefaults(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "meridian.db")
	st, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	reg := datasource.NewRegistry()
	ctx := context.Background()

	opener := func(ctx context.Context, sourceID, sourceLayer string) (datasource.DataSource, error) {
		sr, err := st.GetSource(ctx, sourceID)
		if err != nil {
			return nil, err
		}
		return openSourceForLayer(ctx, sr, sourceLayer, nil)
	}

	if err := ensureDemoEgyptLayer(ctx, st, reg, opener, t.TempDir()); err != nil {
		t.Fatalf("seed demo layer: %v", err)
	}

	layer, err := st.GetLayerByName(ctx, demoEgyptLayerName)
	if err != nil {
		t.Fatalf("demo layer not created: %v", err)
	}
	if layer.SourceID == "" {
		t.Fatalf("demo layer source id is empty")
	}

	src, err := st.GetSource(ctx, layer.SourceID)
	if err != nil {
		t.Fatalf("get demo source: %v", err)
	}
	if src.Name != demoEgyptSourceName {
		t.Fatalf("unexpected source name: %s", src.Name)
	}

	var cfg struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		t.Fatalf("decode source config: %v", err)
	}
	if cfg.Path == "" {
		t.Fatalf("demo source path is empty")
	}
	if _, ok := reg.Resolve(demoEgyptLayerName); !ok {
		t.Fatalf("demo layer was not registered in datasource registry")
	}
}

func TestEnsureDemoEgyptLayerIsIdempotent(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "meridian.db")
	st, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	reg := datasource.NewRegistry()
	ctx := context.Background()

	opener := func(ctx context.Context, sourceID, sourceLayer string) (datasource.DataSource, error) {
		sr, err := st.GetSource(ctx, sourceID)
		if err != nil {
			return nil, err
		}
		return openSourceForLayer(ctx, sr, sourceLayer, nil)
	}

	dataDir := t.TempDir()
	if err := ensureDemoEgyptLayer(ctx, st, reg, opener, dataDir); err != nil {
		t.Fatalf("seed demo layer (first): %v", err)
	}
	if err := ensureDemoEgyptLayer(ctx, st, reg, opener, dataDir); err != nil {
		t.Fatalf("seed demo layer (second): %v", err)
	}

	layers, err := st.ListLayers(ctx)
	if err != nil {
		t.Fatalf("list layers: %v", err)
	}
	if len(layers) != 1 {
		t.Fatalf("expected 1 layer after idempotent seed, got %d", len(layers))
	}

	sources, err := st.ListSources(ctx)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source after idempotent seed, got %d", len(sources))
	}

	styles, err := st.ListStyles(ctx)
	if err != nil {
		t.Fatalf("list styles: %v", err)
	}
	if len(styles) != 1 {
		t.Fatalf("expected 1 style after idempotent seed, got %d", len(styles))
	}
}
