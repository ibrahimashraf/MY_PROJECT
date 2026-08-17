package packagemanifest

import (
	"context"
	"time"

	"integin/internal/domain/workpackage"
)

// AssignmentContextResolver loads immutable inspection-specific context after
// device scope and assignment identity have been established server-side.
// Production persistence will implement this after an additive migration.
type AssignmentContextResolver interface {
	GetAssignmentContext(
		ctx context.Context,
		tenantID string,
		organizationID string,
		inspectionID string,
		deviceID string,
		now time.Time,
	) (workpackage.AssignmentContext, error)
}

// StaticAssignmentContextResolver is deterministic test and controlled-source
// composition support. It must not be used as a substitute for PostgreSQL
// assignment-context persistence in a mounted runtime.
type StaticAssignmentContextResolver struct {
	Context workpackage.AssignmentContext
}

func (r StaticAssignmentContextResolver) GetAssignmentContext(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ time.Time,
) (workpackage.AssignmentContext, error) {
	if err := r.Context.Validate(); err != nil {
		return workpackage.AssignmentContext{}, err
	}
	return r.Context, nil
}
