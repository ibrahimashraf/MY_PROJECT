package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/analyticshttp"
	"integin/internal/analyticspg"
)

func NewAnalyticsHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("analytics handler requires database")
	}
	repository, err := analyticspg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return analyticshttp.Handler{Repository: repository}, nil
}
