package timestamp

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/riverqueue/river"
)

// TSABacklogJobMeta carries the identity fields that survive the queued wait
// when the courtroom RFC 3161 authority is unreachable.
type TSABacklogJobMeta struct {
	CertificateID string
	TenantID      string
	OrgID         string
}

// TSABacklogJobArgs is the durable River payload for a store-and-forward TSA
// request buffered because the external authority was unreachable at call
// time. RequestDER keeps the exact RFC 3161 TimeStampReq so the worker can
// replay it verbatim once the authority is reachable again.
type TSABacklogJobArgs struct {
	CertificateID string    `json:"certificate_id"`
	TenantID      string    `json:"tenant_id"`
	OrgID         string    `json:"org_id"`
	Imprint       []byte    `json:"imprint"`     // SHA-256 of the evidence message
	RequestDER    []byte    `json:"request_der"` // RFC 3161 TimeStampReq request
	QueuedAt      time.Time `json:"queued_at"`
}

func (TSABacklogJobArgs) Kind() string {
	return "tsa_store_and_forward"
}

// BacklogSink is the injectable durable sink used by StoreAndForward. Tests
// fake it; production wraps queue.Queue via queue.NewTSABacklogSink.
type BacklogSink interface {
	Enqueue(ctx context.Context, job TSABacklogJobArgs) error
}

// StoreAndForwardResult is the verdict of a store-and-forward timestamp call.
type StoreAndForwardResult struct {
	Token  Token
	Queued bool
	Job    *TSABacklogJobArgs
}

// StoreAndForward attempts an immediate RFC 3161 stamp; on a transient
// transport failure it buffers the request as a durable backlog job and
// returns a QUEUED verdict instead of failing the caller hard. When no
// backlog sink is configured (store-and-forward disabled) transient failures
// fail hard exactly like the plain TSA path.
type StoreAndForward struct {
	transport Transport
	roots     *x509.CertPool
	backlog   BacklogSink
	now       func() time.Time
}

// NewStoreAndForward assembles the fallback stamp path from a provisioned TSA.
// backlog may be nil to disable store-and-forward.
func NewStoreAndForward(tsa *TSA, backlog BacklogSink) (*StoreAndForward, error) {
	if tsa == nil || tsa.transport == nil {
		return nil, errors.New("timestamp authority transport is required")
	}
	if tsa.roots == nil {
		return nil, errors.New("timestamp authority roots are required")
	}
	return &StoreAndForward{transport: tsa.transport, roots: tsa.roots, backlog: backlog, now: time.Now}, nil
}

// Timestamp is the store-and-forward entrypoint. It builds the request, calls
// the authority, verifies the token, and on transient failure enqueues the
// request for durable re-processing before returning Queued.
func (s *StoreAndForward) Timestamp(ctx context.Context, message []byte, meta TSABacklogJobMeta) (StoreAndForwardResult, error) {
	req, err := BuildRequest(message)
	if err != nil {
		return StoreAndForwardResult{}, err
	}
	resp, err := s.transport.RoundTrip(ctx, req)
	if err != nil {
		if IsTransient(err) && s.backlog != nil {
			imprint := sha256.Sum256(message)
			job := TSABacklogJobArgs{
				CertificateID: meta.CertificateID,
				TenantID:      meta.TenantID,
				OrgID:         meta.OrgID,
				Imprint:       append([]byte(nil), imprint[:]...),
				RequestDER:    append([]byte(nil), req...),
				QueuedAt:      s.now().UTC(),
			}
			if enqErr := s.backlog.Enqueue(ctx, job); enqErr != nil {
				return StoreAndForwardResult{}, fmt.Errorf("buffer tsa backlog job: %w", enqErr)
			}
			return StoreAndForwardResult{Queued: true, Job: &job}, nil
		}
		return StoreAndForwardResult{}, err
	}
	sum := sha256.Sum256(message)
	genTime, err := Verify(s.roots, resp, sum[:], s.now())
	if err != nil {
		return StoreAndForwardResult{}, err
	}
	return StoreAndForwardResult{Token: Token{Response: resp, GenTime: genTime}}, nil
}

// TSAStoreAndForwardWorker replays a buffered request through the TSA. A
// transient error propagates so River backoffs and retries the durable job.
type TSAStoreAndForwardWorker struct {
	river.WorkerDefaults[TSABacklogJobArgs]
	tsa *TSA
	now func() time.Time
}

// NewTSAStoreAndForwardWorker assembles the worker over a provisioned TSA.
func NewTSAStoreAndForwardWorker(tsa *TSA) *TSAStoreAndForwardWorker {
	return &TSAStoreAndForwardWorker{tsa: tsa, now: time.Now}
}

// Work implements river.Worker[TSABacklogJobArgs].
func (w *TSAStoreAndForwardWorker) Work(ctx context.Context, job *river.Job[TSABacklogJobArgs]) error {
	if w.tsa == nil || w.tsa.transport == nil || w.tsa.roots == nil {
		return errors.New("tsa authority is not configured")
	}
	args := job.Args
	if len(args.RequestDER) == 0 {
		return errors.New("tsa backlog job missing request der")
	}
	if len(args.Imprint) == 0 {
		return errors.New("tsa backlog job missing imprint")
	}
	resp, err := w.tsa.transport.RoundTrip(ctx, args.RequestDER)
	if err != nil {
		return err
	}
	if _, err := Verify(w.tsa.roots, resp, args.Imprint, w.now()); err != nil {
		return fmt.Errorf("tsa backlog verification failed: %w", err)
	}
	return nil
}
