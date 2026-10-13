package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"integin/internal/certificatehttp"
	"integin/internal/certificatepg"
	"integin/internal/domain/certificateauthority"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/storage"
)

func NewCertificateHandler(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	return NewCertificateHandlerWithStore(database, validator, resolver, nil)
}

func NewCertificateHandlerWithStore(database *sql.DB, validator *oidcauth.Validator, resolver identity.Resolver, store storage.Store) (http.Handler, error) {
	if database == nil {
		return nil, errors.New("certificate handler requires database")
	}
	repository, err := certificatepg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	return NewCertificateHandlerFromLifecycleWithStore(repository, validator, resolver, store)
}

func NewCertificateHandlerFromLifecycle(lifecycle certificatehttp.Lifecycle, validator *oidcauth.Validator, resolver identity.Resolver) (http.Handler, error) {
	return NewCertificateHandlerFromLifecycleWithStore(lifecycle, validator, resolver, nil)
}

func NewCertificateHandlerFromLifecycleWithStore(lifecycle certificatehttp.Lifecycle, validator *oidcauth.Validator, resolver identity.Resolver, store storage.Store) (http.Handler, error) {
	if lifecycle == nil || validator == nil || resolver == nil {
		return nil, errors.New("certificate handler requires lifecycle, OIDC validator, and identity resolver")
	}
	var artifactRetriever certificatehttp.ArtifactRetriever
	if store != nil {
		if repo, ok := lifecycle.(*certificatepg.Repository); ok {
			artifactRetriever = &certificateArtifactStoreAdapter{
				repository: repo,
				store:      store,
			}
		}
	}
	return certificatehttp.Handler{
		Validator: validator,
		Actors: certificatehttp.LocalActorResolver{
			Memberships: resolver,
		},
		Lifecycle: lifecycle,
		Artifacts: artifactRetriever,
	}, nil
}

type certificateArtifactStoreAdapter struct {
	repository *certificatepg.Repository
	store      storage.Store
}

func (a *certificateArtifactStoreAdapter) GetArtifact(ctx context.Context, actor certificateauthority.ActorContext, certificateID, artifactType string) (certificatepg.ArtifactRecord, error) {
	return a.repository.GetArtifact(ctx, actor, certificateID, artifactType)
}

func (a *certificateArtifactStoreAdapter) Get(ctx context.Context, key string) (storage.Object, error) {
	return a.store.Get(ctx, key)
}
