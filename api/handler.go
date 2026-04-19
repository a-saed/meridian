package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"meridian/internal/blobstore"
	"meridian/internal/datasource"
	"meridian/internal/observability"
	"meridian/internal/store"
)

// LayerOpener opens a live DataSource for the given source ID and source layer name.
// It is called by the API whenever a layer is created or updated, so the registry
// stays in sync without a server restart.
type LayerOpener func(ctx context.Context, sourceID, sourceLayer string) (datasource.DataSource, error)

// NewHandler builds and returns the admin REST API router.
// uploadDir is the directory where uploaded GeoJSON files are saved when object storage is disabled.
// blob may be nil; when non-nil, GeoJSON uploads are stored in S3-compatible object storage.
func NewHandler(st store.Store, reg *datasource.Registry, opener LayerOpener, uploadDir string, blob *blobstore.Client, ring *observability.RingBuffer) http.Handler {
	r := chi.NewRouter()

	uh := newUploadHandler(uploadDir, blob)
	r.Post("/api/v1/upload", uh.upload)

	sr := &sourcesHandler{store: st}
	r.Post("/api/v1/sources", sr.create)
	r.Get("/api/v1/sources", sr.list)
	r.Delete("/api/v1/sources/{id}", sr.delete)

	lr := &layersHandler{store: st, registry: reg, opener: opener}
	r.Post("/api/v1/layers", lr.create)
	r.Get("/api/v1/layers", lr.list)
	r.Put("/api/v1/layers/{id}", lr.update)
	r.Delete("/api/v1/layers/{id}", lr.delete)

	sth := &stylesHandler{store: st}
	r.Post("/api/v1/styles", sth.create)
	r.Get("/api/v1/styles", sth.list)
	r.Delete("/api/v1/styles/{id}", sth.delete)

	r.Get("/api/v1/logs", (&logsHandler{ring: ring}).list)

	return r
}
