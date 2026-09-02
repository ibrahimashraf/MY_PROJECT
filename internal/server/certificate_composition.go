package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/certificatehttp"
	"integin/internal/certificatepg"
	"integin/internal/identity"
	"integin/internal/oidcauth"
)

func NewCertificateHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil || validator == nil || resolver == nil {
		return nil, errors.New("certificate handler requires database, OIDC validator, and identity resolver")
	}
	repository, err := certificatepg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return certificatehttp.Handler{
		Validator: validator,
		Actors: certificatehttp.LocalActorResolver{
			Memberships: resolver,
		},
		Lifecycle: repository,
	}, nil
}