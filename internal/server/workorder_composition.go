package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/workorderauth"
	"integin/internal/workorderhttp"
	"integin/internal/workorderpg"
)

func NewWorkOrderPartialSubmissionHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil || validator == nil || resolver == nil {
		return nil, errors.New("work-order handler requires database, OIDC validator, and identity resolver")
	}
	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		return nil, err
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		return nil, err
	}
	return workorderhttp.Handler{Validator: validator, Resolver: resolver, Service: service}, nil
}

func NewWorkOrderEvidenceHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil || validator == nil || resolver == nil {
		return nil, errors.New("work-order evidence handler requires database, OIDC validator, and identity resolver")
	}
	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		return nil, err
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		return nil, err
	}
	return workorderhttp.EvidenceHandler{Validator: validator, Resolver: resolver, Service: service}, nil
}

func NewWorkOrderHandoverHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil || validator == nil || resolver == nil {
		return nil, errors.New("work-order handover handler requires database, OIDC validator, and identity resolver")
	}
	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		return nil, err
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		return nil, err
	}
	return workorderhttp.HandoverHandler{Validator: validator, Resolver: resolver, Service: service}, nil
}
