package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/inspectionhttp"
)

func NewInspectionHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("inspection handler requires database")
	}
	return inspectionhttp.Handler{DB: database}, nil
}
