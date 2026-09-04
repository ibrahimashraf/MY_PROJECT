package queue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
)

// CertificateRenderJobArgs defines the durable River payload for background PDF/A-4b rendering.
type CertificateRenderJobArgs struct {
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	CertificateID  string `json:"certificate_id"`
	InspectionID   string `json:"inspection_id"`
	ActorUserID    string `json:"actor_user_id"`
}

func (CertificateRenderJobArgs) Kind() string {
	return "certificate_render"
}

// SealedEvidenceJobArgs defines the durable River payload for background evidence hashing & S3 storage.
type SealedEvidenceJobArgs struct {
	TenantID       string   `json:"tenant_id"`
	OrganizationID string   `json:"organization_id"`
	EvidencePackID string   `json:"evidence_pack_id"`
	ArtifactHashes []string `json:"artifact_hashes"`
	ActorUserID    string   `json:"actor_user_id"`
}

func (SealedEvidenceJobArgs) Kind() string {
	return "sealed_evidence_pack"
}

// Queue coordinates durable job enqueuing and worker execution via River on PostgreSQL.
type Queue struct {
	client *river.Client[*sql.Tx]
	db     *sql.DB
}

// NewQueue initializes a River client backed by database/sql PostgreSQL connection pool.
func NewQueue(ctx context.Context, db *sql.DB, workers *river.Workers) (*Queue, error) {
	if db == nil {
		return nil, errors.New("database handle is required")
	}

	driver := riverdatabasesql.New(db)
	client, err := river.NewClient[*sql.Tx](driver, &river.Config{
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize river client: %w", err)
	}

	return &Queue{
		client: client,
		db:     db,
	}, nil
}

// Client returns the underlying River client.
func (q *Queue) Client() *river.Client[*sql.Tx] {
	return q.client
}

// InsertTx enqueues a job within an active database transaction.
// This guarantees that if the business transaction rolls back, the job is never scheduled.
func (q *Queue) InsertTx(ctx context.Context, tx *sql.Tx, args river.JobArgs) error {
	if tx == nil {
		return errors.New("active transaction is required for transactional enqueuing")
	}
	_, err := q.client.InsertTx(ctx, tx, args, nil)
	return err
}
