package certificate

import (
	"testing"
	"time"

	"integin/internal/shared/events"
	"integin/internal/shared/types"
)

func newTestCertificate(t *testing.T) Certificate {
	t.Helper()
	result, err := New("certificate-1", "tenant-1", "org-1", types.EnvironmentLive, "CERT-001", "inspection-1", 1, "asset-1", "inspector-1", "creator-1", time.Date(2027, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestCertificateFullLifecycle(t *testing.T) {
	certificate := newTestCertificate(t)
	if err := certificate.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Approve("approver-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Sign("signer-1"); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	if err := certificate.Issue("issuer-1", issuedAt); err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Issued || certificate.IssuedAt().Location() != time.UTC {
		t.Fatalf("unexpected issued certificate: %s %s", certificate.Status(), certificate.IssuedAt().Location())
	}
	if len(certificate.Events()) != 4 {
		t.Fatalf("expected four events, got %d", len(certificate.Events()))
	}
	for _, event := range certificate.Events() {
		if event.TenantID() != "tenant-1" {
			t.Fatal("certificate event lost tenant id")
		}
	}
	if certificate.Events()[3].EventType() != events.CertificateIssued {
		t.Fatal("expected issued event")
	}
}

func TestCertificateEnforcesSeparationOfDuties(t *testing.T) {
	certificate := newTestCertificate(t)
	if err := certificate.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Approve("inspector-1", nil); err == nil {
		t.Fatal("inspector approval should be rejected")
	}
	if certificate.Status() != PendingApproval {
		t.Fatal("failed approval must not mutate status")
	}
	exception := &SeparationException{GrantedBy: "system-admin-1", Reason: "documented emergency exception"}
	if err := certificate.Approve("inspector-1", exception); err != nil {
		t.Fatal(err)
	}
	if certificate.Exception() == nil {
		t.Fatal("exception should be recorded")
	}
}

func TestCertificateRejectsInspectorSigningAndIssuing(t *testing.T) {
	certificate := newTestCertificate(t)
	if err := certificate.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Approve("approver-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Sign("inspector-1"); err == nil {
		t.Fatal("inspector signing should be rejected")
	}
	if certificate.Status() != Approved {
		t.Fatal("failed signing must not mutate status")
	}
	if err := certificate.Sign("signer-1"); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Issue("inspector-1", time.Now()); err == nil {
		t.Fatal("inspector issuing should be rejected")
	}
}

func TestCertificateRevocationExpiryAndSupersession(t *testing.T) {
	revoked := newTestCertificate(t)
	if err := revoked.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := revoked.Approve("approver-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := revoked.Sign("signer-1"); err != nil {
		t.Fatal(err)
	}
	if err := revoked.Issue("issuer-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := revoked.Revoke("authority-1", "unsafe asset condition"); err != nil {
		t.Fatal(err)
	}
	if revoked.Status() != Revoked || revoked.RevocationReason() == "" {
		t.Fatal("revocation not recorded")
	}

	expired := newTestCertificate(t)
	if err := expired.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := expired.Approve("approver-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := expired.Sign("signer-1"); err != nil {
		t.Fatal(err)
	}
	if err := expired.Issue("issuer-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := expired.Expire(time.Date(2027, 8, 13, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if expired.Status() != Expired {
		t.Fatal("expiry not recorded")
	}

	superseded := newTestCertificate(t)
	if err := superseded.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := superseded.Approve("approver-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := superseded.Sign("signer-1"); err != nil {
		t.Fatal(err)
	}
	if err := superseded.Issue("issuer-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := superseded.Supersede("authority-1", "certificate-2"); err != nil {
		t.Fatal(err)
	}
	if superseded.Status() != Superseded || superseded.SupersededByID() != "certificate-2" {
		t.Fatal("supersession not recorded")
	}
}

func TestCertificateInvalidTransitionsDoNotMutate(t *testing.T) {
	certificate := newTestCertificate(t)
	if err := certificate.Issue("issuer-1", time.Now()); err == nil {
		t.Fatal("draft should not issue directly")
	}
	if certificate.Status() != Draft {
		t.Fatal("invalid transition mutated status")
	}
}
