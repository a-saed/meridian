package file

import (
	"context"
	"encoding/json"
	"fmt"
)

// S3ObjectGetter fetches raw bytes for a GeoJSON object in S3-compatible storage.
type S3ObjectGetter interface {
	GetS3Object(ctx context.Context, bucket, key string) ([]byte, error)
}

// OpenGeoJSONSource opens a geojson datasource from store.SourceRecord.Config JSON.
// If config uses S3, getter must be non-nil.
func OpenGeoJSONSource(ctx context.Context, configJSON []byte, s3 S3ObjectGetter) (*GeoJSONSource, error) {
	var cfg GeoJSONSourceConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("geojson: config json: %w", err)
	}
	if cfg.S3 != nil {
		if s3 == nil {
			return nil, fmt.Errorf("geojson: s3 object configured but object storage is not enabled on server")
		}
		if cfg.S3.Bucket == "" || cfg.S3.Key == "" {
			return nil, fmt.Errorf("geojson: s3.bucket and s3.key are required")
		}
		data, err := s3.GetS3Object(ctx, cfg.S3.Bucket, cfg.S3.Key)
		if err != nil {
			return nil, err
		}
		return NewGeoJSONSourceFromBytes(data)
	}
	if cfg.Path != "" {
		return NewGeoJSONSource(cfg.Path)
	}
	return nil, fmt.Errorf("geojson: config must set path or s3")
}
