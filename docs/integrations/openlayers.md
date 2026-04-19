# OpenLayers Integration

Add a Meridian WMS layer to an OpenLayers map.

## Minimal example

```html
<!DOCTYPE html>
<html>
<head>
  <script src="https://cdn.jsdelivr.net/npm/ol/dist/ol.js"></script>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/ol/ol.css" />
  <style>#map { width: 100%; height: 100vh; }</style>
</head>
<body>
  <div id="map"></div>
  <script>
    var map = new ol.Map({
      target: 'map',
      view: new ol.View({
        projection: 'EPSG:4326',
        center: [31, 30],
        zoom: 6
      }),
      layers: [
        new ol.layer.Image({
          source: new ol.source.ImageWMS({
            url: 'http://localhost:8080/wms',
            params: {
              LAYERS: 'my_layer',   // your layer name
              VERSION: '1.3.0',
              FORMAT: 'image/png'
            },
            projection: 'EPSG:4326'
          })
        })
      ]
    });
  </script>
</body>
</html>
```

## Using EPSG:3857

```javascript
var map = new ol.Map({
  target: 'map',
  view: new ol.View({
    center: ol.proj.fromLonLat([31, 30]),
    zoom: 6
  }),
  layers: [
    new ol.layer.Tile({
      source: new ol.source.TileWMS({
        url: 'http://localhost:8080/wms',
        params: {
          LAYERS: 'my_layer',   // your layer name
          VERSION: '1.3.0',
          FORMAT: 'image/png'
        }
      })
    })
  ]
});
```

`TileWMS` (tiled) is more efficient for large maps; `ImageWMS` (single image) is simpler and better for small extents or overlays.
