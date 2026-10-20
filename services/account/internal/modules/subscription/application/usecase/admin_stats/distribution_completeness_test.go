package admin_stats

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
)

// 031 T010 — completeness and clipping for the distribution report
// (FR-012, plan D4, data-model §6).
//
// The zero value of a `time.Time` bound means UNBOUNDED, never the zero
// timestamp. The old query treated the zero time as a real instant, which is
// why a period request with only one side silently included or dropped rows.

func TestResolveDistributionWindow(t *testing.T) {
	cutover := time.Date(2026, 10, 20, 5, 0, 0, 0, time.UTC)
	before := cutover.AddDate(0, 0, -10)
	after := cutover.AddDate(0, 0, 10)
	oneDayBeforeCutover := cutover.AddDate(0, 0, -1)

	cases := []struct {
		name     string
		from     time.Time
		to       time.Time
		cutover  *time.Time
		wantFrom time.Time
		want     string
	}{
		{
			name:     "both bounds zero is the complete current snapshot",
			from:     time.Time{},
			to:       time.Time{},
			cutover:  &cutover,
			wantFrom: time.Time{},
			want:     DistributionCompletenessComplete,
		},
		{
			name:     "period ending before the cutover is unavailable",
			from:     before,
			to:       oneDayBeforeCutover,
			cutover:  &cutover,
			wantFrom: before,
			want:     DistributionCompletenessUnavailable,
		},
		{
			name:     "period ending exactly at the cutover is unavailable",
			from:     before,
			to:       cutover,
			cutover:  &cutover,
			wantFrom: before,
			want:     DistributionCompletenessUnavailable,
		},
		{
			name:     "period straddling the cutover is partial and clipped",
			from:     before,
			to:       after,
			cutover:  &cutover,
			wantFrom: cutover,
			want:     DistributionCompletenessPartial,
		},
		{
			name:     "from-only period starting before the cutover is partial and clipped",
			from:     before,
			to:       time.Time{},
			cutover:  &cutover,
			wantFrom: cutover,
			want:     DistributionCompletenessPartial,
		},
		{
			name:     "to-only period ending after the cutover is partial and clipped",
			from:     time.Time{},
			to:       after,
			cutover:  &cutover,
			wantFrom: cutover,
			want:     DistributionCompletenessPartial,
		},
		{
			name:     "period starting at the cutover is complete",
			from:     cutover,
			to:       after,
			cutover:  &cutover,
			wantFrom: cutover,
			want:     DistributionCompletenessComplete,
		},
		{
			name:     "period starting after the cutover is complete",
			from:     after,
			to:       after.AddDate(0, 0, 10),
			cutover:  &cutover,
			wantFrom: after,
			want:     DistributionCompletenessComplete,
		},
		{
			name:     "missing cutover makes a period request unavailable",
			from:     before,
			to:       after,
			cutover:  nil,
			wantFrom: before,
			want:     DistributionCompletenessUnavailable,
		},
		{
			name:     "missing cutover keeps the no-period snapshot complete",
			from:     time.Time{},
			to:       time.Time{},
			cutover:  nil,
			wantFrom: time.Time{},
			want:     DistributionCompletenessComplete,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFrom, got := resolveDistributionWindow(tc.from, tc.to, tc.cutover)
			if got != tc.want {
				t.Fatalf("completeness = %q, want %q", got, tc.want)
			}
			if !gotFrom.Equal(tc.wantFrom) {
				t.Fatalf("effective from = %s, want %s", gotFrom, tc.wantFrom)
			}
		})
	}
}

// fakePeriodRepo scripts the two repository methods the distribution use case
// calls. The embedded interface keeps the other methods unimplemented so a
// future change that starts calling them fails loudly instead of silently
// returning a zero value.
type fakePeriodRepo struct {
	repositories.IUserPlanPeriodRepository

	cutover *entities.PlanHistoryMeta
	items   []repositories.SubscriptionDistributionDTO

	distributionCalls int
	lastFrom          time.Time
	lastTo            time.Time
}

func (f *fakePeriodRepo) GetHistoryCutover(context.Context) (*entities.PlanHistoryMeta, error) {
	return f.cutover, nil
}

func (f *fakePeriodRepo) GetDistribution(_ context.Context, from, to time.Time) ([]repositories.SubscriptionDistributionDTO, error) {
	f.distributionCalls++
	f.lastFrom, f.lastTo = from, to
	return f.items, nil
}

func TestGetSubscriptionDistributionMarksPeriodUnavailableWithoutCutover(t *testing.T) {
	repo := &fakePeriodRepo{cutover: nil}
	uc := NewAdminStatsUseCase(nil, nil, repo, logger.New("test", "", "error", false))

	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	result, err := uc.GetSubscriptionDistribution(context.Background(), from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DataCompleteness != DistributionCompletenessUnavailable {
		t.Fatalf("dataCompleteness = %q, want %q", result.DataCompleteness, DistributionCompletenessUnavailable)
	}
	if result.DataAvailableFrom != nil {
		t.Fatalf("dataAvailableFrom = %v, want nil: the cutover date is unknown", result.DataAvailableFrom)
	}
	if len(result.Items) != 0 {
		t.Fatalf("items = %v, want an empty list", result.Items)
	}
	if repo.distributionCalls != 0 {
		t.Fatalf("the report must not run a period query it cannot answer, got %d call(s)", repo.distributionCalls)
	}
}

func TestGetSubscriptionDistributionClipsAPartialPeriod(t *testing.T) {
	cutover := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	repo := &fakePeriodRepo{
		cutover: &entities.PlanHistoryMeta{ID: "cutover", StartedAt: cutover},
		items: []repositories.SubscriptionDistributionDTO{
			{PlanCode: "free", UserCount: 4},
		},
	}
	uc := NewAdminStatsUseCase(nil, nil, repo, logger.New("test", "", "error", false))

	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	result, err := uc.GetSubscriptionDistribution(context.Background(), from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DataCompleteness != DistributionCompletenessPartial {
		t.Fatalf("dataCompleteness = %q, want %q", result.DataCompleteness, DistributionCompletenessPartial)
	}
	if result.DataAvailableFrom == nil || !result.DataAvailableFrom.Equal(cutover) {
		t.Fatalf("dataAvailableFrom = %v, want %s", result.DataAvailableFrom, cutover)
	}
	if !repo.lastFrom.Equal(cutover) {
		t.Fatalf("query from = %s, want the cutover %s (partial data before it is not counted)", repo.lastFrom, cutover)
	}
	if !repo.lastTo.Equal(to) {
		t.Fatalf("query to = %s, want the requested %s", repo.lastTo, to)
	}
	if len(result.Items) != 1 || result.Items[0].UserCount != 4 {
		t.Fatalf("items = %+v, want the repository buckets", result.Items)
	}
}

func TestGetSubscriptionDistributionSnapshotKeepsTheCutoverDate(t *testing.T) {
	cutover := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	repo := &fakePeriodRepo{
		cutover: &entities.PlanHistoryMeta{ID: "cutover", StartedAt: cutover},
	}
	uc := NewAdminStatsUseCase(nil, nil, repo, logger.New("test", "", "error", false))

	result, err := uc.GetSubscriptionDistribution(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DataCompleteness != DistributionCompletenessComplete {
		t.Fatalf("dataCompleteness = %q, want %q", result.DataCompleteness, DistributionCompletenessComplete)
	}
	if result.DataAvailableFrom == nil || !result.DataAvailableFrom.Equal(cutover) {
		t.Fatalf("dataAvailableFrom = %v, want %s", result.DataAvailableFrom, cutover)
	}
	if !repo.lastFrom.IsZero() || !repo.lastTo.IsZero() {
		t.Fatalf("snapshot query = [%s, %s), want both bounds still zero", repo.lastFrom, repo.lastTo)
	}
}
