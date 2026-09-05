package syncstate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrConflict = errors.New("sync-state record conflicts with existing data")

type PostgresRepository struct {
	DB *sql.DB
}

func NewPostgresRepository(db *sql.DB) (*PostgresRepository, error) {
	if db == nil {
		return nil, errors.New("database handle is required")
	}
	return &PostgresRepository{DB: db}, nil
}

func (r *PostgresRepository) beginTenant(ctx context.Context, tenantID string) (*sql.Tx, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database handle is required")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true)`, tenantID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r *PostgresRepository) GetDevice(ctx context.Context, tenantID, deviceID string) (DeviceRecord, error) {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return DeviceRecord{}, err
	}
	defer tx.Rollback()
	var device DeviceRecord
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT device_id, tenant_id, organization_id, user_id, key_id, public_key, state, authority_epoch, enrolled_at, updated_at, revoked_at, COALESCE(revocation_reason, '') FROM device_registry WHERE tenant_id = $1 AND device_id = $2`, tenantID, deviceID).Scan(&device.DeviceID, &device.TenantID, &device.OrganizationID, &device.UserID, &device.KeyID, &device.PublicKey, &state, &device.AuthorityEpoch, &device.EnrolledAt, &device.UpdatedAt, &device.RevokedAt, &device.RevocationReason); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeviceRecord{}, ErrNotFound
		}
		return DeviceRecord{}, err
	}
	device.State = DeviceState(state)
	return device, tx.Commit()
}

func (r *PostgresRepository) ListDevices(ctx context.Context, tenantID string) ([]DeviceRecord, error) {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT device_id, tenant_id, organization_id, user_id, key_id, public_key, state, authority_epoch, enrolled_at, updated_at, revoked_at, COALESCE(revocation_reason, '') FROM device_registry WHERE tenant_id = $1 ORDER BY device_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	devices := make([]DeviceRecord, 0)
	for rows.Next() {
		var device DeviceRecord
		var state string
		if err := rows.Scan(&device.DeviceID, &device.TenantID, &device.OrganizationID, &device.UserID, &device.KeyID, &device.PublicKey, &state, &device.AuthorityEpoch, &device.EnrolledAt, &device.UpdatedAt, &device.RevokedAt, &device.RevocationReason); err != nil {
			return nil, err
		}
		device.State = DeviceState(state)
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *PostgresRepository) beginTenantOrg(ctx context.Context, tenantID, orgID string) (*sql.Tx, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database handle is required")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, tenantID, orgID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r *PostgresRepository) SaveDevice(ctx context.Context, device DeviceRecord) error {
	if err := ValidateDeviceRecord(device); err != nil {
		return err
	}
	tx, err := r.beginTenantOrg(ctx, device.TenantID, device.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO device_registry (device_id, tenant_id, organization_id, user_id, key_id, public_key, state, authority_epoch, enrolled_at, updated_at, revoked_at, revocation_reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')) ON CONFLICT (tenant_id, device_id) DO UPDATE SET organization_id = EXCLUDED.organization_id, user_id = EXCLUDED.user_id, key_id = EXCLUDED.key_id, public_key = EXCLUDED.public_key, state = EXCLUDED.state, authority_epoch = EXCLUDED.authority_epoch, updated_at = EXCLUDED.updated_at, revoked_at = EXCLUDED.revoked_at, revocation_reason = EXCLUDED.revocation_reason WHERE device_registry.tenant_id = EXCLUDED.tenant_id`, device.DeviceID, device.TenantID, device.OrganizationID, device.UserID, device.KeyID, device.PublicKey, string(device.State), device.AuthorityEpoch, device.EnrolledAt, device.UpdatedAt, device.RevokedAt, device.RevocationReason)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO sync_device_state (device_id, tenant_id, organization_id, updated_at) VALUES ($1,$2,$3,$4) ON CONFLICT (tenant_id, device_id) DO NOTHING`, device.DeviceID, device.TenantID, device.OrganizationID, device.UpdatedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostgresRepository) GetAuthority(ctx context.Context, tenantID, authorityID string) (AuthorityRecord, error) {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return AuthorityRecord{}, err
	}
	defer tx.Rollback()
	var authority AuthorityRecord
	var scopesJSON []byte
	if err := tx.QueryRowContext(ctx, `SELECT authority_id, tenant_id, organization_id, device_id, user_id, authority_epoch, scopes, procedure_version, issued_at, expires_at, signature, revoked_at FROM authority_package WHERE tenant_id = $1 AND authority_id = $2`, tenantID, authorityID).Scan(&authority.AuthorityID, &authority.TenantID, &authority.OrganizationID, &authority.DeviceID, &authority.UserID, &authority.AuthorityEpoch, &scopesJSON, &authority.ProcedureVersion, &authority.IssuedAt, &authority.ExpiresAt, &authority.Signature, &authority.RevokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthorityRecord{}, ErrNotFound
		}
		return AuthorityRecord{}, err
	}
	if err := json.Unmarshal(scopesJSON, &authority.Scopes); err != nil {
		return AuthorityRecord{}, fmt.Errorf("decode authority scopes: %w", err)
	}
	return authority, tx.Commit()
}

func (r *PostgresRepository) ListAuthorities(ctx context.Context, tenantID string) ([]AuthorityRecord, error) {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT authority_id, tenant_id, organization_id, device_id, user_id, authority_epoch, scopes, procedure_version, issued_at, expires_at, signature, revoked_at FROM authority_package WHERE tenant_id = $1 AND revoked_at IS NULL ORDER BY authority_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	authorities := make([]AuthorityRecord, 0)
	for rows.Next() {
		var authority AuthorityRecord
		var scopesJSON []byte
		if err := rows.Scan(&authority.AuthorityID, &authority.TenantID, &authority.OrganizationID, &authority.DeviceID, &authority.UserID, &authority.AuthorityEpoch, &scopesJSON, &authority.ProcedureVersion, &authority.IssuedAt, &authority.ExpiresAt, &authority.Signature, &authority.RevokedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(scopesJSON, &authority.Scopes); err != nil {
			return nil, fmt.Errorf("decode authority scopes: %w", err)
		}
		authorities = append(authorities, authority)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return authorities, nil
}

func (r *PostgresRepository) SaveAuthority(ctx context.Context, authority AuthorityRecord) error {
	if authority.AuthorityID == "" || authority.TenantID == "" || authority.OrganizationID == "" || authority.DeviceID == "" || authority.UserID == "" || authority.AuthorityEpoch == 0 || authority.ProcedureVersion == "" || authority.IssuedAt.IsZero() || authority.ExpiresAt.IsZero() || !authority.ExpiresAt.After(authority.IssuedAt) || len(authority.Signature) == 0 {
		return errors.New("authority identity, scope, validity, and signature are required")
	}
	scopes, err := json.Marshal(authority.Scopes)
	if err != nil {
		return err
	}
	tx, err := r.beginTenant(ctx, authority.TenantID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO authority_package (authority_id, tenant_id, organization_id, device_id, user_id, authority_epoch, scopes, procedure_version, issued_at, expires_at, signature, revoked_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (authority_id) DO UPDATE SET scopes = EXCLUDED.scopes, expires_at = EXCLUDED.expires_at, signature = EXCLUDED.signature, revoked_at = EXCLUDED.revoked_at WHERE authority_package.tenant_id = EXCLUDED.tenant_id`, authority.AuthorityID, authority.TenantID, authority.OrganizationID, authority.DeviceID, authority.UserID, authority.AuthorityEpoch, scopes, authority.ProcedureVersion, authority.IssuedAt, authority.ExpiresAt, authority.Signature, authority.RevokedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostgresRepository) RevokeAuthority(ctx context.Context, tenantID, authorityID string, at time.Time) error {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE authority_package SET revoked_at = $1 WHERE tenant_id = $2 AND authority_id = $3 AND revoked_at IS NULL`, at.UTC(), tenantID, authorityID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

func (r *PostgresRepository) GetLastAcceptedSequence(ctx context.Context, tenantID, deviceID string) (uint64, error) {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var sequence uint64
	if err := tx.QueryRowContext(ctx, `SELECT last_accepted_sequence FROM sync_device_state WHERE tenant_id = $1 AND device_id = $2`, tenantID, deviceID).Scan(&sequence); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return sequence, tx.Commit()
}

func (r *PostgresRepository) SaveReceipt(ctx context.Context, receipt Receipt) error {
	if err := ValidateReceipt(receipt); err != nil {
		return err
	}
	tx, err := r.beginTenant(ctx, receipt.TenantID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var existingHash, existingOutcome string
	err = tx.QueryRowContext(ctx, `SELECT payload_hash, outcome FROM sync_receipt WHERE tenant_id = $1 AND transaction_id = $2`, receipt.TenantID, receipt.TransactionID).Scan(&existingHash, &existingOutcome)
	switch {
	case err == nil:
		if existingHash != receipt.PayloadHash {
			return ErrConflict
		}
		if existingOutcome == receipt.Outcome {
			return tx.Commit()
		}
		if existingOutcome != "HELD" || receipt.Outcome != "APPLIED" {
			return ErrConflict
		}
		if _, err = tx.ExecContext(ctx, `UPDATE sync_receipt SET outcome = $1, reason = $2, received_at = $3 WHERE tenant_id = $4 AND transaction_id = $5 AND outcome = 'HELD'`, receipt.Outcome, receipt.Reason, receipt.ReceivedAt, receipt.TenantID, receipt.TransactionID); err != nil {
			return err
		}
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, `INSERT INTO sync_receipt (transaction_id, tenant_id, organization_id, device_id, user_id, sequence_number, operation, entity_id, payload_hash, outcome, reason, captured_at, received_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, receipt.TransactionID, receipt.TenantID, receipt.OrganizationID, receipt.DeviceID, receipt.UserID, receipt.SequenceNumber, receipt.Operation, receipt.EntityID, receipt.PayloadHash, receipt.Outcome, receipt.Reason, receipt.CapturedAt, receipt.ReceivedAt)
		if err != nil {
			return err
		}
	default:
		return err
	}
	if receipt.Outcome == "APPLIED" {
		result, err := tx.ExecContext(ctx, `UPDATE sync_device_state SET last_accepted_sequence = $1, updated_at = $2 WHERE tenant_id = $3 AND device_id = $4 AND last_accepted_sequence + 1 = $1`, receipt.SequenceNumber, receipt.ReceivedAt, receipt.TenantID, receipt.DeviceID)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrConflict
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM sync_held_transaction WHERE tenant_id = $1 AND transaction_id = $2`, receipt.TenantID, receipt.TransactionID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresRepository) GetReceipt(ctx context.Context, tenantID, transactionID string) (Receipt, error) {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return Receipt{}, err
	}
	defer tx.Rollback()
	var receipt Receipt
	if err := tx.QueryRowContext(ctx, `SELECT transaction_id, tenant_id, organization_id, device_id, user_id, sequence_number, operation, entity_id, payload_hash, outcome, reason, captured_at, received_at FROM sync_receipt WHERE tenant_id = $1 AND transaction_id = $2`, tenantID, transactionID).Scan(&receipt.TransactionID, &receipt.TenantID, &receipt.OrganizationID, &receipt.DeviceID, &receipt.UserID, &receipt.SequenceNumber, &receipt.Operation, &receipt.EntityID, &receipt.PayloadHash, &receipt.Outcome, &receipt.Reason, &receipt.CapturedAt, &receipt.ReceivedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Receipt{}, ErrNotFound
		}
		return Receipt{}, err
	}
	return receipt, tx.Commit()
}

func (r *PostgresRepository) SaveHeld(ctx context.Context, held HeldTransaction) error {
	if err := ValidateReceipt(held.Receipt); err != nil {
		return err
	}
	if len(held.Envelope) == 0 || held.ExpectedSequence == 0 || held.FirstHeldAt.IsZero() || held.LastAttemptAt.IsZero() || held.LastError == "" {
		return errors.New("held transaction envelope, sequence, timestamps, and error are required")
	}
	tx, err := r.beginTenant(ctx, held.Receipt.TenantID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var existingHash, existingOutcome string
	err = tx.QueryRowContext(ctx, `SELECT payload_hash, outcome FROM sync_receipt WHERE tenant_id = $1 AND transaction_id = $2`, held.Receipt.TenantID, held.Receipt.TransactionID).Scan(&existingHash, &existingOutcome)
	if err == nil {
		if existingHash != held.Receipt.PayloadHash || existingOutcome != "HELD" {
			return ErrConflict
		}
		if _, err = tx.ExecContext(ctx, `UPDATE sync_receipt SET reason = $1, received_at = $2 WHERE tenant_id = $3 AND transaction_id = $4`, held.Receipt.Reason, held.Receipt.ReceivedAt, held.Receipt.TenantID, held.Receipt.TransactionID); err != nil {
			return err
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if _, err = tx.ExecContext(ctx, `INSERT INTO sync_receipt (transaction_id, tenant_id, organization_id, device_id, user_id, sequence_number, operation, entity_id, payload_hash, outcome, reason, captured_at, received_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, held.Receipt.TransactionID, held.Receipt.TenantID, held.Receipt.OrganizationID, held.Receipt.DeviceID, held.Receipt.UserID, held.Receipt.SequenceNumber, held.Receipt.Operation, held.Receipt.EntityID, held.Receipt.PayloadHash, held.Receipt.Outcome, held.Receipt.Reason, held.Receipt.CapturedAt, held.Receipt.ReceivedAt); err != nil {
			return err
		}
	} else {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO sync_held_transaction (transaction_id, tenant_id, organization_id, device_id, sequence_number, expected_sequence, envelope, first_held_at, last_attempt_at, last_error) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (transaction_id) DO UPDATE SET expected_sequence = EXCLUDED.expected_sequence, envelope = EXCLUDED.envelope, last_attempt_at = EXCLUDED.last_attempt_at, last_error = EXCLUDED.last_error WHERE sync_held_transaction.tenant_id = EXCLUDED.tenant_id`, held.Receipt.TransactionID, held.Receipt.TenantID, held.Receipt.OrganizationID, held.Receipt.DeviceID, held.Receipt.SequenceNumber, held.ExpectedSequence, held.Envelope, held.FirstHeldAt, held.LastAttemptAt, held.LastError)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostgresRepository) ListHeld(ctx context.Context, tenantID, deviceID string) ([]HeldTransaction, error) {
	tx, err := r.beginTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT r.transaction_id, r.tenant_id, r.organization_id, r.device_id, r.user_id, r.sequence_number, r.operation, r.entity_id, r.payload_hash, r.outcome, r.reason, r.captured_at, r.received_at, h.expected_sequence, h.envelope, h.first_held_at, h.last_attempt_at, h.last_error FROM sync_held_transaction h JOIN sync_receipt r ON r.transaction_id = h.transaction_id WHERE h.tenant_id = $1 AND h.device_id = $2 ORDER BY h.expected_sequence, h.sequence_number`, tenantID, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]HeldTransaction, 0)
	for rows.Next() {
		var held HeldTransaction
		if err := rows.Scan(&held.Receipt.TransactionID, &held.Receipt.TenantID, &held.Receipt.OrganizationID, &held.Receipt.DeviceID, &held.Receipt.UserID, &held.Receipt.SequenceNumber, &held.Receipt.Operation, &held.Receipt.EntityID, &held.Receipt.PayloadHash, &held.Receipt.Outcome, &held.Receipt.Reason, &held.Receipt.CapturedAt, &held.Receipt.ReceivedAt, &held.ExpectedSequence, &held.Envelope, &held.FirstHeldAt, &held.LastAttemptAt, &held.LastError); err != nil {
			return nil, err
		}
		result = append(result, held)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
