package repository

import (
	"context"
	"database/sql"
	"time"

	"telis-api-gateway/internal/domain"
)

type metricsRepository struct {
	db *sql.DB
}

func NewMetricsRepository(db *sql.DB) domain.MetricsRepository {
	return &metricsRepository{db: db}
}

func (r *metricsRepository) GetTotalCostThisMonth(ctx context.Context) (float64, error) {
	query := `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM agent.token_metrics
		WHERE EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM CURRENT_DATE)
		  AND EXTRACT(YEAR FROM timestamp) = EXTRACT(YEAR FROM CURRENT_DATE)
	`
	var total float64
	err := r.db.QueryRowContext(ctx, query).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *metricsRepository) GetTopUsersByCost(ctx context.Context, limit int) ([]domain.UserCost, error) {
	query := `
		SELECT m.user_id, u.email, u.username as name, SUM(m.cost_usd) as total_cost
		FROM agent.token_metrics m
		JOIN gateway.users u ON m.user_id = u.id
		GROUP BY m.user_id, u.email, u.username
		ORDER BY total_cost DESC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.UserCost
	for rows.Next() {
		var u domain.UserCost
		if err := rows.Scan(&u.UserID, &u.Email, &u.Name, &u.TotalCost); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *metricsRepository) GetDailyUsageTrend(ctx context.Context, days int) ([]domain.DailyUsage, error) {
	query := `
		SELECT DATE(timestamp) as date, SUM(total_tokens) as total_tokens, SUM(cost_usd) as cost_usd
		FROM agent.token_metrics
		WHERE timestamp >= CURRENT_DATE - interval '1 day' * $1
		GROUP BY DATE(timestamp)
		ORDER BY DATE(timestamp) ASC
	`
	rows, err := r.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []domain.DailyUsage
	for rows.Next() {
		var d domain.DailyUsage
		var date time.Time
		if err := rows.Scan(&date, &d.TotalTokens, &d.CostUSD); err != nil {
			return nil, err
		}
		d.Date = date.Format("2006-01-02")
		trends = append(trends, d)
	}
	return trends, nil
}

func (r *metricsRepository) GetMyTotalCostThisMonth(ctx context.Context, userID string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM agent.token_metrics
		WHERE user_id = $1
		  AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM CURRENT_DATE)
		  AND EXTRACT(YEAR FROM timestamp) = EXTRACT(YEAR FROM CURRENT_DATE)
	`
	var total float64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}
