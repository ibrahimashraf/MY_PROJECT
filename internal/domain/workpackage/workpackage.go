// INTEGIN work-package domain foundation: immutable, tenant-scoped package metadata.
// Server authority owns publication and submission acceptance; field clients only consume approved packages.
package workpackage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// PublicationState controls whether a package can be assigned to field devices.
// Draft and retired packages must never be treated as assignable field work.
type PublicationState string

const (
	PublicationDraft      PublicationState = "draft"
	PublicationApproved   PublicationState = "approved"
	PublicationRetired    PublicationState = "retired"
	PublicationSuperseded PublicationState = "superseded"
)

// FieldType is intentionally small and explicit. More specialised controls are
// added only through a versioned schema change, never inferred from a prompt.
type FieldType string

const (
	FieldText       FieldType = "text"
	FieldNumber     FieldType = "number"
	FieldBoolean    FieldType = "boolean"
	FieldChoice     FieldType = "choice"
	FieldPassFailNA FieldType = "pass_fail_na"
)

// FieldDefinition is a stable, renderable field in an approved package.
// ID is durable across compatible revisions and is the only value key accepted
// in a bound submission.
type FieldDefinition struct {
	ID       string    `json:"id"`
	Prompt   string    `json:"prompt"`
	Type     FieldType `json:"type"`
	Required bool      `json:"required"`
	Options  []string  `json:"options,omitempty"`
}

// Section is an ordered form segment. Its fields remain ordered in package
// hashing so every client and server agrees on the captured-field sequence.
type Section struct {
	ID     string            `json:"id"`
	Title  string            `json:"title"`
	Fields []FieldDefinition `json:"fields"`
}

// Package is the immutable approved unit distributed to a device. PackageHash
// covers all rendering and validation semantics except the hash itself.
type Package struct {
	ID              string           `json:"id"`
	TenantID        string           `json:"tenant_id"`
	OrganizationID  string           `json:"organization_id"`
	TemplateCode    string           `json:"template_code"`
	TemplateVersion int              `json:"template_version"`
	PackageVersion  int              `json:"package_version"`
	SchemaVersion   int              `json:"schema_version"`
	State           PublicationState `json:"state"`
	PackageHash     string           `json:"package_hash"`
	Sections        []Section        `json:"sections"`
}

// SubmissionBinding is sent with an inspection payload. It keeps a draft or
// completed inspection bound to the exact package that rendered its form.
type SubmissionBinding struct {
	PackageID        string   `json:"package_id"`
	PackageVersion   int      `json:"package_version"`
	PackageHash      string   `json:"package_hash"`
	CapturedFieldIDs []string `json:"captured_field_ids"`
}

// WithComputedHash returns a copy with the deterministic SHA-256 package hash.
// A package author must compute this hash before approval or distribution.
func (p Package) WithComputedHash() (Package, error) {
	if err := p.validateDefinition(); err != nil {
		return Package{}, err
	}
	canonical, err := p.canonicalBytes()
	if err != nil {
		return Package{}, err
	}
	sum := sha256.Sum256(canonical)
	p.PackageHash = "sha256:" + hex.EncodeToString(sum[:])
	return p, nil
}

// Validate verifies immutable package metadata and that the declared hash
// matches the canonical package definition.
func (p Package) Validate() error {
	if err := p.validateDefinition(); err != nil {
		return err
	}
	if !strings.HasPrefix(p.PackageHash, "sha256:") || len(p.PackageHash) != len("sha256:")+64 {
		return errors.New("work package hash must be a sha256 digest")
	}
	expected, err := p.WithComputedHash()
	if err != nil {
		return err
	}
	if p.PackageHash != expected.PackageHash {
		return errors.New("work package hash does not match package definition")
	}
	return nil
}

// ValidateSubmission enforces package identity, immutable version/hash binding,
// captured field sequence, known fields, required fields, and primitive type
// compatibility before any inspection workflow can accept a payload.
func (p Package) ValidateSubmission(binding SubmissionBinding, values map[string]any) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.State != PublicationApproved {
		return errors.New("submission requires an approved work package")
	}
	if binding.PackageID != p.ID || binding.PackageVersion != p.PackageVersion || binding.PackageHash != p.PackageHash {
		return errors.New("submission binding does not match approved work package")
	}

	fields := p.fieldsByID()
	expectedSequence := p.OrderedFieldIDs()
	if !sameSequence(binding.CapturedFieldIDs, expectedSequence) {
		return errors.New("submission captured-field sequence does not match approved work package")
	}

	for fieldID, value := range values {
		field, found := fields[fieldID]
		if !found {
			return fmt.Errorf("submission includes unknown field %q", fieldID)
		}
		if err := validateFieldValue(field, value); err != nil {
			return err
		}
	}
	for fieldID, field := range fields {
		value, present := values[fieldID]
		if field.Required && (!present || isBlank(value)) {
			return fmt.Errorf("required field %q is missing", fieldID)
		}
	}
	return nil
}

// OrderedFieldIDs is part of the signed package semantics. It deliberately
// preserves the author-approved section and field order.
func (p Package) OrderedFieldIDs() []string {
	ids := make([]string, 0)
	for _, section := range p.Sections {
		for _, field := range section.Fields {
			ids = append(ids, field.ID)
		}
	}
	return ids
}

func (p Package) fieldsByID() map[string]FieldDefinition {
	fields := make(map[string]FieldDefinition)
	for _, section := range p.Sections {
		for _, field := range section.Fields {
			fields[field.ID] = field
		}
	}
	return fields
}

func (p Package) validateDefinition() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.TenantID) == "" || strings.TrimSpace(p.OrganizationID) == "" {
		return errors.New("work package id, tenant id, and organization id are required")
	}
	if strings.TrimSpace(p.TemplateCode) == "" || p.TemplateVersion <= 0 || p.PackageVersion <= 0 || p.SchemaVersion <= 0 {
		return errors.New("work package template code and positive versions are required")
	}
	if p.State != PublicationDraft && p.State != PublicationApproved && p.State != PublicationRetired && p.State != PublicationSuperseded {
		return errors.New("work package state is invalid")
	}
	if len(p.Sections) == 0 {
		return errors.New("work package must include at least one section")
	}
	sectionIDs := make(map[string]struct{})
	fieldIDs := make(map[string]struct{})
	for _, section := range p.Sections {
		if strings.TrimSpace(section.ID) == "" || strings.TrimSpace(section.Title) == "" {
			return errors.New("work package section id and title are required")
		}
		if _, exists := sectionIDs[section.ID]; exists {
			return fmt.Errorf("duplicate section id %q", section.ID)
		}
		sectionIDs[section.ID] = struct{}{}
		if len(section.Fields) == 0 {
			return fmt.Errorf("section %q must include at least one field", section.ID)
		}
		for _, field := range section.Fields {
			if err := validateFieldDefinition(field); err != nil {
				return err
			}
			if _, exists := fieldIDs[field.ID]; exists {
				return fmt.Errorf("duplicate field id %q", field.ID)
			}
			fieldIDs[field.ID] = struct{}{}
		}
	}
	return nil
}

func validateFieldDefinition(field FieldDefinition) error {
	if strings.TrimSpace(field.ID) == "" || strings.TrimSpace(field.Prompt) == "" {
		return errors.New("field id and prompt are required")
	}
	switch field.Type {
	case FieldText, FieldNumber, FieldBoolean, FieldChoice, FieldPassFailNA:
	default:
		return fmt.Errorf("field %q has an unsupported type", field.ID)
	}
	if field.Type == FieldChoice {
		if len(field.Options) == 0 {
			return fmt.Errorf("choice field %q requires options", field.ID)
		}
		seen := make(map[string]struct{})
		for _, option := range field.Options {
			if strings.TrimSpace(option) == "" {
				return fmt.Errorf("choice field %q includes a blank option", field.ID)
			}
			if _, exists := seen[option]; exists {
				return fmt.Errorf("choice field %q includes a duplicate option", field.ID)
			}
			seen[option] = struct{}{}
		}
	}
	if field.Type != FieldChoice && len(field.Options) != 0 {
		return fmt.Errorf("non-choice field %q cannot define options", field.ID)
	}
	return nil
}

func validateFieldValue(field FieldDefinition, value any) error {
	if isBlank(value) {
		return nil
	}
	switch field.Type {
	case FieldText:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field %q requires text", field.ID)
		}
	case FieldNumber:
		switch value.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, json.Number:
		default:
			return fmt.Errorf("field %q requires a number", field.ID)
		}
	case FieldBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %q requires a boolean", field.ID)
		}
	case FieldChoice:
		choice, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %q requires a choice", field.ID)
		}
		for _, option := range field.Options {
			if choice == option {
				return nil
			}
		}
		return fmt.Errorf("field %q has an unapproved choice", field.ID)
	case FieldPassFailNA:
		status, ok := value.(string)
		if !ok || (status != "pass" && status != "fail" && status != "not_applicable") {
			return fmt.Errorf("field %q requires pass, fail, or not_applicable", field.ID)
		}
	}
	return nil
}

func isBlank(value any) bool {
	if value == nil {
		return true
	}
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) == ""
}

func sameSequence(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func (p Package) canonicalBytes() ([]byte, error) {
	type canonicalPackage struct {
		ID              string           `json:"id"`
		TenantID        string           `json:"tenant_id"`
		OrganizationID  string           `json:"organization_id"`
		TemplateCode    string           `json:"template_code"`
		TemplateVersion int              `json:"template_version"`
		PackageVersion  int              `json:"package_version"`
		SchemaVersion   int              `json:"schema_version"`
		State           PublicationState `json:"state"`
		Sections        []Section        `json:"sections"`
	}
	return json.Marshal(canonicalPackage{
		ID:              p.ID,
		TenantID:        p.TenantID,
		OrganizationID:  p.OrganizationID,
		TemplateCode:    p.TemplateCode,
		TemplateVersion: p.TemplateVersion,
		PackageVersion:  p.PackageVersion,
		SchemaVersion:   p.SchemaVersion,
		State:           p.State,
		Sections:        p.Sections,
	})
}
