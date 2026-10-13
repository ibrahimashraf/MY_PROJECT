package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/flaghttp"
	"integin/internal/flagpg"
)

func NewFeatureFlagAdminHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("feature flag admin handler requires database")
	}
	repository, err := flagpg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return flaghttp.Handler{Repository: repository}, nil
}
