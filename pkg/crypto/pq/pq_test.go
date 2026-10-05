package pq

import (
	"testing"
	"time"
)

func TestHybridReceipt_SignAndVerify(t *testing.T) {
	kp, err := GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("failed to generate hybrid key pair: %v", err)
	}

	payload := []byte("{\"proof_load_tonnes\": 125.0, \"status\": \"PASSED\"}")
	receipt, err := SignHybridReceipt("did:integin:asset:crane-777", payload, kp, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to sign hybrid receipt: %v", err)
	}

	ok, err := VerifyHybridReceipt(receipt, payload, kp.Ed25519Pub, kp.LatticePub)
	if err != nil || !ok {
		t.Fatalf("expected hybrid verification success, got ok=%v err=%v", ok, err)
	}

	// Payload tampering test
	tamperedPayload := []byte("{\"proof_load_tonnes\": 999.0, \"status\": \"PASSED\"}")
	ok, err = VerifyHybridReceipt(receipt, tamperedPayload, kp.Ed25519Pub, kp.LatticePub)
	if ok || err == nil {
		t.Fatalf("expected verification failure on tampered payload")
	}

	// Expiry test
	receipt.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	ok, err = VerifyHybridReceipt(receipt, payload, kp.Ed25519Pub, kp.LatticePub)
	if ok || err != ErrExpiredHybridToken {
		t.Fatalf("expected ErrExpiredHybridToken, got %v", err)
	}
}

func BenchmarkVerifyHybridReceipt_HotPath(b *testing.B) {
	kp, _ := GenerateHybridKeyPair()
	payload := []byte("{\"reading\": 42.0}")
	receipt, _ := SignHybridReceipt("did:integin:asset:bench", payload, kp, 24*time.Hour)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyHybridReceipt(receipt, payload, kp.Ed25519Pub, kp.LatticePub)
	}
}
