package search

import "context"

type Repository interface {
	Search(ctx context.Context, req SearchRequest) (SearchResponse, error)
}
