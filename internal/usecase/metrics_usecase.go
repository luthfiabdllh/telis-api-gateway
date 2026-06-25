package usecase

import (
	"context"

	"telis-api-gateway/internal/domain"
)

type metricsUsecase struct {
	repo domain.MetricsRepository
}

func NewMetricsUsecase(repo domain.MetricsRepository) domain.MetricsUsecase {
	return &metricsUsecase{
		repo: repo,
	}
}

func (u *metricsUsecase) GetDashboardMetrics(ctx context.Context) (*domain.DashboardMetrics, error) {
	totalCost, err := u.repo.GetTotalCostThisMonth(ctx)
	if err != nil {
		return nil, err
	}

	topUsers, err := u.repo.GetTopUsersByCost(ctx, 5) // Top 5
	if err != nil {
		return nil, err
	}

	dailyTrend, err := u.repo.GetDailyUsageTrend(ctx, 30) // Last 30 days
	if err != nil {
		return nil, err
	}

	systemOverview, err := u.repo.GetSystemOverview(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.DashboardMetrics{
		TotalCostThisMonth: totalCost,
		TopUsers:           topUsers,
		DailyTrend:         dailyTrend,
		TotalUsers:         systemOverview.TotalUsers,
		TotalDocuments:     systemOverview.TotalDocuments,
		TotalFolders:       systemOverview.TotalFolders,
		DocStatusDist:      systemOverview.DocStatusDist,
		UserRoleDist:       systemOverview.UserRoleDist,
	}, nil
}

func (u *metricsUsecase) GetMyMetrics(ctx context.Context, userID string) (float64, error) {
	return u.repo.GetMyTotalCostThisMonth(ctx, userID)
}
