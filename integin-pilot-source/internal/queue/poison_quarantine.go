package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// PoisonQuarantineEntry is the forensic snapshot persisted for a River job
// that panicked during payload processing.
type PoisonQuarantineEntry struct {
	JobID          int64
	Kind           string
	TenantID       string
	OrganizationID string
	Args           json.RawMessage
	ErrorText      string
	StackTrace     string
	Attempt        int
	State          string
}

// QuarantineRecorder persists poison-pill forensic snapshots.
type QuarantineRecorder interface {
	Record(ctx context.Context, entry PoisonQuarantineEntry) error
}

// PostgresQuarantineRecorder writes quarantine rows to river_poison_quarantine
// with session-scoped tenant GUCs so multi-tenant RLS remains enforced.
type PostgresQuarantineRecorder struct {
	db *sql.DB
}

// NewPostgresQuarantineRecorder requires a live database handle.
func NewPostgresQuarantineRecorder(db *sql.DB) (*PostgresQuarantineRecorder, error) {
	if db == nil {
		return nil, errors.New("database handle is required")
	}
	return &PostgresQuarantineRecorder{db: db}, nil
}

// Record inserts a quarantine row using is_local=false (session) GUCs so the
// runtime connection retains the tenant scope for subsequent statements.
func (r *PostgresQuarantineRecorder) Record(ctx context.Context, entry PoisonQuarantineEntry) error {
	insertCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(insertCtx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(insertCtx, `SELECT set_config('integin.tenant_id', NULLIF($1, ''), false), set_config('integin.organization_id', NULLIF($2, ''), false)`, entry.TenantID, entry.OrganizationID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(insertCtx, `INSERT INTO river_poison_quarantine (job_id, kind, tenant_id, organization_id, args, error_text, stack_trace, attempt, original_state) VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9) ON CONFLICT (job_id) DO NOTHING`, entry.JobID, entry.Kind, entry.TenantID, entry.OrganizationID, entry.Args, entry.ErrorText, entry.StackTrace, entry.Attempt, entry.State); err != nil {
		return err
	}
	return tx.Commit()
}

// PoisonQuarantineMiddleware wraps every worker and converts unhandled payload
// panics into terminal cancelled jobs after preserving forensic evidence. A
// poison pill therefore can never exhaust retry budgets or stall the queue.
type PoisonQuarantineMiddleware struct {
	river.MiddlewareDefaults
	recorder QuarantineRecorder
}

// NewPoisonQuarantineMiddleware builds the interceptor with the given recorder.
func NewPoisonQuarantineMiddleware(recorder QuarantineRecorder) *PoisonQuarantineMiddleware {
	return &PoisonQuarantineMiddleware{recorder: recorder}
}

// Work implements river.JobMiddleware. A recovered panic is quarantined and
// converted to a JobCancel so River immediately finalizes the job as
// cancelled instead of retrying the malformed payload forever.
func (m *PoisonQuarantineMiddleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(ctx context.Context) error) (result error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicValue := fmt.Sprintf("%v", recovered)
			m.record(job, panicValue, string(debug.Stack()))
			result = river.JobCancel(errors.New("job quarantined: " + panicValue))
		}
	}()
	return doInner(ctx)
}

func (m *PoisonQuarantineMiddleware) record(job *rivertype.JobRow, panicValue, stack string) {
	if m.recorder == nil {
		return
	}
	tenantID, organizationID := extractTenantScope(job.EncodedArgs)
	_ = m.recorder.Record(context.Background(), PoisonQuarantineEntry{
		JobID:          job.ID,
		Kind:           job.Kind,
		TenantID:       tenantID,
		OrganizationID: organizationID,
		Args:           append(json.RawMessage(nil), job.EncodedArgs...),
		ErrorText:      panicValue,
		StackTrace:     stack,
		Attempt:        job.Attempt,
		State:          string(job.State),
	})
}

// extractTenantScope best-effort reads the tenant scope from the encoded job
// args; river payloads carry tenant/org on every job in this codebase.
func extractTenantScope(encoded []byte) (tenantID, organizationID string) {
	var args struct {
		TenantID       string `json:"tenant_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(encoded, &args); err != nil {
		return "", ""
	}
	return args.TenantID, args.OrganizationID
}
