package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/pkg/apicontract"
)

const contractTestValidSync = `{"batch_id":"batch_1","tenant_id":"tenant-a","organization_id":"org-a","device_id":"dev_1","authority_id":"auth_1","authority_epoch":7,"sequence":12,"mutations":[{"mutation_id":"m_1","entity_type":"inspection_item","entity_id":"ii_1","operation":"update","payload":{"status":"complete"},"at":"2026-09-10T08:00:00Z","version":3}],"signature":"signed-batch"}`

func loadContractTestValidator(t *testing.T) apicontract.ContractValidator {
	t.Helper()
	v, err := apicontract.NewRequestValidatorFromSpecFile("../../docs/api/openapi_v1.yaml")
	if err != nil {
		t.Fatalf("NewRequestValidatorFromSpecFile: %v", err)
	}
	return v
}

func TestWrapWithContractValidationDisabled(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodPost, apicontract.EndpointSync, bytes.NewReader([]byte(`{"batch_id":1}`)))
	recorder := httptest.NewRecorder()
	wrapWithContractValidation(false, loadContractTestValidator(t), apicontract.EndpointSync, next).ServeHTTP(recorder, request)
	if !called || recorder.Code != http.StatusOK {
		t.Fatalf("disabled gate must pass through, called=%v code=%d", called, recorder.Code)
	}
}

func TestWrapWithContractValidationNilValidator(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodPost, apicontract.EndpointSync, bytes.NewReader([]byte(`{"batch_id":1}`)))
	recorder := httptest.NewRecorder()
	wrapWithContractValidation(true, nil, apicontract.EndpointSync, next).ServeHTTP(recorder, request)
	if !called || recorder.Code != http.StatusOK {
		t.Fatalf("nil validator must pass through, called=%v code=%d", called, recorder.Code)
	}
}

func TestWrapWithContractValidationAcceptsValidBody(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		called = true
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("downstream body read: %v", err)
		}
		if len(body) == 0 {
			t.Fatal("downstream must receive the restored request body")
		}
		writer.WriteHeader(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodPost, apicontract.EndpointSync, bytes.NewReader([]byte(contractTestValidSync)))
	recorder := httptest.NewRecorder()
	wrapWithContractValidation(true, loadContractTestValidator(t), apicontract.EndpointSync, next).ServeHTTP(recorder, request)
	if !called || recorder.Code != http.StatusOK {
		t.Fatalf("valid body must reach handler, called=%v code=%d", called, recorder.Code)
	}
}

func TestWrapWithContractValidationRejectsInvalidBody(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodPost, apicontract.EndpointSync, bytes.NewReader([]byte(`{"batch_id":"batch_1"}`)))
	recorder := httptest.NewRecorder()
	wrapWithContractValidation(true, loadContractTestValidator(t), apicontract.EndpointSync, next).ServeHTTP(recorder, request)
	if called {
		t.Fatal("invalid body must be rejected before the handler")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	var er apicontract.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &er); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if er.Code != http.StatusBadRequest || er.Error == "" {
		t.Fatalf("unexpected error response: %+v", er)
	}
}
