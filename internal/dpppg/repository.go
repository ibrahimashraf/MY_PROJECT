package dpppg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strings"
	"time"

	"integin/internal/domain/dpp"
	"integin/internal/shared/pgtx"
)

var (
	ErrNotFound      = errors.New("dpp entity not found")
	ErrImmutableLock = errors.New("cannot modify or delete dpp in immutable status")
)

type Repository struct {
	db  *sql.DB
	now func() time.Time
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("database connection is required")
	}
	return &Repository{
		db:  db,
		now: time.Now,
	}, nil
}

// ComputeDPPHash calculates the deterministic SHA-256 Merkle hash across the 4 pillars.
func ComputeDPPHash(assetID, assignment, update, use, disposal string) []byte {
	h := sha256.New()
	h.Write([]byte("asset:" + assetID + "\n"))
	h.Write([]byte("assignment:" + assignment + "\n"))
	h.Write([]byte("update:" + update + "\n"))
	h.Write([]byte("use:" + use + "\n"))
	h.Write([]byte("disposal:" + disposal + "\n"))
	sum := h.Sum(nil)
	return sum
}

// CreateProductPassportDPP implements dpp.Repository.
func (r *Repository) CreateProductPassportDPP(ctx context.Context, actor dpp.ActorContext, d dpp.ProductPassportDPP) (dpp.ProductPassportDPP, error) {
	if err := actor.Validate(); err != nil {
		return dpp.ProductPassportDPP{}, err
	}
	if strings.TrimSpace(d.ID) == "" {
		return dpp.ProductPassportDPP{}, errors.New("dpp id is required")
	}
	if strings.TrimSpace(d.AssetID) == "" {
		return dpp.ProductPassportDPP{}, errors.New("asset id is required")
	}
	if d.DPPStatus == "" {
		d.DPPStatus = "ASSIGNMENT"
	}
	if d.DPPVersion <= 0 {
		d.DPPVersion = 1
	}

	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return dpp.ProductPassportDPP{}, err
	}
	defer tx.Rollback()

	now := r.now().UTC()
	query := `
		INSERT INTO product_passport_dpp (
			id, tenant_id, organization_id, asset_id, serial_number, batch_number,
			manufacturer_id, dpp_status, dpp_version, dpp_sha256,
			assignment_payload, update_payload, use_payload, disposal_payload,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''),
			NULLIF($7, ''), $8, $9, $10,
			NULLIF($11, '')::jsonb, NULLIF($12, '')::jsonb, NULLIF($13, '')::jsonb, NULLIF($14, '')::jsonb,
			$15, $15
		) RETURNING id, tenant_id, organization_id, asset_id, COALESCE(serial_number,''), COALESCE(batch_number,''),
		            COALESCE(manufacturer_id,''), dpp_status, dpp_version, dpp_sha256,
		            COALESCE(assignment_payload::text,''), COALESCE(update_payload::text,''),
		            COALESCE(use_payload::text,''), COALESCE(disposal_payload::text,''),
		            created_at, updated_at
	`

	var res dpp.ProductPassportDPP
	var dppHash []byte
	if d.DPPStatus == "IMMUTABLE" {
		dppHash = ComputeDPPHash(d.AssetID, d.AssignmentPayload, d.UpdatePayload, d.UsePayload, d.DisposalPayload)
	}

	err = tx.QueryRowContext(ctx, query,
		d.ID, actor.TenantID, actor.OrganizationID, d.AssetID, d.SerialNumber, d.BatchNumber,
		d.ManufacturerID, d.DPPStatus, d.DPPVersion, dppHash,
		d.AssignmentPayload, d.UpdatePayload, d.UsePayload, d.DisposalPayload,
		now,
	).Scan(
		&res.ID, &res.TenantID, &res.OrganizationID, &res.AssetID, &res.SerialNumber, &res.BatchNumber,
		&res.ManufacturerID, &res.DPPStatus, &res.DPPVersion, &res.DPPHash,
		&res.AssignmentPayload, &res.UpdatePayload, &res.UsePayload, &res.DisposalPayload,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return dpp.ProductPassportDPP{}, err
	}

	if err := tx.Commit(); err != nil {
		return dpp.ProductPassportDPP{}, err
	}
	return res, nil
}

// GetProductPassportDPP implements dpp.Repository.
func (r *Repository) GetProductPassportDPP(ctx context.Context, actor dpp.ActorContext, id string) (dpp.ProductPassportDPP, error) {
	if err := actor.Validate(); err != nil {
		return dpp.ProductPassportDPP{}, err
	}
	if strings.TrimSpace(id) == "" {
		return dpp.ProductPassportDPP{}, errors.New("id is required")
	}

	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return dpp.ProductPassportDPP{}, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id, organization_id, asset_id, COALESCE(serial_number,''), COALESCE(batch_number,''),
		       COALESCE(manufacturer_id,''), dpp_status, dpp_version, dpp_sha256,
		       COALESCE(assignment_payload::text,''), COALESCE(update_payload::text,''),
		       COALESCE(use_payload::text,''), COALESCE(disposal_payload::text,''),
		       created_at, updated_at
		FROM product_passport_dpp
		WHERE id = $1 AND tenant_id = $2 AND organization_id = $3
	`

	var res dpp.ProductPassportDPP
	err = tx.QueryRowContext(ctx, query, id, actor.TenantID, actor.OrganizationID).Scan(
		&res.ID, &res.TenantID, &res.OrganizationID, &res.AssetID, &res.SerialNumber, &res.BatchNumber,
		&res.ManufacturerID, &res.DPPStatus, &res.DPPVersion, &res.DPPHash,
		&res.AssignmentPayload, &res.UpdatePayload, &res.UsePayload, &res.DisposalPayload,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return dpp.ProductPassportDPP{}, ErrNotFound
	}
	if err != nil {
		return dpp.ProductPassportDPP{}, err
	}

	_ = tx.Commit()
	return res, nil
}

// GetProductPassportDPPByAsset implements dpp.Repository.
func (r *Repository) GetProductPassportDPPByAsset(ctx context.Context, actor dpp.ActorContext, assetID string) (dpp.ProductPassportDPP, bool, error) {
	if err := actor.Validate(); err != nil {
		return dpp.ProductPassportDPP{}, false, err
	}
	if strings.TrimSpace(assetID) == "" {
		return dpp.ProductPassportDPP{}, false, errors.New("asset id is required")
	}

	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return dpp.ProductPassportDPP{}, false, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id, organization_id, asset_id, COALESCE(serial_number,''), COALESCE(batch_number,''),
		       COALESCE(manufacturer_id,''), dpp_status, dpp_version, dpp_sha256,
		       COALESCE(assignment_payload::text,''), COALESCE(update_payload::text,''),
		       COALESCE(use_payload::text,''), COALESCE(disposal_payload::text,''),
		       created_at, updated_at
		FROM product_passport_dpp
		WHERE asset_id = $1 AND tenant_id = $2 AND organization_id = $3
	`

	var res dpp.ProductPassportDPP
	err = tx.QueryRowContext(ctx, query, assetID, actor.TenantID, actor.OrganizationID).Scan(
		&res.ID, &res.TenantID, &res.OrganizationID, &res.AssetID, &res.SerialNumber, &res.BatchNumber,
		&res.ManufacturerID, &res.DPPStatus, &res.DPPVersion, &res.DPPHash,
		&res.AssignmentPayload, &res.UpdatePayload, &res.UsePayload, &res.DisposalPayload,
		&res.CreatedAt, &res.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return dpp.ProductPassportDPP{}, false, nil
	}
	if err != nil {
		return dpp.ProductPassportDPP{}, false, err
	}

	_ = tx.Commit()
	return res, true, nil
}

// UpdateProductPassportDPP implements dpp.Repository.
func (r *Repository) UpdateProductPassportDPP(ctx context.Context, actor dpp.ActorContext, d dpp.ProductPassportDPP) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(d.ID) == "" {
		return errors.New("id is required")
	}

	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if already immutable
	var currentStatus string
	err = tx.QueryRowContext(ctx, `SELECT dpp_status FROM product_passport_dpp WHERE id = $1 AND tenant_id = $2 AND organization_id = $3`, d.ID, actor.TenantID, actor.OrganizationID).Scan(&currentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if currentStatus == "IMMUTABLE" {
		return ErrImmutableLock
	}

	var dppHash []byte
	if d.DPPStatus == "IMMUTABLE" {
		dppHash = ComputeDPPHash(d.AssetID, d.AssignmentPayload, d.UpdatePayload, d.UsePayload, d.DisposalPayload)
	}

	query := `
		UPDATE product_passport_dpp
		SET serial_number = COALESCE(NULLIF($1, ''), serial_number),
		    batch_number = COALESCE(NULLIF($2, ''), batch_number),
		    manufacturer_id = COALESCE(NULLIF($3, ''), manufacturer_id),
		    dpp_status = $4,
		    dpp_version = dpp_version + 1,
		    dpp_sha256 = COALESCE($5, dpp_sha256),
		    assignment_payload = COALESCE(NULLIF($6, '')::jsonb, assignment_payload),
		    update_payload = COALESCE(NULLIF($7, '')::jsonb, update_payload),
		    use_payload = COALESCE(NULLIF($8, '')::jsonb, use_payload),
		    disposal_payload = COALESCE(NULLIF($9, '')::jsonb, disposal_payload),
		    updated_at = $10
		WHERE id = $11 AND tenant_id = $12 AND organization_id = $13
	`

	res, err := tx.ExecContext(ctx, query,
		d.SerialNumber, d.BatchNumber, d.ManufacturerID, d.DPPStatus, dppHash,
		d.AssignmentPayload, d.UpdatePayload, d.UsePayload, d.DisposalPayload,
		r.now().UTC(), d.ID, actor.TenantID, actor.OrganizationID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return tx.Commit()
}

// CreateRegulatoryMonitor implements dpp.Repository.
func (r *Repository) CreateRegulatoryMonitor(ctx context.Context, actor dpp.ActorContext, reg dpp.RegulatoryMonitor) (dpp.RegulatoryMonitor, error) {
	if err := actor.Validate(); err != nil {
		return dpp.RegulatoryMonitor{}, err
	}
	if strings.TrimSpace(reg.ID) == "" {
		return dpp.RegulatoryMonitor{}, errors.New("id is required")
	}
	if strings.TrimSpace(reg.RegulationCode) == "" || strings.TrimSpace(reg.Source) == "" {
		return dpp.RegulatoryMonitor{}, errors.New("regulation_code and source are required")
	}
	if reg.Status == "" {
		reg.Status = "MONITORING"
	}

	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return dpp.RegulatoryMonitor{}, err
	}
	defer tx.Rollback()

	now := r.now().UTC()
	query := `
		INSERT INTO regulatory_monitor (
			id, tenant_id, organization_id, source, regulation_code, title,
			effective_date, summary, impact_assessment, status,
			last_checked_at, checked_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			NULLIF($7, '')::date, NULLIF($8, ''), NULLIF($9, ''), $10,
			$11, NULLIF($12, ''), $11, $11
		) RETURNING id, tenant_id, organization_id, source, regulation_code, title,
		            COALESCE(summary, ''), COALESCE(impact_assessment, ''), status,
		            last_checked_at, COALESCE(checked_by, '')
	`

	var dateStr string
	if !reg.EffectiveDate.IsZero() {
		dateStr = reg.EffectiveDate.Format("2006-01-02")
	}

	err = tx.QueryRowContext(ctx, query,
		reg.ID, actor.TenantID, actor.OrganizationID, reg.Source, reg.RegulationCode, reg.Title,
		dateStr, reg.Summary, reg.ImpactAssessment, reg.Status,
		now, reg.CheckedBy,
	).Scan(
		&reg.ID, &reg.TenantID, &reg.OrganizationID, &reg.Source, &reg.RegulationCode, &reg.Title,
		&reg.Summary, &reg.ImpactAssessment, &reg.Status,
		&reg.LastCheckedAt, &reg.CheckedBy,
	)
	if err != nil {
		return dpp.RegulatoryMonitor{}, err
	}

	if err := tx.Commit(); err != nil {
		return dpp.RegulatoryMonitor{}, err
	}
	return reg, nil
}

// GetRegulatoryMonitor implements dpp.Repository.
func (r *Repository) GetRegulatoryMonitor(ctx context.Context, actor dpp.ActorContext, id string) (dpp.RegulatoryMonitor, error) {
	if err := actor.Validate(); err != nil {
		return dpp.RegulatoryMonitor{}, err
	}
	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return dpp.RegulatoryMonitor{}, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id, organization_id, source, regulation_code, title,
		       COALESCE(summary, ''), COALESCE(impact_assessment, ''), status,
		       last_checked_at, COALESCE(checked_by, '')
		FROM regulatory_monitor
		WHERE id = $1 AND tenant_id = $2 AND organization_id = $3
	`

	var reg dpp.RegulatoryMonitor
	err = tx.QueryRowContext(ctx, query, id, actor.TenantID, actor.OrganizationID).Scan(
		&reg.ID, &reg.TenantID, &reg.OrganizationID, &reg.Source, &reg.RegulationCode, &reg.Title,
		&reg.Summary, &reg.ImpactAssessment, &reg.Status,
		&reg.LastCheckedAt, &reg.CheckedBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return dpp.RegulatoryMonitor{}, ErrNotFound
	}
	if err != nil {
		return dpp.RegulatoryMonitor{}, err
	}
	_ = tx.Commit()
	return reg, nil
}

// ListRegulatoryMonitors implements dpp.Repository.
func (r *Repository) ListRegulatoryMonitors(ctx context.Context, actor dpp.ActorContext, source, status string) ([]dpp.RegulatoryMonitor, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id, organization_id, source, regulation_code, title,
		       COALESCE(summary, ''), COALESCE(impact_assessment, ''), status,
		       last_checked_at, COALESCE(checked_by, '')
		FROM regulatory_monitor
		WHERE tenant_id = $1 AND organization_id = $2
		  AND ($3 = '' OR source = $3)
		  AND ($4 = '' OR status = $4)
		ORDER BY created_at DESC
	`
	rows, err := tx.QueryContext(ctx, query, actor.TenantID, actor.OrganizationID, source, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []dpp.RegulatoryMonitor
	for rows.Next() {
		var reg dpp.RegulatoryMonitor
		if err := rows.Scan(
			&reg.ID, &reg.TenantID, &reg.OrganizationID, &reg.Source, &reg.RegulationCode, &reg.Title,
			&reg.Summary, &reg.ImpactAssessment, &reg.Status,
			&reg.LastCheckedAt, &reg.CheckedBy,
		); err != nil {
			return nil, err
		}
		list = append(list, reg)
	}
	_ = tx.Commit()
	return list, nil
}

// UpdateRegulatoryMonitor implements dpp.Repository.
func (r *Repository) UpdateRegulatoryMonitor(ctx context.Context, actor dpp.ActorContext, reg dpp.RegulatoryMonitor) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE regulatory_monitor
		SET title = COALESCE(NULLIF($1, ''), title),
		    summary = COALESCE(NULLIF($2, ''), summary),
		    impact_assessment = COALESCE(NULLIF($3, ''), impact_assessment),
		    status = COALESCE(NULLIF($4, ''), status),
		    last_checked_at = $5,
		    checked_by = COALESCE(NULLIF($6, ''), checked_by),
		    updated_at = $5
		WHERE id = $7 AND tenant_id = $8 AND organization_id = $9
	`
	now := r.now().UTC()
	res, err := tx.ExecContext(ctx, query,
		reg.Title, reg.Summary, reg.ImpactAssessment, reg.Status,
		now, reg.CheckedBy, reg.ID, actor.TenantID, actor.OrganizationID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// CreateComplianceAction implements dpp.Repository.
func (r *Repository) CreateComplianceAction(ctx context.Context, actor dpp.ActorContext, a dpp.ComplianceAction) (dpp.ComplianceAction, error) {
	if err := actor.Validate(); err != nil {
		return dpp.ComplianceAction{}, err
	}
	if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.MonitorID) == "" || strings.TrimSpace(a.ActionType) == "" {
		return dpp.ComplianceAction{}, errors.New("id, monitor_id, and action_type are required")
	}
	if a.Status == "" {
		a.Status = "TODO"
	}

	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return dpp.ComplianceAction{}, err
	}
	defer tx.Rollback()

	now := r.now().UTC()
	query := `
		INSERT INTO compliance_action (
			id, tenant_id, organization_id, monitor_id, action_type, status,
			assignee, due_date, evidence_refs, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			NULLIF($7, ''), NULLIF($8, '')::date, NULLIF($9, '')::jsonb, $10, $10
		) RETURNING id, tenant_id, organization_id, monitor_id, action_type, status,
		            COALESCE(assignee, ''), COALESCE(evidence_refs::text, '')
	`

	var dueStr string
	if !a.DueDate.IsZero() {
		dueStr = a.DueDate.Format("2006-01-02")
	}

	err = tx.QueryRowContext(ctx, query,
		a.ID, actor.TenantID, actor.OrganizationID, a.MonitorID, a.ActionType, a.Status,
		a.Assignee, dueStr, a.EvidenceRefs, now,
	).Scan(
		&a.ID, &a.TenantID, &a.OrganizationID, &a.MonitorID, &a.ActionType, &a.Status,
		&a.Assignee, &a.EvidenceRefs,
	)
	if err != nil {
		return dpp.ComplianceAction{}, err
	}

	if err := tx.Commit(); err != nil {
		return dpp.ComplianceAction{}, err
	}
	return a, nil
}

// GetComplianceAction implements dpp.Repository.
func (r *Repository) GetComplianceAction(ctx context.Context, actor dpp.ActorContext, id string) (dpp.ComplianceAction, error) {
	if err := actor.Validate(); err != nil {
		return dpp.ComplianceAction{}, err
	}
	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return dpp.ComplianceAction{}, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id, organization_id, monitor_id, action_type, status,
		       COALESCE(assignee, ''), COALESCE(evidence_refs::text, '')
		FROM compliance_action
		WHERE id = $1 AND tenant_id = $2 AND organization_id = $3
	`

	var a dpp.ComplianceAction
	err = tx.QueryRowContext(ctx, query, id, actor.TenantID, actor.OrganizationID).Scan(
		&a.ID, &a.TenantID, &a.OrganizationID, &a.MonitorID, &a.ActionType, &a.Status,
		&a.Assignee, &a.EvidenceRefs,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return dpp.ComplianceAction{}, ErrNotFound
	}
	if err != nil {
		return dpp.ComplianceAction{}, err
	}
	_ = tx.Commit()
	return a, nil
}

// ListComplianceActions implements dpp.Repository.
func (r *Repository) ListComplianceActions(ctx context.Context, actor dpp.ActorContext, monitorID, status string) ([]dpp.ComplianceAction, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id, organization_id, monitor_id, action_type, status,
		       COALESCE(assignee, ''), COALESCE(evidence_refs::text, '')
		FROM compliance_action
		WHERE tenant_id = $1 AND organization_id = $2
		  AND ($3 = '' OR monitor_id = $3)
		  AND ($4 = '' OR status = $4)
		ORDER BY created_at DESC
	`
	rows, err := tx.QueryContext(ctx, query, actor.TenantID, actor.OrganizationID, monitorID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []dpp.ComplianceAction
	for rows.Next() {
		var a dpp.ComplianceAction
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.OrganizationID, &a.MonitorID, &a.ActionType, &a.Status,
			&a.Assignee, &a.EvidenceRefs,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	_ = tx.Commit()
	return list, nil
}

// UpdateComplianceAction implements dpp.Repository.
func (r *Repository) UpdateComplianceAction(ctx context.Context, actor dpp.ActorContext, a dpp.ComplianceAction) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	tx, err := pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE compliance_action
		SET status = COALESCE(NULLIF($1, ''), status),
		    assignee = COALESCE(NULLIF($2, ''), assignee),
		    evidence_refs = COALESCE(NULLIF($3, '')::jsonb, evidence_refs),
		    completed_at = CASE WHEN $1 = 'DONE' THEN $4 ELSE completed_at END,
		    updated_at = $4
		WHERE id = $5 AND tenant_id = $6 AND organization_id = $7
	`
	now := r.now().UTC()
	res, err := tx.ExecContext(ctx, query,
		a.Status, a.Assignee, a.EvidenceRefs, now,
		a.ID, actor.TenantID, actor.OrganizationID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}
