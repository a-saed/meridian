package file

// GeoJSONSourceConfig is persisted in store.SourceRecord.Config for type "geojson".
//
// Backward compatibility:
//   - Legacy PoC: only "path" is set (absolute local filesystem path).
//   - Production: "s3" references an object in S3-compatible storage; optional "meta" holds upload metadata.
//
// Exactly one of Path or S3 should be set. If both are set, S3 takes precedence.
type GeoJSONSourceConfig struct {
	Path string         `json:"path,omitempty"`
	S3   *S3ObjectRef   `json:"s3,omitempty"`
	Meta *GeoJSONMeta   `json:"meta,omitempty"`
}

// S3ObjectRef identifies a GeoJSON object in a bucket.
type S3ObjectRef struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// GeoJSONMeta holds optional upload-time metadata (not required for rendering).
type GeoJSONMeta struct {
	FeatureCount  int    `json:"feature_count,omitempty"`
	ChecksumSHA256 string `json:"checksum_sha256,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
	SizeBytes     int    `json:"size_bytes,omitempty"`
}
