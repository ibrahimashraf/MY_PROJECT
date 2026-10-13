package physics

import (
	"math"
	"testing"
)

func TestEvaluateAtmosphere(t *testing.T) {
	// Negative altitude clamps to 0
	neg := EvaluateAtmosphere(-100.0)
	if math.Abs(neg.PressurePa-101325.0) > 1.0 {
		t.Fatalf("Negative altitude should clamp to sea level pressure: %v", neg.PressurePa)
	}

	// Sea level
	sea := EvaluateAtmosphere(0.0)
	if math.Abs(sea.PressurePa-101325.0) > 1.0 {
		t.Fatalf("Sea level pressure mismatch: %v", sea.PressurePa)
	}
	if math.Abs(sea.DensityKgM3-1.225) > 0.01 {
		t.Fatalf("Sea level density mismatch: %v", sea.DensityKgM3)
	}
	// Speed of sound at 15°C ≈ 340.3 m/s
	if math.Abs(sea.SpeedOfSoundMs-340.3) > 1.0 {
		t.Fatalf("Sea level speed of sound mismatch: %v", sea.SpeedOfSoundMs)
	}

	// Mt. Everest summit (~8,848m)
	everest := EvaluateAtmosphere(8848.0)
	if everest.PressurePa > 35000.0 || everest.PressurePa < 29000.0 {
		t.Fatalf("Everest pressure out of range: %v", everest.PressurePa)
	}
	if everest.DensityKgM3 > 0.6 || everest.DensityKgM3 < 0.4 {
		t.Fatalf("Everest air density out of range: %v", everest.DensityKgM3)
	}

	// Stratosphere (> 11,000m isothermal layer)
	strato := EvaluateAtmosphere(15000.0)
	if strato.TemperatureK != 216.65 {
		t.Fatalf("Expected isothermal stratosphere 216.65 K, got %v", strato.TemperatureK)
	}
	if strato.PressurePa >= everest.PressurePa {
		t.Fatalf("Stratosphere pressure must be lower than Everest: %v", strato.PressurePa)
	}
}

func TestAcousticDopplerAndAttenuation(t *testing.T) {
	c := 343.0   // speed of sound
	f0 := 1000.0 // 1 kHz siren

	// Source approaching at 34.3 m/s (~Mach 0.1)
	fObs := AcousticDopplerShift(f0, c, 0, 34.3)
	expectedF := 1000.0 * 343.0 / 308.7
	if math.Abs(fObs-expectedF) > 1e-1 {
		t.Fatalf("Doppler shift mismatch: expected %v, got %v", expectedF, fObs)
	}

	// Sonic boom singularity (v_source = c)
	fBoom := AcousticDopplerShift(f0, c, 0, 343.0)
	if !math.IsInf(fBoom, 1) {
		t.Fatalf("Expected +Inf for Mach 1 singularity, got %v", fBoom)
	}

	// 100 dB at 1m drops over 10m (20 dB geometric loss)
	spl10m := AcousticSphericalAttenuation(100.0, 10.0, 1.0, 0.005)
	if math.Abs(spl10m-79.955) > 1e-2 {
		t.Fatalf("Acoustic attenuation mismatch: expected 79.955, got %v", spl10m)
	}

	// r <= r0 should return initialSPLdB without attenuation
	splClose := AcousticSphericalAttenuation(100.0, 0.5, 1.0, 0.005)
	if splClose != 100.0 {
		t.Fatalf("Expected 100.0 dB for r <= r0, got %v", splClose)
	}
}

func TestOpticsRayleighAndSnell(t *testing.T) {
	blueWavelength := 450e-9
	redWavelength := 700e-9

	blueScattering := RayleighScatteringCoeff(blueWavelength)
	redScattering := RayleighScatteringCoeff(redWavelength)

	// Blue scattered ~5.8x more than red (lambda^-4)
	ratio := blueScattering / redScattering
	expectedRatio := math.Pow(700.0/450.0, 4)
	if math.Abs(ratio-expectedRatio) > 0.1 {
		t.Fatalf("Rayleigh scattering ratio mismatch: expected ~%v, got %v", expectedRatio, ratio)
	}

	// Air (n=1.0) into Crown Glass (n=1.52) at 30 degrees (pi/6 rad)
	theta1 := math.Pi / 6.0
	theta2, tir := SnellsRefraction(theta1, 1.0, 1.52)
	if tir {
		t.Fatal("Air into glass should not produce total internal reflection")
	}
	expectedTheta2 := math.Asin(1.0 / 1.52 * math.Sin(theta1))
	if math.Abs(theta2-expectedTheta2) > 1e-4 {
		t.Fatalf("Snell refraction angle mismatch: expected %v, got %v", expectedTheta2, theta2)
	}

	// Glass (n=1.52) into Air (n=1.0) at 60 degrees -> TIR
	_, tirHighAngle := SnellsRefraction(math.Pi/3.0, 1.52, 1.0)
	if !tirHighAngle {
		t.Fatal("Glass into air at 60 deg must trigger total internal reflection")
	}
}

func TestThermodynamicsStefanBoltzmannAndFourier(t *testing.T) {
	// Sun surface (~5778 K): M = 1.0 * sigma * (5778)^4 ≈ 6.32e7 W/m^2
	sunFlux := BlackbodyRadiantExitance(5778.0, 1.0)
	if sunFlux < 6.0e7 || sunFlux > 6.5e7 {
		t.Fatalf("Solar radiant exitance out of expected range: %v", sunFlux)
	}

	// Zero or negative temp and default emissivity
	if BlackbodyRadiantExitance(0, 1.0) != 0 || BlackbodyRadiantExitance(-100, 1.0) != 0 {
		t.Fatalf("Zero/negative temperature should return 0 radiant exitance")
	}
	defEmiss := BlackbodyRadiantExitance(300.0, 0)
	if defEmiss <= 0 {
		t.Fatalf("Default emissivity should calculate positive radiant exitance")
	}

	// Structural steel plate: k = 50 W/(m*K), T_hot = 373.15 K (100°C), T_cold = 293.15 K (20°C), thickness = 0.02m (20mm)
	// q = 50 * (373.15 - 293.15) / 0.02 = 50 * 80 / 0.02 = 200,000 W/m^2
	qFlux := HeatConductionFourier(50.0, 373.15, 293.15, 0.02)
	if math.Abs(qFlux-200000.0) > 1.0 {
		t.Fatalf("Fourier conduction heat flux mismatch: expected 200000, got %v", qFlux)
	}

	// Zero or negative thickness
	if HeatConductionFourier(50.0, 373.15, 293.15, 0) != 0 || HeatConductionFourier(50.0, 373.15, 293.15, -0.01) != 0 {
		t.Fatalf("Non-positive thickness should return 0 heat conduction")
	}
}

func TestEvaluateHydrostatic(t *testing.T) {
	surface := EvaluateHydrostatic(0, 15.0, 35.0)
	if math.Abs(surface.PressurePa-SeaLevelPressurePa) > 1e-3 {
		t.Fatalf("Surface seawater pressure mismatch: %v", surface.PressurePa)
	}
	if math.Abs(surface.SpeedOfSoundMs-1507.4) > 1.0 {
		t.Fatalf("Surface seawater sound speed mismatch: %v", surface.SpeedOfSoundMs)
	}

	// Negative depth and zero salinity fallbacks
	negDepth := EvaluateHydrostatic(-10.0, 15.0, 0)
	if math.Abs(negDepth.PressurePa-SeaLevelPressurePa) > 1e-3 {
		t.Fatalf("Negative depth should clamp to surface pressure: %v", negDepth.PressurePa)
	}

	deep := EvaluateHydrostatic(1000.0, 4.0, 35.0)
	expectedP := SeaLevelPressurePa + (1025.0 * StandardGravity * 1000.0)
	if math.Abs(deep.PressurePa-expectedP) > 1.0 {
		t.Fatalf("1000m hydrostatic pressure mismatch: expected %v, got %v", expectedP, deep.PressurePa)
	}
	if deep.SpeedOfSoundMs < 1470 || deep.SpeedOfSoundMs > 1510 {
		t.Fatalf("Deep sound speed out of expected range: %v", deep.SpeedOfSoundMs)
	}
}

func TestEvaluatePlanetaryCore(t *testing.T) {
	// Clamped bounds
	negRadius := EvaluatePlanetaryCore(-100.0)
	if negRadius.RadiusM != 0 {
		t.Fatalf("Negative radius should clamp to 0")
	}
	excessRadius := EvaluatePlanetaryCore(1e8)
	if excessRadius.RadiusM != EarthRadiusM {
		t.Fatalf("Excess radius should clamp to EarthRadiusM")
	}

	// 1. Center of Earth (r = 0): solid inner core
	center := EvaluatePlanetaryCore(0)
	if center.LayerName != "INNER_CORE_SOLID" {
		t.Fatalf("Expected INNER_CORE_SOLID at center, got %s", center.LayerName)
	}
	if math.Abs(center.PressurePa-EarthCenterPressurePa) > 1e9 {
		t.Fatalf("Center pressure mismatch: expected ~%v, got %v", EarthCenterPressurePa, center.PressurePa)
	}
	if center.GravityMs2 != 0 {
		t.Fatalf("Gravity at center must be 0, got %v", center.GravityMs2)
	}
	if center.SWaveVelocityMs <= 0 {
		t.Fatalf("Inner core must support shear waves, got %v", center.SWaveVelocityMs)
	}

	// 2. Liquid Outer Core (r = 2,500 km)
	outerCore := EvaluatePlanetaryCore(2500000.0)
	if outerCore.LayerName != "OUTER_CORE_LIQUID" {
		t.Fatalf("Expected OUTER_CORE_LIQUID, got %s", outerCore.LayerName)
	}
	if outerCore.SWaveVelocityMs != 0.0 {
		t.Fatalf("Liquid outer core must have Vs == 0, got %v", outerCore.SWaveVelocityMs)
	}

	// 3. Solid Mantle (r = 5,000 km, depth ~1371 km)
	mantle := EvaluatePlanetaryCore(5000000.0)
	if mantle.LayerName != "MANTLE_SOLID" {
		t.Fatalf("Expected MANTLE_SOLID, got %s", mantle.LayerName)
	}
	if mantle.SWaveVelocityMs <= 0 {
		t.Fatalf("Solid mantle must support shear waves, got %v", mantle.SWaveVelocityMs)
	}

	// 4. Lithospheric surface (r = EarthRadiusM)
	surface := EvaluatePlanetaryCore(EarthRadiusM)
	if surface.LayerName != "CRUST_LITHOSPHERE" {
		t.Fatalf("Expected CRUST_LITHOSPHERE at surface, got %s", surface.LayerName)
	}
	if surface.DepthM != 0 {
		t.Fatalf("Surface depth must be 0, got %v", surface.DepthM)
	}
}
