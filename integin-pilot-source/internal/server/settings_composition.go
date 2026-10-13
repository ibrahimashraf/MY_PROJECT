package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/settingshttp"
	"integin/internal/settingspg"
)

func NewSettingsHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("settings handler requires database")
	}
	repository, err := settingspg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return settingshttp.Handler{Repository: repository}, nil
}
