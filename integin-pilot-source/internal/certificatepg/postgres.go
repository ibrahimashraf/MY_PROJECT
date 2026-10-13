package certificatepg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/certificateauthority"
	"integin/internal/timestamp"
)

type Repository struct {
	db  *sql.DB
	tsa *timestamp.TSA
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("certificate lifecycle database is required")
	}
	return &Repository{db: db}, nil
}

// SetTSAClient attaches a provisioned RFC 3161 authority. When set, every
// certificate issuance is notarized with a trusted time stamp and the
// issuance transaction fails closed if the authority cannot attest. A nil
// TSA keeps legacy issuance behavior.
func (r *Repository) SetTSAClient(tsa *timestamp.TSA) {
	r.tsa = tsa
}

func (r *Repository) CreateDraft(ctx context.Context, actor certificateauthority.ActorContext, request certificateauthority.CreateDraftRequest, now time.Time) (*certificateauthority.Certificate, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, actor.TenantID, actor.OrganizationID); err != nil {
		return nil, err
	}
	inspection, err := loadInspection(ctx, tx, actor, request.InspectionID)
	if err != nil {
		return nil, err
	}
	policy, err := loadPolicy(ctx, tx, actor, request.TemplateCode, request.TemplateVersion)
	if err != nil {
		return nil, err
	}
	certificate, err := certificateauthority.NewDraft(actor, inspection, policy, certificateauthority.DraftRequest{ID: request.CertificateID, TemplateCode: request.TemplateCode, TemplateVersion: request.TemplateVersion, Profile: request.Profile, SelfIssueReason: request.SelfIssueReason}, now)
	if err != nil {
		return nil, err
	}
	if err := insertDraft(ctx, tx, certificate); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, certificate, actor.ActorID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return certificate, nil
}

func loadInspection(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, inspectionID string) (certificateauthority.CanonicalInspection, error) {
	var value certificateauthority.CanonicalInspection
	err := tx.QueryRowContext(ctx, `SELECT id, tenant_id, organization_id, asset_id, inspector_id, revision, lifecycle_state, finalization_state FROM inspection_record WHERE id = $1 AND tenant_id = $2 AND organization_id = $3 FOR UPDATE`, inspectionID, actor.TenantID, actor.OrganizationID).Scan(&value.ID, &value.TenantID, &value.OrganizationID, &value.AssetID, &value.InspectorID, &value.Revision, &value.LifecycleState, &value.FinalizationState)
	if errors.Is(err, sql.ErrNoRows) {
		return certificateauthority.CanonicalInspection{}, fmt.Errorf("canonical inspection is not available")
	}
	return value, err
}

func loadPolicy(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, templateCode string, templateVersion int64) (certificateauthority.Policy, error) {
	var value certificateauthority.Policy
	err := tx.QueryRowContext(ctx, `SELECT p.id, p.template_code, p.template_version, p.policy_version, p.validity_days, p.self_issue_allowed, p.status FROM certificate_policy p JOIN certificate_template t ON t.tenant_id = p.tenant_id AND t.organization_id = p.organization_id AND t.template_code = p.template_code AND t.version = p.template_version WHERE p.tenant_id = $1 AND p.organization_id = $2 AND p.template_code = $3 AND p.template_version = $4 AND p.status = 'APPROVED' AND t.status = 'APPROVED' FOR SHARE`, actor.TenantID, actor.OrganizationID, templateCode, templateVersion).Scan(&value.ID, &value.TemplateCode, &value.TemplateVersion, &value.Version, &value.ValidityDays, &value.SelfIssueAllowed, &value.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return certificateauthority.Policy{}, fmt.Errorf("approved certificate policy is not available")
	}
	return value, err
}

func insertDraft(ctx context.Context, tx *sql.Tx, certificate *certificateauthority.Certificate) error {
	var reason any
	var evidence any
	if certificate.Profile() == certificateauthority.SeniorSelfIssue {
		reason = certificate.SelfIssueReason()
		evidence = "server-derived-policy:" + certificate.PolicyID()
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO certificate_record (id, tenant_id, organization_id, inspection_id, inspection_revision, asset_id, template_code, template_version, policy_id, policy_version, profile, status, created_by, created_at, self_issue_reason, self_issue_policy_evidence) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, certificate.ID(), certificate.TenantID(), certificate.OrganizationID(), certificate.InspectionID(), certificate.InspectionRevision(), certificate.AssetID(), certificate.TemplateCode(), certificate.TemplateVersion(), certificate.PolicyID(), certificate.PolicyVersion(), string(certificate.Profile()), string(certificate.Status()), certificate.CreatedBy(), certificate.CreatedAt(), reason, evidence)
	return err
}

func insertAudit(ctx context.Context, tx *sql.Tx, certificate *certificateauthority.Certificate, actorID string) error {
	evidence, err := json.Marshal(map[string]any{"policy_id": certificate.PolicyID(), "policy_version": certificate.PolicyVersion(), "profile": certificate.Profile()})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO certificate_audit_event (id, tenant_id, organization_id, certificate_id, action, actor_id, occurred_at, policy_evidence) VALUES ($1,$2,$3,$4,'DRAFT_CREATED',$5,$6,$7::jsonb)`, certificate.ID()+":draft_created", certificate.TenantID(), certificate.OrganizationID(), certificate.ID(), strings.TrimSpace(actorID), certificate.CreatedAt(), string(evidence))
	return err
}

func (r *Repository) RecordRenewalAttempt(ctx context.Context, attempt certificateauthority.RenewalAttempt) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, attempt.TenantID, attempt.OrganizationID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO certificate_renewal_attempt (id, tenant_id, organization_id, certificate_id, attempted_at, authority_tenant_id, authority_organization_id, previous_status, new_status, renewal_reason, authority_snapshot, result_state, error_detail)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		attempt.ID, attempt.TenantID, attempt.OrganizationID, attempt.CertificateID, attempt.AttemptedAt, attempt.AuthorityTenantID, attempt.AuthorityOrganizationID, attempt.PreviousStatus, attempt.NewStatus, attempt.RenewalReason, attempt.AuthoritySnapshot, attempt.ResultState, attempt.ErrorDetail)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
