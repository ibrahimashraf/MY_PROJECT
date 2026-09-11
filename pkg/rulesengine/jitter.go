package rulesengine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
)

var (
	ErrInsufficientSamples   = errors.New("jitter: minimum 8 telemetric samples required for harmonic frequency analysis")
	ErrSyntheticJitterDetected = errors.New("jitter: synthetic uniform or zero jitter detected (counterfeit spoofing)")
	ErrInvalidSensorSignature  = errors.New("jitter: sensor hardware BLE enclave signature verification failed")
	ErrZeroDivision            = errors.New("jitter: sample variance is zero")
)

// TelemetricSample represents a single high-frequency load sensor reading.
type TelemetricSample struct {
	TimestampNs int64   `json:"timestamp_ns"` // Nanosecond monotonic hardware clock
	Value       float64 `json:"value"`        // Force / load measurement in kN or tonnes
}

// JitterProfile captures the micro-ripple frequency distribution and entropy metrics.
type JitterProfile struct {
	SampleCount      int     `json:"sample_count"`
	MeanValue        float64 `json:"mean_value"`
	StandardDev      float64 `json:"standard_dev"`
	CoefficientOfVar float64 `json:"coefficient_of_var"`
	HarmonicPeakHz   float64 `json:"harmonic_peak_hz"`
	SpectralEntropy  float64 `json:"spectral_entropy"`
	IsAuthentic      bool    `json:"is_authentic"`
}

// VerifyHarmonicJitter evaluates telemetric load samples for authentic physical sensor micro-ripple.
// Physical strain-gauge load cells exhibit characteristic mechanical vibration harmonics (10Hz-120Hz)
// and non-zero physical Gaussian jitter. Synthetic generators exhibit flat, uniform, or zero variance.
func VerifyHarmonicJitter(samples []TelemetricSample, minEntropy float64) (*JitterProfile, error) {
	n := len(samples)
	if n < 8 {
		return nil, ErrInsufficientSamples
	}

	// 1. Calculate Mean and Variance
	var sum float64
	for _, s := range samples {
		if math.IsNaN(s.Value) || math.IsInf(s.Value, 0) {
			return nil, errors.New("jitter: NaN/Inf reading detected in telemetry")
		}
		sum += s.Value
	}
	mean := sum / float64(n)

	var varianceSum float64
	for _, s := range samples {
		diff := s.Value - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(n)
	stdDev := math.Sqrt(variance)

	// A real physical sensor under mechanical strain NEVER produces exact zero variance.
	if stdDev < 1e-9 {
		return nil, ErrSyntheticJitterDetected
	}

	cv := 0.0
	if math.Abs(mean) > 1e-9 {
		cv = stdDev / math.Abs(mean)
	}

	// 2. Discrete Fourier Transform (DFT) for Dominant Harmonic Frequency Detection
	// Sample frequency estimate from timestamps
	durationNs := samples[n-1].TimestampNs - samples[0].TimestampNs
	if durationNs <= 0 {
		return nil, errors.New("jitter: invalid non-monotonic sample timestamps")
	}
	sampleRateHz := float64(n-1) / (float64(durationNs) / 1e9)

	halfN := n / 2
	powerSpectrum := make([]float64, halfN)
	var totalPower float64
	maxPower := -1.0
	peakBin := 0

	for k := 1; k < halfN; k++ {
		var realPart, imagPart float64
		for t := 0; t < n; t++ {
			angle := 2.0 * math.Pi * float64(k*t) / float64(n)
			normalizedVal := samples[t].Value - mean
			realPart += normalizedVal * math.Cos(angle)
			imagPart -= normalizedVal * math.Sin(angle)
		}
		power := (realPart*realPart + imagPart*imagPart) / float64(n)
		powerSpectrum[k] = power
		totalPower += power
		if power > maxPower {
			maxPower = power
			peakBin = k
		}
	}

	peakFrequencyHz := (float64(peakBin) * sampleRateHz) / float64(n)

	// 3. Shannon Spectral Entropy Calculation
	// Natural micro-vibrations have distributed spectral entropy across the frequency bins.
	// Normalizing by log2(halfN - 1) scales entropy to [0.0, 1.0].
	var entropy float64
	numBins := float64(halfN - 1)
	if totalPower > 0 && numBins > 1 {
		for k := 1; k < halfN; k++ {
			if powerSpectrum[k] > 0 {
				p := powerSpectrum[k] / totalPower
				entropy -= p * math.Log2(p)
			}
		}
		// Normalize by max theoretical entropy
		maxEntropy := math.Log2(numBins)
		if maxEntropy > 0 {
			entropy = entropy / maxEntropy
		}
	}

	if minEntropy > 0 && entropy < minEntropy {
		return nil, ErrSyntheticJitterDetected
	}

	profile := &JitterProfile{
		SampleCount:      n,
		MeanValue:        mean,
		StandardDev:      stdDev,
		CoefficientOfVar: cv,
		HarmonicPeakHz:   peakFrequencyHz,
		SpectralEntropy:  entropy,
		IsAuthentic:      true,
	}

	return profile, nil
}

// VerifySensorBLEBinding verifies that telemetric readings originated from a genuine,
// calibrated hardware load cell paired with the tablet's Secure Enclave over BLE.
// payloadDigest: SHA-256 of raw sample readings
// bleDeviceKey: Hardware sensor shared secret or provisioned enclave key
// sensorSignature: HMAC-SHA256 signature emitted by the certified load cell microcontroller
func VerifySensorBLEBinding(payloadDigest []byte, bleDeviceKey []byte, expectedHexSignature string) error {
	if len(bleDeviceKey) < 16 {
		return errors.New("jitter: BLE enclave pairing key must be at least 16 bytes")
	}
	mac := hmac.New(sha256.New, bleDeviceKey)
	mac.Write(payloadDigest)
	expectedMAC := mac.Sum(nil)

	expectedSigBytes, err := hex.DecodeString(expectedHexSignature)
	if err != nil {
		return ErrInvalidSensorSignature
	}

	if !hmac.Equal(expectedMAC, expectedSigBytes) {
		return ErrInvalidSensorSignature
	}

	return nil
}
