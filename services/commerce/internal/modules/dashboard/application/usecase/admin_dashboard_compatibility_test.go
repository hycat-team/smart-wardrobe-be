package usecase

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	subscriptionDTO "smart-wardrobe-be/services/account/dto/subscription"
	dashboardDTO "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// 031 T039 — compatibility tests for the additive distribution fields.
//
// The JSON direction "old payload -> new struct" is already pinned by
// TestSubscriptionDistributionResDecodesTheLegacyJSONShape in
// admin_dashboard_uc_test.go. This file covers the complementary directions:
// an old client decoding a NEW payload (unknown fields must be ignored), the
// exact calendar-date format of dataAvailableFrom, and proto unknown-field
// tolerance at the Commerce edge.

// TestLegacyDistributionClientDecodesTheNewPayload is FR-015/SC-006 from the
// client side: a consumer that only knows distribution/totalUsers must still
// read those fields from a payload that also carries the two new optional
// fields.
func TestLegacyDistributionClientDecodesTheNewPayload(t *testing.T) {
	newPayload := []byte(`{
		"distribution": [
			{ "planCode": "free", "userCount": 10, "percentage": 41.67 },
			{ "planCode": "premium-monthly", "userCount": 14, "percentage": 58.33 }
		],
		"totalUsers": 24,
		"dataAvailableFrom": "2026-10-20",
		"dataCompleteness": "partial"
	}`)

	// Exactly the 029 client shape: no dataAvailableFrom, no dataCompleteness.
	var legacy struct {
		Distribution []struct {
			PlanCode   string  `json:"planCode"`
			UserCount  int     `json:"userCount"`
			Percentage float64 `json:"percentage"`
		} `json:"distribution"`
		TotalUsers int `json:"totalUsers"`
	}

	if err := json.Unmarshal(newPayload, &legacy); err != nil {
		t.Fatalf("old client must ignore the additive fields, got error: %v", err)
	}
	if legacy.TotalUsers != 24 {
		t.Fatalf("totalUsers = %d, want 24", legacy.TotalUsers)
	}
	if len(legacy.Distribution) != 2 {
		t.Fatalf("distribution has %d row(s), want 2", len(legacy.Distribution))
	}
	if legacy.Distribution[1].PlanCode != "premium-monthly" ||
		legacy.Distribution[1].UserCount != 14 ||
		legacy.Distribution[1].Percentage != 58.33 {
		t.Fatalf("old fields changed: %+v", legacy.Distribution[1])
	}

	// The new fields really are present; the old client simply ignores them.
	var raw map[string]any
	if err := json.Unmarshal(newPayload, &raw); err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	if raw["dataAvailableFrom"] != "2026-10-20" || raw["dataCompleteness"] != "partial" {
		t.Fatalf("new fields are missing from the payload: %v", raw)
	}
}

// TestDataAvailableFromIsAVNCalendarDate pins FR-012/D5: the response field is
// the VN calendar date in YYYY-MM-DD form, never an RFC 3339 timestamp and never
// the UTC day of the cutover instant.
func TestDataAvailableFromIsAVNCalendarDate(t *testing.T) {
	// 18:00 UTC is already 2026-10-21 in VN (+07).
	cutover := time.Date(2026, 10, 20, 18, 0, 0, 0, time.UTC)
	reader := &mockSubscriptionReader{
		distribution: &subscriptionDTO.SubscriptionDistributionResultDTO{
			DataAvailableFrom: &cutover,
			DataCompleteness:  "complete",
		},
	}
	uc := NewAdminDashboardUseCase(
		&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{},
		&mockAICostReader{}, reader,
	)

	res, err := uc.GetSubscriptionDistribution(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.DataAvailableFrom == nil {
		t.Fatal("dataAvailableFrom must be present when the account service provides the cutover")
	}

	payload, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var decoded struct {
		DataAvailableFrom *string `json:"dataAvailableFrom"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.DataAvailableFrom == nil {
		t.Fatalf("dataAvailableFrom disappeared from JSON: %s", payload)
	}

	dateOnly := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateOnly.MatchString(*decoded.DataAvailableFrom) {
		t.Fatalf("dataAvailableFrom = %q, want a YYYY-MM-DD calendar date", *decoded.DataAvailableFrom)
	}
	if *decoded.DataAvailableFrom != timeutils.StartOfDayInVN(cutover).Format("2006-01-02") {
		t.Fatalf("dataAvailableFrom = %q, want the VN day 2026-10-21", *decoded.DataAvailableFrom)
	}
}

// TestDistributionListIgnoresUnknownProtoFields is the wire half of SC-006: a
// newer Account server may add fields this Commerce client does not know, and
// proto3 decoding must preserve the known fields. The equivalent account-side
// test lives in services/account/contract/subscription/transport/mapper_test.go.
func TestDistributionListIgnoresUnknownProtoFields(t *testing.T) {
	item, err := proto.Marshal(&subscriptionv1.SubscriptionDistribution{
		PlanCode:  "free",
		UserCount: 7,
	})
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}

	raw := protowire.AppendTag(nil, 1, protowire.BytesType)
	raw = protowire.AppendBytes(raw, item)
	raw = protowire.AppendTag(raw, 99, protowire.VarintType) // unknown field
	raw = protowire.AppendVarint(raw, 123)

	var wire subscriptionv1.SubscriptionDistributionList
	if err := proto.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("decode a list with an unknown field: %v", err)
	}

	if len(wire.GetItems()) != 1 || wire.GetItems()[0].GetPlanCode() != "free" ||
		wire.GetItems()[0].GetUserCount() != 7 {
		t.Fatalf("known fields were lost while ignoring the unknown one: %+v", wire.GetItems())
	}
	if wire.GetDataCompleteness() != subscriptionv1.DistributionDataCompleteness_DISTRIBUTION_DATA_COMPLETENESS_UNSPECIFIED {
		t.Fatalf("unknown field must not invent a completeness value, got %v", wire.GetDataCompleteness())
	}
	if wire.GetDataAvailableFrom() != nil {
		t.Fatalf("unknown field must not invent a cutover, got %v", wire.GetDataAvailableFrom())
	}
}

// Ensure the response DTO still exposes exactly the documented JSON keys.
// This is the source-side count T040 compares the guide against.
func TestSubscriptionDistributionResJSONKeys(t *testing.T) {
	payload, err := json.Marshal(dashboardDTO.SubscriptionDistributionRes{
		Distribution: []dashboardDTO.SubscriptionDistributionDTO{{
			PlanCode: "free", UserCount: 1, Percentage: 100,
		}},
		TotalUsers: 1,
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(payload, &keys); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, key := range []string{"distribution", "totalUsers"} {
		if _, ok := keys[key]; !ok {
			t.Errorf("required JSON key %q is missing: %s", key, payload)
		}
	}
	// Both optional fields are omitted when absent (omitempty).
	for _, key := range []string{"dataAvailableFrom", "dataCompleteness"} {
		if _, ok := keys[key]; ok {
			t.Errorf("optional JSON key %q must be omitted when empty: %s", key, payload)
		}
	}
}
