package certificatepg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
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
