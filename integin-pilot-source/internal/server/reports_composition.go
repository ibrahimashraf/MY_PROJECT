package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/domain/reports"
	"integin/internal/reportshandler"
	"integin/internal/reportspg"
)

func NewReportsHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("reports handler requires database")
	}
	repository, err := reportspg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	var _ reports.Repository = repository
	return reportshandler.Handler{Repository: repository}, nil
}
