package analytics

import "context"

type Repository interface {
	GetDashboard(ctx context.Context, req DashboardRequest) (DashboardResponse, error)
}
