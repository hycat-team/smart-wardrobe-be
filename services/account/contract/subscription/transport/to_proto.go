package transport

import (
	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OverviewDTOToProto converts the contract DTO into its wire form.
func OverviewDTOToProto(d *subscriptiondto.UserSubscriptionContractDTO) *subscriptionv1.UserSubscriptionOverview {
	if d == nil {
		return nil
	}
	out := &subscriptionv1.UserSubscriptionOverview{
		PlanId:             d.PlanID.String(),
		PlanName:           d.PlanName,
		PlanSlug:           d.PlanSlug,
		IsAutoRenewEnabled: d.IsAutoRenewEnabled,
		MaxWardrobeItems:   int32(d.MaxWardrobeItems),
		MaxOutfits:         int32(d.MaxOutfits),
		AiOutfitDailyQuota: int32(d.AiOutfitDailyQuota),
		AiChatDailyQuota:   int32(d.AiChatDailyQuota),
	}
	if d.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*d.ExpiresAt)
		out.HasExpiresAt = true
	}
	return out
}

// OverviewDTOsToProtoMap converts contract DTOs keyed by user id into the wire map.
func OverviewDTOsToProtoMap(ds map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO) map[string]*subscriptionv1.UserSubscriptionOverview {
	out := make(map[string]*subscriptionv1.UserSubscriptionOverview, len(ds))
	for id, d := range ds {
		out[id.String()] = OverviewDTOToProto(d)
	}
	return out
}

// UserSubscriptionDTOToProto converts the quota read model into its wire form.
func UserSubscriptionDTOToProto(d *subscriptiondto.UserSubscriptionDTO) *subscriptionv1.UserSubscription {
	if d == nil {
		return nil
	}
	out := &subscriptionv1.UserSubscription{
		PlanId:               d.PlanID.String(),
		PlanName:             d.PlanName,
		PlanSlug:             d.PlanSlug,
		IsAutoRenewEnabled:   d.IsAutoRenewEnabled,
		AiOutfitDailyQuota:   int32(d.AiOutfitDailyQuota),
		AiChatDailyQuota:     int32(d.AiChatDailyQuota),
		OutfitRecommendCount: int32(d.OutfitRecommendCount),
		AiUsageCount:         int32(d.AiUsageCount),
		LastResetDate:        timestamppb.New(d.LastResetDate),
	}
	if d.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*d.ExpiresAt)
		out.HasExpiresAt = true
	}
	return out
}

// AdminStatsToProto converts the contract DTO into its wire form. Decimals
// cross the wire as strings to preserve precision; field 7 carries the sum of
// successful payOS amounts over both transaction types (029 R6).
func AdminStatsToProto(d *subscriptiondto.AdminSubscriptionStatsDTO) *subscriptionv1.AdminSubscriptionStats {
	if d == nil {
		return nil
	}
	return &subscriptionv1.AdminSubscriptionStats{
		TotalSubscriptionRevenue: d.TotalSubscriptionRevenue.String(),
		PayosTransactionCount:    int32(d.PayOSTransactionCount),
		PayosSuccessCount:        int32(d.PayOSSuccessCount),
		PayosPendingCount:        int32(d.PayOSPendingCount),
		PayosFailedCount:         int32(d.PayOSFailedCount),
		PayosVoidedCount:         int32(d.PayOSVoidedCount),
		TotalPayosSuccessValue:   d.TotalPayOSSuccessValue.String(),
	}
}

// DistributionToProto converts plan distribution rows into wire messages.
func DistributionToProto(ds []subscriptiondto.SubscriptionDistributionDTO) []*subscriptionv1.SubscriptionDistribution {
	out := make([]*subscriptionv1.SubscriptionDistribution, 0, len(ds))
	for _, d := range ds {
		out = append(out, &subscriptionv1.SubscriptionDistribution{
			PlanCode:  d.PlanCode,
			UserCount: d.UserCount,
		})
	}
	return out
}

// RevenueBreakdownToProto converts revenue rows into wire messages.
func RevenueBreakdownToProto(ds []subscriptiondto.RevenueBreakdownDTO) []*subscriptionv1.RevenueBreakdown {
	out := make([]*subscriptionv1.RevenueBreakdown, 0, len(ds))
	for _, d := range ds {
		out = append(out, &subscriptionv1.RevenueBreakdown{
			PlanCode:         d.PlanCode,
			Month:            timestamppb.New(d.Month),
			TransactionCount: d.TransactionCount,
			Revenue:          d.Revenue.String(),
		})
	}
	return out
}

// RenewalStatsToProto converts renewal statistics into the wire form.
func RenewalStatsToProto(d *subscriptiondto.RenewalStatsDTO) *subscriptionv1.RenewalStats {
	if d == nil {
		return nil
	}
	return &subscriptionv1.RenewalStats{
		TotalAttempts: d.TotalAttempts,
		SuccessCount:  d.SuccessCount,
		FailedCount:   d.FailedCount,
		AvgRetryCount: d.AvgRetryCount,
	}
}
