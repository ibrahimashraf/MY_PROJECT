package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/evidencehttp"
	"integin/internal/evidencepg"
	"integin/internal/evidenceregistration"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/storage"
)

func NewEvidenceMetadataRegistrationHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver, store storage.Store) (http.Handler, error) {
	if database == nil || validator == nil || resolver == nil || store == nil {
		return nil, errors.New("evidence metadata registration handler requires database, OIDC validator, identity resolver, and evidence store")
	}
	repository, err := evidencepg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	service, err := evidenceregistration.NewService(evidenceregistration.Dependencies{Repository: repository, Store: store})
	if err != nil {
		return nil, err
	}
	return evidencehttp.Handler{Validator: validator, Resolver: resolver, Service: service}, nil
}
