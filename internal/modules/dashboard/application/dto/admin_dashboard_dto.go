package dto

import (
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
)

type GetAIMarginAnalyticsQueryReq struct {
	FromDate string `form:"fromDate" json:"fromDate"`
	ToDate   string `form:"toDate" json:"toDate"`
}

type AdminKPICardsRes struct {
	TotalUsers             int       `json:"totalUsers"`
	ActiveUsersDAU         int       `json:"activeUsersDau"`
	TotalBrands            int       `json:"totalBrands"`
	PendingReviewBrands    int       `json:"pendingReviewBrands"`
	SubscriptionRevenueVND float64   `json:"subscriptionRevenueVnd"`
	NetAIMarginVND         float64   `json:"netAiMarginVnd"`
	PayOSSuccessCount      int       `json:"payOSSuccessCount"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type AIMarginAnalyticsRes struct {
	FromDate                time.Time `json:"fromDate"`
	ToDate                  time.Time `json:"toDate"`
	TotalSubscriptionRevVND float64   `json:"totalSubscriptionRevVnd"`
	TotalAIActualCostVND    float64   `json:"totalAiActualCostVnd"`
	NetAIMarginVND          float64   `json:"netAiMarginVnd"`
	MarginPercentage        float64   `json:"marginPercentage"`
	TotalAIPaidTokens       int64     `json:"totalAiPaidTokens"`
	AIRequestsCount         int64     `json:"aiRequestsCount"`
	AIErrorCount            int64     `json:"aiErrorCount"`
}

type PayOSReconciliationRes struct {
	TotalTransactions int     `json:"totalTransactions"`
	SuccessCount      int     `json:"successCount"`
	FailedCount       int     `json:"failedCount"`
	TotalSuccessValue float64 `json:"totalSuccessValue"`
}

type AdminOverviewRes struct {
	KPICards       *AdminKPICardsRes       `json:"kpiCards"`
	AIMargin       *AIMarginAnalyticsRes   `json:"aiMargin"`
	PayOSReconcile *PayOSReconciliationRes `json:"payosReconcile"`
}

// SubscriptionDistribution

// SubscriptionDistributionDTO is a presentation projection of
// subscription_dto.SubscriptionDistributionDTO (int user count + percentage
// computed downstream) — keep field names in sync with the source read model.
type SubscriptionDistributionDTO struct {
	PlanCode   string  `json:"planCode"`
	UserCount  int     `json:"userCount"`
	Percentage float64 `json:"percentage"`
}

type SubscriptionDistributionRes struct {
	Distribution []SubscriptionDistributionDTO `json:"distribution"`
	TotalUsers   int                           `json:"totalUsers"`
}

// RevenueBreakdownDTO is a presentation projection of
// subscription_dto.RevenueBreakdownDTO (month formatted as string, revenue as
// VND float) — keep field names in sync with the source read model.
type RevenueBreakdownDTO struct {
	PlanCode         string  `json:"planCode"`
	Month            string  `json:"month"`
	TransactionCount int     `json:"transactionCount"`
	RevenueVND       float64 `json:"revenueVnd"`
}

type RevenueBreakdownRes struct {
	FromDate        string                `json:"fromDate"`
	ToDate          string                `json:"toDate"`
	Breakdown       []RevenueBreakdownDTO `json:"breakdown"`
	TotalRevenueVND float64               `json:"totalRevenueVnd"`
}

// AI Usage by Operation

// AIUsageByOperationDTO is a presentation projection of
// aiexecutor_dto.AIUsageByOperationDTO (micro-VND cost converted to VND float)
// — keep field names in sync with the source read model.
type AIUsageByOperationDTO struct {
	Operation         string  `json:"operation"`
	LogicalRoute      string  `json:"logicalRoute"`
	RequestCount      int64   `json:"requestCount"`
	TotalCostVND      float64 `json:"totalCostVnd"`
	TotalPromptTokens int64   `json:"totalPromptTokens"`
	TotalOutputTokens int64   `json:"totalOutputTokens"`
}

type AIUsageByOperationRes struct {
	FromDate   string                  `json:"fromDate"`
	ToDate     string                  `json:"toDate"`
	Operations []AIUsageByOperationDTO `json:"operations"`
}

// AI Error Breakdown

type AIErrorBreakdownRes struct {
	TotalErrors   int64                                `json:"totalErrors"`
	TotalRequests int64                                `json:"totalRequests"`
	ErrorRate     float64                              `json:"errorRate"`
	Breakdown     []aiexecutor_dto.AIErrorBreakdownDTO `json:"breakdown"`
}

// Renewal Stats

type RenewalStatsRes struct {
	TotalAttempts int     `json:"totalAttempts"`
	SuccessCount  int     `json:"successCount"`
	FailedCount   int     `json:"failedCount"`
	SuccessRate   float64 `json:"successRate"`
	AvgRetryCount float64 `json:"avgRetryCount"`
}
