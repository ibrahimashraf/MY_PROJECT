// Package packagemanifest issues server-signed, non-mutating work-package manifests.
// It remains transport-agnostic so a route cannot be mounted accidentally.
package packagemanifest

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	domainsync "integin/internal/domain/sync"
	"integin/internal/domain/workpackage"
	"integin/internal/workpackagepg"
)

const (
	// ManifestProtocolVersion identifies this signed manifest grammar.
	ManifestProtocolVersion = "work-package-manifest/v1"
	// ManifestSignatureAlgorithm is the only signature scheme emitted by this issuer.
	ManifestSignatureAlgorithm = "Ed25519"
)

var (
	// ErrNotFound deliberately conceals whether assignment or package lookup failed.
	ErrNotFound = errors.New("approved work package assignment not found")
	// ErrExpired means a server-held assignment or authority has already elapsed.
	ErrExpired = errors.New("approved work package assignment is expired")
	// ErrScopeMismatch means the verified device context and persisted assignment disagree.
	ErrScopeMismatch = errors.New("approved work package assignment scope mismatch")
	// ErrIntegrity means persisted assignment and package data cannot form a safe manifest.
	ErrIntegrity = errors.New("approved work package assignment integrity failure")
)

// Repository is the narrow persistence seam required to resolve a manifest.
// *workpackagepg.Repository satisfies it in production composition.
type Repository interface {
	GetCurrentAssignment(context.Context, string, string, string, string, time.Time) (workpackagepg.Assignment, error)
	GetApproved(context.Context, string, string, string, int) (workpackage.Package, error)
}

// PackageManifest is an immutable, server-signed description of one device assignment.
// The recipient must verify Signature before treating its package binding as current.
type PackageManifest struct {
	ManifestVersion    string                        `json:"manifest_version"`
	TenantID           string                        `json:"tenant_id"`
	OrganizationID     string                        `json:"organization_id"`
	InspectionID       string                        `json:"inspection_id"`
	DeviceID           string                        `json:"device_id"`
	PackageID          string                        `json:"package_id"`
	PackageVersion     int                           `json:"package_version"`
	PackageHash        string                        `json:"package_hash"`
	Package            workpackage.Package           `json:"package"`
	AssignmentContext  workpackage.AssignmentContext `json:"assignment_context"`
	SchemaVersion      int                           `json:"schema_version"`
	AuthorityEpoch     uint64                        `json:"authority_epoch"`
	IssuedAt           time.Time                     `json:"issued_at"`
	ExpiresAt          time.Time                     `json:"expires_at"`
	SignatureAlgorithm string                        `json:"signature_algorithm"`
	KeyID              string                        `json:"key_id"`
	Signature          string                        `json:"signature"`
}

// ManifestIssuer resolves the persisted assignment after device scope has been verified,
// then signs the resulting manifest. It contains no HTTP, replay, or live-runtime wiring.
type ManifestIssuer struct {
	repository      Repository
	contextResolver AssignmentContextResolver
	privateKey      ed25519.PrivateKey
	keyID           string
	maxLifetime     time.Duration
}

// NewManifestIssuer constructs an issuer with a bounded manifest lifetime.
// The caller must supply a deployment-owned signing key only when a future route is composed.
func NewManifestIssuer(repository Repository, contextResolver AssignmentContextResolver, privateKey ed25519.PrivateKey, keyID string, maxLifetime time.Duration) (*ManifestIssuer, error) {
	if repository == nil {
		return nil, errors.New("manifest repository is required")
	}
	if contextResolver == nil {
		return nil, errors.New("manifest assignment context resolver is required")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("manifest signing key is invalid")
	}
	if strings.TrimSpace(keyID) == "" {
		return nil, errors.New("manifest signing key identifier is required")
	}
	if maxLifetime <= 0 {
		return nil, errors.New("manifest maximum lifetime must be positive")
	}
	return &ManifestIssuer{
		repository:      repository,
		contextResolver: contextResolver,
		privateKey:      append(ed25519.PrivateKey(nil), privateKey...),
		keyID:           keyID,
		maxLifetime:     maxLifetime,
	}, nil
}

// Issue resolves and signs the current approved package for an already verified device scope.
func (i *ManifestIssuer) Issue(ctx context.Context, verified domainsync.VerifiedDeviceContext, inspectionID string, at time.Time) (PackageManifest, error) {
	if i == nil || i.repository == nil || i.contextResolver == nil {
		return PackageManifest{}, errors.New("manifest issuer is not configured")
	}
	if err := validateVerifiedContext(verified, inspectionID, at); err != nil {
		return PackageManifest{}, err
	}

	now := at.UTC()
	assignment, err := i.repository.GetCurrentAssignment(ctx, verified.TenantID, verified.OrganizationID, inspectionID, verified.DeviceID, now)
	if err != nil {
		return PackageManifest{}, mapRepositoryError(err)
	}
	if err := validateAssignment(assignment, verified, inspectionID, now); err != nil {
		return PackageManifest{}, err
	}

	pkg, err := i.repository.GetApproved(ctx, verified.TenantID, verified.OrganizationID, assignment.PackageID, assignment.PackageVersion)
	if err != nil {
		return PackageManifest{}, mapRepositoryError(err)
	}
	if err := validatePackage(pkg, assignment); err != nil {
		return PackageManifest{}, err
	}

	expiresAt := earliest(assignment.ExpiresAt.UTC(), verified.AuthorityExpiry.UTC(), now.Add(i.maxLifetime).UTC())
	if !expiresAt.After(now) {
		return PackageManifest{}, ErrExpired
	}
	assignmentContext, err := i.contextResolver.GetAssignmentContext(ctx, verified.TenantID, verified.OrganizationID, inspectionID, verified.DeviceID, at)
	if err != nil {
		return PackageManifest{}, fmt.Errorf("assignment context: %w", err)
	}
	if err := assignmentContext.Validate(); err != nil {
		return PackageManifest{}, fmt.Errorf("assignment context: %w", err)
	}
	manifest := PackageManifest{
		ManifestVersion:    ManifestProtocolVersion,
		TenantID:           verified.TenantID,
		OrganizationID:     verified.OrganizationID,
		InspectionID:       inspectionID,
		DeviceID:           verified.DeviceID,
		PackageID:          pkg.ID,
		PackageVersion:     pkg.PackageVersion,
		PackageHash:        pkg.PackageHash,
		AssignmentContext:  assignmentContext,
		SchemaVersion:      pkg.SchemaVersion,
		AuthorityEpoch:     verified.AuthorityEpoch,
		IssuedAt:           now,
		ExpiresAt:          expiresAt,
		SignatureAlgorithm: ManifestSignatureAlgorithm,
		KeyID:              i.keyID,
	}
	manifest.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(i.privateKey, []byte(canonicalManifest(manifest))))
	return manifest, nil
}

func validateVerifiedContext(verified domainsync.VerifiedDeviceContext, inspectionID string, at time.Time) error {
	if strings.TrimSpace(verified.TenantID) == "" || strings.TrimSpace(verified.OrganizationID) == "" || strings.TrimSpace(verified.DeviceID) == "" || strings.TrimSpace(inspectionID) == "" {
		return ErrScopeMismatch
	}
	if verified.AuthorityEpoch == 0 || verified.AuthorityExpiry.IsZero() || !verified.AuthorityExpiry.After(at) {
		return ErrExpired
	}
	return nil
}

func validateAssignment(assignment workpackagepg.Assignment, verified domainsync.VerifiedDeviceContext, inspectionID string, at time.Time) error {
	if assignment.TenantID != verified.TenantID || assignment.OrganizationID != verified.OrganizationID || assignment.DeviceID != verified.DeviceID || assignment.InspectionID != inspectionID {
		return ErrScopeMismatch
	}
	if assignment.AuthorityEpoch <= 0 || uint64(assignment.AuthorityEpoch) != verified.AuthorityEpoch {
		return ErrScopeMismatch
	}
	if assignment.ExpiresAt.IsZero() || !assignment.ExpiresAt.After(at) {
		return ErrExpired
	}
	if strings.TrimSpace(assignment.PackageID) == "" || assignment.PackageVersion <= 0 {
		return ErrIntegrity
	}
	return nil
}

func validatePackage(pkg workpackage.Package, assignment workpackagepg.Assignment) error {
	if pkg.TenantID != assignment.TenantID || pkg.OrganizationID != assignment.OrganizationID || pkg.ID != assignment.PackageID || pkg.PackageVersion != assignment.PackageVersion {
		return ErrScopeMismatch
	}
	if pkg.State != workpackage.PublicationApproved || strings.TrimSpace(pkg.PackageHash) == "" || pkg.SchemaVersion <= 0 {
		return ErrIntegrity
	}
	return nil
}

func mapRepositoryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("manifest repository: %w", err)
}

func earliest(values ...time.Time) time.Time {
	result := values[0]
	for _, value := range values[1:] {
		if value.Before(result) {
			result = value
		}
	}
	return result
}

func canonicalManifest(manifest PackageManifest) string {
	return strings.Join([]string{
		manifest.ManifestVersion,
		manifest.TenantID,
		manifest.OrganizationID,
		manifest.InspectionID,
		manifest.DeviceID,
		manifest.PackageID,
		fmt.Sprintf("%d", manifest.PackageVersion),
		manifest.PackageHash,
		canonicalAssignmentContext(manifest.AssignmentContext),
		fmt.Sprintf("%d", manifest.SchemaVersion),
		fmt.Sprintf("%d", manifest.AuthorityEpoch),
		manifest.IssuedAt.UTC().Format(time.RFC3339Nano),
		manifest.ExpiresAt.UTC().Format(time.RFC3339Nano),
		manifest.SignatureAlgorithm,
		manifest.KeyID,
	}, "|")
}

func canonicalAssignmentContext(context workpackage.AssignmentContext) string {
	pairs := make([]string, 0, len(context.FieldAssetIDs))
	for fieldID, assetID := range context.FieldAssetIDs {
		pairs = append(pairs, fieldID+"="+assetID)
	}
	sort.Strings(pairs)
	return strings.Join([]string{
		context.RootAssetID,
		context.InspectionType,
		context.ProcedureVersion,
		context.ScheduledAt.UTC().Format(time.RFC3339Nano),
		strings.Join(pairs, ","),
	}, "|")
}
