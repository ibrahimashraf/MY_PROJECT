package advisory

import (
	"errors"
	"math"
	"strings"
	"sync"
	"time"
)

type ModelStatus string

const (
	ModelApproved   ModelStatus = "APPROVED"
	ModelDeprecated ModelStatus = "DEPRECATED"
)

type PromptStatus string

const (
	PromptActive     PromptStatus = "ACTIVE"
	PromptSuperseded PromptStatus = "SUPERSEDED"
)

var (
	ErrModelNotApproved       = errors.New("advisory: model is not approved")
	ErrPromptNotApproved      = errors.New("advisory: prompt version is not active")
	ErrZoneNotAllowedForModel = errors.New("advisory: zone not allowed for model")
	ErrBlockingMustBeFalse    = errors.New("advisory: blocking must be false")
	ErrInvalidDisposition     = errors.New("advisory: invalid disposition")
)

type ModelRegistration struct {
	ID           string
	Provider     string
	ModelName    string
	Version      string
	Status       ModelStatus
	MaxTokens    int
	AllowedZones []Zone
}

type PromptRegistration struct {
	ID               string
	Version          string
	SystemPromptHash string
	InputSchemaHash  string
	Status           PromptStatus
}

type AuditRecord struct {
	ID            string
	TenantID      string
	InsightID     string
	ModelID       string
	PromptVersion string
	Zone          Zone
	Lens          string
	EvidenceRefs  []string
	Confidence    float64
	Blocking      bool
}

type InspectorFeedback struct {
	ID          string
	TenantID    string
	AuditID     string
	InspectorID string
	Disposition string
	Notes       string
	CreatedAt   time.Time
}

type Registry struct {
	mu       sync.RWMutex
	models   map[string]ModelRegistration
	prompts  map[string]PromptRegistration
	audits   map[string]AuditRecord
	feedback map[string]InspectorFeedback
}

func NewRegistry() *Registry {
	return &Registry{
		models:   make(map[string]ModelRegistration),
		prompts:  make(map[string]PromptRegistration),
		audits:   make(map[string]AuditRecord),
		feedback: make(map[string]InspectorFeedback),
	}
}

func (r *Registry) RegisterModel(m ModelRegistration) error {
	if strings.TrimSpace(m.ID) == "" || strings.TrimSpace(m.Provider) == "" || strings.TrimSpace(m.ModelName) == "" || strings.TrimSpace(m.Version) == "" {
		return errors.New("advisory: model id, provider, model name, and version are required")
	}
	if m.Status != ModelApproved && m.Status != ModelDeprecated {
		return errors.New("advisory: model status must be APPROVED or DEPRECATED")
	}
	if m.MaxTokens <= 0 {
		return errors.New("advisory: max tokens must be positive")
	}
	zones := append([]Zone(nil), m.AllowedZones...)
	clone := m
	clone.AllowedZones = zones
	r.mu.Lock()
	r.models[clone.ID] = clone
	r.mu.Unlock()
	return nil
}

func (r *Registry) RegisterPrompt(p PromptRegistration) error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Version) == "" {
		return errors.New("advisory: prompt id and version are required")
	}
	if strings.TrimSpace(p.SystemPromptHash) == "" || strings.TrimSpace(p.InputSchemaHash) == "" {
		return errors.New("advisory: prompt hashes are required")
	}
	if p.Status != PromptActive && p.Status != PromptSuperseded {
		return errors.New("advisory: prompt status must be ACTIVE or SUPERSEDED")
	}
	r.mu.Lock()
	r.prompts[p.ID] = p
	r.mu.Unlock()
	return nil
}

func (r *Registry) ValidateAdvisoryInvocation(modelID, promptVersion string, zone Zone) error {
	r.mu.RLock()
	m, okModel := r.models[modelID]
	p, okPrompt := r.prompts[promptVersion]
	if !okPrompt {
		// Allow lookup by prompt ID or by prompt version string.
		for _, candidate := range r.prompts {
			if candidate.Version == promptVersion {
				p, okPrompt = candidate, true
				break
			}
		}
	}
	r.mu.RUnlock()
	if !okModel || m.Status != ModelApproved {
		return ErrModelNotApproved
	}
	if !okPrompt || p.Status != PromptActive {
		return ErrPromptNotApproved
	}
	if !AIAllowed(zone) {
		return ErrZoneNotAllowedForModel
	}
	for _, allowed := range m.AllowedZones {
		if allowed == zone {
			return nil
		}
	}
	return ErrZoneNotAllowedForModel
}

func (r *Registry) RecordAudit(a AuditRecord) error {
	if a.Blocking {
		return ErrBlockingMustBeFalse
	}
	if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.InsightID) == "" {
		return errors.New("advisory: audit id, tenant id, and insight id are required")
	}
	if math.IsNaN(a.Confidence) || math.IsInf(a.Confidence, 0) || a.Confidence < 0 || a.Confidence > 1 {
		return errors.New("advisory: confidence must be between 0 and 1")
	}
	record := a
	record.EvidenceRefs = append([]string(nil), a.EvidenceRefs...)
	r.mu.Lock()
	r.audits[record.ID] = record
	r.mu.Unlock()
	return nil
}

func (r *Registry) SubmitFeedback(f InspectorFeedback) error {
	switch f.Disposition {
	case "ACCEPTED", "REJECTED", "IGNORED", "CORRECTED":
	default:
		return ErrInvalidDisposition
	}
	if strings.TrimSpace(f.ID) == "" || strings.TrimSpace(f.TenantID) == "" || strings.TrimSpace(f.AuditID) == "" || strings.TrimSpace(f.InspectorID) == "" {
		return errors.New("advisory: feedback id, tenant id, audit id, and inspector id are required")
	}
	r.mu.Lock()
	r.feedback[f.ID] = f
	r.mu.Unlock()
	return nil
}
