package advisory

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func validModel() ModelRegistration {
	return ModelRegistration{
		ID:        "model-1",
		Provider:  "provider-1",
		ModelName: "model-name",
		Version:   "1.0",
		Status:    ModelApproved,
		MaxTokens: 4096,
		AllowedZones: []Zone{
			ZoneMonitoring, ZoneRegulation, ZoneNDTDefect, ZoneLiftingDefect,
		},
	}
}

func validPrompt() PromptRegistration {
	return PromptRegistration{
		ID:               "prompt-1",
		Version:          "v1",
		SystemPromptHash: "abc",
		InputSchemaHash:  "def",
		Status:           PromptActive,
	}
}

func validAudit() AuditRecord {
	return AuditRecord{
		ID:            "audit-1",
		TenantID:      "tenant-1",
		InsightID:     "insight-1",
		ModelID:       "model-1",
		PromptVersion: "v1",
		Zone:          ZoneMonitoring,
		Lens:          "TREND_ANOMALY",
		EvidenceRefs:  []string{"event-1"},
		Confidence:    0.8,
		Blocking:      false,
	}
}

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry()
	if err := r.RegisterModel(validModel()); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterPrompt(validPrompt()); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRegisterModelApprovedAndDeprecated(t *testing.T) {
	r := testRegistry(t)

	deprecated := validModel()
	deprecated.ID = "model-deprecated"
	deprecated.Status = ModelDeprecated
	if err := r.RegisterModel(deprecated); err != nil {
		t.Fatal(err)
	}

	if err := r.ValidateAdvisoryInvocation("model-1", "v1", ZoneMonitoring); err != nil {
		t.Fatalf("approved model should validate: %v", err)
	}
	if err := r.ValidateAdvisoryInvocation("model-deprecated", "v1", ZoneMonitoring); !errors.Is(err, ErrModelNotApproved) {
		t.Fatalf("deprecated model should be rejected, got %v", err)
	}
	if err := r.ValidateAdvisoryInvocation("model-missing", "v1", ZoneMonitoring); !errors.Is(err, ErrModelNotApproved) {
		t.Fatalf("unknown model should be rejected, got %v", err)
	}
}

func TestRegisterPromptActiveAndSuperseded(t *testing.T) {
	r := testRegistry(t)

	superseded := validPrompt()
	superseded.ID = "prompt-superseded"
	superseded.Version = "v0"
	superseded.Status = PromptSuperseded
	if err := r.RegisterPrompt(superseded); err != nil {
		t.Fatal(err)
	}

	if err := r.ValidateAdvisoryInvocation("model-1", "v1", ZoneMonitoring); err != nil {
		t.Fatalf("active prompt should validate: %v", err)
	}
	if err := r.ValidateAdvisoryInvocation("model-1", "v0", ZoneMonitoring); !errors.Is(err, ErrPromptNotApproved) {
		t.Fatalf("superseded prompt should be rejected, got %v", err)
	}
	if err := r.ValidateAdvisoryInvocation("model-1", "missing", ZoneMonitoring); !errors.Is(err, ErrPromptNotApproved) {
		t.Fatalf("unknown prompt should be rejected, got %v", err)
	}
}

func TestPromptLookupByVersion(t *testing.T) {
	r := testRegistry(t)
	if err := r.ValidateAdvisoryInvocation("model-1", "v1", ZoneMonitoring); err != nil {
		t.Fatalf("prompt lookup by version should validate: %v", err)
	}
}

func TestZoneRestrictionsPerModel(t *testing.T) {
	r := testRegistry(t)

	narrow := validModel()
	narrow.ID = "model-narrow"
	narrow.AllowedZones = []Zone{ZoneMonitoring}
	if err := r.RegisterModel(narrow); err != nil {
		t.Fatal(err)
	}

	if err := r.ValidateAdvisoryInvocation("model-narrow", "v1", ZoneMonitoring); err != nil {
		t.Fatalf("allowed zone should validate: %v", err)
	}
	if err := r.ValidateAdvisoryInvocation("model-narrow", "v1", ZoneRegulation); !errors.Is(err, ErrZoneNotAllowedForModel) {
		t.Fatalf("zone outside model allowlist should be rejected, got %v", err)
	}
}

func TestAIFreeZonesAlwaysRejected(t *testing.T) {
	r := testRegistry(t)
	zones := []Zone{
		ZoneVerdict, ZoneCertificate, ZoneCalibration, ZoneAuthorization,
		ZoneSyncSecurity, ZonePublicQR, ZoneInspectionApproval,
	}
	for _, zone := range zones {
		if err := r.ValidateAdvisoryInvocation("model-1", "v1", zone); !errors.Is(err, ErrZoneNotAllowedForModel) {
			t.Fatalf("AI-free zone %q should always be rejected, got %v", zone, err)
		}
	}
}

func TestRecordAuditRejectsBlocking(t *testing.T) {
	r := testRegistry(t)
	audit := validAudit()
	audit.Blocking = true
	if err := r.RecordAudit(audit); !errors.Is(err, ErrBlockingMustBeFalse) {
		t.Fatalf("blocking audit should be rejected, got %v", err)
	}
}

func TestRecordAuditRejectsInvalidConfidence(t *testing.T) {
	r := testRegistry(t)
	for _, c := range []float64{-0.1, 1.1, 2, -1} {
		audit := validAudit()
		audit.ID = "audit-" + string(rune('a'+int(c*10)))
		audit.Confidence = c
		if err := r.RecordAudit(audit); err == nil {
			t.Fatalf("confidence %v should be rejected", c)
		}
	}
}

func TestRecordAuditRejectsNaNInfConfidence(t *testing.T) {
	r := testRegistry(t)
	for _, c := range []float64{nan64(), inf64()} {
		audit := validAudit()
		audit.Confidence = c
		if err := r.RecordAudit(audit); err == nil {
			t.Fatalf("confidence %v should be rejected", c)
		}
	}
}

func TestRecordAuditStoresAndValidates(t *testing.T) {
	r := testRegistry(t)
	if err := r.RecordAudit(validAudit()); err != nil {
		t.Fatal(err)
	}
	empty := validAudit()
	empty.ID, empty.TenantID, empty.InsightID = "", "", ""
	if err := r.RecordAudit(empty); err == nil {
		t.Fatal("empty audit identity should be rejected")
	}
}

func TestSubmitFeedbackDispositionValidation(t *testing.T) {
	r := testRegistry(t)
	feedback := InspectorFeedback{ID: "fb-1", TenantID: "tenant-1", AuditID: "audit-1", InspectorID: "inspector-1", Disposition: "ACCEPTED", CreatedAt: time.Now()}
	if err := r.SubmitFeedback(feedback); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"ACCEPTED", "REJECTED", "IGNORED", "CORRECTED"} {
		feedback.ID = "fb-" + d
		feedback.Disposition = d
		if err := r.SubmitFeedback(feedback); err != nil {
			t.Fatalf("disposition %q should be accepted: %v", d, err)
		}
	}
	for _, d := range []string{"APPROVED", "PENDING", "", "RANDOM"} {
		feedback.Disposition = d
		if err := r.SubmitFeedback(feedback); !errors.Is(err, ErrInvalidDisposition) {
			t.Fatalf("disposition %q should be rejected, got %v", d, err)
		}
	}
}

func TestSubmitFeedbackRequiresIdentity(t *testing.T) {
	r := testRegistry(t)
	feedback := InspectorFeedback{Disposition: "IGNORED"}
	if err := r.SubmitFeedback(feedback); err == nil {
		t.Fatal("empty feedback identity should be rejected")
	}
}

func TestRegisterRejectsInvalidInputs(t *testing.T) {
	r := NewRegistry()
	model := validModel()
	model.ID = ""
	if err := r.RegisterModel(model); err == nil {
		t.Fatal("empty model id should be rejected")
	}
	model = validModel()
	model.Status = "BOGUS"
	if err := r.RegisterModel(model); err == nil {
		t.Fatal("invalid model status should be rejected")
	}
	prompt := validPrompt()
	prompt.SystemPromptHash = ""
	if err := r.RegisterPrompt(prompt); err == nil {
		t.Fatal("empty prompt hash should be rejected")
	}
	prompt = validPrompt()
	prompt.Status = "BOGUS"
	if err := r.RegisterPrompt(prompt); err == nil {
		t.Fatal("invalid prompt status should be rejected")
	}
}

func TestRegistryConcurrency(t *testing.T) {
	r := testRegistry(t)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			audit := validAudit()
			audit.ID = "audit-" + string(rune('a'+i%26))
			if err := r.RecordAudit(audit); err != nil {
				t.Errorf("RecordAudit: %v", err)
			}
			if err := r.ValidateAdvisoryInvocation("model-1", "v1", ZoneMonitoring); err != nil {
				t.Errorf("Validate: %v", err)
			}
		}(i)
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m := validModel()
			m.ID = "model-" + string(rune('a'+i))
			if err := r.RegisterModel(m); err != nil {
				t.Errorf("RegisterModel: %v", err)
			}
		}(i)
	}
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			f := InspectorFeedback{ID: "fb-" + string(rune('a'+i)), TenantID: "tenant-1", AuditID: "audit-1", InspectorID: "inspector-1", Disposition: "ACCEPTED", CreatedAt: time.Now()}
			if err := r.SubmitFeedback(f); err != nil {
				t.Errorf("SubmitFeedback: %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func nan64() float64 {
	return math.NaN()
}

func inf64() float64 {
	return math.Inf(0)
}
