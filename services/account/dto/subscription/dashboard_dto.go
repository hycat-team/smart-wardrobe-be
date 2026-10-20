package dto

import (
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
)

type AdminSubscriptionStatsDTO = repositories.AdminSubscriptionStatsDTO

type SubscriptionDistributionDTO = repositories.SubscriptionDistributionDTO

// SubscriptionDistributionResultDTO is the distribution result with its
// history-completeness metadata (031 contract delta §4). DataAvailableFrom is
// the cutover instant; the Commerce edge formats it to the VN calendar date
// (YYYY-MM-DD). DataCompleteness is complete | partial | unavailable.
type SubscriptionDistributionResultDTO = repositories.SubscriptionDistributionResult

type RevenueBreakdownDTO = repositories.RevenueBreakdownDTO

type RenewalStatsDTO = repositories.RenewalStatsDTO
