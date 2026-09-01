// Package evidenceexport derives a sealed evidence manifest only from
// tenant-scoped registered metadata and freshly re-verified storage objects.
package evidenceexport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/exportmanifest"
	"integin/internal/storage"
)

const ProcedureVersion = "integin-stage-a-evidence-projection/v1"

var (
	ErrNilRepository       = errors.New("evidence export requires a metadata repository")
	ErrNilStore            = errors.New("evidence export requires object storage")
	ErrNoEvidence          = errors.New("evidence export has no registered metadata")
	ErrScopeMismatch       = errors.New("evidence export metadata scope mismatch")
	ErrMissingObject       = errors.New("evidence export object is unavailable")
	ErrObjectKeyMismatch   = errors.New("evidence export object key mismatch")
	ErrContentTypeMismatch = errors.New("evidence export object content type mismatch")
	ErrByteCountMismatch   = errors.New("evidence export ciphertext byte count mismatch")
	ErrDigestMismatch      = errors.New("evidence export ciphertext digest mismatch")
	ErrMixedPrivacy        = errors.New("evidence export requires a single privacy policy set")
)

type Projection struct {
	metadata evidence.Repository
	store    storage.Store
	now      func() time.Time
}

type Request struct {
	ExportID          string
	Exporter          exportmanifest.ExporterIdentity
	ApprovalReference string
}

func New(metadata evidence.Repository, store storage.Store, now func() time.Time) (*Projection, error) {
	if metadata == nil {
		return nil, ErrNilRepository
	}
	if store == nil {
		return nil, ErrNilStore
	}
	if now == nil {
		now = time.Now
	}
	return &Projection{metadata: metadata, store: store, now: now}, nil
}

func (p *Projection) Export(ctx context.Context, actor evidence.ActorContext, request Request) (exportmanifest.Manifest, error) {
	if err := actor.Validate(); err != nil {
		return exportmanifest.Manifest{}, err
	}
	if strings.TrimSpace(request.ExportID) == "" || strings.TrimSpace(request.Exporter.SubjectReference) == "" || strings.TrimSpace(request.Exporter.MembershipReference) == "" || strings.TrimSpace(request.ApprovalReference) == "" {
		return exportmanifest.Manifest{}, errors.New("export id, server-derived exporter identity, and approval reference are required")
	}
	records, err := p.metadata.ListByTenantOrganization(ctx, actor)
	if err != nil {
		return exportmanifest.Manifest{}, err
	}
	if len(records) == 0 {
		return exportmanifest.Manifest{}, ErrNoEvidence
	}

	manifest := exportmanifest.Manifest{
		ManifestVersion:  exportmanifest.Version,
		ExportID:         request.ExportID,
		TenantID:         actor.TenantID,
		OrganizationID:   actor.OrganizationID,
		CreatedAt:        p.now().UTC(),
		Exporter:         request.Exporter,
		ProcedureVersion: ProcedureVersion,
		Approval:         exportmanifest.ApprovalReference{ExportApprovalReference: request.ApprovalReference},
		Recovery:         exportmanifest.RecoveryMetadata{RestoreVerificationResult: "verified-live-object-read"},
	}
	for _, metadata := range records {
		if err := verifyAndAppend(ctx, p.store, actor, &manifest, metadata); err != nil {
			return exportmanifest.Manifest{}, err
		}
	}
	manifest.ObjectCount = uint64(len(manifest.Evidence))
	if err := manifest.Seal(); err != nil {
		return exportmanifest.Manifest{}, err
	}
	if err := manifest.Verify(); err != nil {
		return exportmanifest.Manifest{}, err
	}
	return manifest, nil
}

func verifyAndAppend(ctx context.Context, store storage.Store, actor evidence.ActorContext, manifest *exportmanifest.Manifest, metadata evidence.Metadata) error {
	if metadata.TenantID != actor.TenantID || metadata.OrganizationID != actor.OrganizationID {
		return fmt.Errorf("%w: evidence %s", ErrScopeMismatch, metadata.ID)
	}
	object, err := store.Get(ctx, metadata.ObjectKey)
	if err != nil {
		return fmt.Errorf("%w: evidence %s", ErrMissingObject, metadata.ID)
	}
	if object.Key != metadata.ObjectKey {
		return fmt.Errorf("%w: evidence %s", ErrObjectKeyMismatch, metadata.ID)
	}
	if object.ContentType != metadata.ContentType {
		return fmt.Errorf("%w: evidence %s", ErrContentTypeMismatch, metadata.ID)
	}
	if int64(len(object.Data)) != metadata.CiphertextBytes {
		return fmt.Errorf("%w: evidence %s", ErrByteCountMismatch, metadata.ID)
	}
	digest := sha256.Sum256(object.Data)
	if hex.EncodeToString(digest[:]) != metadata.CiphertextSHA256 {
		return fmt.Errorf("%w: evidence %s", ErrDigestMismatch, metadata.ID)
	}
	privacy := exportmanifest.PrivacyMetadata{
		Classification:     metadata.Classification,
		RetentionReference: metadata.RetentionReference,
		HoldState:          metadata.HoldState,
		RedactionPolicyRef: metadata.RedactionPolicyRef,
	}
	if len(manifest.Evidence) == 0 {
		manifest.Privacy = privacy
	} else if manifest.Privacy != privacy {
		return fmt.Errorf("%w: evidence %s", ErrMixedPrivacy, metadata.ID)
	}
	manifest.Evidence = append(manifest.Evidence, exportmanifest.EvidenceRecord{
		EvidenceID:          metadata.ID,
		ObjectKey:           metadata.ObjectKey,
		ContentType:         metadata.ContentType,
		CapturedAt:          metadata.CapturedAt,
		InspectionReference: metadata.InspectionID,
		EntityReference:     metadata.InspectionID,
		PlaintextSHA256:     metadata.PlaintextSHA256,
		CiphertextSHA256:    metadata.CiphertextSHA256,
		CiphertextBytes:     uint64CiphertextBytes(metadata.CiphertextBytes),
		EncryptionAlgorithm: metadata.EncryptionAlgorithm,
		EncryptionKeyRef:    metadata.EncryptionKeyRef,
		DeviceID:            metadata.DeviceID,
		AuthorityID:         metadata.AuthorityID,
		AuthorityEpoch:      metadata.AuthorityEpoch,
		TransactionID:       metadata.TransactionID,
		ReceiptID:           metadata.ReceiptID,
		SignatureAlgorithm:  metadata.SignatureAlgorithm,
		KeyID:               metadata.KeyID,
	})
	return nil
}

func uint64CiphertextBytes(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}
