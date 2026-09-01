package auditlogpg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"integin/internal/domain/auditlog"
)

var ErrNilDB = errors.New("audit log postgres repository requires a database")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Append(ctx context.Context, entry auditlog.Entry) (auditlog.Entry, error) {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%s:%s:%s:%d", entry.TenantID, entry.OrganizationID, entry.EventType, time.Now().UnixNano())
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return auditlog.Entry{}, err
	}
	defer tx.Rollback()

	// Lock the last entry for this tenant/org to prevent concurrent forks
	var lastHash string
	err = tx.QueryRowContext(ctx,
		`SELECT entry_hash FROM audit_log WHERE tenant_id = $1 AND organization_id = $2 ORDER BY created_at DESC LIMIT 1 FOR UPDATE`,
		entry.TenantID, entry.OrganizationID,
	).Scan(&lastHash)
	if err != nil && err != sql.ErrNoRows {
		return auditlog.Entry{}, err
	}
	if err == sql.ErrNoRows {
		lastHash = auditlog.GenesisHash()
	}

	entry.PreviousHash = lastHash
	entryData := entry.EventType + string(entry.EntityType) + entry.EntityID + entry.ActorID + string(entry.Action) + entry.CreatedAt.Format(time.RFC3339Nano)
	entry.EntryHash = auditlog.ComputeHash(entry.PreviousHash, entryData)

	oldValueJSON, _ := json.Marshal(entry.OldValue)
	newValueJSON, _ := json.Marshal(entry.NewValue)
	metadataJSON, _ := json.Marshal(entry.Metadata)

	_, err = tx.ExecContext(ctx,
		`INSERT INTO audit_log (id, tenant_id, organization_id, event_type, entity_type, entity_id,
		        actor_id, actor_name, action, old_value, new_value, metadata, ip_address, user_agent,
		        previous_hash, entry_hash, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		entry.ID, entry.TenantID, entry.OrganizationID, entry.EventType, entry.EntityType, entry.EntityID,
		entry.ActorID, entry.ActorName, string(entry.Action), oldValueJSON, newValueJSON, metadataJSON,
		entry.IPAddress, entry.UserAgent, entry.PreviousHash, entry.EntryHash, entry.CreatedAt,
	)
	if err != nil {
		return auditlog.Entry{}, err
	}

	if err := tx.Commit(); err != nil {
		return auditlog.Entry{}, err
	}

	return entry, nil
}

func (r *Repository) Query(ctx context.Context, req auditlog.QueryRequest) (auditlog.QueryResponse, error) {
	if req.TenantID == "" || req.OrganizationID == "" {
		return auditlog.QueryResponse{}, errors.New("tenant_id and organization_id are required")
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 200 {
		req.Limit = 200
	}

	where := "tenant_id = $1 AND organization_id = $2"
	args := []interface{}{req.TenantID, req.OrganizationID}
	argIdx := 3

	if req.EntityType != "" {
		where += fmt.Sprintf(" AND entity_type = $%d", argIdx)
		args = append(args, req.EntityType)
		argIdx++
	}
	if req.EntityID != "" {
		where += fmt.Sprintf(" AND entity_id = $%d", argIdx)
		args = append(args, req.EntityID)
		argIdx++
	}
	if req.ActorID != "" {
		where += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, req.ActorID)
		argIdx++
	}
	if req.EventType != "" {
		where += fmt.Sprintf(" AND event_type = $%d", argIdx)
		args = append(args, req.EventType)
		argIdx++
	}
	if req.From != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *req.From)
		argIdx++
	}
	if req.To != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *req.To)
		argIdx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log WHERE %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return auditlog.QueryResponse{}, err
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, organization_id, event_type, entity_type, entity_id,
		        actor_id, actor_name, action, old_value, new_value, metadata,
		        ip_address, user_agent, previous_hash, entry_hash, created_at
		 FROM audit_log WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, req.Limit, req.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return auditlog.QueryResponse{}, err
	}
	defer rows.Close()

	var entries []auditlog.Entry
	for rows.Next() {
		var entry auditlog.Entry
		var oldValueJSON, newValueJSON, metadataJSON []byte
		var action string
		if err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.OrganizationID, &entry.EventType,
			&entry.EntityType, &entry.EntityID, &entry.ActorID, &entry.ActorName,
			&action, &oldValueJSON, &newValueJSON, &metadataJSON,
			&entry.IPAddress, &entry.UserAgent, &entry.PreviousHash, &entry.EntryHash,
			&entry.CreatedAt,
		); err != nil {
			return auditlog.QueryResponse{}, err
		}
		entry.Action = auditlog.Action(action)
		if oldValueJSON != nil {
			_ = json.Unmarshal(oldValueJSON, &entry.OldValue)
		}
		if newValueJSON != nil {
			_ = json.Unmarshal(newValueJSON, &entry.NewValue)
		}
		if metadataJSON != nil {
			_ = json.Unmarshal(metadataJSON, &entry.Metadata)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return auditlog.QueryResponse{}, err
	}

	return auditlog.QueryResponse{Entries: entries, Total: total}, nil
}

func (r *Repository) VerifyChain(ctx context.Context, tenantID, organizationID string, from, to *time.Time) (auditlog.VerifyResult, error) {
	if tenantID == "" || organizationID == "" {
		return auditlog.VerifyResult{}, errors.New("tenant_id and organization_id are required")
	}
	where := "tenant_id = $1 AND organization_id = $2"
	args := []interface{}{tenantID, organizationID}
	argIdx := 3

	if from != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *from)
		argIdx++
	}
	if to != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *to)
		argIdx++
	}
	_ = argIdx // used conditionally

	query := fmt.Sprintf(
		`SELECT id, event_type, entity_type, entity_id, actor_id, action,
		        previous_hash, entry_hash, created_at
		 FROM audit_log WHERE %s ORDER BY created_at ASC`,
		where,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return auditlog.VerifyResult{}, err
	}
	defer rows.Close()

	var result auditlog.VerifyResult
	result.Valid = true
	previousHash := auditlog.GenesisHash()

	for rows.Next() {
		var entry auditlog.Entry
		var action string
		if err := rows.Scan(
			&entry.ID, &entry.EventType, &entry.EntityType, &entry.EntityID,
			&entry.ActorID, &action, &entry.PreviousHash, &entry.EntryHash,
			&entry.CreatedAt,
		); err != nil {
			return auditlog.VerifyResult{}, err
		}
		entry.Action = auditlog.Action(action)

		if entry.PreviousHash != previousHash {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("entry %s: previous_hash mismatch", entry.ID))
		}

		entryData := entry.EventType + string(entry.EntityType) + entry.EntityID + entry.ActorID + string(entry.Action) + entry.CreatedAt.Format(time.RFC3339Nano)
		expectedHash := auditlog.ComputeHash(entry.PreviousHash, entryData)
		if expectedHash != entry.EntryHash {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("entry %s: entry_hash mismatch", entry.ID))
		}

		previousHash = entry.EntryHash
	}
	if err := rows.Err(); err != nil {
		return auditlog.VerifyResult{}, err
	}

	return result, nil
}
