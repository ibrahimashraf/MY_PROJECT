package assurance_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	
	"integin/internal/domain/assurance"
)

func TestRecord_EvaluateTransition(t *testing.T) {
	record := &assurance.Record{
		ID:           uuid.Must(uuid.NewV4()),
		State:        assurance.StateInProgress,
		LamportClock: 10,
		Revision:     1,
		EffectiveAt:  time.Now(),
	}

	// Test normal transition
	err := record.EvaluateTransition(assurance.StateEvidencePending, 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.State != assurance.StateEvidencePending {
		t.Errorf("expected state %s, got %s", assurance.StateEvidencePending, record.State)
	}
	if record.LamportClock != 11 {
		t.Errorf("expected lamport clock to advance to 11, got %d", record.LamportClock)
	}
	if record.Revision != 2 {
		t.Errorf("expected revision to advance to 2, got %d", record.Revision)
	}

	// Test terminal state blockage
	record.State = assurance.StateCondemned
	err = record.EvaluateTransition(assurance.StateAssured, 12)
	if err != assurance.ErrTerminalState {
		t.Errorf("expected ErrTerminalState, got %v", err)
	}
	if record.State != assurance.StateCondemned {
		t.Errorf("expected state to remain CONDEMNED, got %s", record.State)
	}

	// Test severity monotonicity blockage (NON_COMPLIANT -> PLANNED)
	record.State = assurance.StateNonCompliant
	err = record.EvaluateTransition(assurance.StatePlanned, 13)
	if err != assurance.ErrDemotionRejected {
		t.Errorf("expected ErrDemotionRejected, got %v", err)
	}
	if record.State != assurance.StateNonCompliant {
		t.Errorf("expected state to remain NON_COMPLIANT, got %s", record.State)
	}
}
