package scheduling

import (
	"errors"
	"testing"
	"time"

	"integin/pkg/onboarding"
)

var (
	startOfMonth = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	midMonth     = time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	endOfMonth   = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
)

func approvedCredential() onboarding.InspectorCredential {
	return onboarding.InspectorCredential{
		CredentialID:       "cred-1",
		InspectorID:        "inspector-1",
		Discipline:         "lifting.inspection",
		VerificationStatus: onboarding.QualApproved,
		ValidFrom:          startOfMonth,
		ExpiresAt:          endOfMonth,
	}
}

func currentCompetency(skill string) TechnicianCompetency {
	return TechnicianCompetency{
		ID:              "comp-" + skill,
		TechnicianID:    "inspector-1",
		EquipmentTypeID: skill,
		ExpiresAt:       endOfMonth,
		Status:          CompetencyStatusCurrent,
	}
}

func TestTechnicianCompetencyIsCurrent(t *testing.T) {
	cases := []struct {
		name string
		comp TechnicianCompetency
		at   time.Time
		want bool
	}{
		{"current unexpired", currentCompetency("lifting.inspection"), midMonth, true},
		{"current zero expiry", TechnicianCompetency{Status: CompetencyStatusCurrent}, midMonth, true},
		{"expired status", TechnicianCompetency{Status: CompetencyStatusExpired, ExpiresAt: endOfMonth}, midMonth, false},
		{"past expiry", currentCompetency("lifting.inspection"), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.comp.IsCurrent(tc.at); got != tc.want {
				t.Fatalf("IsCurrent(%v) = %v, want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestCheckAssignmentSkillsAllowsVerifiedInspector(t *testing.T) {
	skills := []string{"lifting.inspection", "geotech.assessment"}
	comps := []TechnicianCompetency{currentCompetency("lifting.inspection"), currentCompetency("geotech.assessment")}
	if err := CheckAssignmentSkills(midMonth, approvedCredential(), skills, comps); err != nil {
		t.Fatalf("verified inspector should be allowed: %v", err)
	}
}

func TestCheckAssignmentSkillsAllowsNoSkillRequirements(t *testing.T) {
	if err := CheckAssignmentSkills(midMonth, approvedCredential(), nil, nil); err != nil {
		t.Fatalf("assignment without required skills should be allowed: %v", err)
	}
}

func TestCheckAssignmentSkillsDeniesUnapprovedCredential(t *testing.T) {
	cred := approvedCredential()
	for _, status := range []onboarding.QualificationVerificationStatus{onboarding.QualPendingReview, onboarding.QualExpired, onboarding.QualRevoked} {
		cred.VerificationStatus = status
		if err := CheckAssignmentSkills(midMonth, cred, []string{"lifting.inspection"}, nil); err == nil {
			t.Fatalf("credential %q should be denied", status)
		} else if !errors.Is(err, ErrAssignmentNotVerified) {
			t.Fatalf("expected ErrAssignmentNotVerified wrap, got %v", err)
		}
	}
}

func TestCheckAssignmentSkillsDeniesOutOfWindowCredential(t *testing.T) {
	cred := approvedCredential()
	if err := CheckAssignmentSkills(startOfMonth.Add(-time.Hour), cred, []string{"lifting.inspection"}, nil); err == nil {
		t.Fatal("credential before valid_from should be denied")
	}
	if err := CheckAssignmentSkills(endOfMonth, cred, []string{"lifting.inspection"}, nil); err == nil {
		t.Fatal("credential on expiry should be denied")
	}
}

func TestCheckAssignmentSkillsDeniesIncompleteSkillMatrix(t *testing.T) {
	cred := approvedCredential()
	if err := CheckAssignmentSkills(midMonth, cred, []string{"lifting.inspection", "geotech.assessment"}, []TechnicianCompetency{currentCompetency("lifting.inspection")}); err == nil {
		t.Fatal("missing skill-matrix competency should be denied")
	}
	if err := CheckAssignmentSkills(midMonth, cred, []string{"lifting.inspection"}, nil); err == nil {
		t.Fatal("empty skill matrix should be denied when skills are required")
	}
}

func TestCheckAssignmentSkillsDeniesExpiredCompetency(t *testing.T) {
	comp := currentCompetency("lifting.inspection")
	comp.Status = CompetencyStatusExpired
	if err := CheckAssignmentSkills(midMonth, approvedCredential(), []string{"lifting.inspection"}, []TechnicianCompetency{comp}); err == nil {
		t.Fatal("expired competency should be denied")
	}
}

func TestAssignEntryGatesCalendarEntry(t *testing.T) {
	entry := &CalendarEntry{ID: "cal-1", WorkOrderID: "wo-1", TechnicianID: "inspector-1"}
	skills := []string{"lifting.inspection"}
	comps := []TechnicianCompetency{currentCompetency("lifting.inspection")}
	if err := AssignEntry(midMonth, entry, approvedCredential(), skills, comps); err != nil {
		t.Fatalf("verified assignment should pass the gate: %v", err)
	}
	if !entry.CompetencyVerified {
		t.Fatal("verified assignment should be marked CompetencyVerified")
	}
	denied := &CalendarEntry{ID: "cal-2", WorkOrderID: "wo-2", TechnicianID: "inspector-1"}
	if err := AssignEntry(midMonth, denied, approvedCredential(), skills, nil); err == nil {
		t.Fatal("assigning without a skill-matrix competency should be denied")
	}
	if denied.CompetencyVerified {
		t.Fatal("denied assignment must leave CompetencyVerified unset")
	}
}
