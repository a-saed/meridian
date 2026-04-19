package wms_test

import (
	"net/url"
	"testing"

	"meridian/internal/wms"
)

func TestParseGetMapRequest_Valid(t *testing.T) {
	q := url.Values{
		"SERVICE": {"WMS"},
		"VERSION": {"1.3.0"},
		"REQUEST": {"GetMap"},
		"LAYERS":  {"roads,buildings"},
		"STYLES":  {"", ""},
		"CRS":     {"EPSG:3857"},
		"BBOX":    {"-20037508,-20037508,20037508,20037508"},
		"WIDTH":   {"512"},
		"HEIGHT":  {"512"},
		"FORMAT":  {"image/png"},
	}

	req, err := wms.ParseGetMapRequest(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Layers) != 2 {
		t.Errorf("want 2 layers, got %d", len(req.Layers))
	}
	if req.Layers[0] != "roads" {
		t.Errorf("want first layer 'roads', got %s", req.Layers[0])
	}
	if req.Width != 512 || req.Height != 512 {
		t.Errorf("want 512x512, got %dx%d", req.Width, req.Height)
	}
}

func TestParseGetMapRequest_4326AxisSwap(t *testing.T) {
	// WMS 1.3.0 + EPSG:4326: BBOX is miny,minx,maxy,maxx (lat/lon)
	// Parser must normalize to minx,miny,maxx,maxy (lon/lat)
	q := url.Values{
		"SERVICE": {"WMS"},
		"VERSION": {"1.3.0"},
		"REQUEST": {"GetMap"},
		"LAYERS":  {"roads"},
		"STYLES":  {""},
		"CRS":     {"EPSG:4326"},
		"BBOX":    {"31.0,34.0,33.0,36.0"}, // miny,minx,maxy,maxx
		"WIDTH":   {"256"},
		"HEIGHT":  {"256"},
		"FORMAT":  {"image/png"},
	}

	req, err := wms.ParseGetMapRequest(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After normalization: minx=34, miny=31, maxx=36, maxy=33
	if req.Bbox[0] != 34.0 {
		t.Errorf("want Bbox[0]=34.0 (minLon after swap), got %f", req.Bbox[0])
	}
}

func TestParseGetMapRequest_MissingLayers(t *testing.T) {
	q := url.Values{
		"VERSION": {"1.3.0"},
		"CRS":     {"EPSG:4326"},
		"BBOX":    {"31,34,33,36"},
		"WIDTH":   {"256"},
		"HEIGHT":  {"256"},
	}
	_, err := wms.ParseGetMapRequest(q)
	if err == nil {
		t.Fatal("expected error for missing LAYERS")
	}
}

func TestParseGetMapRequest_InvalidWidth(t *testing.T) {
	q := url.Values{
		"LAYERS": {"roads"},
		"CRS":    {"EPSG:4326"},
		"BBOX":   {"31,34,33,36"},
		"WIDTH":  {"abc"},
		"HEIGHT": {"256"},
	}
	_, err := wms.ParseGetMapRequest(q)
	if err == nil {
		t.Fatal("expected error for non-numeric WIDTH")
	}
}

func TestParseGetFeatureInfoRequest_Valid(t *testing.T) {
	q := url.Values{
		"SERVICE":      {"WMS"},
		"VERSION":      {"1.3.0"},
		"REQUEST":      {"GetFeatureInfo"},
		"LAYERS":       {"roads"},
		"QUERY_LAYERS": {"roads"},
		"CRS":          {"EPSG:4326"},
		"BBOX":         {"31.5,35.5,32.5,36.5"}, // miny,minx,maxy,maxx
		"WIDTH":        {"1000"},
		"HEIGHT":       {"1000"},
		"I":            {"400"},
		"J":            {"600"},
		"INFO_FORMAT":  {"application/json"},
		"FEATURE_COUNT": {"3"},
	}

	req, err := wms.ParseGetFeatureInfoRequest(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.CRS != "EPSG:4326" {
		t.Fatalf("unexpected CRS: %s", req.CRS)
	}
	if req.Bbox[0] != 35.5 || req.Bbox[1] != 31.5 || req.Bbox[2] != 36.5 || req.Bbox[3] != 32.5 {
		t.Fatalf("expected normalized bbox [35.5 31.5 36.5 32.5], got %#v", req.Bbox)
	}
	if req.I != 400 || req.J != 600 {
		t.Fatalf("unexpected pixel coordinates: I=%d J=%d", req.I, req.J)
	}
	if req.FeatureCount != 3 {
		t.Fatalf("unexpected feature count: %d", req.FeatureCount)
	}
}

func TestParseGetFeatureInfoRequest_LowercaseKeys(t *testing.T) {
	q := url.Values{
		"version": {"1.3.0"},
		"layers":  {"roads"},
		"crs":     {"EPSG:4326"},
		"bbox":    {"31.5,35.5,32.5,36.5"},
		"width":   {"256"},
		"height":  {"256"},
		"i":       {"128"},
		"j":       {"128"},
	}
	req, err := wms.ParseGetFeatureInfoRequest(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.QueryLayers) != 1 || req.QueryLayers[0] != "roads" {
		t.Fatalf("expected QUERY_LAYERS default to LAYERS, got %#v", req.QueryLayers)
	}
}
