package certificatepg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"integin/internal/domain/certificatetemplate"
)

type PublicProjection struct {
	CertificateNumber string
	Status            string
	IssuedAt          time.Time
	ExpiresAt         time.Time
	AssetID           string
	AssetSerialNumber string
	AssetDescription  string
	AssetType         string
	InspectionType    string
	TestScope         json.RawMessage
}

type RenderData struct {
	CertificateID         string
	CertificateNumber     string
	Status                string
	IssuedAt              time.Time
	ExpiresAt             time.Time
	TemplateSnapshot      []byte
	CellSnapshot          []byte
	PublicBindingSnapshot []byte
	SnapshotSHA256        []byte
	TemplateCells         []certificatetemplate.Cell
	TemplateDef           certificatetemplate.Definition
	PageCount             int
	PageWidth             float64
	PageHeight            float64
	QRPage                int
	QRRectangle           certificatetemplate.Rectangle
}

func (r *Repository) VerifyPublic(ctx context.Context, rawToken string) (PublicProjection, bool, error) {
	rawToken = strings.TrimSpace(rawToken)
	if len(rawToken) < 32 || len(rawToken) > 128 {
		return PublicProjection{}, false, nil
	}
	digest := sha256.Sum256([]byte(rawToken))
	var projection PublicProjection
	var bindingSnapshot []byte
	err := r.db.QueryRowContext(ctx, `SELECT c.certificate_number,c.status,c.issued_at,c.expires_at,c.asset_id,COALESCE(s.public_binding_snapshot,'{}'::jsonb) FROM certificate_record c LEFT JOIN certificate_snapshot s ON s.certificate_id=c.id WHERE c.public_token_digest=$1 AND c.status IN ('ISSUED','EXPIRED','REVOKED','SUPERSEDED')`, digest[:]).Scan(&projection.CertificateNumber, &projection.Status, &projection.IssuedAt, &projection.ExpiresAt, &projection.AssetID, &bindingSnapshot)
	if err == sql.ErrNoRows {
		return PublicProjection{}, false, nil
	}
	if err != nil {
		return PublicProjection{}, false, err
	}
	var bindings map[string]json.RawMessage
	if err := json.Unmarshal(bindingSnapshot, &bindings); err != nil {
		return PublicProjection{}, false, err
	}
	_ = json.Unmarshal(bindings["asset.serial_number"], &projection.AssetSerialNumber)
	_ = json.Unmarshal(bindings["asset.description"], &projection.AssetDescription)
	_ = json.Unmarshal(bindings["asset.type"], &projection.AssetType)
	_ = json.Unmarshal(bindings["inspection.type"], &projection.InspectionType)
	if scope, ok := bindings["inspection.public_scope"]; ok && string(scope) != "null" {
		projection.TestScope = append(json.RawMessage(nil), scope...)
	}
	return projection, true, nil
}

func (r *Repository) GetRenderData(ctx context.Context, rawToken string) (RenderData, bool, error) {
	rawToken = strings.TrimSpace(rawToken)
	if len(rawToken) < 32 || len(rawToken) > 128 {
		return RenderData{}, false, nil
	}
	digest := sha256.Sum256([]byte(rawToken))

	var data RenderData
	var templateSnapshot, cellSnapshot, publicBindingSnapshot, snapshotSHA256 []byte
	var templateID string
	var templateCode string
	var templateVersion int64
	var catalogVersion int64

	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.certificate_number, c.status, c.issued_at, c.expires_at,
		       c.template_code, c.template_version,
		       s.template_snapshot, s.cell_snapshot, s.public_binding_snapshot, s.snapshot_sha256,
		       t.catalog_version, t.page_count, t.page_width_points, t.page_height_points,
		       t.id
		FROM certificate_record c
		LEFT JOIN certificate_snapshot s ON s.certificate_id = c.id
		LEFT JOIN certificate_template t ON t.tenant_id = c.tenant_id AND t.organization_id = c.organization_id AND t.template_code = c.template_code AND t.version = c.template_version
		WHERE c.public_token_digest = $1 AND c.status IN ('ISSUED','EXPIRED','REVOKED','SUPERSEDED')
	`, digest[:]).Scan(
		&data.CertificateID, &data.CertificateNumber, &data.Status, &data.IssuedAt, &data.ExpiresAt,
		&templateCode, &templateVersion,
		&templateSnapshot, &cellSnapshot, &publicBindingSnapshot, &snapshotSHA256,
		&catalogVersion, &data.PageCount, &data.PageWidth, &data.PageHeight,
		&templateID,
	)
	if err == sql.ErrNoRows {
		return RenderData{}, false, nil
	}
	if err != nil {
		return RenderData{}, false, err
	}

	data.TemplateSnapshot = templateSnapshot
	data.CellSnapshot = cellSnapshot
	data.PublicBindingSnapshot = publicBindingSnapshot
	data.SnapshotSHA256 = snapshotSHA256

	rows, err := r.db.QueryContext(ctx, `
		SELECT cell_id, page_number, x_points, y_points, width_points, height_points,
		       kind, binding_key, static_text, label, fit_policy, max_lines, required,
		       checkbox_values, condition_binding_key, condition_operator, condition_literal,
		       repeat_source, max_items
		FROM certificate_template_cell
		WHERE tenant_id = (SELECT tenant_id FROM certificate_record WHERE id = $1)
		  AND organization_id = (SELECT organization_id FROM certificate_record WHERE id = $1)
		  AND template_id = $2
		ORDER BY page_number, cell_id
	`, data.CertificateID, templateID)
	if err != nil {
		return RenderData{}, false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cell certificatetemplate.Cell
		var checkboxValues []byte
		err := rows.Scan(
			&cell.ID, &cell.PageNumber, &cell.Rectangle.X, &cell.Rectangle.Y, &cell.Rectangle.Width, &cell.Rectangle.Height,
			&cell.Kind, &cell.BindingKey, &cell.StaticText, &cell.Label, &cell.FitPolicy, &cell.MaxLines, &cell.Required,
			&checkboxValues, &cell.Applicability, &cell.Applicability, &cell.Applicability,
			&cell.RepeatSource, &cell.MaxItems,
		)
		if err != nil {
			return RenderData{}, false, err
		}
		if len(checkboxValues) > 0 {
			if err := json.Unmarshal(checkboxValues, &cell.CheckboxValues); err != nil {
				return RenderData{}, false, err
			}
		}
		data.TemplateCells = append(data.TemplateCells, cell)
	}
	if err := rows.Err(); err != nil {
		return RenderData{}, false, err
	}

	var templateDef certificatetemplate.Definition
	if err := json.Unmarshal(templateSnapshot, &templateDef); err != nil {
		return RenderData{}, false, err
	}
	data.TemplateDef = templateDef

	data.QRPage = 1
	data.QRRectangle = certificatetemplate.Rectangle{X: 140, Y: 20, Width: 40, Height: 40}

	return data, true, nil
}
