package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type LoadChartDataStatus string

const (
	LoadChartDataVerified LoadChartDataStatus = "VERIFIED"
	LoadChartDataDraft    LoadChartDataStatus = "DRAFT"
	LoadChartDataDemo     LoadChartDataStatus = "DEMO"
)

type LoadChartUnit string

const LoadChartUnitTonne LoadChartUnit = "t"

var (
	ErrLoadChartIDRequired             = errors.New("load chart id is required")
	ErrLoadChartOEMRequired            = errors.New("load chart OEM is required")
	ErrLoadChartModelRequired          = errors.New("load chart model is required")
	ErrLoadChartUnitUnsupported        = errors.New("load chart unit is unsupported")
	ErrLoadChartStatusInvalid          = errors.New("load chart data status is invalid")
	ErrLoadChartProvenanceRequired     = errors.New("load chart provenance is incomplete")
	ErrLoadChartHashInvalid            = errors.New("load chart source hash is invalid")
	ErrLoadChartEffectiveDateInvalid   = errors.New("load chart effective dates are invalid")
	ErrLoadChartConfigurationsRequired = errors.New("load chart configurations are required")
	ErrLoadChartConfigurationID        = errors.New("load chart configuration id is required")
	ErrLoadChartConfigurationDuplicate = errors.New("load chart configuration id is duplicated")
	ErrLoadChartConfigurationValues    = errors.New("load chart configuration values are invalid")
	ErrLoadChartPointsRequired         = errors.New("load chart points are required")
	ErrLoadChartPointValues            = errors.New("load chart point values are invalid")
	ErrLoadChartPointOrder             = errors.New("load chart points are not strictly ordered by radius")
	ErrLoadChartNotVerified            = errors.New("load chart is not verified for authoritative use")
	ErrLoadChartConfigurationNotFound  = errors.New("load chart configuration was not found")
	ErrLoadChartPointNotFound          = errors.New("load chart point was not found")
)

type LoadChartProvenance struct {
	Source          string `json:"source"`
	DocumentID      string `json:"document_id"`
	Revision        string `json:"revision"`
	RightsReference string `json:"rights_reference"`
	SHA256          string `json:"sha256"`
}

type LoadChartPoint struct {
	RadiusM   float64 `json:"radius_m"`
	CapacityT float64 `json:"capacity_t"`
}

type LoadChartConfiguration struct {
	ID              string           `json:"id"`
	BoomLengthM     float64          `json:"boom_length_m"`
	JibLengthM      float64          `json:"jib_length_m"`
	CounterweightT  float64          `json:"counterweight_t"`
	OutriggerSpanXM float64          `json:"outrigger_span_x_m"`
	OutriggerSpanZM float64          `json:"outrigger_span_z_m"`
	Points          []LoadChartPoint `json:"points"`
}

type LoadChart struct {
	ID             string                   `json:"id"`
	OEM            string                   `json:"oem"`
	Model          string                   `json:"model"`
	Unit           LoadChartUnit            `json:"unit"`
	Status         LoadChartDataStatus      `json:"status"`
	Provenance     LoadChartProvenance      `json:"provenance"`
	EffectiveFrom  time.Time                `json:"effective_from"`
	EffectiveTo    *time.Time               `json:"effective_to,omitempty"`
	Configurations []LoadChartConfiguration `json:"configurations"`
}

func (c LoadChart) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return ErrLoadChartIDRequired
	}
	if strings.TrimSpace(c.OEM) == "" {
		return ErrLoadChartOEMRequired
	}
	if strings.TrimSpace(c.Model) == "" {
		return ErrLoadChartModelRequired
	}
	if c.Unit != LoadChartUnitTonne {
		return fmt.Errorf("%w: %q", ErrLoadChartUnitUnsupported, c.Unit)
	}
	switch c.Status {
	case LoadChartDataVerified, LoadChartDataDraft, LoadChartDataDemo:
	default:
		return fmt.Errorf("%w: %q", ErrLoadChartStatusInvalid, c.Status)
	}
	if err := validateLoadChartProvenance(c.Provenance); err != nil {
		return err
	}
	if c.EffectiveFrom.IsZero() {
		return fmt.Errorf("%w: effective_from is required", ErrLoadChartEffectiveDateInvalid)
	}
	if c.EffectiveTo != nil && !c.EffectiveTo.After(c.EffectiveFrom) {
		return fmt.Errorf("%w: effective_to must be after effective_from", ErrLoadChartEffectiveDateInvalid)
	}
	if len(c.Configurations) == 0 {
		return ErrLoadChartConfigurationsRequired
	}
	seen := make(map[string]struct{}, len(c.Configurations))
	for _, config := range c.Configurations {
		if strings.TrimSpace(config.ID) == "" {
			return ErrLoadChartConfigurationID
		}
		if _, exists := seen[config.ID]; exists {
			return fmt.Errorf("%w: %q", ErrLoadChartConfigurationDuplicate, config.ID)
		}
		seen[config.ID] = struct{}{}
		if err := validateLoadChartConfiguration(config); err != nil {
			return err
		}
	}
	return nil
}

func (c LoadChart) ValidateForAuthority() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Status != LoadChartDataVerified {
		return fmt.Errorf("%w: %s", ErrLoadChartNotVerified, c.Status)
	}
	return nil
}

func (c LoadChart) EffectiveAt(at time.Time) (bool, error) {
	if err := c.Validate(); err != nil {
		return false, err
	}
	if at.IsZero() {
		return false, fmt.Errorf("%w: evaluation time is required", ErrLoadChartEffectiveDateInvalid)
	}
	if at.Before(c.EffectiveFrom) {
		return false, nil
	}
	if c.EffectiveTo != nil && !at.Before(*c.EffectiveTo) {
		return false, nil
	}
	return true, nil
}

func (c LoadChart) LookupCapacityExact(configurationID string, radiusM float64) (float64, error) {
	if err := c.ValidateForAuthority(); err != nil {
		return 0, err
	}
	if math.IsNaN(radiusM) || math.IsInf(radiusM, 0) || radiusM <= 0 {
		return 0, fmt.Errorf("%w: radius must be finite and > 0", ErrLoadChartPointValues)
	}
	for _, config := range c.Configurations {
		if config.ID != configurationID {
			continue
		}
		for _, point := range config.Points {
			if point.RadiusM == radiusM {
				return point.CapacityT, nil
			}
		}
		return 0, fmt.Errorf("%w: configuration %q radius %g", ErrLoadChartPointNotFound, configurationID, radiusM)
	}
	return 0, fmt.Errorf("%w: %q", ErrLoadChartConfigurationNotFound, configurationID)
}

func validateLoadChartProvenance(provenance LoadChartProvenance) error {
	if strings.TrimSpace(provenance.Source) == "" ||
		strings.TrimSpace(provenance.DocumentID) == "" ||
		strings.TrimSpace(provenance.Revision) == "" ||
		strings.TrimSpace(provenance.RightsReference) == "" {
		return ErrLoadChartProvenanceRequired
	}
	if !validLoadChartSHA256(provenance.SHA256) {
		return fmt.Errorf("%w: %q", ErrLoadChartHashInvalid, provenance.SHA256)
	}
	return nil
}

func validLoadChartSHA256(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validateLoadChartConfiguration(config LoadChartConfiguration) error {
	if !finitePositive(config.BoomLengthM) ||
		math.IsNaN(config.JibLengthM) || math.IsInf(config.JibLengthM, 0) || config.JibLengthM < 0 ||
		math.IsNaN(config.CounterweightT) || math.IsInf(config.CounterweightT, 0) || config.CounterweightT < 0 ||
		!finitePositive(config.OutriggerSpanXM) || !finitePositive(config.OutriggerSpanZM) {
		return fmt.Errorf("%w: configuration %q", ErrLoadChartConfigurationValues, config.ID)
	}
	if len(config.Points) == 0 {
		return fmt.Errorf("%w: configuration %q", ErrLoadChartPointsRequired, config.ID)
	}
	previousRadius := math.Inf(-1)
	for _, point := range config.Points {
		if !finitePositive(point.RadiusM) || !finitePositive(point.CapacityT) {
			return fmt.Errorf("%w: configuration %q", ErrLoadChartPointValues, config.ID)
		}
		if point.RadiusM <= previousRadius {
			return fmt.Errorf("%w: configuration %q", ErrLoadChartPointOrder, config.ID)
		}
		previousRadius = point.RadiusM
	}
	return nil
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
