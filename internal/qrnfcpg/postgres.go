package qrnfcpg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"integin/internal/domain/qrnfc"
)

var (
	ErrNilDB    = errors.New("QR/NFC entry postgres repository requires a database")
	ErrNotFound = errors.New("QR/NFC entry not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Issue(ctx context.Context, actor qrnfc.ActorContext, entry qrnfc.Entry) (qrnfc.Entry, string, error) {
	if err := actor.Validate(); err != nil {
		return qrnfc.Entry{}, "", err
	}
	entry.TenantID = actor.TenantID
	entry.OrganizationID = actor.OrganizationID
	entry.Status = qrnfc.EntryStatusActive
	entry.CreatedBy = actor.ActorID
	entry.CreatedAt = time.Now().UTC()
	if entry.IssuedAt.IsZero() {
		entry.IssuedAt = entry.CreatedAt
	}
	if entry.TokenVersion <= 0 {
		entry.TokenVersion = 1
	}
	if err := entry.Validate(); err != nil {
		return qrnfc.Entry{}, "", err
	}
	rawToken, digest, err := qrnfc.GenerateToken()
	if err != nil {
		return qrnfc.Entry{}, "", err
	}
	entry.TokenDigest = digest
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return qrnfc.Entry{}, "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO qr_nfc_entry (id, tenant_id, organization_id, entitlement_id, asset_id, entry_type, token_digest, token_version, status, issued_at, expires_at, created_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		entry.ID, actor.TenantID, actor.OrganizationID, entry.EntitlementID, entry.AssetID,
		string(entry.EntryType), entry.TokenDigest, entry.TokenVersion, string(entry.Status),
		entry.IssuedAt, entry.ExpiresAt, entry.CreatedBy, entry.CreatedAt); err != nil {
		return qrnfc.Entry{}, "", err
	}
	stored, err := loadEntry(ctx, tx, actor, entry.ID)
	if err != nil {
		return qrnfc.Entry{}, "", err
	}
	return stored, rawToken, tx.Commit()
}

func (r *Repository) Get(ctx context.Context, actor qrnfc.ActorContext, entryID string) (qrnfc.Entry, error) {
	if err := actor.Validate(); err != nil {
		return qrnfc.Entry{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return qrnfc.Entry{}, err
	}
	defer tx.Rollback()
	stored, err := loadEntry(ctx, tx, actor, entryID)
	if err != nil {
		return qrnfc.Entry{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) VerifyByDigest(ctx context.Context, actor qrnfc.ActorContext, tokenDigest string) (qrnfc.Entry, error) {
	if err := actor.Validate(); err != nil {
		return qrnfc.Entry{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return qrnfc.Entry{}, err
	}
	defer tx.Rollback()
	var entry qrnfc.Entry
	err = tx.QueryRowContext(ctx,
		`SELECT id, entitlement_id, asset_id, entry_type, token_digest, token_version, status, issued_at, expires_at, revoked_at, COALESCE(revoked_by,''), created_by, created_at
		 FROM qr_nfc_entry WHERE tenant_id=$1 AND organization_id=$2 AND token_digest=$3 AND status='ACTIVE'`,
		actor.TenantID, actor.OrganizationID, tokenDigest).Scan(
		&entry.ID, &entry.EntitlementID, &entry.AssetID, &entry.EntryType, &entry.TokenDigest,
		&entry.TokenVersion, &entry.Status, &entry.IssuedAt, &entry.ExpiresAt,
		&entry.RevokedAt, &entry.RevokedBy, &entry.CreatedBy, &entry.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return qrnfc.Entry{}, qrnfc.ErrNotFound
	}
	if err != nil {
		return qrnfc.Entry{}, err
	}
	entry.TenantID = actor.TenantID
	entry.OrganizationID = actor.OrganizationID
	if entry.IsExpired(time.Now().UTC()) {
		return entry, qrnfc.ErrExpired
	}
	return entry, tx.Commit()
}

func (r *Repository) Revoke(ctx context.Context, actor qrnfc.ActorContext, entryID string) (qrnfc.Entry, error) {
	if err := actor.Validate(); err != nil {
		return qrnfc.Entry{}, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return qrnfc.Entry{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx,
		`UPDATE qr_nfc_entry SET status = 'REVOKED', revoked_at = $1, revoked_by = $2
		 WHERE tenant_id = $3 AND organization_id = $4 AND id = $5 AND status = 'ACTIVE'`,
		now, actor.ActorID, actor.TenantID, actor.OrganizationID, entryID)
	if err != nil {
		return qrnfc.Entry{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return qrnfc.Entry{}, err
	}
	if count == 0 {
		entry, err := loadEntry(ctx, tx, actor, entryID)
		if errors.Is(err, ErrNotFound) {
			return qrnfc.Entry{}, qrnfc.ErrNotFound
		}
		if err != nil {
			return qrnfc.Entry{}, err
		}
		return qrnfc.Entry{}, fmt.Errorf("entry is not ACTIVE (current: %s)", entry.Status)
	}
	stored, err := loadEntry(ctx, tx, actor, entryID)
	if err != nil {
		return qrnfc.Entry{}, err
	}
	return stored, tx.Commit()
}

func (r *Repository) ListByAsset(ctx context.Context, actor qrnfc.ActorContext, assetID string) ([]qrnfc.Entry, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx,
		`SELECT id, entitlement_id, asset_id, entry_type, token_digest, token_version, status, issued_at, expires_at, revoked_at, COALESCE(revoked_by,''), created_by, created_at
		 FROM qr_nfc_entry WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3
		 ORDER BY issued_at DESC`,
		actor.TenantID, actor.OrganizationID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []qrnfc.Entry
	for rows.Next() {
		var entry qrnfc.Entry
		if err := rows.Scan(
			&entry.ID, &entry.EntitlementID, &entry.AssetID, &entry.EntryType, &entry.TokenDigest,
			&entry.TokenVersion, &entry.Status, &entry.IssuedAt, &entry.ExpiresAt,
			&entry.RevokedAt, &entry.RevokedBy, &entry.CreatedBy, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entry.TenantID = actor.TenantID
		entry.OrganizationID = actor.OrganizationID
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, tx.Commit()
}

func (r *Repository) LogAccess(ctx context.Context, actor qrnfc.ActorContext, log qrnfc.EntryLog) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	log.TenantID = actor.TenantID
	log.OrganizationID = actor.OrganizationID
	if log.AccessedAt.IsZero() {
		log.AccessedAt = time.Now().UTC()
	}
	if err := log.Validate(); err != nil {
		return err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO qr_nfc_entry_log (id, tenant_id, organization_id, entry_id, entry_type, accessed_at, accessor_id, accessor_ip, outcome)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9)`,
		log.ID, actor.TenantID, actor.OrganizationID, log.EntryID, string(log.EntryType),
		log.AccessedAt, log.AccessorID, log.AccessorIP, string(log.Outcome)); err != nil {
		return err
	}
	return tx.Commit()
}

func loadEntry(ctx context.Context, tx *sql.Tx, actor qrnfc.ActorContext, entryID string) (qrnfc.Entry, error) {
	var entry qrnfc.Entry
	err := tx.QueryRowContext(ctx,
		`SELECT id, entitlement_id, asset_id, entry_type, token_digest, token_version, status, issued_at, expires_at, revoked_at, COALESCE(revoked_by,''), created_by, created_at
		 FROM qr_nfc_entry WHERE tenant_id=$1 AND organization_id=$2 AND id=$3`,
		actor.TenantID, actor.OrganizationID, entryID).Scan(
		&entry.ID, &entry.EntitlementID, &entry.AssetID, &entry.EntryType, &entry.TokenDigest,
		&entry.TokenVersion, &entry.Status, &entry.IssuedAt, &entry.ExpiresAt,
		&entry.RevokedAt, &entry.RevokedBy, &entry.CreatedBy, &entry.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return qrnfc.Entry{}, ErrNotFound
	}
	if err != nil {
		return qrnfc.Entry{}, err
	}
	entry.TenantID = actor.TenantID
	entry.OrganizationID = actor.OrganizationID
	return entry, nil
}

func (r *Repository) begin(ctx context.Context, actor qrnfc.ActorContext) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id',$1,true), set_config('integin.organization_id',$2,true)`, actor.TenantID, actor.OrganizationID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}
