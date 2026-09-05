package server

import (
	"context"
	"database/sql"
	"testing"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type compositionTestResolver struct{}

func (compositionTestResolver) Resolve(context.Context, identity.PrincipalKey) (identity.Membership, error) {
	return identity.Membership{}, nil
}

func TestNewWorkOrderPartialSubmissionHandlerFailsClosedOnMissingDependency(t *testing.T) {
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
			if handler, err := NewWorkOrderPartialSubmissionHandler(test.database, test.validator, test.resolver); handler != nil || err == nil {
				t.Fatalf("expected dependency failure, handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestNewWorkOrderPartialSubmissionHandlerBuildsWithoutConnecting(t *testing.T) {
	db, err := sql.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	handler, err := NewWorkOrderPartialSubmissionHandler(db, &oidcauth.Validator{}, compositionTestResolver{})
	if err != nil {
		t.Fatalf("expected local composition to succeed without connecting, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed handler")
	}
}

func TestNewWorkOrderHandoverHandlerFailsClosedOnMissingDependency(t *testing.T) {
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
			if handler, err := NewWorkOrderHandoverHandler(test.database, test.validator, test.resolver); handler != nil || err == nil {
				t.Fatalf("expected dependency failure, handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestNewWorkOrderHandoverHandlerBuildsWithoutConnecting(t *testing.T) {
	db, err := sql.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	handler, err := NewWorkOrderHandoverHandler(db, &oidcauth.Validator{}, compositionTestResolver{})
	if err != nil {
		t.Fatalf("expected local composition to succeed without connecting, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected composed handler")
	}
}
