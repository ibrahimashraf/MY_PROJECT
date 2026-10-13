package geodesy

import "math"

// GeodeticToENU converts a target geodetic coordinate into the local
// East-North-Up frame anchored at the reference coordinate. Expressing
// positions as ENU offsets from a nearby reference avoids the floating-point
// catastrophic cancellation that occurs when subtracting large ECEF
// magnitudes directly at kilometer scales.
func GeodeticToENU(target, ref GeodeticCoord) [3]float64 {
	tEcef := GeodeticToECEF(target)
	rEcef := GeodeticToECEF(ref)

	dx := tEcef.X - rEcef.X
	dy := tEcef.Y - rEcef.Y
	dz := tEcef.Z - rEcef.Z

	lat := degToRad(ref.LatDeg)
	lon := degToRad(ref.LonDeg)

	sinLat := math.Sin(lat)
	cosLat := math.Cos(lat)
	sinLon := math.Sin(lon)
	cosLon := math.Cos(lon)

	e := -sinLon*dx + cosLon*dy
	n := -sinLat*cosLon*dx - sinLat*sinLon*dy + cosLat*dz
	u := cosLat*cosLon*dx + cosLat*sinLon*dy + sinLat*dz

	return [3]float64{e, n, u}
}

// ENUToGeodetic converts a local East-North-Up offset back into absolute
// geodetic coordinates relative to the reference frame. The inverse rotation
// is the transpose of the orthonormal ENU basis, so the round trip is
// algebraically lossless up to floating-point precision.
func ENUToGeodetic(enu [3]float64, ref GeodeticCoord) GeodeticCoord {
	lat := degToRad(ref.LatDeg)
	lon := degToRad(ref.LonDeg)

	sinLat := math.Sin(lat)
	cosLat := math.Cos(lat)
	sinLon := math.Sin(lon)
	cosLon := math.Cos(lon)

	de := enu[0]
	dn := enu[1]
	du := enu[2]

	dx := -sinLon*de - sinLat*cosLon*dn + cosLat*cosLon*du
	dy := cosLon*de - sinLat*sinLon*dn + cosLat*sinLon*du
	dz := cosLat*dn + sinLat*du

	refEcef := GeodeticToECEF(ref)
	return ECEFToGeodetic(ECEFCoord{
		X: refEcef.X + dx,
		Y: refEcef.Y + dy,
		Z: refEcef.Z + dz,
	})
}
