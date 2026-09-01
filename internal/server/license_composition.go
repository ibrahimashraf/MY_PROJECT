package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/domain/license"
	"integin/internal/licensehttp"
	"integin/internal/licensepg"
)

func NewLicenseHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("license handler requires database")
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
	return licensehttp.Handler{LicenseService: service}, nil
}
