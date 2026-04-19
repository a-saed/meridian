# Leaflet.js Integration

Add a Meridian WMS layer to a Leaflet map.

## Minimal example

```html
<!DOCTYPE html>
<html>
<head>
  <link rel="stylesheet" href="https://unpkg.com/leaflet/dist/leaflet.css" />
  <style>#map { height: 100vh; }</style>
</head>
<body>
  <div id="map"></div>
  <script src="https://unpkg.com/leaflet/dist/leaflet.js"></script>
  <script>
    // EPSG:4326 map (matches Meridian's default CRS)
    var map = L.map('map', {
      crs: L.CRS.EPSG4326,
      center: [30, 31],
      zoom: 6
    });

    // Meridian WMS layer
    L.tileLayer.wms('http://localhost:8080/wms', {
      layers: 'my_layer',       // your layer name
      format: 'image/png',
      transparent: true,
      version: '1.3.0',
      crs: L.CRS.EPSG4326
    }).addTo(map);
  </script>
</body>
</html>
```

## Using EPSG:3857 (Web Mercator)

If your layer supports `EPSG:3857`, you can use the default Leaflet CRS (no `crs` override needed):

```javascript
var map = L.map('map').setView([30, 31], 6);

L.tileLayer.wms('http://localhost:8080/wms', {
  layers: 'my_layer',
  format: 'image/png',
  transparent: true,
  version: '1.3.0'
}).addTo(map);
```

## Common issues

**Blank tiles:** Check that your layer's bounding box covers the area you're viewing, and that the CRS in your Leaflet config matches a CRS your layer supports.

**CORS errors:** Add `Access-Control-Allow-Origin: *` to your reverse proxy config, or run Meridian behind a proxy that sets CORS headers.
