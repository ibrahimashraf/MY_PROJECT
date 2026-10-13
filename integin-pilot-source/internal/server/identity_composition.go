package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/auditlogpg"
	"integin/internal/identity"
	"integin/internal/identityhttp"
	"integin/internal/oidcauth"
)

// NewIdentityGrantHandler builds the operator identity-grant endpoint. It
// requires the database, the OIDC validator, and the identity resolver; the
// audit ledger repository is constructed alongside so every grant is recorded.
func NewIdentityGrantHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil || validator == nil || resolver == nil {
		return nil, errors.New("identity grant handler requires database, OIDC validator, and identity resolver")
	}
	auditor, err := auditlogpg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return identityhttp.Handler{DB: database, Validator: validator, Resolver: resolver, Auditor: auditor}, nil
}
