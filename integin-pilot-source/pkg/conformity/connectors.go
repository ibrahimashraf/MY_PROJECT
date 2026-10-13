// Package conformity provides enterprise connector harnesses that dispatch
// asset administration shell exports into SAP PM (BAPI/RFC/REST) and IBM
// Maximo (REST/OSLC MBO) staging and CI environments — plus deterministic
// in-memory staging implementations. Network behavior is simulated (transient
// failures + retries); no third-party SDK is required.
package conformity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"integin/pkg/domain"
)

var (
	ErrTransientNetwork     = errors.New("conformity: simulated network failure (retry safe)")
	ErrInvalidSAPPayload    = errors.New("conformity: invalid SAP PM export payload")
	ErrInvalidMaximoPayload = errors.New("conformity: invalid Maximo export payload")
)

// SAPResponse is the SAP BAPI dispatch verdict.
type SAPResponse struct {
	Status        string // OK | QUEUED | REJECTED
	TransactionID string // SAP BAPI transaction (tcode) id
}

// MaximoResponse is the Maximo OSLC MBO sync verdict.
type MaximoResponse struct {
	Status string // OK | QUEUED | REJECTED
	Count  int    // MBO records accepted in the sync batch
}

// SAPConnector posts Plant Maintenance exports into SAP via BAPI.
type SAPConnector interface {
	PostEquipment(ctx context.Context, export domain.SAPPMExport) (SAPResponse, error)
}

// MaximoConnector syncs asset/location MBO exports into IBM Maximo.
type MaximoConnector interface {
	SyncAssets(ctx context.Context, export domain.MaximoAssetExport) (MaximoResponse, error)
}

// ValidateSAPPayload checks the SAP PM export against the CFIHOS mapping
// contract: at least one record must be present, every equipment record must
// carry an equipment number, and every AUSP characteristic row must be tagged
// with a CFIHOS classifier.
func ValidateSAPPayload(export domain.SAPPMExport) error {
	if len(export.EquipmentRecords) == 0 && len(export.FunctionalLocations) == 0 {
		return fmt.Errorf("%w: no equipment or functional location records", ErrInvalidSAPPayload)
	}
	for _, r := range export.EquipmentRecords {
		if r.EquipmentNumber == "" {
			return fmt.Errorf("%w: equipment record missing equipment number", ErrInvalidSAPPayload)
		}
	}
	for _, c := range export.CharacteristicValues {
		if !validCFIHOS(c.CFIHOSClassifier) {
			return fmt.Errorf("%w: characteristic %q has invalid CFIHOS classifier %q", ErrInvalidSAPPayload, c.CharacteristicName, c.CFIHOSClassifier)
		}
	}
	return nil
}

// ValidateMaximoPayload checks the Maximo export against the CFIHOS mapping
// contract: at least one MBO record must be present, every ASSET record must
// carry ASSETNUM, and every record must carry a CFIHOS classification.
func ValidateMaximoPayload(export domain.MaximoAssetExport) error {
	if len(export.Assets) == 0 && len(export.Locations) == 0 {
		return fmt.Errorf("%w: no asset or location MBO records", ErrInvalidMaximoPayload)
	}
	for _, a := range export.Assets {
		if a.AssetNum == "" {
			return fmt.Errorf("%w: asset record missing ASSETNUM", ErrInvalidMaximoPayload)
		}
		if !validCFIHOS(a.Classification) {
			return fmt.Errorf("%w: asset %q has invalid CFIHOS classification %q", ErrInvalidMaximoPayload, a.AssetNum, a.Classification)
		}
	}
	for _, l := range export.Locations {
		if l.Location == "" {
			return fmt.Errorf("%w: location record missing LOCATION", ErrInvalidMaximoPayload)
		}
		if !validCFIHOS(l.Classification) {
			return fmt.Errorf("%w: location %q has invalid CFIHOS classification %q", ErrInvalidMaximoPayload, l.Location, l.Classification)
		}
	}
	return nil
}

func validCFIHOS(classifier string) bool {
	return strings.HasPrefix(classifier, "CFIHOS.") && classifier != "CFIHOS."
}

// StagingSAPConnector is a deterministic in-memory SAP BAPI harness for
// staging and CI. It records every accepted export, validates the CFIHOS
// mapping, and simulates transient network failures when staged.
type StagingSAPConnector struct {
	mu           sync.Mutex
	submitted    []domain.SAPPMExport
	failuresLeft int
	nextTxn      uint64
}

// NewStagingSAPConnector returns an idle staging connector.
func NewStagingSAPConnector() *StagingSAPConnector {
	return &StagingSAPConnector{}
}

// PostEquipment validates the export and records it. A staged transient
// failure returns ErrTransientNetwork with a QUEUED status and records nothing.
func (c *StagingSAPConnector) PostEquipment(_ context.Context, export domain.SAPPMExport) (SAPResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failuresLeft > 0 {
		c.failuresLeft--
		return SAPResponse{Status: "QUEUED"}, ErrTransientNetwork
	}
	if err := ValidateSAPPayload(export); err != nil {
		return SAPResponse{Status: "REJECTED"}, err
	}
	c.submitted = append(c.submitted, cloneSAP(export))
	c.nextTxn++
	return SAPResponse{Status: "OK", TransactionID: fmt.Sprintf("BAPI-%05d", c.nextTxn)}, nil
}

// FailNext stages n consecutive transient network failures (deterministic).
func (c *StagingSAPConnector) FailNext(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failuresLeft = n
}

// Submitted returns copies of the recorded exports in submission order.
func (c *StagingSAPConnector) Submitted() []domain.SAPPMExport {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]domain.SAPPMExport, 0, len(c.submitted))
	for _, e := range c.submitted {
		out = append(out, cloneSAP(e))
	}
	return out
}

// PostEquipmentRetry dispatches the export, transparently retrying transient
// network failures for up to attempts total attempts.
func PostEquipmentRetry(ctx context.Context, c SAPConnector, export domain.SAPPMExport, attempts int) (SAPResponse, error) {
	return retry(attempts, func() (SAPResponse, error) { return c.PostEquipment(ctx, export) })
}

// StagingMaximoConnector is a deterministic in-memory Maximo OSLC MBO harness
// for staging and CI. It records every accepted export, validates the CFIHOS
// mapping, and simulates transient network failures when staged.
type StagingMaximoConnector struct {
	mu           sync.Mutex
	submitted    []domain.MaximoAssetExport
	failuresLeft int
}

// NewStagingMaximoConnector returns an idle staging connector.
func NewStagingMaximoConnector() *StagingMaximoConnector {
	return &StagingMaximoConnector{}
}

// SyncAssets validates the export and records it. A staged transient failure
// returns ErrTransientNetwork with a QUEUED status and records nothing.
func (c *StagingMaximoConnector) SyncAssets(_ context.Context, export domain.MaximoAssetExport) (MaximoResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failuresLeft > 0 {
		c.failuresLeft--
		return MaximoResponse{Status: "QUEUED"}, ErrTransientNetwork
	}
	if err := ValidateMaximoPayload(export); err != nil {
		return MaximoResponse{Status: "REJECTED"}, err
	}
	c.submitted = append(c.submitted, cloneMaximo(export))
	return MaximoResponse{Status: "OK", Count: len(export.Assets) + len(export.Locations)}, nil
}

// FailNext stages n consecutive transient network failures (deterministic).
func (c *StagingMaximoConnector) FailNext(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failuresLeft = n
}

// Submitted returns copies of the recorded exports in submission order.
func (c *StagingMaximoConnector) Submitted() []domain.MaximoAssetExport {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]domain.MaximoAssetExport, 0, len(c.submitted))
	for _, e := range c.submitted {
		out = append(out, cloneMaximo(e))
	}
	return out
}

// SyncAssetsRetry syncs the export, transparently retrying transient network
// failures for up to attempts total attempts.
func SyncAssetsRetry(ctx context.Context, c MaximoConnector, export domain.MaximoAssetExport, attempts int) (MaximoResponse, error) {
	return retry(attempts, func() (MaximoResponse, error) { return c.SyncAssets(ctx, export) })
}

func retry[T any](attempts int, fn func() (T, error)) (T, error) {
	if attempts < 1 {
		attempts = 1
	}
	for attempt := 0; ; attempt++ {
		resp, err := fn()
		if err == nil || !errors.Is(err, ErrTransientNetwork) || attempt+1 >= attempts {
			return resp, err
		}
	}
}

func cloneSAP(e domain.SAPPMExport) domain.SAPPMExport {
	out := e
	out.EquipmentRecords = append([]domain.EquipmentMasterRecord(nil), e.EquipmentRecords...)
	out.FunctionalLocations = append([]domain.FunctionalLocationRecord(nil), e.FunctionalLocations...)
	out.CharacteristicValues = append([]domain.CharacteristicRecord(nil), e.CharacteristicValues...)
	return out
}

func cloneMaximo(e domain.MaximoAssetExport) domain.MaximoAssetExport {
	out := e
	out.Assets = append([]domain.MaximoAsset(nil), e.Assets...)
	out.Locations = append([]domain.MaximoLocation(nil), e.Locations...)
	return out
}
