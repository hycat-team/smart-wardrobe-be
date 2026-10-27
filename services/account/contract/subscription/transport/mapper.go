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
		// 033 D2/delta §5: an old Account server leaves field 11 unset, so this
		// reads false and personalization stays off instead of erroring.
		PersonalizationEnabled: pb.GetPersonalizationEnabled(),
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
// Both decimals cross the wire as strings; field 7
// (total_payos_success_value) sums successful payOS amounts over both
// transaction types (029 R6).
func AdminStatsToDTO(pb *subscriptionv1.AdminSubscriptionStats) (*subscriptiondto.AdminSubscriptionStatsDTO, error) {
	if pb == nil {
		return nil, nil
	}
	revenue, err := parseDecimal(pb.GetTotalSubscriptionRevenue())
	if err != nil {
		return nil, err
	}
	payosSuccessValue, err := parseDecimal(pb.GetTotalPayosSuccessValue())
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
		TotalPayOSSuccessValue:   payosSuccessValue,
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

// DistributionListToDTO converts the wire list into the interval-based result
// (031 contract delta §3).
//
// An old Account server leaves the two new fields unset: the absent timestamp
// stays nil and UNSPECIFIED becomes an empty completeness string, so the
// Commerce DTO's `omitempty` drops both keys and the old JSON shape survives.
func DistributionListToDTO(pb *subscriptionv1.SubscriptionDistributionList) *subscriptiondto.SubscriptionDistributionResultDTO {
	if pb == nil {
		return nil
	}
	out := &subscriptiondto.SubscriptionDistributionResultDTO{
		Items:            DistributionToDTOs(pb.GetItems()),
		DataCompleteness: distributionCompletenessFromProto(pb.GetDataCompleteness()),
	}
	if ts := pb.GetDataAvailableFrom(); ts != nil {
		availableFrom := ts.AsTime()
		out.DataAvailableFrom = &availableFrom
	}
	return out
}

// distributionCompletenessFromProto maps the enum back to the lowercase result
// string. UNSPECIFIED and unknown values become "", which means "omit the
// field" — never a made-up "complete".
func distributionCompletenessFromProto(value subscriptionv1.DistributionDataCompleteness) string {
	switch value {
	case subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_COMPLETE:
		return "complete"
	case subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_PARTIAL:
		return "partial"
	case subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_UNAVAILABLE:
		return "unavailable"
	default:
		return ""
	}
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
