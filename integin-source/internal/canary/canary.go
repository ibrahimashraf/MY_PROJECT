package canary

import (
	"errors"
	"fmt"
	"strings"
)

type Health string

const (
	HealthUnknown Health = "UNKNOWN"
	HealthHealthy Health = "HEALTHY"
	HealthFailed  Health = "FAILED"
)

type Deployment struct {
	ID             string
	Candidate      string
	Baseline       string
	Health         Health
	TrafficPercent int
	RollbackReady  bool
}

func New(id, candidate, baseline string) (Deployment, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(candidate) == "" || strings.TrimSpace(baseline) == "" {
		return Deployment{}, errors.New("deployment id, candidate, and baseline are required")
	}
	return Deployment{ID: id, Candidate: candidate, Baseline: baseline, Health: HealthUnknown}, nil
}
func (d *Deployment) SetHealth(health Health) { d.Health = health }
func (d *Deployment) SetTraffic(percent int) error {
	if percent < 0 || percent > 100 {
		return errors.New("traffic percent must be between 0 and 100")
	}
	if d.Health != HealthHealthy && percent > 0 {
		return fmt.Errorf("healthy canary is required before traffic shift")
	}
	d.TrafficPercent = percent
	return nil
}
func (d *Deployment) MarkRollbackReady(ready bool) { d.RollbackReady = ready }
