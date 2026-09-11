package structural

import (
	"fmt"
	"math"
)

// Material defines density and strength invariants.
type Material struct {
	Name            string
	DensityKgM3     float64 // kg/m^3 (e.g. Structural Steel ≈ 7850 kg/m^3, Concrete ≈ 2400 kg/m^3)
	YieldStrengthPa float64 // Pascals (e.g. S355 steel = 355 MPa)
	SafetyFactor    float64 // e.g. 1.5 - 2.0
}

// BoundingBox defines 3D extent.
type BoundingBox struct {
	LengthM float64 // X axis
	WidthM  float64 // Y axis
	HeightM float64 // Z axis
}

// Volume computes solid volume.
func (b BoundingBox) Volume() float64 {
	return b.LengthM * b.WidthM * b.HeightM
}

// StructuralBody represents a discrete load element with mass properties.
type StructuralBody struct {
	ID        string
	Material  Material
	Size      BoundingBox
	Position  [3]float64 // Local origin
	LocalCoG  [3]float64 // Offset relative to position
}

// TotalMassKg computes mass = volume * density.
func (b StructuralBody) TotalMassKg() float64 {
	return b.Size.Volume() * b.Material.DensityKgM3
}

// GlobalCoG returns world-space center of gravity.
func (b StructuralBody) GlobalCoG() [3]float64 {
	return [3]float64{
		b.Position[0] + b.LocalCoG[0],
		b.Position[1] + b.LocalCoG[1],
		b.Position[2] + b.LocalCoG[2],
	}
}

// MultiBodyAssembly calculates aggregate weight and system Center of Gravity.
type MultiBodyAssembly struct {
	Bodies []StructuralBody
}

// AggregateProperties computes total mass, total weight (Newtons), and system CoG.
func (a MultiBodyAssembly) AggregateProperties() (massKg float64, weightN float64, cog [3]float64, err error) {
	if len(a.Bodies) == 0 {
		return 0, 0, [3]float64{}, fmt.Errorf("assembly contains no bodies")
	}

	const g = 9.80665 // m/s^2
	var totalMass float64
	var sumMoment [3]float64

	for _, b := range a.Bodies {
		m := b.TotalMassKg()
		if m <= 0 {
			return 0, 0, [3]float64{}, fmt.Errorf("invalid body mass: %v", m)
		}
		totalMass += m
		gCoG := b.GlobalCoG()
		sumMoment[0] += m * gCoG[0]
		sumMoment[1] += m * gCoG[1]
		sumMoment[2] += m * gCoG[2]
	}

	cog = [3]float64{
		sumMoment[0] / totalMass,
		sumMoment[1] / totalMass,
		sumMoment[2] / totalMass,
	}
	weightN = totalMass * g
	return totalMass, weightN, cog, nil
}

// OutriggerPad represents a load transfer foundation pad.
type OutriggerPad struct {
	ID          string
	AreaM2      float64 // e.g., 2.0m x 2.0m steel mat = 4.0 m^2
	AllowablePa float64 // Permissible bearing pressure (e.g. compacted soil = 200 kPa = 200,000 Pa)
}

// GroundBearingPressure evaluates contact pressure: P = ReactionForce / PadArea.
func GroundBearingPressure(reactionForceN float64, pad OutriggerPad) (pressurePa float64, safe bool) {
	if pad.AreaM2 <= 0.01 {
		return math.Inf(1), false
	}
	pressurePa = reactionForceN / pad.AreaM2
	safe = pressurePa <= pad.AllowablePa
	return pressurePa, safe
}

// WindLoadAssessment evaluates aerodynamic drag load on exposed surface area.
// Dynamic pressure q = 0.5 * rho * v^2
// Drag force F_wind = q * Area * Cd
func WindLoadAssessment(windSpeedMps float64, airDensityKgM3 float64, exposedAreaM2 float64, dragCoeff float64) (forceN float64, dynamicPressurePa float64) {
	if airDensityKgM3 <= 0 {
		airDensityKgM3 = 1.225 // Standard sea level air density
	}
	if dragCoeff <= 0 {
		dragCoeff = 1.2 // Bluff body / crane boom default
	}

	dynamicPressurePa = 0.5 * airDensityKgM3 * (windSpeedMps * windSpeedMps)
	forceN = dynamicPressurePa * exposedAreaM2 * dragCoeff
	return forceN, dynamicPressurePa
}

// CombinedOverturningMoment calculates total overturning moment at pivot from gravity + wind.
func CombinedOverturningMoment(weightN float64, cogEccentricityM float64, windForceN float64, windCenterHeightM float64) float64 {
	momentGravity := weightN * cogEccentricityM
	momentWind := windForceN * windCenterHeightM
	return momentGravity + momentWind
}

// BeamStressAnalysis computes max bending stress: sigma = M * y / I
// For rectangular cross-section: I = b * h^3 / 12, y = h / 2 -> sigma = 6 * M / (b * h^2)
func BeamStressAnalysis(bendingMomentNm float64, widthM float64, heightM float64, material Material) (stressPa float64, utilization float64, safe bool) {
	if widthM <= 0.001 || heightM <= 0.001 {
		return math.Inf(1), math.Inf(1), false
	}

	sectionModulus := (widthM * heightM * heightM) / 6.0
	stressPa = bendingMomentNm / sectionModulus

	allowableStress := material.YieldStrengthPa / material.SafetyFactor
	utilization = stressPa / allowableStress
	safe = utilization <= 1.0

	return stressPa, utilization, safe
}
