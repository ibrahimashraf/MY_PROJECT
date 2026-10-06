package atexhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/atexhttp"
	"integin/pkg/atex"
)

func TestATEXHTTP_Evaluate(t *testing.T) {
	handler := atexhttp.NewHandler()

	loop := atex.IntrinsicSafetyLoop{
		ApparatusDID:   "did:integin:apparatus:sensor-01",
		BarrierDID:     "did:integin:barrier:mtl-5511",
		Zone:           atex.Zone1,
		EnvironmentGas: atex.GroupIIB,
		ApparatusGas:   atex.GroupIIC,
		ApparatusTemp:  atex.T4,
		AutoIgnitionC:  200.0,
		ApparatusParams: atex.IntrinsicallySafeParameters{
			Ui: 30.0,
			Ii: 100.0,
			Pi: 750.0,
			Ci: 10.0,
			Li: 0.1,
		},
		BarrierParams: atex.BarrierParameters{
			Uo: 28.0,
			Io: 93.0,
			Po: 650.0,
			Co: 80.0,
			Lo: 4.0,
		},
		Cable: atex.CableParameters{
			LengthMeters: 100.0,
			CcablePerM:   0.1,
			LcablePerM:   0.005,
		},
	}
	raw, _ := json.Marshal(loop)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/atex/evaluate", bytes.NewReader(raw))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res atex.VerificationResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !res.IsCompliant {
		t.Errorf("expected compliant loop, got %+v", res)
	}
}
