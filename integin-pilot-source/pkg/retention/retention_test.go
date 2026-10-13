package retention

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestActiveLegalHoldBlocksPurge(t *testing.T) {
	ctx := context.Background()
	holds := []LegalHold{
		{ID: "h1", TenantID: "t1", EntityType: "inspection", EntityID: "e1", Status: LegalHoldActive},
	}
	ok, err := CanPurge(ctx, "t1", "inspection", "e1", time.Now().Add(-800*24*time.Hour), holds, nil)
	if ok || err != ErrLegalHoldActive {
		t.Fatalf("expected ErrLegalHoldActive, got ok=%v err=%v", ok, err)
	}
}

func TestReleasedHoldAllowsPurgeAfterRetentionWindow(t *testing.T) {
	ctx := context.Background()
	holds := []LegalHold{
		{ID: "h1", TenantID: "t1", EntityType: "inspection", EntityID: "e1", Status: LegalHoldReleased},
	}
	policy := &RetentionPolicy{RetentionDays: 365}
	ok, err := CanPurge(ctx, "t1", "inspection", "e1", time.Now().Add(-400*24*time.Hour), holds, policy)
	if err != nil || !ok {
		t.Fatalf("expected purge allowed, got ok=%v err=%v", ok, err)
	}
}

func TestRetentionWindowPreventsPrematureDeletion(t *testing.T) {
	ctx := context.Background()
	policy := &RetentionPolicy{RetentionDays: 730}
	ok, err := CanPurge(ctx, "t1", "inspection", "e1", time.Now().Add(-100*24*time.Hour), nil, policy)
	if ok || err != ErrRetentionPeriodActive {
		t.Fatalf("expected ErrRetentionPeriodActive, got ok=%v err=%v", ok, err)
	}
}

func TestExportSelfApprovalProhibited(t *testing.T) {
	ctx := context.Background()
	approval := &ExportApproval{
		ID:          "a1",
		TenantID:    "t1",
		ExportID:    "ex1",
		RequestedBy: "user1",
		Status:      ExportApprovalApproved,
		CreatedAt:   time.Now(),
	}
	err := AuthorizeExport(ctx, approval, "user1", "user1")
	if err != ErrSelfApprovalProhibited {
		t.Fatalf("expected ErrSelfApprovalProhibited, got %v", err)
	}
}

func TestExportNotApprovedStatusRejected(t *testing.T) {
	ctx := context.Background()
	approval := &ExportApproval{
		ID:          "a1",
		TenantID:    "t1",
		ExportID:    "ex1",
		RequestedBy: "user1",
		Status:      ExportApprovalPending,
		CreatedAt:   time.Now(),
	}
	err := AuthorizeExport(ctx, approval, "user1", "approver1")
	if err != ErrExportNotApproved {
		t.Fatalf("expected ErrExportNotApproved, got %v", err)
	}
}

func TestDeletionCertificateDeterministic(t *testing.T) {
	seed := []byte("secret-seed-1")
	ts := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	c1 := GenerateDeletionCertificate("t1", "inspection", "e1", "actor1", ts, seed)
	c2 := GenerateDeletionCertificate("t1", "inspection", "e1", "actor1", ts, seed)
	if c1.TombstoneHash != c2.TombstoneHash {
		t.Fatalf("identical inputs produced different hashes: %s vs %s", c1.TombstoneHash, c2.TombstoneHash)
	}
}

func TestDeletionCertificateTamperDetection(t *testing.T) {
	seed := []byte("secret-seed-1")
	ts := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	c1 := GenerateDeletionCertificate("t1", "inspection", "e1", "actor1", ts, seed)
	c2 := GenerateDeletionCertificate("t1", "inspection", "e1", "attacker", ts, seed)
	if c1.TombstoneHash == c2.TombstoneHash {
		t.Fatal("tampered input should produce different hash")
	}
}

func TestCanPurgeNilPolicyAllowsPurge(t *testing.T) {
	ctx := context.Background()
	ok, err := CanPurge(ctx, "t1", "inspection", "e1", time.Now().Add(-1*time.Hour), nil, nil)
	if err != nil || !ok {
		t.Fatalf("expected purge allowed with nil policy, got ok=%v err=%v", ok, err)
	}
}

func TestExportUnauthorizedApprover(t *testing.T) {
	ctx := context.Background()
	approval := &ExportApproval{
		ID:          "a1",
		TenantID:    "t1",
		ExportID:    "ex1",
		RequestedBy: "user1",
		Status:      ExportApprovalApproved,
		CreatedAt:   time.Now(),
	}
	err := AuthorizeExport(ctx, approval, "user2", "approver1")
	if err != ErrUnauthorizedApprover {
		t.Fatalf("expected ErrUnauthorizedApprover, got %v", err)
	}
}

func TestCanPurgeDifferentTenantHoldIgnored(t *testing.T) {
	ctx := context.Background()
	holds := []LegalHold{
		{ID: "h1", TenantID: "t2", EntityType: "inspection", EntityID: "e1", Status: LegalHoldActive},
	}
	ok, err := CanPurge(ctx, "t1", "inspection", "e1", time.Now().Add(-800*24*time.Hour), holds, nil)
	if err != nil || !ok {
		t.Fatalf("expected purge allowed (different tenant hold), got ok=%v err=%v", ok, err)
	}
}

func TestConcurrentEvaluationChecks(t *testing.T) {
	ctx := context.Background()
	holds := []LegalHold{
		{ID: "h1", TenantID: "t1", EntityType: "inspection", EntityID: "e1", Status: LegalHoldActive},
	}
	approval := &ExportApproval{
		ID:          "a1",
		TenantID:    "t1",
		ExportID:    "ex1",
		RequestedBy: "user1",
		Status:      ExportApprovalApproved,
		CreatedAt:   time.Now(),
	}
	var wg sync.WaitGroup
	const goroutines = 8
	const iterations = 2000
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				ok, err := CanPurge(ctx, "t1", "inspection", "e1", time.Now().Add(-800*24*time.Hour), holds, nil)
				if !ok && err != ErrLegalHoldActive {
					t.Errorf("unexpected CanPurge result: ok=%v err=%v", ok, err)
					return
				}
				if err := AuthorizeExport(ctx, approval, "user1", "approver1"); err != nil {
					t.Errorf("unexpected AuthorizeExport error: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
