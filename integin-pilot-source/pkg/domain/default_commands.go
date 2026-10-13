package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCustodyRejected = errors.New("custody transfer validation rejected")
)

// RegisterDefaultCommands binds core production business handlers to the domain Dispatcher.
func RegisterDefaultCommands(d *Dispatcher) {
	// 1. Asset Custody Transfer Command: "passport.custody.transfer"
	d.Register("passport.custody.transfer", func(ctx context.Context, cmd Command) (json.RawMessage, error) {
		var req struct {
			AssetDID         string `json:"asset_did"`
			RecipientTenantID string `json:"recipient_tenant_id"`
			CountryCodeISO2  string `json:"country_code_iso2"`
			AuthorizedBy     string `json:"authorized_by"`
		}
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
		}

		if req.AssetDID == "" || req.RecipientTenantID == "" {
			return nil, ErrCustodyRejected
		}

		// Perform state transition
		event := CustodyTransferEvent{
			EventID:          fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			PreviousTenantID: cmd.TenantID,
			NewTenantID:      req.RecipientTenantID,
			CountryCodeISO2:  req.CountryCodeISO2,
			TransferDate:     time.Now().UTC(),
			AuthorizedBy:     req.AuthorizedBy,
		}

		return json.Marshal(map[string]interface{}{
			"asset_did": req.AssetDID,
			"status":    "TRANSFERRED",
			"event":     event,
		})
	})

	// 2. Asset Status Update Command: "passport.status.update"
	d.Register("passport.status.update", func(ctx context.Context, cmd Command) (json.RawMessage, error) {
		var req struct {
			AssetDID string      `json:"asset_did"`
			Status   AssetStatus `json:"status"`
			Reason   string      `json:"reason"`
		}
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
		}

		if req.AssetDID == "" || req.Status == "" {
			return nil, fmt.Errorf("%w: asset_did and status are required", ErrInvalidParams)
		}

		return json.Marshal(map[string]interface{}{
			"asset_did": req.AssetDID,
			"status":    req.Status,
			"updated":   true,
			"at":        time.Now().UTC(),
		})
	})

	// 3. Evidence Export Trigger: "evidence.export.package"
	d.Register("evidence.export.package", func(ctx context.Context, cmd Command) (json.RawMessage, error) {
		var req struct {
			ExportID   string   `json:"export_id"`
			EvidenceIDs []string `json:"evidence_ids"`
		}
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
		}

		if req.ExportID == "" {
			req.ExportID = fmt.Sprintf("exp-%d", time.Now().UnixNano())
		}

		return json.Marshal(map[string]interface{}{
			"export_id":      req.ExportID,
			"tenant_id":      cmd.TenantID,
			"evidence_count": len(req.EvidenceIDs),
			"state":          "QUEUED",
		})
	})
}
