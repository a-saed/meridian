# WMS Reference

Meridian implements OGC WMS 1.3.0. All requests go to `GET /wms`.

---

## GetCapabilities

Returns an XML document describing all available layers, supported CRS, and bounding boxes. WMS clients (QGIS, Leaflet, OpenLayers) call this automatically when you connect.

```
GET /wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetCapabilities
```

---

## GetMap

Returns a rendered map image.

```
GET /wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&LAYERS=...&...
```

### Parameters

| Parameter | Required | Example | Notes |
|---|---|---|---|
| `SERVICE` | yes | `WMS` | Always `WMS` |
| `VERSION` | yes | `1.3.0` | Only `1.3.0` supported |
| `REQUEST` | yes | `GetMap` | |
| `LAYERS` | yes | `roads` | Layer name. Comma-separated; Phase 1 renders the first layer only |
| `STYLES` | yes | `` | Leave empty to use the layer's configured style |
| `CRS` | yes | `EPSG:4326` | See supported CRS below |
| `BBOX` | yes | See below | Bounding box — axis order depends on CRS |
| `WIDTH` | yes | `512` | Image width in pixels |
| `HEIGHT` | yes | `512` | Image height in pixels |
| `FORMAT` | no | `image/png` | `image/png` (default) or `image/jpeg` |

---

## BBOX axis order — the #1 gotcha

WMS 1.3.0 uses **different axis orders** depending on the CRS. This catches almost everyone:

| CRS | BBOX order | Example |
|---|---|---|
| `EPSG:4326` | `minLat,minLon,maxLat,maxLon` | `22,24.7,31.9,36.9` (Egypt) |
| `EPSG:3857` | `minX,minY,maxX,maxY` (meters) | `2748160,2502567,4108226,3751743` (Egypt) |
| `CRS:84` | `minLon,minLat,maxLon,maxLat` | `24.7,22,36.9,31.9` |

Meridian handles this automatically — you just need to pass the correct order for your chosen CRS.

**Quick check:** for EPSG:4326, latitude (-90 to 90) comes before longitude (-180 to 180).

---

## Supported CRS

| CRS | Description |
|---|---|
| `EPSG:4326` | WGS84 geographic (latitude/longitude) |
| `EPSG:3857` | Web Mercator (meters) — used by Google Maps, OSM |
| `CRS:84` | Same as EPSG:4326 but with lon/lat axis order |

---

## Caching

Meridian caches rendered map images in an LRU cache keyed on the full request URL. Identical requests (same layers, bbox, size, CRS) are served from cache without re-rendering. Cache is in-memory and cleared on server restart.

---

## Example requests

**Egypt area, EPSG:4326:**
```bash
curl -o egypt.png \
  "http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap\
&LAYERS=test_layer&STYLES=&CRS=EPSG:4326\
&BBOX=22,24.7,31.9,36.9&WIDTH=1200&HEIGHT=900&FORMAT=image/png"
```

**World extent, EPSG:3857:**
```bash
curl -o world.png \
  "http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap\
&LAYERS=my_layer&STYLES=&CRS=EPSG:3857\
&BBOX=-20037508,-20037508,20037508,20037508&WIDTH=1024&HEIGHT=1024&FORMAT=image/png"
```
