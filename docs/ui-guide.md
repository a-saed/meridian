# UI Guide

Meridian includes a built-in web UI for managing sources, styles, and layers without using the API directly.

Open `http://localhost:8080` in your browser.

## Navigation

The sidebar has three sections:

- **Sources** — register data backends (GeoJSON files, PostGIS tables)
- **Styles** — define visual appearance (colors, stroke, opacity)
- **Layers** — combine a source and a style into a named WMS layer

Work in order: register a source → create a style → create a layer.

---

## Register a data source

A source tells Meridian where your geographic data lives.

**GeoJSON file:**
1. Go to **Sources** in the sidebar
2. Click **Upload GeoJSON**
3. Enter a **Name** for the source (e.g. `my_cities`)
4. Select your `.geojson` or `.json` file (max 32 MB)
5. Click **Save** — Meridian uploads the file and registers the source

**PostGIS table:**
1. Go to **Sources** → **Add Source**
2. Set the **Type** dropdown to **postgis**
3. Fill in:
   - **Name** — a unique identifier (e.g. `roads`)
   - **Connection string** — `postgres://user:pass@host:5432/dbname`
   - **Table** — `schema.table_name` (e.g. `public.roads`)
   - **Geometry column** — column containing the geometry (e.g. `geom`)
4. Click **Save**

> **Note:** After adding a PostGIS source, restart Meridian to load it into memory.

---

## Create a style

Styles define how a layer looks when rendered.

1. Go to **Styles** → **New Style**
2. Set:
   - **Name** — unique identifier
   - **Fill color** — interior color for polygons / fill for points (hex, e.g. `#3388ff`)
   - **Stroke color** — outline color (hex, e.g. `#ffffff`)
   - **Stroke width** — outline thickness in pixels (e.g. `1.5`)
   - **Opacity** — 0.0 (transparent) to 1.0 (fully opaque)
3. Click **Save**

---

## Create a layer

Layers combine a source and a style into something WMS clients can request by name.

1. Go to **Layers** → **New Layer**
2. Fill in:
   - **Name** — used in WMS `LAYERS=` parameter. Must be unique. No spaces.
   - **Title** — human-readable name shown in GetCapabilities
   - **Source** — select from your registered sources
   - **Style** — select from your styles
   - **SRS** — supported coordinate systems (select `EPSG:4326` and/or `EPSG:3857`)
   - **Bounding box** — geographic extent `[minLon, minLat, maxLon, maxLat]` in EPSG:4326
3. Click **Save**

---

## Preview a layer

After creating a layer, request a map tile by opening this URL in your browser (replace `<layer-name>` and the BBOX with your layer's values):

```
http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&LAYERS=<layer-name>&STYLES=&CRS=EPSG:4326&BBOX=<minLat,minLon,maxLat,maxLon>&WIDTH=800&HEIGHT=600&FORMAT=image/png
```

**Example** — the built-in Egypt demo layer:

```
http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap&LAYERS=test_layer&STYLES=&CRS=EPSG:4326&BBOX=22,24.7,31.9,36.9&WIDTH=800&HEIGHT=600&FORMAT=image/png
```

> Remember: for `EPSG:4326` the BBOX order is `minLat,minLon,maxLat,maxLon` — latitude before longitude. See [WMS Reference](wms-reference.md) for details.

---

## Edit or delete

- **Layers:** Go to **Layers**, click a layer row to open the edit form, or click **Delete** to remove it.
- **Styles:** Go to **Styles**, click **Delete** to remove a style. Styles cannot be edited — delete and recreate if you need to change one.
- **Sources:** Go to **Sources**, click **Delete** to remove a source. Sources cannot be edited — delete and re-register if you need to change the configuration.

> Deleting a source or style that is referenced by a layer will break that layer. Remove the layer first.
