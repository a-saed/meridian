package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"meridian/internal/blobstore"
	"meridian/internal/datasource/file"
)

type uploadHandler struct {
	dir  string
	blob *blobstore.Client
}

func newUploadHandler(dir string, blob *blobstore.Client) *uploadHandler {
	return &uploadHandler{dir: dir, blob: blob}
}

func (h *uploadHandler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	if ext != ".geojson" && ext != ".json" {
		http.Error(w, "only .geojson and .json files are accepted", http.StatusUnsupportedMediaType)
		return
	}

	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	meta, err := inspectGeoJSONFeatureCollection(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Object storage path (production)
	if h.blob != nil {
		bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
		if bucket == "" {
			http.Error(w, "S3_BUCKET not configured", http.StatusInternalServerError)
			return
		}
		key := "uploads/" + uuid.NewString() + ext
		if err := h.blob.PutObject(ctx, bucket, key, "application/geo+json", data); err != nil {
			http.Error(w, "object storage write failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		cfg := file.GeoJSONSourceConfig{
			S3: &file.S3ObjectRef{Bucket: bucket, Key: key},
			Meta: &file.GeoJSONMeta{
				FeatureCount:   meta.FeatureCount,
				ChecksumSHA256: meta.ChecksumSHA256,
				ContentType:    "application/geo+json",
				SizeBytes:      len(data),
			},
		}
		raw, _ := json.Marshal(cfg)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"config": cfg,
			// convenience for older clients
			"path": "",
			"raw":  json.RawMessage(raw),
		})
		return
	}

	// Local filesystem fallback (PoC / dev without object storage)
	base := filepath.Base(hdr.Filename)
	dest := filepath.Join(h.dir, base)
	if _, statErr := os.Stat(dest); statErr == nil {
		stem := strings.TrimSuffix(base, ext)
		b := make([]byte, 2)
		if _, randErr := rand.Read(b); randErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		dest = filepath.Join(h.dir, stem+"_"+hex.EncodeToString(b)+ext)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	cfg := file.GeoJSONSourceConfig{
		Path: abs,
		Meta: &file.GeoJSONMeta{
			FeatureCount:   meta.FeatureCount,
			ChecksumSHA256: meta.ChecksumSHA256,
			ContentType:    "application/geo+json",
			SizeBytes:      len(data),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"path":   abs,
		"config": cfg,
	})
}
