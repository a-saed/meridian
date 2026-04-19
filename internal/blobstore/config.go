package blobstore

import (
	"fmt"
	"os"
	"strings"
)

// Config holds optional S3-compatible object storage settings.
// When Bucket is empty, blob storage is disabled and uploads fall back to local disk.
type Config struct {
	Endpoint       string // e.g. https://minio:9000 (empty = AWS default)
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	KeyPrefix    string // logical prefix for object keys, e.g. "meridian/geojson"
	UsePathStyle bool   // path-style URLs (required for most custom S3 endpoints)
}

// LoadFromEnv reads blob store settings from environment variables.
//
// S3_BUCKET — if set, object storage is enabled for GeoJSON uploads and sources.
// S3_REGION (default us-east-1)
// S3_ENDPOINT — custom endpoint for MinIO / GCS S3 interop
// S3_ACCESS_KEY, S3_SECRET_KEY
// S3_KEY_PREFIX — optional key prefix
// S3_USE_PATH_STYLE — "1" or "true" for path-style addressing
func LoadFromEnv() Config {
	c := Config{
		Region:    envOr("S3_REGION", "us-east-1"),
		Bucket:    strings.TrimSpace(os.Getenv("S3_BUCKET")),
		Endpoint:  strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		AccessKey: strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		SecretKey: strings.TrimSpace(os.Getenv("S3_SECRET_KEY")),
		KeyPrefix: strings.Trim(strings.TrimSpace(os.Getenv("S3_KEY_PREFIX")), "/"),
	}
	ps := strings.EqualFold(os.Getenv("S3_USE_PATH_STYLE"), "true") || os.Getenv("S3_USE_PATH_STYLE") == "1"
	// Custom endpoints (MinIO, etc.) require path-style unless explicitly disabled.
	c.UsePathStyle = ps || strings.TrimSpace(c.Endpoint) != ""
	if c.Bucket != "" && (c.AccessKey == "" || c.SecretKey == "") {
		// Misconfiguration: caller should log and disable or fail fast at startup.
		return c
	}
	return c
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

// Enabled returns true when S3 uploads should be used.
func (c Config) Enabled() bool {
	return c.Bucket != "" && c.AccessKey != "" && c.SecretKey != ""
}

// Validate returns an error if Enabled but misconfigured.
func (c Config) Validate() error {
	if !c.Enabled() {
		return nil
	}
	if c.Region == "" {
		return fmt.Errorf("blobstore: S3_REGION is required when S3_BUCKET is set")
	}
	return nil
}
