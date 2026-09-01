package certificatepg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/certificateauthority"
)

// QuickSetValidity maps the UX quick-set (Bswagic 120s 3/6/12 months) to policy validity_days.
// No new GRANT; just a helper so callers don't hard-code days. 3m=90d, 6m=180d, 12m=365d.
func QuickSetValidityDays(months int) (int64, error) {
	switch months {
	case 3:
		return 90, nil
	case 6:
		return 180, nil
	case 12:
		return 365, nil
	default:
		return 0, fmt.Errorf("quick-set supports only 3, 6, 12 months")
	}
}

// ExpiredBucket mirrors Bswagic 390s buckets: Expired vs Expire in 1m/2w/10d.
type ExpiredBucket struct {
	ID        string
	AssetID   string
	Status    string
	ExpiresAt time.Time
	Bucket    string // "expired" | "1m" | "2w" | "10d" | "future"
}

// ListExpired returns ISSUED→EXPIRED that are already expired (expires_at <= now).
// Uses existing certificate_record_scope_status_idx; RLS via set_config; no new GRANT.
func (r *Repository) ListExpired(ctx context.Context, actor certificateauthority.ActorContext, now time.Time) ([]ExpiredBucket, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return r.listByExpiryWindow(ctx, actor, now, func() (string, []any) {
		return `status = 'EXPIRED'`, nil
	})
}

// ListExpiring returns ISSUED certs whose expires_at is in (now, now+window].
// Window helpers match Bswagic buckets: 10d, 2w (14d), 1m (30d).
func (r *Repository) ListExpiring(ctx context.Context, actor certificateauthority.ActorContext, now time.Time, window time.Duration) ([]ExpiredBucket, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	if window <= 0 {
		return nil, fmt.Errorf("window is required")
	}
	return r.listByExpiryWindow(ctx, actor, now, func() (string, []any) {
		return `status = 'ISSUED' AND expires_at > $3 AND expires_at <= $4`, []any{now.UTC(), now.UTC().Add(window)}
	})
}

func (r *Repository) listByExpiryWindow(ctx context.Context, actor certificateauthority.ActorContext, now time.Time, where func() (string, []any)) ([]ExpiredBucket, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := setScope(ctx, tx, actor); err != nil {
		return nil, err
	}
	clause, extra := where()
	args := []any{actor.TenantID, actor.OrganizationID}
	query := fmt.Sprintf(`SELECT id, asset_id, status, expires_at FROM certificate_record WHERE tenant_id=$1 AND organization_id=$2 AND %s ORDER BY expires_at DESC, id`, clause) //nolint:G201 // clause is internal whitelist
	if len(extra) > 0 {
		// extra already contains now/window; shift placeholders
		// caller used $3/$4, so append after $1/$2
		args = append(args, extra...)
	}
	// normalize placeholder numbering when extra is nil (EXPIRED case)
	if len(extra) == 0 {
		query = strings.ReplaceAll(query, "$3", "$1")
		query = strings.ReplaceAll(query, "$4", "$2")
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExpiredBucket
	for rows.Next() {
		var b ExpiredBucket
		if err := rows.Scan(&b.ID, &b.AssetID, &b.Status, &b.ExpiresAt); err != nil {
			return nil, err
		}
		switch {
		case !b.ExpiresAt.IsZero() && !b.ExpiresAt.After(now.UTC()):
			b.Bucket = "expired"
		case b.ExpiresAt.After(now.UTC()) && !b.ExpiresAt.After(now.UTC().Add(10*24*time.Hour)):
			b.Bucket = "10d"
		case b.ExpiresAt.After(now.UTC().Add(10*24*time.Hour)) && !b.ExpiresAt.After(now.UTC().Add(14*24*time.Hour)):
			b.Bucket = "2w"
		case b.ExpiresAt.After(now.UTC().Add(14*24*time.Hour)) && !b.ExpiresAt.After(now.UTC().Add(30*24*time.Hour)):
			b.Bucket = "1m"
		default:
			b.Bucket = "future"
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
