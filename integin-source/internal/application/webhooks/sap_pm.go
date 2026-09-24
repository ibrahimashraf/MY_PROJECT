package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"integin/internal/domain/assurance"
)

// SAPWorkOrderRequest represents the outbound JSON payload expected by SAP PM.
type SAPWorkOrderRequest struct {
	EquipmentID string `json:"equipment_id"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	ReportedBy  string `json:"reported_by"`
	ReportedAt  string `json:"reported_at"`
}

// EnterpriseWebhookService handles outbound system integrations.
type EnterpriseWebhookService interface {
	DispatchAssuranceState(ctx context.Context, tenantID, assetID uuid.UUID, state assurance.AssuranceState, reasons json.RawMessage) error
}

type sapWebhookService struct {
	client  *http.Client
	baseURL string
}

func NewSAPWebhookService(baseURL string) EnterpriseWebhookService {
	return &sapWebhookService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (s *sapWebhookService) DispatchAssuranceState(ctx context.Context, tenantID, assetID uuid.UUID, state assurance.AssuranceState, reasons json.RawMessage) error {
	// Trigger outbound Work Orders for critical safety failures AND provisional safety failures trapped in Triage
	if state != assurance.StateNonCompliant && state != assurance.StateCondemned && state != assurance.StateProvisionallyNonCompliant {
		return nil // No-op for routine states
	}

	payload := SAPWorkOrderRequest{
		EquipmentID: assetID.String(),
		Priority:    "URGENT",
		Description: fmt.Sprintf("INTEGIN Assurance trigger: Asset declared %s. Reasons: %s", state, string(reasons)),
		ReportedBy:  "INTEGIN_ASSURANCE_SYSTEM",
		ReportedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1/sap/work-orders", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to build SAP PM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String()) // Pass tenant context to middleware

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to dispatch SAP PM webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SAP PM webhook rejected with status: %d", resp.StatusCode)
	}

	return nil
}
