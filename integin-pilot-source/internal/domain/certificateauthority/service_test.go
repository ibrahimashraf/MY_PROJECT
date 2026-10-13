package certificateauthority

import (
	"context"
	"testing"
	"time"
)

type draftRepositoryStub struct {
	inspection    CanonicalInspection
	policy        Policy
	stored        *Certificate
	receivedActor ActorContext
	calls         int
}

func (s *draftRepositoryStub) CreateDraft(_ context.Context, actor ActorContext, request CreateDraftRequest, now time.Time) (*Certificate, error) {
	s.calls++
	s.receivedActor = actor
	certificate, err := NewDraft(actor, s.inspection, s.policy, DraftRequest{ID: request.CertificateID, TemplateCode: request.TemplateCode, TemplateVersion: request.TemplateVersion, Profile: request.Profile, SelfIssueReason: request.SelfIssueReason}, now)
	if err != nil {
		return nil, err
	}
	s.stored = certificate
	return certificate, nil
}

func TestServiceCreatesDraftThroughAtomicRepositoryBoundary(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	repository := &draftRepositoryStub{inspection: inspection(), policy: policy(false)}
	service, err := NewService(repository, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	inspector := actor("inspector-a", "certificate.prepare")
	certificate, err := service.CreateDraft(context.Background(), inspector, CreateDraftRequest{CertificateID: "certificate-a", InspectionID: "inspection-a", TemplateCode: "lifting", TemplateVersion: 3, Profile: IndependentReview})
	if err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Draft || repository.stored != certificate {
		t.Fatalf("draft not stored: %#v", certificate)
	}
	if repository.calls != 1 || repository.receivedActor.TenantID != "tenant-a" || repository.receivedActor.OrganizationID != "org-a" {
		t.Fatal("atomic repository did not receive derived actor scope")
	}
}

func TestServiceRejectsIncompleteRequestBeforeRepositoryAccess(t *testing.T) {
	repository := &draftRepositoryStub{inspection: inspection(), policy: policy(false)}
	service, err := NewService(repository, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateDraft(context.Background(), actor("inspector-a", "certificate.prepare"), CreateDraftRequest{CertificateID: "certificate-a", InspectionID: "inspection-a", TemplateCode: "lifting", Profile: IndependentReview})
	if err == nil {
		t.Fatal("expected incomplete request rejection")
	}
	if repository.calls != 0 {
		t.Fatal("repository should not be reached for incomplete request")
	}
}
