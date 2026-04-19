package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/paulmach/orb/geojson"
)

type geoJSONInspect struct {
	FeatureCount  int
	ChecksumSHA256 string
}

func inspectGeoJSONFeatureCollection(data []byte) (geoJSONInspect, error) {
	var out geoJSONInspect
	sum := sha256.Sum256(data)
	out.ChecksumSHA256 = hex.EncodeToString(sum[:])
	fc, err := geojson.UnmarshalFeatureCollection(data)
	if err != nil {
		return out, fmt.Errorf("invalid GeoJSON FeatureCollection: %w", err)
	}
	out.FeatureCount = len(fc.Features)
	return out, nil
}
