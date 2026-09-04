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
	if database == nil || validator == nil || resolver == nil {
		return nil, errors.New("certificate handler requires database, OIDC validator, and identity resolver")
	}
	repository, err := certificatepg.NewRepository(database)
	if err != nil {
		return nil, err
	}
	var artifactRetriever certificatehttp.ArtifactRetriever
	if store != nil {
		artifactRetriever = &certificateArtifactStoreAdapter{
			repository: repository,
			store:      store,
		}
	}
	return certificatehttp.Handler{
		Validator: validator,
		Actors: certificatehttp.LocalActorResolver{
			Memberships: resolver,
		},
		Lifecycle: repository,
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