package timestamp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"sync"
	"testing"

	"github.com/riverqueue/river"
)

type statusTransport struct {
	code int
}

func (t statusTransport) RoundTrip(_ context.Context, _ []byte) ([]byte, error) {
	return nil, &StatusError{Code: t.code}
}

type recorderSink struct {
	mu   sync.Mutex
	jobs []TSABacklogJobArgs
}

func (s *recorderSink) Enqueue(_ context.Context, job TSABacklogJobArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
	return nil
}

func (s *recorderSink) first() TSABacklogJobArgs {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[0]
}

func (s *recorderSink) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs)
}

func storeAndForwardTestTSA(t *testing.T, transport Transport) *TSA {
	t.Helper()
	cert, key := testTSARoot(t)
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	if transport == nil {
		transport = &echoTransport{cert: cert, key: key, genTime: testGenTime}
	}
	tsa, err := New(transport, roots)
	if err != nil {
		t.Fatal(err)
	}
	return tsa
}

func TestStoreAndForwardImmediateSuccess(t *testing.T) {
	tsa := storeAndForwardTestTSA(t, nil)
	sf, err := NewStoreAndForward(tsa, &recorderSink{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := sf.Timestamp(context.Background(), []byte(testMessage), TSABacklogJobMeta{CertificateID: "C-1", TenantID: "T-1", OrgID: "O-1"})
	if err != nil {
		t.Fatalf("Timestamp: %v", err)
	}
	if result.Queued {
		t.Fatal("immediate success marked as queued")
	}
	if result.Job != nil {
		t.Fatal("immediate success produced a backlog job")
	}
	if len(result.Token.Response) == 0 || result.Token.GenTime.IsZero() {
		t.Fatalf("token = %+v", result.Token)
	}
}

func TestStoreAndForwardTransientFailureEnqueuesBacklog(t *testing.T) {
	tsa := storeAndForwardTestTSA(t, statusTransport{code: 503})
	sink := &recorderSink{}
	sf, err := NewStoreAndForward(tsa, sink)
	if err != nil {
		t.Fatal(err)
	}

	result, err := sf.Timestamp(context.Background(), []byte(testMessage), TSABacklogJobMeta{CertificateID: "C-9", TenantID: "T-9", OrgID: "O-9"})
	if err != nil {
		t.Fatalf("Timestamp transient failure should queue, got error: %v", err)
	}
	if !result.Queued || result.Job == nil {
		t.Fatalf("expected queued verdict with job, got %+v", result)
	}
	if sink.len() != 1 {
		t.Fatalf("sink recorded %d jobs, want 1", sink.len())
	}
	job := sink.first()
	if job.CertificateID != "C-9" || job.TenantID != "T-9" || job.OrgID != "O-9" {
		t.Fatalf("job identity = %+v", job)
	}
	imprint := sha256.Sum256([]byte(testMessage))
	if !bytes.Equal(job.Imprint, imprint[:]) {
		t.Fatal("job imprint does not match message digest")
	}
	if len(job.RequestDER) == 0 {
		t.Fatal("job missing request der")
	}
	if job.QueuedAt.IsZero() {
		t.Fatal("job missing queued timestamp")
	}
	if job.Kind() != "tsa_store_and_forward" {
		t.Fatalf("job kind = %q", job.Kind())
	}
}

func TestStoreAndForwardNonTransientFailsHard(t *testing.T) {
	tsa := storeAndForwardTestTSA(t, statusTransport{code: 400})
	sink := &recorderSink{}
	sf, err := NewStoreAndForward(tsa, sink)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sf.Timestamp(context.Background(), []byte(testMessage), TSABacklogJobMeta{})
	var statusErr *StatusError
	if err == nil || !errors.As(err, &statusErr) || statusErr.Code != 400 {
		t.Fatalf("err = %v, want 400 StatusError", err)
	}
	if sink.len() != 0 {
		t.Fatal("non-transient failure must not enqueue a backlog job")
	}
}

func TestStoreAndForwardTransientDisabledFailsHard(t *testing.T) {
	tsa := storeAndForwardTestTSA(t, statusTransport{code: 503})
	sf, err := NewStoreAndForward(tsa, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sf.Timestamp(context.Background(), []byte(testMessage), TSABacklogJobMeta{})
	if err == nil {
		t.Fatal("store-and-forward disabled: transient failure must fail hard")
	}
}

func TestTSAStoreAndForwardWorkerProcessesBacklog(t *testing.T) {
	tsa := storeAndForwardTestTSA(t, nil)
	worker := NewTSAStoreAndForwardWorker(tsa)

	imprint := sha256.Sum256([]byte(testMessage))
	req, err := BuildRequest([]byte(testMessage))
	if err != nil {
		t.Fatal(err)
	}
	job := &river.Job[TSABacklogJobArgs]{Args: TSABacklogJobArgs{
		CertificateID: "C-9",
		Imprint:       imprint[:],
		RequestDER:    req,
	}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("worker processing failed: %v", err)
	}
}

func TestTSAStoreAndForwardWorkerMissingRequestFailsClosed(t *testing.T) {
	tsa := storeAndForwardTestTSA(t, nil)
	worker := NewTSAStoreAndForwardWorker(tsa)
	job := &river.Job[TSABacklogJobArgs]{Args: TSABacklogJobArgs{Imprint: goldenMS}}
	if err := worker.Work(context.Background(), job); err == nil {
		t.Fatal("worker accepted a job without request der")
	}
}

func TestIsTransientClassification(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{&StatusError{Code: 503}, true},
		{&StatusError{Code: 500}, true},
		{&StatusError{Code: 400}, false},
		{&StatusError{Code: 429}, false},
		{context.DeadlineExceeded, true},
		{nil, false},
		{errors.New("permanent verification failure"), false},
	} {
		if got := IsTransient(tc.err); got != tc.want {
			t.Errorf("IsTransient(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
