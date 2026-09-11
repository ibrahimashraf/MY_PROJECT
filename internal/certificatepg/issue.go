package certificatepg

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"integin/internal/domain/certificateauthority"
)

type issueState struct {
	templateSnapshot      []byte
	cellSnapshot          []byte
	publicBindingSnapshot []byte
	validityDays          int
	profile               string
	inspectorID           string
}

func (r *Repository) Issue(ctx context.Context, actor certificateauthority.ActorContext, certificateID string, now time.Time) (IssueResult, error) {
	if err := validateActor(actor); err != nil {
		return IssueResult{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return IssueResult{}, err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return IssueResult{}, err
	}
	state, err := loadIssueState(ctx, tx, actor, certificateID)
	if err != nil {
		return IssueResult{}, err
	}
	if state.validityDays <= 0 || state.validityDays > 3650 {
		return IssueResult{}, fmt.Errorf("certificate validity policy is invalid")
	}
	if err := authorizeTransition(actor, state.profile, state.inspectorID, "certificate.issue"); err != nil {
		return IssueResult{}, err
	}
	number, err := allocateNumber(ctx, tx, actor, now)
	if err != nil {
		return IssueResult{}, err
	}
	token, digest, err := newPublicToken()
	if err != nil {
		return IssueResult{}, err
	}
	expiresAt := now.UTC().AddDate(0, 0, state.validityDays)
	if err := persistIssued(ctx, tx, actor, certificateID, number, digest, expiresAt, now); err != nil {
		return IssueResult{}, err
	}
	if err := persistSnapshot(ctx, tx, actor, certificateID, state.templateSnapshot, state.cellSnapshot, state.publicBindingSnapshot); err != nil {
		return IssueResult{}, err
	}
	evidence := map[string]any{"certificate_number": number}
	if r.tsa != nil {
		stamp, err := r.tsa.Timestamp(ctx, snapshotDigest(state.templateSnapshot, state.cellSnapshot, state.publicBindingSnapshot), now)
		if err != nil {
			return IssueResult{}, fmt.Errorf("trusted timestamp failed: %w", err)
		}
		evidence["timestamp_token_der"] = base64.StdEncoding.EncodeToString(stamp.Response)
		evidence["timestamp_gen_time"] = stamp.GenTime.UTC().Format(time.RFC3339)
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "ISSUED", now, evidence); err != nil {
		return IssueResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return IssueResult{}, err
	}
	return IssueResult{CertificateNumber: number, PublicToken: token, ExpiresAt: expiresAt}, nil
}

func newPublicToken() (string, []byte, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes)
	digest := sha256.Sum256([]byte(raw))
	return raw, digest[:], nil
}

func loadIssueState(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, certificateID string) (issueState, error) {
	var state issueState
	var policyID, assetID, inspectionID string
	var inspectionRevision int64
	query := `SELECT to_jsonb(t), jsonb_build_object('inspection.id',i.id,'inspection.asset_id',i.asset_id,'inspection.inspector_id',i.inspector_id,'inspection.lifecycle_state',i.lifecycle_state,'inspection.revision',i.revision,'inspection.finalization_state',i.finalization_state), p.validity_days, c.profile, i.inspector_id, c.policy_id, i.asset_id, i.id, i.revision FROM certificate_record c JOIN certificate_policy p ON p.id=c.policy_id AND p.tenant_id=c.tenant_id AND p.organization_id=c.organization_id AND p.policy_version=c.policy_version JOIN certificate_template t ON t.tenant_id=c.tenant_id AND t.organization_id=c.organization_id AND t.template_code=c.template_code AND t.version=c.template_version JOIN inspection_record i ON i.id=c.inspection_id AND i.tenant_id=c.tenant_id AND i.organization_id=c.organization_id WHERE c.id=$1 AND c.tenant_id=$2 AND c.organization_id=$3 AND c.status='SIGNED' AND p.status='APPROVED' AND t.status='APPROVED' FOR UPDATE`
	err := tx.QueryRowContext(ctx, query, certificateID, actor.TenantID, actor.OrganizationID).Scan(&state.templateSnapshot, &state.cellSnapshot, &state.validityDays, &state.profile, &state.inspectorID, &policyID, &assetID, &inspectionID, &inspectionRevision)
	if err == sql.ErrNoRows {
		return issueState{}, fmt.Errorf("certificate is not available for issuance")
	}
	if err != nil {
		return state, err
	}
	state.publicBindingSnapshot, err = materializePublicBindingSnapshot(ctx, tx, actor.TenantID, actor.OrganizationID, policyID, assetID, inspectionID, inspectionRevision)
	return state, err
}

func allocateNumber(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, now time.Time) (string, error) {
	var value int64
	query := `INSERT INTO certificate_number_sequence (tenant_id,organization_id,next_value,updated_at) VALUES ($1,$2,2,$3) ON CONFLICT (tenant_id,organization_id) DO UPDATE SET next_value=certificate_number_sequence.next_value+1,updated_at=EXCLUDED.updated_at RETURNING next_value-1`
	if err := tx.QueryRowContext(ctx, query, actor.TenantID, actor.OrganizationID, now.UTC()).Scan(&value); err != nil {
		return "", err
	}
	return fmt.Sprintf("CERT-%08d", value), nil
}

func persistIssued(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, certificateID, number string, digest []byte, expiresAt, now time.Time) error {
	query := `UPDATE certificate_record SET status='ISSUED',certificate_number=$1,issued_by=$2,issued_at=$3,expires_at=$4,public_token_digest=$5 WHERE id=$6 AND tenant_id=$7 AND organization_id=$8 AND status='SIGNED'`
	result, err := tx.ExecContext(ctx, query, number, actor.ActorID, now.UTC(), expiresAt, digest, certificateID, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate issuance conflict")
	}
	return nil
}

func persistSnapshot(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, certificateID string, templateSnapshot, cellSnapshot, publicBindingSnapshot []byte) error {
	digest := snapshotDigest(templateSnapshot, cellSnapshot, publicBindingSnapshot)
	query := `INSERT INTO certificate_snapshot (certificate_id,tenant_id,organization_id,template_snapshot,cell_snapshot,public_binding_snapshot,snapshot_sha256) VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb,$7)`
	_, err := tx.ExecContext(ctx, query, certificateID, actor.TenantID, actor.OrganizationID, string(templateSnapshot), string(cellSnapshot), string(publicBindingSnapshot), digest[:])
	return err
}

// snapshotDigest is the SHA-256 of the concatenated issue-time snapshots. It
// is the exact byte sequence persisted as certificate_snapshot.snapshot_sha256
// and, when a trusted timestamp authority is provisioned, the TST message
// imprint — making the stored digest independently notarizable.
func snapshotDigest(templateSnapshot, cellSnapshot, publicBindingSnapshot []byte) []byte {
	sum := sha256.Sum256(append(append(append([]byte{}, templateSnapshot...), cellSnapshot...), publicBindingSnapshot...))
	return sum[:]
}
