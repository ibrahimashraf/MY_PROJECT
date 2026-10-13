package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrencyGateActiveRequestsUnderLoad(t *testing.T) {
	cg := newConcurrencyGateFromEnv()

	const workers = 800
	total := int64(workers)

	started := make(chan struct{})
	release := make(chan struct{})
	var startedCount atomic.Int64
	var finishedCount atomic.Int64

	handler := cg.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedCount.Add(1)
		if startedCount.Load() == total {
			close(started)
		}
		<-release
	}))

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
			if rec.Code != http.StatusOK {
				t.Errorf("unexpected status %d", rec.Code)
			}
			finishedCount.Add(1)
		}()
	}

	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatalf("only %d of %d requests admitted before timeout", startedCount.Load(), total)
	}

	if got := cg.ActiveRequests(); got != total {
		t.Errorf("ActiveRequests() = %d, want %d", got, total)
	}
	if got := cg.ActiveHealthRequests(); got != 0 {
		t.Errorf("ActiveHealthRequests() = %d, want 0", got)
	}

	close(release)
	wg.Wait()
	if got := cg.ActiveRequests(); got != 0 {
		t.Errorf("ActiveRequests() after drain = %d, want 0", got)
	}
	if got := finishedCount.Load(); got != total {
		t.Errorf("finished = %d, want %d", got, total)
	}
}

func TestConcurrencyGateActiveHealthRequests(t *testing.T) {
	cg := newConcurrencyGateFromEnv()

	const total = 8
	started := make(chan struct{})
	release := make(chan struct{})
	var startedCount atomic.Int64

	handler := cg.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedCount.Add(1)
		if startedCount.Load() == total {
			close(started)
		}
		<-release
	}))

	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		}()
	}

	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatalf("only %d of %d health requests admitted", startedCount.Load(), total)
	}

	if got := cg.ActiveHealthRequests(); got != int64(total) {
		t.Errorf("ActiveHealthRequests() = %d, want %d", got, total)
	}
	if got := cg.ActiveRequests(); got != 0 {
		t.Errorf("ActiveRequests() = %d, want 0", got)
	}

	close(release)
	wg.Wait()
	if got := cg.ActiveHealthRequests(); got != 0 {
		t.Errorf("ActiveHealthRequests() after drain = %d, want 0", got)
	}
}
