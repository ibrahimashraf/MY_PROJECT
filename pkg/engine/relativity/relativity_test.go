package relativity

import (
	"math"
	"testing"
)

func TestLorentzTimeDilationAndLengthContraction(t *testing.T) {
	// Proper time 1s at v = 0.6c -> gamma = 1 / sqrt(1 - 0.36) = 1 / 0.8 = 1.25
	dilated, gamma := TimeDilation(1.0, 0.6*SpeedOfLight)
	if math.Abs(gamma-1.25) > 1e-6 {
		t.Fatalf("expected gamma 1.25, got %v", gamma)
	}
	if math.Abs(dilated-1.25) > 1e-6 {
		t.Fatalf("expected dilated time 1.25, got %v", dilated)
	}

	// Length contraction: rest length 10m at 0.6c -> L = 10 * 0.8 = 8m
	contracted := LengthContraction(10.0, 0.6*SpeedOfLight)
	if math.Abs(contracted-8.0) > 1e-6 {
		t.Fatalf("expected length 8m, got %v", contracted)
	}

	// Superluminal / speed of light limits
	infDilated, infGamma := TimeDilation(1.0, SpeedOfLight)
	if !math.IsInf(infDilated, 1) || !math.IsInf(infGamma, 1) {
		t.Fatalf("expected +Inf at speed of light")
	}
	zeroL := LengthContraction(10.0, SpeedOfLight)
	if zeroL != 0.0 {
		t.Fatalf("expected 0 length at speed of light, got %v", zeroL)
	}
}

func TestRelativisticVelocityAddition(t *testing.T) {
	// 0.8c + 0.8c: (0.8 + 0.8)/(1 + 0.64) = 1.6 / 1.64 ≈ 0.9756c
	u := 0.8 * SpeedOfLight
	v := 0.8 * SpeedOfLight
	composed := RelativisticVelocityAddition(u, v)
	expected := (1.6 / 1.64) * SpeedOfLight
	if math.Abs(composed-expected) > 1e-3 {
		t.Fatalf("expected composed velocity %v, got %v", expected, composed)
	}

	// c + c = c
	cAdded := RelativisticVelocityAddition(SpeedOfLight, SpeedOfLight)
	if math.Abs(cAdded-SpeedOfLight) > 1e-3 {
		t.Fatalf("expected c + c = c, got %v", cAdded)
	}

	// Denominator 0 edge case: u = c, v = -c -> 1 - 1 = 0
	infComp := RelativisticVelocityAddition(SpeedOfLight, -SpeedOfLight)
	if !math.IsInf(infComp, 1) {
		t.Fatalf("expected Inf when denom is 0, got %v", infComp)
	}
}

func TestRelativisticEnergyMomentum(t *testing.T) {
	// Rest mass 1 kg at rest
	rest := RelativisticEnergyMomentum(1.0, 0.0)
	expectedRestE := SpeedOfLight * SpeedOfLight
	if math.Abs(rest.RestEnergyJ-expectedRestE) > 1e-3 {
		t.Fatalf("expected E=mc^2 %v, got %v", expectedRestE, rest.RestEnergyJ)
	}
	if rest.KineticEnergy != 0 || rest.MomentumKgmS != 0 || rest.Gamma != 1.0 {
		t.Fatalf("rest particle must have zero kinetic energy, momentum and gamma=1")
	}

	// 1 kg at 0.6c -> gamma = 1.25, totalE = 1.25 mc^2, p = 1.25 * m * 0.6c = 0.75 mc
	rel := RelativisticEnergyMomentum(1.0, 0.6*SpeedOfLight)
	if math.Abs(rel.Gamma-1.25) > 1e-6 {
		t.Fatalf("expected gamma 1.25, got %v", rel.Gamma)
	}
	expectedTotalE := 1.25 * expectedRestE
	if math.Abs(rel.TotalEnergyJ-expectedTotalE) > 1.0 {
		t.Fatalf("expected total energy %v, got %v", expectedTotalE, rel.TotalEnergyJ)
	}

	// Invariant relation: E^2 = (p*c)^2 + (m*c^2)^2
	pc := rel.MomentumKgmS * SpeedOfLight
	lhs := rel.TotalEnergyJ * rel.TotalEnergyJ
	rhs := (pc * pc) + (rel.RestEnergyJ * rel.RestEnergyJ)
	if math.Abs(lhs-rhs)/lhs > 1e-6 {
		t.Fatalf("Energy-momentum invariant violated: lhs %v != rhs %v", lhs, rhs)
	}

	// Limit at c
	cLim := RelativisticEnergyMomentum(1.0, SpeedOfLight)
	if !math.IsInf(cLim.TotalEnergyJ, 1) {
		t.Fatalf("expected +Inf total energy at c")
	}
}

func TestRelativisticDoppler(t *testing.T) {
	f0 := 1e9 // 1 GHz
	v := 0.6 * SpeedOfLight

	// Approaching: sqrt((1 + 0.6) / (1 - 0.6)) = sqrt(1.6 / 0.4) = sqrt(4) = 2.0
	fBlue := RelativisticDoppler(f0, v, true)
	if math.Abs(fBlue-2.0e9) > 1e-3 {
		t.Fatalf("expected blue shift 2.0 GHz, got %v", fBlue)
	}

	// Receding: sqrt(0.4 / 1.6) = sqrt(0.25) = 0.5
	fRed := RelativisticDoppler(f0, v, false)
	if math.Abs(fRed-0.5e9) > 1e-3 {
		t.Fatalf("expected red shift 0.5 GHz, got %v", fRed)
	}

	// Transverse: f' = f0 * sqrt(1 - 0.36) = 0.8 * f0
	fTrans := TransverseRelativisticDoppler(f0, v)
	if math.Abs(fTrans-0.8e9) > 1e-3 {
		t.Fatalf("expected transverse Doppler 0.8 GHz, got %v", fTrans)
	}

	// Boundary at c
	if !math.IsInf(RelativisticDoppler(f0, SpeedOfLight, true), 1) {
		t.Fatalf("expected Inf approaching at c")
	}
	if RelativisticDoppler(f0, SpeedOfLight, false) != 0.0 {
		t.Fatalf("expected 0 receding at c")
	}
	if TransverseRelativisticDoppler(f0, SpeedOfLight) != 0.0 {
		t.Fatalf("expected 0 transverse at c")
	}
}

func TestGeneralRelativityAndEinsteinInvariants(t *testing.T) {
	// 1. Schwarzschild Radius: Solar mass = 1.989e30 kg -> rs ≈ 2953 m
	rs := SchwarzschildRadius(1.989e30)
	if rs < 2900 || rs > 3100 {
		t.Fatalf("solar Schwarzschild radius out of range: %v", rs)
	}
	if SchwarzschildRadius(0) != 0 {
		t.Fatalf("zero mass should return zero rs")
	}

	// 2. Gravitational Deflection of Light (Einstein 1919 Solar Eclipse Prediction):
	// Solar radius R_sun ≈ 6.9634e8 m -> theta ≈ 1.751 arcseconds (8.488e-6 rad)
	thetaRad := EinsteinLightDeflection(1.989e30, 6.9634e8)
	thetaArcsec := thetaRad * (180.0 * 3600.0 / math.Pi)
	if math.Abs(thetaArcsec-1.751) > 0.05 {
		t.Fatalf("Einstein 1919 light deflection mismatch: expected ~1.751 arcsec, got %v", thetaArcsec)
	}
	if EinsteinLightDeflection(1.989e30, 0) != 0 {
		t.Fatalf("non-positive impact parameter should return 0")
	}

	// 3. Gravitational Time Dilation & Redshift
	factor := GravitationalTimeDilation(5.972e24, 6371e3) // Earth surface
	if factor <= 0 || factor > 1.0 {
		t.Fatalf("Earth gravitational dilation out of range: %v", factor)
	}
	zg := GravitationalRedshift(5.972e24, 6371e3)
	if zg <= 0 || zg > 1e-8 {
		t.Fatalf("Earth gravitational redshift out of range: %v", zg)
	}

	// Horizon singularity: inside event horizon
	if GravitationalTimeDilation(1.989e30, 1000.0) != 0.0 {
		t.Fatalf("expected 0 inside event horizon")
	}
	if !math.IsInf(GravitationalRedshift(1.989e30, 1000.0), 1) {
		t.Fatalf("expected Inf redshift inside horizon")
	}

	// 4. Einstein Field Equations Coupling Constant kappa = 8*pi*G / c^4 ≈ 2.076e-43
	kappa := EinsteinCouplingConstant()
	expectedKappa := (8.0 * math.Pi * GravitationalConstant) / math.Pow(SpeedOfLight, 4)
	if math.Abs(kappa-expectedKappa) > 1e-45 {
		t.Fatalf("Einstein coupling constant mismatch: expected %v, got %v", expectedKappa, kappa)
	}

	// 5. Gravitational Wave Quadrupole Radiation (Einstein 1918)
	pGW := GravitationalWaveQuadrupoleLuminosity(1e35)
	if pGW <= 0 {
		t.Fatalf("expected positive GW radiation power")
	}

	// 6. Geodetic Precession (Gravity Probe B: Earth orbit ~6.6 arcsec/yr)
	prec := GeodeticPrecession(5.972e24, 7000e3, 7500.0)
	if prec <= 0 {
		t.Fatalf("expected positive geodetic precession")
	}
	if GeodeticPrecession(5.972e24, 0, 7500.0) != 0 {
		t.Fatalf("zero radius should return 0 precession")
	}

	// 7. Kepler Orbital Period & Hubble Law
	T := KeplerOrbitalPeriod(6778e3, 5.972e24)
	if T < 5400 || T > 5800 {
		t.Fatalf("ISS orbital period out of range: %v", T)
	}
	if KeplerOrbitalPeriod(0, 5.972e24) != 0 || KeplerOrbitalPeriod(6778e3, 0) != 0 {
		t.Fatalf("zero orbital params should return 0")
	}

	vHubble := HubbleRecessionalVelocity(10.0, 70.0)
	if vHubble != 700000.0 {
		t.Fatalf("Hubble velocity mismatch: expected 700000 m/s, got %v", vHubble)
	}
	vHubbleDef := HubbleRecessionalVelocity(10.0, 0)
	if vHubbleDef != 700000.0 {
		t.Fatalf("Default Hubble constant mismatch: %v", vHubbleDef)
	}
}
