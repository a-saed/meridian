# Meridian

A fast, modern WMS 1.3.0 server written in pure Go. Drop-in replacement for GeoServer — no JVM, no CGO, single 23 MB binary.

## What it does

Meridian serves geographic data as map images over the OGC Web Map Service (WMS) protocol. You register data sources (PostGIS tables, GeoJSON files), create layers that point to those sources, and clients (QGIS, Leaflet, OpenLayers, etc.) request rendered map images via standard WMS URLs.

```
Client                    Meridian                      Data
------                    --------                      ----
GET /wms?REQUEST=GetMap → parse & validate params
                        → cache lookup (LRU)
                        → query datasource ──────────→ PostGIS / GeoJSON
                        → reproject features
                        → render image (worker pool)
                        ← PNG / JPEG response
```

---

## Architecture

```
cmd/server/main.go          ← wires everything, graceful shutdown

internal/
  wms/                      ← OGC WMS HTTP handler
    handler.go              ← ServeHTTP, GetMap flow, cache logic
    request.go              ← parse & validate WMS query params
    capabilities.go         ← GetCapabilities XML builder

  api/                      ← REST admin API (chi router)
    handler.go              ← route registration
    sources.go              ← /api/v1/sources CRUD
    layers.go               ← /api/v1/layers  CRUD
    styles.go               ← /api/v1/styles  CRUD

  datasource/               ← spatial data backends
    source.go               ← DataSource interface + Feature/Query types
    registry.go             ← thread-safe name → DataSource map
    file/geojson.go         ← GeoJSON loader with R-tree spatial index
    postgis/source.go       ← pgx pool, ST_Intersects queries

  renderer/
    renderer.go             ← Renderer interface + Request type
    vector.go               ← gg-based vector renderer (points/lines/polygons)
    pool.go                 ← bounded semaphore (GOMAXPROCS×2 concurrent renders)

  store/
    store.go                ← Store interface (sources, layers, styles)
    sqlite.go               ← modernc.org/sqlite implementation

  cache/cache.go            ← LRU response cache (hashicorp/golang-lru)
  proj/proj.go              ← pure-Go EPSG:4326 ↔ EPSG:3857 transforms
  style/style.go            ← Style struct (colors, stroke, opacity)
```

**Key design rules:**
- `internal/` packages are never imported from outside the binary
- Dependencies only flow downward: `wms` → `datasource`, `renderer`, `proj`, `cache`. Never the reverse.
- Every cross-boundary type is an interface (`DataSource`, `Renderer`, `Store`) — swap implementations without touching callers
- `cmd/server/main.go` is the only place where concrete types are wired together

---

## Running locally

**Requirements:** Go 1.21+

```bash
git clone <repo>
cd meridian
go run ./cmd/server
```

Server starts on `:8080`. Config is stored in `meridian.db` (SQLite, created automatically).

On startup, Meridian also seeds/refreshes a built-in default layer named `test_layer` (GeoJSON points + polygon in Egypt). This gives you a known-good rendering target for quick validation and aligns the UI map to a visible extent.

**Environment variables:**

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | Listen address |
| `DB_PATH` | `meridian.db` | SQLite database path |

---

## Quick render smoke test

You can verify rendering immediately with the seeded demo layer:

```bash
curl -o egypt-demo.png \
  "http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap\
&LAYERS=test_layer&STYLES=&CRS=EPSG:4326\
&BBOX=22,24.7,31.9,36.9&WIDTH=1200&HEIGHT=900&FORMAT=image/png"
```

Open `egypt-demo.png` — you should see Cairo/Alexandria points and a polygon near Giza.

---

## Running with Docker Compose

Starts Meridian + a PostGIS 15 database:

```bash
docker compose up --build -d
```

PostGIS is available at `localhost:5432` (user: `gis`, password: `gis`, db: `gis`).  
Meridian is available at `http://localhost:8080`.

```bash
docker compose down       # stop
docker compose down -v    # stop and delete data volumes
```

---

## Quick-start: serve your first layer

### Option A — GeoJSON file

**1. Put your file somewhere Meridian can read it.**

If running locally:
```bash
cp my_data.geojson /tmp/my_data.geojson
```

If running in Docker, mount it:
```yaml
# docker-compose.yml — add to the meridian service:
volumes:
  - ./my_data.geojson:/data/my_data.geojson
  - meridian_data:/data/db
```
Then set `DB_PATH: /data/db/meridian.db`.

**2. Register the data source:**
```bash
curl -s -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_cities",
    "type": "geojson",
    "config": { "path": "/tmp/my_data.geojson" }
  }'
```

Response (save the `id`):
```json
{"ID":"a1b2c3d4-...","Name":"my_cities","Type":"geojson","Config":...}
```

**3. Create a style:**
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

Response (save the `id`):
```json
{"ID":"e5f6g7h8-...","Name":"blue_fill",...}
```

**4. Create a layer:**
```bash
curl -s -X POST http://localhost:8080/api/v1/layers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "cities",
    "title": "World Cities",
    "source_id": "a1b2c3d4-...",
    "style_id":  "e5f6g7h8-...",
    "srs": ["EPSG:4326", "EPSG:3857"],
    "bbox": [-180, -90, 180, 90]
  }'
```

**5. Request a map tile:**
```bash
curl -o map.png \
  "http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap\
&LAYERS=cities&STYLES=&CRS=EPSG:4326\
&BBOX=-180,-90,180,90&WIDTH=1024&HEIGHT=512&FORMAT=image/png"

open map.png   # macOS
xdg-open map.png  # Linux
```

---

### Option B — PostGIS table

**1. Register the data source** (use the internal Docker hostname `postgis` when running via Compose):
```bash
curl -s -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_postgis",
    "type": "postgis",
    "config": {
      "conn_string": "postgres://gis:gis@postgis:5432/gis",
      "table": "public.roads",
      "geom_column": "geom"
    }
  }'
```

For a local PostGIS (not Docker), use `localhost` instead of `postgis`.

**2. Create a style and layer** — same as steps 3–4 above.

> **Note:** PostGIS sources are opened at startup. If you register a new source via the API while the server is running, restart it (or `docker compose restart meridian`) to load the new source into the registry.

---

## Connecting from QGIS

1. Layer → Add Layer → Add WMS/WMTS Layer
2. New connection:
   - URL: `http://localhost:8080/wms`
3. Click **Connect** — your layers appear automatically from GetCapabilities
4. Select a layer, click **Add**

---

## Connecting from Leaflet

```javascript
L.tileLayer.wms("http://localhost:8080/wms", {
  layers: "cities",
  format: "image/png",
  transparent: true,
  version: "1.3.0",
  crs: L.CRS.EPSG4326
}).addTo(map);
```

---

## API reference

All endpoints return/accept JSON. No authentication in Phase 1.

### Data sources

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/sources` | Register a new data source |
| `GET` | `/api/v1/sources` | List all sources |
| `DELETE` | `/api/v1/sources/:id` | Delete a source |

**POST body — GeoJSON source:**
```json
{
  "name": "my_geojson",
  "type": "geojson",
  "config": {
    "path": "/absolute/path/to/file.geojson"
  }
}
```

**POST body — PostGIS source:**
```json
{
  "name": "my_postgis",
  "type": "postgis",
  "config": {
    "conn_string": "postgres://user:pass@host:5432/dbname",
    "table": "schema.table_name",
    "geom_column": "geom"
  }
}
```

---

### Layers

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/layers` | Create a layer |
| `GET` | `/api/v1/layers` | List all layers |
| `PUT` | `/api/v1/layers/:id` | Update a layer |
| `DELETE` | `/api/v1/layers/:id` | Delete a layer |

**POST body:**
```json
{
  "name": "roads",
  "title": "Road Network",
  "source_id": "<uuid from source creation>",
  "source_layer": "public.roads",
  "style_id": "<uuid from style creation>",
  "srs": ["EPSG:4326", "EPSG:3857"],
  "bbox": [-180, -90, 180, 90]
}
```

- `name` — used in WMS `LAYERS=` parameter. Must be unique.
- `title` — human-readable name shown in GetCapabilities.
- `source_layer` — the PostGIS table name. Ignored for GeoJSON sources.
- `bbox` — native bounding box `[minLon, minLat, maxLon, maxLat]` in EPSG:4326. Shown in GetCapabilities.

---

### Styles

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/styles` | Create a style |
| `GET` | `/api/v1/styles` | List all styles |
| `DELETE` | `/api/v1/styles/:id` | Delete a style |

**POST body:**
```json
{
  "name": "my_style",
  "fill_color": "#3388ff",
  "stroke_color": "#ffffff",
  "stroke_width": 1.0,
  "opacity": 1.0
}
```

All fields except `name` are optional and default to the values above.

---

### WMS endpoint

```
GET /wms
```

Supported operations:

| `REQUEST=` | Description |
|---|---|
| `GetCapabilities` | Returns XML describing all available layers |
| `GetMap` | Returns a rendered map image |

**GetMap parameters:**

| Parameter | Required | Example | Notes |
|---|---|---|---|
| `SERVICE` | yes | `WMS` | |
| `VERSION` | yes | `1.3.0` | |
| `REQUEST` | yes | `GetMap` | |
| `LAYERS` | yes | `roads` | Comma-separated; Phase 1 renders the first layer only |
| `STYLES` | yes | `` | Leave empty to use the layer's configured style |
| `CRS` | yes | `EPSG:4326` | Supported: `EPSG:4326`, `EPSG:3857`, `CRS:84` |
| `BBOX` | yes | `-180,-90,180,90` | Axis order: lon/lat for 3857, lat/lon for 4326 per WMS 1.3.0 spec |
| `WIDTH` | yes | `512` | Image width in pixels |
| `HEIGHT` | yes | `512` | Image height in pixels |
| `FORMAT` | no | `image/png` | `image/png` (default) or `image/jpeg` |

**BBOX axis order note:** WMS 1.3.0 swaps the axis order for `EPSG:4326` — the spec requires `miny,minx,maxy,maxx` (lat/lon). Meridian handles this automatically. For `EPSG:3857` use the standard `minx,miny,maxx,maxy` order.

---

### Health check

```
GET /health
→ {"status":"ok"}
```

---

## Project structure at a glance

```
meridian/
├── cmd/server/main.go          entry point — wires all packages together
├── api/                        REST admin API (not internal — intentional,
│                               allows future separation into a separate binary)
├── internal/
│   ├── wms/                    OGC protocol logic
│   ├── datasource/             spatial query interfaces and implementations
│   ├── renderer/               image generation
│   ├── store/                  config persistence
│   ├── cache/                  LRU response cache
│   ├── proj/                   coordinate reprojection
│   └── style/                  rendering style definition
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── go.sum
```

---

## Running tests

```bash
go test ./...                          # all unit tests
go test -tags integration ./...       # include PostGIS integration tests
                                       # (requires TEST_POSTGIS_CONN env var)
```

---

## Phase roadmap

| Phase | Scope |
|---|---|
| **1 (current)** | WMS GetMap + GetCapabilities, PostGIS + GeoJSON sources, REST config API, LRU cache, bounded worker pool, single binary |
| 2 | WFS GetFeature, WMTS tile caching, GeoTIFF raster support |
| 3 | Full SLD style parsing, remote upstream WMS proxy, web UI |
| 4 | Horizontal scaling (shared Redis cache), Helm chart |
