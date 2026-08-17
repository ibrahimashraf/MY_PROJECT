package packagemanifest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	domainsync "integin/internal/domain/sync"
	"integin/internal/domain/workpackage"
	"integin/internal/workpackagepg"
)

func TestManifestIssuerRejectsMissingAssignment(t *testing.T) {
	now := time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
	issuer := newTestIssuer(t, manifestStore{assignmentErr: sql.ErrNoRows})

	_, err := issuer.Issue(context.Background(), verifiedContext(now), "inspection-1", now)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestManifestIssuerRejectsExpiredAssignment(t *testing.T) {
	now := time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
	issuer := newTestIssuer(t, manifestStore{assignment: validAssignment(now.Add(-time.Second))})

	_, err := issuer.Issue(context.Background(), verifiedContext(now), "inspection-1", now)
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestManifestIssuerReturnsSignedBoundManifest(t *testing.T) {
	now := time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
	pkg := validPackage()
	store := manifestStore{assignment: validAssignment(now.Add(2 * time.Hour)), pkg: pkg}
	privateKey, issuer := newTestIssuerWithKey(t, store)

	manifest, err := issuer.Issue(context.Background(), verifiedContext(now), "inspection-1", now)
	if err != nil {
		t.Fatalf("issue manifest: %v", err)
	}
	if manifest.ManifestVersion != ManifestProtocolVersion || manifest.PackageID != pkg.ID || manifest.PackageVersion != pkg.PackageVersion || manifest.PackageHash != pkg.PackageHash || manifest.SchemaVersion != pkg.SchemaVersion {
		t.Fatalf("unexpected manifest package binding: %#v", manifest)
	}
	if manifest.AuthorityEpoch != 7 || !manifest.ExpiresAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("unexpected manifest scope or expiry: %#v", manifest)
	}
	signature, err := base64.RawStdEncoding.DecodeString(manifest.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	if !ed25519.Verify(privateKey.Public().(ed25519.PublicKey), []byte(canonicalManifest(manifest)), signature) {
		t.Fatal("manifest signature did not verify")
	}
}

type manifestStore struct {
	assignment    workpackagepg.Assignment
	assignmentErr error
	pkg           workpackage.Package
	packageErr    error
}

func (s manifestStore) GetCurrentAssignment(context.Context, string, string, string, string, time.Time) (workpackagepg.Assignment, error) {
	return s.assignment, s.assignmentErr
}

func (s manifestStore) GetApproved(context.Context, string, string, string, int) (workpackage.Package, error) {
	return s.pkg, s.packageErr
}

func newTestIssuer(t *testing.T, store manifestStore) *ManifestIssuer {
	t.Helper()
	_, issuer := newTestIssuerWithKey(t, store)
	return issuer
}

func newTestIssuerWithKey(t *testing.T, store manifestStore) (ed25519.PrivateKey, *ManifestIssuer) {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	issuer, err := NewManifestIssuer(store, privateKey, "manifest-key-1", 30*time.Minute)
	if err != nil {
		t.Fatalf("new issuer: %v", err)
	}
	return privateKey, issuer
}

func verifiedContext(now time.Time) domainsync.VerifiedDeviceContext {
	return domainsync.VerifiedDeviceContext{
		TenantID:        "tenant-1",
		OrganizationID:  "organization-1",
		DeviceID:        "device-1",
		AuthorityID:     "authority-1",
		AuthorityEpoch:  7,
		ProofExpiresAt:  now.Add(5 * time.Minute),
		AuthorityExpiry: now.Add(90 * time.Minute),
	}
}

func validAssignment(expiresAt time.Time) workpackagepg.Assignment {
	return workpackagepg.Assignment{
		TenantID:       "tenant-1",
		OrganizationID: "organization-1",
		InspectionID:   "inspection-1",
		DeviceID:       "device-1",
		PackageID:      "package-1",
		PackageVersion: 2,
		AuthorityEpoch: 7,
		AssignedAt:     expiresAt.Add(-time.Hour),
		ExpiresAt:      expiresAt,
	}
}

func validPackage() workpackage.Package {
	return workpackage.Package{
		ID:              "package-1",
		TenantID:        "tenant-1",
		OrganizationID:  "organization-1",
		TemplateCode:    "crane-inspection",
		TemplateVersion: 1,
		PackageVersion:  2,
		SchemaVersion:   1,
		State:           workpackage.PublicationApproved,
		PackageHash:     "sha256:manifest-test-package-hash",
	}
}
