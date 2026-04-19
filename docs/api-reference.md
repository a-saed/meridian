# API Reference

All endpoints return and accept JSON. Base URL: `http://localhost:8080`

---

## Sources

### POST /api/v1/sources

Register a new data source.

**GeoJSON source:**
```bash
curl -s -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_cities",
    "type": "geojson",
    "config": {
      "path": "/absolute/path/to/cities.geojson"
    }
  }'
```

**PostGIS source:**
```bash
curl -s -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_roads",
    "type": "postgis",
    "config": {
      "conn_string": "postgres://gis:gis@localhost:5432/gis",
      "table": "public.roads",
      "geom_column": "geom"
    }
  }'
```

**Response (201 Created):**
```json
{
  "ID": "a1b2c3d4-e5f6-...",
  "Name": "my_cities",
  "Type": "geojson",
  "Config": { "path": "/absolute/path/to/cities.geojson" }
}
```

---

### GET /api/v1/sources

List all registered sources.

```bash
curl -s http://localhost:8080/api/v1/sources
```

**Response (200 OK):**
```json
[
  {
    "ID": "a1b2c3d4-...",
    "Name": "my_cities",
    "Type": "geojson",
    "Config": { "path": "/absolute/path/to/cities.geojson" }
  }
]
```

---

### DELETE /api/v1/sources/:id

Delete a source by ID.

```bash
curl -s -X DELETE http://localhost:8080/api/v1/sources/a1b2c3d4-...
```

**Response:** `204 No Content`

---

### POST /api/v1/sources/upload

Upload a GeoJSON file. Meridian saves the file and returns its absolute path — use that path when registering the source.

```bash
curl -s -X POST http://localhost:8080/api/v1/sources/upload \
  -F "file=@/local/path/to/data.geojson"
```

**Response (201 Created):**
```json
{ "path": "/absolute/server/path/to/data.geojson" }
```

Accepts `.geojson` and `.json` files up to 32 MB.

---

## Styles

### POST /api/v1/styles

```bash
curl -s -X POST http://localhost:8080/api/v1/styles \
  -H "Content-Type: application/json" \
  -d '{
    "name": "blue_fill",
    "fill_color": "#3388ff",
    "stroke_color": "#ffffff",
    "stroke_width": 1.5,
    "opacity": 0.9
  }'
```

All fields except `name` are optional. Defaults: fill `#3388ff`, stroke `#ffffff`, stroke_width `1.0`, opacity `1.0`.

**Response (201 Created):**
```json
{
  "ID": "e5f6g7h8-...",
  "Name": "blue_fill",
  "FillColor": "#3388ff",
  "StrokeColor": "#ffffff",
  "StrokeWidth": 1.5,
  "Opacity": 0.9
}
```

---

### GET /api/v1/styles

```bash
curl -s http://localhost:8080/api/v1/styles
```

**Response (200 OK):**
```json
[
  {
    "ID": "e5f6g7h8-...",
    "Name": "blue_fill",
    "FillColor": "#3388ff",
    "StrokeColor": "#ffffff",
    "StrokeWidth": 1.5,
    "Opacity": 0.9
  }
]
```

---

### DELETE /api/v1/styles/:id

```bash
curl -s -X DELETE http://localhost:8080/api/v1/styles/e5f6g7h8-...
```

**Response:** `204 No Content`

---

## Layers

### POST /api/v1/layers

```bash
curl -s -X POST http://localhost:8080/api/v1/layers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "cities",
    "title": "World Cities",
    "source_id": "a1b2c3d4-...",
    "style_id": "e5f6g7h8-...",
    "srs": ["EPSG:4326", "EPSG:3857"],
    "bbox": [-180, -90, 180, 90]
  }'
```

| Field | Required | Description |
|---|---|---|
| `name` | yes | WMS `LAYERS=` parameter value. Unique, no spaces. |
| `title` | no | Human-readable name in GetCapabilities |
| `source_id` | yes | ID from source creation |
| `style_id` | yes | ID from style creation |
| `source_layer` | no | PostGIS table name (ignored for GeoJSON) |
| `srs` | no | Supported CRS list. Defaults to `["EPSG:4326"]` |
| `bbox` | no | `[minLon, minLat, maxLon, maxLat]` in EPSG:4326 |

**Response (201 Created):**
```json
{
  "ID": "f1a2b3c4-...",
  "Name": "cities",
  "Title": "World Cities",
  "SourceID": "a1b2c3d4-...",
  "StyleID": "e5f6g7h8-...",
  "SourceLayer": "",
  "SRS": ["EPSG:4326", "EPSG:3857"],
  "BBox": [-180, -90, 180, 90],
  "GeomType": "Point"
}
```

---

### GET /api/v1/layers

```bash
curl -s http://localhost:8080/api/v1/layers
```

**Response (200 OK):**
```json
[
  {
    "ID": "f1a2b3c4-...",
    "Name": "cities",
    "Title": "World Cities",
    "SourceID": "a1b2c3d4-...",
    "StyleID": "e5f6g7h8-...",
    "SourceLayer": "",
    "SRS": ["EPSG:4326", "EPSG:3857"],
    "BBox": [-180, -90, 180, 90],
    "GeomType": "Point"
  }
]
```

---

### PUT /api/v1/layers/:id

Update an existing layer. Same body as POST.

```bash
curl -s -X PUT http://localhost:8080/api/v1/layers/layer-id-... \
  -H "Content-Type: application/json" \
  -d '{ "name": "cities", "title": "Updated Title", "source_id": "...", "style_id": "..." }'
```

**Response (200 OK):**
```json
{
  "ID": "f1a2b3c4-...",
  "Name": "cities",
  "Title": "Updated Title",
  "SourceID": "a1b2c3d4-...",
  "StyleID": "e5f6g7h8-...",
  "SourceLayer": "",
  "SRS": ["EPSG:4326", "EPSG:3857"],
  "BBox": [-180, -90, 180, 90],
  "GeomType": "Point"
}
```

---

### DELETE /api/v1/layers/:id

```bash
curl -s -X DELETE http://localhost:8080/api/v1/layers/layer-id-...
```

**Response:** `204 No Content`

---

## Health

### GET /health

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

---

## Error responses

All errors return plain text with an appropriate HTTP status code:

| Status | Meaning |
|---|---|
| 400 | Bad request — missing required field or invalid JSON |
| 404 | Resource not found |
| 415 | Unsupported media type (e.g. non-GeoJSON upload) |
| 500 | Internal server error |
