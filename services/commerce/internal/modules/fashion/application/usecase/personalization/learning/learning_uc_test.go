package learning

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// nopLearningLogger keeps the debug lines of the use case out of the test
// output. A nil logger would panic on the disabled/duplicate paths.
type nopLearningLogger struct{}

func (nopLearningLogger) Debug(string, ...zap.Field) {}
func (nopLearningLogger) Info(string, ...zap.Field)  {}
func (nopLearningLogger) Warn(string, ...zap.Field)  {}
func (nopLearningLogger) Error(string, ...zap.Field) {}
func (nopLearningLogger) Fatal(string, ...zap.Field) {}

// fakeSignalRepo is an in-memory IStyleSignalRepository whose InsertIfAbsent
// honours the PRIMARY KEY contract: the same id is inserted once.
type fakeSignalRepo struct {
	inserted    map[uuid.UUID]bool
	signals     []*entities.StyleSignal
	insertCalls int
	insertErr   error
}

func (r *fakeSignalRepo) InsertIfAbsent(_ context.Context, signal *entities.StyleSignal) (bool, error) {
	r.insertCalls++
	if r.insertErr != nil {
		return false, r.insertErr
	}
	if r.inserted == nil {
		r.inserted = map[uuid.UUID]bool{}
	}
	if r.inserted[signal.ID] {
		return false, nil
	}
	r.inserted[signal.ID] = true
	r.signals = append(r.signals, signal)
	return true, nil
}

func (r *fakeSignalRepo) CountByUserID(_ context.Context, _ uuid.UUID) (int64, error) {
	return int64(len(r.signals)), nil
}

// fakeProfileRepo mirrors UpdateLearnedWithLock: mutate runs on a copy and only
// the learned_* columns are persisted. It also flags a mutate callback that
// changed an explicit preference, which is what the real repository would have
// silently ignored.
type fakeProfileRepo struct {
	rows            map[uuid.UUID]*entities.UserStyleProfile
	updateCalls     int
	updateErr       error
	explicitMutated bool
}

func newFakeProfileRepo() *fakeProfileRepo {
	return &fakeProfileRepo{rows: map[uuid.UUID]*entities.UserStyleProfile{}}
}

func (r *fakeProfileRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*entities.UserStyleProfile, error) {
	row, ok := r.rows[userID]
	if !ok {
		return nil, nil
	}
	return cloneLearningProfile(row), nil
}

func (r *fakeProfileRepo) Upsert(_ context.Context, profile *entities.UserStyleProfile) error {
	r.rows[profile.UserID] = cloneLearningProfile(profile)
	return nil
}

func (r *fakeProfileRepo) Delete(_ context.Context, userID uuid.UUID) error {
	delete(r.rows, userID)
	return nil
}

func (r *fakeProfileRepo) UpdateLearnedWithLock(
	_ context.Context,
	userID uuid.UUID,
	mutate func(profile *entities.UserStyleProfile) error,
) (*entities.UserStyleProfile, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}

	stored, ok := r.rows[userID]
	if !ok {
		stored = &entities.UserStyleProfile{
			UserID:         userID,
			Styles:         entities.StyleList{},
			ExcludedStyles: entities.StyleList{},
			Enabled:        true,
			LearnedStyles:  entities.StyleWeightMap{},
		}
	}

	work := cloneLearningProfile(stored)
	if err := mutate(work); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(work.Styles, stored.Styles) ||
		!reflect.DeepEqual(work.ExcludedStyles, stored.ExcludedStyles) ||
		work.Enabled != stored.Enabled {
		r.explicitMutated = true
	}

	persisted := cloneLearningProfile(stored)
	persisted.LearnedStyles = cloneWeights(work.LearnedStyles)
	persisted.LearnedEmbedding = append(entities.Vector(nil), work.LearnedEmbedding...)
	persisted.LearnedAt = work.LearnedAt
	r.rows[userID] = persisted
	r.updateCalls++
	return cloneLearningProfile(persisted), nil
}

// fakeItemRepo answers only GetByIDs; the rest of the interface is never used by
// the use case.
type fakeItemRepo struct {
	repositories.IFashionItemRepository
	items []*entities.FashionItem
	calls int
	err   error
}

func (r *fakeItemRepo) GetByIDs(_ context.Context, _ []uuid.UUID) ([]*entities.FashionItem, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return r.items, nil
}

type directUnitOfWork struct {
	executes int
}

func (u *directUnitOfWork) Execute(ctx context.Context, fn func(context.Context) error) error {
	u.executes++
	return fn(ctx)
}

type learningFixture struct {
	uc          uc_interfaces.IStyleLearningUseCase
	profileRepo *fakeProfileRepo
	signalRepo  *fakeSignalRepo
	itemRepo    *fakeItemRepo
	uow         *directUnitOfWork
	userID      uuid.UUID
}

func newLearningFixture(t *testing.T, enabled bool) *learningFixture {
	t.Helper()

	cfg := &config.Config{}
	cfg.Personalization.Enabled = enabled
	cfg.Personalization.LearningDecay = 0.9

	profileRepo := newFakeProfileRepo()
	signalRepo := &fakeSignalRepo{}
	itemRepo := &fakeItemRepo{}
	uow := &directUnitOfWork{}

	return &learningFixture{
		uc: NewStyleLearningUseCase(
			cfg, nopLearningLogger{}, signalRepo, profileRepo, itemRepo, uow,
		),
		profileRepo: profileRepo,
		signalRepo:  signalRepo,
		itemRepo:    itemRepo,
		uow:         uow,
		userID:      uuid.New(),
	}
}

func (f *learningFixture) seedProfile(profile *entities.UserStyleProfile) uuid.UUID {
	if profile.UserID == uuid.Nil {
		profile.UserID = f.userID
	}
	f.profileRepo.rows[profile.UserID] = profile
	return profile.UserID
}

func styleItem(style string) *entities.FashionItem {
	return &entities.FashionItem{Style: &style}
}

func vectorItem(style string, embedding ...float32) *entities.FashionItem {
	item := styleItem(style)
	item.Embedding = entities.Vector(embedding)
	return item
}

func signalInput(userID uuid.UUID, itemIDs ...uuid.UUID) dto.StyleSignalInput {
	return dto.StyleSignalInput{
		SignalID:       uuid.New(),
		UserID:         userID,
		Source:         SignalSourceOutfitSaved,
		OutfitID:       uuidPtr(uuid.New()),
		FashionItemIDs: itemIDs,
		OccurredAt:     time.Now().UTC(),
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID { return &id }

// --- tests -------------------------------------------------------------------

// decay: an outfit without usable metadata still ages the learned profile, and
// learned_at records the update.
func TestApplySignalDecaysPreviousWeights(t *testing.T) {
	fixture := newLearningFixture(t, true)
	userID := fixture.seedProfile(&entities.UserStyleProfile{
		UserID:         fixture.userID,
		Styles:         entities.StyleList{"minimalist"},
		ExcludedStyles: entities.StyleList{"streetwear"},
		Enabled:        false,
		LearnedStyles:  entities.StyleWeightMap{"minimalist": 1.0},
	})
	itemID := uuid.New()
	fixture.itemRepo.items = []*entities.FashionItem{{}}

	input := signalInput(userID, itemID)
	if err := fixture.uc.ApplySignal(context.Background(), input); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	stored := fixture.profileRepo.rows[userID]
	if got, want := stored.LearnedStyles["minimalist"], 0.9; math.Abs(got-want) > 1e-9 {
		t.Fatalf("learned weight = %v, want %v (decay 0.9)", got, want)
	}
	if stored.LearnedAt == nil {
		t.Fatal("learned_at must be set by a learning update")
	}
	if fixture.profileRepo.explicitMutated {
		t.Fatal("the learning path must not touch styles/excluded_styles/enabled")
	}
	if fixture.uow.executes != 1 {
		t.Fatalf("uow executes = %d, want 1", fixture.uow.executes)
	}
}

// Source weights: each source contributes its documented weight; a dislike is
// clamped away instead of storing a negative value.
func TestApplySignalSourceWeights(t *testing.T) {
	cases := []struct {
		source     string
		wantWeight float64
		wantCode   int16
	}{
		{SignalSourceOutfitSaved, 1.0, entities.StyleSignalSourceOutfitSaved},
		{SignalSourceFeedbackLike, 0.5, entities.StyleSignalSourceFeedbackLike},
		{SignalSourceFeedbackSaved, 0.25, entities.StyleSignalSourceFeedbackSaved},
		{SignalSourceFeedbackDislike, 0, entities.StyleSignalSourceFeedbackDislike},
	}

	for _, testCase := range cases {
		t.Run(testCase.source, func(t *testing.T) {
			fixture := newLearningFixture(t, true)
			fixture.seedProfile(&entities.UserStyleProfile{UserID: fixture.userID, Enabled: true})
			fixture.itemRepo.items = []*entities.FashionItem{styleItem("minimalist")}

			input := signalInput(fixture.userID, uuid.New())
			input.Source = testCase.source
			if err := fixture.uc.ApplySignal(context.Background(), input); err != nil {
				t.Fatalf("ApplySignal: %v", err)
			}

			stored := fixture.profileRepo.rows[fixture.userID]
			got := stored.LearnedStyles["minimalist"]
			if math.Abs(got-testCase.wantWeight) > 1e-9 {
				t.Fatalf("learned weight = %v, want %v for source %q", got, testCase.wantWeight, testCase.source)
			}
			if _, negative := stored.LearnedStyles["minimalist"]; negative && got < 0 {
				t.Fatalf("a learned weight must never be negative: %v", got)
			}
			if fixture.signalRepo.signals[0].Source != testCase.wantCode {
				t.Fatalf("stored source = %d, want %d", fixture.signalRepo.signals[0].Source, testCase.wantCode)
			}
		})
	}
}

// Normalized counts: two different styles in one outfit contribute 0.5 each.
func TestApplySignalUsesNormalizedStyleCounts(t *testing.T) {
	fixture := newLearningFixture(t, true)
	fixture.seedProfile(&entities.UserStyleProfile{UserID: fixture.userID, Enabled: true})
	fixture.itemRepo.items = []*entities.FashionItem{styleItem("minimalist"), styleItem("streetwear")}

	if err := fixture.uc.ApplySignal(context.Background(), signalInput(fixture.userID, uuid.New(), uuid.New())); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	learned := fixture.profileRepo.rows[fixture.userID].LearnedStyles
	if len(learned) != 2 {
		t.Fatalf("learned = %v, want two styles", learned)
	}
	for style, weight := range learned {
		if math.Abs(weight-0.5) > 1e-9 {
			t.Fatalf("learned[%s] = %v, want 0.5", style, weight)
		}
	}
}

// Top-K: only the ten strongest labels survive a signal.
func TestApplySignalKeepsTopTenStyles(t *testing.T) {
	fixture := newLearningFixture(t, true)
	learned := entities.StyleWeightMap{}
	for index := 0; index < 11; index++ {
		learned["style-"+string(rune('a'+index))] = 1.1 - float64(index)*0.1
	}
	userID := fixture.seedProfile(&entities.UserStyleProfile{
		UserID:        fixture.userID,
		Enabled:       true,
		LearnedStyles: learned,
	})
	fixture.itemRepo.items = []*entities.FashionItem{{}}

	if err := fixture.uc.ApplySignal(context.Background(), signalInput(userID, uuid.New())); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	stored := fixture.profileRepo.rows[userID].LearnedStyles
	if len(stored) != topLearnedStyles {
		t.Fatalf("learned size = %d, want %d (%v)", len(stored), topLearnedStyles, stored)
	}
	if _, kept := stored["style-k"]; kept {
		t.Fatalf("the weakest style must be dropped: %v", stored)
	}
}

// Idempotency: a redelivered signal is dropped before the profile is touched, so
// the decay is not applied twice (FR-037).
func TestApplySignalIsIdempotentForDuplicateSignalID(t *testing.T) {
	fixture := newLearningFixture(t, true)
	userID := fixture.seedProfile(&entities.UserStyleProfile{
		UserID:        fixture.userID,
		Enabled:       true,
		LearnedStyles: entities.StyleWeightMap{"minimalist": 1.0},
	})
	fixture.itemRepo.items = []*entities.FashionItem{styleItem("minimalist")}

	input := signalInput(userID, uuid.New())
	for attempt := 0; attempt < 2; attempt++ {
		if err := fixture.uc.ApplySignal(context.Background(), input); err != nil {
			t.Fatalf("ApplySignal attempt %d: %v", attempt, err)
		}
	}

	if fixture.signalRepo.insertCalls != 2 {
		t.Fatalf("insert calls = %d, want 2 (the duplicate must reach the DB)", fixture.signalRepo.insertCalls)
	}
	if fixture.profileRepo.updateCalls != 1 {
		t.Fatalf("profile updates = %d, want 1 (no double learning)", fixture.profileRepo.updateCalls)
	}
	// One decay (0.9) plus one observation (1.0) = 1.9, not 2.71.
	if got := fixture.profileRepo.rows[userID].LearnedStyles["minimalist"]; math.Abs(got-1.9) > 1e-9 {
		t.Fatalf("learned weight = %v, want 1.9", got)
	}
}

// An item without an embedding (still being analysed) must not wipe or decay the
// learned vector.
func TestApplySignalWithoutEmbeddingsKeepsLearnedEmbedding(t *testing.T) {
	fixture := newLearningFixture(t, true)
	userID := fixture.seedProfile(&entities.UserStyleProfile{
		UserID:           fixture.userID,
		Enabled:          true,
		LearnedStyles:    entities.StyleWeightMap{},
		LearnedEmbedding: entities.Vector{1, 0},
	})
	fixture.itemRepo.items = []*entities.FashionItem{styleItem("minimalist")}

	if err := fixture.uc.ApplySignal(context.Background(), signalInput(userID, uuid.New())); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	stored := fixture.profileRepo.rows[userID]
	if !reflect.DeepEqual(stored.LearnedEmbedding, entities.Vector{1, 0}) {
		t.Fatalf("learned embedding = %v, want it untouched", stored.LearnedEmbedding)
	}
	if len(stored.LearnedStyles) != 1 {
		t.Fatalf("learned styles = %v, want the style to be learned anyway", stored.LearnedStyles)
	}
}

// learned_embedding = normalize(decay·old + (1-decay)·mean(item embeddings)).
func TestApplySignalBlendsLearnedEmbedding(t *testing.T) {
	fixture := newLearningFixture(t, true)
	userID := fixture.seedProfile(&entities.UserStyleProfile{
		UserID:           fixture.userID,
		Enabled:          true,
		LearnedStyles:    entities.StyleWeightMap{},
		LearnedEmbedding: entities.Vector{1, 0},
	})
	fixture.itemRepo.items = []*entities.FashionItem{
		vectorItem("minimalist", 1, 0),
		vectorItem("minimalist", 0, 1),
	}

	if err := fixture.uc.ApplySignal(context.Background(), signalInput(userID, uuid.New(), uuid.New())); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	stored := fixture.profileRepo.rows[userID].LearnedEmbedding
	// 0.9·[1,0] + 0.1·[0.7071,0.7071] = [0.9707,0.0707], normalised.
	want := []float32{0.997378, 0.072651}
	if len(stored) != len(want) {
		t.Fatalf("learned embedding length = %d, want %d", len(stored), len(want))
	}
	for index := range want {
		if math.Abs(float64(stored[index]-want[index])) > 1e-4 {
			t.Fatalf("learned embedding = %v, want ~%v", stored, want)
		}
	}
	var sumSquares float64
	for _, value := range stored {
		sumSquares += float64(value) * float64(value)
	}
	if math.Abs(math.Sqrt(sumSquares)-1) > 1e-4 {
		t.Fatalf("learned embedding must be a unit vector, norm = %v", math.Sqrt(sumSquares))
	}
}

// Explicit preferences live in columns the learning path never writes; the use
// case must not even mutate them in memory (FR-038).
func TestApplySignalKeepsExplicitPreferences(t *testing.T) {
	fixture := newLearningFixture(t, true)
	userID := fixture.seedProfile(&entities.UserStyleProfile{
		UserID:         fixture.userID,
		Styles:         entities.StyleList{"elegant"},
		ExcludedStyles: entities.StyleList{"streetwear"},
		Enabled:        true,
		LearnedStyles:  entities.StyleWeightMap{},
	})
	fixture.itemRepo.items = []*entities.FashionItem{styleItem("minimalist")}

	if err := fixture.uc.ApplySignal(context.Background(), signalInput(userID, uuid.New())); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	stored := fixture.profileRepo.rows[userID]
	if !reflect.DeepEqual(stored.Styles, entities.StyleList{"elegant"}) {
		t.Fatalf("styles = %v, want the declared values", stored.Styles)
	}
	if !reflect.DeepEqual(stored.ExcludedStyles, entities.StyleList{"streetwear"}) {
		t.Fatalf("excluded_styles = %v, want the declared values", stored.ExcludedStyles)
	}
	if !stored.Enabled {
		t.Fatal("enabled must stay untouched")
	}
	if fixture.profileRepo.explicitMutated {
		t.Fatal("the mutate callback changed an explicit preference")
	}
}

// The operator kill switch acks the message without writing anything - even a
// malformed one must not be dead-lettered while learning is off.
func TestApplySignalKillSwitchIsNoOp(t *testing.T) {
	fixture := newLearningFixture(t, false)

	input := signalInput(fixture.userID, uuid.New())
	if err := fixture.uc.ApplySignal(context.Background(), input); err != nil {
		t.Fatalf("ApplySignal must succeed when disabled, got %v", err)
	}
	if err := fixture.uc.ApplySignal(context.Background(), dto.StyleSignalInput{}); err != nil {
		t.Fatalf("a malformed signal must also be acked while disabled, got %v", err)
	}
	if fixture.signalRepo.insertCalls != 0 || fixture.profileRepo.updateCalls != 0 {
		t.Fatalf("disabled learning wrote something: inserts=%d updates=%d",
			fixture.signalRepo.insertCalls, fixture.profileRepo.updateCalls)
	}
	if fixture.uow.executes != 0 {
		t.Fatalf("disabled learning opened %d transactions, want 0", fixture.uow.executes)
	}
}

// A malformed signal is rejected before touching the database.
func TestApplySignalRejectsInvalidInput(t *testing.T) {
	cases := map[string]func(userID uuid.UUID) dto.StyleSignalInput{
		"missing signal id": func(userID uuid.UUID) dto.StyleSignalInput {
			input := signalInput(userID, uuid.New())
			input.SignalID = uuid.Nil
			return input
		},
		"missing user id": func(userID uuid.UUID) dto.StyleSignalInput {
			input := signalInput(userID, uuid.New())
			input.UserID = uuid.Nil
			return input
		},
		"no fashion item": func(userID uuid.UUID) dto.StyleSignalInput {
			return signalInput(userID)
		},
		"only nil fashion items": func(userID uuid.UUID) dto.StyleSignalInput {
			return signalInput(userID, uuid.Nil)
		},
		"unknown source": func(userID uuid.UUID) dto.StyleSignalInput {
			input := signalInput(userID, uuid.New())
			input.Source = "not_a_source"
			return input
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			fixture := newLearningFixture(t, true)
			err := fixture.uc.ApplySignal(context.Background(), build(fixture.userID))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !isInvalidSignal(err) {
				t.Fatalf("error = %v, want ErrInvalidSignal", err)
			}
			if fixture.signalRepo.insertCalls != 0 || fixture.profileRepo.updateCalls != 0 {
				t.Fatalf("an invalid signal must not reach the repositories")
			}
		})
	}
}

// An empty source is accepted as "outfit_saved" because that is the only topic
// carrying a signal without an explicit source.
func TestApplySignalDefaultsMissingSource(t *testing.T) {
	fixture := newLearningFixture(t, true)
	fixture.seedProfile(&entities.UserStyleProfile{UserID: fixture.userID, Enabled: true})
	fixture.itemRepo.items = []*entities.FashionItem{styleItem("minimalist")}

	input := signalInput(fixture.userID, uuid.New())
	input.Source = ""
	if err := fixture.uc.ApplySignal(context.Background(), input); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	signal := fixture.signalRepo.signals[0]
	if signal.Source != entities.StyleSignalSourceOutfitSaved {
		t.Fatalf("source = %d, want outfit_saved", signal.Source)
	}
	if signal.OutfitID == nil || *signal.OutfitID != *input.OutfitID {
		t.Fatalf("outfit id = %v, want %v", signal.OutfitID, input.OutfitID)
	}
	if !signal.CreatedAt.Equal(input.OccurredAt) {
		t.Fatalf("created_at = %v, want the payload occurredAt %v", signal.CreatedAt, input.OccurredAt)
	}
}

// Duplicated fashion item ids in one payload are stored once; the DB CHECK
// requires a non-empty array, and a duplicate must not double the style count.
func TestApplySignalDeduplicatesFashionItemIDs(t *testing.T) {
	fixture := newLearningFixture(t, true)
	fixture.seedProfile(&entities.UserStyleProfile{UserID: fixture.userID, Enabled: true})
	fixture.itemRepo.items = []*entities.FashionItem{styleItem("minimalist")}
	itemID := uuid.New()

	if err := fixture.uc.ApplySignal(context.Background(), signalInput(fixture.userID, itemID, itemID)); err != nil {
		t.Fatalf("ApplySignal: %v", err)
	}

	if got := len(fixture.signalRepo.signals[0].FashionItemIDs); got != 1 {
		t.Fatalf("stored fashion items = %d, want 1", got)
	}
	if got := fixture.profileRepo.rows[fixture.userID].LearnedStyles["minimalist"]; math.Abs(got-1) > 1e-9 {
		t.Fatalf("learned weight = %v, want 1 (one item, one observation)", got)
	}
}

func isInvalidSignal(err error) bool {
	return errors.Is(err, ErrInvalidSignal)
}

func cloneLearningProfile(profile *entities.UserStyleProfile) *entities.UserStyleProfile {
	if profile == nil {
		return nil
	}
	clone := *profile
	if profile.Styles != nil {
		clone.Styles = append(entities.StyleList(nil), profile.Styles...)
	}
	if profile.ExcludedStyles != nil {
		clone.ExcludedStyles = append(entities.StyleList(nil), profile.ExcludedStyles...)
	}
	clone.LearnedStyles = cloneWeights(profile.LearnedStyles)
	if profile.LearnedEmbedding != nil {
		clone.LearnedEmbedding = append(entities.Vector(nil), profile.LearnedEmbedding...)
	}
	return &clone
}

func cloneWeights(weights entities.StyleWeightMap) entities.StyleWeightMap {
	if weights == nil {
		return nil
	}
	clone := make(entities.StyleWeightMap, len(weights))
	for style, weight := range weights {
		clone[style] = weight
	}
	return clone
}
