# Quickstart

Get Meridian running and serving your first map in under 10 minutes.

**Prerequisites:** Go 1.21+ OR Docker

---

## Step 1: Run the server

First, clone the repository and change into its directory:

```bash
git clone https://github.com/your-org/meridian.git
cd meridian
```

> Replace `your-org/meridian` with the actual repository path if different.

**With Go:**
```bash
go run ./cmd/server
```

**With Docker:**
```bash
docker compose up --build -d
```

The server starts on `http://localhost:8080`. You'll see:
```
Meridian listening on :8080
```

---

## Step 2: Verify with the built-in demo layer

Meridian seeds a demo layer called `test_layer` on startup — points and a polygon in Egypt. Request a map tile to confirm rendering works:

```bash
curl -o demo.png \
  "http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap\
&LAYERS=test_layer&STYLES=&CRS=EPSG:4326\
&BBOX=22,24.7,31.9,36.9&WIDTH=1200&HEIGHT=900&FORMAT=image/png"
```

Open `demo.png` — you should see Cairo/Alexandria points and a polygon near Giza.

---

## Step 3: Register your own GeoJSON file

**Option A — via the UI** (easiest):
1. Open `http://localhost:8080` in your browser
2. Go to **Sources** → **Upload GeoJSON**
3. Select your `.geojson` file — Meridian saves it and returns the file path

**Option B — via the API:**
```bash
# Upload the file
curl -s -X POST http://localhost:8080/api/v1/sources/upload \
  -F "file=@/path/to/your/data.geojson"
# → {"path":"/absolute/path/to/data.geojson"}

# Register the source (use the path from the upload response)
curl -s -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_data",
    "type": "geojson",
    "config": { "path": "/absolute/path/to/data.geojson" }
  }'
# → {"ID":"...","Name":"my_data",...}  ← save the ID
```

---

## Step 4: Create a style

```bash
curl -s -X POST http://localhost:8080/api/v1/styles \
  -H "Content-Type: application/json" \
  -d '{
    "name": "blue",
    "fill_color": "#3388ff",
    "stroke_color": "#ffffff",
    "stroke_width": 1.5,
    "opacity": 0.9
  }'
# → {"ID":"...","Name":"blue",...}  ← save the ID
```

---

## Step 5: Create a layer

```bash
curl -s -X POST http://localhost:8080/api/v1/layers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_layer",
    "title": "My Layer",
    "source_id": "<source ID from step 3>",
    "style_id": "<style ID from step 4>",
    "srs": ["EPSG:4326", "EPSG:3857"],
    "bbox": [-180, -90, 180, 90]
  }'
```

---

## Step 6: Request a map tile

```bash
curl -o map.png \
  "http://localhost:8080/wms?SERVICE=WMS&VERSION=1.3.0&REQUEST=GetMap\
&LAYERS=my_layer&STYLES=&CRS=EPSG:4326\
&BBOX=-90,-180,90,180&WIDTH=1024&HEIGHT=512&FORMAT=image/png"

xdg-open map.png   # Linux
open map.png       # macOS
```

---

## Next steps

- [UI Guide](ui-guide.md) — manage everything through the browser
- [Leaflet Integration](integrations/leaflet.md) — embed your layer in a web map
- [QGIS Integration](integrations/qgis.md) — connect from QGIS desktop
- [Installation](installation.md) — deploy to a server
