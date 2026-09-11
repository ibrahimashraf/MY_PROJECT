package qfield

import (
	"math"
	"testing"
)

func TestCasimirForce(t *testing.T) {
	f := CasimirForce(10e-9) // 10nm plate separation
	if f >= 0 {
		t.Fatal("Casimir force must be attractive (negative)")
	}
	// At 10nm: F ≈ -pi^2 * hbar * c / (240 * 1e-36) ≈ very large negative per unit area
	if math.IsInf(f, 0) || math.IsNaN(f) {
		t.Fatal("Casimir force returned infinity or NaN")
	}
}

func TestQuantumTunneling(t *testing.T) {
	// Electron tunneling through 0.5 eV barrier with 0.1 eV KE, 1nm width
	barrierJ := 0.5 * ElementaryCharge
	energyJ := 0.1 * ElementaryCharge
	T := QuantumTunnelingProbability(barrierJ, energyJ, 1e-9, ElectronMassKg)
	if T <= 0 || T >= 1.0 {
		t.Fatalf("Tunneling probability out of (0,1): %v", T)
	}

	// Above barrier -> T = 1.0
	T2 := QuantumTunnelingProbability(0.1*ElementaryCharge, 0.5*ElementaryCharge, 1e-9, ElectronMassKg)
	if T2 != 1.0 {
		t.Fatal("Above-barrier transmission should be 1.0")
	}
}

func TestDecoherenceTime(t *testing.T) {
	// At 300K (room temperature): tau_d should be extremely short (~25 femtoseconds)
	tau := DecoherenceTime(300.0)
	if tau <= 0 {
		t.Fatal("Decoherence time must be positive")
	}
	// At absolute zero -> infinite coherence
	tauZero := DecoherenceTime(0.0)
	if !math.IsInf(tauZero, 1) {
		t.Fatal("At T=0, decoherence time should be infinite")
	}
}
