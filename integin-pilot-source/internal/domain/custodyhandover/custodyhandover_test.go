package custodyhandover

import (
	"testing"
	"time"
)

func TestActorContextValidation(t *testing.T) {
	tests := []struct {
		name    string
		actor   ActorContext
		wantErr bool
	}{
		{
			name: "valid actor",
			actor: ActorContext{
				TenantID:       "tenant-1",
				OrganizationID: "org-1",
				ActorID:        "user-1",
			},
			wantErr: false,
		},
		{
			name: "missing tenant",
			actor: ActorContext{
				TenantID:       "",
				OrganizationID: "org-1",
				ActorID:        "user-1",
			},
			wantErr: true,
		},
		{
			name: "missing org",
			actor: ActorContext{
				TenantID:       "tenant-1",
				OrganizationID: "",
				ActorID:        "user-1",
			},
			wantErr: true,
		},
		{
			name: "missing actor id",
			actor: ActorContext{
				TenantID:       "tenant-1",
				OrganizationID: "org-1",
				ActorID:        "   ",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.actor.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustodyRecordValidation(t *testing.T) {
	now := time.Now().UTC()
	validHash := ComputeVerificationHash("t1", "org1", "asset-1", "loc-1", "cust-1", CustodyActionCheckIn, now)

	rec := CustodyRecord{
		ID:               "cust-rec-1",
		TenantID:         "t1",
		OrganizationID:   "org1",
		AssetID:          "asset-1",
		WorkOrderID:      "wo-1",
		LocationID:       "loc-1",
		SiteName:         "North Terminal",
		CustodianID:      "cust-1",
		Action:           CustodyActionCheckIn,
		VerificationHash: validHash,
		RecordedBy:       "inspector-1",
		RecordedAt:       now,
	}

	if err := rec.Validate(); err != nil {
		t.Fatalf("expected valid custody record, got error: %v", err)
	}

	// Missing ID
	invalid := rec
	invalid.ID = ""
	if err := invalid.Validate(); err == nil {
		t.Fatalf("expected error on missing id, got nil")
	}

	// Missing Hash
	invalid = rec
	invalid.VerificationHash = ""
	if err := invalid.Validate(); err == nil {
		t.Fatalf("expected error on missing verification hash, got nil")
	}

	// Invalid action
	invalid = rec
	invalid.Action = "UNKNOWN_ACTION"
	if err := invalid.Validate(); err == nil {
		t.Fatalf("expected error on invalid action, got nil")
	}
}

func TestHandoverStateTransitions(t *testing.T) {
	cases := []struct {
		from    HandoverState
		to      HandoverState
		allowed bool
	}{
		{HandoverStateRequested, HandoverStateAcknowledged, true},
		{HandoverStateRequested, HandoverStateApproved, true},
		{HandoverStateRequested, HandoverStateRejected, true},
		{HandoverStateRequested, HandoverStateCancelled, true},
		{HandoverStateRequested, HandoverStateTransferred, false},

		{HandoverStateAcknowledged, HandoverStateApproved, true},
		{HandoverStateAcknowledged, HandoverStateRejected, true},
		{HandoverStateAcknowledged, HandoverStateCancelled, true},
		{HandoverStateAcknowledged, HandoverStateTransferred, false},

		{HandoverStateApproved, HandoverStateTransferred, true},
		{HandoverStateApproved, HandoverStateCancelled, true},
		{HandoverStateApproved, HandoverStateRejected, false},

		{HandoverStateTransferred, HandoverStateRequested, false},
		{HandoverStateRejected, HandoverStateApproved, false},
		{HandoverStateCancelled, HandoverStateTransferred, false},
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			res := CanTransition(tc.from, tc.to)
			if res != tc.allowed {
				t.Fatalf("CanTransition(%s, %s) = %v; want %v", tc.from, tc.to, res, tc.allowed)
			}
		})
	}
}

func TestHandoverRequestValidation(t *testing.T) {
	now := time.Now().UTC()

	req := HandoverRequest{
		ID:               "ho-1",
		TenantID:         "tenant-1",
		OrganizationID:   "org-1",
		WorkOrderID:      "wo-100",
		FromAssignmentID: "asg-1",
		ToAssignmentID:   "asg-2",
		FromSiteID:       "site-north",
		ToSiteID:         "site-south",
		State:            HandoverStateRequested,
		Reason:           "Shift change handover",
		RequestedBy:      "inspector-1",
		RequestedAt:      now,
		Revision:         1,
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid handover request, got error: %v", err)
	}

	// Same from/to assignment should fail
	badReq := req
	badReq.ToAssignmentID = badReq.FromAssignmentID
	if err := badReq.Validate(); err == nil {
		t.Fatalf("expected error when from and to assignment are identical")
	}

	// Revision < 1 should fail
	badReq = req
	badReq.Revision = 0
	if err := badReq.Validate(); err == nil {
		t.Fatalf("expected error when revision < 1")
	}

	// State ACKNOWLEDGED without acknowledge metadata
	ackReq := req
	ackReq.State = HandoverStateAcknowledged
	if err := ackReq.Validate(); err == nil {
		t.Fatalf("expected error for ACKNOWLEDGED state without acknowledged_by")
	}
	ackReq.AcknowledgedBy = "inspector-2"
	ackReq.AcknowledgedAt = &now
	if err := ackReq.Validate(); err != nil {
		t.Fatalf("expected valid ACKNOWLEDGED state, got: %v", err)
	}

	// State TRANSFERRED without transferred metadata
	transReq := req
	transReq.State = HandoverStateTransferred
	if err := transReq.Validate(); err == nil {
		t.Fatalf("expected error for TRANSFERRED state without transferred_by")
	}
	transReq.TransferredBy = "supervisor-1"
	transReq.TransferredAt = &now
	if err := transReq.Validate(); err != nil {
		t.Fatalf("expected valid TRANSFERRED state, got: %v", err)
	}

	// State REJECTED without rejection_reason
	rejReq := req
	rejReq.State = HandoverStateRejected
	if err := rejReq.Validate(); err == nil {
		t.Fatalf("expected error for REJECTED state without rejection_reason")
	}
	rejReq.RejectionReason = "Asset physically incomplete"
	if err := rejReq.Validate(); err != nil {
		t.Fatalf("expected valid REJECTED state, got: %v", err)
	}
}
