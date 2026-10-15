package transport_test

import (
	"testing"
	"time"

	subscriptiontransport "smart-wardrobe-be/services/account/contract/subscription/transport"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestOverviewRoundTrip(t *testing.T) {
	expires := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	original := &subscriptiondto.UserSubscriptionContractDTO{
		PlanID:             uuid.New(),
		PlanName:           "Premium",
		PlanSlug:           "premium",
		ExpiresAt:          &expires,
		IsAutoRenewEnabled: true,
		MaxWardrobeItems:   500,
		MaxOutfits:         200,
		AiOutfitDailyQuota: 20,
		AiChatDailyQuota:   50,
	}

	wire := subscriptiontransport.OverviewDTOToProto(original)
	back, err := subscriptiontransport.OverviewToDTO(wire)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}

	if back.PlanID != original.PlanID || back.PlanSlug != original.PlanSlug {
		t.Fatalf("plan identity mismatch: %+v", back)
	}
	if back.ExpiresAt == nil || !back.ExpiresAt.Equal(expires) {
		t.Fatalf("expires mismatch: %+v", back.ExpiresAt)
	}
	if back.MaxWardrobeItems != original.MaxWardrobeItems || back.AiChatDailyQuota != original.AiChatDailyQuota {
		t.Fatalf("quota mismatch: %+v", back)
	}
}

func TestOverviewMapRoundTrip(t *testing.T) {
	uid := uuid.New()
	original := map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO{
		uid: {PlanID: uuid.New(), PlanName: "Free", PlanSlug: "free"},
	}

	wire := subscriptiontransport.OverviewDTOsToProtoMap(original)
	back, err := subscriptiontransport.OverviewMapToDTOs(wire)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}
	got, ok := back[uid]
	if !ok || got.PlanSlug != "free" {
		t.Fatalf("map round-trip mismatch: %+v", back)
	}
}

func TestAdminStatsDecimalRoundTrip(t *testing.T) {
	original := &subscriptiondto.AdminSubscriptionStatsDTO{
		TotalSubscriptionRevenue: decimal.NewFromFloat(1234567.89),
		PayOSTransactionCount:    100,
		PayOSSuccessCount:        97,
		PayOSPendingCount:        1,
		PayOSFailedCount:         1,
		PayOSVoidedCount:         1,
		// 029 T040 (field 7): SUM(amount) of successful payOS transactions over
		// BOTH types; differs from the subscription-only revenue on purpose.
		TotalPayOSSuccessValue: decimal.NewFromFloat(2345678.91),
	}

	wire := subscriptiontransport.AdminStatsToProto(original)
	back, err := subscriptiontransport.AdminStatsToDTO(wire)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}
	if !back.TotalSubscriptionRevenue.Equal(original.TotalSubscriptionRevenue) {
		t.Fatalf("decimal precision lost: %s vs %s", back.TotalSubscriptionRevenue, original.TotalSubscriptionRevenue)
	}
	if !back.TotalPayOSSuccessValue.Equal(original.TotalPayOSSuccessValue) {
		t.Fatalf("field 7 decimal precision lost: %s vs %s", back.TotalPayOSSuccessValue, original.TotalPayOSSuccessValue)
	}
	if back.PayOSPendingCount != 1 || back.PayOSFailedCount != 1 || back.PayOSVoidedCount != 1 {
		t.Fatalf("status breakdown lost across the wire: %+v", back)
	}
}

func TestRevenueBreakdownDecimalRoundTrip(t *testing.T) {
	original := []subscriptiondto.RevenueBreakdownDTO{{
		PlanCode:         "premium",
		Month:            time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		TransactionCount: 42,
		Revenue:          decimal.NewFromFloat(999.99),
	}}

	wire := subscriptiontransport.RevenueBreakdownToProto(original)
	back, err := subscriptiontransport.RevenueBreakdownToDTOs(wire)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}
	if len(back) != 1 || !back[0].Revenue.Equal(original[0].Revenue) {
		t.Fatalf("revenue round-trip mismatch: %+v", back)
	}
	if !back[0].Month.Equal(original[0].Month) {
		t.Fatalf("month mismatch: %+v", back[0].Month)
	}
}
