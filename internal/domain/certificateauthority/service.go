package certificateauthority

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DraftRepository interface {
	CreateDraft(context.Context, ActorContext, CreateDraftRequest, time.Time) (*Certificate, error)
}

type CreateDraftRequest struct {
	CertificateID   string
	InspectionID    string
	TemplateCode    string
	TemplateVersion int64
	Profile         Profile
	SelfIssueReason string
}

type Service struct {
	repository DraftRepository
	now        func() time.Time
}

func NewService(repository DraftRepository, now func() time.Time) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("certificate authority repository is required")
	}
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, now: now}, nil
}

func (s *Service) CreateDraft(ctx context.Context, actor ActorContext, request CreateDraftRequest) (*Certificate, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.CertificateID) == "" || strings.TrimSpace(request.InspectionID) == "" || strings.TrimSpace(request.TemplateCode) == "" || request.TemplateVersion <= 0 {
		return nil, fmt.Errorf("certificate draft request is incomplete")
	}
	return s.repository.CreateDraft(ctx, actor, request, s.now().UTC())
}
