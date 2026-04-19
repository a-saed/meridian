# QGIS Integration

Connect QGIS to Meridian as a WMS source.

## Add the WMS connection

1. Open QGIS
2. Go to **Layer** → **Add Layer** → **Add WMS/WMTS Layer**
3. Click **New** to create a connection
4. Fill in:
   - **Name:** Meridian (or any label)
   - **URL:** `http://localhost:8080/wms`
5. Click **OK**
6. Click **Connect**

QGIS fetches the GetCapabilities document and lists all available layers.

## Add a layer to the map

1. Select your layer from the list
2. Click **Add**
3. Close the dialog

The layer appears on your QGIS canvas.

## Coordinate reference system

If QGIS asks you to select a CRS, choose **EPSG:4326** or **EPSG:3857** — whichever your layer supports. You configured this when creating the layer in Meridian.

## Refresh after changes

If you add or modify layers in Meridian while QGIS is connected, click **Connect** again to reload the capabilities list.
