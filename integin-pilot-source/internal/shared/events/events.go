package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"integin/internal/shared/types"
)

// Envelope is an immutable domain-event value. Its fields are private and its
// payload accessor returns a defensive copy, so callers cannot mutate an event
// after construction or deserialization.
type Envelope struct {
	eventID        string
	eventType      string
	schemaVersion  int
	tenantID       string
	organizationID string
	environment    string
	aggregateID    string
	aggregateType  string
	actorID        string
	correlationID  string
	causationID    string
	occurredAt     time.Time
	payload        []byte
}

type envelopeJSON struct {
	EventID        string          `json:"event_id"`
	EventType      string          `json:"event_type"`
	SchemaVersion  int             `json:"schema_version"`
	TenantID       string          `json:"tenant_id"`
	OrganizationID string          `json:"organization_id"`
	Environment    string          `json:"environment"`
	AggregateID    string          `json:"aggregate_id"`
	AggregateType  string          `json:"aggregate_type"`
	ActorID        string          `json:"actor_id,omitempty"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	CausationID    string          `json:"causation_id,omitempty"`
	OccurredAt     time.Time       `json:"occurred_at"`
	Payload        json.RawMessage `json:"payload"`
}

func NewEnvelope(eventID, eventType, tenantID, organizationID, environment, aggregateType, aggregateID string, payload any, occurredAt time.Time) (Envelope, error) {
	if strings.TrimSpace(eventID) == "" || strings.TrimSpace(eventType) == "" {
		return Envelope{}, errors.New("event_id and event_type are required")
	}
	context := types.TenantContext{TenantID: tenantID, OrganizationID: organizationID, Environment: environment}
	if err := context.Validate(); err != nil {
		return Envelope{}, err
	}
	if strings.TrimSpace(aggregateType) == "" || strings.TrimSpace(aggregateID) == "" {
		return Envelope{}, errors.New("aggregate type and id are required")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		eventID: eventID, eventType: eventType, schemaVersion: 1,
		tenantID: tenantID, organizationID: organizationID, environment: environment,
		aggregateType: aggregateType, aggregateID: aggregateID,
		occurredAt: occurredAt.UTC(), payload: append([]byte(nil), raw...),
	}, nil
}

func (e Envelope) EventID() string        { return e.eventID }
func (e Envelope) EventType() string      { return e.eventType }
func (e Envelope) SchemaVersion() int     { return e.schemaVersion }
func (e Envelope) TenantID() string       { return e.tenantID }
func (e Envelope) OrganizationID() string { return e.organizationID }
func (e Envelope) Environment() string    { return e.environment }
func (e Envelope) AggregateID() string    { return e.aggregateID }
func (e Envelope) AggregateType() string  { return e.aggregateType }
func (e Envelope) ActorID() string        { return e.actorID }
func (e Envelope) CorrelationID() string  { return e.correlationID }
func (e Envelope) CausationID() string    { return e.causationID }
func (e Envelope) OccurredAt() time.Time  { return e.occurredAt }
func (e Envelope) Payload() []byte        { return append([]byte(nil), e.payload...) }

func (e Envelope) MarshalJSON() ([]byte, error) {
	payload := append(json.RawMessage(nil), e.payload...)
	if len(payload) == 0 {
		payload = json.RawMessage("null")
	}
	return json.Marshal(envelopeJSON{
		EventID: e.eventID, EventType: e.eventType, SchemaVersion: e.schemaVersion,
		TenantID: e.tenantID, OrganizationID: e.organizationID, Environment: e.environment,
		AggregateID: e.aggregateID, AggregateType: e.aggregateType, ActorID: e.actorID,
		CorrelationID: e.correlationID, CausationID: e.causationID, OccurredAt: e.occurredAt,
		Payload: payload,
	})
}

func (e *Envelope) UnmarshalJSON(data []byte) error {
	var wire envelopeJSON
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if strings.TrimSpace(wire.EventID) == "" || strings.TrimSpace(wire.EventType) == "" {
		return errors.New("event_id and event_type are required")
	}
	context := types.TenantContext{TenantID: wire.TenantID, OrganizationID: wire.OrganizationID, Environment: wire.Environment}
	if err := context.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(wire.AggregateType) == "" || strings.TrimSpace(wire.AggregateID) == "" {
		return errors.New("aggregate type and id are required")
	}
	if wire.SchemaVersion <= 0 {
		return errors.New("schema_version must be positive")
	}
	e.eventID, e.eventType, e.schemaVersion = wire.EventID, wire.EventType, wire.SchemaVersion
	e.tenantID, e.organizationID, e.environment = wire.TenantID, wire.OrganizationID, wire.Environment
	e.aggregateID, e.aggregateType = wire.AggregateID, wire.AggregateType
	e.actorID, e.correlationID, e.causationID = wire.ActorID, wire.CorrelationID, wire.CausationID
	e.occurredAt, e.payload = wire.OccurredAt.UTC(), append([]byte(nil), wire.Payload...)
	return nil
}

func (e Envelope) EqualPayload(other Envelope) bool { return bytes.Equal(e.payload, other.payload) }

const (
	InspectionStarted             = "InspectionStarted"
	FindingRecorded               = "FindingRecorded"
	InspectionSubmitted           = "InspectionSubmitted"
	InspectionReturned            = "InspectionReturned"
	InspectionApproved            = "InspectionApproved"
	CertificateIssued             = "CertificateIssued"
	CertificateRevoked            = "CertificateRevoked"
	AssetMoved                    = "AssetMoved"
	DeviceRevoked                 = "DeviceRevoked"
	AIInsightGenerated            = "AIInsightGenerated"
	FeatureFlagChanged            = "FeatureFlagChanged"
	NotificationPreferenceChanged = "NotificationPreferenceChanged"
	EmailDeliveryStatusChanged    = "EmailDeliveryStatusChanged"
	CertificateVerified           = "CertificateVerified"
	QRVerificationRequested       = "QRVerificationRequested"
	CalibrationRecorded           = "CalibrationRecorded"
	CalibrationExpired            = "CalibrationExpired"
	PressureTestRecorded          = "PressureTestRecorded"
)

type FindingRecordedPayload struct {
	InspectionID string `json:"inspection_id"`
	AssetID      string `json:"asset_id"`
	ItemID       string `json:"item_id"`
	Response     string `json:"response"`
	Severity     string `json:"severity,omitempty"`
}
type InspectionSubmittedPayload struct {
	InspectionID  string `json:"inspection_id"`
	Revision      int    `json:"revision"`
	OverallResult string `json:"overall_result"`
}
type CertificateIssuedPayload struct {
	CertificateID     string `json:"certificate_id"`
	CertificateNumber string `json:"certificate_number"`
	Revision          int    `json:"revision"`
}
type AIInsightPayload struct {
	InsightID string `json:"insight_id"`
	Lens      string `json:"lens"`
	Blocking  bool   `json:"blocking"`
}
