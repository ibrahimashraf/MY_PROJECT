package calibration

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	sharedcalibration "integin/internal/shared/calibration"
	"integin/internal/shared/types"
)

type ExpiryHook func(sharedcalibration.Record)
type Service struct {
	mu        sync.RWMutex
	records   map[string]sharedcalibration.Record
	onExpired ExpiryHook
}

func New(onExpired ExpiryHook) *Service {
	return &Service{records: make(map[string]sharedcalibration.Record), onExpired: onExpired}
}
func (s *Service) Create(record sharedcalibration.Record) error {
	if err := record.Validate(); err != nil {
		return err
	}
	record.CalibrationDate = record.CalibrationDate.UTC()
	record.NextDueDate = record.NextDueDate.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	key := record.TenantID + "|" + record.ID
	if _, exists := s.records[key]; exists {
		return errors.New("calibration record already exists")
	}
	s.records[key] = record
	return nil
}
func (s *Service) Get(tenantID, id string) (sharedcalibration.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[tenantID+"|"+id]
	if !ok {
		return sharedcalibration.Record{}, errors.New("calibration record not found")
	}
	return record, nil
}
func (s *Service) ExpireDue(at time.Time) []sharedcalibration.Record {
	at = at.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	expired := make([]sharedcalibration.Record, 0)
	for key, record := range s.records {
		if record.Status == types.CalibrationActive && !at.Before(record.NextDueDate) {
			record.Status = types.CalibrationExpired
			s.records[key] = record
			expired = append(expired, record)
			if s.onExpired != nil {
				s.onExpired(record)
			}
		}
	}
	return expired
}
func (s *Service) Supersede(tenantID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenantID + "|" + id
	record, ok := s.records[key]
	if !ok {
		return errors.New("calibration record not found")
	}
	if record.Status != types.CalibrationActive {
		return fmt.Errorf("calibration cannot supersede from %s", record.Status)
	}
	record.Status = types.CalibrationSuperseded
	s.records[key] = record
	return nil
}
func (s *Service) SubmissionAllowed(tenantID, equipmentID string, at time.Time) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, record := range s.records {
		if record.TenantID == tenantID && record.EquipmentID == equipmentID && record.Status == types.CalibrationActive && !at.UTC().Before(record.NextDueDate) {
			return errors.New("calibration is expired")
		}
	}
	for _, record := range s.records {
		if record.TenantID == tenantID && record.EquipmentID == equipmentID && record.Status == types.CalibrationActive {
			return nil
		}
	}
	return errors.New("active calibration record is required")
}
func ValidateEquipmentID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("equipment id is required")
	}
	return nil
}
