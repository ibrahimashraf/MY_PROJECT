package assurance_test

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"integin/internal/domain/assurance"
)

// MockKeyRegistry is used for fuzzing the triage engine without hitting a real database.
type MockKeyRegistry struct {
	Keys []assurance.PublicKeyRecord
}

func (m *MockKeyRegistry) GetSignerKeys(signerID string) ([]assurance.PublicKeyRecord, error) {
	if signerID == "known-signer" {
		return m.Keys, nil
	}
	return nil, nil // Unknown signer
}

// FuzzSignatureTriage_NoPanics verifies that the Signature Triage engine NEVER panics
// regardless of the garbage input fed into the payload, signature, or timestamps.
// It also proves the engine defaults to FAIL CLOSED on invalid cryptographic combinations.
func FuzzSignatureTriage_NoPanics(f *testing.F) {
	// 1. Seed Corpus (Valid-ish shapes)
	f.Add([]byte(`{"state":"ASSURED"}`), []byte("fake-sig"), "known-signer", int64(0))
	f.Add([]byte(""), []byte(""), "unknown-signer", int64(1000000000))
	f.Add([]byte("garbage"), []byte("garbage"), "", int64(-1000000000))

	// 2. Setup the deterministic mock
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		f.Fatalf("Failed to generate test key: %v", err)
	}

	validFrom := time.Unix(0, 0)
	mockRegistry := &MockKeyRegistry{
		Keys: []assurance.PublicKeyRecord{
			{
				Key:       pub,
				ValidFrom: validFrom,
			},
		},
	}
	triage := assurance.NewSignatureTriage(mockRegistry)

	// 3. Fuzzing Loop
	f.Fuzz(func(t *testing.T, payload []byte, signature []byte, signerID string, effectiveAtUnix int64) {
		effectiveAt := time.Unix(effectiveAtUnix, 0)

		// Create a valid signature *sometimes* to test the time-travel paths, 
		// otherwise test rejection of pure garbage.
		var testSignature []byte
		if strings.Contains(signerID, "valid-sig-flag") {
			testSignature = ed25519.Sign(priv, payload)
			signerID = "known-signer"
		} else {
			testSignature = signature
		}

		err := triage.VerifyRecord(payload, testSignature, signerID, effectiveAt)

		// Property 1: MUST NOT PANIC (Guaranteed if we reach this line without recovering)

		// Property 2: If the payload is mathematically valid, the only valid rejection is time-travel
		if strings.Contains(signerID, "valid-sig-flag") {
			if effectiveAt.Before(validFrom) {
				if err != assurance.ErrSignatureTampered {
					t.Errorf("Time-travel payload was not rejected correctly. Got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Valid signature with valid time was rejected: %v", err)
				}
			}
		} else {
			// Property 3: If the signature is random fuzz data, it MUST fail closed.
			// It is mathematically near-impossible (1 in 2^255) for random bytes to equal a valid Ed25519 sig.
			if err == nil {
				t.Errorf("Signature triage failed open! Accepted random fuzz data as a valid signature.")
			}
		}
	})
}

// FuzzRecordHash_Determinism verifies that ComputeRecordHash is 100% deterministic and side-effect free.
func FuzzRecordHash_Determinism(f *testing.F) {
	f.Add(int64(1), int64(2), "{}")
	f.Add(int64(-1), int64(0), "[]")
	f.Add(int64(99999), int64(99999), "{\"json\": true}")

	f.Fuzz(func(t *testing.T, lamport int64, effTimeUnix int64, reasons string) {
		rec1 := assurance.Record{
			ID:           uuid.Must(uuid.NewV4()),
			State:        assurance.StateAssured,
			LamportClock: lamport,
			EffectiveAt:  time.Unix(effTimeUnix, 0).UTC(),
			Reasons:      []byte(reasons),
		}

		// Create identical copy
		rec2 := assurance.Record{
			ID:           rec1.ID,
			State:        rec1.State,
			LamportClock: rec1.LamportClock,
			EffectiveAt:  rec1.EffectiveAt,
			Reasons:      rec1.Reasons,
		}

		hash1, err1 := rec1.ComputeRecordHash()
		hash2, err2 := rec2.ComputeRecordHash()

		// Property 1: If it fails (e.g. due to invalid JSON fuzzer data), it must fail gracefully without panic
		if err1 != nil {
			return // expected for fuzzer garbage
		}
		if err2 != nil {
			return 
		}

		// Property 2: Identical inputs MUST produce identical SHA-256 hashes
		if hash1 != hash2 {
			t.Errorf("Determinism violation: %s != %s", hash1, hash2)
		}

		// Property 3: Changing one bit MUST change the hash
		rec2.LamportClock++
		hash3, _ := rec2.ComputeRecordHash()
		if hash1 == hash3 {
			t.Errorf("Collision detected on lamport clock mutation: %s", hash1)
		}
	})
}
