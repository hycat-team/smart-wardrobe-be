package profile

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashioerrors "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// nopProfileLogger keeps the use case logging out of the test output. The fail-soft
// embedding path logs a warning, so a nil logger would panic there.
type nopProfileLogger struct{}

func (nopProfileLogger) Debug(string, ...zap.Field) {}
func (nopProfileLogger) Info(string, ...zap.Field)  {}
func (nopProfileLogger) Warn(string, ...zap.Field)  {}
func (nopProfileLogger) Error(string, ...zap.Field) {}
func (nopProfileLogger) Fatal(string, ...zap.Field) {}

// fakeStyleProfileRepo is an in-memory IUserStyleProfileRepository. It stores
// copies so a test asserting the persisted row cannot be satisfied by the value
// the use case still holds.
type fakeStyleProfileRepo struct {
	rows      map[uuid.UUID]*entities.UserStyleProfile
	upserts   int
	deletes   int
	getErr    error
	upsertErr error
	deleteErr error
}

func newFakeStyleProfileRepo() *fakeStyleProfileRepo {
	return &fakeStyleProfileRepo{rows: map[uuid.UUID]*entities.UserStyleProfile{}}
}

func (r *fakeStyleProfileRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*entities.UserStyleProfile, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	row, ok := r.rows[userID]
	if !ok {
		return nil, nil
	}
	return cloneStyleProfile(row), nil
}

func (r *fakeStyleProfileRepo) Upsert(_ context.Context, profile *entities.UserStyleProfile) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.upserts++
	r.rows[profile.UserID] = cloneStyleProfile(profile)
	return nil
}

func (r *fakeStyleProfileRepo) Delete(_ context.Context, userID uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deletes++
	delete(r.rows, userID)
	return nil
}

func (r *fakeStyleProfileRepo) UpdateLearnedWithLock(
	_ context.Context,
	userID uuid.UUID,
	mutate func(profile *entities.UserStyleProfile) error,
) (*entities.UserStyleProfile, error) {
	row, ok := r.rows[userID]
	if !ok {
		row = &entities.UserStyleProfile{UserID: userID, Enabled: true}
	}
	if err := mutate(row); err != nil {
		return nil, err
	}
	r.rows[userID] = cloneStyleProfile(row)
	return cloneStyleProfile(row), nil
}

func cloneStyleProfile(profile *entities.UserStyleProfile) *entities.UserStyleProfile {
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
	if profile.StyleEmbedding != nil {
		clone.StyleEmbedding = append(entities.Vector(nil), profile.StyleEmbedding...)
	}
	if profile.LearnedEmbedding != nil {
		clone.LearnedEmbedding = append(entities.Vector(nil), profile.LearnedEmbedding...)
	}
	if profile.EmbeddingHash != nil {
		hash := *profile.EmbeddingHash
		clone.EmbeddingHash = &hash
	}
	if profile.EmbeddingModel != nil {
		model := *profile.EmbeddingModel
		clone.EmbeddingModel = &model
	}
	return &clone
}

// fakeSubscriptionContract serves the plan entitlement of one test user.
type fakeSubscriptionContract struct {
	overview *sub_dto.UserSubscriptionContractDTO
	err      error
	calls    int
}

func (c *fakeSubscriptionContract) GetUserSubscriptionOverview(
	_ context.Context,
	_ uuid.UUID,
) (*sub_dto.UserSubscriptionContractDTO, error) {
	c.calls++
	return c.overview, c.err
}

func (c *fakeSubscriptionContract) GetUserSubscriptionOverviews(
	_ context.Context,
	_ []uuid.UUID,
) (map[uuid.UUID]*sub_dto.UserSubscriptionContractDTO, error) {
	return nil, c.err
}

// fakeStyleEmbeddingAI is a scripted IAIOrchestrator: the embedding call either
// fails or returns one configurable vector.
type fakeStyleEmbeddingAI struct {
	vector []float32
	err    error
	calls  int
}

func (f *fakeStyleEmbeddingAI) AnalyzeImage(context.Context, string, string) (string, error) {
	return "", nil
}

func (f *fakeStyleEmbeddingAI) GenerateEmbeddings(_ context.Context, _ []string) ([][]float32, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if f.vector == nil {
		return [][]float32{{}}, nil
	}
	return [][]float32{f.vector}, nil
}

func (f *fakeStyleEmbeddingAI) GenerateChatText(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (string, error) {
	return "", nil
}

func (f *fakeStyleEmbeddingAI) GenerateRecommendationText(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (string, error) {
	return "", nil
}

func (f *fakeStyleEmbeddingAI) GenerateChatTextStream(context.Context, string, string, aiexecutor_dto.OrchestratorOptions) (<-chan string, <-chan error) {
	text := make(chan string)
	errs := make(chan error)
	close(text)
	close(errs)
	return text, errs
}

func testProfileConfig(enabled bool, maxStyles int) *config.Config {
	return &config.Config{
		Personalization: config.Personalization{
			Enabled:                 enabled,
			MaxStyles:               maxStyles,
			EmbeddingTimeoutSeconds: 1,
		},
		RAG: config.RAG{RecommendationEmbeddingDimension: 768},
		AI: config.AIServiceConfig{
			EmbeddingPrimary: config.APIProviderConfig{Model: "test-embedding-model"},
		},
	}
}

func unitEmbedding(first float32) []float32 {
	vector := make([]float32, 768)
	if len(vector) > 0 {
		vector[0] = first
	}
	if len(vector) > 1 {
		vector[1] = 1
	}
	return vector
}

type profileHarness struct {
	uc       *StyleProfileUseCase
	repo     *fakeStyleProfileRepo
	sub      *fakeSubscriptionContract
	ai       *fakeStyleEmbeddingAI
	wardrobe *fakeWardrobeContract
	items    *fakeStyleItemRepo
}

// testMaxWardrobeItems is the plan cap the fake contract records; the use case
// must pass it through to GetActiveWardrobeItemsByUserID.
const testMaxWardrobeItems = 50

func newProfileHarness(cfg *config.Config, entitled bool) *profileHarness {
	repo := newFakeStyleProfileRepo()
	sub := &fakeSubscriptionContract{overview: &sub_dto.UserSubscriptionContractDTO{
		PersonalizationEnabled: entitled,
		MaxWardrobeItems:       testMaxWardrobeItems,
	}}
	ai := &fakeStyleEmbeddingAI{vector: unitEmbedding(1)}
	wardrobe := &fakeWardrobeContract{}
	items := &fakeStyleItemRepo{}
	return &profileHarness{
		uc: &StyleProfileUseCase{
			cfg:              cfg,
			logger:           nopProfileLogger{},
			aiService:        ai,
			profileRepo:      repo,
			userSubContract:  sub,
			wardrobeContract: wardrobe,
			fashionItemRepo:  items,
		},
		repo:     repo,
		sub:      sub,
		ai:       ai,
		wardrobe: wardrobe,
		items:    items,
	}
}

// fakeWardrobeContract answers only GetActiveWardrobeItemsByUserID; the rest of
// the contract interface is never reached by the style profile use case.
type fakeWardrobeContract struct {
	wardrobe_contract.IWardrobeContract
	rows  []*wardrobe_dto.WardrobeItemContractDTO
	err   error
	calls int
	limit int
}

func (c *fakeWardrobeContract) GetActiveWardrobeItemsByUserID(
	_ context.Context,
	_ uuid.UUID,
	limit int,
) ([]*wardrobe_dto.WardrobeItemContractDTO, error) {
	c.calls++
	c.limit = limit
	if c.err != nil {
		return nil, c.err
	}
	return c.rows, nil
}

// fakeStyleItemRepo answers only GetByIDs, which is all SuggestStyles needs.
type fakeStyleItemRepo struct {
	repositories.IFashionItemRepository
	items []*entities.FashionItem
	ids   []uuid.UUID
	err   error
	calls int
}

func (r *fakeStyleItemRepo) GetByIDs(_ context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
	r.calls++
	r.ids = append([]uuid.UUID(nil), ids...)
	if r.err != nil {
		return nil, r.err
	}
	return r.items, nil
}

func assertAppErrorStatus(t *testing.T, err error, wantStatus int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error with status %d, got nil", wantStatus)
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %T (%v)", err, err)
	}
	if appErr.Status != wantStatus {
		t.Fatalf("expected status %d, got %d (%v)", wantStatus, appErr.Status, err)
	}
}

func ptrStrings(values ...string) *[]string {
	return &values
}

func ptrBool(value bool) *bool {
	return &value
}

func TestUpdateProfileNormalizesSubmittedStyles(t *testing.T) {
	// "Tối giản" in NFC and in NFD must collapse to the same value.
	nfc := "Tối giản"
	nfd := "To\u0302\u0301i gia\u0309n"

	cases := []struct {
		name   string
		input  []string
		expect []string
	}{
		{
			name:   "trim and dedupe case-insensitively keeps first spelling",
			input:  []string{"  minimalist ", "Minimalist", "ELEGANT"},
			expect: []string{"minimalist", "ELEGANT"},
		},
		{
			name:   "unicode composition is normalized before dedupe",
			input:  []string{nfd, nfc},
			expect: []string{nfc},
		},
		{
			name:   "empty and whitespace-only values are dropped",
			input:  []string{"", "   ", "casual"},
			expect: []string{"casual"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfileHarness(testProfileConfig(true, 4), true)
			userID := uuid.New()

			res, err := h.uc.UpdateProfile(context.Background(), userID, dto.UpdateStyleProfileReq{
				Styles: ptrStrings(tc.input...),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Join(res.Styles, "|") != strings.Join(tc.expect, "|") {
				t.Fatalf("expected styles %v, got %v", tc.expect, res.Styles)
			}
			stored := h.repo.rows[userID]
			if stored == nil {
				t.Fatal("expected the profile to be persisted")
			}
			if strings.Join([]string(stored.Styles), "|") != strings.Join(tc.expect, "|") {
				t.Fatalf("expected stored styles %v, got %v", tc.expect, stored.Styles)
			}
		})
	}
}

func TestUpdateProfileRejectsListAboveMaxStyles(t *testing.T) {
	for _, field := range []string{"styles", "excludedStyles"} {
		t.Run(field, func(t *testing.T) {
			h := newProfileHarness(testProfileConfig(true, 2), true)
			input := dto.UpdateStyleProfileReq{}
			values := ptrStrings("minimalist", "elegant", "vintage")
			if field == "styles" {
				input.Styles = values
			} else {
				input.ExcludedStyles = values
			}

			_, err := h.uc.UpdateProfile(context.Background(), uuid.New(), input)

			if !errors.Is(err, fashioerrors.ErrStyleLimitExceeded(2)) {
				t.Fatalf("expected STYLE_LIMIT_EXCEEDED, got %v", err)
			}
			assertAppErrorStatus(t, err, http.StatusBadRequest)
			if h.repo.upserts != 0 {
				t.Fatalf("expected no write when the request is rejected, got %d", h.repo.upserts)
			}
		})
	}
}

func TestUpdateProfileAllowsClearingTheList(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)
	userID := uuid.New()
	h.repo.rows[userID] = &entities.UserStyleProfile{
		UserID:         userID,
		Styles:         entities.StyleList{"minimalist"},
		ExcludedStyles: entities.StyleList{"streetwear"},
		Enabled:        true,
		CreatedAt:      time.Now().Add(-time.Hour),
	}

	res, err := h.uc.UpdateProfile(context.Background(), userID, dto.UpdateStyleProfileReq{
		Styles: ptrStrings(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Styles) != 0 {
		t.Fatalf("expected an empty styles list, got %v", res.Styles)
	}
	if strings.Join(res.ExcludedStyles, "|") != "streetwear" {
		t.Fatalf("expected excluded styles to be preserved, got %v", res.ExcludedStyles)
	}
	if h.repo.upserts != 1 {
		t.Fatalf("expected the empty list to be persisted, got %d writes", h.repo.upserts)
	}
}

func TestUpdateProfileRejectsValueLongerThanFiftyRunes(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "fifty runes accepted", value: strings.Repeat("á", 50)},
		{name: "fifty one runes rejected", value: strings.Repeat("á", 51), wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfileHarness(testProfileConfig(true, 4), true)

			_, err := h.uc.UpdateProfile(context.Background(), uuid.New(), dto.UpdateStyleProfileReq{
				Styles: ptrStrings(tc.value),
			})
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, fashioerrors.ErrStyleValueInvalid(styleValueMaxLength)) {
				t.Fatalf("expected STYLE_VALUE_INVALID, got %v", err)
			}
			assertAppErrorStatus(t, err, http.StatusBadRequest)
		})
	}
}

func TestUpdateProfileKeepsListsWhenEmbeddingFails(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)
	h.ai.err = errors.New("embedding provider down")
	userID := uuid.New()

	res, err := h.uc.UpdateProfile(context.Background(), userID, dto.UpdateStyleProfileReq{
		Styles:         ptrStrings("minimalist", "elegant"),
		ExcludedStyles: ptrStrings("streetwear"),
	})
	if err != nil {
		t.Fatalf("embedding failure must stay fail-soft, got %v", err)
	}
	if res.HasEmbedding {
		t.Fatal("expected hasEmbedding=false when the provider failed")
	}
	if h.repo.upserts != 1 {
		t.Fatalf("expected the profile to be saved, got %d writes", h.repo.upserts)
	}

	stored := h.repo.rows[userID]
	if stored == nil {
		t.Fatal("expected the profile to be persisted")
	}
	if stored.StyleEmbedding != nil {
		t.Fatalf("expected style_embedding to stay NULL, got %v", stored.StyleEmbedding)
	}
	if stored.EmbeddingHash != nil {
		t.Fatalf("expected embedding_hash to stay NULL, got %q", *stored.EmbeddingHash)
	}
	if len(stored.Styles) != 2 {
		t.Fatalf("expected the declared styles to be saved, got %v", stored.Styles)
	}
}

func TestUpdateProfileRetriesEmbeddingOnNextPut(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)
	h.ai.err = errors.New("embedding provider down")
	userID := uuid.New()

	if _, err := h.uc.UpdateProfile(context.Background(), userID, dto.UpdateStyleProfileReq{
		Styles: ptrStrings("minimalist"),
	}); err != nil {
		t.Fatalf("first PUT must not fail: %v", err)
	}
	if h.repo.rows[userID].EmbeddingHash != nil {
		t.Fatal("expected no embedding hash after the failed call")
	}

	h.ai.err = nil
	res, err := h.uc.UpdateProfile(context.Background(), userID, dto.UpdateStyleProfileReq{
		Styles: ptrStrings("minimalist"),
	})
	if err != nil {
		t.Fatalf("second PUT must not fail: %v", err)
	}
	if !res.HasEmbedding {
		t.Fatal("expected the second PUT to retry the embedding")
	}
	if h.ai.calls != 2 {
		t.Fatalf("expected two provider calls (failure then retry), got %d", h.ai.calls)
	}
	stored := h.repo.rows[userID]
	if len(stored.StyleEmbedding) != 768 {
		t.Fatalf("expected a 768-dimension vector, got %d", len(stored.StyleEmbedding))
	}
	if stored.EmbeddingHash == nil || *stored.EmbeddingHash == "" {
		t.Fatal("expected embedding_hash to be written after the retry")
	}
}

func TestUpdateProfileSkipsProviderWhenHashIsUnchanged(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)
	userID := uuid.New()

	if _, err := h.uc.UpdateProfile(context.Background(), userID, dto.UpdateStyleProfileReq{
		Styles: ptrStrings("minimalist"),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.ai.calls != 1 {
		t.Fatalf("expected one provider call, got %d", h.ai.calls)
	}

	if _, err := h.uc.UpdateProfile(context.Background(), userID, dto.UpdateStyleProfileReq{
		Styles: ptrStrings(" minimalist "),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.ai.calls != 1 {
		t.Fatalf("expected the unchanged hash to skip the provider, got %d calls", h.ai.calls)
	}
}

func TestUpdateProfilePersistsDisabledOnFirstInsert(t *testing.T) {
	cases := []struct {
		name    string
		enabled *bool
		expect  bool
	}{
		{name: "defaults to enabled", enabled: nil, expect: true},
		{name: "explicit disabled is persisted", enabled: ptrBool(false), expect: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfileHarness(testProfileConfig(true, 4), true)
			userID := uuid.New()

			input := dto.UpdateStyleProfileReq{Styles: ptrStrings("minimalist")}
			input.Enabled = tc.enabled

			res, err := h.uc.UpdateProfile(context.Background(), userID, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Enabled != tc.expect {
				t.Fatalf("expected response enabled=%v, got %v", tc.expect, res.Enabled)
			}
			if got := h.repo.rows[userID].Enabled; got != tc.expect {
				t.Fatalf("expected stored enabled=%v, got %v", tc.expect, got)
			}
		})
	}
}

func TestUpdateProfileRequiresPersonalizationEntitlement(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), false)

	_, err := h.uc.UpdateProfile(context.Background(), uuid.New(), dto.UpdateStyleProfileReq{
		Styles: ptrStrings("minimalist"),
	})

	if !errors.Is(err, fashioerrors.ErrPersonalizationNotAvailable()) {
		t.Fatalf("expected PERSONALIZATION_NOT_AVAILABLE, got %v", err)
	}
	assertAppErrorStatus(t, err, http.StatusForbidden)
	if h.repo.upserts != 0 {
		t.Fatalf("expected no write without entitlement, got %d", h.repo.upserts)
	}
}

func TestUpdateProfileRefusesWhenKillSwitchIsOff(t *testing.T) {
	h := newProfileHarness(testProfileConfig(false, 4), true)

	_, err := h.uc.UpdateProfile(context.Background(), uuid.New(), dto.UpdateStyleProfileReq{
		Styles: ptrStrings("minimalist"),
	})

	if !errors.Is(err, fashioerrors.ErrPersonalizationDisabled()) {
		t.Fatalf("expected PERSONALIZATION_DISABLED, got %v", err)
	}
	assertAppErrorStatus(t, err, http.StatusForbidden)
	if h.repo.upserts != 0 {
		t.Fatalf("expected no write while the kill switch is off, got %d", h.repo.upserts)
	}
}

func TestGetProfileIsReadableWithoutEntitlement(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), false)
	userID := uuid.New()
	h.repo.rows[userID] = &entities.UserStyleProfile{
		UserID:    userID,
		Styles:    entities.StyleList{"minimalist"},
		Enabled:   true,
		UpdatedAt: time.Now(),
	}

	res, err := h.uc.GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("GET must stay readable after losing entitlement: %v", err)
	}
	if res.PersonalizationAvailable {
		t.Fatal("expected personalizationAvailable=false")
	}
	if strings.Join(res.Styles, "|") != "minimalist" {
		t.Fatalf("expected the stored styles, got %v", res.Styles)
	}
}

func TestGetProfileReturnsDefaultsForAbsentRow(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)

	res, err := h.uc.GetProfile(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Styles) != 0 || len(res.ExcludedStyles) != 0 {
		t.Fatalf("expected empty lists, got %+v", res)
	}
	if !res.Enabled || !res.PersonalizationAvailable {
		t.Fatalf("expected enabled and available defaults, got %+v", res)
	}
	if res.MaxStyles != 4 {
		t.Fatalf("expected maxStyles=4, got %d", res.MaxStyles)
	}
	if res.UpdatedAt != nil {
		t.Fatalf("expected no updatedAt for an absent row, got %v", res.UpdatedAt)
	}
	if h.repo.upserts != 0 {
		t.Fatal("GET must not create a row")
	}
}

func TestDeleteProfileChecksKillSwitchAndEntitlement(t *testing.T) {
	cases := []struct {
		name       string
		cfgEnabled bool
		entitled   bool
		wantErr    error
	}{
		{
			name:       "missing entitlement",
			cfgEnabled: true,
			entitled:   false,
			wantErr:    fashioerrors.ErrPersonalizationNotAvailable(),
		},
		{
			name:       "kill switch off",
			cfgEnabled: false,
			entitled:   true,
			wantErr:    fashioerrors.ErrPersonalizationDisabled(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfileHarness(testProfileConfig(tc.cfgEnabled, 4), tc.entitled)

			err := h.uc.DeleteProfile(context.Background(), uuid.New())

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if h.repo.deletes != 0 {
				t.Fatalf("expected no delete, got %d", h.repo.deletes)
			}
		})
	}
}

func TestDeleteProfileIsIdempotentWhenEntitled(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)
	userID := uuid.New()
	h.repo.rows[userID] = &entities.UserStyleProfile{UserID: userID, Enabled: true}

	if err := h.uc.DeleteProfile(context.Background(), userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.uc.DeleteProfile(context.Background(), userID); err != nil {
		t.Fatalf("second delete must stay idempotent: %v", err)
	}
	if h.repo.deletes != 2 {
		t.Fatalf("expected two delete calls, got %d", h.repo.deletes)
	}
}

// --- US5 / T070: cold-start style suggestions -------------------------------

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

func ptrStringValue(value string) *string {
	return &value
}

// wardrobeRow builds one active wardrobe row pointing at a fashion item.
func wardrobeRow(fashionItemID uuid.UUID) *wardrobe_dto.WardrobeItemContractDTO {
	return &wardrobe_dto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: ptrUUID(fashionItemID),
	}
}

// styledItem builds the metadata the fashion repository would return. A nil
// style models an item the AI has not labelled yet.
func styledItem(style *string) *entities.FashionItem {
	return &entities.FashionItem{Style: style}
}

func assertSuggestion(t *testing.T, got dto.StyleSuggestionItem, value string, count int, ratio float64) {
	t.Helper()
	if got.Value != value || got.ItemCount != count || got.Ratio != ratio {
		t.Fatalf("expected {%s %d %v}, got %+v", value, count, ratio, got)
	}
}

func TestSuggestStylesReturnsEmptySampleWithoutStyledItems(t *testing.T) {
	cases := []struct {
		name  string
		rows  []*wardrobe_dto.WardrobeItemContractDTO
		items []*entities.FashionItem
	}{
		{name: "no active wardrobe item"},
		{
			name:  "items whose style is missing or blank",
			rows:  []*wardrobe_dto.WardrobeItemContractDTO{wardrobeRow(uuid.New()), wardrobeRow(uuid.New())},
			items: []*entities.FashionItem{styledItem(nil), styledItem(ptrStringValue("   "))},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfileHarness(testProfileConfig(true, 4), true)
			h.wardrobe.rows = tc.rows
			h.items.items = tc.items

			res, err := h.uc.SuggestStyles(context.Background(), uuid.New())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.SampleSize != 0 {
				t.Fatalf("expected sampleSize=0, got %d", res.SampleSize)
			}
			// The envelope contract requires "styles": [] rather than null.
			if res.Styles == nil {
				t.Fatal("expected an empty non-nil styles slice")
			}
			if len(res.Styles) != 0 {
				t.Fatalf("expected no suggestion, got %v", res.Styles)
			}
			if h.wardrobe.limit != testMaxWardrobeItems {
				t.Fatalf("expected the plan limit %d, got %d", testMaxWardrobeItems, h.wardrobe.limit)
			}
			if tc.rows == nil && h.items.calls != 0 {
				t.Fatalf("expected no fashion lookup without an active item, got %d", h.items.calls)
			}
		})
	}
}

func TestSuggestStylesNormalizesLabelsAndSortsByCount(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)

	repeated := uuid.New()
	rows := []*wardrobe_dto.WardrobeItemContractDTO{
		wardrobeRow(repeated),
		wardrobeRow(uuid.New()),
		wardrobeRow(uuid.New()),
		wardrobeRow(uuid.New()),
		wardrobeRow(uuid.New()),
		// A second wardrobe row for an already counted fashion item must not
		// inflate the sample or the distinct id list.
		wardrobeRow(repeated),
	}
	h.wardrobe.rows = rows
	h.items.items = []*entities.FashionItem{
		styledItem(ptrStringValue("Tối giản")),   // known synonym -> minimalist
		styledItem(ptrStringValue("minimalist")), // canonical
		styledItem(ptrStringValue("  MINIMALIST ")),
		styledItem(ptrStringValue("casual")), // unknown passes through
		styledItem(ptrStringValue(" Elegant")),
	}

	res, err := h.uc.SuggestStyles(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.SampleSize != 5 {
		t.Fatalf("expected sampleSize=5 (only styled items count), got %d", res.SampleSize)
	}
	if len(res.Styles) != 3 {
		t.Fatalf("expected 3 distinct labels, got %v", res.Styles)
	}
	// Sorted by itemCount desc, then value asc.
	assertSuggestion(t, res.Styles[0], "minimalist", 3, 0.6)
	assertSuggestion(t, res.Styles[1], "casual", 1, 0.2)
	assertSuggestion(t, res.Styles[2], "elegant", 1, 0.2)

	// The duplicate wardrobe row must collapse to a single fashion item id.
	if len(h.items.ids) != 5 {
		t.Fatalf("expected 5 distinct fashion item ids, got %v", h.items.ids)
	}
	seen := make(map[uuid.UUID]struct{}, len(h.items.ids))
	for _, id := range h.items.ids {
		if _, duplicated := seen[id]; duplicated {
			t.Fatalf("expected distinct fashion item ids, got %v", h.items.ids)
		}
		seen[id] = struct{}{}
	}
}

func TestSuggestStylesRoundsRatioToTwoDecimals(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)
	h.wardrobe.rows = []*wardrobe_dto.WardrobeItemContractDTO{
		wardrobeRow(uuid.New()),
		wardrobeRow(uuid.New()),
		wardrobeRow(uuid.New()),
	}
	h.items.items = []*entities.FashionItem{
		styledItem(ptrStringValue("minimalist")),
		styledItem(ptrStringValue("minimalist")),
		styledItem(ptrStringValue("casual")),
	}

	res, err := h.uc.SuggestStyles(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SampleSize != 3 {
		t.Fatalf("expected sampleSize=3, got %d", res.SampleSize)
	}
	// 2/3 -> 0.67 and 1/3 -> 0.33, not the raw repeating decimals.
	assertSuggestion(t, res.Styles[0], "minimalist", 2, 0.67)
	assertSuggestion(t, res.Styles[1], "casual", 1, 0.33)
}

func TestSuggestStylesRequiresPersonalizationEntitlement(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), false)
	h.wardrobe.rows = []*wardrobe_dto.WardrobeItemContractDTO{wardrobeRow(uuid.New())}

	_, err := h.uc.SuggestStyles(context.Background(), uuid.New())

	if !errors.Is(err, fashioerrors.ErrPersonalizationNotAvailable()) {
		t.Fatalf("expected PERSONALIZATION_NOT_AVAILABLE, got %v", err)
	}
	assertAppErrorStatus(t, err, http.StatusForbidden)
	if h.wardrobe.calls != 0 {
		t.Fatalf("expected no wardrobe read without entitlement, got %d", h.wardrobe.calls)
	}
	if h.items.calls != 0 {
		t.Fatalf("expected no fashion lookup without entitlement, got %d", h.items.calls)
	}
}

func TestSuggestStylesNeverPersistsTheResult(t *testing.T) {
	h := newProfileHarness(testProfileConfig(true, 4), true)
	userID := uuid.New()
	h.wardrobe.rows = []*wardrobe_dto.WardrobeItemContractDTO{wardrobeRow(uuid.New())}
	h.items.items = []*entities.FashionItem{styledItem(ptrStringValue("streetwear"))}

	res, err := h.uc.SuggestStyles(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SampleSize != 1 || len(res.Styles) != 1 {
		t.Fatalf("expected one suggestion, got %+v", res)
	}
	// FR-046: a suggestion is never applied implicitly.
	if h.repo.upserts != 0 {
		t.Fatalf("expected no profile write, got %d", h.repo.upserts)
	}
	if _, exists := h.repo.rows[userID]; exists {
		t.Fatal("expected no profile row to be created")
	}
	if h.ai.calls != 0 {
		t.Fatalf("expected no embedding call, got %d", h.ai.calls)
	}
}

func TestSuggestStylesIsReadableWhileKillSwitchIsOff(t *testing.T) {
	// contracts §8: personalization.enabled=false blocks writes, not reads.
	h := newProfileHarness(testProfileConfig(false, 4), true)
	h.wardrobe.rows = []*wardrobe_dto.WardrobeItemContractDTO{wardrobeRow(uuid.New())}
	h.items.items = []*entities.FashionItem{styledItem(ptrStringValue("elegant"))}

	res, err := h.uc.SuggestStyles(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("the kill switch must not block suggestions: %v", err)
	}
	if res.SampleSize != 1 {
		t.Fatalf("expected a suggestion, got %+v", res)
	}
	assertSuggestion(t, res.Styles[0], "elegant", 1, 1)
}
