package certificatepg

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/certificateauthority"
)

func (r *Repository) Renew(ctx context.Context, actor certificateauthority.ActorContext, certificateID, reason string, now time.Time) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Capabilities == nil || !actor.Capabilities["certificate.renew"] {
		return fmt.Errorf("actor lacks certificate.renew capability")
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return fmt.Errorf("renewal reason is required")
	}
	token, err := newRenewalToken()
	if err != nil {
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
	var profile string
	row := tx.QueryRowContext(ctx, `UPDATE certificate_record SET renewal_count = renewal_count + 1, last_renewal_attempt = $1, renewal_authority_token = $2 WHERE id = $3 AND tenant_id = $4 AND organization_id = $5 AND status = 'ISSUED' RETURNING profile`, now.UTC(), token, certificateID, actor.TenantID, actor.OrganizationID)
	if err := row.Scan(&profile); err == sql.ErrNoRows {
		return fmt.Errorf("certificate is not available for renewal")
	} else if err != nil {
		return err
	}
	if err := insertRenewalAttempt(ctx, tx, actor, certificateID, profile, reason, now); err != nil {
		return err
	}
	if err := insertLifecycleAudit(ctx, tx, actor, certificateID, "RENEWED", now, map[string]any{"reason": strings.TrimSpace(reason)}); err != nil {
		return err
	}
	return tx.Commit()
}

func newRenewalToken() ([]byte, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	return token, nil
}

func insertRenewalAttempt(ctx context.Context, tx *sql.Tx, actor certificateauthority.ActorContext, certificateID, profile, reason string, now time.Time) error {
	snapshot, err := json.Marshal(map[string]any{"profile": profile})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO certificate_renewal_attempt (id, tenant_id, organization_id, certificate_id, attempted_at, authority_tenant_id, authority_organization_id, previous_status, new_status, renewal_reason, authority_snapshot, result_state) VALUES ($1,$2,$3,$4,$5,$6,$7,'ISSUED','ISSUED',$8,$9::jsonb,'SUCCESS')`, certificateID+":renew:"+now.UTC().Format("20060102150405.000000000"), actor.TenantID, actor.OrganizationID, certificateID, now.UTC(), actor.TenantID, actor.OrganizationID, strings.TrimSpace(reason), string(snapshot))
	return err
}
