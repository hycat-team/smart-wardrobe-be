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
		// 033 D2/delta §5: field 11 is additive; false is the safe default.
		PersonalizationEnabled: d.PersonalizationEnabled,
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

// DistributionResultToProto converts the interval-based result into the wire
// list (031 contract delta §3).
//
// The two new fields are additive (numbers 2 and 3): an old client keeps
// reading `items`. An empty DataCompleteness (UNSPECIFIED) and a nil cutover
// mean "an old Account server answered"; both fields stay unset so the
// Commerce edge omits them from JSON instead of inventing a status.
func DistributionResultToProto(d *subscriptiondto.SubscriptionDistributionResultDTO) *subscriptionv1.SubscriptionDistributionList {
	if d == nil {
		return nil
	}
	out := &subscriptionv1.SubscriptionDistributionList{
		Items:            DistributionToProto(d.Items),
		DataCompleteness: distributionCompletenessToProto(d.DataCompleteness),
	}
	if d.DataAvailableFrom != nil {
		out.DataAvailableFrom = timestamppb.New(*d.DataAvailableFrom)
	}
	return out
}

// distributionCompletenessToProto maps the lowercase result string onto the
// enum. An empty or unknown value maps to UNSPECIFIED, which the client reads
// back as "omit the field" (old-server compatibility).
func distributionCompletenessToProto(value string) subscriptionv1.DistributionDataCompleteness {
	switch value {
	case "complete":
		return subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_COMPLETE
	case "partial":
		return subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_PARTIAL
	case "unavailable":
		return subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_UNAVAILABLE
	default:
		return subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_UNSPECIFIED
	}
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
