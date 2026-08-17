package workpackagepg

import (
	"database/sql"
	"testing"
)

func TestNewRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database rejection")
	}
}

func TestNewRepositoryAcceptsDatabaseHandle(t *testing.T) {
	repository, err := NewRepository(&sql.DB{})
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	if repository == nil {
		t.Fatal("expected repository")
	}
}
