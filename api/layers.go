package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"meridian/internal/datasource"
	"meridian/internal/store"
)

type layersHandler struct {
	store    store.Store
	registry *datasource.Registry
	opener   LayerOpener
}

type createLayerRequest struct {
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	SourceID    string    `json:"source_id"`
	SourceLayer string    `json:"source_layer"`
	StyleID     string    `json:"style_id"`
	SRS         []string  `json:"srs"`
	Bbox        []float64 `json:"bbox"` // [minX, minY, maxX, maxY]
}

func (h *layersHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createLayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.SourceID == "" {
		http.Error(w, "name and source_id are required", http.StatusBadRequest)
		return
	}

	srs := req.SRS
	if len(srs) == 0 {
		srs = []string{"EPSG:4326"}
	}
	srsJSON, _ := json.Marshal(srs)

	bbox := req.Bbox
	if len(bbox) != 4 {
		bbox = []float64{-180, -90, 180, 90}
	}

	rec := store.LayerRecord{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Title:       req.Title,
		SourceID:    req.SourceID,
		SourceLayer: req.SourceLayer,
		StyleID:     req.StyleID,
		SRS:         string(srsJSON),
		MinX:        bbox[0], MinY: bbox[1], MaxX: bbox[2], MaxY: bbox[3],
	}

	src, err := h.openLayerSource(r.Context(), rec)
	if err != nil {
		http.Error(w, "invalid layer source: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.CreateLayer(r.Context(), rec); err != nil {
		_ = src.Close()
		http.Error(w, "create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Register the live datasource so the layer is queryable immediately.
	// If the source knows its bounds and the user left bbox at world-default,
	// update the layer record with the actual data extent.
	h.swapLayerSource(rec, src)
	if b, ok := src.(datasource.Bounder); ok && isWorldBbox(rec) {
		bound := b.Bounds()
		rec.MinX, rec.MinY = bound.Min[0], bound.Min[1]
		rec.MaxX, rec.MaxY = bound.Max[0], bound.Max[1]
		_ = h.store.UpdateLayer(r.Context(), rec)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rec)
}

// layerResponse extends LayerRecord with runtime-derived fields.
type layerResponse struct {
	store.LayerRecord
	GeomType string `json:"GeomType"` // "point" | "line" | "polygon" | "mixed" | ""
}

func (h *layersHandler) list(w http.ResponseWriter, r *http.Request) {
	layers, err := h.store.ListLayers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if layers == nil {
		layers = []store.LayerRecord{}
	}
	resp := make([]layerResponse, len(layers))
	for i, l := range layers {
		resp[i] = layerResponse{LayerRecord: l}
		if src, ok := h.registry.Resolve(l.Name); ok {
			if gt, ok := src.(datasource.GeomTyper); ok {
				resp[i].GeomType = gt.GeomType()
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *layersHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req createLayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	srsJSON, _ := json.Marshal(req.SRS)
	bbox := req.Bbox
	if len(bbox) != 4 {
		bbox = []float64{-180, -90, 180, 90}
	}

	rec := store.LayerRecord{
		ID:          id,
		Name:        req.Name,
		Title:       req.Title,
		SourceID:    req.SourceID,
		SourceLayer: req.SourceLayer,
		StyleID:     req.StyleID,
		SRS:         string(srsJSON),
		MinX:        bbox[0], MinY: bbox[1], MaxX: bbox[2], MaxY: bbox[3],
	}

	src, err := h.openLayerSource(r.Context(), rec)
	if err != nil {
		http.Error(w, "invalid layer source: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateLayer(r.Context(), rec); err != nil {
		_ = src.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Swap in a fresh datasource; close the old one.
	// Also refresh bbox from source if it was left at world-default.
	h.swapLayerSource(rec, src)
	if b, ok := src.(datasource.Bounder); ok && isWorldBbox(rec) {
		bound := b.Bounds()
		rec.MinX, rec.MinY = bound.Min[0], bound.Min[1]
		rec.MaxX, rec.MaxY = bound.Max[0], bound.Max[1]
		_ = h.store.UpdateLayer(r.Context(), rec)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

func (h *layersHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Look up the layer name before deleting so we can unregister it.
	layers, err := h.store.ListLayers(r.Context())
	if err == nil {
		for _, l := range layers {
			if l.ID == id {
				if old := h.registry.Unregister(l.Name); old != nil {
					old.Close()
				}
				break
			}
		}
	}

	if err := h.store.DeleteLayer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isWorldBbox returns true when the layer bbox is still the world-default,
// meaning no explicit bbox was provided by the user.
func isWorldBbox(rec store.LayerRecord) bool {
	return rec.MinX == -180 && rec.MinY == -90 && rec.MaxX == 180 && rec.MaxY == 90
}

// openLayerSource validates layer source settings by opening a live datasource.
func (h *layersHandler) openLayerSource(ctx context.Context, rec store.LayerRecord) (datasource.DataSource, error) {
	src, err := h.opener(ctx, rec.SourceID, rec.SourceLayer)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, errors.New("source could not be opened")
	}
	return src, nil
}

// swapLayerSource registers a validated datasource and closes any previous one.
func (h *layersHandler) swapLayerSource(rec store.LayerRecord, src datasource.DataSource) {
	if old := h.registry.Swap(rec.Name, src); old != nil {
		old.Close()
	}
	slog.Info("registered layer", "name", rec.Name)
}
