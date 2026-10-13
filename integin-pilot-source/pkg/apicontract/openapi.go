// Package apicontract parses and validates HTTP request/response payloads
// against the canonical OpenAPI v1 contract (docs/api/openapi_v1.yaml) using
// canonical test vector definitions that mirror the Go domain models.
package apicontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	yaml "github.com/goccy/go-yaml"
)

// DefaultSpecPath is the canonical OpenAPI v1 contract relative to the
// repository root.
const DefaultSpecPath = "../../docs/api/openapi_v1.yaml"

// Canonical endpoint paths (match docs/api/openapi_v1.yaml).
const (
	EndpointSync       = "/api/v1/sync"
	EndpointEvidence   = "/api/v1/evidence"
	EndpointEnrollment = "/api/v1/devices/enroll"
	EndpointSession    = "/api/v1/auth/session"
)

// Canonical request schema names (match docs/api/openapi_v1.yaml components).
var endpointRequestSchema = map[string]string{
	EndpointSync:       "SyncBatchRequest",
	EndpointEvidence:   "EvidenceIngestRequest",
	EndpointEnrollment: "EnrollmentRequest",
	EndpointSession:    "SessionRequest",
}

// SchemaProp describes one JSON property of an OpenAPI schema.
type SchemaProp struct {
	Type     string
	Format   string
	Ref      string
	Enum     []string
	ItemRef  string
	ItemType string
}

// Schema is a flattened OpenAPI 3.1 component schema.
type Schema struct {
	Name       string
	Type       string
	Required   []string
	Properties map[string]SchemaProp
}

// PathOperation is a flattened HTTP operation.
type PathOperation struct {
	Path              string
	Method            string
	OperationID       string
	Summary           string
	RequestBodySchema string
	Responses         map[string]string
}

// OpenAPISpec is the loaded, typed view of docs/api/openapi_v1.yaml.
type OpenAPISpec struct {
	OpenAPIVersion string
	Title          string
	Version        string
	Paths          []PathOperation
	Schemas        map[string]*Schema
}

// LoadOpenAPISpec parses an OpenAPI 3.1 document from path.
func LoadOpenAPISpec(path string) (*OpenAPISpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read openapi spec: %w", err)
	}
	var file specFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse openapi spec %s: %w", path, err)
	}
	if !strings.HasPrefix(file.OpenAPI, "3.1") {
		return nil, fmt.Errorf("openapi_version = %q, want 3.1.x", file.OpenAPI)
	}
	if strings.TrimSpace(file.Info.Title) == "" {
		return nil, errors.New("openapi info.title is required")
	}

	spec := &OpenAPISpec{
		OpenAPIVersion: file.OpenAPI,
		Title:          file.Info.Title,
		Version:        file.Info.Version,
		Schemas:        make(map[string]*Schema, len(file.Components.Schemas)),
	}
	for name, schema := range file.Components.Schemas {
		props := make(map[string]SchemaProp, len(schema.Properties))
		for propName, prop := range schema.Properties {
			props[propName] = SchemaProp{
				Type:     prop.Type,
				Format:   prop.Format,
				Ref:      prop.Ref,
				Enum:     prop.Enum,
				ItemRef:  prop.Items.Ref,
				ItemType: prop.Items.Type,
			}
		}
		spec.Schemas[name] = &Schema{
			Name:       name,
			Type:       schema.Type,
			Required:   schema.Required,
			Properties: props,
		}
	}
	for path, ops := range file.Paths {
		for method, op := range map[string]*operationYAML{
			"get": ops.Get, "post": ops.Post, "put": ops.Put,
			"patch": ops.Patch, "delete": ops.Delete,
		} {
			if op == nil {
				continue
			}
			po := PathOperation{
				Path:        path,
				Method:      method,
				OperationID: op.OperationID,
				Summary:     op.Summary,
				Responses:   make(map[string]string, len(op.Responses)),
			}
			if content, ok := op.RequestBody.Content["application/json"]; ok {
				po.RequestBodySchema = schemaName(content.Schema.Ref)
			}
			for code, resp := range op.Responses {
				if content, ok := resp.Content["application/json"]; ok {
					po.Responses[code] = schemaName(content.Schema.Ref)
				}
			}
			spec.Paths = append(spec.Paths, po)
		}
	}
	return spec, nil
}

func schemaName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ref
}

// Operation returns the flattened operation for method+path, if present.
func (s *OpenAPISpec) Operation(method, path string) (*PathOperation, bool) {
	for i := range s.Paths {
		if s.Paths[i].Path == path && s.Paths[i].Method == method {
			return &s.Paths[i], true
		}
	}
	return nil, false
}

// RequestSchema returns the named component schema, or nil.
func (s *OpenAPISpec) RequestSchema(name string) *Schema {
	return s.Schemas[name]
}

// ContractValidator validates request payloads against the canonical contract.
type ContractValidator interface {
	ValidateSyncPayload(payload []byte) error
	ValidateEvidencePayload(payload []byte) error
	ValidateEnrollmentPayload(payload []byte) error
	ValidateSessionPayload(payload []byte) error
}

// RequestValidator enforces the canonical contract for the four core endpoints.
type RequestValidator struct {
	spec *OpenAPISpec
}

// NewRequestValidator binds a validator to a loaded spec, failing when a core
// endpoint schema is missing.
func NewRequestValidator(spec *OpenAPISpec) (*RequestValidator, error) {
	if spec == nil {
		return nil, errors.New("openapi spec is required")
	}
	for endpoint, schemaName := range endpointRequestSchema {
		if spec.RequestSchema(schemaName) == nil {
			return nil, fmt.Errorf("spec missing request schema %q for %s", schemaName, endpoint)
		}
	}
	return &RequestValidator{spec: spec}, nil
}

// NewRequestValidatorFromSpecFile loads the canonical spec and binds a validator.
func NewRequestValidatorFromSpecFile(path string) (*RequestValidator, error) {
	spec, err := LoadOpenAPISpec(path)
	if err != nil {
		return nil, err
	}
	return NewRequestValidator(spec)
}

// Spec exposes the loaded OpenAPI document.
func (v *RequestValidator) Spec() *OpenAPISpec { return v.spec }

// SchemaForEndpoint returns the request schema bound to an endpoint.
func (v *RequestValidator) SchemaForEndpoint(endpoint string) (*Schema, bool) {
	name, ok := endpointRequestSchema[endpoint]
	if !ok {
		return nil, false
	}
	return v.spec.RequestSchema(name), true
}

// ValidateSyncPayload validates a POST /api/v1/sync request body.
func (v *RequestValidator) ValidateSyncPayload(payload []byte) error {
	return v.validate(payload, &SyncVector{}, syncRequiredFields())
}

// ValidateEvidencePayload validates a POST /api/v1/evidence request body.
func (v *RequestValidator) ValidateEvidencePayload(payload []byte) error {
	return v.validate(payload, &EvidenceVector{}, evidenceRequiredFields())
}

// ValidateEnrollmentPayload validates a POST /api/v1/devices/enroll request body.
func (v *RequestValidator) ValidateEnrollmentPayload(payload []byte) error {
	return v.validate(payload, &EnrollmentVector{}, enrollmentRequiredFields())
}

// ValidateSessionPayload validates a POST /api/v1/auth/session request body.
func (v *RequestValidator) ValidateSessionPayload(payload []byte) error {
	return v.validate(payload, &SessionVector{}, sessionRequiredFields())
}

func (v *RequestValidator) validate(payload []byte, dst any, required []string) error {
	if len(bytes.TrimSpace(payload)) == 0 {
		return &ValidationError{Message: "request body is required"}
	}
	if err := checkRequiredFields(payload, required); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return wrapDecodeError(err)
	}
	return nil
}

// ValidationError reports a contract violation with the offending field path.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements error.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("payload invalid: field %q: %s", e.Field, e.Message)
	}
	return "payload invalid: " + e.Message
}

func wrapDecodeError(err error) error {
	text := err.Error()
	const unknownField = "json: unknown field "
	if strings.HasPrefix(text, unknownField) {
		return &ValidationError{Field: strings.Trim(text[len(unknownField):], `"`), Message: "unknown field is not permitted"}
	}
	const structField = "Go struct field "
	if i := strings.Index(text, structField); i >= 0 {
		rest := text[i+len(structField):]
		field := rest
		if j := strings.Index(rest, " "); j >= 0 {
			field = rest[:j]
		}
		if dot := strings.Index(field, "."); dot >= 0 {
			field = field[dot+1:]
		}
		return &ValidationError{Field: field, Message: err.Error()}
	}
	return &ValidationError{Message: err.Error()}
}

func checkRequiredFields(payload []byte, required []string) error {
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return &ValidationError{Message: err.Error()}
	}
	for _, name := range required {
		value, present := obj[name]
		if !present {
			return &ValidationError{Field: name, Message: "required field is missing"}
		}
		if value == nil {
			return &ValidationError{Field: name, Message: "required field must not be null"}
		}
	}
	return nil
}

// SyncVector is the canonical POST /api/v1/sync request payload mirroring
// internal/domain/sync.DeviceProof and mutation batch state.
type SyncVector struct {
	BatchID        string         `json:"batch_id"`
	TenantID       string         `json:"tenant_id"`
	OrganizationID string         `json:"organization_id"`
	DeviceID       string         `json:"device_id"`
	AuthorityID    string         `json:"authority_id"`
	AuthorityEpoch int64          `json:"authority_epoch"`
	Sequence       int64          `json:"sequence"`
	Mutations      []SyncMutation `json:"mutations"`
	Signature      string         `json:"signature"`
}

// SyncMutation is one entity mutation inside a batch.
type SyncMutation struct {
	MutationID string          `json:"mutation_id"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Operation  string          `json:"operation"`
	Payload    json.RawMessage `json:"payload"`
	At         time.Time       `json:"at"`
	Version    int64           `json:"version"`
}

// EvidenceVector is the canonical POST /api/v1/evidence request payload
// mirroring internal/domain/evidencepack.PackRecord.
type EvidenceVector struct {
	EvidenceID          string    `json:"evidence_id"`
	TenantID            string    `json:"tenant_id"`
	OrganizationID      string    `json:"organization_id"`
	InspectionID        string    `json:"inspection_id"`
	ObjectKey           string    `json:"object_key"`
	ContentType         string    `json:"content_type"`
	CapturedAt          time.Time `json:"captured_at"`
	PlaintextSHA256     string    `json:"plaintext_sha256"`
	CiphertextSHA256    string    `json:"ciphertext_sha256"`
	ByteLength          int64     `json:"byte_length"`
	EncryptionAlgorithm string    `json:"encryption_algorithm"`
	DeviceID            string    `json:"device_id"`
	AuthorityID         string    `json:"authority_id"`
	AuthorityEpoch      int64     `json:"authority_epoch"`
	Signature           string    `json:"signature"`
}

// EnrollmentVector is the canonical POST /api/v1/devices/enroll payload
// mirroring internal/domain/device_trust.EnrollmentRequest.
type EnrollmentVector struct {
	RequestID      string          `json:"request_id"`
	TenantID       string          `json:"tenant_id"`
	OrganizationID string          `json:"organization_id"`
	UserID         string          `json:"user_id"`
	DeviceID       string          `json:"device_id"`
	PublicKey      string          `json:"public_key"`
	Nonce          string          `json:"nonce"`
	Signature      string          `json:"signature"`
	RequestedAt    time.Time       `json:"requested_at"`
	Attestation    attestationUnit `json:"attestation,omitempty"`
}

// attestationUnit mirrors the Attestation component schema.
type attestationUnit struct {
	Type    string `json:"type"`
	Issuer  string `json:"issuer"`
	Receipt string `json:"receipt"`
}

// SessionVector is the canonical POST /api/v1/auth/session payload mirroring
// the sync device-proof challenge exchange.
type SessionVector struct {
	SessionID   string    `json:"session_id"`
	DeviceID    string    `json:"device_id"`
	AuthorityID string    `json:"authority_id"`
	Challenge   string    `json:"challenge"`
	Signature   string    `json:"signature"`
	RequestedAt time.Time `json:"requested_at"`
}

// unexported helper aliases removed: validators decode into the exported
// canonical vectors directly.

func syncRequiredFields() []string {
	return []string{"batch_id", "tenant_id", "organization_id", "device_id", "authority_id", "authority_epoch", "sequence", "mutations", "signature"}
}

func evidenceRequiredFields() []string {
	return []string{"evidence_id", "tenant_id", "organization_id", "inspection_id", "object_key", "content_type", "captured_at", "plaintext_sha256", "ciphertext_sha256", "byte_length", "encryption_algorithm", "device_id", "authority_id", "authority_epoch", "signature"}
}

func enrollmentRequiredFields() []string {
	return []string{"request_id", "tenant_id", "organization_id", "user_id", "device_id", "public_key", "nonce", "signature", "requested_at"}
}

func sessionRequiredFields() []string {
	return []string{"session_id", "device_id", "authority_id", "challenge", "signature", "requested_at"}
}

// yaml document types -------------------------------------------------------

type specFile struct {
	OpenAPI    string             `yaml:"openapi"`
	Info       infoYAML           `yaml:"info"`
	Paths      map[string]pathOps `yaml:"paths"`
	Components componentsYAML     `yaml:"components"`
}

type infoYAML struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type pathOps struct {
	Get    *operationYAML `yaml:"get"`
	Post   *operationYAML `yaml:"post"`
	Put    *operationYAML `yaml:"put"`
	Patch  *operationYAML `yaml:"patch"`
	Delete *operationYAML `yaml:"delete"`
}

type operationYAML struct {
	OperationID string                  `yaml:"operationId"`
	Summary     string                  `yaml:"summary"`
	RequestBody bodyYAML                `yaml:"requestBody"`
	Responses   map[string]responseYAML `yaml:"responses"`
}

type bodyYAML struct {
	Content map[string]contentYAML `yaml:"content"`
}

type responseYAML struct {
	Description string                 `yaml:"description"`
	Content     map[string]contentYAML `yaml:"content"`
}

type contentYAML struct {
	Schema schemaRefYAML `yaml:"schema"`
}

type schemaRefYAML struct {
	Ref string `yaml:"$ref"`
}

type componentsYAML struct {
	Schemas map[string]schemaYAML `yaml:"schemas"`
}

type schemaYAML struct {
	Type       string              `yaml:"type"`
	Required   []string            `yaml:"required"`
	Properties map[string]propYAML `yaml:"properties"`
}

type propYAML struct {
	Type   string    `yaml:"type"`
	Format string    `yaml:"format"`
	Ref    string    `yaml:"$ref"`
	Enum   []string  `yaml:"enum"`
	Items  itemsYAML `yaml:"items"`
}

type itemsYAML struct {
	Type string `yaml:"type"`
	Ref  string `yaml:"$ref"`
}
