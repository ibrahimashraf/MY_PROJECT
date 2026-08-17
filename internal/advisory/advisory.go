package advisory

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Zone string

const (
	ZoneVerdict            Zone = "VERDICT"
	ZoneCertificate        Zone = "CERTIFICATE"
	ZoneCalibration        Zone = "CALIBRATION"
	ZoneAuthorization      Zone = "AUTHORIZATION"
	ZoneSyncSecurity       Zone = "SYNC_SECURITY"
	ZonePublicQR           Zone = "PUBLIC_QR"
	ZoneInspectionApproval Zone = "INSPECTION_APPROVAL"
	ZoneMonitoring         Zone = "MONITORING"
	ZoneRegulation         Zone = "REGULATION"
)

func AIAllowed(zone Zone) bool { return zone == ZoneMonitoring || zone == ZoneRegulation }

type Insight struct {
	ID            string
	TenantID      string
	Lens          string
	Title         string
	Summary       string
	Severity      string
	Confidence    float64
	Rationale     string
	EvidenceRefs  []string
	Provider      string
	Model         string
	PromptVersion string
	Blocking      bool
	CreatedAt     time.Time
	Limitations   []string
}

type Request struct {
	TenantID     string
	Zone         Zone
	Lens         string
	Inputs       map[string]any
	EvidenceRefs []string
}
type ProviderResult struct {
	Title         string
	Summary       string
	Severity      string
	Confidence    float64
	Rationale     string
	Provider      string
	Model         string
	PromptVersion string
	Limitations   []string
}
type Provider interface {
	Generate(context.Context, Request) (ProviderResult, error)
}

func Generate(ctx context.Context, provider Provider, request Request, id string, now time.Time) (Insight, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(request.TenantID) == "" {
		return Insight{}, errors.New("insight id and tenant id are required")
	}
	if request.Zone == "" || !AIAllowed(request.Zone) {
		return Insight{}, errors.New("AI invocation is forbidden in this zone")
	}
	if provider == nil {
		return Insight{}, errors.New("advisory provider is required")
	}
	result, err := provider.Generate(ctx, request)
	if err != nil {
		return Insight{}, err
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		return Insight{}, errors.New("confidence must be between 0 and 1")
	}
	if strings.TrimSpace(result.Rationale) == "" {
		return Insight{}, errors.New("rationale is required")
	}
	if now.IsZero() {
		now = time.Now()
	}
	return Insight{ID: id, TenantID: request.TenantID, Lens: request.Lens, Title: result.Title, Summary: result.Summary, Severity: result.Severity, Confidence: result.Confidence, Rationale: result.Rationale, EvidenceRefs: append([]string(nil), request.EvidenceRefs...), Provider: result.Provider, Model: result.Model, PromptVersion: result.PromptVersion, Blocking: false, CreatedAt: now.UTC(), Limitations: append([]string(nil), result.Limitations...)}, nil
}

func EnsureAdvisory(insight Insight) error {
	if insight.Blocking {
		return errors.New("advisory insight cannot be blocking")
	}
	if strings.TrimSpace(insight.Rationale) == "" {
		return errors.New("advisory insight requires rationale")
	}
	return nil
}
