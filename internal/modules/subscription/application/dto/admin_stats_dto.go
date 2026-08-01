package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

type AdminSubscriptionStatsDTO struct {
	TotalSubscriptionRevenue decimal.Decimal
	PayOSTransactionCount    int
	PayOSSuccessCount        int
}

type SubscriptionDistributionDTO struct {
	PlanCode  string `json:"planCode"`
	UserCount int64  `json:"userCount"`
}

type RevenueBreakdownDTO struct {
	PlanCode         string          `json:"planCode"`
	Month            time.Time       `json:"month"`
	TransactionCount int64           `json:"transactionCount"`
	Revenue          decimal.Decimal `json:"revenue"`
}

type RenewalStatsDTO struct {
	TotalAttempts int64   `json:"totalAttempts"`
	SuccessCount  int64   `json:"successCount"`
	FailedCount   int64   `json:"failedCount"`
	AvgRetryCount float64 `json:"avgRetryCount"`
}
