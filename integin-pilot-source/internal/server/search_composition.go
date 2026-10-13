package server

import (
	"database/sql"
	"net/http"

	"integin/internal/searchhttp"
	"integin/internal/searchpg"
)

func NewSearchHandler(database *sql.DB) (http.Handler, error) {
	repo := searchpg.New(database)
	return &searchhttp.Handler{Repository: repo}, nil
}
