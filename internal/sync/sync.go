package sync

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/security"
	"integin/internal/shared/types"
)

type Outcome string

const (
	Applied         Outcome = "APPLIED"
	Duplicate       Outcome = "DUPLICATE"
	Queued          Outcome = "QUEUED"
	Held            Outcome = "HELD"
	Rejected        Outcome = "REJECTED"
	Conflict        Outcome = "CONFLICT"
	SecurityFailure Outcome = "SECURITY_FAILURE"
)

type Transaction struct {
	ProtocolVersion    string
	TransactionID      string
	TenantID           string
	OrganizationID     string
	Environment        string
	DeviceID           string
	UserID             string
	SequenceNumber     uint64
	Operation          string
	EntityID           string
	Payload            []byte
	PayloadHash        string
	CapturedAt         time.Time
	AuthorityID        string
	AuthorityEpoch     uint64
	SignatureAlgorithm string
	KeyID              string
	Signature          string
}

func NewTransaction(transactionID, tenantID, deviceID, userID string, sequence uint64, operation string, payload []byte) Transaction {
	copied := append([]byte(nil), payload...)
	return Transaction{ProtocolVersion: "v1", TransactionID: transactionID, TenantID: tenantID, Environment: "LIVE", DeviceID: deviceID, UserID: userID, SequenceNumber: sequence, Operation: operation, Payload: copied, PayloadHash: HashPayload(copied), CapturedAt: time.Now().UTC(), SignatureAlgorithm: "HMAC-SHA256"}
}

func SignTransaction(transaction Transaction, secret string) Transaction {
	transaction.Signature = signTransaction(transaction, secret)
	transaction.Payload = append([]byte(nil), transaction.Payload...)
	return transaction
}

func SignTransactionEd25519(transaction Transaction, privateKey ed25519.PrivateKey, keyID string) (Transaction, error) {
	transaction.ProtocolVersion = "v1"
	transaction.SignatureAlgorithm = "Ed25519"
	transaction.KeyID = keyID
	signature, err := security.SignDeviceMutation(privateKey, []byte(canonicalTransaction(transaction)))
	if err != nil {
		return Transaction{}, err
	}
	transaction.Signature = base64.StdEncoding.EncodeToString(signature)
	transaction.Payload = append([]byte(nil), transaction.Payload...)
	return transaction, nil
}
func HashPayload(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

type Result struct {
	Outcome          Outcome
	Code             types.ErrorCode
	TransactionID    string
	Reason           string
	ExpectedSequence uint64
	PayloadHash      string
}

type Processor struct {
	secret       string
	devices      map[string]device_trust.Device
	lastSequence map[string]uint64
	transactions map[string]string
	held         map[string]Transaction
	// HMACFallbackCount counts legacy HMAC-SHA256 verifications accepted
	// through the deprecated shared-secret fallback path. Removal target:
	// once every device carries a device-bound Ed25519 key (migration
	// condition: count stays zero across a full release cycle).
	HMACFallbackCount atomic.Int64
}

func NewProcessor(secret string) (*Processor, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("sync signing secret is required")
	}
	return &Processor{secret: secret, devices: make(map[string]device_trust.Device), lastSequence: make(map[string]uint64), transactions: make(map[string]string), held: make(map[string]Transaction)}, nil
}
func (p *Processor) RegisterDevice(device device_trust.Device) { p.devices[device.ID()] = device }
func (p *Processor) HeldTransactions() []Transaction {
	result := make([]Transaction, 0, len(p.held))
	for _, transaction := range p.held {
		transaction.Payload = append([]byte(nil), transaction.Payload...)
		result = append(result, transaction)
	}
	return result
}

// Submit validates an offline transaction and returns an explicit outcome. No
// business state is mutated until identity, authority, hash, signature, and
// sequence checks all pass.
func (p *Processor) Submit(transaction Transaction, authority device_trust.AuthorityPackage, at time.Time) Result {
	result := Result{TransactionID: transaction.TransactionID, PayloadHash: transaction.PayloadHash}
	if strings.TrimSpace(transaction.TransactionID) == "" || strings.TrimSpace(transaction.TenantID) == "" || strings.TrimSpace(transaction.DeviceID) == "" || strings.TrimSpace(transaction.UserID) == "" || strings.TrimSpace(transaction.Operation) == "" || transaction.SequenceNumber == 0 {
		return p.fail(result, types.ErrValidation, Rejected, "transaction identity and sequence are required")
	}
	calculatedHash := HashPayload(transaction.Payload)
	if transaction.PayloadHash != calculatedHash {
		return p.fail(result, types.ErrRejected, SecurityFailure, "payload hash mismatch")
	}
	if existingHash, exists := p.transactions[transaction.TransactionID]; exists {
		if existingHash == transaction.PayloadHash {
			result.Outcome = Duplicate
			result.Reason = "transaction already applied"
			return result
		}
		return p.fail(result, types.ErrConflict, Conflict, "transaction id was reused with different payload")
	}
	device, exists := p.devices[transaction.DeviceID]
	if !exists {
		return p.fail(result, types.ErrUnauthorized, SecurityFailure, "device is not registered")
	}
	if transaction.TenantID != device.TenantID() || transaction.UserID != device.UserID() {
		return p.fail(result, types.ErrTenantMismatch, SecurityFailure, "transaction identity does not match device authority")
	}
	if err := device_trust.ValidateAuthorityPackage(authority, device, p.secret, at); err != nil {
		return p.fail(result, types.ErrUnauthorized, SecurityFailure, err.Error())
	}
	if transaction.SignatureAlgorithm == "Ed25519" {
		publicKey, publicKeyErr := base64.StdEncoding.DecodeString(device.PublicKey())
		signature, signatureErr := base64.StdEncoding.DecodeString(transaction.Signature)
		validKey := publicKeyErr == nil && transaction.KeyID != "" && transaction.KeyID == security.DeviceKeyID(ed25519.PublicKey(publicKey))
		if !validKey || signatureErr != nil || !security.VerifyDeviceMutation(ed25519.PublicKey(publicKey), []byte(canonicalTransaction(transaction)), signature) {
			return p.fail(result, types.ErrRejected, SecurityFailure, "transaction Ed25519 signature is invalid")
		}
	} else {
		// Legacy HMAC-SHA256 shared-secret fallback, DEPRECATED.
		// Exists only for pre-key tablets. Keyed devices must use Ed25519.
		// Removal: when HMACFallbackCount stays zero across a release cycle,
		// delete this branch and the single-secret signing path.
		if p.deviceHasRegisteredKey(device) {
			return p.fail(result, types.ErrRejected, SecurityFailure, "device carries registered key material; HMAC-SHA256 shared-secret signing is deprecated")
		}
		p.HMACFallbackCount.Add(1)
		log.Printf("sync: HMAC-SHA256 transaction %s accepted via DEPRECATED shared-secret fallback; migrate device %s to Ed25519 device-bound keys", transaction.TransactionID, transaction.DeviceID)
		if !hmac.Equal([]byte(transaction.Signature), []byte(signTransaction(transaction, p.secret))) {
			return p.fail(result, types.ErrRejected, SecurityFailure, "transaction HMAC signature is invalid")
		}
	}
	expected := p.lastSequence[transaction.DeviceID] + 1
	result.ExpectedSequence = expected
	if transaction.SequenceNumber > expected {
		p.held[transaction.TransactionID] = cloneTransaction(transaction)
		result.Outcome, result.Code, result.Reason = Held, types.ErrHeld, fmt.Sprintf("sequence gap: expected %d", expected)
		return result
	}
	if transaction.SequenceNumber < expected {
		return p.fail(result, types.ErrConflict, Conflict, fmt.Sprintf("stale sequence: expected %d", expected))
	}
	p.lastSequence[transaction.DeviceID] = transaction.SequenceNumber
	p.transactions[transaction.TransactionID] = transaction.PayloadHash
	result.Outcome, result.Reason = Applied, "transaction applied"
	return result
}

func (p *Processor) fail(result Result, code types.ErrorCode, outcome Outcome, reason string) Result {
	result.Code, result.Outcome, result.Reason = code, outcome, reason
	return result
}
func cloneTransaction(transaction Transaction) Transaction {
	transaction.Payload = append([]byte(nil), transaction.Payload...)
	return transaction
}
func signTransaction(transaction Transaction, secret string) string {
	canonical := canonicalTransaction(transaction)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// deviceHasRegisteredKey reports whether the device carries a valid Ed25519
// public key suitable for device-bound signing. Pre-key tablets store a
// non-key placeholder (e.g. "public-key") that fails this decode; only those
// devices may fall through to the deprecated HMAC-SHA256 shared-secret path.
func (p *Processor) deviceHasRegisteredKey(device device_trust.Device) bool {
	pub, err := base64.StdEncoding.DecodeString(device.PublicKey())
	return err == nil && len(pub) == ed25519.PublicKeySize
}

func canonicalTransaction(transaction Transaction) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%d|%s|%s|%s|%s|%d|%s|%s|%s", transaction.ProtocolVersion, transaction.TransactionID, transaction.TenantID, transaction.OrganizationID, transaction.Environment, transaction.DeviceID, transaction.UserID, transaction.SequenceNumber, transaction.Operation, transaction.EntityID, transaction.PayloadHash, transaction.AuthorityID, transaction.AuthorityEpoch, transaction.CapturedAt.UTC().Format(time.RFC3339Nano), transaction.SignatureAlgorithm, transaction.KeyID)
}
