package geodesy

import (
	"fmt"
	"math"
)

// WGS84 Ellipsoid Constants (GPS & Google Earth standard)
const (
	WGS84_A       = 6378137.0                                                 // Semi-major axis a (meters)
	WGS84_F       = 1.0 / 298.257223563                                       // Flattening f
	WGS84_B       = WGS84_A * (1.0 - WGS84_F)                                 // Semi-minor axis b ≈ 6356752.314245m
	WGS84_E2      = (WGS84_A*WGS84_A - WGS84_B*WGS84_B) / (WGS84_A * WGS84_A) // First eccentricity squared e^2
	WGS84_EPrime2 = (WGS84_A*WGS84_A - WGS84_B*WGS84_B) / (WGS84_B * WGS84_B) // Second eccentricity squared e'^2
)

// GeodeticCoord represents geographic coordinates (Latitude, Longitude in degrees, Altitude in meters above WGS84 ellipsoid).
type GeodeticCoord struct {
	LatDeg float64
	LonDeg float64
	AltM   float64
}

// ECEFCoord represents Earth-Centered Earth-Fixed cartesian coordinate (X, Y, Z in meters).
type ECEFCoord struct {
	X float64
	Y float64
	Z float64
}

func degToRad(deg float64) float64 { return deg * (math.Pi / 180.0) }
func radToDeg(rad float64) float64 { return rad * (180.0 / math.Pi) }

// GeodeticToECEF converts WGS84 (Lat, Lon, Alt) to 3D Cartesian ECEF (X, Y, Z).
func GeodeticToECEF(geo GeodeticCoord) ECEFCoord {
	latRad := degToRad(geo.LatDeg)
	lonRad := degToRad(geo.LonDeg)

	sinLat := math.Sin(latRad)
	cosLat := math.Cos(latRad)
	sinLon := math.Sin(lonRad)
	cosLon := math.Cos(lonRad)

	// Prime vertical radius of curvature N(phi)
	N := WGS84_A / math.Sqrt(1.0-WGS84_E2*sinLat*sinLat)

	x := (N + geo.AltM) * cosLat * cosLon
	y := (N + geo.AltM) * cosLat * sinLon
	z := (N*(1.0-WGS84_E2) + geo.AltM) * sinLat

	return ECEFCoord{X: x, Y: y, Z: z}
}

// ECEFToGeodetic converts 3D Cartesian ECEF to WGS84 (Lat, Lon, Alt) via Bowring's high-precision algorithm.
func ECEFToGeodetic(ecef ECEFCoord) GeodeticCoord {
	p := math.Hypot(ecef.X, ecef.Y)
	if p < 1e-6 {
		// Near geographic poles
		lat := 90.0
		if ecef.Z < 0 {
			lat = -90.0
		}
		alt := math.Abs(ecef.Z) - WGS84_B
		return GeodeticCoord{LatDeg: lat, LonDeg: 0, AltM: alt}
	}

	theta := math.Atan2(ecef.Z*WGS84_A, p*WGS84_B)
	sinTheta := math.Sin(theta)
	cosTheta := math.Cos(theta)

	num := ecef.Z + WGS84_EPrime2*WGS84_B*sinTheta*sinTheta*sinTheta
	denom := p - WGS84_E2*WGS84_A*cosTheta*cosTheta*cosTheta
	latRad := math.Atan2(num, denom)

	lonRad := math.Atan2(ecef.Y, ecef.X)

	sinLat := math.Sin(latRad)
	N := WGS84_A / math.Sqrt(1.0-WGS84_E2*sinLat*sinLat)
	alt := (p / math.Cos(latRad)) - N

	return GeodeticCoord{
		LatDeg: radToDeg(latRad),
		LonDeg: radToDeg(lonRad),
		AltM:   alt,
	}
}

// HaversineDistance computes great-circle distance between two geographic points on Earth sphere.
func HaversineDistance(p1, p2 GeodeticCoord) float64 {
	lat1 := degToRad(p1.LatDeg)
	lat2 := degToRad(p2.LatDeg)
	dLat := lat2 - lat1
	dLon := degToRad(p2.LonDeg - p1.LonDeg)

	a := math.Sin(dLat/2.0)*math.Sin(dLat/2.0) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2.0)*math.Sin(dLon/2.0)
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))

	return WGS84_A * c
}

// SlippyTile represents Google Maps / OSM standard slippy tile coordinates (X, Y, Zoom).
type SlippyTile struct {
	X    int
	Y    int
	Zoom int
}

// LatLonToSlippyTile converts (Lat, Lon) into Slippy map tile index at zoom level.
func LatLonToSlippyTile(latDeg, lonDeg float64, zoom int) SlippyTile {
	latRad := degToRad(latDeg)
	n := math.Pow(2.0, float64(zoom))

	x := int(math.Floor((lonDeg + 180.0) / 360.0 * n))
	y := int(math.Floor((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n))

	return SlippyTile{X: x, Y: y, Zoom: zoom}
}

// SlippyTileToBounds returns bounding geodetic coordinates [North, West, South, East] of tile.
func SlippyTileToBounds(tile SlippyTile) (north, west, south, east float64) {
	n := math.Pow(2.0, float64(tile.Zoom))

	west = float64(tile.X)/n*360.0 - 180.0
	east = float64(tile.X+1)/n*360.0 - 180.0

	northRad := math.Atan(math.Sinh(math.Pi * (1.0 - 2.0*float64(tile.Y)/n)))
	southRad := math.Atan(math.Sinh(math.Pi * (1.0 - 2.0*float64(tile.Y+1)/n)))

	north = radToDeg(northRad)
	south = radToDeg(southRad)
	return north, west, south, east
}

// SphericalPolygonArea computes enclosed area in square meters using spherical excess (Girard's theorem).
func SphericalPolygonArea(coords []GeodeticCoord) (float64, error) {
	n := len(coords)
	if n < 3 {
		return 0, fmt.Errorf("polygon requires at least 3 coordinates")
	}

	totalExcess := 0.0
	for i := 0; i < n; i++ {
		p1 := coords[i]
		p2 := coords[(i+1)%n]

		lam1 := degToRad(p1.LonDeg)
		lam2 := degToRad(p2.LonDeg)
		phi1 := degToRad(p1.LatDeg)
		phi2 := degToRad(p2.LatDeg)

		dLam := lam2 - lam1
		totalExcess += dLam * (2.0 + math.Sin(phi1) + math.Sin(phi2))
	}

	area := math.Abs(totalExcess * WGS84_A * WGS84_A / 2.0)
	return area, nil
}
