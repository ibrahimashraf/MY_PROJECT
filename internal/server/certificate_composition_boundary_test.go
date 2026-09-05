package server

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/storage"
)

func TestNewCertificateHandlerFailsClosedOnMissingDependency(t *testing.T) {
	resolver := compositionTestResolver{}
	validator := &oidcauth.Validator{}
	db, err := sql.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		name      string
		database  *sql.DB
		validator *oidcauth.Validator
		resolver  identity.Resolver
	}{
		{name: "missing database", database: nil, validator: validator, resolver: resolver},
		{name: "missing validator", database: db, validator: nil, resolver: resolver},
		{name: "missing resolver", database: db, validator: validator, resolver: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if handler, err := NewCertificateHandler(test.database, test.validator, test.resolver); handler != nil || err == nil {
				t.Fatalf("expected dependency failure, handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestNewCertificateHandlerBuildsWithoutConnecting(t *testing.T) {
	db, err := sql.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	handler, err := NewCertificateHandler(db, &oidcauth.Validator{}, compositionTestResolver{})
	if err != nil {
		t.Fatalf("expected local composition to succeed without connecting, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed certificate handler")
	}
}

func TestNewCertificateHandlerWithArtifactStore(t *testing.T) {
	db, err := sql.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := storage.NewInMemoryStore()
	handler, err := NewCertificateHandlerWithStore(db, &oidcauth.Validator{}, compositionTestResolver{}, store)
	if err != nil {
		t.Fatalf("expected composition with store to succeed, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed certificate handler")
	}
}
