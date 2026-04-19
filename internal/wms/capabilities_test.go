package wms_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"meridian/internal/store"
	"meridian/internal/wms"
)

func TestCapabilitiesXML(t *testing.T) {
	layers := []store.LayerRecord{
		{
			Name:  "roads",
			Title: "Road Network",
			SRS:   `["EPSG:4326","EPSG:3857"]`,
			MinX:  -180, MinY: -90, MaxX: 180, MaxY: 90,
		},
	}

	xmlBytes, err := wms.BuildCapabilities("http://localhost:8080/wms", layers)
	if err != nil {
		t.Fatalf("build capabilities: %v", err)
	}

	xmlStr := string(xmlBytes)

	if !strings.Contains(xmlStr, "roads") {
		t.Error("expected layer name 'roads' in capabilities")
	}
	if !strings.Contains(xmlStr, "Road Network") {
		t.Error("expected layer title in capabilities")
	}
	if !strings.Contains(xmlStr, "EPSG:4326") {
		t.Error("expected CRS in capabilities")
	}

	// Must be valid XML
	var v any
	if err := xml.Unmarshal(xmlBytes, &v); err != nil {
		t.Errorf("capabilities is not valid XML: %v", err)
	}
}
