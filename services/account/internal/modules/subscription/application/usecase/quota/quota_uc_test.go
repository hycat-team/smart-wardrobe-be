package quota

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	sharedrepos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

// 025 T059-T062b: the AI usage counter must not lose an increment.
//
// The defect is old and pre-dates the campaign. `UpdateOutfitQuota` and
// `UpdateAiChatQuota` did read → add → write as five separate statements with no
// transaction, so two concurrent AI calls both read 29, both computed 30, and both
// wrote 30. Three concurrent calls wrote 30. The customer was charged for one
// generation and got one, and the counters drifted further apart with every
// request until the day reset hid it.
//
// `GetByUserIDWithLock` (`user_daily_quota_repo_impl.go`) has existed the whole
// time and had NO production caller. That is what the fix connects.

// serialQuotaRepo is a quota row behind a row lock, with an optional per-call
// delay that widens the window between the read and the write.
//
// The lock models `SELECT ... FOR UPDATE` inside a transaction: the second caller
// blocks until the first commits. The delay is what makes this a REGRESSION test
// rather than a race hunt — without it the goroutines interleave loosely, each
// read may land after the previous write, and the test passes even with the fix
// removed.
//
// `committed` is set by the unit of work, not by the repository: the row lock is
// released at COMMIT, which is the behaviour the fix depends on and the reason
// the lock is taken on the READ rather than around the whole method.
type serialQuotaRepo struct {
	mu       sync.Mutex
	quota    *entities.UserDailyQuota
	updates  int
	delay    time.Duration
	lockedBy bool
	// waiters counts how many callers blocked on the row lock, so a test can
	// assert the lock is actually CONTENDED (T059) rather than merely present.
	waiters int
}

func (r *serialQuotaRepo) read(userID uuid.UUID) *entities.UserDailyQuota {
	if r.quota == nil {
		return nil
	}
	copied := *r.quota
	return &copied
}

func (r *serialQuotaRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.read(userID), nil
}

// GetByUserIDWithLock takes the row lock and holds it until the transaction ends.
//
// `release` is what the unit of work calls at COMMIT. Modelling the release
// explicitly — rather than unlocking inside Update — is the point: a lock released
// by the write would make the read-modify-write atomic for exactly one call and
// would silently stop being atomic the moment a second read were added.
func (r *serialQuotaRepo) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	for {
		r.mu.Lock()
		if !r.lockedBy {
			r.lockedBy = true
			copied := r.read(userID)
			r.mu.Unlock()
			return copied, nil
		}
		r.waiters++
		r.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
}

// release drops the row lock, modelling COMMIT or ROLLBACK.
func (r *serialQuotaRepo) release() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lockedBy = false
}

func (r *serialQuotaRepo) Create(ctx context.Context, e *entities.UserDailyQuota) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quota = e
	return nil
}

func (r *serialQuotaRepo) CreateIfAbsent(ctx context.Context, e *entities.UserDailyQuota) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.quota != nil {
		return false, nil
	}
	r.quota = e
	return true, nil
}

func (r *serialQuotaRepo) Update(ctx context.Context, e *entities.UserDailyQuota) error {
	// The delay sits between the locked read and the write, which is exactly where
	// the real race lives. Without it a broken implementation's goroutines may each
	// read after the previous write, and the test would pass.
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates++
	r.quota = e
	return nil
}

func (r *serialQuotaRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.UserDailyQuota, error) {
	return nil, nil
}

func (r *serialQuotaRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.UserDailyQuota, error) {
	return nil, nil
}

func (r *serialQuotaRepo) GetAll(ctx context.Context) ([]*entities.UserDailyQuota, error) {
	return nil, nil
}

func (r *serialQuotaRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func (r *serialQuotaRepo) BatchUpdate(ctx context.Context, es []*entities.UserDailyQuota) error {
	return nil
}

func (r *serialQuotaRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

// lockReleaseUow releases the fake's row lock when the closure returns, modelling
// `pg_advisory_xact_lock` / `FOR UPDATE` being released at COMMIT or ROLLBACK.
//
// The release is unconditional, including on error. A lock that outlives a
// rollback is the leak the transaction-scoped lock was chosen to avoid, and a
// fake that released only on success would let that pass.
type lockReleaseUow struct {
	repo *serialQuotaRepo
}

func (u lockReleaseUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	defer u.repo.release()
	return fn(ctx)
}

var _ repositories.IUserDailyQuotaRepository = (*serialQuotaRepo)(nil)
var _ sharedrepos.IUnitOfWork = lockReleaseUow{}

// quotaRaceUseCase builds the use case with the serialising quota repository.
//
// The plan allows 30 a day, because the counters in these tests START near the
// limit — 29 of 30 is the case FR-054 was written about, and a test that started
// at 0 of 2 would prove nothing about the boundary.
func quotaRaceUseCase(t *testing.T, quota *entities.UserDailyQuota, delay time.Duration) (*UserQuotaUseCase, *serialQuotaRepo, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	plan := quotaTestPlan()
	plan.Slug = "premium-monthly"
	plan.AiOutfitDailyQuota = 30
	plan.AiChatDailyQuota = 30
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: plan.ID,
		SubscriptionPlan:   plan,
		CurrentPlanCode:    plan.Slug,
		CurrentPlanKind:    plan.PlanKind,
	}
	repo := &serialQuotaRepo{quota: quota, delay: delay}
	uc := NewUserQuotaUseCase(
		repo,
		&fakeQuotaSubRepo{sub: sub},
		&fakeQuotaPlanRepo{plan: plan},
		nil,
		lockReleaseUow{repo: repo},
	).(*UserQuotaUseCase)
	return uc, repo, userID
}

// TestUpdateOutfitQuotaDoesNotLoseConcurrentIncrements is T059 / SC-010.
//
// Three concurrent increments from 29 of 30 must end at 30, and exactly ONE of the
// three may succeed.
//
// Both halves matter, and neither alone distinguishes the fix from the bug:
//
//   - the COUNTER alone. The broken code produced 30 as well — three unlocked
//     read-modify-writes all computed 30 and all wrote 30. Asserting only the final
//     value would pass on the code that loses increments.
//   - the SUCCESS COUNT alone is not enough either, because "how many were
//     allowed" is only meaningful next to what the counter ended up as.
//
// Together they are the whole rule: from 29 of 30 there is exactly ONE slot left,
// so exactly one request may consume it and the other two must be refused with the
// business error. The broken code let all three through — which is the customer
// being charged for three generations and given one.
func TestUpdateOutfitQuotaDoesNotLoseConcurrentIncrements(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 29,
		LastResetDate:        time.Now(),
	}
	uc, repo, _ := quotaRaceUseCase(t, quota, 2*time.Millisecond)

	const racers = 3
	var wg sync.WaitGroup
	errs := make([]error, racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = uc.UpdateOutfitQuota(context.Background(), userID, 1)
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded, refused := 0, 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		// A refusal must be the BUSINESS error, not a database failure: the two
		// mean opposite things to the caller — one is "come back tomorrow", the
		// other is "something is broken". The comparison is on the message,
		// because `apperror` factories return distinct values rather than a
		// sentinel, which is the project's convention (errors.go).
		if !strings.Contains(err.Error(), subscriptionerrors.ErrAiOutfitQuotaExceeded().Error()) {
			t.Fatalf("expected either success or the business refusal, got %v", err)
		}
		refused++
	}
	if succeeded != 1 || refused != 2 {
		t.Fatalf("expected exactly 1 grant and 2 refusals from 29 of 30, got %d and %d — with no transaction, all three read 29 and all three wrote 30", succeeded, refused)
	}
	stored, _ := repo.GetByUserID(context.Background(), userID)
	if stored.OutfitRecommendCount != 30 {
		t.Fatalf("expected the counter to end at 30, got %d", stored.OutfitRecommendCount)
	}
}

// TestUpdateOutfitQuotaRejectsBeyondLimit is T060 / SC-010: the 31st is refused
// with a business error, and refused AFTER the lock, so the refusal is decided on
// the current number rather than on a stale read.
func TestUpdateOutfitQuotaRejectsBeyondLimit(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 30,
		LastResetDate:        time.Now(),
	}
	uc, repo, _ := quotaRaceUseCase(t, quota, 0)

	err := uc.UpdateOutfitQuota(context.Background(), userID, 1)
	if err == nil {
		t.Fatalf("expected a business error for the 31st generation (SC-010)")
	}
	stored, _ := repo.GetByUserID(context.Background(), userID)
	if stored.OutfitRecommendCount != 30 {
		t.Fatalf("a refused increment must not change the counter, got %d", stored.OutfitRecommendCount)
	}
	if repo.updates != 0 {
		t.Fatalf("expected no write for a refused increment, got %d writes", repo.updates)
	}
}

// TestUpdateAiChatQuotaDoesNotLoseConcurrentIncrements is T065's requirement: the
// chatbot counter had the same defect in its own method, and a separate method
// would have kept it after the first was fixed.
//
// Room for all three (4 of 30), so this measures the ARITHMETIC rather than the
// ceiling: the broken code leaves the counter at 5 no matter how many increments
// arrive, because all three read 4.
func TestUpdateAiChatQuotaDoesNotLoseConcurrentIncrements(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:        userID,
		AiUsageCount:  4,
		LastResetDate: time.Now(),
	}
	uc, repo, _ := quotaRaceUseCase(t, quota, 2*time.Millisecond)

	const racers = 3
	var wg sync.WaitGroup
	errs := make([]error, racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = uc.UpdateAiChatQuota(context.Background(), userID, 1)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("racer %d: unexpected error: %v", i, err)
		}
	}
	stored, _ := repo.GetByUserID(context.Background(), userID)
	if stored.AiUsageCount != 7 {
		t.Fatalf("expected 7 after 3 concurrent increments from 4, got %d — an increment was LOST", stored.AiUsageCount)
	}
}

// TestGetAndResetDailyQuotaIsNotBlockedByAWrite is T061 / FR-055.
//
// The read path is called on EVERY AI request, so if it queued behind the writers
// it would serialise the whole AI path behind its own accounting. FR-055 asks for
// the read to keep working while a write is in flight, and that means the read
// must NOT take the lock.
//
// The fake distinguishes the two: `GetByUserID` is lock-free, so a read issued
// while the writer holds the lock returns immediately. The test would DEADLOCK if
// the read took the same lock, which is a failure mode better caught by a timeout
// than by an assertion.
func TestGetAndResetDailyQuotaIsNotBlockedByAWrite(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:        userID,
		AiUsageCount:  1,
		LastResetDate: time.Now(),
	}
	// A long delay: the writer holds the lock for 300ms.
	uc, repo, _ := quotaRaceUseCase(t, quota, 300*time.Millisecond)

	writeDone := make(chan error, 1)
	go func() {
		writeDone <- uc.UpdateOutfitQuota(context.Background(), userID, 1)
	}()

	// Let the writer take the lock, then read while it holds it.
	time.Sleep(30 * time.Millisecond)

	readDone := make(chan error, 1)
	go func() {
		_, err := uc.GetAndResetDailyQuota(context.Background(), userID)
		readDone <- err
	}()

	select {
	case err := <-readDone:
		if err != nil {
			t.Fatalf("unexpected error on the read path: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("GetAndResetDailyQuota blocked behind an in-flight write (FR-055): the read path must stay lock-free")
	}

	if err := <-writeDone; err != nil {
		t.Fatalf("the concurrent write failed: %v", err)
	}
	if repo.updates < 1 {
		t.Fatalf("expected the write to have been persisted")
	}
}

// TestGetAndResetDailyQuotaResetsOnANewDayWithoutManualAction is T062 / FR-054:
// the counter returns to zero when the day rolls over, with no cron job and no
// operator action.
func TestGetAndResetDailyQuotaResetsOnANewDayWithoutManualAction(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 5,
		AiUsageCount:         9,
		LastResetDate:        time.Now().AddDate(0, 0, -1),
	}
	uc, repo, _ := quotaRaceUseCase(t, quota, 0)

	dto, err := uc.GetAndResetDailyQuota(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.OutfitRecommendCount != 0 || dto.AiUsageCount != 0 {
		t.Fatalf("expected the counters to reset on a new day, got %+v", dto)
	}
	if repo.updates != 1 {
		t.Fatalf("expected exactly 1 reset write, got %d", repo.updates)
	}
	stored, _ := repo.GetByUserID(context.Background(), userID)
	if stored.LastResetDate.Before(time.Now().Add(-25 * time.Hour)) {
		t.Fatalf("expected LastResetDate advanced to the current day, got %s", stored.LastResetDate)
	}
}

// TestGetAndResetDailyQuotaSameDayWritesNothing is the other half of FR-054, and
// it is what keeps the read path cheap: on the same day there is nothing to do,
// so there is nothing to write — and no transaction to open.
func TestGetAndResetDailyQuotaSameDayWritesNothing(t *testing.T) {
	userID := uuid.New()
	quota := &entities.UserDailyQuota{
		UserID:               userID,
		OutfitRecommendCount: 2,
		AiUsageCount:         1,
		LastResetDate:        time.Now(),
	}
	uc, repo, _ := quotaRaceUseCase(t, quota, 0)

	dto, err := uc.GetAndResetDailyQuota(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.OutfitRecommendCount != 2 || dto.AiUsageCount != 1 {
		t.Fatalf("expected the same-day counters preserved, got %+v", dto)
	}
	if repo.updates != 0 {
		t.Fatalf("expected no write on the same day, got %d", repo.updates)
	}
}
