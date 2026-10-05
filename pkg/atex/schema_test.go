package atex

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"
)

func validLoop() IntrinsicSafetyLoop {
	return IntrinsicSafetyLoop{
		ApparatusDID:   "did:integin:tool:ultrasonic-gauge-01",
		BarrierDID:     "did:integin:barrier:mtl-5500",
		Zone:           Zone1,
		EnvironmentGas: GroupIIB,
		ApparatusGas:   GroupIIC, // IIC apparatus is backward safe for IIB
		ApparatusTemp:  T4,       // 135 C
		AutoIgnitionC:  180.0,    // Gas ignites at 180 C -> T4 (135 C) is safe
		ApparatusParams: IntrinsicallySafeParameters{
			Ui: 30.0, // Can withstand 30V
			Ii: 0.15, // Can withstand 150mA
			Pi: 1.0,  // Can withstand 1.0W
			Ci: 0.05, // 0.05 uF
			Li: 0.02, // 0.02 uH
		},
		BarrierParams: BarrierParameters{
			Uo: 28.0, // Barrier outputs 28V (<= 30V) -> PASS
			Io: 0.10, // Barrier outputs 100mA (<= 150mA) -> PASS
			Po: 0.80, // Barrier outputs 0.8W (<= 1.0W) -> PASS
			Co: 0.20, // Barrier drives up to 0.20 uF
			Lo: 0.10, // Barrier drives up to 0.10 uH
		},
		Cable: CableParameters{
			LengthMeters: 100.0,
			CcablePerM:   0.0005, // 0.05 uF total
			LcablePerM:   0.0002, // 0.02 uH total
		},
	}
}

func TestValidateIntrinsicSafetyLoop_Success(t *testing.T) {
	loop := validLoop()
	res, err := ValidateIntrinsicSafetyLoop(loop)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if !res.IsCompliant || res.IsolationStatus != "GALVANICALLY_SECURED" {
		t.Fatalf("expected compliant secured loop, got %+v", res)
	}
}

func TestValidateIntrinsicSafetyLoop_Breaches(t *testing.T) {
	// 1. Voltage breach (Uo > Ui)
	loopV := validLoop()
	loopV.BarrierParams.Uo = 35.0 // > 30.0V Ui
	res, err := ValidateIntrinsicSafetyLoop(loopV)
	if err != ErrVoltageBreach || res.IsCompliant {
		t.Fatalf("expected ErrVoltageBreach, got err=%v res=%+v", err, res)
	}

	// 2. Capacitance breach (Ci + Ccable > Co)
	loopC := validLoop()
	loopC.Cable.LengthMeters = 500.0 // 0.25 uF cable + 0.05 uF Ci = 0.30 uF > 0.20 uF Co
	res, err = ValidateIntrinsicSafetyLoop(loopC)
	if err != ErrCapacitanceBreach || res.IsCompliant {
		t.Fatalf("expected ErrCapacitanceBreach, got err=%v res=%+v", err, res)
	}

	// 3. Thermal lockout (Surface temp >= Auto-ignition)
	loopT := validLoop()
	loopT.ApparatusTemp = T2 // 300 C surface temp
	loopT.AutoIgnitionC = 250.0 // Gas ignites at 250 C < 300 C
	res, err = ValidateIntrinsicSafetyLoop(loopT)
	if err != ErrTemperatureExceeded || res.IsCompliant {
		t.Fatalf("expected ErrTemperatureExceeded, got err=%v res=%+v", err, res)
	}

	// 4. Incompatible Gas Group
	loopG := validLoop()
	loopG.ApparatusGas = GroupIIA   // Certified only for IIA
	loopG.EnvironmentGas = GroupIIC // Ambient requires IIC
	res, err = ValidateIntrinsicSafetyLoop(loopG)
	if err != ErrInvalidGasGroup || res.IsCompliant {
		t.Fatalf("expected ErrInvalidGasGroup, got err=%v res=%+v", err, res)
	}
}

func TestIssueZonePass(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	loop := validLoop()
	pass, err := IssueZonePass(loop, priv, 8*time.Hour)
	if err != nil {
		t.Fatalf("failed to issue zone pass: %v", err)
	}

	if pass.PassDID == "" || pass.SignatureHex == "" {
		t.Fatalf("zone pass missing DID or signature: %+v", pass)
	}

	// Signature verification
	sigBytes, _ := hex.DecodeString(pass.SignatureHex)
	digestBytes, _ := hex.DecodeString(pass.Digest)
	if !ed25519.Verify(pub, digestBytes, sigBytes) {
		t.Fatalf("signature verification failed")
	}
}

func BenchmarkValidateIntrinsicSafetyLoop_HotPath(b *testing.B) {
	loop := validLoop()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateIntrinsicSafetyLoop(loop)
	}
}
