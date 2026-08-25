package workorderpg

import (
	"context"

	"integin/internal/domain/workorder"
)

// WithinTransaction intentionally delegates to the callback because Repository
// mutation methods own the authoritative SQL transaction and tenant context.
func (r *Repository) WithinTransaction(ctx context.Context, actor workorder.ActorContext, fn func(context.Context, workorder.Repository) error) error {
	if r == nil {
		return ErrNilDB
	}
	return fn(ctx, r)
}
