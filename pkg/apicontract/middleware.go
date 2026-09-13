package apicontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrorResponse is the uniform error envelope returned when a request body
// fails contract validation. It matches the ErrorResponse component schema in
// docs/api/openapi_v1.yaml.
type ErrorResponse struct {
	Error  string `json:"error"`
	Code   int    `json:"code"`
	Detail string `json:"detail,omitempty"`
}

// ContractValidationMiddleware returns HTTP middleware that validates the
// request body of a canonical OpenAPI v1 endpoint against its contract schema
// before passing the request downstream. endpointType must be one of the
// Endpoint* constants; the read body is restored so the wrapped handler (and
// any inner middleware) can parse it.
//
// A body that violates the contract is rejected with HTTP 400 and a structured
// ErrorResponse. An unrecognized endpointType fails closed with HTTP 500 so a
// routing misconfiguration can never silently skip validation.
func ContractValidationMiddleware(validator ContractValidator, endpointType string) func(http.Handler) http.Handler {
	validate := validateForEndpoint(validator, endpointType)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if validate == nil {
				writeContractError(writer, "unknown endpoint type for contract validation: "+endpointType, http.StatusInternalServerError, "")
				return
			}
			if request.Method != http.MethodPost {
				next.ServeHTTP(writer, request)
				return
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				writeContractError(writer, "unreadable request body", http.StatusBadRequest, "")
				return
			}
			request.Body = io.NopCloser(bytes.NewReader(body))
			if err := validate(body); err != nil {
				var ve *ValidationError
				detail := ""
				if errors.As(err, &ve) {
					detail = ve.Field
				}
				writeContractError(writer, err.Error(), http.StatusBadRequest, detail)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func validateForEndpoint(validator ContractValidator, endpointType string) func([]byte) error {
	switch endpointType {
	case EndpointSync:
		return validator.ValidateSyncPayload
	case EndpointEvidence:
		return validator.ValidateEvidencePayload
	case EndpointEnrollment:
		return validator.ValidateEnrollmentPayload
	case EndpointSession:
		return validator.ValidateSessionPayload
	default:
		return nil
	}
}

func writeContractError(writer http.ResponseWriter, message string, status int, detail string) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(ErrorResponse{Error: message, Code: status, Detail: detail}); err != nil {
		http.Error(writer, http.StatusText(status), status)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if _, err := writer.Write(buf.Bytes()); err != nil {
		return
	}
}
