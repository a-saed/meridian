package proj

import "math"

const earthRadius = 6378137.0

// To3857 converts WGS84 lon/lat (EPSG:4326) to Web Mercator x/y (EPSG:3857).
func To3857(lon, lat float64) (x, y float64) {
	x = lon * math.Pi / 180 * earthRadius
	latRad := lat * math.Pi / 180
	y = math.Log(math.Tan(math.Pi/4+latRad/2)) * earthRadius
	return
}

// To4326 converts Web Mercator x/y (EPSG:3857) to WGS84 lon/lat (EPSG:4326).
func To4326(x, y float64) (lon, lat float64) {
	lon = x / earthRadius * 180 / math.Pi
	lat = (2*math.Atan(math.Exp(y/earthRadius)) - math.Pi/2) * 180 / math.Pi
	return
}

// BoundTo3857 converts a [minLon,minLat,maxLon,maxLat] EPSG:4326 bound to EPSG:3857.
func BoundTo3857(b [4]float64) [4]float64 {
	minX, minY := To3857(b[0], b[1])
	maxX, maxY := To3857(b[2], b[3])
	return [4]float64{minX, minY, maxX, maxY}
}

// BoundTo4326 converts a [minX,minY,maxX,maxY] EPSG:3857 bound to EPSG:4326.
func BoundTo4326(b [4]float64) [4]float64 {
	minLon, minLat := To4326(b[0], b[1])
	maxLon, maxLat := To4326(b[2], b[3])
	return [4]float64{minLon, minLat, maxLon, maxLat}
}

// IsSupportedCRS returns true for CRS codes this package can transform.
func IsSupportedCRS(crs string) bool {
	switch crs {
	case "EPSG:4326", "CRS:84", "EPSG:3857", "EPSG:900913":
		return true
	}
	return false
}

// NormalizeBound converts any supported CRS bound to EPSG:4326 [minLon,minLat,maxLon,maxLat].
func NormalizeBound(bbox [4]float64, crs string) [4]float64 {
	switch crs {
	case "EPSG:3857", "EPSG:900913":
		return BoundTo4326(bbox)
	default: // EPSG:4326, CRS:84
		return bbox
	}
}
