package physics

import (
	"math"
	"testing"
)

func TestEvaluateAtmosphere(t *testing.T) {
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
	// Pressure should drop to ~31.4 kPa
	if everest.PressurePa > 35000.0 || everest.PressurePa < 29000.0 {
		t.Fatalf("Everest pressure out of range: %v", everest.PressurePa)
	}
	// Density drops to ~0.47 kg/m^3 (thin air)
	if everest.DensityKgM3 > 0.6 || everest.DensityKgM3 < 0.4 {
		t.Fatalf("Everest air density out of range: %v", everest.DensityKgM3)
	}
}

func TestAcousticDopplerAndAttenuation(t *testing.T) {
	c := 343.0 // speed of sound
	f0 := 1000.0 // 1 kHz siren

	// Source approaching at 34.3 m/s (~Mach 0.1)
	fObs := AcousticDopplerShift(f0, c, 0, 34.3)
	// f' = 1000 * 343 / (343 - 34.3) = 1000 * 343 / 308.7 = 1111.11 Hz
	expectedF := 1000.0 * 343.0 / 308.7
	if math.Abs(fObs-expectedF) > 1e-1 {
		t.Fatalf("Doppler shift mismatch: expected %v, got %v", expectedF, fObs)
	}

	// 100 dB at 1m drops over 10m (20 dB geometric loss)
	spl10m := AcousticSphericalAttenuation(100.0, 10.0, 1.0, 0.005)
	// 100 - 20*log10(10) - 0.005*9 = 100 - 20 - 0.045 = 79.955 dB
	if math.Abs(spl10m-79.955) > 1e-2 {
		t.Fatalf("Acoustic attenuation mismatch: expected 79.955, got %v", spl10m)
	}
}

func TestOpticsRayleighAndSnell(t *testing.T) {
	// Blue light (450 nm) vs Red light (700 nm)
	blueWavelength := 450e-9
	redWavelength := 700e-9

	blueScattering := RayleighScatteringCoeff(blueWavelength)
	redScattering := RayleighScatteringCoeff(redWavelength)

	// Rayleigh ratio: (700 / 450)^4 ≈ 5.86x more blue scattering -> blue skies!
	ratio := blueScattering / redScattering
	expectedRatio := math.Pow(700.0/450.0, 4)
	if math.Abs(ratio-expectedRatio) > 0.05 {
		t.Fatalf("Rayleigh blue-to-red ratio mismatch: expected ~%v, got %v", expectedRatio, ratio)
	}

	// Snell's law: Air (n1=1.0) into Water (n2=1.333) at 45°
	theta2, tir := SnellsRefraction(math.Pi/4.0, 1.0, 1.333)
	if tir {
		t.Fatal("Air into water should not produce total internal reflection")
	}
	// sin(theta2) = sin(45) / 1.333 = 0.7071 / 1.333 = 0.5304 -> theta2 ≈ 0.559 rad (32.03°)
	expectedAngle := math.Asin(math.Sin(math.Pi/4.0) / 1.333)
	if math.Abs(theta2-expectedAngle) > 1e-4 {
		t.Fatalf("Snell refraction angle mismatch: expected %v, got %v", expectedAngle, theta2)
	}

	// Water into Air at 60° (past critical angle ~48.6°) -> Total Internal Reflection
	_, tirWaterAir := SnellsRefraction(math.Pi/3.0, 1.333, 1.0)
	if !tirWaterAir {
		t.Fatal("Water to air at 60 deg must trigger total internal reflection")
	}
}

func TestThermodynamicsStefanBoltzmannAndFourier(t *testing.T) {
	// Sun surface temperature T ≈ 5778 K
	sunExitance := BlackbodyRadiantExitance(5778.0, 1.0)
	// sigma * 5778^4 ≈ 5.67037e-8 * 1.1147e15 ≈ 6.32e7 W/m^2
	if sunExitance < 6e7 || sunExitance > 6.5e7 {
		t.Fatalf("Sun radiant exitance out of expected range: %v", sunExitance)
	}

	// Fourier conduction: Aluminum (k = 205 W/(m*K)), 100°C to 20°C across 0.05m plate
	heatFlux := HeatConductionFourier(205.0, 373.15, 293.15, 0.05)
	// q = 205 * 80 / 0.05 = 328,000 W/m^2
	if math.Abs(heatFlux-328000.0) > 1e-3 {
		t.Fatalf("Fourier heat flux mismatch: expected 328000, got %v", heatFlux)
	}
}
