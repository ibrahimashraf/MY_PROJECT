package rulesengine

import (
	"fmt"
	"math"
)

// GroundBearingPressure models the outrigger / mat bearing check:
//
//	sigma_actual = P_i / A_mat   (must be <= sigma_allowable)
//
// where P_i is the vertical load carried by one outrigger pad and A_mat is the
// effective bearing area of the outrigger mat (or directly of the pad when no
// spreader mat is used). A zero or negative area is a hard refusal: a rig
// without bearing area sinks into the ground.
func GroundBearingPressure(padLoadT float64, bearingAreaM2 float64) (actualKPa float64, passed bool, err error) {
	if math.IsNaN(padLoadT) || math.IsNaN(bearingAreaM2) ||
		math.IsInf(padLoadT, 0) || math.IsInf(bearingAreaM2, 0) {
		return 0, false, fmt.Errorf("geotech: NaN/Inf input rejected")
	}
	if padLoadT < 0 {
		return 0, false, fmt.Errorf("geotech: pad load must be >= 0, got %g", padLoadT)
	}
	if bearingAreaM2 <= 0 {
		return 0, false, fmt.Errorf("geotech: bearing area must be > 0 m^2, got %g (no ground support)", bearingAreaM2)
	}
	// 1 tonne-force ~= 9.80665 kN; pressure in kPa = kN / m^2.
	actual := (padLoadT * 9.80665) / bearingAreaM2
	return actual, actual > 0 && !math.IsInf(actual, 0), nil
}

// BearingPressureResult wraps the geotech check together with the allowable
// bound so callers get a single deterministic verdict.
type BearingPressureResult struct {
	ActualKPa      float64 `json:"actual_kpa"`
	AllowableKPa   float64 `json:"allowable_kpa"`
	Passed         bool    `json:"passed"`
	FactorOfSafety float64 `json:"factor_of_safety"`
}

// CheckBearingPressure performs the full ground bearing pressure gate:
// actual pressure must be below allowable with the configured factor of
// safety already folded into `allowable` by the caller.
func CheckBearingPressure(padLoadT float64, bearingAreaM2 float64, allowableKPa float64) (BearingPressureResult, error) {
	actual, ok, err := GroundBearingPressure(padLoadT, bearingAreaM2)
	if err != nil {
		return BearingPressureResult{}, err
	}
	if !ok || allowableKPa <= 0 {
		return BearingPressureResult{
			ActualKPa: actual, AllowableKPa: allowableKPa,
			Passed: false, FactorOfSafety: 0,
		}, nil
	}
	fos := allowableKPa / actual
	if math.IsNaN(fos) || math.IsInf(fos, 0) {
		return BearingPressureResult{ActualKPa: actual, AllowableKPa: allowableKPa, Passed: false, FactorOfSafety: 0}, nil
	}
	return BearingPressureResult{
		ActualKPa: actual, AllowableKPa: allowableKPa,
		Passed: actual <= allowableKPa, FactorOfSafety: fos,
	}, nil
}

// TrackPressureResult holds the BRE 470 crawler track pressure distribution.
type TrackPressureResult struct {
	MaxPressureKPa float64 `json:"max_pressure_kpa"`
	MinPressureKPa float64 `json:"min_pressure_kpa"`
	EffectiveLenM  float64 `json:"effective_len_m"`
	EccentricityM  float64 `json:"eccentricity_m"`
	IsTriangular   bool    `json:"is_triangular"`
	Passed         bool    `json:"passed"`
}

// BRE470TrackPressure calculates crawler crane track ground bearing pressure
// in accordance with BRE 470 (Working platforms for tracked plant).
// trackLoadT: total vertical load on the track (tonnes).
// trackLenM, trackWidthM: contact length and shoe width (meters).
// momentKNm: overturning moment along the track axis (kNm).
// allowableKPa: subgrade permissible bearing capacity (kPa).
func BRE470TrackPressure(trackLoadT, trackLenM, trackWidthM, momentKNm, allowableKPa float64) (TrackPressureResult, error) {
	if math.IsNaN(trackLoadT) || math.IsNaN(trackLenM) || math.IsNaN(trackWidthM) ||
		math.IsNaN(momentKNm) || math.IsNaN(allowableKPa) ||
		math.IsInf(trackLoadT, 0) || math.IsInf(trackLenM, 0) || math.IsInf(trackWidthM, 0) {
		return TrackPressureResult{}, fmt.Errorf("geotech: NaN/Inf input rejected")
	}
	if trackLoadT <= 0 {
		return TrackPressureResult{}, fmt.Errorf("geotech: track load must be > 0 t, got %g", trackLoadT)
	}
	if trackLenM <= 0 || trackWidthM <= 0 {
		return TrackPressureResult{}, fmt.Errorf("geotech: track dimensions must be > 0 m, got len=%g width=%g", trackLenM, trackWidthM)
	}
	if allowableKPa <= 0 {
		return TrackPressureResult{}, fmt.Errorf("geotech: allowable pressure must be > 0 kPa, got %g", allowableKPa)
	}

	pKN := trackLoadT * 9.80665
	e := math.Abs(momentKNm) / pKN

	// Tipping boundary: e >= L/2
	if e >= trackLenM/2.0 {
		return TrackPressureResult{
			EccentricityM: e,
			Passed:        false,
		}, fmt.Errorf("geotech: track load eccentricity %g m exceeds tipping limit %g m (machine overturning)", e, trackLenM/2.0)
	}

	middleThird := trackLenM / 6.0
	area := trackLenM * trackWidthM

	if e <= middleThird {
		// Middle third: trapezoidal distribution across entire track length
		qMax := (pKN / area) * (1.0 + (6.0*e)/trackLenM)
		qMin := (pKN / area) * (1.0 - (6.0*e)/trackLenM)
		return TrackPressureResult{
			MaxPressureKPa: qMax,
			MinPressureKPa: qMin,
			EffectiveLenM:  trackLenM,
			EccentricityM:  e,
			IsTriangular:   false,
			Passed:         qMax <= allowableKPa,
		}, nil
	}

	// Outside middle third: triangular distribution with partial lift-off (Meyerhof effective length)
	lEff := 3.0 * (trackLenM/2.0 - e)
	qMax := (2.0 * pKN) / (trackWidthM * lEff)
	return TrackPressureResult{
		MaxPressureKPa: qMax,
		MinPressureKPa: 0.0,
		EffectiveLenM:  lEff,
		EccentricityM:  e,
		IsTriangular:   true,
		Passed:         qMax <= allowableKPa,
	}, nil
}

// MatStructuralResult holds BS 5975 / CIRIA C703 spreader mat verification.
type MatStructuralResult struct {
	BendingMomentKNmM float64 `json:"bending_moment_knm_m"`
	ShearForceKNm     float64 `json:"shear_force_kn_m"`
	BendingStressMPa  float64 `json:"bending_stress_mpa"`
	Passed            bool    `json:"passed"`
}

// BS5975MatStructuralCheck checks the cantilever bending and shear stress in
// a timber or steel outrigger spreader mat per BS 5975 and CIRIA C703 §5.
// padLoadT: vertical load on outrigger (tonnes).
// matLenM, matWidthM: dimensions of spreader mat (meters).
// padLenM: contact dimension of crane outrigger foot along length (meters).
// matThicknessM: mat depth/thickness (meters).
// allowableBendingStressMPa: permissible material bending strength (e.g., 8-12 MPa for Ekki timber, 165-250 MPa for steel).
func BS5975MatStructuralCheck(padLoadT, matLenM, matWidthM, padLenM, matThicknessM, allowableBendingStressMPa float64) (MatStructuralResult, error) {
	if math.IsNaN(padLoadT) || math.IsNaN(matLenM) || math.IsNaN(matWidthM) ||
		math.IsNaN(padLenM) || math.IsNaN(matThicknessM) || math.IsNaN(allowableBendingStressMPa) {
		return MatStructuralResult{}, fmt.Errorf("geotech: NaN input rejected")
	}
	if padLoadT <= 0 || matLenM <= 0 || matWidthM <= 0 || padLenM <= 0 || matThicknessM <= 0 || allowableBendingStressMPa <= 0 {
		return MatStructuralResult{}, fmt.Errorf("geotech: all dimensions and allowable stresses must be > 0")
	}
	if padLenM >= matLenM {
		return MatStructuralResult{}, fmt.Errorf("geotech: pad size %g m must be smaller than mat size %g m", padLenM, matLenM)
	}

	// Uniform ground pressure under mat
	matArea := matLenM * matWidthM
	qKPa := (padLoadT * 9.80665) / matArea

	// Cantilever projection past pad edge
	cantileverM := (matLenM - padLenM) / 2.0

	// Maximum bending moment at pad face (kNm per metre width)
	mBending := (qKPa * cantileverM * cantileverM) / 2.0

	// Maximum one-way shear at pad face (kN per metre width)
	vShear := qKPa * cantileverM

	// Section modulus Z = (1.0 * t^2) / 6 per unit metre width (m^3/m)
	// Bending stress sigma = M / Z (kPa) = M / ((t^2)/6). In MPa: kPa / 1000.
	bendingStressMPa := (mBending / ((matThicknessM * matThicknessM) / 6.0)) / 1000.0

	return MatStructuralResult{
		BendingMomentKNmM: mBending,
		ShearForceKNm:     vShear,
		BendingStressMPa:  bendingStressMPa,
		Passed:            bendingStressMPa <= allowableBendingStressMPa,
	}, nil
}
