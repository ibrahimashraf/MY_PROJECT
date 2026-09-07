package calibration

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sharedcalibration "integin/internal/shared/calibration"
	"integin/internal/shared/pgtx"
)

var ErrNilDB = errors.New("calibration postgres store requires a database")

var (
	sqlUpsert = `INSERT INTO tool_calibration_registry
		(id, tenant_id, organization_id, equipment_id, serial_number, lab_certificate_ref, uncertainty_tolerance, calibration_date, next_due_date, technician_id, result, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (tenant_id, organization_id, id) DO UPDATE SET
			equipment_id = EXCLUDED.equipment_id,
			serial_number = EXCLUDED.serial_number,
			lab_certificate_ref = EXCLUDED.lab_certificate_ref,
			uncertainty_tolerance = EXCLUDED.uncertainty_tolerance,
			calibration_date = EXCLUDED.calibration_date,
			next_due_date = EXCLUDED.next_due_date,
			technician_id = EXCLUDED.technician_id,
			result = EXCLUDED.result,
			status = EXCLUDED.status`

	sqlHasUnexpiredActive = `SELECT EXISTS (
		SELECT 1 FROM tool_calibration_registry
		WHERE equipment_id = $1 AND status = 'ACTIVE' AND next_due_date > $2)`
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Store{db: db}, nil
}

// Upsert writes the record under the tenant/org scope, creating or replacing
// the existing row. Requires an existing calibration service record.
func (s *Store) Upsert(ctx context.Context, tenantID, orgID string, record sharedcalibration.Record) error {
	if err := record.Validate(); err != nil {
		return err
	}
	tx, err := pgtx.BeginScope(ctx, s.db, tenantID, orgID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, sqlUpsert,
		record.ID, tenantID, orgID, record.EquipmentID,
		record.SerialNumber, record.LabCertificateRef, record.UncertaintyTolerance,
		record.CalibrationDate.UTC(), record.NextDueDate.UTC(),
		record.TechnicianID, record.Result, string(record.Status)); err != nil {
		return err
	}
	return tx.Commit()
}

// SubmissionBlocked returns true when no ACTIVE row with an unexpired
// next_due_date covers the equipment at the given instant. Expired-or-missing
// calibration hard-blocks submission (mirrors Service.SubmissionAllowed).
func (s *Store) SubmissionBlocked(ctx context.Context, tenantID, orgID, equipmentID string, at time.Time) (bool, error) {
	if err := ValidateEquipmentID(equipmentID); err != nil {
		return false, err
	}
	tx, err := pgtx.BeginScope(ctx, s.db, tenantID, orgID)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var unexpired bool
	if err := tx.QueryRowContext(ctx, sqlHasUnexpiredActive, equipmentID, at.UTC()).Scan(&unexpired); err != nil {
		return false, err
	}
	return !unexpired, tx.Commit()
}
