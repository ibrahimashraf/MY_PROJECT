package federationhttp_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/federation"
	"integin/internal/federationhttp"
)

func TestFederationHTTP_ServeHTTP(t *testing.T) {
	pubSA, privSA, _ := ed25519.GenerateKey(rand.Reader)
	gateEU := federation.NewFederationGatekeeper(federation.CellEUWest01)
	gateEU.RegisterCell(federation.CellSaudiCentral01, pubSA)

	handler := federationhttp.NewHandler(gateEU)

	gateSA := federation.NewFederationGatekeeper(federation.CellSaudiCentral01)
	msg, err := gateSA.DispatchCrossCellEnvelope(
		federation.CellEUWest01,
		"tenant-ksa-01",
		federation.ClassAnonymizedTelemetric,
		"did:integin:asset:turbine-4",
		10,
		[]byte("vibration=0.02mm,rpm=3000"),
		privSA,
	)
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	raw, _ := json.Marshal(msg)

	// 1. Success
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/replicate", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Replay Conflict (409)
	reqReplay := httptest.NewRequest(http.MethodPost, "/api/v1/federation/replicate", bytes.NewReader(raw))
	recReplay := httptest.NewRecorder()

	handler.ServeHTTP(recReplay, reqReplay)
	if recReplay.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on replay, got %d", recReplay.Code)
	}
}
