package apicontract

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContractValidationMiddlewareAcceptsValidBodies(t *testing.T) {
	v := loadTestValidator(t)
	cases := []struct {
		name     string
		endpoint string
		vec      any
	}{
		{"sync", EndpointSync, validSync(t)},
		{"evidence", EndpointEvidence, validEvidence(t)},
		{"enrollment", EndpointEnrollment, validEnrollment(t)},
		{"session", EndpointSession, validSession(t)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				called = true
				if _, err := io.ReadAll(request.Body); err != nil {
					t.Fatalf("downstream body read: %v", err)
				}
				writer.WriteHeader(http.StatusOK)
			})
			request := httptest.NewRequest(http.MethodPost, tc.endpoint, bytes.NewReader(mustJSON(t, tc.vec)))
			recorder := httptest.NewRecorder()
			ContractValidationMiddleware(v, tc.endpoint)(next).ServeHTTP(recorder, request)
			if !called {
				t.Fatal("downstream handler was not called for a valid body")
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
		})
	}
}

func TestContractValidationMiddlewareRejectsInvalidBodies(t *testing.T) {
	v := loadTestValidator(t)
	cases := []struct {
		name       string
		endpoint   string
		body       []byte
		wantDetail string
		wantMsgSub string
	}{
		{"missing required field", EndpointSync, mustJSON(t, dropField(t, validSync(t), "tenant_id")), "tenant_id", "required field is missing"},
		{"malformed json", EndpointSync, []byte(`{"batch_id":`), "", "payload invalid"},
		{"wrong json type", EndpointSession, []byte(`[1,2,3]`), "", "cannot unmarshal array"},
		{"empty body", EndpointEvidence, nil, "", "request body is required"},
		{"unknown field", EndpointEnrollment, mustJSON(t, mutateMap(t, validEnrollment(t), "sneaky_field", "x")), "sneaky_field", "unknown field"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				called = true
				writer.WriteHeader(http.StatusOK)
			})
			request := httptest.NewRequest(http.MethodPost, tc.endpoint, bytes.NewReader(tc.body))
			recorder := httptest.NewRecorder()
			ContractValidationMiddleware(v, tc.endpoint)(next).ServeHTTP(recorder, request)
			if called {
				t.Fatal("downstream handler must not run for an invalid body")
			}
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
			var er ErrorResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &er); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if er.Code != http.StatusBadRequest {
				t.Fatalf("error code = %d, want 400", er.Code)
			}
			if er.Error == "" {
				t.Fatal("error message must not be empty")
			}
			if tc.wantDetail != "" && er.Detail != tc.wantDetail {
				t.Fatalf("detail = %q, want %q", er.Detail, tc.wantDetail)
			}
			if !bytes.Contains([]byte(er.Error), []byte(tc.wantMsgSub)) {
				t.Fatalf("error %q must contain %q", er.Error, tc.wantMsgSub)
			}
		})
	}
}

func TestContractValidationMiddlewareFailsClosedOnUnknownEndpoint(t *testing.T) {
	v := loadTestValidator(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/unknown", bytes.NewReader(mustJSON(t, validSync(t))))
	recorder := httptest.NewRecorder()
	ContractValidationMiddleware(v, "/api/v1/unknown")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("downstream must not run for an unknown endpoint type")
	})).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	var er ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &er); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if er.Code != 500 || er.Error == "" {
		t.Fatalf("unexpected error response: %+v", er)
	}
}

func TestContractValidationMiddlewarePassesThroughNonPost(t *testing.T) {
	v := loadTestValidator(t)
	called := false
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, EndpointSync, nil)
	recorder := httptest.NewRecorder()
	ContractValidationMiddleware(v, EndpointSync)(next).ServeHTTP(recorder, request)
	if !called {
		t.Fatal("non-POST request must pass through to the handler")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}

func mutateMap(t *testing.T, v any, field string, value any) map[string]any {
	t.Helper()
	m := vectorToMap(t, v)
	m[field] = value
	return m
}
