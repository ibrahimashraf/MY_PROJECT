package rulesengine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"
)

func TestVerifyHarmonicJitterAuthenticSensor(t *testing.T) {
	// Generate 32 samples simulating real physical crane load cell with 25Hz mechanical ripple
	// and realistic Gaussian micro-noise (100 Hz sampling rate)
	samples := make([]TelemetricSample, 32)
	baseLoad := 50.0 // 50 kN baseline load
	sampleRate := 100.0
	intervalNs := int64(1e9 / sampleRate)

	for i := 0; i < 32; i++ {
		timeSec := float64(i) / sampleRate
		// 25Hz structural vibration harmonic + secondary harmonics + physical sensor noise
		harmonicRipple := 0.8 * math.Sin(2.0*math.Pi*25.0*timeSec)
		harmonicSecondary := 0.3 * math.Cos(2.0*math.Pi*12.0*timeSec)
		thermalJitter := 0.15 * math.Sin(float64(i*13+5))

		samples[i] = TelemetricSample{
			TimestampNs: int64(i) * intervalNs,
			Value:       baseLoad + harmonicRipple + harmonicSecondary + thermalJitter,
		}
	}

	profile, err := VerifyHarmonicJitter(samples, 0.2)
	if err != nil {
		t.Fatalf("expected authentic jitter verification to pass, got: %v", err)
	}

	if !profile.IsAuthentic {
		t.Errorf("expected IsAuthentic to be true")
	}

	if profile.StandardDev <= 0 {
		t.Errorf("expected non-zero standard deviation, got %f", profile.StandardDev)
	}

	if profile.HarmonicPeakHz < 20.0 || profile.HarmonicPeakHz > 30.0 {
		t.Errorf("expected peak harmonic near 25Hz, got %f Hz", profile.HarmonicPeakHz)
	}
}

func TestVerifyHarmonicJitterRejectsSyntheticFlatSignal(t *testing.T) {
	// Perfectly flat synthetic signal (exact copy-paste counterfeit)
	samples := make([]TelemetricSample, 16)
	for i := 0; i < 16; i++ {
		samples[i] = TelemetricSample{
			TimestampNs: int64(i * 10000000),
			Value:       50.000000,
		}
	}

	_, err := VerifyHarmonicJitter(samples, 0.5)
	if err != ErrSyntheticJitterDetected {
		t.Fatalf("expected ErrSyntheticJitterDetected for flat signal, got: %v", err)
	}
}

func TestVerifyHarmonicJitterRejectsInsufficientSamples(t *testing.T) {
	samples := make([]TelemetricSample, 4)
	for i := 0; i < 4; i++ {
		samples[i] = TelemetricSample{
			TimestampNs: int64(i * 10000000),
			Value:       50.0 + float64(i),
		}
	}

	_, err := VerifyHarmonicJitter(samples, 0.5)
	if err != ErrInsufficientSamples {
		t.Fatalf("expected ErrInsufficientSamples, got: %v", err)
	}
}

func TestVerifySensorBLEBinding(t *testing.T) {
	key := []byte("hardware-ble-enclave-key-12345678")
	payloadDigest := sha256.Sum256([]byte("simulated-sensor-readings-telemetry-block"))

	mac := hmac.New(sha256.New, key)
	mac.Write(payloadDigest[:])
	validSig := hex.EncodeToString(mac.Sum(nil))

	// 1. Valid Signature
	if err := VerifySensorBLEBinding(payloadDigest[:], key, validSig); err != nil {
		t.Fatalf("expected valid BLE binding to pass, got: %v", err)
	}

	// 2. Tampered Key
	wrongKey := []byte("wrong-unpaired-ble-hardware-key!")
	if err := VerifySensorBLEBinding(payloadDigest[:], wrongKey, validSig); err != ErrInvalidSensorSignature {
		t.Fatalf("expected ErrInvalidSensorSignature for forged key, got: %v", err)
	}

	// 3. Tampered Payload
	corruptDigest := sha256.Sum256([]byte("corrupted-telemetry"))
	if err := VerifySensorBLEBinding(corruptDigest[:], key, validSig); err != ErrInvalidSensorSignature {
		t.Fatalf("expected ErrInvalidSensorSignature for tampered digest, got: %v", err)
	}
}
