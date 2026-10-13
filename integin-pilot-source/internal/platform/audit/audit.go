package audit

import (
	"errors"
	"strings"
	"sync"
	"time"
)

type Record struct {
	ID            string
	TenantID      string
	Environment   string
	ActorType     string
	ActorID       string
	Action        string
	ResourceType  string
	ResourceID    string
	Outcome       string
	CorrelationID string
	Context       map[string]any
	CreatedAt     time.Time
}

type Ledger struct {
	mu      sync.RWMutex
	records []Record
}

func NewLedger() *Ledger { return &Ledger{} }
func (l *Ledger) Append(record Record) error {
	for field, value := range map[string]string{"id": record.ID, "tenant_id": record.TenantID, "environment": record.Environment, "actor_type": record.ActorType, "action": record.Action, "resource_type": record.ResourceType, "outcome": record.Outcome} {
		if strings.TrimSpace(value) == "" {
			return errors.New(field + " is required")
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	record.CreatedAt = record.CreatedAt.UTC()
	record.Context = cloneContext(record.Context)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, record)
	return nil
}
func (l *Ledger) List(tenantID string) []Record {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Record, 0)
	for _, record := range l.records {
		if tenantID == "" || record.TenantID == tenantID {
			record.Context = cloneContext(record.Context)
			result = append(result, record)
		}
	}
	return result
}
func cloneContext(context map[string]any) map[string]any {
	cloned := make(map[string]any, len(context))
	for key, value := range context {
		cloned[key] = value
	}
	return cloned
}
