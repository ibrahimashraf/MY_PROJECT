package queue

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/riverqueue/river/rivertype"
)

type captureRecorder struct {
	mu      sync.Mutex
	entries []PoisonQuarantineEntry
}

func (c *captureRecorder) Record(_ context.Context, entry PoisonQuarantineEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, entry)
	return nil
}

func (c *captureRecorder) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func jobRow() *rivertype.JobRow {
	return &rivertype.JobRow{
		ID:          41,
		Attempt:     2,
		Kind:        "certificate_render",
		EncodedArgs: []byte(`{"tenant_id":"t1","organization_id":"o1","certificate_id":"c1"}`),
		State:       rivertype.JobStateRunning,
		MaxAttempts: 25,
	}
}

func TestPoisonQuarantineMiddlewareCatchesPanics(t *testing.T) {
	recorder := &captureRecorder{}
	mw := NewPoisonQuarantineMiddleware(recorder)
	job := jobRow()

	var err error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("panic must be contained by the middleware, leaked %v", recovered)
			}
		}()
		err = mw.Work(context.Background(), job, func(ctx context.Context) error {
			panic("exploded payload: encoding/json: cannot unmarshal")
		})
	}()

	if err == nil {
		t.Fatal("expected a terminal error after quarantine")
	}
	if recorder.count() != 1 {
		t.Fatalf("expected exactly 1 quarantine record, got %d", recorder.count())
	}
	entry := recorder.entries[0]
	if entry.JobID != 41 || entry.Kind != "certificate_render" {
		t.Fatalf("unexpected forensic identity %+v", entry)
	}
	if entry.TenantID != "t1" || entry.OrganizationID != "o1" {
		t.Fatalf("tenant scope not preserved: %+v", entry)
	}
	if entry.Attempt != 2 || entry.State != "running" {
		t.Fatalf("unexpected attempt/state snapshot %+v", entry)
	}
	if !strings.Contains(entry.ErrorText, "exploded payload") {
		t.Fatalf("panic reason missing from quarantine: %q", entry.ErrorText)
	}
	if entry.StackTrace == "" {
		t.Fatal("stack trace must be preserved for forensics")
	}
	if len(entry.Args) == 0 || string(entry.Args) != string(job.EncodedArgs) {
		t.Fatalf("encoded args not preserved: %q", entry.Args)
	}
}

func TestPoisonQuarantineMiddlewarePassesOrdinaryErrorsThrough(t *testing.T) {
	recorder := &captureRecorder{}
	mw := NewPoisonQuarantineMiddleware(recorder)
	sentinel := errors.New("transient failure")

	err := mw.Work(context.Background(), jobRow(), func(ctx context.Context) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("ordinary errors must pass through untouched, got %v", err)
	}
	if recorder.count() != 0 {
		t.Fatalf("no quarantine expected for non-panic errors, got %d", recorder.count())
	}
}

func TestPoisonQuarantineMiddlewareAllowsSuccess(t *testing.T) {
	recorder := &captureRecorder{}
	mw := NewPoisonQuarantineMiddleware(recorder)
	if err := mw.Work(context.Background(), jobRow(), func(ctx context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("success must not be affected, got %v", err)
	}
	if recorder.count() != 0 {
		t.Fatalf("no quarantine expected on success, got %d", recorder.count())
	}
}

func TestExtractTenantScope(t *testing.T) {
	if tenant, org := extractTenantScope([]byte(`{"tenant_id":"t9","organization_id":"o9"}`)); tenant != "t9" || org != "o9" {
		t.Fatalf("unexpected scope extraction %q/%q", tenant, org)
	}
	if tenant, org := extractTenantScope([]byte(`{}`)); tenant != "" || org != "" {
		t.Fatalf("empty args must yield empty scope, got %q/%q", tenant, org)
	}
	if tenant, org := extractTenantScope([]byte(`not json`)); tenant != "" || org != "" {
		t.Fatalf("malformed args must yield empty scope, got %q/%q", tenant, org)
	}
	if tenant, org := extractTenantScope(nil); tenant != "" || org != "" {
		t.Fatalf("nil args must yield empty scope, got %q/%q", tenant, org)
	}
}
