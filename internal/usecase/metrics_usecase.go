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

func (u *metricsUsecase) GetDashboardMetrics(ctx context.Context, startDate, endDate string) (*domain.DashboardMetrics, error) {
	totalCost, err := u.repo.GetTotalCostThisMonth(ctx)
	if err != nil {
		return nil, err
	}

	topUsers, err := u.repo.GetTopUsersByCost(ctx, 5, startDate, endDate) // Top 5
	if err != nil {
		return nil, err
	}

	dailyTrend, err := u.repo.GetDailyUsageTrend(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	systemOverview, err := u.repo.GetSystemOverview(ctx, startDate, endDate)
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

func (u *metricsUsecase) GetMyMetrics(ctx context.Context, userID string) (*domain.MyMetrics, error) {
	totalCost, err := u.repo.GetMyTotalCostThisMonth(ctx, userID)
	if err != nil {
		return nil, err
	}

	dailyTrend, err := u.repo.GetMyDailyUsageTrend(ctx, userID, 30) // last 30 days
	if err != nil {
		return nil, err
	}

	recentActivities, err := u.repo.GetMyRecentActivity(ctx, userID, 5) // top 5 recent
	if err != nil {
		return nil, err
	}

	return &domain.MyMetrics{
		TotalCostThisMonth: totalCost,
		DailyTrend:         dailyTrend,
		RecentActivities:   recentActivities,
	}, nil
}

func (u *metricsUsecase) GetRiskHeatmap(ctx context.Context) ([]domain.RiskHeatmap, error) {
	return u.repo.GetRiskHeatmap(ctx)
}

func (u *metricsUsecase) GetExpiringContracts(ctx context.Context) ([]domain.ExpiringContract, error) {
	return u.repo.GetExpiringContracts(ctx)
}
