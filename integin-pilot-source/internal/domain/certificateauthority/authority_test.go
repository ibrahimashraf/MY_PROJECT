package certificateauthority

import (
	"testing"
	"time"
)

func actor(id string, capabilities ...string) ActorContext {
	values := make(map[string]bool, len(capabilities))
	for _, capability := range capabilities {
		values[capability] = true
	}
	return ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: id, Capabilities: values}
}

func inspection() CanonicalInspection {
	return CanonicalInspection{ID: "inspection-a", TenantID: "tenant-a", OrganizationID: "org-a", AssetID: "asset-a", InspectorID: "inspector-a", Revision: 7, LifecycleState: "APPROVED", FinalizationState: "FINALIZED"}
}

func policy(selfIssue bool) Policy {
	return Policy{ID: "policy-a", TemplateCode: "lifting", TemplateVersion: 3, Version: 1, Status: "APPROVED", ValidityDays: 365, SelfIssueAllowed: selfIssue}
}

func TestIndependentReviewRequiresDistinctAuthorities(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	inspector := actor("inspector-a", "certificate.prepare")
	certificate, err := NewDraft(inspector, inspection(), policy(false), DraftRequest{ID: "certificate-a", TemplateCode: "lifting", TemplateVersion: 3, Profile: IndependentReview}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := certificate.Submit(inspector); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Review(actor("inspector-a", "certificate.review"), now); err == nil {
		t.Fatal("expected inspector review denial")
	}
	reviewer := actor("reviewer-a", "certificate.review")
	if err := certificate.Review(reviewer, now); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Sign(actor("reviewer-a", "certificate.sign"), now); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Issue(actor("issuer-a", "certificate.issue"), policy(false), now); err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Issued || certificate.IssuedBy() != "issuer-a" || certificate.ExpiresAt().IsZero() {
		t.Fatalf("unexpected issue result: %#v", certificate)
	}
}

func TestSeniorSelfIssueRequiresPolicyCapabilityAndReason(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	senior := actor("inspector-a", "certificate.prepare", "certificate.review", "certificate.sign", "certificate.issue", "certificate.self_issue")
	request := DraftRequest{ID: "certificate-a", TemplateCode: "lifting", TemplateVersion: 3, Profile: SeniorSelfIssue, SelfIssueReason: "remote emergency assignment"}
	certificate, err := NewDraft(senior, inspection(), policy(true), request, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := certificate.Submit(senior); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Review(senior, now); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Sign(senior, now); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Issue(senior, policy(true), now); err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Issued || certificate.SelfIssueReason() == "" {
		t.Fatalf("unexpected senior self issue result: %#v", certificate)
	}
}

func TestCertificateDraftRejectsIneligibleInspectionAndSelfIssueBypass(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	senior := actor("inspector-a", "certificate.prepare", "certificate.self_issue")
	rejected := inspection()
	rejected.LifecycleState = "REJECTED"
	if _, err := NewDraft(senior, rejected, policy(true), DraftRequest{ID: "certificate-a", TemplateCode: "lifting", TemplateVersion: 3, Profile: SeniorSelfIssue, SelfIssueReason: "reason"}, now); err == nil {
		t.Fatal("expected rejected inspection denial")
	}
	if _, err := NewDraft(senior, inspection(), policy(false), DraftRequest{ID: "certificate-a", TemplateCode: "lifting", TemplateVersion: 3, Profile: SeniorSelfIssue, SelfIssueReason: "reason"}, now); err == nil {
		t.Fatal("expected policy denial")
	}
	if _, err := NewDraft(senior, inspection(), policy(true), DraftRequest{ID: "certificate-a", TemplateCode: "lifting", TemplateVersion: 3, Profile: SeniorSelfIssue}, now); err == nil {
		t.Fatal("expected missing reason denial")
	}
}

func TestCertificateCorrectionStatesStayControlled(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	inspector := actor("inspector-a", "certificate.prepare")
	certificate, err := NewDraft(inspector, inspection(), policy(false), DraftRequest{ID: "certificate-a", TemplateCode: "lifting", TemplateVersion: 3, Profile: IndependentReview}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := certificate.Revoke(actor("authority-a", "certificate.revoke"), "reason", now); err == nil {
		t.Fatal("expected draft revoke denial")
	}
	if err := certificate.Submit(inspector); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Review(actor("reviewer-a", "certificate.review"), now); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Sign(actor("signer-a", "certificate.sign"), now); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Issue(actor("issuer-a", "certificate.issue"), policy(false), now); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Revoke(actor("authority-a", "certificate.revoke"), "unsafe condition", now); err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Revoked {
		t.Fatalf("unexpected revoke status: %s", certificate.Status())
	}
}
