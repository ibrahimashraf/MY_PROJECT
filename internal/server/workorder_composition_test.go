package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewMuxMountsWorkOrderRouteOnlyWhenHandlerProvided(t *testing.T) {
	missing := NewMux(Dependencies{})
	missingResponse := httptest.NewRecorder()
	missing.ServeHTTP(missingResponse, httptest.NewRequest(http.MethodPost, "/work-orders/partial-submissions", nil))
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing work-order handler status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}

	mounted := NewMux(Dependencies{WorkOrderHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/work-orders/partial-submissions" {
			t.Fatalf("mounted path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})})
	mountedResponse := httptest.NewRecorder()
	mounted.ServeHTTP(mountedResponse, httptest.NewRequest(http.MethodPost, "/work-orders/partial-submissions", nil))
	if mountedResponse.Code != http.StatusNoContent {
		t.Fatalf("mounted work-order handler status = %d, want %d", mountedResponse.Code, http.StatusNoContent)
	}

	missingHandoverResponse := httptest.NewRecorder()
	missing.ServeHTTP(missingHandoverResponse, httptest.NewRequest(http.MethodPost, "/work-orders/handovers/", nil))
	if missingHandoverResponse.Code != http.StatusNotFound {
		t.Fatalf("missing handover handler status = %d, want %d", missingHandoverResponse.Code, http.StatusNotFound)
	}

	mountedHandover := NewMux(Dependencies{WorkOrderHandoverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})})
	mountedHandoverResponse := httptest.NewRecorder()
	mountedHandover.ServeHTTP(mountedHandoverResponse, httptest.NewRequest(http.MethodPost, "/work-orders/handovers/wo-1", nil))
	if mountedHandoverResponse.Code != http.StatusNoContent {
		t.Fatalf("mounted handover handler status = %d, want %d", mountedHandoverResponse.Code, http.StatusNoContent)
	}
}
