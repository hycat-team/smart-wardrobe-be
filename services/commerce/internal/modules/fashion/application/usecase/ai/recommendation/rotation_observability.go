package recommendation

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/ranking"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// recordRotationObservability emits the rotation log line of one recommendation
// and refreshes last_recommended_at best-effort (033 US3/T056, FR-054/FR-068).
//
// It never fails the request: observability and the recommended-at write are
// tolerated failures and must not block the response. It returns the completed
// stats so tests can assert the emitted counters.
func (uc *OutfitRecommendationUseCase) recordRotationObservability(
	ctx context.Context,
	userID uuid.UUID,
	recommendationID uuid.UUID,
	finalRes *dto.RecommendedOutfitRes,
	filter *candidateFilterResult,
	settings rotationSettings,
	now time.Time,
) types.RerankStats {
	if uc == nil || finalRes == nil || filter == nil {
		return types.RerankStats{}
	}

	returned := collectReturnedItems(finalRes, filter)
	stats := filter.Stats
	stats.ReturnedItemIDs = returned.fashionIDs
	stats.StaleItemIDs = intersectIDs(returned.fashionIDs, stats.StaleItemIDs)
	stats.CooldownItemIDs = cooldownItemIDs(returned, settings.recency, now)
	stats.RecentlyUsedRepeatRatio = repeatRatio(returned, settings.recency.RecentlyWornDays, now, false)
	stats.RecentlyRecommendedRepeatRatio = repeatRatio(returned, settings.recency.RecentlyRecommendedDays, now, true)
	stats.PersonalStyleMatchRatio = personalStyleMatchRatio(returned, filter)

	if uc.logger != nil {
		uc.logger.Info("Outfit recommendation rotation",
			zap.String("recommendation_id", recommendationID.String()),
			zap.String("user_id", userID.String()),
			zap.Bool("personalized", filter.Personalization.Active),
			zap.Int("candidate_count", len(filter.Candidates)),
			zap.Int("stale_slots_added", stats.StaleSlotsAdded),
			zap.Int("cooldown_substitutions", stats.CooldownSubstitutions),
			zap.Float64("recently_used_repeat_ratio", stats.RecentlyUsedRepeatRatio),
			zap.Float64("recently_recommended_repeat_ratio", stats.RecentlyRecommendedRepeatRatio),
			zap.Float64("personal_style_match_ratio", stats.PersonalStyleMatchRatio),
			zap.Strings("returned_item_ids", stats.ReturnedItemIDs),
			zap.Strings("stale_item_ids", stats.StaleItemIDs),
			zap.Strings("stale_eligible_item_ids", stats.StaleEligibleItemIDs),
			zap.Strings("cooldown_item_ids", stats.CooldownItemIDs),
		)
	}

	if len(returned.wardrobeIDs) == 0 || uc.wardrobeContract == nil {
		return stats
	}
	if err := uc.wardrobeContract.TouchLastRecommendedAt(ctx, returned.wardrobeIDs, now); err != nil {
		if uc.logger != nil {
			uc.logger.Warn("Touch last recommended at failed",
				zap.String("recommendation_id", recommendationID.String()),
				zap.String("user_id", userID.String()),
				zap.Error(err),
			)
		}
	}
	return stats
}

// returnedRotationItems is the rotation view of the items actually returned to
// the client: primary plus alternatives of every outfit group. Suggestions are
// excluded on purpose: they are catalog items, not items of the user's wardrobe.
type returnedRotationItems struct {
	// fashionIDs are unique fashion item ids in response order.
	fashionIDs []string
	// wardrobeIDs are the ids that must receive last_recommended_at. Brand mock
	// items are excluded because they are not wardrobe rows.
	wardrobeIDs []uuid.UUID
	// rows maps a returned fashion item id to its wardrobe row (nil for brand
	// items, which carry no recency signal).
	rows map[string]*wardrobeDto.WardrobeItemContractDTO
}

// collectReturnedItems maps the response groups back to pool candidates so the
// rotation log can classify exactly what the client received.
func collectReturnedItems(finalRes *dto.RecommendedOutfitRes, filter *candidateFilterResult) returnedRotationItems {
	promptByWardrobeID := make(map[uuid.UUID]types.CandidateForPrompt, len(filter.Candidates))
	for _, candidate := range filter.Candidates {
		if candidate.Item != nil {
			promptByWardrobeID[candidate.Item.ID] = candidate
		}
	}

	returned := returnedRotationItems{
		rows: make(map[string]*wardrobeDto.WardrobeItemContractDTO),
	}
	seenFashionID := map[string]bool{}
	seenWardrobeID := map[uuid.UUID]bool{}

	appendItem := func(item *dto.RecommendedItemRes) {
		if item == nil {
			return
		}
		candidate, known := promptByWardrobeID[item.ID]
		fashionID := item.ID.String()
		var row *wardrobeDto.WardrobeItemContractDTO
		if known && candidate.Item != nil && candidate.Item.FashionItemID != nil {
			fashionID = candidate.Item.FashionItemID.String()
			row = candidate.Item
		}
		if known && candidate.Item != nil {
			returned.rows[fashionID] = row
		}
		if !seenFashionID[fashionID] {
			seenFashionID[fashionID] = true
			returned.fashionIDs = append(returned.fashionIDs, fashionID)
		}
		if item.ItemContext != string(outfititemcontext.BrandItem) && !seenWardrobeID[item.ID] {
			seenWardrobeID[item.ID] = true
			returned.wardrobeIDs = append(returned.wardrobeIDs, item.ID)
		}
	}

	for _, group := range finalRes.Items {
		if group == nil {
			continue
		}
		appendItem(group.Primary)
		for _, alternative := range group.Alternatives {
			appendItem(alternative)
		}
	}
	return returned
}

// intersectIDs returns the ids of left that also appear in right, keeping left
// order. It is used to prove StaleItemIDs is a subset of the returned items.
func intersectIDs(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(right))
	for _, id := range right {
		wanted[id] = struct{}{}
	}
	intersection := make([]string, 0, len(left))
	for _, id := range left {
		if _, ok := wanted[id]; ok {
			intersection = append(intersection, id)
		}
	}
	if len(intersection) == 0 {
		return nil
	}
	return intersection
}

// cooldownItemIDs returns the returned items that sit inside the rotation
// cooldown window (033 FR-059).
func cooldownItemIDs(returned returnedRotationItems, recency ranking.RecencyWindow, now time.Time) []string {
	ids := make([]string, 0, len(returned.fashionIDs))
	for _, fashionID := range returned.fashionIDs {
		if ranking.IsInRecencyCooldown(returned.rows[fashionID], recency, now) {
			ids = append(ids, fashionID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

// repeatRatio is the share of returned items whose use (or recommendation)
// signal falls inside the given window. An empty response has ratio 0.
func repeatRatio(returned returnedRotationItems, windowDays int, now time.Time, recommend bool) float64 {
	if len(returned.fashionIDs) == 0 {
		return 0
	}
	withinWindow := 0
	for _, fashionID := range returned.fashionIDs {
		row := returned.rows[fashionID]
		if row == nil {
			continue
		}
		var signal *time.Time
		if recommend {
			signal = row.LastRecommendedAt
		} else {
			signal = row.LastUsedAt
		}
		if signal == nil {
			continue
		}
		if ranking.DaysSince(*signal, now) < windowDays {
			withinWindow++
		}
	}
	return float64(withinWindow) / float64(len(returned.fashionIDs))
}

// personalStyleMatchRatio is 1 when at least one returned item matches the
// user's declared or learned style, 0 otherwise (033 FR-068, SC-002).
func personalStyleMatchRatio(returned returnedRotationItems, filter *candidateFilterResult) float64 {
	labels := append([]string{}, filter.Personalization.ExplicitStyles...)
	labels = append(labels, filter.Personalization.LearnedStyles...)
	if len(labels) == 0 || len(returned.fashionIDs) == 0 {
		return 0
	}
	for _, fashionID := range returned.fashionIDs {
		row := returned.rows[fashionID]
		if row == nil || row.FashionItemID == nil {
			continue
		}
		if suggestionMatchesStyles(filter.FashionItemMap[*row.FashionItemID], labels) {
			return 1
		}
	}
	return 0
}
