package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"meridian/api"
	"meridian/internal/blobstore"
	"meridian/internal/cache"
	"meridian/internal/datasource"
	"meridian/internal/datasource/file"
	"meridian/internal/datasource/postgis"
	"meridian/internal/metrics"
	"meridian/internal/observability"
	"meridian/internal/renderer"
	"meridian/internal/store"
	"meridian/internal/ui"
	"meridian/internal/wms"
)

func main() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	ring := observability.NewRingBuffer(500)
	logger := slog.New(observability.NewRingBufHandler(jsonHandler, ring))
	slog.SetDefault(logger)

	dbPath := envOr("DB_PATH", "meridian.db")
	if abs, absErr := filepath.Abs(dbPath); absErr == nil {
		dbPath = abs
	}
	st, err := store.NewSQLite(dbPath)
	if err != nil {
		slog.Error("open store", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	reg := datasource.NewRegistry()

	blobCfg := blobstore.LoadFromEnv()
	if blobCfg.Bucket != "" && !blobCfg.Enabled() {
		slog.Warn("S3_BUCKET set but S3 credentials incomplete; GeoJSON uploads will use local disk until fixed")
	}
	var blob *blobstore.Client
	if blobCfg.Enabled() {
		var berr error
		blob, berr = blobstore.NewClient(blobCfg)
		if berr != nil {
			slog.Error("init object storage", "err", berr)
			os.Exit(1)
		}
		slog.Info("object storage enabled", "bucket", blobCfg.Bucket, "endpoint", blobCfg.Endpoint)
	}

	// LayerOpener is called by the API whenever a layer is created or updated.
	// It looks up the source record and opens a live DataSource for that layer.
	opener := func(ctx context.Context, sourceID, sourceLayer string) (datasource.DataSource, error) {
		sr, err := st.GetSource(ctx, sourceID)
		if err != nil {
			return nil, err
		}
		return openSourceForLayer(ctx, sr, sourceLayer, blob)
	}

	if err := loadSources(st, reg, blob); err != nil {
		slog.Error("load sources", "err", err)
		os.Exit(1)
	}
	if err := ensureDemoEgyptLayer(context.Background(), st, reg, opener, filepath.Dir(dbPath)); err != nil {
		slog.Warn("seed demo layer", "err", err)
	}

	c, err := cache.New(512)
	if err != nil {
		slog.Error("init cache", "err", err)
		os.Exit(1)
	}

	m := metrics.Global
	rend := renderer.WithPool(renderer.NewVectorRenderer())

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(observability.RequestLogger())
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/metrics", metrics.Handler(m))
	r.Mount("/wms", wms.NewHandler(st, reg, rend, c, m))
	// UI routes must be registered before the root API mount,
	// which acts as a catch-all for all unmatched paths.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, ui.Static, "static/index.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", ui.StaticHandler()))
	uploadDir := filepath.Dir(dbPath)
	r.Mount("/", api.NewHandler(st, reg, opener, uploadDir, blob, ring))

	addr := envOr("ADDR", ":8080")
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("shutdown complete")
}

// loadSources opens a DataSource per layer (not per source) so each layer gets
// its own table config via source_layer. This also means the registry is keyed
// correctly: one entry per layer name, not per source name.
func loadSources(st store.Store, reg *datasource.Registry, blob *blobstore.Client) error {
	ctx := context.Background()

	layers, err := st.ListLayers(ctx)
	if err != nil {
		return err
	}

	for _, lr := range layers {
		sr, err := st.GetSource(ctx, lr.SourceID)
		if err != nil {
			slog.Warn("source not found for layer", "layer", lr.Name, "source_id", lr.SourceID)
			continue
		}
		src, err := openSourceForLayer(ctx, sr, lr.SourceLayer, blob)
		if err != nil {
			slog.Warn("failed to open source for layer", "layer", lr.Name, "err", err)
			continue
		}
		if src == nil {
			continue
		}
		reg.Register(lr.Name, src)
		slog.Info("registered layer", "name", lr.Name)
	}

	return nil
}

// openSourceForLayer opens a DataSource for a given SourceRecord, using
// sourceLayer to override the table name for PostGIS sources.
// This is the single place where source type dispatch lives.
func openSourceForLayer(ctx context.Context, sr store.SourceRecord, sourceLayer string, blob *blobstore.Client) (datasource.DataSource, error) {
	switch sr.Type {
	case "postgis":
		var cfg postgis.Config
		if err := json.Unmarshal(sr.Config, &cfg); err != nil {
			return nil, err
		}
		// source_layer on the layer record overrides the table in the source config.
		// This allows one PostGIS source (connection) to back multiple layers on
		// different tables.
		if sourceLayer != "" {
			cfg.Table = sourceLayer
		}
		return postgis.New(context.Background(), cfg)
	case "geojson":
		return file.OpenGeoJSONSource(ctx, sr.Config, blob)
	default:
		slog.Warn("unknown source type, skipping", "type", sr.Type)
		return nil, nil
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
