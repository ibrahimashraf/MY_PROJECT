package assurance_test

import (
	"crypto/ed25519"
	"testing"
	"time"

	"integin/internal/domain/assurance"
)

type mockKeyRegistry struct {
	keys []assurance.PublicKeyRecord
}

func (m *mockKeyRegistry) GetSignerKeys(signerID string) ([]assurance.PublicKeyRecord, error) {
	return m.keys, nil
}

func TestSignatureTriage_VerifyRecord(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	payload := []byte(`{"asset_id":"123","state":"ASSURED"}`)
	signature := ed25519.Sign(priv, payload)
	
	now := time.Now()

	registry := &mockKeyRegistry{
		keys: []assurance.PublicKeyRecord{
			{
				Key:       pub,
				ValidFrom: now.Add(-48 * time.Hour), // Valid since 2 days ago
			},
		},
	}
	triage := assurance.NewSignatureTriage(registry)

	// 1. Happy Path (Valid Signature, Active Key)
	err := triage.VerifyRecord(payload, signature, "inspector-1", now)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	// 2. Tampering (Invalid Signature)
	badSignature := make([]byte, ed25519.SignatureSize)
	copy(badSignature, signature)
	badSignature[0] ^= 0xFF // Flip a bit
	
	err = triage.VerifyRecord(payload, badSignature, "inspector-1", now)
	if err != assurance.ErrSignatureTampered {
		t.Errorf("Expected ErrSignatureTampered, got %v", err)
	}

	// 3. Temporal Triage: 24h Revocation Grace Period
	revokedAt := now.Add(-12 * time.Hour) // Revoked 12 hours ago
	registry.keys[0].RevokedAt = &revokedAt

	err = triage.VerifyRecord(payload, signature, "inspector-1", now)
	if err != assurance.ErrQuarantineRequired {
		t.Errorf("Expected ErrQuarantineRequired, got %v", err)
	}

	// 4. Temporal Triage: Past Grace Period (Hard Reject)
	revokedAtPast := now.Add(-48 * time.Hour) // Revoked 2 days ago
	registry.keys[0].RevokedAt = &revokedAtPast

	err = triage.VerifyRecord(payload, signature, "inspector-1", now)
	if err != assurance.ErrKeyRevoked {
		t.Errorf("Expected ErrKeyRevoked, got %v", err)
	}
}
