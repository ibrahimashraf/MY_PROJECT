package workpackagepg

import (
	"context"
	"testing"
	"time"
)

func TestGetCurrentAssignmentRejectsUnavailableRepository(t *testing.T) {
	var repository *Repository
	_, err := repository.GetCurrentAssignment(
		context.Background(),
		"tenant-1",
		"org-1",
		"inspection-1",
		"device-1",
		time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("expected unavailable repository error")
	}
}
