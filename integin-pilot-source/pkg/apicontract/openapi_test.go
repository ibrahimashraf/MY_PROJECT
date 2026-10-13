package apicontract

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func loadTestValidator(t *testing.T) *RequestValidator {
	t.Helper()
	v, err := NewRequestValidatorFromSpecFile("../../docs/api/openapi_v1.yaml")
	if err != nil {
		t.Fatalf("NewRequestValidatorFromSpecFile: %v", err)
	}
	return v
}

func TestLoadOpenAPISpec(t *testing.T) {
	spec, err := LoadOpenAPISpec(DefaultSpecPath)
	if err != nil {
		t.Fatalf("LoadOpenAPISpec: %v", err)
	}
	if !strings.HasPrefix(spec.OpenAPIVersion, "3.1") {
		t.Fatalf("OpenAPIVersion = %q, want 3.1.x", spec.OpenAPIVersion)
	}
	wantOps := []struct{ method, path string }{
		{"post", EndpointSync},
		{"post", EndpointEvidence},
		{"post", EndpointEnrollment},
		{"post", EndpointSession},
	}
	for _, op := range wantOps {
		got, ok := spec.Operation(op.method, op.path)
		if !ok {
			t.Fatalf("missing %s %s", op.method, op.path)
		}
		if got.RequestBodySchema == "" {
			t.Fatalf("%s %s must declare a request body schema", op.method, op.path)
		}
	}
	wantResponseCodes := map[string][]string{
		EndpointSync:       {"200", "400", "401", "409"},
		EndpointEvidence:   {"201", "400", "401", "403"},
		EndpointEnrollment: {"200", "400", "401"},
		EndpointSession:    {"200", "401", "403"},
	}
	for endpoint, codes := range wantResponseCodes {
		op, _ := spec.Operation("post", endpoint)
		for _, code := range codes {
			if _, ok := op.Responses[code]; !ok {
				t.Fatalf("%s must declare response %s", endpoint, code)
			}
		}
	}
}

func TestSpecSchemaMatchesCanonicalVectors(t *testing.T) {
	spec, err := LoadOpenAPISpec(DefaultSpecPath)
	if err != nil {
		t.Fatalf("LoadOpenAPISpec: %v", err)
	}
	vectors := []struct {
		endpoint string
		vec      any
	}{
		{EndpointSync, &SyncVector{}},
		{EndpointEvidence, &EvidenceVector{}},
		{EndpointEnrollment, &EnrollmentVector{}},
		{EndpointSession, &SessionVector{}},
	}
	for _, pair := range vectors {
		schema := spec.RequestSchema(endpointRequestSchema[pair.endpoint])
		if schema == nil {
			t.Fatalf("missing schema for %s", pair.endpoint)
		}
		fieldNames := jsonFieldNames(t, pair.vec)
		for _, required := range schema.Required {
			if !fieldNames[required] {
				t.Fatalf("schema %s requires %q but %T has no such field", schema.Name, required, pair.vec)
			}
		}
	}
}

func jsonFieldNames(t *testing.T, v any) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		t.Fatalf("expected struct, got %s", typ.Kind())
	}
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		names[strings.Split(tag, ",")[0]] = true
	}
	return names
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func vectorToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func validSync(t *testing.T) SyncVector {
	t.Helper()
	return SyncVector{
		BatchID:        "batch_1",
		TenantID:       "tenant-a",
		OrganizationID: "org-a",
		DeviceID:       "dev_1",
		AuthorityID:    "auth_1",
		AuthorityEpoch: 7,
		Sequence:       12,
		Mutations: []SyncMutation{{
			MutationID: "m_1",
			EntityType: "inspection_item",
			EntityID:   "ii_1",
			Operation:  "update",
			Payload:    json.RawMessage(`{"status":"complete"}`),
			At:         time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC),
			Version:    3,
		}},
		Signature: "signed-batch",
	}
}

func validEvidence(t *testing.T) EvidenceVector {
	t.Helper()
	return EvidenceVector{
		EvidenceID:          "ev_1001",
		TenantID:            "tenant-a",
		OrganizationID:      "org-a",
		InspectionID:        "ins_1",
		ObjectKey:           "exports/tenant-a/ev_1001.enc",
		ContentType:         "application/octet-stream",
		CapturedAt:          time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC),
		PlaintextSHA256:     strings.Repeat("a", 64),
		CiphertextSHA256:    strings.Repeat("b", 64),
		ByteLength:          4096,
		EncryptionAlgorithm: "AES-256-GCM",
		DeviceID:            "dev_1",
		AuthorityID:         "auth_1",
		AuthorityEpoch:      7,
		Signature:           "signed-evidence",
	}
}

func validEnrollment(t *testing.T) EnrollmentVector {
	t.Helper()
	return EnrollmentVector{
		RequestID:      "req_1",
		TenantID:       "tenant-a",
		OrganizationID: "org-a",
		UserID:         "u_42",
		DeviceID:       "dev_1",
		PublicKey:      strings.Repeat("01", 32),
		Nonce:          "challenge-nonce",
		Signature:      "pop-signature",
		RequestedAt:    time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC),
	}
}

func validSession(t *testing.T) SessionVector {
	t.Helper()
	return SessionVector{
		SessionID:   "sess_1",
		DeviceID:    "dev_1",
		AuthorityID: "auth_1",
		Challenge:   "server-challenge",
		Signature:   "signed-challenge",
		RequestedAt: time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC),
	}
}

func TestValidateValidVectors(t *testing.T) {
	v := loadTestValidator(t)
	cases := []struct {
		name string
		run  func([]byte) error
		vec  any
	}{
		{"sync", v.ValidateSyncPayload, validSync(t)},
		{"evidence", v.ValidateEvidencePayload, validEvidence(t)},
		{"enrollment", v.ValidateEnrollmentPayload, validEnrollment(t)},
		{"session", v.ValidateSessionPayload, validSession(t)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(mustJSON(t, tc.vec)); err != nil {
				t.Fatalf("valid vector rejected: %v", err)
			}
		})
	}
}

func TestRequestValidatorImplementsContractValidator(t *testing.T) {
	v := loadTestValidator(t)
	var _ ContractValidator = v
	if err := v.ValidateSyncPayload(mustJSON(t, validSync(t))); err != nil {
		t.Fatalf("interface implementation: %v", err)
	}
}

func TestValidateRejectsEmptyBody(t *testing.T) {
	v := loadTestValidator(t)
	err := v.ValidateSessionPayload(nil)
	if err == nil {
		t.Fatal("empty body must be rejected")
	}
	if !strings.Contains(err.Error(), "request body is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMissingRequiredFields(t *testing.T) {
	v := loadTestValidator(t)
	cases := []struct {
		name      string
		run       func([]byte) error
		payload   map[string]any
		wantField string
	}{
		{"sync missing tenant_id", v.ValidateSyncPayload, dropField(t, validSync(t), "tenant_id"), "tenant_id"},
		{"sync missing mutations", v.ValidateSyncPayload, dropField(t, validSync(t), "mutations"), "mutations"},
		{"evidence missing object_key", v.ValidateEvidencePayload, dropField(t, validEvidence(t), "object_key"), "object_key"},
		{"evidence missing byte_length", v.ValidateEvidencePayload, dropField(t, validEvidence(t), "byte_length"), "byte_length"},
		{"enrollment missing public_key", v.ValidateEnrollmentPayload, dropField(t, validEnrollment(t), "public_key"), "public_key"},
		{"enrollment missing nonce", v.ValidateEnrollmentPayload, dropField(t, validEnrollment(t), "nonce"), "nonce"},
		{"session missing challenge", v.ValidateSessionPayload, dropField(t, validSession(t), "challenge"), "challenge"},
		{"session missing signature", v.ValidateSessionPayload, dropField(t, validSession(t), "signature"), "signature"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(mustJSON(t, tc.payload))
			if err == nil {
				t.Fatalf("payload missing %q must be rejected", tc.wantField)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			if ve.Field != tc.wantField {
				t.Fatalf("field = %q, want %q (%s)", ve.Field, tc.wantField, err)
			}
			if !strings.Contains(err.Error(), "required field is missing") {
				t.Fatalf("unexpected message: %v", err)
			}
		})
	}
}

func TestValidateNullRequiredField(t *testing.T) {
	v := loadTestValidator(t)
	m := vectorToMap(t, validSession(t))
	m["challenge"] = nil
	err := v.ValidateSessionPayload(mustJSON(t, m))
	if err == nil {
		t.Fatal("null required field must be rejected")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "challenge" {
		t.Fatalf("expected challenge field error, got %v", err)
	}
	if !strings.Contains(err.Error(), "must not be null") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestValidateWrongTypes(t *testing.T) {
	v := loadTestValidator(t)
	cases := []struct {
		name      string
		run       func([]byte) error
		vec       any
		mutate    func(m map[string]any)
		wantField string
	}{
		{"sync epoch as string", v.ValidateSyncPayload, validSync(t), func(m map[string]any) {
			m["authority_epoch"] = "seven"
		}, "authority_epoch"},
		{"sync mutations as string", v.ValidateSyncPayload, validSync(t), func(m map[string]any) {
			m["mutations"] = "not-an-array"
		}, "mutations"},
		{"evidence byte_length as string", v.ValidateEvidencePayload, validEvidence(t), func(m map[string]any) {
			m["byte_length"] = "4096"
		}, "byte_length"},
		{"evidence captured_at as number", v.ValidateEvidencePayload, validEvidence(t), func(m map[string]any) {
			m["captured_at"] = 12345
		}, "captured_at"},
		{"enrollment device_id as bool", v.ValidateEnrollmentPayload, validEnrollment(t), func(m map[string]any) {
			m["device_id"] = true
		}, "device_id"},
		{"session requested_at as bool", v.ValidateSessionPayload, validSession(t), func(m map[string]any) {
			m["requested_at"] = false
		}, "requested_at"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := vectorToMap(t, tc.vec)
			tc.mutate(m)
			err := tc.run(mustJSON(t, m))
			if err == nil {
				t.Fatalf("payload with %q as wrong type must be rejected", tc.wantField)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			if ve.Field != tc.wantField {
				t.Fatalf("field = %q, want %q (%s)", ve.Field, tc.wantField, err)
			}
			if !strings.Contains(ve.Message, "cannot unmarshal") {
				t.Fatalf("expected type error, got: %v", err)
			}
		})
	}
}

func TestValidateTimestampTypeError(t *testing.T) {
	v := loadTestValidator(t)
	m := vectorToMap(t, validEvidence(t))
	m["captured_at"] = "not-a-date"
	err := v.ValidateEvidencePayload(mustJSON(t, m))
	if err == nil {
		t.Fatal("malformed timestamp must be rejected")
	}
	if !strings.Contains(err.Error(), "cannot parse") && !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsUnknownFields(t *testing.T) {
	v := loadTestValidator(t)
	m := vectorToMap(t, validEnrollment(t))
	m["nonsense_field"] = "sneaky"
	err := v.ValidateEnrollmentPayload(mustJSON(t, m))
	if err == nil {
		t.Fatal("unknown field must be rejected")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "nonsense_field" {
		t.Fatalf("field = %q, want nonsense_field", ve.Field)
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestValidateRejectsMalformedJSON(t *testing.T) {
	v := loadTestValidator(t)
	err := v.ValidateSyncPayload([]byte(`{"batch_id":`))
	if err == nil {
		t.Fatal("malformed json must be rejected")
	}
	if !strings.Contains(err.Error(), "payload invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWrongJsonType(t *testing.T) {
	v := loadTestValidator(t)
	err := v.ValidateSyncPayload([]byte(`[1,2,3]`))
	if err == nil {
		t.Fatal("array body must be rejected")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal array") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func dropField(t *testing.T, v any, field string) map[string]any {
	t.Helper()
	m := vectorToMap(t, v)
	delete(m, field)
	return m
}

func TestNewRequestValidatorRejectsMissingSchema(t *testing.T) {
	spec, err := LoadOpenAPISpec(DefaultSpecPath)
	if err != nil {
		t.Fatalf("LoadOpenAPISpec: %v", err)
	}
	delete(spec.Schemas, "SessionRequest")
	if _, err := NewRequestValidator(spec); err == nil {
		t.Fatal("validator must fail when a core schema is missing")
	}
}

func TestNewRequestValidatorNilSpec(t *testing.T) {
	if _, err := NewRequestValidator(nil); err == nil {
		t.Fatal("nil spec must error")
	}
}
