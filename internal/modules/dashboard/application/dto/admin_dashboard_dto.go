package dto

import "time"

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
