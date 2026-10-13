package timecal

import "math"

// JulianDayNumber converts Gregorian calendar date to Julian Day Number (JDN).
func JulianDayNumber(year, month, day int) int {
	a := (14 - month) / 12
	y := year + 4800 - a
	m := month + 12*a - 3
	return day + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
}

// JDNToGregorian converts Julian Day Number back to Gregorian calendar.
func JDNToGregorian(jdn int) (year, month, day int) {
	a := jdn + 32044
	b := (4*a + 3) / 146097
	c := a - (146097*b)/4
	d := (4*c + 3) / 1461
	e := c - (1461*d)/4
	m := (5*e + 2) / 153
	day = e - (153*m+2)/5 + 1
	month = m + 3 - 12*(m/10)
	year = 100*b + d - 4800 + m/10
	return year, month, day
}

// HijriFromGregorian converts Gregorian date to Islamic Hijri calendar (approximate).
func HijriFromGregorian(year, month, day int) (hYear, hMonth, hDay int) {
	jdn := JulianDayNumber(year, month, day)
	// Islamic epoch: 1 Muharram 1 AH = Julian Day 1948439
	const islamicEpoch = 1948439
	daysSinceEpoch := jdn - islamicEpoch

	hYear = int(math.Floor(float64(daysSinceEpoch)/354.367)) + 1
	startOfYear := int(math.Round(float64(hYear-1)*354.367)) + islamicEpoch
	dayInYear := jdn - startOfYear
	hMonth = dayInYear/30 + 1
	hDay = dayInYear%30 + 1
	if hMonth > 12 {
		hMonth = 12
	}
	return hYear, hMonth, hDay
}

// GPSTimeToUTC converts GPS week + seconds to UTC accounting for leap seconds.
func GPSTimeToUTC(gpsWeek int, gpsSeconds float64, leapSeconds int) float64 {
	// GPS epoch: Jan 6, 1980 00:00:00 UTC
	const gpsEpochJDN = 2444244
	totalSeconds := float64(gpsWeek)*604800.0 + gpsSeconds
	// Subtract leap seconds (GPS does not track leap seconds, UTC does)
	return totalSeconds - float64(leapSeconds)
}

// LorentzSatelliteClockCorrection computes GPS satellite clock bias from relativistic effects.
// Delta_t = -2 * sqrt(mu) / c^2 * e * sqrt(a) * sin(E)
// where mu = GM_earth, a = semi-major axis, e = eccentricity, E = eccentric anomaly
func LorentzSatelliteClockCorrection(semiMajorAxisM, eccentricity, eccentricAnomalyRad float64) float64 {
	const G = 6.67430e-11
	const MEarth = 5.972e24
	const c = 299792458.0
	mu := G * MEarth
	return (-2.0 * math.Sqrt(mu) / (c * c)) * eccentricity * math.Sqrt(semiMajorAxisM) * math.Sin(eccentricAnomalyRad)
}
