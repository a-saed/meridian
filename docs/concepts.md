# Concepts

You don't need deep GIS knowledge to use Meridian — but a few terms will save you a lot of confusion.

## What is WMS?

WMS (Web Map Service) is an OGC standard protocol for requesting rendered map images over HTTP. A client (your browser, QGIS, Leaflet) sends a URL describing what it wants — which layers, what area, what size — and the server responds with a PNG or JPEG image.

```
GET /wms?REQUEST=GetMap&LAYERS=roads&BBOX=...&WIDTH=512&HEIGHT=512
→ PNG image of the roads layer for that area
```

Meridian is a WMS server. It speaks this protocol so any WMS-compatible client can connect without custom integration code. You bring the data (a GeoJSON file or a PostGIS table); Meridian handles the rest, serving rendered map images to any WMS-compatible client.

## Key terms

**Source** — where the geographic data lives. Meridian supports two source types:
- **GeoJSON** — a `.geojson` file on disk
- **PostGIS** — a table in a PostgreSQL/PostGIS database

**Layer** — a named, renderable view of a source. Layers are what WMS clients reference by name. One source can back multiple layers (e.g., same data with different styles or bounding boxes).

**Style** — visual properties for rendering a layer: fill color, stroke color, stroke width, opacity.

**CRS (Coordinate Reference System)** — defines how coordinates map to locations on Earth. The two you'll encounter most:
- **EPSG:4326** — latitude/longitude in degrees (GPS coordinates). This is what most GeoJSON files use.
- **EPSG:3857** — "Web Mercator" in meters. This is what Google Maps, OpenStreetMap, and most web tile services use.

Meridian supports both. You specify which one you want in your WMS request via the `CRS=` parameter.

**Bounding box (BBOX)** — a rectangle defining the geographic extent of a request or layer: `minX,minY,maxX,maxY`. For EPSG:4326 WMS 1.3.0 requests the axis order is swapped to `minLat,minLon,maxLat,maxLon` — see [WMS Reference](wms-reference.md) for details.

## Further reading

- [OGC WMS standard](https://www.ogc.org/standards/wms)
- [EPSG registry](https://epsg.io)
- [GeoJSON spec](https://geojson.org)
