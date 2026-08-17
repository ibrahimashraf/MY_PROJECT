package packagemanifest

import (
	"context"
	"testing"
	"time"

	"integin/internal/domain/workpackage"
)

type contextResolverFake struct {
	tenantID       string
	organizationID string
	inspectionID   string
	deviceID       string
	called         bool
}

func (f *contextResolverFake) GetAssignmentContext(_ context.Context, tenantID, organizationID, inspectionID, deviceID string, _ time.Time) (workpackage.AssignmentContext, error) {
	f.called = true
	f.tenantID, f.organizationID, f.inspectionID, f.deviceID = tenantID, organizationID, inspectionID, deviceID
	return workpackage.AssignmentContext{RootAssetID: "asset-1", InspectionType: "thorough", ProcedureVersion: "v1", ScheduledAt: time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC)}, nil
}

func TestAssignmentContextResolverContractUsesServerScope(t *testing.T) {
	resolver := &contextResolverFake{}
	_, err := resolver.GetAssignmentContext(context.Background(), "tenant-1", "organization-1", "inspection-1", "device-1", time.Now())
	if err != nil || !resolver.called {
		t.Fatalf("expected resolver call without error: %v", err)
	}
	if resolver.tenantID != "tenant-1" || resolver.organizationID != "organization-1" || resolver.inspectionID != "inspection-1" || resolver.deviceID != "device-1" {
		t.Fatal("resolver scope was not preserved")
	}
}

func TestStaticAssignmentContextResolverRejectsIncompleteContext(t *testing.T) {
	resolver := StaticAssignmentContextResolver{}
	if _, err := resolver.GetAssignmentContext(context.Background(), "tenant-1", "organization-1", "inspection-1", "device-1", time.Now()); err == nil {
		t.Fatal("expected incomplete context rejection")
	}
}
