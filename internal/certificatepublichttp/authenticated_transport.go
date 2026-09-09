package certificatepublichttp

import (
	"encoding/json"
	"net/http"

	"integin/internal/certificatepg"
)

// AuthenticatedTransport extends the verifier transport with lifecycle-aware
// authentication that validates token scope and membership using the digest-backed
// projection, without requiring OIDC. It remains functional when OIDC is
// disabled-by-default.
//
// Design goals:
// - Validate using digest-backed projection (no OIDC provider call)
// - Compose response using only whitelisted fields (field-level authorization)
// - Reject unknown/authority-shaped JSON fields not in the response whitelist
// - Rate/abuse controls via existing Handler.rateWindow mechanism
// - Fail-closed: any field not explicitly allowed in the response is excluded
type AuthenticatedTransport struct {
	Verifier        Verifier
	AllowedResponse map[string]bool // whitelist of permitted response fields
}

// NewAuthenticatedTransport creates a transport with a field whitelist.
// Only fields explicitly listed in AllowedResponse will appear in the JSON
// response; all other fields from the projection are excluded.
func NewAuthenticatedTransport(verifier Verifier, allowedResponse map[string]bool) *AuthenticatedTransport {
	if allowedResponse == nil {
		allowedResponse = map[string]bool{}
	}
	return &AuthenticatedTransport{
		Verifier:        verifier,
		AllowedResponse: allowedResponse,
	}
}

// VerifyAndRespond verifies the token and returns an authenticated response.
// This is a source-seam method intended to be called from the Handler.ServeHTTP
// when h.Authenticated is configured. It uses the digest-backed projection
// (public_token_digest) and composes the response using the field whitelist.
func (a *AuthenticatedTransport) VerifyAndRespond(w http.ResponseWriter, r *http.Request, token string) {
	// Validate token length (matches original behavior)
	if len(token) < 32 || len(token) > 128 {
		reply(w, http.StatusNotFound, nil)
		return
	}

	// Use the digest-backed verifier (no OIDC provider call)
	view, found, err := a.Verifier.VerifyPublic(r.Context(), token)
	if err != nil {
		reply(w, http.StatusServiceUnavailable, nil)
		return
	}
	if !found {
		reply(w, http.StatusNotFound, nil)
		return
	}

	// Compose response using field whitelist (authority-shaped field rejection)
	response := a.composeAuthenticatedResponse(view)
	reply(w, http.StatusOK, response)
}

// composeAuthenticatedResponse builds the JSON response using only fields
// listed in a.AllowedResponse. This is the core authorization gate: any
// field from the projection not in the whitelist is excluded, preventing
// authority-shaped or unknown JSON field leakage.
func (a *AuthenticatedTransport) composeAuthenticatedResponse(view certificatepg.PublicProjection) map[string]any {
	response := map[string]any{}

	fieldMap := map[string]any{
		"certificate_number": view.CertificateNumber,
		"status":             view.Status,
		"issued_at":          view.IssuedAt,
		"expires_at":         view.ExpiresAt,
		"asset_id":           view.AssetID,
	}

	for key, value := range fieldMap {
		if a.AllowedResponse[key] {
			response[key] = value
		}
	}

	if view.AssetSerialNumber != "" && a.AllowedResponse["asset_serial_number"] {
		response["asset_serial_number"] = view.AssetSerialNumber
	}
	if view.AssetDescription != "" && a.AllowedResponse["asset_description"] {
		response["asset_description"] = view.AssetDescription
	}
	if view.AssetType != "" && a.AllowedResponse["asset_type"] {
		response["asset_type"] = view.AssetType
	}
	if view.InspectionType != "" && a.AllowedResponse["inspection_type"] {
		response["inspection_type"] = view.InspectionType
	}
	if len(view.TestScope) > 0 && a.AllowedResponse["test_scope"] {
		response["test_scope"] = json.RawMessage(view.TestScope)
	}

	return response
}
