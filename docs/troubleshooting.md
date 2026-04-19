# Troubleshooting

---

## Blank or white map image

**Symptom:** WMS returns a valid PNG but it's completely blank or white.

**Causes and fixes:**

1. **Wrong BBOX axis order for EPSG:4326** — the most common cause. For EPSG:4326, WMS 1.3.0 requires `minLat,minLon,maxLat,maxLon`. If you pass `minLon,minLat,maxLon,maxLat` the bbox will be interpreted incorrectly and may fall outside your data.
   → See [WMS Reference — BBOX axis order](wms-reference.md#bbox-axis-order--the-1-gotcha)

2. **BBOX doesn't intersect your data** — your layer's data may not cover the area you're requesting.
   → Check your layer's configured bounding box: `curl http://localhost:8080/api/v1/layers` and look at the `BBox` field. Use those coordinates in your WMS request.

3. **Layer has no data** — the source file may be empty or unreadable.
   → Verify: `curl http://localhost:8080/api/v1/layers` and check `GeomType` is not empty.

---

## "source not found" or layer not rendering

**Symptom:** API or WMS returns an error about a missing source.

**Cause:** The source was registered but the server hasn't loaded it (PostGIS sources), or the source was deleted.

**Fix:**
- If the source is a **PostGIS source**, restart Meridian to load it:
  ```bash
  docker compose restart meridian
  # or
  systemctl restart meridian
  ```
- If the source was **deleted**, re-register it via the API or UI. Check what sources exist: `curl http://localhost:8080/api/v1/sources`

---

## PostGIS connection refused

**Symptom:** Error like `dial tcp: connect: connection refused` when registering a PostGIS source.

**Fixes:**
- Verify PostGIS is running: `docker compose ps` or `pg_isready -h localhost -p 5432`
- Check your `conn_string` — use `postgis` as the hostname if running via Docker Compose, `localhost` if running PostGIS directly
- Confirm credentials: `psql postgres://gis:gis@localhost:5432/gis -c "SELECT 1"`

---

## GeoJSON file not found

**Symptom:** Source registration succeeds but the layer renders blank or the server logs show a file read error.

**Fixes:**
- Always use an **absolute path** in the source config, not a relative path
- If running in Docker, ensure the file is mounted into the container:
  ```yaml
  volumes:
    - ./my_data.geojson:/data/my_data.geojson
  ```
  Then use `/data/my_data.geojson` as the path. Verify the file is visible inside the container:
  ```bash
  docker compose exec meridian ls /data/
  ```

---

## Layer not appearing in GetCapabilities / QGIS

**Symptom:** You created a layer but it doesn't appear when QGIS connects.

**Fix:** Verify the layer exists:
```bash
curl http://localhost:8080/api/v1/layers
```
If it's there, click **Connect** again in QGIS to reload capabilities. See [WMS Reference](wms-reference.md) for details on the GetCapabilities endpoint.

---

## QGIS shows "no layers found"

**Symptom:** QGIS connects but lists no layers.

**Fixes:**
- Confirm the WMS URL is exactly `http://localhost:8080/wms` (no trailing slash, no path variations)
- Check that at least one layer exists: `curl http://localhost:8080/api/v1/layers`

---

## Slow first render, fast subsequent renders

**Expected behavior.** The first GetMap request renders fresh from the data source. Subsequent identical requests hit the LRU cache and return instantly. Cache is cleared on server restart.

---

## Upload fails with "only .geojson and .json files are accepted"

**Cause:** You're uploading a file with a different extension (e.g. `.txt`, `.zip`, `.kml`).

**Fix:** Convert your file to GeoJSON first.

Using `ogr2ogr` (part of GDAL):
```bash
ogr2ogr -f GeoJSON output.geojson input.kml   # KML → GeoJSON
ogr2ogr -f GeoJSON output.geojson input.shp   # Shapefile → GeoJSON
```

Online tools: [mapshaper.org](https://mapshaper.org), [geojson.io](https://geojson.io)

---

## Still stuck?

Check the server logs first:
```bash
docker compose logs meridian   # Docker
./meridian 2>&1 | tail -50     # binary
```

The logs show every WMS request, rendering errors, and source load failures.
