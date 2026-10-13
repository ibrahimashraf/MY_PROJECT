package domain

import (
	"errors"
	"time"

	"integin/pkg/id"
)

// AASItemClass discriminates the two CFIHOS-affiliated entities an IEC 63278
// Asset Administration Shell can represent: the facility slot (tag / functional
// location) and the physical serialized machine installed into it.
type AASItemClass string

const (
	ClassFunctionalLocation AASItemClass = "FUNCTIONAL_LOCATION" // e.g. "TAG-RIG01-CRANE-A"
	ClassInstalledEquipment AASItemClass = "INSTALLED_EQUIPMENT" // e.g. "SER-LIEBHERR-99401"
)

const (
	MountActionMount   = "MOUNT"
	MountActionUnmount = "UNMOUNT"
)

var (
	ErrEmptyAASID              = errors.New("asset administration shell id cannot be empty")
	ErrInvalidAASClass         = errors.New("asset administration shell class is invalid")
	ErrAASNoFunctionalLocation = errors.New("shell is not a CFIHOS functional-location tag")
	ErrAASNoInstalledEquipment = errors.New("shell is not an installed equipment item")
	ErrEquipmentAlreadyMounted = errors.New("installed equipment item is already mounted to a functional location")
	ErrTagOccupied             = errors.New("functional location already holds a mounted equipment item")
	ErrNoMountedEquipment      = errors.New("functional location has no mounted equipment item")
	ErrMountedSerialMismatch   = errors.New("unmount serial does not match the mounted equipment item")
)

// DocumentRef references a sealed external artifact (mill test certificate,
// NDT report, capacity chart, sealed certificate).
type DocumentRef struct {
	DocumentID string    `json:"document_id"`
	Title      string    `json:"title"`
	SealedAt   time.Time `json:"sealed_at"`
	SealedBy   string    `json:"sealed_by"`
	Hash       string    `json:"hash"`
}

// ProofLoadRecord documents one proof-load test result on the asset.
type ProofLoadRecord struct {
	TestID      string    `json:"test_id"`
	PerformedAt time.Time `json:"performed_at"`
	LoadTonnes  float64   `json:"load_tonnes"`
	Result      string    `json:"result"` // "PASS" | "FAIL"
}

// TechnicalSpecification is the AAS submodel holding OEM metadata, mill test
// certificates, rated SWL, material heat numbers, and OEM capacity charts.
type TechnicalSpecification struct {
	OEM                  string        `json:"oem"`
	Model                string        `json:"model"`
	SerialNumber         string        `json:"serial_number"`
	RatedSWL             string        `json:"rated_swl"`
	MaterialHeatNumbers  []string      `json:"material_heat_numbers,omitempty"`
	MillTestCertificates []DocumentRef `json:"mill_test_certificates,omitempty"`
	CapacityCharts       []DocumentRef `json:"capacity_charts,omitempty"`
}

// OperationalState is the AAS submodel holding the live operational snapshot:
// custody tenant, ISO jurisdiction, accumulated load cycles, status, and the
// fenced custody epoch.
type OperationalState struct {
	CustodyTenantID        string      `json:"custody_tenant_id"`
	JurisdictionISOCode    string      `json:"jurisdiction_iso_code"`
	AccumulativeLoadCycles uint64      `json:"accumulative_load_cycles"`
	CurrentStatus          AssetStatus `json:"current_status"`
	Epoch                  uint64      `json:"epoch"`
	LastUpdatedAt          time.Time   `json:"last_updated_at"`
}

// AssuranceEvidence is the AAS submodel holding sealed certificates, NDT
// reports, and proof-load test records.
type AssuranceEvidence struct {
	SealedCertificates []DocumentRef     `json:"sealed_certificates,omitempty"`
	NDTReports         []DocumentRef     `json:"ndt_reports,omitempty"`
	ProofLoadTests     []ProofLoadRecord `json:"proof_load_tests,omitempty"`
}

// AssetSubmodels is the canonical IEC 63278 submodel container of an AAS.
type AssetSubmodels struct {
	TechnicalSpecification TechnicalSpecification `json:"technical_specification"`
	OperationalState       OperationalState       `json:"operational_state"`
	AssuranceEvidence      AssuranceEvidence      `json:"assurance_evidence"`
}

// FunctionalLocation is the CFIHOS tag / functional location: the designated
// asset slot in the facility hierarchy (e.g. "TAG-RIG01-CRANE-A").
type FunctionalLocation struct {
	TagID         string `json:"tag_id"`
	FacilityID    string `json:"facility_id"`
	HierarchyPath string `json:"hierarchy_path"`
	Description   string `json:"description"`
	MountedSerial string `json:"mounted_serial,omitempty"`
}

// InstalledEquipment is the CFIHOS physical serialized machine (e.g. serial
// "SER-LIEBHERR-99401"). It owns its own passport (DID identity + fenced
// custody epoch) and independent DPP lifecycle.
type InstalledEquipment struct {
	SerialNumber string                 `json:"serial_number"`
	Description  string                 `json:"description,omitempty"`
	Passport     UniversalAssetPassport `json:"passport"`
	MountedTagID string                 `json:"mounted_tag_id,omitempty"`
}

// MountEvent records one mount/unmount transition. Each shell keeps its own
// independent mount history.
type MountEvent struct {
	EventID         string    `json:"event_id"`
	Action          string    `json:"action"`
	TagID           string    `json:"tag_id"`
	EquipmentSerial string    `json:"equipment_serial"`
	OccurredAt      time.Time `json:"occurred_at"`
	AuthorizedBy    string    `json:"authorized_by"`
}

// AssetAdministrationShell is the canonical IEC 63278 container unifying the
// UniversalAssetPassport (DID identity) and the ProductPassportDPP
// (regulatory state) with the three sovereign twin submodels. Depending on
// its Class it represents either a CFIHOS functional-location tag or an
// installed equipment item.
type AssetAdministrationShell struct {
	ID           string              `json:"id"`
	Class        AASItemClass        `json:"class"`
	Location     *FunctionalLocation `json:"functional_location,omitempty"`
	Equipment    *InstalledEquipment `json:"installed_equipment,omitempty"`
	Submodels    AssetSubmodels      `json:"submodels"`
	MountHistory []MountEvent        `json:"mount_history,omitempty"`
}

// NewFunctionalLocationAAS builds a tag shell for a CFIHOS functional location.
func NewFunctionalLocationAAS(id string, loc FunctionalLocation) *AssetAdministrationShell {
	return &AssetAdministrationShell{
		ID:       id,
		Class:    ClassFunctionalLocation,
		Location: &loc,
	}
}

// NewInstalledEquipmentAAS builds a machine shell for a physical serialized
// equipment item, unifying the embedded universal passport with the sovereign
// twin submodels.
func NewInstalledEquipmentAAS(id string, eq InstalledEquipment) *AssetAdministrationShell {
	return &AssetAdministrationShell{
		ID:        id,
		Class:     ClassInstalledEquipment,
		Equipment: &eq,
	}
}

// Validate enforces that the shell observes the class-specific identity
// contract.
func (s AssetAdministrationShell) Validate() error {
	if s.ID == "" {
		return ErrEmptyAASID
	}
	switch s.Class {
	case ClassFunctionalLocation:
		if s.Location == nil || s.Location.TagID == "" {
			return ErrAASNoFunctionalLocation
		}
	case ClassInstalledEquipment:
		if s.Equipment == nil || s.Equipment.SerialNumber == "" {
			return ErrAASNoInstalledEquipment
		}
		if err := s.Equipment.Passport.Validate(); err != nil {
			return err
		}
	default:
		return ErrInvalidAASClass
	}
	return nil
}

// MountEquipment binds an installed equipment shell onto this
// functional-location tag shell. Both shells record the transition in their
// own independent mount history; the equipment item keeps its own passport,
// submodels, and lifecycle untouched.
func (s *AssetAdministrationShell) MountEquipment(eq *AssetAdministrationShell, authorizedBy string) error {
	if s.Class != ClassFunctionalLocation || s.Location == nil {
		return ErrAASNoFunctionalLocation
	}
	if eq == nil || eq.Class != ClassInstalledEquipment || eq.Equipment == nil {
		return ErrAASNoInstalledEquipment
	}
	if s.Location.MountedSerial != "" {
		return ErrTagOccupied
	}
	if eq.Equipment.MountedTagID != "" {
		return ErrEquipmentAlreadyMounted
	}

	eventID, err := id.NewV7()
	if err != nil {
		return err
	}
	ev := MountEvent{
		EventID:         eventID,
		Action:          MountActionMount,
		TagID:           s.ID,
		EquipmentSerial: eq.Equipment.SerialNumber,
		OccurredAt:      time.Now().UTC(),
		AuthorizedBy:    authorizedBy,
	}
	s.Location.MountedSerial = eq.Equipment.SerialNumber
	eq.Equipment.MountedTagID = s.ID
	s.MountHistory = append(s.MountHistory, ev)
	eq.MountHistory = append(eq.MountHistory, ev)
	return nil
}

// UnmountEquipment detaches the given installed equipment shell from this
// functional-location tag shell. Independent history on both shells is
// preserved.
func (s *AssetAdministrationShell) UnmountEquipment(eq *AssetAdministrationShell, authorizedBy string) error {
	if s.Class != ClassFunctionalLocation || s.Location == nil {
		return ErrAASNoFunctionalLocation
	}
	if eq == nil || eq.Class != ClassInstalledEquipment || eq.Equipment == nil {
		return ErrAASNoInstalledEquipment
	}
	if s.Location.MountedSerial == "" {
		return ErrNoMountedEquipment
	}
	if s.Location.MountedSerial != eq.Equipment.SerialNumber {
		return ErrMountedSerialMismatch
	}

	eventID, err := id.NewV7()
	if err != nil {
		return err
	}
	ev := MountEvent{
		EventID:         eventID,
		Action:          MountActionUnmount,
		TagID:           s.ID,
		EquipmentSerial: eq.Equipment.SerialNumber,
		OccurredAt:      time.Now().UTC(),
		AuthorizedBy:    authorizedBy,
	}
	s.Location.MountedSerial = ""
	eq.Equipment.MountedTagID = ""
	s.MountHistory = append(s.MountHistory, ev)
	eq.MountHistory = append(eq.MountHistory, ev)
	return nil
}

// SyncOperationalSubmodel mirrors the embedded passport identity state into
// the OperationalState submodel so the two never diverge.
func (s *AssetAdministrationShell) SyncOperationalSubmodel() error {
	if s.Class != ClassInstalledEquipment || s.Equipment == nil {
		return ErrAASNoInstalledEquipment
	}
	p := s.Equipment.Passport
	custodyTenant := ""
	if n := len(p.ChainOfCustody); n > 0 {
		custodyTenant = p.ChainOfCustody[n-1].NewTenantID
	}
	s.Submodels.OperationalState.CustodyTenantID = custodyTenant
	s.Submodels.OperationalState.JurisdictionISOCode = p.JurisdictionCode
	s.Submodels.OperationalState.CurrentStatus = p.CurrentStatus
	s.Submodels.OperationalState.Epoch = p.Epoch
	s.Submodels.OperationalState.LastUpdatedAt = time.Now().UTC()
	return nil
}

// TransferEquipmentCustody applies a fenced custody transfer to the installed
// equipment item's passport (the single epoch authority, ExpectedEpoch ==
// CurrentEpoch) and keeps the OperationalState submodel synchronized. Tag
// shells carry no custody epoch and are rejected.
func (s *AssetAdministrationShell) TransferEquipmentCustody(t CustodyTransferEvent) error {
	if s.Class != ClassInstalledEquipment || s.Equipment == nil {
		return ErrAASNoInstalledEquipment
	}
	if err := s.Equipment.Passport.TransferCustodyFenced(t); err != nil {
		return err
	}
	s.Submodels.OperationalState.CustodyTenantID = t.NewTenantID
	s.Submodels.OperationalState.JurisdictionISOCode = t.CountryCodeISO2
	s.Submodels.OperationalState.CurrentStatus = s.Equipment.Passport.CurrentStatus
	s.Submodels.OperationalState.Epoch = s.Equipment.Passport.Epoch
	s.Submodels.OperationalState.LastUpdatedAt = time.Now().UTC()
	return nil
}
