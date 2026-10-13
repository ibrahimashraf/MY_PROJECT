// Package devicetrust provides Postgres persistence for the device trust
// enrollment flow. Every query runs under a transaction-scoped tenant/org
// session configuration so the composite RLS policy in migration 0075
// isolates rows per tenant/organization.
package devicetrust

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"integin/internal/domain/device_trust"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB           = errors.New("device trust enrollment store requires a database")
	ErrRequestNotFound = errors.New("enrollment request not found")
)

var (
	sqlSaveRequest = `INSERT INTO device_enrollment_requests
		(request_id, tenant_id, organization_id, user_id, device_id, public_key, nonce, signature, attestation, status, requested_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	sqlGetRequest = `SELECT request_id, tenant_id, organization_id, user_id, device_id, public_key, nonce, signature, attestation, status, requested_at, approved_by, rejected_by, rejection_reason
		FROM device_enrollment_requests
		WHERE tenant_id = $1 AND organization_id = $2 AND request_id = $3`

	sqlUpdateStatus = `UPDATE device_enrollment_requests
		SET status = $1, approved_by = $2, rejected_by = $3, rejection_reason = $4, updated_at = now()
		WHERE tenant_id = $5 AND organization_id = $6 AND request_id = $7`

	sqlListPending = `SELECT request_id, tenant_id, organization_id, user_id, device_id, public_key, nonce, signature, attestation, requested_at
		FROM device_enrollment_requests
		WHERE tenant_id = $1 AND organization_id = $2 AND status = 'PENDING'
		ORDER BY requested_at ASC`
)

// Store persists EnrollmentRequest lifecycle state in device_enrollment_requests.
type Store struct{ db *sql.DB }

// NewStore validates its database handle and returns a ready enrollment store.
func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Store{db: db}, nil
}

// SaveRequest persists a PENDING enrollment request under the tenant scope
// carried by the request itself. Duplicate (tenant, org, request_id) conflicts
// surface as a unique-violation error from the database.
func (s *Store) SaveRequest(ctx context.Context, req *device_trust.EnrollmentRequest) error {
	if err := validateRequest(req); err != nil {
		return err
	}
	attestation, err := json.Marshal(req.Attestation)
	if err != nil {
		return err
	}
	tx, err := pgtx.BeginScope(ctx, s.db, req.TenantID, req.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, sqlSaveRequest,
		req.RequestID, req.TenantID, req.OrganizationID, req.UserID, req.DeviceID,
		req.PublicKey, req.Nonce, req.Signature, attestation,
		string(req.Status), req.RequestedAt.UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

// GetRequest loads an enrollment request by its composite identity. Unknown
// requests return ErrRequestNotFound.
func (s *Store) GetRequest(ctx context.Context, tenantID, orgID, requestID string) (*device_trust.EnrollmentRequest, error) {
	if strings.TrimSpace(requestID) == "" {
		return nil, errors.New("request_id is required")
	}
	tx, err := pgtx.BeginScope(ctx, s.db, tenantID, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var (
		req             device_trust.EnrollmentRequest
		attestation     []byte
		status          string
		approvedBy      string
		rejectedBy      string
		rejectionReason string
	)
	if err := tx.QueryRowContext(ctx, sqlGetRequest, tenantID, orgID, requestID).Scan(
		&req.RequestID, &req.TenantID, &req.OrganizationID, &req.UserID, &req.DeviceID,
		&req.PublicKey, &req.Nonce, &req.Signature, &attestation, &status, &req.RequestedAt,
		&approvedBy, &rejectedBy, &rejectionReason); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	if len(attestation) > 0 {
		if err := json.Unmarshal(attestation, &req.Attestation); err != nil {
			return nil, err
		}
	}
	req.Status = device_trust.EnrollmentStatus(status)
	req.ApprovedBy = approvedBy
	req.RejectedBy = rejectedBy
	req.RejectionReason = rejectionReason
	return &req, tx.Commit()
}

// UpdateStatus transitions a request's lifecycle to APPROVED or REJECTED under
// the tenant scope, recording the acting admin and, for rejections, the
// reason. Unknown requests return ErrRequestNotFound.
func (s *Store) UpdateStatus(ctx context.Context, tenantID, orgID, requestID string, status device_trust.EnrollmentStatus, actor, reason string) error {
	if strings.TrimSpace(requestID) == "" {
		return errors.New("request_id is required")
	}
	approvedBy, rejectedBy, rejectionReason := "", "", ""
	switch status {
	case device_trust.EnrollmentApproved:
		if strings.TrimSpace(actor) == "" {
			return errors.New("approving tenant admin is required")
		}
		approvedBy = actor
	case device_trust.EnrollmentRejected:
		if strings.TrimSpace(actor) == "" {
			return errors.New("rejecting tenant admin is required")
		}
		if strings.TrimSpace(reason) == "" {
			return errors.New("rejection reason is required")
		}
		rejectedBy = actor
		rejectionReason = reason
	default:
		return fmt.Errorf("cannot update enrollment request to %s", status)
	}
	tx, err := pgtx.BeginScope(ctx, s.db, tenantID, orgID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, sqlUpdateStatus,
		string(status), approvedBy, rejectedBy, rejectionReason, tenantID, orgID, requestID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrRequestNotFound
	}
	return tx.Commit()
}

// ListPending returns every PENDING enrollment request for the tenant scope.
func (s *Store) ListPending(ctx context.Context, tenantID, orgID string) ([]device_trust.EnrollmentRequest, error) {
	tx, err := pgtx.BeginScope(ctx, s.db, tenantID, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, sqlListPending, tenantID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]device_trust.EnrollmentRequest, 0)
	for rows.Next() {
		var (
			req         device_trust.EnrollmentRequest
			attestation []byte
			status      string
		)
		if err := rows.Scan(&req.RequestID, &req.TenantID, &req.OrganizationID, &req.UserID,
			&req.DeviceID, &req.PublicKey, &req.Nonce, &req.Signature, &attestation, &status, &req.RequestedAt); err != nil {
			return nil, err
		}
		if len(attestation) > 0 {
			if err := json.Unmarshal(attestation, &req.Attestation); err != nil {
				return nil, err
			}
		}
		req.Status = device_trust.EnrollmentStatus(status)
		out = append(out, req)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit()
}

// validateRequest enforces the minimal row contract before any database work.
func validateRequest(req *device_trust.EnrollmentRequest) error {
	if req == nil {
		return errors.New("enrollment request is required")
	}
	for field, value := range map[string]string{
		"request_id": req.RequestID, "tenant_id": req.TenantID, "organization_id": req.OrganizationID,
		"user_id": req.UserID, "device_id": req.DeviceID, "public_key": req.PublicKey,
		"nonce": req.Nonce, "signature": req.Signature,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if req.RequestedAt.IsZero() {
		return errors.New("requested_at is required")
	}
	switch req.Status {
	case device_trust.EnrollmentPending, device_trust.EnrollmentApproved, device_trust.EnrollmentRejected:
	default:
		return fmt.Errorf("invalid enrollment status: %s", req.Status)
	}
	return nil
}
