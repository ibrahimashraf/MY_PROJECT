package evidenceregistration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/domain/workorder"
	"integin/internal/storage"
	"integin/internal/workorderauth"
)

var (
	ErrUnavailable       = errors.New("evidence registration dependencies are unavailable")
	ErrDenied            = errors.New("evidence registration is not authorized")
	ErrObjectUnavailable = errors.New("registered evidence object is unavailable")
	ErrObjectMismatch    = errors.New("registered evidence object does not match metadata")
)

type Repository interface {
	RegisterForActiveAssignment(context.Context, workorder.ActorContext, evidence.Metadata) (evidence.Metadata, bool, error)
}

type Service interface {
	Register(context.Context, RegisterCommand) (RegistrationResult, error)
}

type Dependencies struct {
	Repository Repository
	Store      storage.Store
}

type service struct{ dependencies Dependencies }

func NewService(dependencies Dependencies) (Service, error) {
	if dependencies.Repository == nil || dependencies.Store == nil {
		return nil, ErrUnavailable
	}
	return service{dependencies: dependencies}, nil
}

// RegisterCommand intentionally has no tenant, organization, actor, or object-key
// fields. Scope and storage location are derived by the service from Actor and ID.
type RegisterCommand struct {
	Actor               workorder.ActorContext
	EvidenceID          string
	InspectionID        string
	ContentType         string
	CiphertextBytes     int64
	PlaintextSHA256     string
	CiphertextSHA256    string
	CapturedAt          time.Time
	DeviceID            string
	AuthorityID         string
	AuthorityEpoch      uint64
	TransactionID       string
	ReceiptID           string
	SignatureAlgorithm  string
	KeyID               string
	EncryptionAlgorithm string
	EncryptionKeyRef    string
	Classification      string
	RetentionReference  string
	HoldState           string
	RedactionPolicyRef  string
}

type RegistrationResult struct {
	Metadata evidence.Metadata
	Inserted bool
}

func (s service) Register(ctx context.Context, command RegisterCommand) (RegistrationResult, error) {
	if err := command.Actor.Validate(); err != nil {
		return RegistrationResult{}, err
	}
	if !canRegister(command.Actor) {
		return RegistrationResult{}, ErrDenied
	}
	metadata := evidence.Metadata{
		ID: command.EvidenceID, TenantID: command.Actor.TenantID, OrganizationID: command.Actor.OrganizationID,
		InspectionID: command.InspectionID, ObjectKey: objectKey(command.Actor, command.EvidenceID), ContentType: command.ContentType,
		CiphertextBytes: command.CiphertextBytes, PlaintextSHA256: command.PlaintextSHA256, CiphertextSHA256: command.CiphertextSHA256,
		CapturedAt: command.CapturedAt, DeviceID: command.DeviceID, AuthorityID: command.AuthorityID, AuthorityEpoch: command.AuthorityEpoch,
		TransactionID: command.TransactionID, ReceiptID: command.ReceiptID, SignatureAlgorithm: command.SignatureAlgorithm, KeyID: command.KeyID,
		EncryptionAlgorithm: command.EncryptionAlgorithm, EncryptionKeyRef: command.EncryptionKeyRef, Classification: command.Classification,
		RetentionReference: command.RetentionReference, HoldState: command.HoldState, RedactionPolicyRef: command.RedactionPolicyRef,
		RegisteredBy: command.Actor.ActorID,
	}
	evidenceActor := evidence.ActorContext{TenantID: command.Actor.TenantID, OrganizationID: command.Actor.OrganizationID, ActorID: command.Actor.ActorID}
	if err := metadata.ValidateForRegistration(evidenceActor); err != nil {
		return RegistrationResult{}, err
	}
	object, err := s.dependencies.Store.Get(ctx, metadata.ObjectKey)
	if err != nil {
		return RegistrationResult{}, fmt.Errorf("%w: %v", ErrObjectUnavailable, err)
	}
	if object.Key != metadata.ObjectKey || strings.TrimSpace(object.ContentType) != metadata.ContentType || int64(len(object.Data)) != metadata.CiphertextBytes {
		return RegistrationResult{}, ErrObjectMismatch
	}
	digest := sha256.Sum256(object.Data)
	if hex.EncodeToString(digest[:]) != metadata.CiphertextSHA256 {
		return RegistrationResult{}, ErrObjectMismatch
	}
	stored, inserted, err := s.dependencies.Repository.RegisterForActiveAssignment(ctx, command.Actor, metadata)
	if err != nil {
		return RegistrationResult{}, err
	}
	return RegistrationResult{Metadata: stored, Inserted: inserted}, nil
}

func canRegister(actor workorder.ActorContext) bool {
	if actor.Role != "inspector" && actor.Role != "administrator" {
		return false
	}
	for _, capability := range actor.Capabilities {
		if capability == workorderauth.CapabilitySubmitPartial {
			return true
		}
	}
	return false
}

func objectKey(actor workorder.ActorContext, evidenceID string) string {
	return actor.TenantID + "/" + actor.OrganizationID + "/evidence/" + evidenceID
}
