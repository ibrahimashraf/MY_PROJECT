package certificatepg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type publicBindingRule struct {
	key      string
	required bool
}

type publicAssetFacts struct {
	found       bool
	serial      string
	description string
	assetType   string
}

func materializePublicBindingSnapshot(ctx context.Context, tx *sql.Tx, actorTenant, actorOrganization, policyID, assetID, inspectionID string, inspectionRevision int64) ([]byte, error) {
	rules, err := loadPublicBindingRules(ctx, tx, actorTenant, actorOrganization, policyID)
	if err != nil {
		return nil, err
	}
	values := map[string]any{}
	if len(rules) == 0 {
		return json.Marshal(values)
	}
	var asset publicAssetFacts
	assetLoaded := false
	var scope any
	scopeLoaded := false
	for _, rule := range rules {
		switch rule.key {
		case "asset.id":
			values[rule.key] = assetID
		case "asset.serial_number", "asset.description", "asset.type":
			if !assetLoaded {
				asset, err = loadPublicAssetFacts(ctx, tx, actorTenant, actorOrganization, assetID)
				if err != nil {
					return nil, err
				}
				assetLoaded = true
			}
			if !asset.found {
				if rule.required {
					return nil, fmt.Errorf("required public asset binding %s is unavailable", rule.key)
				}
				continue
			}
			if rule.key == "asset.serial_number" {
				values[rule.key] = asset.serial
			}
			if rule.key == "asset.description" {
				values[rule.key] = asset.description
			}
			if rule.key == "asset.type" {
				values[rule.key] = asset.assetType
			}
		case "inspection.type", "inspection.public_scope":
			if !scopeLoaded {
				scope, err = loadPublicInspectionScope(ctx, tx, actorTenant, actorOrganization, inspectionID, inspectionRevision)
				if err != nil {
					return nil, err
				}
				scopeLoaded = true
			}
			if scope == nil {
				if rule.required {
					return nil, fmt.Errorf("required public inspection binding %s is unavailable", rule.key)
				}
				continue
			}
			scopeObject := scope.(map[string]any)
			if rule.key == "inspection.type" {
				values[rule.key] = scopeObject["inspection_type"]
			} else {
				values[rule.key] = scopeObject
			}
		default:
			return nil, fmt.Errorf("unsupported public binding key %s", rule.key)
		}
	}
	return json.Marshal(values)
}

func loadPublicBindingRules(ctx context.Context, tx *sql.Tx, tenantID, organizationID, policyID string) ([]publicBindingRule, error) {
	rows, err := tx.QueryContext(ctx, `SELECT binding_key,required FROM certificate_policy_public_binding WHERE tenant_id=$1 AND organization_id=$2 AND policy_id=$3 ORDER BY display_order`, tenantID, organizationID, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []publicBindingRule
	for rows.Next() {
		var rule publicBindingRule
		if err := rows.Scan(&rule.key, &rule.required); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func loadPublicAssetFacts(ctx context.Context, tx *sql.Tx, tenantID, organizationID, assetID string) (publicAssetFacts, error) {
	var facts publicAssetFacts
	err := tx.QueryRowContext(ctx, `SELECT serial_number,description,asset_type FROM asset_registry WHERE tenant_id=$1 AND organization_id=$2 AND asset_id=$3 AND lifecycle_state='ACTIVE' FOR SHARE`, tenantID, organizationID, assetID).Scan(&facts.serial, &facts.description, &facts.assetType)
	if err == sql.ErrNoRows {
		return facts, nil
	}
	if err != nil {
		return facts, err
	}
	facts.found = true
	return facts, nil
}

func loadPublicInspectionScope(ctx context.Context, tx *sql.Tx, tenantID, organizationID, inspectionID string, inspectionRevision int64) (any, error) {
	var scopeID, inspectionType, resultState string
	var taxonomyVersion int64
	err := tx.QueryRowContext(ctx, `SELECT id,inspection_type,taxonomy_version,result_state FROM inspection_public_scope WHERE tenant_id=$1 AND organization_id=$2 AND inspection_id=$3 AND inspection_revision=$4 FOR SHARE`, tenantID, organizationID, inspectionID, inspectionRevision).Scan(&scopeID, &inspectionType, &taxonomyVersion, &resultState)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT scope_code,display_label,outcome FROM inspection_public_scope_item WHERE tenant_id=$1 AND organization_id=$2 AND scope_id=$3 ORDER BY display_order`, tenantID, organizationID, scopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var code, label, outcome string
		if err := rows.Scan(&code, &label, &outcome); err != nil {
			return nil, err
		}
		items = append(items, map[string]string{"code": code, "label": label, "outcome": outcome})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"inspection_type": inspectionType, "taxonomy_version": taxonomyVersion, "result_state": resultState, "items": items}, nil
}
