package server

import (
	"net/http"

	"integin/pkg/apicontract"
)

// wrapWithContractValidation mounts OpenAPI v1 request-body validation on an
// mux endpoint when enabled and a validator is configured. It returns the
// handler untouched otherwise, so the validation gate can be switched off
// without changing route wiring.
func wrapWithContractValidation(enabled bool, validator apicontract.ContractValidator, endpointType string, next http.Handler) http.Handler {
	if !enabled || validator == nil {
		return next
	}
	return apicontract.ContractValidationMiddleware(validator, endpointType)(next)
}
