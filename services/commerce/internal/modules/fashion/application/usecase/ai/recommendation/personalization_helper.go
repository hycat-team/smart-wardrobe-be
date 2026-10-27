package recommendation

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	personalization_shared "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"
)

// defaultLearnedStyleWeight is the score weight of a learned-style match used
// when personalization.learned_style_weight is not configured. It mirrors the
// shipped config value and plan D4.
const defaultLearnedStyleWeight = 0.10

// loadPersonalizationContext reads the plan entitlement and the user's style
// profile, then builds the soft personalization context used by the rest of the
// pipeline (033 US1).
//
// It is deliberately fail-soft: any lookup error, missing entitlement, a
// disabled profile or the global kill switch yields the zero context, so a
// recommendation still succeeds exactly as it did before 033.
func (uc *OutfitRecommendationUseCase) loadPersonalizationContext(
	ctx context.Context,
	userID uuid.UUID,
	intent dto.ParsedIntent,
) types.PersonalizationContext {
	if uc == nil || uc.cfg == nil || !uc.cfg.Personalization.Enabled ||
		uc.profileRepo == nil || uc.userSubContract == nil {
		return types.PersonalizationContext{}
	}

	overview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		uc.warnPersonalization("subscription_overview_error", err)
		return types.PersonalizationContext{}
	}
	if overview == nil || !overview.PersonalizationEnabled {
		return types.PersonalizationContext{}
	}

	profile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		uc.warnPersonalization("profile_lookup_error", err)
		return types.PersonalizationContext{}
	}

	return buildPersonalizationContext(uc.cfg.Personalization, profile, intent)
}

// buildPersonalizationContext is the pure part of T029: given the loaded profile
// and the parsed request intent it produces the context without touching any
// repository. Keeping it pure makes the request-wins and weight rules unit
// testable without a database.
//
// The request-level style target always wins: the profile never rewrites
// intent.StyleTarget, and the blend weight drops to
// style_blend_weight_with_request when the request carries a style of its own.
func buildPersonalizationContext(
	cfg config.Personalization,
	profile *entities.UserStyleProfile,
	intent dto.ParsedIntent,
) types.PersonalizationContext {
	if profile == nil || !profile.Enabled {
		return types.PersonalizationContext{}
	}

	explicitStyles := personalization_shared.NormalizeStyleLabels(profile.Styles)
	excludedStyles := personalization_shared.NormalizeStyleLabels(profile.ExcludedStyles)
	learnedStyles := orderedLearnedStyles(profile.LearnedStyles)

	explicitVector := sliceutils.NormalizeVector([]float32(profile.StyleEmbedding))
	learnedVector := sliceutils.NormalizeVector([]float32(profile.LearnedEmbedding))

	var styleVector []float32
	switch {
	case len(explicitVector) > 0 && len(learnedVector) > 0:
		styleVector = personalization_shared.BlendVectors(explicitVector, learnedVector, cfg.LearnedEmbeddingWeight)
	case len(explicitVector) > 0:
		styleVector = explicitVector
	case len(learnedVector) > 0:
		styleVector = learnedVector
	}

	hasRequestStyle := len(intent.StyleTarget) > 0
	styleWeight := cfg.StyleBlendWeight
	if hasRequestStyle {
		styleWeight = cfg.StyleBlendWeightWithRequest
	}

	learnedStyleWeight := cfg.LearnedStyleWeight
	if learnedStyleWeight <= 0 {
		learnedStyleWeight = defaultLearnedStyleWeight
	}

	active := len(explicitStyles) > 0 || len(excludedStyles) > 0 ||
		len(learnedStyles) > 0 || len(styleVector) > 0
	if !active {
		return types.PersonalizationContext{}
	}

	return types.PersonalizationContext{
		Active:             true,
		ExplicitStyles:     explicitStyles,
		ExcludedStyles:     excludedStyles,
		LearnedStyles:      learnedStyles,
		StyleVector:        styleVector,
		LearnedVector:      learnedVector,
		StyleWeight:        styleWeight,
		LearnedStyleWeight: learnedStyleWeight,
		HasRequestStyle:    hasRequestStyle,
	}
}

// mergePersonalizationIntoIntent copies the profile signals onto the intent so
// the scorer can read them without threading another parameter. StyleTarget and
// ExcludedStyles keep whatever value the request/parser produced.
func mergePersonalizationIntoIntent(
	intent dto.ParsedIntent,
	pc types.PersonalizationContext,
) dto.ParsedIntent {
	if !pc.HasStyleSignals() {
		return intent
	}
	intent.PersonalStyle = append([]string(nil), pc.ExplicitStyles...)
	intent.LearnedStyle = append([]string(nil), pc.LearnedStyles...)
	intent.AvoidStyle = append([]string(nil), pc.ExcludedStyles...)
	intent.LearnedStyleWeight = pc.LearnedStyleWeight
	return intent
}

// orderedLearnedStyles returns the learned style labels ordered by decreasing
// learned weight. Ties break by label so the prompt is deterministic. Labels are
// normalised through the shared taxonomy helper afterwards.
func orderedLearnedStyles(learned entities.StyleWeightMap) []string {
	if len(learned) == 0 {
		return nil
	}
	labels := make([]string, 0, len(learned))
	for label := range learned {
		labels = append(labels, label)
	}
	sort.Slice(labels, func(i, j int) bool {
		left, right := learned[labels[i]], learned[labels[j]]
		if left != right {
			return left > right
		}
		return labels[i] < labels[j]
	})
	return personalization_shared.NormalizeStyleLabels(labels)
}

// warnPersonalization logs a fail-soft personalization lookup failure. Internal
// user ids are logged by the caller's existing logs; only the reason is added
// here so style text never reaches the log.
func (uc *OutfitRecommendationUseCase) warnPersonalization(reason string, err error) {
	if uc.logger == nil {
		return
	}
	uc.logger.Warn("Outfit recommendation personalization skipped",
		zap.String("reason", reason),
		zap.Error(err),
	)
}
