package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"meridian/internal/store"
)

type stylesHandler struct{ store store.Store }

func (h *stylesHandler) create(w http.ResponseWriter, r *http.Request) {
	var req store.StyleRecord
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	req.ID = uuid.NewString()
	if req.FillColor == "" {
		req.FillColor = "#3388ff"
	}
	if req.StrokeColor == "" {
		req.StrokeColor = "#ffffff"
	}
	if req.StrokeWidth == 0 {
		req.StrokeWidth = 1.0
	}
	if req.Opacity == 0 {
		req.Opacity = 1.0
	}

	if err := h.store.CreateStyle(r.Context(), req); err != nil {
		http.Error(w, "create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (h *stylesHandler) list(w http.ResponseWriter, r *http.Request) {
	styles, err := h.store.ListStyles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if styles == nil {
		styles = []store.StyleRecord{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(styles)
}

func (h *stylesHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteStyle(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
