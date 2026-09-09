package certificatepg

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"integin/internal/domain/certificateauthority"
	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/domain/certificatetemplate"
)

// GetRenderJobInput loads the certificate, its immutable snapshot, and template definitions
// to construct the sealed domainrender.JobInput required for rendering.
func (r *Repository) GetRenderJobInput(ctx context.Context, actor certificateauthority.ActorContext, certificateID string) (domainrender.JobInput, error) {
	if err := validateActor(actor); err != nil {
		return domainrender.JobInput{}, err
	}
	if strings.TrimSpace(certificateID) == "" {
		return domainrender.JobInput{}, errors.New("certificate id is required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domainrender.JobInput{}, err
	}
	defer tx.Rollback()

	if err := setScope(ctx, tx, actor); err != nil {
		return domainrender.JobInput{}, err
	}

	query := `
		SELECT c.id, c.certificate_number, c.status, c.issued_at, c.expires_at, c.public_token_digest,
		       s.template_snapshot, s.cell_snapshot, s.public_binding_snapshot, s.snapshot_sha256
		FROM certificate_record c
		JOIN certificate_snapshot s ON s.certificate_id = c.id AND s.tenant_id = c.tenant_id AND s.organization_id = c.organization_id
		WHERE c.id = $1 AND c.tenant_id = $2 AND c.organization_id = $3
	`

	var (
		certID             string
		certNum            string
		status             string
		issuedAt           sql.NullTime
		expiresAt          sql.NullTime
		publicTokenDigest  []byte
		templateSnapshot   []byte
		cellSnapshot       []byte
		publicBindingBytes []byte
		snapshotSHA256     []byte
	)

	err = tx.QueryRowContext(ctx, query, certificateID, actor.TenantID, actor.OrganizationID).Scan(
		&certID, &certNum, &status, &issuedAt, &expiresAt, &publicTokenDigest,
		&templateSnapshot, &cellSnapshot, &publicBindingBytes, &snapshotSHA256,
	)
	if err == sql.ErrNoRows {
		return domainrender.JobInput{}, errors.New("certificate or snapshot not found")
	}
	if err != nil {
		return domainrender.JobInput{}, err
	}

	// Unmarshal template definition
	var tmplDef certificatetemplate.Definition
	if len(templateSnapshot) > 0 {
		if err := json.Unmarshal(templateSnapshot, &tmplDef); err != nil {
			return domainrender.JobInput{}, fmt.Errorf("failed to unmarshal template definition: %w", err)
		}
	}

	// Unmarshal public bindings
	var publicBindings map[string]any
	if len(publicBindingBytes) > 0 {
		if err := json.Unmarshal(publicBindingBytes, &publicBindings); err != nil {
			return domainrender.JobInput{}, fmt.Errorf("failed to unmarshal public bindings: %w", err)
		}
	} else {
		publicBindings = make(map[string]any)
	}

	// Query cells from certificate_template_cell
	cellQuery := `
		SELECT cell_id, page_number, x_points, y_points, width_points, height_points, kind,
		       COALESCE(binding_key, ''), COALESCE(static_text, ''), COALESCE(label, ''),
		       fit_policy, COALESCE(max_lines, 0), required, checkbox_values,
		       COALESCE(condition_binding_key, ''), COALESCE(condition_operator, ''), COALESCE(condition_literal, ''),
		       COALESCE(repeat_source, ''), COALESCE(max_items, 0)
		FROM certificate_template_cell
		WHERE tenant_id = $1 AND organization_id = $2 AND template_id = $3
		ORDER BY page_number ASC, cell_id ASC
	`
	rows, err := tx.QueryContext(ctx, cellQuery, actor.TenantID, actor.OrganizationID, tmplDef.ID)
	if err != nil {
		return domainrender.JobInput{}, fmt.Errorf("querying template cells: %w", err)
	}
	defer rows.Close()

	var cells []certificatetemplate.Cell
	for rows.Next() {
		var cell certificatetemplate.Cell
		var kind, fitPolicy, operator string
		var checkboxValues []byte
		var condBinding, condLiteral, repeatSource string
		err := rows.Scan(
			&cell.ID, &cell.PageNumber, &cell.Rectangle.X, &cell.Rectangle.Y, &cell.Rectangle.Width, &cell.Rectangle.Height,
			&kind, &cell.BindingKey, &cell.StaticText, &cell.Label, &fitPolicy, &cell.MaxLines, &cell.Required,
			&checkboxValues, &condBinding, &operator, &condLiteral,
			&repeatSource, &cell.MaxItems,
		)
		if err != nil {
			return domainrender.JobInput{}, fmt.Errorf("scanning cell: %w", err)
		}
		cell.Kind = certificatetemplate.PresentationKind(kind)
		cell.FitPolicy = certificatetemplate.FitPolicy(fitPolicy)
		cell.RepeatSource = certificatetemplate.BindingKey(repeatSource)
		if len(checkboxValues) > 0 && string(checkboxValues) != "null" {
			_ = json.Unmarshal(checkboxValues, &cell.CheckboxValues)
		}
		if condBinding != "" && operator != "" {
			cell.Applicability = &certificatetemplate.Applicability{
				BindingKey: certificatetemplate.BindingKey(condBinding),
				Operator:   certificatetemplate.ConditionOperator(operator),
				Literal:    condLiteral,
			}
		}
		cells = append(cells, cell)
	}
	if err := rows.Err(); err != nil {
		return domainrender.JobInput{}, err
	}

	if err := tx.Commit(); err != nil {
		return domainrender.JobInput{}, err
	}

	digestHex := hex.EncodeToString(publicTokenDigest)
	verURL := fmt.Sprintf("/verify/%s", digestHex)

	return domainrender.JobInput{
		TenantID:             actor.TenantID,
		OrganizationID:       actor.OrganizationID,
		CertificateID:        certID,
		CertificateNumber:    certNum,
		Status:               status,
		IssuedAt:             issuedAt.Time,
		ExpiresAt:            expiresAt.Time,
		PublicTokenDigestHex: digestHex,
		VerificationURL:      verURL,
		TemplateDef:          tmplDef,
		TemplateCells:        cells,
		PublicBindings:       publicBindings,
		SnapshotSHA256:       snapshotSHA256,
	}, nil
}
