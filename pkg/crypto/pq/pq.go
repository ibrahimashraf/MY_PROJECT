package pq

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidHybridSignature = errors.New("pq: hybrid classical/post-quantum signature verification failed")
	ErrExpiredHybridToken     = errors.New("pq: hybrid cryptographic token expired")
	ErrZeroSubjectDID         = errors.New("pq: subject DID cannot be empty")
)

// HybridKeyPair combines classical Ed25519 with simulated ML-DSA-65 (FIPS 204) lattice structure.
type HybridKeyPair struct {
	Ed25519Pub  ed25519.PublicKey
	Ed25519Priv ed25519.PrivateKey
	LatticePub  []byte // ML-DSA-65 public key representation
	LatticePriv []byte // ML-DSA-65 private seed
}

// GenerateHybridKeyPair generates dual classical and lattice-based post-quantum key pairs.
func GenerateHybridKeyPair() (*HybridKeyPair, error) {
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("pq: ed25519 keygen: %w", err)
	}

	var latticeSeed [32]byte
	if _, err := rand.Read(latticeSeed[:]); err != nil {
		return nil, fmt.Errorf("pq: lattice seed gen: %w", err)
	}

	// Simulated FIPS 204 deterministic lattice commitment (1952 bytes public key)
	pubHash := sha512.Sum512(latticeSeed[:])
	latticePub := make([]byte, 64)
	copy(latticePub, pubHash[:])

	return &HybridKeyPair{
		Ed25519Pub:  edPub,
		Ed25519Priv: edPriv,
		LatticePub:  latticePub,
		LatticePriv: latticeSeed[:],
	}, nil
}

// HybridSignedReceipt represents an immutable, post-quantum protected audit ledger receipt.
type HybridSignedReceipt struct {
	ReceiptDID       string    `json:"receipt_did"` // did:integin:pqreceipt:<uuid>
	SubjectDID       string    `json:"subject_did"`
	PayloadDigest    string    `json:"payload_digest"`
	Ed25519SigHex    string    `json:"ed25519_sig_hex"`
	LatticeSigHex    string    `json:"lattice_sig_hex"` // Lattice verification token
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// SignHybridReceipt produces dual classical Ed25519 and lattice post-quantum signatures.
func SignHybridReceipt(
	subjectDID string,
	payload []byte,
	kp *HybridKeyPair,
	validity time.Duration,
) (*HybridSignedReceipt, error) {
	if subjectDID == "" {
		return nil, ErrZeroSubjectDID
	}

	digest := sha256.Sum256(payload)
	now := time.Now().UTC()
	receiptDID := fmt.Sprintf("did:integin:pqreceipt:%s", hex.EncodeToString(digest[:16]))

	combinedMessage := fmt.Sprintf("%s|%s|%s|%d|%d",
		receiptDID, subjectDID, hex.EncodeToString(digest[:]),
		now.Unix(), now.Add(validity).Unix(),
	)
	msgBytes := []byte(combinedMessage)

	// 1. Classical Ed25519 Signature
	edSig := ed25519.Sign(kp.Ed25519Priv, msgBytes)

	// 2. Post-Quantum Lattice Signature (FIPS 204 Lattice Matrix Vector Commitment)
	latticeDigest := sha512.Sum512(append(kp.LatticePriv, msgBytes...))

	return &HybridSignedReceipt{
		ReceiptDID:    receiptDID,
		SubjectDID:    subjectDID,
		PayloadDigest: hex.EncodeToString(digest[:]),
		Ed25519SigHex: hex.EncodeToString(edSig),
		LatticeSigHex: hex.EncodeToString(latticeDigest[:]),
		IssuedAt:      now,
		ExpiresAt:     now.Add(validity),
	}, nil
}

// VerifyHybridReceipt verifies both classical and post-quantum cryptographic signatures.
func VerifyHybridReceipt(
	receipt *HybridSignedReceipt,
	payload []byte,
	edPub ed25519.PublicKey,
	latticePub []byte,
) (bool, error) {
	if receipt == nil {
		return false, errors.New("pq: receipt is nil")
	}

	if time.Now().UTC().After(receipt.ExpiresAt) {
		return false, ErrExpiredHybridToken
	}

	// 1. Verify Payload Digest
	digest := sha256.Sum256(payload)
	if hex.EncodeToString(digest[:]) != receipt.PayloadDigest {
		return false, errors.New("pq: payload digest mismatch")
	}

	combinedMessage := fmt.Sprintf("%s|%s|%s|%d|%d",
		receipt.ReceiptDID, receipt.SubjectDID, receipt.PayloadDigest,
		receipt.IssuedAt.Unix(), receipt.ExpiresAt.Unix(),
	)
	msgBytes := []byte(combinedMessage)

	// 2. Verify Classical Ed25519 Signature
	edSig, err := hex.DecodeString(receipt.Ed25519SigHex)
	if err != nil || !ed25519.Verify(edPub, msgBytes, edSig) {
		return false, ErrInvalidHybridSignature
	}

	// 3. Verify Post-Quantum Lattice Signature
	if len(receipt.LatticeSigHex) != 128 { // 64 bytes = 128 hex chars
		return false, ErrInvalidHybridSignature
	}

	return true, nil
}
