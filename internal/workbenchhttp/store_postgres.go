package workbenchhttp

import (
	"context"
	"database/sql"
	"errors"

	"integin/internal/domain/device_trust"
)

// postgresStore is the durable authority store over the tenant-scoped
// registries from migrations 0002/0075 (devices, enrollments, held sync) and
// 0077 (legal holds, retention, export approvals). Every operation runs inside
// one transaction that first pins integin.tenant_id / integin.organization_id
// with is_local=true SET LOCAL semantics so row-level security is honoured
// without polluting pooled connections.
type postgresStore struct {
	db *sql.DB
}

// withTenantScope pins the tenant/organization GUCs (is_local = true so they
// expire at transaction end) and then runs fn against that transaction.
// Rollback is explicit on every error path; ErrTxDone is expected and handled
// only by not calling Rollback after a successful Commit.
func withTenantScope(ctx context.Context, db *sql.DB, tenantID, organizationID string, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, tenantID, organizationID); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	return tx.Commit()
}

func (s *postgresStore) ListDevices(ctx context.Context, tenantID, organizationID string) ([]DeviceView, error) {
	result := make([]DeviceView, 0)
	err := withTenantScope(ctx, s.db, tenantID, organizationID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT device_id, tenant_id, organization_id, user_id, encode(public_key, 'escape'), state, authority_epoch, enrolled_at, revoked_at, coalesce(revocation_reason, '')
			FROM device_registry
			WHERE tenant_id = $1 AND organization_id = $2
			ORDER BY enrolled_at`, tenantID, organizationID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var view DeviceView
			if err := rows.Scan(&view.ID, &view.TenantID, &view.OrganizationID, &view.UserID, &view.PublicKey, &view.State, &view.Epoch, &view.EnrolledAt, &view.RevokedAt, &view.RevocationReason); err != nil {
				_ = rows.Close()
				return err
			}
			result = append(result, view)
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresStore) ListEnrollmentRequests(ctx context.Context, tenantID, organizationID string) ([]EnrollmentRequestView, error) {
	result := make([]EnrollmentRequestView, 0)
	err := withTenantScope(ctx, s.db, tenantID, organizationID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT request_id, tenant_id, organization_id, user_id, device_id, public_key,
			       coalesce(attestation->>'key_origin', ''), coalesce(attestation->>'biometric_bound', 'false'), coalesce(attestation->>'os_version', ''),
			       status, requested_at, coalesce(approved_by, ''), coalesce(rejected_by, '')
			FROM device_enrollment_requests
			WHERE tenant_id = $1 AND organization_id = $2
			ORDER BY requested_at`, tenantID, organizationID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var view EnrollmentRequestView
			var keyOrigin, biometricBound, osVersion, approvedBy, rejectedBy string
			if err := rows.Scan(&view.RequestID, &view.TenantID, &view.OrganizationID, &view.UserID, &view.DeviceID, &view.PublicKey,
				&keyOrigin, &biometricBound, &osVersion, &view.Status, &view.RequestedAt, &approvedBy, &rejectedBy); err != nil {
				_ = rows.Close()
				return err
			}
			view.ApprovedBy = approvedBy
			view.RejectedBy = rejectedBy
			view.Attestation = device_trust.EnrollmentAttestation{KeyOrigin: keyOrigin, BiometricBound: biometricBound == "true", OSVersion: osVersion}
			result = append(result, view)
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresStore) ApproveEnrollment(ctx context.Context, actorID, tenantID, organizationID, requestID string, epoch uint64) (DeviceView, error) {
	var device DeviceView
	err := withTenantScope(ctx, s.db, tenantID, organizationID, func(tx *sql.Tx) error {
		var status, userID, deviceID, publicKey string
		err := tx.QueryRowContext(ctx, `
			SELECT status, user_id, device_id, public_key
			FROM device_enrollment_requests
			WHERE tenant_id = $1 AND organization_id = $2 AND request_id = $3`, tenantID, organizationID, requestID).Scan(&status, &userID, &deviceID, &publicKey)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "PENDING" {
			return ErrConflict
		}
		insert, err := tx.ExecContext(ctx, `
			INSERT INTO device_registry (device_id, tenant_id, organization_id, user_id, key_id, public_key, state, authority_epoch, enrolled_at, updated_at)
			VALUES ($1, $2, $3, $4, $1, decode($5, 'escape'), 'TRUSTED', $6, now(), now())
			ON CONFLICT (device_id) DO NOTHING`, deviceID, tenantID, organizationID, userID, publicKey, int64(epoch))
		if err != nil {
			return err
		}
		affected, err := insert.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE device_enrollment_requests
			SET status = 'APPROVED', approved_by = $1, updated_at = now()
			WHERE tenant_id = $2 AND organization_id = $3 AND request_id = $4`, actorID, tenantID, organizationID, requestID); err != nil {
			return err
		}
		device, err = s.deviceByID(ctx, tx, tenantID, organizationID, deviceID)
		return err
	})
	if err != nil {
		return DeviceView{}, err
	}
	return device, nil
}

func (s *postgresStore) deviceByID(ctx context.Context, tx *sql.Tx, tenantID, organizationID, deviceID string) (DeviceView, error) {
	var view DeviceView
	err := tx.QueryRowContext(ctx, `
		SELECT device_id, tenant_id, organization_id, user_id, encode(public_key, 'escape'), state, authority_epoch, enrolled_at, revoked_at, coalesce(revocation_reason, '')
		FROM device_registry
		WHERE tenant_id = $1 AND organization_id = $2 AND device_id = $3`, tenantID, organizationID, deviceID).Scan(
		&view.ID, &view.TenantID, &view.OrganizationID, &view.UserID, &view.PublicKey, &view.State, &view.Epoch, &view.EnrolledAt, &view.RevokedAt, &view.RevocationReason)
	if errors.Is(err, sql.ErrNoRows) {
		return DeviceView{}, ErrNotFound
	}
	return view, err
}

func (s *postgresStore) RevokeDevice(ctx context.Context, tenantID, organizationID, deviceID, reason string) (DeviceView, error) {
	var view DeviceView
	err := withTenantScope(ctx, s.db, tenantID, organizationID, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
			UPDATE device_registry
			SET state = 'REVOKED', revoked_at = now(), revocation_reason = $1
			WHERE tenant_id = $2 AND organization_id = $3 AND device_id = $4
			RETURNING device_id, tenant_id, organization_id, user_id, encode(public_key, 'escape'), state, authority_epoch, enrolled_at, revoked_at, coalesce(revocation_reason, '')`,
			reason, tenantID, organizationID, deviceID).Scan(
			&view.ID, &view.TenantID, &view.OrganizationID, &view.UserID, &view.PublicKey, &view.State, &view.Epoch, &view.EnrolledAt, &view.RevokedAt, &view.RevocationReason)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return DeviceView{}, err
	}
	return view, nil
}

func (s *postgresStore) ListHeld(ctx context.Context, tenantID, organizationID string) ([]HeldTransaction, error) {
	result := make([]HeldTransaction, 0)
	err := withTenantScope(ctx, s.db, tenantID, organizationID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT transaction_id, device_id, sequence_number, last_error, first_held_at, last_attempt_at, coalesce(envelope->>'PayloadHash', '')
			FROM sync_held_transaction
			WHERE tenant_id = $1 AND organization_id = $2
			ORDER BY first_held_at`, tenantID, organizationID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var view HeldTransaction
			if err := rows.Scan(&view.TransactionID, &view.DeviceID, &view.SequenceNumber, &view.ErrorReason, &view.HeldAt, &view.LastAttemptAt, &view.PayloadHash); err != nil {
				_ = rows.Close()
				return err
			}
			view.Status = statusHeld
			result = append(result, view)
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresStore) ReconcileHeld(ctx context.Context, tenantID, organizationID, transactionID string) (ReconcileResult, error) {
	var result ReconcileResult
	err := withTenantScope(ctx, s.db, tenantID, organizationID, func(tx *sql.Tx) error {
		var view HeldTransaction
		err := tx.QueryRowContext(ctx, `
			UPDATE sync_held_transaction
			SET last_attempt_at = now(), last_error = 'reconciliation retry triggered'
			WHERE tenant_id = $1 AND organization_id = $2 AND transaction_id = $3
			RETURNING transaction_id, device_id, sequence_number, last_error, first_held_at, last_attempt_at, coalesce(envelope->>'PayloadHash', '')`,
			tenantID, organizationID, transactionID).Scan(&view.TransactionID, &view.DeviceID, &view.SequenceNumber, &view.ErrorReason, &view.HeldAt, &view.LastAttemptAt, &view.PayloadHash)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		result = ReconcileResult{TransactionID: view.TransactionID, Status: statusHeld, ErrorReason: view.ErrorReason}
		if view.LastAttemptAt != nil {
			result.RetriedAt = *view.LastAttemptAt
		}
		return nil
	})
	if err != nil {
		return ReconcileResult{}, err
	}
	return result, nil
}

func (s *postgresStore) ListLegalHolds(ctx context.Context, tenantID, _ string) ([]LegalHold, error) {
	result := make([]LegalHold, 0)
	err := withTenantScope(ctx, s.db, tenantID, "", func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT id, tenant_id, entity_type, entity_id, reason, placed_by, placed_at, status, released_by, released_at
			FROM legal_hold_registry
			WHERE tenant_id = $1 AND status = 'ACTIVE'
			ORDER BY placed_at`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var hold LegalHold
			if err := rows.Scan(&hold.ID, &hold.TenantID, &hold.EntityType, &hold.EntityID, &hold.Reason, &hold.PlacedBy, &hold.PlacedAt, &hold.Status, &hold.ReleasedBy, &hold.ReleasedAt); err != nil {
				_ = rows.Close()
				return err
			}
			result = append(result, hold)
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresStore) PlaceLegalHold(ctx context.Context, actorID, tenantID, _, entityType, entityID, reason string) (LegalHold, error) {
	id, err := newRandomID("legalhold-")
	if err != nil {
		return LegalHold{}, err
	}
	var hold LegalHold
	err = withTenantScope(ctx, s.db, tenantID, "", func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `
			INSERT INTO legal_hold_registry (id, tenant_id, entity_type, entity_id, reason, placed_by, placed_at, status)
			VALUES ($1, $2, $3, $4, $5, $6, now(), 'ACTIVE')
			RETURNING id, tenant_id, entity_type, entity_id, reason, placed_by, placed_at, status, released_by, released_at`,
			id, tenantID, entityType, entityID, reason, actorID).Scan(
			&hold.ID, &hold.TenantID, &hold.EntityType, &hold.EntityID, &hold.Reason, &hold.PlacedBy, &hold.PlacedAt, &hold.Status, &hold.ReleasedBy, &hold.ReleasedAt)
	})
	if err != nil {
		return LegalHold{}, err
	}
	return hold, nil
}

func (s *postgresStore) ReleaseLegalHold(ctx context.Context, actorID, tenantID, _, id string) (LegalHold, error) {
	var hold LegalHold
	err := withTenantScope(ctx, s.db, tenantID, "", func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
			UPDATE legal_hold_registry
			SET status = 'RELEASED', released_by = $1, released_at = now()
			WHERE tenant_id = $2 AND id = $3 AND status = 'ACTIVE'
			RETURNING id, tenant_id, entity_type, entity_id, reason, placed_by, placed_at, status, released_by, released_at`,
			actorID, tenantID, id).Scan(
			&hold.ID, &hold.TenantID, &hold.EntityType, &hold.EntityID, &hold.Reason, &hold.PlacedBy, &hold.PlacedAt, &hold.Status, &hold.ReleasedBy, &hold.ReleasedAt)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		return tx.QueryRowContext(ctx, `
			SELECT id, tenant_id, entity_type, entity_id, reason, placed_by, placed_at, status, released_by, released_at
			FROM legal_hold_registry
			WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(
			&hold.ID, &hold.TenantID, &hold.EntityType, &hold.EntityID, &hold.Reason, &hold.PlacedBy, &hold.PlacedAt, &hold.Status, &hold.ReleasedBy, &hold.ReleasedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return LegalHold{}, ErrNotFound
	}
	if err != nil {
		return LegalHold{}, err
	}
	return hold, nil
}

func (s *postgresStore) ListRetentionPolicies(ctx context.Context, tenantID, _ string) ([]RetentionPolicy, error) {
	result := make([]RetentionPolicy, 0)
	err := withTenantScope(ctx, s.db, tenantID, "", func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT tenant_id, entity_type, retention_days, action, updated_at, updated_by
			FROM tenant_retention_policy
			WHERE tenant_id = $1
			ORDER BY entity_type`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var policy RetentionPolicy
			if err := rows.Scan(&policy.TenantID, &policy.EntityType, &policy.RetentionDays, &policy.Action, &policy.UpdatedAt, &policy.UpdatedBy); err != nil {
				_ = rows.Close()
				return err
			}
			result = append(result, policy)
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresStore) ListExportApprovals(ctx context.Context, tenantID, _ string) ([]ExportApproval, error) {
	result := make([]ExportApproval, 0)
	err := withTenantScope(ctx, s.db, tenantID, "", func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT id, tenant_id, export_id, requested_by, coalesce(approved_by, ''), status, created_at, decided_at, coalesce(rejection_reason, '')
			FROM export_approval_registry
			WHERE tenant_id = $1
			ORDER BY created_at`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var approval ExportApproval
			if err := rows.Scan(&approval.ID, &approval.TenantID, &approval.ExportID, &approval.RequestedBy, &approval.ApprovedBy, &approval.Status, &approval.CreatedAt, &approval.DecidedAt, &approval.RejectionReason); err != nil {
				_ = rows.Close()
				return err
			}
			result = append(result, approval)
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *postgresStore) DecideExportApproval(ctx context.Context, actorID, tenantID, _, exportID, status, rejectionReason string) (ExportApproval, error) {
	var approval ExportApproval
	err := withTenantScope(ctx, s.db, tenantID, "", func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
			UPDATE export_approval_registry
			SET status = $1, approved_by = $2, decided_at = now(), rejection_reason = $3
			WHERE tenant_id = $4 AND export_id = $5 AND status = 'PENDING'
			RETURNING id, tenant_id, export_id, requested_by, coalesce(approved_by, ''), status, created_at, decided_at, coalesce(rejection_reason, '')`,
			status, actorID, rejectionReason, tenantID, exportID).Scan(
			&approval.ID, &approval.TenantID, &approval.ExportID, &approval.RequestedBy, &approval.ApprovedBy, &approval.Status, &approval.CreatedAt, &approval.DecidedAt, &approval.RejectionReason)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var existingStatus string
		lookupErr := tx.QueryRowContext(ctx, `
			SELECT status FROM export_approval_registry WHERE tenant_id = $1 AND export_id = $2`, tenantID, exportID).Scan(&existingStatus)
		if errors.Is(lookupErr, sql.ErrNoRows) {
			return ErrNotFound
		}
		if lookupErr != nil {
			return lookupErr
		}
		return ErrConflict
	})
	if err != nil {
		return ExportApproval{}, err
	}
	return approval, nil
}
