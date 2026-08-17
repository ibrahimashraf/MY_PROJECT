package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
)

func TestNewMuxMountsHealthAndSyncRoutes(t *testing.T) {
	processor, err := domainsync.NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-1", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(device, "auth-1", "secret", []string{"inspection.perform"}, time.Now().UTC(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	mux := NewMux(Dependencies{SyncProcessor: processor, Devices: []device_trust.Device{device}, Authorities: []device_trust.AuthorityPackage{authority}})
	health := httptest.NewRecorder()
	mux.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}

	method := httptest.NewRecorder()
	mux.ServeHTTP(method, httptest.NewRequest(http.MethodGet, "/sync", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("sync method status = %d", method.Code)
	}
}

func TestNewMuxDoesNotMountOIDCSessionRouteWithoutDependency(t *testing.T) {
	mux := NewMux(Dependencies{})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/identity/session", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("OIDC-disabled session route status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
