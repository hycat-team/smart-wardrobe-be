package feedback

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_caching "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/caching"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/learning"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// nopFeedbackLogger keeps the warning lines of the use case out of the test
// output (a nil logger is tolerated, but a fake keeps the intent explicit).
type nopFeedbackLogger struct{}

func (nopFeedbackLogger) Debug(string, ...zap.Field) {}
func (nopFeedbackLogger) Info(string, ...zap.Field)  {}
func (nopFeedbackLogger) Warn(string, ...zap.Field)  {}
func (nopFeedbackLogger) Error(string, ...zap.Field) {}
func (nopFeedbackLogger) Fatal(string, ...zap.Field) {}

// fakeSnapshotCache is an in-memory IRecommendationSnapshotCache.
type fakeSnapshotCache struct {
	snapshots map[uuid.UUID]*uc_caching.RecommendationSnapshot
	getErr    error
	gets      int
}

func newFakeSnapshotCache() *fakeSnapshotCache {
	return &fakeSnapshotCache{snapshots: map[uuid.UUID]*uc_caching.RecommendationSnapshot{}}
}

func (c *fakeSnapshotCache) Set(_ context.Context, id uuid.UUID, snapshot *uc_caching.RecommendationSnapshot) error {
	c.snapshots[id] = snapshot
	return nil
}

func (c *fakeSnapshotCache) Get(_ context.Context, id uuid.UUID) (*uc_caching.RecommendationSnapshot, error) {
	c.gets++
	if c.getErr != nil {
		return nil, c.getErr
	}
	return c.snapshots[id], nil
}

func (c *fakeSnapshotCache) Delete(_ context.Context, id uuid.UUID) error {
	delete(c.snapshots, id)
	return nil
}

// fakeLearningUseCase mirrors ApplySignal idempotency: the same SignalID is
// learned once even though every call is counted.
type fakeLearningUseCase struct {
	calls   int
	applied []dto.StyleSignalInput
	seen    map[uuid.UUID]bool
	err     error
}

func newFakeLearningUseCase() *fakeLearningUseCase {
	return &fakeLearningUseCase{seen: map[uuid.UUID]bool{}}
}

func (l *fakeLearningUseCase) ApplySignal(_ context.Context, input dto.StyleSignalInput) error {
	l.calls++
	if l.err != nil {
		return l.err
	}
	if l.seen[input.SignalID] {
		return nil
	}
	l.seen[input.SignalID] = true
	l.applied = append(l.applied, input)
	return nil
}

type feedbackFixture struct {
	uc       *RecommendationFeedbackUseCase
	cache    *fakeSnapshotCache
	learning *fakeLearningUseCase
	userID   uuid.UUID
}

func newFeedbackFixture() *feedbackFixture {
	cache := newFakeSnapshotCache()
	learningUC := newFakeLearningUseCase()
	return &feedbackFixture{
		uc: &RecommendationFeedbackUseCase{
			logger:          nopFeedbackLogger{},
			snapshotCache:   cache,
			learningUseCase: learningUC,
		},
		cache:    cache,
		learning: learningUC,
		userID:   uuid.New(),
	}
}

func (f *feedbackFixture) seedSnapshot(userID uuid.UUID, ids ...uuid.UUID) uuid.UUID {
	recommendationID := uuid.New()
	f.cache.snapshots[recommendationID] = &uc_caching.RecommendationSnapshot{
		UserID:         userID,
		FashionItemIDs: ids,
		Styles:         []string{"minimalist"},
		CreatedAt:      time.Now().UTC(),
	}
	return recommendationID
}

func feedbackRequest(recommendationID uuid.UUID, action string) dto.SubmitRecommendationFeedbackReq {
	return dto.SubmitRecommendationFeedbackReq{
		RecommendationID: recommendationID,
		Action:           action,
	}
}

func requireAppError(t *testing.T, err error, status int) *apperror.Error {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want an *apperror.Error", err)
	}
	if appErr.Status != status {
		t.Fatalf("status = %d, want %d (%s)", appErr.Status, status, appErr.Message)
	}
	return appErr
}

// --- tests -------------------------------------------------------------------

func TestSubmitFeedbackLoadsSnapshotAndAppliesSignal(t *testing.T) {
	fixture := newFeedbackFixture()
	itemIDs := []uuid.UUID{uuid.New(), uuid.New()}
	recommendationID := fixture.seedSnapshot(fixture.userID, itemIDs...)

	res, err := fixture.uc.SubmitFeedback(
		context.Background(), fixture.userID, feedbackRequest(recommendationID, "like"),
	)
	if err != nil {
		t.Fatalf("SubmitFeedback: %v", err)
	}
	if res == nil || !res.Accepted {
		t.Fatalf("response = %+v, want accepted=true", res)
	}
	if fixture.cache.gets != 1 {
		t.Fatalf("cache gets = %d, want 1", fixture.cache.gets)
	}
	if len(fixture.learning.applied) != 1 {
		t.Fatalf("applied signals = %d, want 1", len(fixture.learning.applied))
	}

	signal := fixture.learning.applied[0]
	if signal.UserID != fixture.userID {
		t.Fatalf("signal user = %s, want %s", signal.UserID, fixture.userID)
	}
	if signal.Source != learning.SignalSourceFeedbackLike {
		t.Fatalf("source = %q, want %q", signal.Source, learning.SignalSourceFeedbackLike)
	}
	if signal.RecommendationID == nil || *signal.RecommendationID != recommendationID {
		t.Fatalf("recommendation id = %v, want %s", signal.RecommendationID, recommendationID)
	}
	if len(signal.FashionItemIDs) != len(itemIDs) {
		t.Fatalf("item ids = %v, want %v", signal.FashionItemIDs, itemIDs)
	}
	for i, id := range itemIDs {
		if signal.FashionItemIDs[i] != id {
			t.Fatalf("item ids = %v, want %v", signal.FashionItemIDs, itemIDs)
		}
	}
	if signal.OccurredAt.IsZero() {
		t.Fatal("occurredAt must be set")
	}
}

func TestSubmitFeedbackSourceMapping(t *testing.T) {
	cases := map[string]string{
		"like":    learning.SignalSourceFeedbackLike,
		"dislike": learning.SignalSourceFeedbackDislike,
		"saved":   learning.SignalSourceFeedbackSaved,
	}
	for action, wantSource := range cases {
		t.Run(action, func(t *testing.T) {
			fixture := newFeedbackFixture()
			recommendationID := fixture.seedSnapshot(fixture.userID, uuid.New())

			if _, err := fixture.uc.SubmitFeedback(
				context.Background(), fixture.userID, feedbackRequest(recommendationID, action),
			); err != nil {
				t.Fatalf("SubmitFeedback: %v", err)
			}
			if got := fixture.learning.applied[0].Source; got != wantSource {
				t.Fatalf("source = %q, want %q", got, wantSource)
			}
		})
	}
}

func TestSubmitFeedbackIsIdempotentForDuplicateSubmission(t *testing.T) {
	fixture := newFeedbackFixture()
	recommendationID := fixture.seedSnapshot(fixture.userID, uuid.New())
	request := feedbackRequest(recommendationID, "like")

	if _, err := fixture.uc.SubmitFeedback(context.Background(), fixture.userID, request); err != nil {
		t.Fatalf("first SubmitFeedback: %v", err)
	}
	if _, err := fixture.uc.SubmitFeedback(context.Background(), fixture.userID, request); err != nil {
		t.Fatalf("second SubmitFeedback: %v", err)
	}

	if fixture.learning.calls != 2 {
		t.Fatalf("learning calls = %d, want 2", fixture.learning.calls)
	}
	if len(fixture.learning.applied) != 1 {
		t.Fatalf("learned signals = %d, want 1 (the deterministic id must dedupe)", len(fixture.learning.applied))
	}
}

func TestSubmitFeedbackMissingSnapshotIsNotFound(t *testing.T) {
	fixture := newFeedbackFixture()

	_, err := fixture.uc.SubmitFeedback(
		context.Background(), fixture.userID, feedbackRequest(uuid.New(), "like"),
	)
	requireAppError(t, err, http.StatusNotFound)
	if len(fixture.learning.applied) != 0 {
		t.Fatal("a missing snapshot must not learn a signal")
	}
}

func TestSubmitFeedbackZeroRecommendationIDIsNotFound(t *testing.T) {
	fixture := newFeedbackFixture()

	_, err := fixture.uc.SubmitFeedback(
		context.Background(), fixture.userID, feedbackRequest(uuid.Nil, "like"),
	)
	requireAppError(t, err, http.StatusNotFound)
	if fixture.cache.gets != 0 {
		t.Fatalf("cache gets = %d, want 0", fixture.cache.gets)
	}
}

func TestSubmitFeedbackCacheErrorDegradesToNotFound(t *testing.T) {
	fixture := newFeedbackFixture()
	fixture.cache.getErr = errors.New("redis down")

	_, err := fixture.uc.SubmitFeedback(
		context.Background(), fixture.userID, feedbackRequest(uuid.New(), "like"),
	)
	requireAppError(t, err, http.StatusNotFound)
	if len(fixture.learning.applied) != 0 {
		t.Fatal("a cache error must not learn a signal")
	}
}

func TestSubmitFeedbackRejectsForeignSnapshot(t *testing.T) {
	fixture := newFeedbackFixture()
	recommendationID := fixture.seedSnapshot(uuid.New(), uuid.New())

	_, err := fixture.uc.SubmitFeedback(
		context.Background(), fixture.userID, feedbackRequest(recommendationID, "like"),
	)
	requireAppError(t, err, http.StatusForbidden)
	if len(fixture.learning.applied) != 0 {
		t.Fatal("a foreign snapshot must not learn a signal")
	}
}

func TestSubmitFeedbackRejectsUnknownAction(t *testing.T) {
	fixture := newFeedbackFixture()
	recommendationID := fixture.seedSnapshot(fixture.userID, uuid.New())

	_, err := fixture.uc.SubmitFeedback(
		context.Background(), fixture.userID, feedbackRequest(recommendationID, "love"),
	)
	requireAppError(t, err, http.StatusBadRequest)
	if fixture.cache.gets != 0 {
		t.Fatalf("cache gets = %d, want 0: the action is validated first", fixture.cache.gets)
	}
}

func TestSubmitFeedbackPropagatesLearningError(t *testing.T) {
	fixture := newFeedbackFixture()
	recommendationID := fixture.seedSnapshot(fixture.userID, uuid.New())
	fixture.learning.err = errors.New("database down")

	_, err := fixture.uc.SubmitFeedback(
		context.Background(), fixture.userID, feedbackRequest(recommendationID, "like"),
	)
	if err == nil || err.Error() != "database down" {
		t.Fatalf("error = %v, want the learning error", err)
	}
}

func TestDeterministicFeedbackSignalID(t *testing.T) {
	userID := uuid.New()
	recommendationID := uuid.New()

	base := deterministicFeedbackSignalID(userID, recommendationID, "like")
	if again := deterministicFeedbackSignalID(userID, recommendationID, "like"); again != base {
		t.Fatalf("same input produced %s then %s", base, again)
	}
	if other := deterministicFeedbackSignalID(userID, recommendationID, "saved"); other == base {
		t.Fatal("different actions must produce different signal ids")
	}
	if other := deterministicFeedbackSignalID(uuid.New(), recommendationID, "like"); other == base {
		t.Fatal("different users must produce different signal ids")
	}
	if other := deterministicFeedbackSignalID(userID, uuid.New(), "like"); other == base {
		t.Fatal("different recommendations must produce different signal ids")
	}
}
