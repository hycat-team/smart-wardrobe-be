package transport_test

import (
	"testing"
	"time"

	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	subscriptiontransport "smart-wardrobe-be/services/account/contract/subscription/transport"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
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
		// 033 T008: field 11 must survive the round-trip when enabled.
		PersonalizationEnabled: true,
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
	if !back.PersonalizationEnabled {
		t.Fatalf("personalizationEnabled lost across the wire: %+v", back)
	}
}

// 033 T008: the additive field must be number 11 so an older Account server
// (which does not know it) and a newer client agree on the wire while the
// unknown-field rule keeps old payloads decodable.
func TestOverviewPersonalizationFieldNumber(t *testing.T) {
	field := (&subscriptionv1.UserSubscriptionOverview{}).ProtoReflect().
		Descriptor().Fields().ByName("personalization_enabled")
	if field == nil {
		t.Fatal("personalization_enabled is missing from the proto descriptor")
	}
	if got := field.Number(); got != 11 {
		t.Fatalf("personalization_enabled field number = %d, want 11", got)
	}
}

// 033 T008: an old Account server sends no field 11 at all. The client must
// read false rather than fail, which safely leaves personalization off.
func TestOverviewOldPayloadDefaultsPersonalizationOff(t *testing.T) {
	oldWire := &subscriptionv1.UserSubscriptionOverview{
		PlanId:             uuid.New().String(),
		PlanName:           "Free",
		PlanSlug:           "free",
		MaxWardrobeItems:   50,
		MaxOutfits:         20,
		AiOutfitDailyQuota: 3,
		AiChatDailyQuota:   1,
	}
	raw, err := proto.Marshal(oldWire)
	if err != nil {
		t.Fatalf("marshal the old payload: %v", err)
	}

	var decoded subscriptionv1.UserSubscriptionOverview
	if err := proto.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode the old payload: %v", err)
	}
	back, err := subscriptiontransport.OverviewToDTO(&decoded)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}
	if back.PersonalizationEnabled {
		t.Fatalf("an absent field 11 must decode as false, got %+v", back)
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

// 031 T016: the two additive fields must survive both directions, while an old
// server that leaves them unset must keep the old shape (no invented status).
func TestDistributionResultRoundTrip(t *testing.T) {
	cutover := time.Date(2026, 10, 20, 5, 30, 0, 0, time.UTC)
	cases := []struct {
		name         string
		completeness string
		wantEnum     subscriptionv1.DistributionDataCompleteness
	}{
		{"complete", "complete", subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_COMPLETE},
		{"partial", "partial", subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_PARTIAL},
		{"unavailable", "unavailable", subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_UNAVAILABLE},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := &subscriptiondto.SubscriptionDistributionResultDTO{
				Items: []subscriptiondto.SubscriptionDistributionDTO{
					{PlanCode: "free", UserCount: 3},
					{PlanCode: "premium-monthly", UserCount: 2},
				},
				DataAvailableFrom: &cutover,
				DataCompleteness:  tc.completeness,
			}

			wire := subscriptiontransport.DistributionResultToProto(original)
			if wire.GetDataCompleteness() != tc.wantEnum {
				t.Fatalf("wire completeness = %v, want %v", wire.GetDataCompleteness(), tc.wantEnum)
			}
			if wire.GetDataAvailableFrom() == nil || !wire.GetDataAvailableFrom().AsTime().Equal(cutover) {
				t.Fatalf("wire cutover = %v, want %s", wire.GetDataAvailableFrom(), cutover)
			}

			back := subscriptiontransport.DistributionListToDTO(wire)
			if back.DataCompleteness != tc.completeness {
				t.Fatalf("completeness round-trip = %q, want %q", back.DataCompleteness, tc.completeness)
			}
			if back.DataAvailableFrom == nil || !back.DataAvailableFrom.Equal(cutover) {
				t.Fatalf("cutover round-trip = %v, want %s", back.DataAvailableFrom, cutover)
			}
			if len(back.Items) != 2 || back.Items[1].PlanCode != "premium-monthly" || back.Items[1].UserCount != 2 {
				t.Fatalf("items round-trip mismatch: %+v", back.Items)
			}
		})
	}
}

func TestDistributionListWithoutMetadataOmitsBothFields(t *testing.T) {
	// This is what an old Account server sends: items only, so fields 2 and 3
	// are unset. The Commerce edge must omit both JSON keys rather than claim
	// the period is complete or invent a cutover date.
	wire := &subscriptionv1.SubscriptionDistributionList{
		Items: []*subscriptionv1.SubscriptionDistribution{
			{PlanCode: "free", UserCount: 5},
		},
	}

	back := subscriptiontransport.DistributionListToDTO(wire)
	if back == nil {
		t.Fatal("expected a result for a non-nil wire list")
	}
	if back.DataAvailableFrom != nil {
		t.Fatalf("dataAvailableFrom = %v, want nil (absent on the wire)", back.DataAvailableFrom)
	}
	if back.DataCompleteness != "" {
		t.Fatalf("dataCompleteness = %q, want the empty string for UNSPECIFIED", back.DataCompleteness)
	}
	if len(back.Items) != 1 || back.Items[0].PlanCode != "free" {
		t.Fatalf("items = %+v, want the old payload preserved", back.Items)
	}
}

func TestDistributionListIgnoresUnknownProtoFields(t *testing.T) {
	// A newer Account server may add fields this client does not know. Proto3
	// decoding must preserve the known fields and drop the unknown one.
	item, err := proto.Marshal(&subscriptionv1.SubscriptionDistribution{PlanCode: "free", UserCount: 7})
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}

	raw := protowire.AppendTag(nil, 1, protowire.BytesType)
	raw = protowire.AppendBytes(raw, item)
	raw = protowire.AppendTag(raw, 99, protowire.VarintType)
	raw = protowire.AppendVarint(raw, 123)

	var wire subscriptionv1.SubscriptionDistributionList
	if err := proto.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("decode a list with an unknown field: %v", err)
	}

	back := subscriptiontransport.DistributionListToDTO(&wire)
	if len(back.Items) != 1 || back.Items[0].UserCount != 7 {
		t.Fatalf("known fields were lost while ignoring the unknown one: %+v", back.Items)
	}
	if back.DataCompleteness != "" || back.DataAvailableFrom != nil {
		t.Fatalf("unknown/absent metadata must stay omitted, got %+v", back)
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
