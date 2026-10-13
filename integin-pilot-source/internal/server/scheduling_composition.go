package server

import (
	"database/sql"
	"net/http"

	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/schedulinghttp"
)

func NewSchedulingHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	return schedulinghttp.Handler{
		Validator: validator,
		Resolver:  resolver,
	}, nil
}
