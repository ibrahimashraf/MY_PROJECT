package domain

import (
	"errors"
	"fmt"
	"time"

	"integin/pkg/id"
)

type AssetStatus string

const (
	AssetStatusOperational    AssetStatus = "OPERATIONAL"
	AssetStatusInspected      AssetStatus = "INSPECTED"
	AssetStatusQuarantined    AssetStatus = "QUARANTINED"
	AssetStatusDecommissioned AssetStatus = "DECOMMISSIONED"
)

var (
	ErrEmptyAssetDID          = errors.New("asset DID cannot be empty")
	ErrEmptyManufacturer      = errors.New("manufacturer cannot be empty")
	ErrEmptySerialNumber      = errors.New("chassis serial number cannot be empty")
	ErrInvalidCustodyTransfer = errors.New("custody transfer missing required recipient or tenant")
	ErrEpochMismatch          = errors.New("custody transfer expected epoch does not match current passport epoch")
	ErrInvalidEventID         = errors.New("custody transfer event id is not a valid UUIDv7")
)

// CustodyTransferEvent documents a cross-border or inter-company transfer of an asset.
type CustodyTransferEvent struct {
	EventID          string    `json:"event_id"`
	PreviousTenantID string    `json:"previous_tenant_id"`
	NewTenantID      string    `json:"new_tenant_id"`
	CountryCodeISO2  string    `json:"country_code_iso2"` // Location of transfer
	TransferDate     time.Time `json:"transfer_date"`
	AuthorizedBy     string    `json:"authorized_by"`
	ReceiptHash      string    `json:"receipt_hash"`
	ExpectedEpoch    uint64    `json:"expected_epoch"`
}

// UniversalAssetPassport is the core W3C digital identity document for an industrial asset.
type UniversalAssetPassport struct {
	AssetDID          string                 `json:"asset_did"`          // "did:integin:asset:7f8a9e...421c"
	Manufacturer      string                 `json:"manufacturer"`       // e.g. "Liebherr", "Kato"
	ModelNumber       string                 `json:"model_number"`       // e.g. "NK-1000"
	ChassisSerial     string                 `json:"chassis_serial"`     // Physical serial number
	EquipmentCategory string                 `json:"equipment_category"` // Defined by dynamic standard category
	RatedSWL          string                 `json:"rated_swl"`          // e.g. "100 Ton"
	CustomAttributes  map[string]interface{} `json:"custom_attributes"`  // 100% dynamic user-defined fields
	CurrentStatus     AssetStatus            `json:"current_status"`     // "OPERATIONAL", "QUARANTINED"
	JurisdictionCode  string                 `json:"jurisdiction_code"`  // Active ISO country code ("SA", "US", "DE")
	RegisteredAt      time.Time              `json:"registered_at"`
	Epoch             uint64                 `json:"epoch"`
	ChainOfCustody    []CustodyTransferEvent `json:"chain_of_custody"`
}

// Validate ensures all required foundational identity fields are satisfied.
func (p UniversalAssetPassport) Validate() error {
	if p.AssetDID == "" {
		return ErrEmptyAssetDID
	}
	if p.Manufacturer == "" {
		return ErrEmptyManufacturer
	}
	if p.ChassisSerial == "" {
		return ErrEmptySerialNumber
	}
	return nil
}

// Quarantine locks the asset under safety hold and updates status.
func (p *UniversalAssetPassport) Quarantine(reason string) {
	p.CurrentStatus = AssetStatusQuarantined
	if p.CustomAttributes == nil {
		p.CustomAttributes = make(map[string]interface{})
	}
	p.CustomAttributes["quarantine_reason"] = reason
	p.CustomAttributes["quarantine_timestamp"] = time.Now().UTC().Format(time.RFC3339)
}

// TransferCustody appends a verified transfer record to the asset history.
func (p *UniversalAssetPassport) TransferCustody(transfer CustodyTransferEvent) error {
	if transfer.NewTenantID == "" || transfer.CountryCodeISO2 == "" {
		return ErrInvalidCustodyTransfer
	}
	p.ChainOfCustody = append(p.ChainOfCustody, transfer)
	p.JurisdictionCode = transfer.CountryCodeISO2
	return nil
}

// TransferCustodyFenced applies a fenced custody transfer. The transfer's
// ExpectedEpoch must equal the passport's current Epoch, otherwise the
// transfer is a stale replay or split-brain write and ErrEpochMismatch is
// returned. On success the epoch advances by one.
func (p *UniversalAssetPassport) TransferCustodyFenced(transfer CustodyTransferEvent) error {
	if transfer.ExpectedEpoch != p.Epoch {
		return ErrEpochMismatch
	}
	if transfer.EventID == "" {
		eventID, err := id.NewV7()
		if err != nil {
			return fmt.Errorf("generate custody event id: %w", err)
		}
		transfer.EventID = eventID
	} else if !id.IsValidV7(transfer.EventID) {
		return ErrInvalidEventID
	}
	if transfer.NewTenantID == "" || transfer.CountryCodeISO2 == "" {
		return ErrInvalidCustodyTransfer
	}

	p.Epoch++
	p.ChainOfCustody = append(p.ChainOfCustody, transfer)
	p.JurisdictionCode = transfer.CountryCodeISO2
	return nil
}
