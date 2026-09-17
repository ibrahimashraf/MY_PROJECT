package certificatepg

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/certificate"
	"integin/internal/domain/certificateauthority"
)

type IssueResult struct {
	CertificateNumber string
	PublicToken       string
	ExpiresAt         time.Time
}

func (r *Repository) Review(ctx context.Context, actor certificateauthority.ActorContext, certificateID string, now time.Time) error {
	return r.transition(ctx, actor, certificateID, "PENDING_REVIEW", "APPROVED", "certificate.review", "REVIEWED", now)
}

func (r *Repository) Sign(ctx context.Context, actor certificateauthority.ActorContext, certificateID string, event certificate.SignatureEvent, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return err
	}
	if event.SignerID != actor.ActorID {
		return fmt.Errorf("signature signer must match authenticated actor")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	profile, inspectorID, err := loadAuthority(ctx, tx, actor, certificateID, "APPROVED")
	if err != nil {
		return err
	}
	if err := authorizeTransition(actor, profile, inspectorID, "certificate.sign"); err != nil {
		return err
	}
	if err := verifySignatureEvidence(ctx, tx, actor, certificateID, event); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE certificate_record SET status = $1, signed_by = $2, signed_at = $3 WHERE id = $4 AND tenant_id = $5 AND organization_id = $6 AND status = $7`, "SIGNED", actor.ActorID, now.UTC(), certificateID, actor.TenantID, actor.OrganizationID, "APPROVED")
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate transition conflict")
	}
	evidence := map[string]any{
		"profile":           profile,
		"signer_name":       strings.TrimSpace(event.SignerName),
		"capacity":          event.Capacity,
		"statement_version": strings.TrimSpace(event.StatementVersion),
		"image_sha256_hex":  event.ImageSHA256Hex,
		"image_bytes":       event.ImageBytes,
		"image_evidence_id": strings.TrimSpace(event.ImageEvidenceID),
		"snapshot_sha256":   event.SnapshotSHA256,
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "SIGNED", now, evidence); err != nil {
		return err
	}
	return tx.Commit()
}

// verifySignatureEvidence binds the signature to a registered evidence blob:
// the row must exist in tenant scope, its plaintext digest must match the
// submitted image digest, it must be an image, and it must belong to the same
// inspection as the certificate being signed.
func verifySignatureEvidence(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, certificateID string, event certificate.SignatureEvent) error {
	var certInspectionID string
	err := tx.QueryRowContext(ctx, `SELECT inspection_id FROM certificate_record WHERE id = $1 AND tenant_id = $2 AND organization_id = $3`, certificateID, actor.TenantID, actor.OrganizationID).Scan(&certInspectionID)
	if err != nil {
		return err
	}
	var plaintextSHA, contentType, evidenceInspectionID string
	err = tx.QueryRowContext(ctx, `SELECT plaintext_sha256, content_type, inspection_id FROM evidence_metadata WHERE id = $1 AND tenant_id = $2 AND organization_id = $3`, strings.TrimSpace(event.ImageEvidenceID), actor.TenantID, actor.OrganizationID).Scan(&plaintextSHA, &contentType, &evidenceInspectionID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("signature image evidence is not registered")
	}
	if err != nil {
		return err
	}
	if plaintextSHA != event.ImageSHA256Hex {
		return fmt.Errorf("signature image digest does not match registered evidence")
	}
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("signature evidence is not an image")
	}
	if evidenceInspectionID != certInspectionID {
		return fmt.Errorf("signature evidence belongs to a different inspection")
	}
	return nil
}

// WaiveSign records a waived sign-off: no ink was captured, and the waiver
// authority plus reason stand in for the signature. The granter must be the
// authenticated actor performing the waiver.
func (r *Repository) WaiveSign(ctx context.Context, actor certificateauthority.ActorContext, certificateID, grantedBy, reason, capacity, authorizedBy string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Capabilities == nil || !actor.Capabilities["certificate.sign"] {
		return fmt.Errorf("actor lacks certificate.sign capability")
	}
	if strings.TrimSpace(grantedBy) == "" || grantedBy != actor.ActorID {
		return fmt.Errorf("waiver granter must match authenticated actor")
	}
	if strings.TrimSpace(reason) == "" || len(reason) > certificate.MaxTextFieldChars {
		return fmt.Errorf("waiver reason is required")
	}
	switch capacity {
	case certificate.CapacityClient, certificate.CapacityVerifier:
	default:
		return fmt.Errorf("waiver capacity must be client or verifier")
	}
	if len(authorizedBy) > certificate.MaxTextFieldChars {
		return fmt.Errorf("waiver authorizer is invalid")
	}
	if strings.TrimSpace(authorizedBy) != "" && strings.TrimSpace(authorizedBy) == grantedBy {
		return fmt.Errorf("waiver authorizer must differ from granter")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	profile, inspectorID, err := loadAuthority(ctx, tx, actor, certificateID, "APPROVED")
	if err != nil {
		return err
	}
	if err := authorizeTransition(actor, profile, inspectorID, "certificate.sign"); err != nil {
		return err
	}
	if actor.ActorID == inspectorID && strings.TrimSpace(authorizedBy) == "" {
		return fmt.Errorf("inspector waiver requires an office authorizer")
	}
	result, err := tx.ExecContext(ctx, `UPDATE certificate_record SET status = $1, signed_by = $2, signed_at = $3 WHERE id = $4 AND tenant_id = $5 AND organization_id = $6 AND status = $7`, "SIGNED", actor.ActorID, now.UTC(), certificateID, actor.TenantID, actor.OrganizationID, "APPROVED")
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate transition conflict")
	}
	evidence := map[string]any{
		"profile":       profile,
		"waived":        true,
		"granted_by":    actor.ActorID,
		"reason":        strings.TrimSpace(reason),
		"capacity":      capacity,
		"authorized_by": strings.TrimSpace(authorizedBy),
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "SIGN_WAIVED", now, evidence); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) Attest(ctx context.Context, actor certificateauthority.ActorContext, certificateID, attestorName, qualificationBasis, statementVersion, snapshotSHA256Hex string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Capabilities == nil || !actor.Capabilities["certificate.attest"] {
		return fmt.Errorf("actor lacks certificate.attest capability")
	}
	if strings.TrimSpace(attestorName) == "" || len(attestorName) > certificate.MaxTextFieldChars {
		return fmt.Errorf("attestor name is required")
	}
	if strings.TrimSpace(qualificationBasis) == "" || len(qualificationBasis) > certificate.MaxTextFieldChars {
		return fmt.Errorf("qualification basis is required")
	}
	if strings.TrimSpace(statementVersion) == "" || len(statementVersion) > certificate.MaxTextFieldChars {
		return fmt.Errorf("statement version is required")
	}
	if _, err := hex.DecodeString(snapshotSHA256Hex); err != nil || len(snapshotSHA256Hex) != 64 {
		return fmt.Errorf("snapshot digest must be 64 hex characters")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	var inspectorID, status string
	err = tx.QueryRowContext(ctx, `SELECT i.inspector_id, c.status FROM certificate_record c JOIN inspection_record i ON i.id = c.inspection_id AND i.tenant_id = c.tenant_id AND i.organization_id = c.organization_id WHERE c.id = $1 AND c.tenant_id = $2 AND c.organization_id = $3`, certificateID, actor.TenantID, actor.OrganizationID).Scan(&inspectorID, &status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("certificate is not available for attestation")
	}
	if err != nil {
		return err
	}
	if status != "DRAFT" && status != "PENDING_REVIEW" && status != "APPROVED" {
		return fmt.Errorf("cannot attest certificate in status %s", status)
	}
	if actor.ActorID != inspectorID {
		return fmt.Errorf("only the assigned inspector can attest findings")
	}
	evidence := map[string]any{
		"attestor_name":       strings.TrimSpace(attestorName),
		"qualification_basis": strings.TrimSpace(qualificationBasis),
		"statement_version":   strings.TrimSpace(statementVersion),
		"snapshot_sha256":     snapshotSHA256Hex,
		"capacity":            certificate.CapacityInspector,
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "ATTESTED", now, evidence); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) transition(ctx context.Context, actor certificateauthority.ActorContext, certificateID, expected, next, capability, action string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	profile, inspectorID, err := loadAuthority(ctx, tx, actor, certificateID, expected)
	if err != nil {
		return err
	}
	if err := authorizeTransition(actor, profile, inspectorID, capability); err != nil {
		return err
	}
	// Column prefix is derived from a fixed internal whitelist; never from user input.
	// transition currently serves Review only.
	if next != "APPROVED" {
		return fmt.Errorf("unsupported transition target %q", next)
	}
	field := "reviewed"
	query := fmt.Sprintf(`UPDATE certificate_record SET status = $1, %s_by = $2, %s_at = $3 WHERE id = $4 AND tenant_id = $5 AND organization_id = $6 AND status = $7`, field, field) //nolint:gosec // field is a fixed internal whitelist (reviewed|signed)
	result, err := tx.ExecContext(ctx, query, next, actor.ActorID, now.UTC(), certificateID, actor.TenantID, actor.OrganizationID, expected)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate transition conflict")
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, action, now, map[string]any{"profile": profile}); err != nil {
		return err
	}
	return tx.Commit()
}

func validateActor(actor certificateauthority.ActorContext) error {
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.OrganizationID) == "" || strings.TrimSpace(actor.ActorID) == "" {
		return fmt.Errorf("certificate actor context is incomplete")
	}
	return nil
}

func setScope(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext) error {
	_, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, actor.TenantID, actor.OrganizationID)
	return err
}

func loadAuthority(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, certificateID, expected string) (string, string, error) {
	var profile, inspectorID string
	err := tx.QueryRowContext(ctx, `SELECT c.profile, i.inspector_id FROM certificate_record c JOIN inspection_record i ON i.id = c.inspection_id AND i.tenant_id = c.tenant_id AND i.organization_id = c.organization_id WHERE c.id = $1 AND c.tenant_id = $2 AND c.organization_id = $3 AND c.status = $4 FOR UPDATE`, certificateID, actor.TenantID, actor.OrganizationID, expected).Scan(&profile, &inspectorID)
	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("certificate is not available for transition")
	}
	return profile, inspectorID, err
}

func authorizeTransition(actor certificateauthority.ActorContext, profile, inspectorID, capability string) error {
	if actor.Capabilities == nil || !actor.Capabilities[capability] {
		return fmt.Errorf("actor lacks %s capability", capability)
	}
	switch profile {
	case string(certificateauthority.IndependentReview):
		if actor.ActorID == inspectorID {
			return fmt.Errorf("independent review forbids inspector authority")
		}
	case string(certificateauthority.SeniorSelfIssue):
		if actor.ActorID != inspectorID || !actor.Capabilities["certificate.self_issue"] {
			return fmt.Errorf("senior self issue authority is not satisfied")
		}
	default:
		return fmt.Errorf("certificate profile is invalid")
	}
	return nil
}

func insertLifecycleAudit(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, certificateID, action string, now time.Time, evidence map[string]any) error {
	bytes, err := json.Marshal(evidence)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO certificate_audit_event (id, tenant_id, organization_id, certificate_id, action, actor_id, occurred_at, policy_evidence) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb)`, certificateID+":"+strings.ToLower(action)+":"+now.UTC().Format("20060102150405.000000000"), actor.TenantID, actor.OrganizationID, certificateID, action, actor.ActorID, now.UTC(), string(bytes))
	return err
}

func (r *Repository) Submit(ctx context.Context, actor certificateauthority.ActorContext, certificateID string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Capabilities == nil || !actor.Capabilities["certificate.prepare"] {
		return fmt.Errorf("actor lacks certificate.prepare capability")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE certificate_record SET status='PENDING_REVIEW' WHERE id=$1 AND tenant_id=$2 AND organization_id=$3 AND created_by=$4 AND status='DRAFT'`, certificateID, actor.TenantID, actor.OrganizationID, actor.ActorID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate is not available for submission")
	}
	return tx.Commit()
}

func (r *Repository) Revoke(ctx context.Context, actor certificateauthority.ActorContext, certificateID, reason string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Capabilities == nil || !actor.Capabilities["certificate.revoke"] {
		return fmt.Errorf("actor lacks certificate.revoke capability")
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return fmt.Errorf("revocation reason is required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE certificate_record SET status='REVOKED',revoked_by=$1,revoked_at=$2,revocation_reason=$3 WHERE id=$4 AND tenant_id=$5 AND organization_id=$6 AND status IN ('ISSUED','EXPIRED')`, actor.ActorID, now.UTC(), strings.TrimSpace(reason), certificateID, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate is not available for revocation")
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "REVOKED", now, map[string]any{"reason": strings.TrimSpace(reason)}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) Supersede(ctx context.Context, actor certificateauthority.ActorContext, certificateID, replacementID string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Capabilities == nil || !actor.Capabilities["certificate.supersede"] {
		return fmt.Errorf("actor lacks certificate.supersede capability")
	}
	if strings.TrimSpace(replacementID) == "" || replacementID == certificateID {
		return fmt.Errorf("replacement certificate id is invalid")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE certificate_record SET status='SUPERSEDED',superseded_by_id=$1 WHERE id=$2 AND tenant_id=$3 AND organization_id=$4 AND status IN ('ISSUED','EXPIRED')`, replacementID, certificateID, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate is not available for supersession")
	}
	result, err = tx.ExecContext(ctx, `UPDATE certificate_record replacement SET supersedes_id=$1 FROM certificate_record original WHERE replacement.id=$2 AND original.id=$1 AND replacement.tenant_id=$3 AND replacement.organization_id=$4 AND original.tenant_id=$3 AND original.organization_id=$4 AND replacement.asset_id=original.asset_id AND replacement.status='ISSUED' AND replacement.supersedes_id IS NULL`, certificateID, replacementID, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return err
	}
	changed, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("replacement certificate is not available")
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "SUPERSEDED", now, map[string]any{"replacement_id": replacementID}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) Expire(ctx context.Context, actor certificateauthority.ActorContext, certificateID string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Capabilities == nil || !actor.Capabilities["certificate.expire"] {
		return fmt.Errorf("actor lacks certificate.expire capability")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE certificate_record SET status='EXPIRED' WHERE id=$1 AND tenant_id=$2 AND organization_id=$3 AND status='ISSUED' AND expires_at <= $4`, certificateID, actor.TenantID, actor.OrganizationID, now.UTC())
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("certificate is not available for expiry")
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "EXPIRED", now, map[string]any{}); err != nil {
		return err
	}
	return tx.Commit()
}
