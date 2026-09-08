package server

import (
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/platform/calibration"
	"integin/internal/workorderauth"
	"integin/internal/workorderhttp"
	"integin/internal/workorderpg"
)

func NewWorkOrderPartialSubmissionHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("work-order handler requires database")
	}
	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		return nil, err
	}
	calibrationStore, err := calibration.NewStore(database)
	if err != nil {
		return nil, err
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New(), CalibrationGate: calibrationStore})
	if err != nil {
		return nil, err
	}
	return NewWorkOrderPartialSubmissionHandlerFromService(service, validator, resolver)
}

func NewWorkOrderPartialSubmissionHandlerFromService(service workorder.Service, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if service == nil || validator == nil || resolver == nil {
		return nil, errors.New("work-order handler requires service, OIDC validator, and identity resolver")
	}
	return workorderhttp.Handler{Validator: validator, Resolver: resolver, Service: service}, nil
}

func NewWorkOrderEvidenceHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("work-order evidence handler requires database")
	}
	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		return nil, err
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		return nil, err
	}
	return NewWorkOrderEvidenceHandlerFromService(service, validator, resolver)
}

func NewWorkOrderEvidenceHandlerFromService(service workorder.Service, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if service == nil || validator == nil || resolver == nil {
		return nil, errors.New("work-order evidence handler requires service, OIDC validator, and identity resolver")
	}
	return workorderhttp.EvidenceHandler{Validator: validator, Resolver: resolver, Service: service}, nil
}

func NewWorkOrderHandoverHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("work-order handover handler requires database")
	}
	repository, err := workorderpg.NewRepository(database, workorderpg.NewPostgresInspectionMembershipValidator())
	if err != nil {
		return nil, err
	}
	service, err := workorder.NewService(workorder.ServiceDependencies{Repository: repository, Transactions: repository, Authorizer: workorderauth.New()})
	if err != nil {
		return nil, err
	}
	return NewWorkOrderHandoverHandlerFromService(service, validator, resolver)
}

func NewWorkOrderHandoverHandlerFromService(service workorder.Service, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	if service == nil || validator == nil || resolver == nil {
		return nil, errors.New("work-order handover handler requires service, OIDC validator, and identity resolver")
	}
	return workorderhttp.HandoverHandler{Validator: validator, Resolver: resolver, Service: service}, nil
}
