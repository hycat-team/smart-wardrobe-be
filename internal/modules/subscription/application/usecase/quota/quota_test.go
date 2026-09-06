package quota

import (
	"context"
	"testing"
	"time"

	_ "smart-wardrobe-be/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---- Fakes ----

type fakeQuotaRepo struct {
	quota   *entities.UserDailyQuota
	updates int
	creates int
}

func (f *fakeQuotaRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	return f.quota, nil
}
func (f *fakeQuotaRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	return f.quota, nil
}
func (f *fakeQuotaRepo) Create(ctx context.Context, e *entities.UserDailyQuota) error {
	f.creates++
	f.quota = e
	return nil
}
func (f *fakeQuotaRepo) Update(ctx context.Context, e *entities.UserDailyQuota) error {
	f.updates++
	return nil
}
func (f *fakeQuotaRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserDailyQuota, error) {
	return nil, nil
}
func (f *fakeQuotaRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserDailyQuota, error) {
	return nil, nil
}
func (f *fakeQuotaRepo) GetAll(ctx context.Context) ([]*entities.UserDailyQuota, error) {
	return nil, nil
}
func (f *fakeQuotaRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeQuotaRepo) BatchUpdate(ctx context.Context, es []*entities.UserDailyQuota) error {
	return nil
}
func (f *fakeQuotaRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeQuotaSubRepo struct {
	sub *entities.UserSubscription
}

func (f *fakeQuotaSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.sub, nil
}
func (f *fakeQuotaSubRepo) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	return nil
}
func (f *fakeQuotaSubRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	return f.sub, nil
}
func (f *fakeQuotaSubRepo) GetByUserIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) GetSubscriptionsExpiringWithin(ctx context.Context, from, to time.Time) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) BulkCreate(ctx context.Context, subs []*entities.UserSubscription) error {
	return nil
}
func (f *fakeQuotaSubRepo) GetDistributionByPlanCode(ctx context.Context) ([]repositories.SubscriptionDistributionDTO, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) ListAdmin(ctx context.Context, filter repositories.AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error) {
	return nil, 0, nil
}
func (f *fakeQuotaSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) GetAll(ctx context.Context) ([]*entities.UserSubscription, error) {
	return nil, nil
}
func (f *fakeQuotaSubRepo) Create(ctx context.Context, e *entities.UserSubscription) error {
	return nil
}
func (f *fakeQuotaSubRepo) Update(ctx context.Context, e *entities.UserSubscription) error {
	return nil
}
func (f *fakeQuotaSubRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeQuotaSubRepo) BatchUpdate(ctx context.Context, es []*entities.UserSubscription) error {
	return nil
}
func (f *fakeQuotaSubRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type fakeQuotaPlanRepo struct {
	plan *entities.SubscriptionPlan
}

func (f *fakeQuotaPlanRepo) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	return f.plan, nil
}
func (f *fakeQuotaPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlan, error) {
	return f.plan, nil
}
func (f *fakeQuotaPlanRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	return []*entities.SubscriptionPlan{f.plan}, nil
}
func (f *fakeQuotaPlanRepo) GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	return f.plan, nil
}
func (f *fakeQuotaPlanRepo) GetBySlugAnyStatus(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	return f.plan, nil
}
func (f *fakeQuotaPlanRepo) ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeQuotaPlanRepo) GetAll(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}
func (f *fakeQuotaPlanRepo) Create(ctx context.Context, e *entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeQuotaPlanRepo) Update(ctx context.Context, e *entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeQuotaPlanRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeQuotaPlanRepo) BatchUpdate(ctx context.Context, es []*entities.SubscriptionPlan) error {
	return nil
}
func (f *fakeQuotaPlanRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// ---- Helpers ----

func quotaTestPlan() *entities.SubscriptionPlan {
	return &entities.SubscriptionPlan{
		Slug:               "premium",
		Name:               "Premium",
		IsActive:           true,
		Price:              decimal.NewFromInt(99000),
		AiOutfitDailyQuota: 2,
		AiChatDailyQuota:   5,
		DurationDays:       durationDaysPtr2(30),
	}
}

func durationDaysPtr2(d int) *int { return &d }

func newQuotaTestUseCase(t *testing.T, quota *entities.UserDailyQuota) (*UserQuotaUseCase, *fakeQuotaRepo, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	plan := quotaTestPlan()
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: plan.ID,
		SubscriptionPlan:   plan,
		CurrentPlanCode:    plan.Slug,
		CurrentPlanKind:    plan.PlanKind,
	}
	quotaRepo := &fakeQuotaRepo{quota: quota}
	subRepo := &fakeQuotaSubRepo{sub: sub}
	planRepo := &fakeQuotaPlanRepo{plan: plan}
	uc := NewUserQuotaUseCase(quotaRepo, subRepo, planRepo)
	return uc.(*UserQuotaUseCase), quotaRepo, userID
}

// ---- Tests ----

func TestGetAndResetDailyQuotaReturnsSameDayCounters(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 1,
		AiUsageCount:         2,
		LastResetDate:        time.Now(),
	}
	uc, quotaRepo, userID := newQuotaTestUseCase(t, quota)

	res, err := uc.GetAndResetDailyQuota(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OutfitRecommendCount != 1 || res.AiUsageCount != 2 {
		t.Fatalf("expected counters preserved, got %+v", res)
	}
	if quotaRepo.updates != 0 {
		t.Fatalf("expected no reset update on same day, got %d updates", quotaRepo.updates)
	}
}

func TestGetAndResetDailyQuotaResetsCountersOnNewDay(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 5,
		AiUsageCount:         9,
		LastResetDate:        time.Now().AddDate(0, 0, -1),
	}
	uc, quotaRepo, userID := newQuotaTestUseCase(t, quota)

	res, err := uc.GetAndResetDailyQuota(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OutfitRecommendCount != 0 || res.AiUsageCount != 0 {
		t.Fatalf("expected counters reset, got %+v", res)
	}
	if quotaRepo.updates != 1 {
		t.Fatalf("expected one reset update, got %d", quotaRepo.updates)
	}
}

func TestUpdateOutfitQuotaRejectsBeyondDailyLimit(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 1,
		LastResetDate:        time.Now(),
	}
	uc, _, userID := newQuotaTestUseCase(t, quota) // plan allows 2 per day

	err := uc.UpdateOutfitQuota(context.Background(), userID, 2)
	if err == nil {
		t.Fatal("expected outfit quota exceeded error, got nil")
	}
}

func TestUpdateOutfitQuotaAcceptsWithinDailyLimit(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 1,
		LastResetDate:        time.Now(),
	}
	uc, quotaRepo, userID := newQuotaTestUseCase(t, quota)

	if err := uc.UpdateOutfitQuota(context.Background(), userID, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quotaRepo.updates != 1 {
		t.Fatalf("expected quota update persisted, got %d updates", quotaRepo.updates)
	}
}

func TestUpdateAiChatQuotaRejectsBeyondDailyLimit(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:        userID,
		AiUsageCount:  5,
		LastResetDate: time.Now(),
	}
	uc, _, userID := newQuotaTestUseCase(t, quota) // plan allows 5 per day

	err := uc.UpdateAiChatQuota(context.Background(), userID, 1)
	if err == nil {
		t.Fatal("expected ai chat quota exceeded error, got nil")
	}
}

func (f *fakeQuotaSubRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserSubscription, error) {
	return nil, nil
}
