package planetary

import (
	"math"
)

const (
	// WGS84 Constants
	WGS84a = 6378137.0         // Semi-major axis (meters)
	WGS84b = 6356752.31424518  // Semi-minor axis (meters)
	WGS84eSq = 0.00669437999014 // e^2 = 1 - (b^2 / a^2)
)

// LatLonAltToECEF converts latitude, longitude (in degrees), and altitude (in meters) to ECEF XYZ (in meters).
func LatLonAltToECEF(lat, lon, alt float64) (x, y, z float64) {
	latRad := lat * math.Pi / 180.0
	lonRad := lon * math.Pi / 180.0

	sinLat, cosLat := math.Sincos(latRad)
	sinLon, cosLon := math.Sincos(lonRad)

	N := WGS84a / math.Sqrt(1.0-WGS84eSq*sinLat*sinLat)

	x = (N + alt) * cosLat * cosLon
	y = (N + alt) * cosLat * sinLon
	z = (N*(1.0-WGS84eSq) + alt) * sinLat
	return
}

// ECEFToLatLonAlt converts ECEF XYZ to latitude, longitude, and altitude for round-trip testing.
func ECEFToLatLonAlt(x, y, z float64) (lat, lon, alt float64) {
	p := math.Sqrt(x*x + y*y)
	if p == 0 {
		lon = 0
		if z >= 0 {
			lat = 90
		} else {
			lat = -90
		}
		alt = math.Abs(z) - WGS84b
		return
	}

	lon = math.Atan2(y, x) * 180.0 / math.Pi

	latRad := math.Atan2(z, p*(1.0-WGS84eSq))
	var N float64
	for i := 0; i < 5; i++ {
		sinLat := math.Sin(latRad)
		N = WGS84a / math.Sqrt(1.0-WGS84eSq*sinLat*sinLat)
		latRad = math.Atan2(z+WGS84eSq*N*sinLat, p)
	}

	lat = latRad * 180.0 / math.Pi
	alt = p/math.Cos(latRad) - N
	return
}

// ECEFToENU converts an ECEF point to East-North-Up given a reference ECEF and its Lat/Lon.
func ECEFToENU(x, y, z, refX, refY, refZ, refLat, refLon float64) (e, n, u float64) {
	dx := x - refX
	dy := y - refY
	dz := z - refZ

	latRad := refLat * math.Pi / 180.0
	lonRad := refLon * math.Pi / 180.0

	sinLat, cosLat := math.Sincos(latRad)
	sinLon, cosLon := math.Sincos(lonRad)

	e = -sinLon*dx + cosLon*dy
	n = -sinLat*cosLon*dx - sinLat*sinLon*dy + cosLat*dz
	u = cosLat*cosLon*dx + cosLat*sinLon*dy + sinLat*dz

	return
}

// ECEFDistance calculates the Euclidean distance between two ECEF points.
func ECEFDistance(x1, y1, z1, x2, y2, z2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
