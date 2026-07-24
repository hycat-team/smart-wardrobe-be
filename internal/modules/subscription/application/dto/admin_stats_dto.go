package dto

import "github.com/shopspring/decimal"

type AdminSubscriptionStatsDTO struct {
	TotalSubscriptionRevenue decimal.Decimal
	PayOSTransactionCount    int
	PayOSSuccessCount        int
}
