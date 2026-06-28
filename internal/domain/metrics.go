package domain

import (
	"context"
	"time"
)

type TokenMetric struct {
	ID               int       `json:"id"`
	UserID           string    `json:"user_id"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	ModelName        string    `json:"model_name"`
	Timestamp        time.Time `json:"timestamp"`
}

type UserCost struct {
	UserID   string  `json:"user_id"`
	Email    string  `json:"email"`     // Joined from users table
	Name     string  `json:"name"`      // Joined from users table
	TotalCost float64 `json:"total_cost"`
}

type DailyUsage struct {
	Date        string  `json:"date"`
	TotalTokens int     `json:"total_tokens"`
	CostUSD     float64 `json:"cost_usd"`
}

type DocStatusDist struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type UserRoleDist struct {
	Role  string `json:"role"`
	Count int    `json:"count"`
}

type SystemOverview struct {
	TotalUsers     int64
	TotalDocuments int64
	TotalFolders   int64
	DocStatusDist  []DocStatusDist
	UserRoleDist   []UserRoleDist
}

type RiskHeatmap struct {
	BusinessUnit string `json:"business_unit"`
	DocumentType string `json:"document_type"`
	RiskLevel    string `json:"risk_level"`
	Count        int    `json:"count"`
}

type ExpiringContract struct {
	ID          string  `json:"id"`
	Filename    string  `json:"filename"`
	DocumentType string  `json:"document_type"`
	RiskLevel   string  `json:"risk_level"`
	VendorName  string  `json:"vendor_name"`
	ExpiryDate  string  `json:"expiry_date"`
}

type RecentActivity struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Tokens    int       `json:"tokens"`
	CostUSD   float64   `json:"cost"`
	Timestamp time.Time `json:"timestamp"`
}

type MyMetrics struct {
	TotalCostThisMonth float64          `json:"total_cost_this_month"`
	DailyTrend         []DailyUsage     `json:"daily_trend"`
	RecentActivities   []RecentActivity `json:"recent_activities"`
}

type DashboardRegulatoryImpact struct {
	ID                   string `json:"id"`
	ImpactLevel          string `json:"impact_level"`
	RegulationName       string `json:"regulation_name"`
	InternalDocumentName string `json:"internal_document_name"`
	CreatedAt            string `json:"created_at"`
}

type DashboardMetrics struct {
	TotalCostThisMonth float64         `json:"total_cost_this_month"`
	TopUsers           []UserCost      `json:"top_users"`
	DailyTrend         []DailyUsage    `json:"daily_trend"`
	TotalUsers         int64           `json:"total_users"`
	TotalDocuments     int64           `json:"total_documents"`
	TotalFolders       int64           `json:"total_folders"`
	DocStatusDist      []DocStatusDist `json:"doc_status_dist"`
	UserRoleDist       []UserRoleDist  `json:"user_role_dist"`
}

type MetricsRepository interface {
	GetTotalCostThisMonth(ctx context.Context) (float64, error)
	GetTopUsersByCost(ctx context.Context, limit int) ([]UserCost, error)
	GetDailyUsageTrend(ctx context.Context, days int) ([]DailyUsage, error)
	GetMyTotalCostThisMonth(ctx context.Context, userID string) (float64, error)
	GetMyDailyUsageTrend(ctx context.Context, userID string, days int) ([]DailyUsage, error)
	GetMyRecentActivity(ctx context.Context, userID string, limit int) ([]RecentActivity, error)
	GetSystemOverview(ctx context.Context) (*SystemOverview, error)
	GetRiskHeatmap(ctx context.Context) ([]RiskHeatmap, error)
	GetExpiringContracts(ctx context.Context) ([]ExpiringContract, error)
}

type MetricsUsecase interface {
	GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error)
	GetMyMetrics(ctx context.Context, userID string) (*MyMetrics, error)
	GetRiskHeatmap(ctx context.Context) ([]RiskHeatmap, error)
	GetExpiringContracts(ctx context.Context) ([]ExpiringContract, error)
}
