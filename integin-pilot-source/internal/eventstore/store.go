package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"integin/internal/shared/events"
)

var ErrOptimisticConcurrency = errors.New("optimistic concurrency check failed")
var ErrEventNotFound = errors.New("event stream not found")

type StoredEvent struct {
	AggregateType    string
	AggregateID      string
	AggregateVersion int
	Event            events.Envelope
}

type Store interface {
	Append(ctx context.Context, aggregateType, aggregateID string, expectedVersion int, event events.Envelope) (int, error)
	Load(ctx context.Context, aggregateType, aggregateID string) ([]StoredEvent, error)
}

type stream struct{ events []StoredEvent }

type InMemoryStore struct {
	mu      sync.RWMutex
	streams map[string]stream
}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{streams: make(map[string]stream)} }

func (s *InMemoryStore) Append(ctx context.Context, aggregateType, aggregateID string, expectedVersion int, event events.Envelope) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if aggregateType == "" || aggregateID == "" {
		return 0, errors.New("aggregate type and id are required")
	}
	if event.AggregateType() != aggregateType || event.AggregateID() != aggregateID {
		return 0, errors.New("event aggregate identity mismatch")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := aggregateType + "|" + aggregateID
	current := s.streams[key]
	if expectedVersion != len(current.events) {
		return 0, ErrOptimisticConcurrency
	}
	version := len(current.events) + 1
	current.events = append(current.events, StoredEvent{AggregateType: aggregateType, AggregateID: aggregateID, AggregateVersion: version, Event: event})
	s.streams[key] = current
	return version, nil
}

func (s *InMemoryStore) Load(ctx context.Context, aggregateType, aggregateID string) ([]StoredEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	current, ok := s.streams[aggregateType+"|"+aggregateID]
	if !ok || len(current.events) == 0 {
		return nil, ErrEventNotFound
	}
	copied := make([]StoredEvent, len(current.events))
	copy(copied, current.events)
	return copied, nil
}

type PostgresStore struct{ DB *sql.DB }

func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, errors.New("database handle is required")
	}
	return &PostgresStore{DB: db}, nil
}

func (s *PostgresStore) Append(ctx context.Context, aggregateType, aggregateID string, expectedVersion int, event events.Envelope) (int, error) {
	if s == nil || s.DB == nil {
		return 0, errors.New("database handle is required")
	}
	if event.AggregateType() != aggregateType || event.AggregateID() != aggregateID {
		return 0, errors.New("event aggregate identity mismatch")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var current int
	err = tx.QueryRowContext(ctx, `SELECT aggregate_version FROM event_log WHERE aggregate_type = $1 AND aggregate_id = $2 ORDER BY aggregate_version DESC LIMIT 1 FOR UPDATE`, aggregateType, aggregateID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		current = 0
	} else if err != nil {
		return 0, err
	}
	if current != expectedVersion {
		return 0, ErrOptimisticConcurrency
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return 0, err
	}
	version := current + 1
	_, err = tx.ExecContext(ctx, `INSERT INTO event_log (event_id, tenant_id, organization_id, environment, aggregate_type, aggregate_id, aggregate_version, event_type, schema_version, occurred_at, payload) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, event.EventID(), event.TenantID(), event.OrganizationID(), event.Environment(), aggregateType, aggregateID, version, event.EventType(), event.SchemaVersion(), event.OccurredAt(), payload)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return version, nil
}

func (s *PostgresStore) Load(ctx context.Context, aggregateType, aggregateID string) ([]StoredEvent, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("database handle is required")
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT event_id, tenant_id, organization_id, environment, aggregate_type, aggregate_id, aggregate_version, event_type, schema_version, occurred_at, payload FROM event_log WHERE aggregate_type = $1 AND aggregate_id = $2 ORDER BY aggregate_version ASC`, aggregateType, aggregateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]StoredEvent, 0)
	for rows.Next() {
		var eventID, tenantID, organizationID, environment, storedType, storedID, eventType string
		var version, schemaVersion int
		var occurredAt interface{}
		var payload []byte
		if err := rows.Scan(&eventID, &tenantID, &organizationID, &environment, &storedType, &storedID, &version, &eventType, &schemaVersion, &occurredAt, &payload); err != nil {
			return nil, err
		}
		var event events.Envelope
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("decode event %s: %w", eventID, err)
		}
		result = append(result, StoredEvent{AggregateType: storedType, AggregateID: storedID, AggregateVersion: version, Event: event})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, ErrEventNotFound
	}
	return result, nil
}
