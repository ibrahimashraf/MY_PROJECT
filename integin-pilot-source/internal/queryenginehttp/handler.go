package queryenginehttp

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"integin/internal/shared/httpresponse"
	"integin/internal/shared/pgtx"
	"integin/pkg/queryengine"
)

// Handler serves PostgREST-style dynamic queries gated by tenant RLS and schema whitelist.
type Handler struct {
	Registry *queryengine.SchemaRegistry
	DB       *sql.DB
}

// NewHandler constructs an HTTP handler with the provided schema registry and database.
func NewHandler(registry *queryengine.SchemaRegistry, db *sql.DB) *Handler {
	return &Handler{
		Registry: registry,
		DB:       db,
	}
}

// CanonicalSchema returns the production schema whitelist for core tables.
func CanonicalSchema() map[string][]string {
	return map[string][]string{
		"inspection_record": {
			"id", "tenant_id", "organization_id", "work_order_id", "asset_id",
			"inspector_id", "lifecycle_state", "revision", "finalization_state",
			"created_at", "updated_at",
		},
		"work_order": {
			"id", "tenant_id", "organization_id", "asset_id", "assigned_to",
			"status", "priority", "created_at", "updated_at",
		},
		"evidence_metadata": {
			"id", "tenant_id", "organization_id", "inspection_id",
			"object_key", "content_type", "ciphertext_bytes", "device_id",
			"authority_id", "authority_epoch", "transaction_id", "receipt_id",
			"signature_algorithm", "key_id", "classification", "hold_state",
			"registered_by",
		},
		"certificate_record": {
			"id", "tenant_id", "organization_id", "certificate_number",
			"status", "asset_id", "issued_at", "created_at",
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.Registry == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "query engine registry not configured")
		return
	}

	table := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/query"), "/")
	if table == "" {
		httpresponse.JSON(w, http.StatusOK, map[string]any{
			"tables": h.Registry.Tables(),
		})
		return
	}

	// Validate tenant headers
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}

	q, err := queryengine.ParseQuery(r.URL.Query())
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid query parameters: "+err.Error())
		return
	}

	// Default limit if unstated or excessive
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}

	sqlText, args, err := h.Registry.BuildSELECT(table, q)
	if err != nil {
		if errors.Is(err, queryengine.ErrUnknownTable) {
			httpresponse.Error(w, http.StatusNotFound, "table not found or not permitted")
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "invalid query: "+err.Error())
		return
	}

	args[0] = tenantID
	args[1] = orgID

	if h.DB == nil {
		// Dry-run mode when no DB handle is wired
		httpresponse.JSON(w, http.StatusOK, map[string]any{
			"table":   table,
			"dry_run": true,
			"sql":     sqlText,
		})
		return
	}

	tx, err := pgtx.BeginScope(r.Context(), h.DB, tenantID, orgID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to establish database scope")
		return
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(r.Context(), sqlText, args...)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "query execution failed")
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "failed to read column metadata")
		return
	}

	var results []map[string]any
	for rows.Next() {
		colVals := make([]any, len(cols))
		colValPtrs := make([]any, len(cols))
		for i := range colVals {
			colValPtrs[i] = &colVals[i]
		}

		if err := rows.Scan(colValPtrs...); err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "failed to scan row")
			return
		}

		rowMap := make(map[string]any, len(cols))
		for i, colName := range cols {
			val := colVals[i]
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "row iteration error")
		return
	}

	_ = tx.Commit()

	if results == nil {
		results = []map[string]any{}
	}

	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"table":  table,
		"count":  len(results),
		"limit":  q.Limit,
		"offset": q.Offset,
		"data":   results,
	})
}
