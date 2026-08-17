package workpackagepg

import (
	"context"
	"testing"
	"time"
)

func TestGetAssignmentContextRejectsNilRepository(t *testing.T) {
	var repository *Repository
	_, err := repository.GetAssignmentContext(context.Background(), "tenant-1", "organization-1", "inspection-1", "device-1", time.Now())
	if err == nil {
		t.Fatal("expected nil repository rejection")
	}
}
