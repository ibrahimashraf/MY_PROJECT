package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/traininghttp"
	"integin/internal/trainingpg"
)

func NewTrainingHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("training handler requires database")
	}
	repository, err := trainingpg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return traininghttp.Handler{Repository: repository}, nil
}
