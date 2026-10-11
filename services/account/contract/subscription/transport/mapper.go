// Package transport maps subscription wire types (proto) to the owner's
// contract DTOs, so both the gRPC server and remote clients reuse one mapping
// while domain types stay encapsulated (design-pattern.md "Cross-service
// transport mapping (A')").
package transport

import (
	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OverviewToDTO converts one wire subscription overview into the contract DTO.
func OverviewToDTO(pb *subscriptionv1.UserSubscriptionOverview) (*subscriptiondto.UserSubscriptionContractDTO, error) {
	if pb == nil {
		return nil, nil
	}
	planID, err := parseOptionalUUID(pb.GetPlanId())
	if err != nil {
		return nil, err
	}

	out := &subscriptiondto.UserSubscriptionContractDTO{
		PlanID:             planID,
		PlanName:           pb.GetPlanName(),
		PlanSlug:           pb.GetPlanSlug(),
		IsAutoRenewEnabled: pb.GetIsAutoRenewEnabled(),
		MaxWardrobeItems:   int(pb.GetMaxWardrobeItems()),
		MaxOutfits:         int(pb.GetMaxOutfits()),
		AiOutfitDailyQuota: int(pb.GetAiOutfitDailyQuota()),
		AiChatDailyQuota:   int(pb.GetAiChatDailyQuota()),
	}
	if pb.GetHasExpiresAt() && pb.GetExpiresAt() != nil {
		expiresAt := pb.GetExpiresAt().AsTime()
		out.ExpiresAt = &expiresAt
	}
	return out, nil
}

// OverviewMapToDTOs converts the wire overview map into contract DTOs keyed by user id.
func OverviewMapToDTOs(pb map[string]*subscriptionv1.UserSubscriptionOverview) (map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO, error) {
	out := make(map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO, len(pb))
	for key, item := range pb {
		userID, err := uuid.Parse(key)
		if err != nil {
			continue
		}
		d, err := OverviewToDTO(item)
		if err != nil {
			return nil, err
		}
		out[userID] = d
	}
	return out, nil
}

// UserSubscriptionToDTO converts the wire quota read model into the contract DTO.
func UserSubscriptionToDTO(pb *subscriptionv1.UserSubscription) (*subscriptiondto.UserSubscriptionDTO, error) {
	if pb == nil {
		return nil, nil
	}
	planID, err := parseOptionalUUID(pb.GetPlanId())
	if err != nil {
		return nil, err
	}

	out := &subscriptiondto.UserSubscriptionDTO{
		PlanID:               planID,
		PlanName:             pb.GetPlanName(),
		PlanSlug:             pb.GetPlanSlug(),
		IsAutoRenewEnabled:   pb.GetIsAutoRenewEnabled(),
		AiOutfitDailyQuota:   int(pb.GetAiOutfitDailyQuota()),
		AiChatDailyQuota:     int(pb.GetAiChatDailyQuota()),
		OutfitRecommendCount: int(pb.GetOutfitRecommendCount()),
		AiUsageCount:         int(pb.GetAiUsageCount()),
	}
	if pb.GetHasExpiresAt() && pb.GetExpiresAt() != nil {
		expiresAt := pb.GetExpiresAt().AsTime()
		out.ExpiresAt = &expiresAt
	}
	if ts := pb.GetLastResetDate(); ts != nil {
		out.LastResetDate = ts.AsTime()
	}
	return out, nil
}

// AdminStatsToDTO converts wire subscription statistics into the contract DTO.
func AdminStatsToDTO(pb *subscriptionv1.AdminSubscriptionStats) (*subscriptiondto.AdminSubscriptionStatsDTO, error) {
	if pb == nil {
		return nil, nil
	}
	revenue, err := parseDecimal(pb.GetTotalSubscriptionRevenue())
	if err != nil {
		return nil, err
	}
	return &subscriptiondto.AdminSubscriptionStatsDTO{
		TotalSubscriptionRevenue: revenue,
		PayOSTransactionCount:    int(pb.GetPayosTransactionCount()),
		PayOSSuccessCount:        int(pb.GetPayosSuccessCount()),
		PayOSPendingCount:        int(pb.GetPayosPendingCount()),
		PayOSFailedCount:         int(pb.GetPayosFailedCount()),
		PayOSVoidedCount:         int(pb.GetPayosVoidedCount()),
	}, nil
}

// DistributionToDTOs converts the wire plan distribution list into contract DTOs.
func DistributionToDTOs(pb []*subscriptionv1.SubscriptionDistribution) []subscriptiondto.SubscriptionDistributionDTO {
	out := make([]subscriptiondto.SubscriptionDistributionDTO, 0, len(pb))
	for _, item := range pb {
		out = append(out, subscriptiondto.SubscriptionDistributionDTO{
			PlanCode:  item.GetPlanCode(),
			UserCount: item.GetUserCount(),
		})
	}
	return out
}

// RevenueBreakdownToDTOs converts the wire revenue list into contract DTOs.
func RevenueBreakdownToDTOs(pb []*subscriptionv1.RevenueBreakdown) ([]subscriptiondto.RevenueBreakdownDTO, error) {
	out := make([]subscriptiondto.RevenueBreakdownDTO, 0, len(pb))
	for _, item := range pb {
		revenue, err := parseDecimal(item.GetRevenue())
		if err != nil {
			return nil, err
		}
		entry := subscriptiondto.RevenueBreakdownDTO{
			PlanCode:         item.GetPlanCode(),
			TransactionCount: item.GetTransactionCount(),
			Revenue:          revenue,
		}
		if ts := item.GetMonth(); ts != nil {
			entry.Month = ts.AsTime()
		}
		out = append(out, entry)
	}
	return out, nil
}

// RenewalStatsToDTO converts wire renewal statistics into the contract DTO.
func RenewalStatsToDTO(pb *subscriptionv1.RenewalStats) *subscriptiondto.RenewalStatsDTO {
	if pb == nil {
		return nil
	}
	return &subscriptiondto.RenewalStatsDTO{
		TotalAttempts: pb.GetTotalAttempts(),
		SuccessCount:  pb.GetSuccessCount(),
		FailedCount:   pb.GetFailedCount(),
		AvgRetryCount: pb.GetAvgRetryCount(),
	}
}

func parseOptionalUUID(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		wrapErr := apperror.NewInternalError(apperror.GenericInternalMessage)
		wrapErr.Cause = err
		return uuid.Nil, wrapErr
	}
	return id, nil
}

func parseDecimal(raw string) (decimal.Decimal, error) {
	if raw == "" {
		return decimal.Zero, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil {
		wrapErr := apperror.NewInternalError(apperror.GenericInternalMessage)
		wrapErr.Cause = err
		return decimal.Zero, wrapErr
	}
	return value, nil
}
