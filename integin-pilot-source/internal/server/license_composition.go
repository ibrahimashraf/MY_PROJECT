package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/domain/license"
	"integin/internal/identity"
	"integin/internal/licensehttp"
	"integin/internal/licensepg"
	"integin/internal/oidcauth"
)

func NewLicenseHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("license handler requires database")
	}
	if validator == nil {
		return nil, errors.New("license handler requires OIDC validator")
	}
	if resolver == nil {
		return nil, errors.New("license handler requires identity resolver")
	}
	repository, err := licensepg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	service, err := license.NewService(license.ServiceDependencies{
		Repository:   repository,
		Transactions: repository,
	})
	if err != nil {
		return nil, err
	}
	return licensehttp.Handler{
		Validator:      licensehttp.ValidatorWrapper{Validator: validator},
		Resolver:       resolver,
		LicenseService: service,
	}, nil
}
