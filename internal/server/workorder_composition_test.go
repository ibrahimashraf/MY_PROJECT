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
}
