package environment

import "testing"

func TestReleasePromotionHealthAndRollback(t *testing.T) {
	release, err := NewRelease("release-1", "2026.08.13")
	if err != nil {
		t.Fatal(err)
	}
	if err := release.Approve(); err != nil {
		t.Fatal(err)
	}
	if err := release.Promote([]string{"tenant-1"}); err != nil {
		t.Fatal(err)
	}
	if release.Environment != Live || release.State != ReleasePromoted || len(release.SelectedTenants) != 1 {
		t.Fatalf("unexpected promotion: %#v", release)
	}
	if err := release.HealthCheck(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := release.AddRecoveryPoint(RecoveryPoint{ID: "backup-1", StorageKey: "backups/release-1"}); err != nil {
		t.Fatal(err)
	}
	if err := release.Rollback("health regression"); err != nil {
		t.Fatal(err)
	}
	if release.State != ReleaseRolledBack || release.Environment != Testing || len(release.RecoveryPoints) != 1 {
		t.Fatalf("unexpected rollback: %#v", release)
	}
}

func TestReleaseHealthFailureIsExplicit(t *testing.T) {
	release, err := NewRelease("release-1", "2026.08.13")
	if err != nil {
		t.Fatal(err)
	}
	if err := release.Approve(); err != nil {
		t.Fatal(err)
	}
	if err := release.Promote([]string{"tenant-1"}); err != nil {
		t.Fatal(err)
	}
	if err := release.HealthCheck(func() error { return assertError{} }); err == nil || release.State != ReleaseFailed {
		t.Fatalf("expected explicit failed release: %#v %v", release, err)
	}
}

type assertError struct{}

func (assertError) Error() string { return "health failed" }
