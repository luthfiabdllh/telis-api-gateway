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

func (r *metricsRepository) GetSystemOverview(ctx context.Context) (*domain.SystemOverview, error) {
	overview := &domain.SystemOverview{}

	// 1. Total Users
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM gateway.users").Scan(&overview.TotalUsers); err != nil {
		return nil, err
	}

	// 2. Total Documents (excluding DELETED)
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM ingestion.documents WHERE status != 'DELETED'").Scan(&overview.TotalDocuments); err != nil {
		return nil, err
	}

	// 3. Total Folders
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM ingestion.folders").Scan(&overview.TotalFolders); err != nil {
		return nil, err
	}

	// 4. Document Status Distribution
	docQuery := `SELECT status, count(*) FROM ingestion.documents WHERE status != 'DELETED' GROUP BY status`
	docRows, err := r.db.QueryContext(ctx, docQuery)
	if err != nil {
		return nil, err
	}
	defer docRows.Close()

	for docRows.Next() {
		var dist domain.DocStatusDist
		if err := docRows.Scan(&dist.Status, &dist.Count); err != nil {
			return nil, err
		}
		overview.DocStatusDist = append(overview.DocStatusDist, dist)
	}

	// 5. User Role Distribution
	roleQuery := `
		SELECT r.name, count(u.id) 
		FROM gateway.roles r 
		LEFT JOIN gateway.users u ON r.id = u.role_id 
		GROUP BY r.name
	`
	roleRows, err := r.db.QueryContext(ctx, roleQuery)
	if err != nil {
		return nil, err
	}
	defer roleRows.Close()

	for roleRows.Next() {
		var dist domain.UserRoleDist
		if err := roleRows.Scan(&dist.Role, &dist.Count); err != nil {
			return nil, err
		}
		overview.UserRoleDist = append(overview.UserRoleDist, dist)
	}

	return overview, nil
}

func (r *metricsRepository) GetRiskHeatmap(ctx context.Context) ([]domain.RiskHeatmap, error) {
	query := `
		SELECT 
			COALESCE(business_unit, 'Unspecified') as business_unit,
			COALESCE(document_type, 'OTHER') as document_type,
			COALESCE(risk_level, 'UNKNOWN') as risk_level,
			COUNT(*) as count
		FROM ingestion.documents 
		WHERE status != 'DELETED' AND is_deprecated = FALSE
		GROUP BY business_unit, document_type, risk_level
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var heatmap []domain.RiskHeatmap
	for rows.Next() {
		var h domain.RiskHeatmap
		if err := rows.Scan(&h.BusinessUnit, &h.DocumentType, &h.RiskLevel, &h.Count); err != nil {
			return nil, err
		}
		heatmap = append(heatmap, h)
	}
	return heatmap, nil
}

func (r *metricsRepository) GetExpiringContracts(ctx context.Context) ([]domain.ExpiringContract, error) {
	query := `
		SELECT 
			id, filename, COALESCE(document_type, ''), COALESCE(risk_level, ''), COALESCE(vendor_name, ''), expiry_date 
		FROM ingestion.documents 
		WHERE expiry_date IS NOT NULL 
		AND expiry_date <= CURRENT_DATE + INTERVAL '30 days'
		AND status != 'DELETED' AND is_deprecated = FALSE
		ORDER BY expiry_date ASC
		LIMIT 20
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []domain.ExpiringContract
	for rows.Next() {
		var c domain.ExpiringContract
		var expiryDate time.Time
		if err := rows.Scan(&c.ID, &c.Filename, &c.DocumentType, &c.RiskLevel, &c.VendorName, &expiryDate); err != nil {
			return nil, err
		}
		c.ExpiryDate = expiryDate.Format(time.RFC3339)
		contracts = append(contracts, c)
	}
	return contracts, nil
}

func (r *metricsRepository) GetRegulatoryImpacts(ctx context.Context) ([]domain.DashboardRegulatoryImpact, error) {
	query := `
		SELECT 
			ri.id, ri.impact_level, 
			COALESCE(d1.filename, 'Unknown Regulation') as regulation_name, 
			COALESCE(d2.filename, 'Unknown Document') as internal_document_name,
			ri.created_at
		FROM legal_engine.regulatory_impacts ri
		LEFT JOIN ingestion.documents d1 ON ri.regulation_id = d1.id
		LEFT JOIN ingestion.documents d2 ON ri.internal_document_id = d2.id
		ORDER BY ri.created_at DESC
		LIMIT 20
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var impacts []domain.DashboardRegulatoryImpact
	for rows.Next() {
		var i domain.DashboardRegulatoryImpact
		var createdAt time.Time
		if err := rows.Scan(&i.ID, &i.ImpactLevel, &i.RegulationName, &i.InternalDocumentName, &createdAt); err != nil {
			return nil, err
		}
		i.CreatedAt = createdAt.Format(time.RFC3339)
		impacts = append(impacts, i)
	}
	return impacts, nil
}
