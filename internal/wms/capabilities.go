package wms

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"meridian/internal/store"
)

// BuildCapabilities generates a WMS 1.3.0 GetCapabilities XML document.
func BuildCapabilities(serviceURL string, layers []store.LayerRecord) ([]byte, error) {
	type BoundingBox struct {
		CRS  string  `xml:"CRS,attr"`
		MinX float64 `xml:"minx,attr"`
		MinY float64 `xml:"miny,attr"`
		MaxX float64 `xml:"maxx,attr"`
		MaxY float64 `xml:"maxy,attr"`
	}
	type Layer struct {
		Name        string        `xml:"Name"`
		Title       string        `xml:"Title"`
		CRS         []string      `xml:"CRS"`
		BoundingBox []BoundingBox `xml:"BoundingBox"`
	}
	type OnlineResource struct {
		Href string `xml:"xlink:href,attr"`
	}
	type Get struct {
		OnlineResource OnlineResource `xml:"OnlineResource"`
	}
	type HTTP struct {
		Get Get `xml:"Get"`
	}
	type DCPType struct {
		HTTP HTTP `xml:"HTTP"`
	}
	type GetCapabilities struct {
		Format  string  `xml:"Format"`
		DCPType DCPType `xml:"DCPType"`
	}
	type GetMap struct {
		Format  []string `xml:"Format"`
		DCPType DCPType  `xml:"DCPType"`
	}
	type Request struct {
		GetCapabilities GetCapabilities `xml:"GetCapabilities"`
		GetMap          GetMap          `xml:"GetMap"`
	}
	type RootLayer struct {
		Title  string  `xml:"Title"`
		Layers []Layer `xml:"Layer"`
	}
	type Capability struct {
		Request Request   `xml:"Request"`
		Layer   RootLayer `xml:"Layer"`
	}
	type WMSCapabilities struct {
		XMLName    xml.Name   `xml:"WMS_Capabilities"`
		Version    string     `xml:"version,attr"`
		Capability Capability `xml:"Capability"`
	}

	cap := WMSCapabilities{Version: "1.3.0"}
	cap.Capability.Request.GetCapabilities.Format = "text/xml"
	cap.Capability.Request.GetCapabilities.DCPType.HTTP.Get.OnlineResource.Href = serviceURL
	cap.Capability.Request.GetMap.Format = []string{"image/png", "image/jpeg"}
	cap.Capability.Request.GetMap.DCPType.HTTP.Get.OnlineResource.Href = serviceURL
	cap.Capability.Layer.Title = "Meridian"

	for _, lr := range layers {
		var srsList []string
		if err := json.Unmarshal([]byte(lr.SRS), &srsList); err != nil {
			srsList = []string{"EPSG:4326"}
		}

		bboxes := make([]BoundingBox, 0, len(srsList))
		for _, crs := range srsList {
			bboxes = append(bboxes, BoundingBox{
				CRS: crs, MinX: lr.MinX, MinY: lr.MinY, MaxX: lr.MaxX, MaxY: lr.MaxY,
			})
		}

		cap.Capability.Layer.Layers = append(cap.Capability.Layer.Layers, Layer{
			Name:        lr.Name,
			Title:       lr.Title,
			CRS:         srsList,
			BoundingBox: bboxes,
		})
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(cap); err != nil {
		return nil, fmt.Errorf("capabilities: encode: %w", err)
	}
	return buf.Bytes(), nil
}
