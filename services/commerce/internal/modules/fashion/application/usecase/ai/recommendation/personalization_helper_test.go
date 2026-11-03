package recommendation

import (
	"context"
	"math"
	"reflect"
	"testing"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// nopRecommendationLogger keeps the fail-soft warnings out of the test output
// without pulling in the file logger.
type nopRecommendationLogger struct{}

func (nopRecommendationLogger) Debug(string, ...zap.Field) {}
func (nopRecommendationLogger) Info(string, ...zap.Field)  {}
func (nopRecommendationLogger) Warn(string, ...zap.Field)  {}
func (nopRecommendationLogger) Error(string, ...zap.Field) {}
func (nopRecommendationLogger) Fatal(string, ...zap.Field) {}

// fakePersonalizationProfileRepo returns one canned profile.
type fakePersonalizationProfileRepo struct {
	profile *entities.UserStyleProfile
	err     error
}

func (r *fakePersonalizationProfileRepo) GetByUserID(context.Context, uuid.UUID) (*entities.UserStyleProfile, error) {
	return r.profile, r.err
}

func (r *fakePersonalizationProfileRepo) Upsert(context.Context, *entities.UserStyleProfile) error {
	return nil
}

func (r *fakePersonalizationProfileRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (r *fakePersonalizationProfileRepo) UpdateLearnedWithLock(
	context.Context,
	uuid.UUID,
	func(*entities.UserStyleProfile) error,
) (*entities.UserStyleProfile, error) {
	return nil, nil
}

// fakePersonalizationSubContract serves one plan overview.
type fakePersonalizationSubContract struct {
	overview *sub_dto.UserSubscriptionContractDTO
	err      error
}

func (c *fakePersonalizationSubContract) GetUserSubscriptionOverview(
	context.Context,
	uuid.UUID,
) (*sub_dto.UserSubscriptionContractDTO, error) {
	return c.overview, c.err
}

func (c *fakePersonalizationSubContract) GetUserSubscriptionOverviews(
	context.Context,
	[]uuid.UUID,
) (map[uuid.UUID]*sub_dto.UserSubscriptionContractDTO, error) {
	return nil, c.err
}

func entitledOverview() *sub_dto.UserSubscriptionContractDTO {
	return &sub_dto.UserSubscriptionContractDTO{PersonalizationEnabled: true}
}

func personalizationConfig() config.Personalization {
	return config.Personalization{
		Enabled:                     true,
		MaxStyles:                   4,
		StyleBlendWeight:            0.30,
		StyleBlendWeightWithRequest: 0.10,
		LearnedStyleWeight:          0.15,
		LearnedEmbeddingWeight:      0.20,
		EmbeddingTimeoutSeconds:     2,
	}
}

// oneHotVector returns a unit vector whose only non-zero component sits at index.
func oneHotVector(index int) []float32 {
	vector := make([]float32, 768)
	vector[index] = 1
	return vector
}

func TestLoadPersonalizationContextReturnsZeroWhenUnusable(t *testing.T) {
	enabledCfg := &config.Config{Personalization: personalizationConfig()}
	activeProfile := &entities.UserStyleProfile{
		UserID:  uuid.New(),
		Styles:  entities.StyleList{"minimalist"},
		Enabled: true,
	}

	cases := []struct {
		name     string
		cfg      *config.Config
		overview *sub_dto.UserSubscriptionContractDTO
		profile  *entities.UserStyleProfile
	}{
		{
			name:     "kill switch off",
			cfg:      &config.Config{Personalization: config.Personalization{Enabled: false}},
			overview: entitledOverview(),
			profile:  activeProfile,
		},
		{
			name:     "no entitlement",
			cfg:      enabledCfg,
			overview: &sub_dto.UserSubscriptionContractDTO{PersonalizationEnabled: false},
			profile:  activeProfile,
		},
		{
			name:     "nil overview",
			cfg:      enabledCfg,
			overview: nil,
			profile:  activeProfile,
		},
		{
			name:     "profile absent",
			cfg:      enabledCfg,
			overview: entitledOverview(),
			profile:  nil,
		},
		{
			name:     "profile disabled",
			cfg:      enabledCfg,
			overview: entitledOverview(),
			profile: &entities.UserStyleProfile{
				UserID:  uuid.New(),
				Styles:  entities.StyleList{"minimalist"},
				Enabled: false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := &OutfitRecommendationUseCase{
				cfg:             tc.cfg,
				logger:          nopRecommendationLogger{},
				profileRepo:     &fakePersonalizationProfileRepo{profile: tc.profile},
				userSubContract: &fakePersonalizationSubContract{overview: tc.overview},
			}

			got := uc.loadPersonalizationContext(context.Background(), uuid.New(), dto.ParsedIntent{})
			if !reflect.DeepEqual(got, types.PersonalizationContext{}) {
				t.Fatalf("expected the zero context, got %+v", got)
			}
		})
	}
}

func TestBuildPersonalizationContextPopulatesDeclaredLearnedAndExcluded(t *testing.T) {
	profile := &entities.UserStyleProfile{
		UserID:           uuid.New(),
		Styles:           entities.StyleList{" minimalist "},
		ExcludedStyles:   entities.StyleList{"streetwear"},
		Enabled:          true,
		StyleEmbedding:   entities.Vector(oneHotVector(0)),
		LearnedEmbedding: entities.Vector(oneHotVector(1)),
		LearnedStyles:    entities.StyleWeightMap{"vintage": 0.8, "minimalist": 0.5},
	}

	pc := buildPersonalizationContext(personalizationConfig(), profile, dto.ParsedIntent{})

	if !pc.Active {
		t.Fatal("expected an active context")
	}
	if !reflect.DeepEqual(pc.ExplicitStyles, []string{"minimalist"}) {
		t.Fatalf("expected the normalized declared styles, got %v", pc.ExplicitStyles)
	}
	if !reflect.DeepEqual(pc.ExcludedStyles, []string{"streetwear"}) {
		t.Fatalf("expected the declared excluded styles, got %v", pc.ExcludedStyles)
	}
	if !reflect.DeepEqual(pc.LearnedStyles, []string{"vintage", "minimalist"}) {
		t.Fatalf("expected learned styles ordered by weight, got %v", pc.LearnedStyles)
	}
	if math.Abs(pc.LearnedStyleWeight-0.15) > 1e-9 {
		t.Fatalf("expected learned_style_weight=0.15, got %v", pc.LearnedStyleWeight)
	}
	if math.Abs(pc.StyleWeight-0.30) > 1e-9 {
		t.Fatalf("expected style_blend_weight=0.30 without a request style, got %v", pc.StyleWeight)
	}
	if pc.HasRequestStyle {
		t.Fatal("expected hasRequestStyle=false")
	}

	if len(pc.LearnedVector) != 768 || math.Abs(float64(pc.LearnedVector[1])-1) > 1e-6 {
		t.Fatalf("expected the normalized learned vector, got %.4f/%d", pc.LearnedVector[1], len(pc.LearnedVector))
	}
	if len(pc.StyleVector) != 768 {
		t.Fatalf("expected a 768-dimension style vector, got %d", len(pc.StyleVector))
	}
	if pc.StyleVector[0] <= pc.StyleVector[1] || pc.StyleVector[1] <= 0 {
		t.Fatalf("expected the declared embedding to dominate the merge, got %.4f/%.4f", pc.StyleVector[0], pc.StyleVector[1])
	}
	if norm := vectorNorm(pc.StyleVector); math.Abs(norm-1) > 1e-5 {
		t.Fatalf("expected a unit style vector, got norm %.6f", norm)
	}
}

func TestBuildPersonalizationContextLetsTheRequestStyleWin(t *testing.T) {
	profile := &entities.UserStyleProfile{
		UserID:  uuid.New(),
		Styles:  entities.StyleList{"minimalist"},
		Enabled: true,
	}
	intent := dto.ParsedIntent{StyleTarget: []string{"streetwear"}}

	pc := buildPersonalizationContext(personalizationConfig(), profile, intent)
	if !pc.HasRequestStyle {
		t.Fatal("expected hasRequestStyle=true")
	}
	if math.Abs(pc.StyleWeight-0.10) > 1e-9 {
		t.Fatalf("expected style_blend_weight_with_request=0.10, got %v", pc.StyleWeight)
	}

	merged := mergePersonalizationIntoIntent(intent, pc)
	if !reflect.DeepEqual(merged.StyleTarget, []string{"streetwear"}) {
		t.Fatalf("the profile must not overwrite the request style, got %v", merged.StyleTarget)
	}
	if !reflect.DeepEqual(merged.PersonalStyle, []string{"minimalist"}) {
		t.Fatalf("expected the declared style on the intent, got %v", merged.PersonalStyle)
	}
	if math.Abs(merged.LearnedStyleWeight-0.15) > 1e-9 {
		t.Fatalf("expected the configured learned weight on the intent, got %v", merged.LearnedStyleWeight)
	}
}

func TestBuildPersonalizationContextDefaultsLearnedStyleWeight(t *testing.T) {
	cfg := personalizationConfig()
	cfg.LearnedStyleWeight = 0
	profile := &entities.UserStyleProfile{
		UserID:        uuid.New(),
		Enabled:       true,
		LearnedStyles: entities.StyleWeightMap{"minimalist": 0.9},
	}

	pc := buildPersonalizationContext(cfg, profile, dto.ParsedIntent{})

	if math.Abs(pc.LearnedStyleWeight-0.10) > 1e-9 {
		t.Fatalf("expected the default 0.10, got %v", pc.LearnedStyleWeight)
	}
}

func TestBuildPersonalizationContextMergesOnlyTheAvailableVectors(t *testing.T) {
	declaredOnly := buildPersonalizationContext(personalizationConfig(), &entities.UserStyleProfile{
		UserID:         uuid.New(),
		Enabled:        true,
		Styles:         entities.StyleList{"minimalist"},
		StyleEmbedding: entities.Vector(oneHotVector(0)),
	}, dto.ParsedIntent{})
	if len(declaredOnly.StyleVector) != 768 || declaredOnly.StyleVector[0] != 1 {
		t.Fatalf("expected the declared embedding when learned is absent, got %v", declaredOnly.StyleVector[:2])
	}
	if declaredOnly.LearnedVector != nil {
		t.Fatalf("expected no learned vector, got %v", declaredOnly.LearnedVector)
	}

	learnedOnly := buildPersonalizationContext(personalizationConfig(), &entities.UserStyleProfile{
		UserID:           uuid.New(),
		Enabled:          true,
		LearnedStyles:    entities.StyleWeightMap{"vintage": 0.7},
		LearnedEmbedding: entities.Vector(oneHotVector(3)),
	}, dto.ParsedIntent{})
	if len(learnedOnly.StyleVector) != 768 || learnedOnly.StyleVector[3] != 1 {
		t.Fatalf("expected the learned embedding when declared is absent, got %v", learnedOnly.StyleVector[:4])
	}
	if learnedOnly.ExplicitStyles != nil {
		t.Fatalf("expected no declared styles, got %v", learnedOnly.ExplicitStyles)
	}
}

func TestMergePersonalizationIntoIntentLeavesIntentUntouchedWithoutSignals(t *testing.T) {
	intent := dto.ParsedIntent{StyleTarget: []string{"streetwear"}}

	merged := mergePersonalizationIntoIntent(intent, types.PersonalizationContext{})

	if !reflect.DeepEqual(merged, intent) {
		t.Fatalf("expected the intent to stay unchanged, got %+v", merged)
	}
}

// vectorNorm is the L2 norm used to assert the blend produced a unit vector.
func vectorNorm(vector []float32) float64 {
	sum := 0.0
	for _, value := range vector {
		sum += float64(value) * float64(value)
	}
	return math.Sqrt(sum)
}

// assertNormalizedVector fails when got is not the L2 normalization of want.
func assertNormalizedVector(t *testing.T, got, want []float32) {
	t.Helper()
	normalized := sliceutils.NormalizeVector(want)
	if len(got) != len(normalized) {
		t.Fatalf("expected %d dimensions, got %d", len(normalized), len(got))
	}
	for i := range normalized {
		if math.Abs(float64(got[i]-normalized[i])) > 1e-6 {
			t.Fatalf("component %d differs: got %.6f want %.6f", i, got[i], normalized[i])
		}
	}
}
