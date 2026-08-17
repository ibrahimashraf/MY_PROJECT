package workpackagepg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GetCurrentAssignment resolves an unexpired approved-package assignment for a
// specific tenant, organization, inspection, and device. Callers remain
// responsible for authenticating the device and then loading the referenced
// approved package through GetApproved.
func (r *Repository) GetCurrentAssignment(
	ctx context.Context,
	tenantID string,
	organizationID string,
	inspectionID string,
	deviceID string,
	now time.Time,
) (Assignment, error) {
	if r == nil || r.db == nil {
		return Assignment{}, errors.New("work package repository is unavailable")
	}
	tenantID = strings.TrimSpace(tenantID)
	organizationID = strings.TrimSpace(organizationID)
	inspectionID = strings.TrimSpace(inspectionID)
	deviceID = strings.TrimSpace(deviceID)
	if tenantID == "" || organizationID == "" || inspectionID == "" || deviceID == "" {
		return Assignment{}, errors.New("assignment tenant, organization, inspection, and device are required")
	}
	if now.IsZero() {
		return Assignment{}, errors.New("assignment lookup time is required")
	}

	tx, err := r.scopedTx(ctx, tenantID, organizationID)
	if err != nil {
		return Assignment{}, err
	}
	defer tx.Rollback()

	var assignment Assignment
	err = tx.QueryRowContext(ctx, `
		SELECT tenant_id, organization_id, inspection_id, device_id,
		       package_id, package_version, authority_epoch, expires_at, assigned_at
		FROM work_package_assignment
		WHERE tenant_id = $1
		  AND organization_id = $2
		  AND inspection_id = $3
		  AND device_id = $4
		  AND expires_at > $5`,
		tenantID,
		organizationID,
		inspectionID,
		deviceID,
		now.UTC(),
	).Scan(
		&assignment.TenantID,
		&assignment.OrganizationID,
		&assignment.InspectionID,
		&assignment.DeviceID,
		&assignment.PackageID,
		&assignment.PackageVersion,
		&assignment.AuthorityEpoch,
		&assignment.ExpiresAt,
		&assignment.AssignedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	if err != nil {
		return Assignment{}, fmt.Errorf("get current work package assignment: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Assignment{}, fmt.Errorf("commit current work package assignment: %w", err)
	}
	return assignment, nil
}
