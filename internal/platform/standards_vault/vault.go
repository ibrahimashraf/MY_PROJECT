package standards_vault

import (
	"context"
	"errors"
	"strings"
	"sync"

	"integin/internal/storage"
)

type Clause struct {
	Number  string
	Title   string
	Content string
}
type Standard struct {
	ID          string
	TenantID    string
	Code        string
	Version     string
	Title       string
	IssuingBody string
	FileKey     string
	FileType    string
	Checksum    string
	Clauses     []Clause
}
type Vault struct {
	mu        sync.RWMutex
	standards map[string]Standard
	objects   storage.Store
}

func New(objects storage.Store) *Vault {
	return &Vault{standards: make(map[string]Standard), objects: objects}
}
func (v *Vault) Add(standard Standard) error {
	for field, value := range map[string]string{"id": standard.ID, "tenant_id": standard.TenantID, "code": standard.Code, "version": standard.Version, "title": standard.Title} {
		if strings.TrimSpace(value) == "" {
			return errors.New(field + " is required")
		}
	}
	standard.Clauses = append([]Clause(nil), standard.Clauses...)
	v.mu.Lock()
	defer v.mu.Unlock()
	v.standards[standard.TenantID+"|"+standard.ID] = standard
	return nil
}
func (v *Vault) Get(tenantID, id string) (Standard, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	standard, ok := v.standards[tenantID+"|"+id]
	if !ok {
		return Standard{}, errors.New("standard not found")
	}
	standard.Clauses = append([]Clause(nil), standard.Clauses...)
	return standard, nil
}
func (v *Vault) Search(tenantID, query string) []Standard {
	v.mu.RLock()
	defer v.mu.RUnlock()
	query = strings.ToLower(strings.TrimSpace(query))
	result := make([]Standard, 0)
	for _, standard := range v.standards {
		if standard.TenantID != tenantID {
			continue
		}
		haystack := strings.ToLower(standard.Code + " " + standard.Version + " " + standard.Title + " " + standard.IssuingBody)
		for _, clause := range standard.Clauses {
			haystack += " " + strings.ToLower(clause.Number+" "+clause.Title+" "+clause.Content)
		}
		if query == "" || strings.Contains(haystack, query) {
			standard.Clauses = append([]Clause(nil), standard.Clauses...)
			result = append(result, standard)
		}
	}
	return result
}
func (v *Vault) Download(tenantID, id string) (storage.Object, error) {
	standard, err := v.Get(tenantID, id)
	if err != nil {
		return storage.Object{}, err
	}
	if v.objects == nil {
		return storage.Object{}, errors.New("object storage is not configured")
	}
	return v.objects.Get(context.Background(), standard.FileKey)
}
