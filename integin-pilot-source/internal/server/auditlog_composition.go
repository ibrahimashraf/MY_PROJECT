package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/auditloghttp"
	"integin/internal/auditlogpg"
)

func NewAuditLogHandler(database *sql.DB) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("audit log handler requires database")
	}
	repository, err := auditlogpg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return auditloghttp.Handler{AuditLogRepository: repository}, nil
}
