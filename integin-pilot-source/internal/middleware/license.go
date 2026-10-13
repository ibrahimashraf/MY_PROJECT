package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type licenseKey struct{}

type LicenseInfo struct {
	Valid                  bool            `json:"valid"`
	Tier                   string          `json:"tier"`
	Status                 string          `json:"status"`
	MaxInspectors          int             `json:"max_inspectors"`
	MaxInspectionsPerMonth int             `json:"max_inspections_per_month"`
	Features               map[string]bool `json:"features"`
}

func LicenseEnforcement(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-ID")
			orgID := r.Header.Get("X-Organization-ID")
			if tenantID == "" || orgID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":  "tenant_required",
					"detail": "X-Tenant-ID and X-Organization-ID headers are required for licensed routes",
				})
				return
			}

			info, err := checkLicense(r.Context(), db, tenantID, orgID)
			if err != nil || !info.Valid {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":  "license_invalid",
					"detail": "Valid license required. Contact support or renew.",
				})
				return
			}

			ctx := context.WithValue(r.Context(), licenseKey{}, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LicenseFromContext(ctx context.Context) (LicenseInfo, bool) {
	info, ok := ctx.Value(licenseKey{}).(LicenseInfo)
	return info, ok
}

func checkLicense(ctx context.Context, db *sql.DB, tenantID, orgID string) (LicenseInfo, error) {
	var info LicenseInfo
	var featuresJSON []byte
	var expiresAt sql.NullTime

	err := db.QueryRowContext(ctx,
		`SELECT tier, status, expires_at, max_inspectors, max_inspections_per_month, features
		 FROM tenant_license
		 WHERE tenant_id = $1 AND organization_id = $2 AND status IN ('trial', 'active')
		 ORDER BY created_at DESC LIMIT 1`,
		tenantID, orgID,
	).Scan(&info.Tier, &info.Status, &expiresAt, &info.MaxInspectors, &info.MaxInspectionsPerMonth, &featuresJSON)

	if err == sql.ErrNoRows {
		return LicenseInfo{Valid: false}, nil
	}
	if err != nil {
		return LicenseInfo{}, err
	}

	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return LicenseInfo{Valid: false, Tier: info.Tier, Status: "expired"}, nil
	}

	if featuresJSON != nil {
		_ = json.Unmarshal(featuresJSON, &info.Features)
	}
	info.Valid = true
	return info, nil
}
