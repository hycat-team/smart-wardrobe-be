package repositories

import (
	"time"

	"github.com/shopspring/decimal"
)

// Admin stats read models owned by the domain layer (feature 017);
// application/dto aliases it for handler compatibility.
type AdminSubscriptionStatsDTO struct {
	// TotalSubscriptionRevenue keeps the subscriptions-only scope: SUM(amount)
	// over status = success AND transaction_type = direct_purchase (029 R6).
	TotalSubscriptionRevenue decimal.Decimal
	PayOSTransactionCount    int
	PayOSSuccessCount        int
	// Status breakdown of PayOSTransactionCount. The three buckets plus the
	// success count cover every deposit status exactly once, so a report can
	// stop deriving "failed" as total − success. All five counters span BOTH
	// transaction types (direct purchase + wallet top-up) — 029 R6.
	PayOSPendingCount int
	PayOSFailedCount  int
	PayOSVoidedCount  int
	// TotalPayOSSuccessValue is SUM(amount) over successful payOS transactions
	// of both types: the money that actually moved, as opposed to
	// TotalSubscriptionRevenue, which is the subscription slice of it.
	TotalPayOSSuccessValue decimal.Decimal
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
