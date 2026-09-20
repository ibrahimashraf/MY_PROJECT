package schedulinghttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/domain/scheduling"
	"integin/pkg/onboarding"
)

func TestHandler_CheckSkills(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	handler := Handler{
		Now: func() time.Time { return now },
	}

	validReq := checkSkillsRequest{
		Credential: onboarding.InspectorCredential{
			InspectorID:        "insp-1",
			VerificationStatus: onboarding.QualApproved,
			ValidFrom:          now.Add(-24 * time.Hour),
			ExpiresAt:          now.Add(24 * time.Hour),
		},
		RequiredSkills: []string{"skill-crane"},
		Competencies: []scheduling.TechnicianCompetency{
			{
				EquipmentTypeID: "skill-crane",
				Status:          scheduling.CompetencyStatusCurrent,
				ExpiresAt:       now.Add(48 * time.Hour),
			},
		},
	}

	body, _ := json.Marshal(validReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduling/check-skills", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res checkSkillsResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if !res.Passed {
		t.Fatalf("expected passed true, got false")
	}
}
