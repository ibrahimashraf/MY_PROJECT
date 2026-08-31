package tag

import (
	"context"
	"errors"
	"time"
)

type AssetTag struct {
	ID             string
	TenantID       string
	OrganizationID string
	AssetID        string
	TagType        string
	TagUID         string
	Status         string
	ProvisionedAt  time.Time
	ProvisionedBy  string
	CreatedAt      time.Time
}

type PhotoMarkup struct {
	ID             string
	TenantID       string
	OrganizationID string
	EvidenceID     string
	MarkupType     string
	Coordinates    string
	Color          string
	Label          string
	CreatedBy      string
	CreatedAt      time.Time
}

type Repository interface {
	CreateAssetTag(ctx context.Context, actor ActorContext, t AssetTag) (AssetTag, error)
	GetAssetTag(ctx context.Context, actor ActorContext, id string) (AssetTag, error)
	GetAssetTagByUID(ctx context.Context, actor ActorContext, tagType, tagUID string) (AssetTag, bool, error)
	ListAssetTags(ctx context.Context, actor ActorContext, assetID string) ([]AssetTag, error)
	UpdateAssetTag(ctx context.Context, actor ActorContext, t AssetTag) error
	DeleteAssetTag(ctx context.Context, actor ActorContext, id string) error

	CreatePhotoMarkup(ctx context.Context, actor ActorContext, m PhotoMarkup) (PhotoMarkup, error)
	GetPhotoMarkup(ctx context.Context, actor ActorContext, id string) (PhotoMarkup, error)
	ListPhotoMarkups(ctx context.Context, actor ActorContext, evidenceID string) ([]PhotoMarkup, error)
	UpdatePhotoMarkup(ctx context.Context, actor ActorContext, m PhotoMarkup) error
	DeletePhotoMarkup(ctx context.Context, actor ActorContext, id string) error
}

type ActorContext struct {
	TenantID       string
	OrganizationID string
	ActorID        string
}

func (a ActorContext) Validate() error {
	if a.TenantID == "" || a.OrganizationID == "" || a.ActorID == "" {
		return ErrInvalidActor
	}
	return nil
}

var ErrInvalidActor = errors.New("tag actor context is invalid")
