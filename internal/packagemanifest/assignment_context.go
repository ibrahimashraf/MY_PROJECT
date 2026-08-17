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
