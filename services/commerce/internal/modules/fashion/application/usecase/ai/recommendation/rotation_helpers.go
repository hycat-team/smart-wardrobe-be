package recommendation

import (
	"context"
	"sort"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/ranking"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"

	"github.com/google/uuid"
)

// Rotation defaults mirror the shipped rag block so a zero config (older
// environment, unit test) still rotates instead of behaving as before 033.
const (
	defaultStaleCandidateSlots = 4
	defaultStaleEligibleDays   = 30
	defaultCooldownTolerance   = 0.15
)

// rotationSettings is the read-only snapshot of the rag rotation knobs for one
// recommendation request.
type rotationSettings struct {
	recency             ranking.RecencyWindow
	staleEligibleDays   int
	staleCandidateSlots int
	cooldownTolerance   float64
}

// rotationSettings reads the rotation knobs from config and replaces any
// missing value with the shipped default.
func (uc *OutfitRecommendationUseCase) rotationSettings() rotationSettings {
	settings := rotationSettings{}
	if uc != nil && uc.cfg != nil {
		settings.recency = ranking.RecencyWindow{
			RecentlyWornDays:        uc.cfg.RAG.RecentlyWornPenaltyDays,
			RecentlyRecommendedDays: uc.cfg.RAG.RecentlyRecommendedPenaltyDays,
			LongUnwornDays:          uc.cfg.RAG.LongUnwornBonusDays,
		}
		settings.staleEligibleDays = uc.cfg.RAG.StaleEligibleDays
		settings.staleCandidateSlots = uc.cfg.RAG.StaleCandidateSlots
		settings.cooldownTolerance = uc.cfg.RAG.CooldownScoreTolerance
	}
	settings.recency = settings.recency.Normalized()
	if settings.staleEligibleDays <= 0 {
		settings.staleEligibleDays = defaultStaleEligibleDays
	}
	if settings.staleCandidateSlots <= 0 {
		settings.staleCandidateSlots = defaultStaleCandidateSlots
	}
	if settings.cooldownTolerance <= 0 {
		settings.cooldownTolerance = defaultCooldownTolerance
	}
	return settings
}

// staleCandidateSelection is the rotation slice of the candidate pool.
type staleCandidateSelection struct {
	// Candidates fill the reserved stale slots (at most stale_candidate_slots).
	Candidates []types.CandidateForRanking
	// SlotIDs are the fashion item ids of Candidates, in staleness order.
	SlotIDs []string
	// EligibleIDs are every stale-eligible fashion item id of the user, not
	// only the reserved slots. It is the SC-004 denominator.
	EligibleIDs []string
}

// dedupeWardrobeItemsByFashionItemID merges copies of the same fashion item into
// one entry, keeping the most recent last_used_at and last_recommended_at so a
// copy that was just worn flags the whole item (033 FR-061, US3/T055/T061).
// Order is preserved; items without a fashion item id are kept as they are.
func dedupeWardrobeItemsByFashionItemID(items []*wardrobeDto.WardrobeItemContractDTO) []*wardrobeDto.WardrobeItemContractDTO {
	if len(items) <= 1 {
		return items
	}
	deduped := make([]*wardrobeDto.WardrobeItemContractDTO, 0, len(items))
	positionByFashionID := make(map[uuid.UUID]int, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.FashionItemID == nil {
			deduped = append(deduped, item)
			continue
		}
		fashionID := *item.FashionItemID
		position, seen := positionByFashionID[fashionID]
		if !seen {
			merged := *item
			positionByFashionID[fashionID] = len(deduped)
			deduped = append(deduped, &merged)
			continue
		}
		merged := deduped[position]
		merged.LastUsedAt = latestTime(merged.LastUsedAt, item.LastUsedAt)
		merged.LastRecommendedAt = latestTime(merged.LastRecommendedAt, item.LastRecommendedAt)
	}
	return deduped
}

// latestTime returns the later of two optional instants, ignoring nil.
func latestTime(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		return candidate
	}
	return current
}

// fetchStaleCandidates loads the user's stale-eligible items and reserves at most
// stale_candidate_slots of them for the pool (033 US3/T053/T058).
//
// It is fail-soft: a contract error yields no stale slots instead of blocking the
// recommendation. The contract is asked for a wide limit (the user's plan
// maximum) because the 30-day filter and the hard filters run here, in Fashion,
// where the wearing role is known.
func (uc *OutfitRecommendationUseCase) fetchStaleCandidates(
	ctx context.Context,
	userID uuid.UUID,
	maxWardrobeItems int,
	retrievalQuery types.RecommendationRetrievalQuery,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
	settings rotationSettings,
	now time.Time,
) staleCandidateSelection {
	if uc == nil || uc.wardrobeContract == nil {
		return staleCandidateSelection{}
	}

	items, err := uc.wardrobeContract.GetStaleActiveWardrobeItemsByUserID(ctx, userID, maxWardrobeItems)
	if err != nil {
		uc.warnSuggestion("stale_items_lookup_error", err)
		return staleCandidateSelection{}
	}
	if len(items) == 0 {
		return staleCandidateSelection{}
	}

	// Dedupe with the max signals of the copies before deciding what is stale,
	// so an item worn on any copy is not pulled back into the pool (FR-061).
	eligible := eligibleStaleItems(
		dedupeWardrobeItemsByFashionItemID(items),
		retrievalQuery,
		fashionItemMap,
		settings.staleEligibleDays,
		now,
	)
	if len(eligible) == 0 {
		return staleCandidateSelection{}
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		return staleCandidateBefore(eligible[i], eligible[j])
	})

	slotCount := settings.staleCandidateSlots
	if slotCount > len(eligible) {
		slotCount = len(eligible)
	}

	selection := staleCandidateSelection{
		Candidates:  make([]types.CandidateForRanking, 0, slotCount),
		SlotIDs:     make([]string, 0, slotCount),
		EligibleIDs: make([]string, 0, len(eligible)),
	}
	for _, item := range eligible {
		selection.EligibleIDs = append(selection.EligibleIDs, fashionItemIDString(item))
	}
	for index := 0; index < slotCount; index++ {
		item := eligible[index]
		// No retrieval score/rank: the stale slot is a rotation reservation, not a
		// search hit, so it must not collect the retrieval relevance bonus.
		selection.Candidates = append(selection.Candidates, types.CandidateForRanking{
			Item:            item,
			Source:          types.CandidateSourceStale,
			RetrievalSource: string(types.CandidateSourceStale),
			ItemContext:     outfititemcontext.UserWardrobe,
		})
		selection.SlotIDs = append(selection.SlotIDs, fashionItemIDString(item))
	}
	return selection
}

// eligibleStaleItems keeps the items that have not been used for longer than
// staleEligibleDays (never used counts as stale) and still pass the request's
// hard filters with a resolvable wearing role.
func eligibleStaleItems(
	items []*wardrobeDto.WardrobeItemContractDTO,
	retrievalQuery types.RecommendationRetrievalQuery,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
	staleEligibleDays int,
	now time.Time,
) []*wardrobeDto.WardrobeItemContractDTO {
	cutoff := now.AddDate(0, 0, -staleEligibleDays)
	eligible := make([]*wardrobeDto.WardrobeItemContractDTO, 0, len(items))
	for _, item := range items {
		if item == nil || item.FashionItemID == nil {
			continue
		}
		if item.LastUsedAt != nil && !item.LastUsedAt.Before(cutoff) {
			continue
		}
		if !ranking.MatchesFallbackHardFilters(item, retrievalQuery.HardFilters, fashionItemMap) {
			continue
		}
		if ranking.MatchesFallbackExcludedTerms(item, retrievalQuery.ExcludedTerms, fashionItemMap) {
			continue
		}
		role := wearingRoleForFashionItem(fashionItemOf(item, fashionItemMap))
		if role == "" || role == "other" {
			continue
		}
		eligible = append(eligible, item)
	}
	return eligible
}

// staleCandidateBefore orders stale items by
// `last_used_at ASC NULLS FIRST, last_recommended_at ASC NULLS FIRST`, with the
// wardrobe item id as a deterministic final tie-breaker.
func staleCandidateBefore(left, right *wardrobeDto.WardrobeItemContractDTO) bool {
	if before, decided := nullsFirstTimeBefore(left.LastUsedAt, right.LastUsedAt); decided {
		return before
	}
	if before, decided := nullsFirstTimeBefore(left.LastRecommendedAt, right.LastRecommendedAt); decided {
		return before
	}
	return left.ID.String() < right.ID.String()
}

// nullsFirstTimeBefore compares two optional instants with NULLS FIRST
// semantics. The second return value reports whether the comparison decided the
// order (false means "equal or both nil").
func nullsFirstTimeBefore(left, right *time.Time) (bool, bool) {
	switch {
	case left == nil && right == nil:
		return false, false
	case left == nil:
		return true, true
	case right == nil:
		return false, true
	case left.Equal(*right):
		return false, false
	default:
		return left.Before(*right), true
	}
}

// dedupeRankingCandidatesByFashionItemID collapses duplicate copies of the same
// fashion item inside the pool. A stale entry wins over any other source because
// its slot was explicitly reserved; otherwise the first occurrence wins.
func dedupeRankingCandidatesByFashionItemID(candidates []types.CandidateForRanking) []types.CandidateForRanking {
	if len(candidates) <= 1 {
		return candidates
	}
	deduped := make([]types.CandidateForRanking, 0, len(candidates))
	positionByFashionID := make(map[uuid.UUID]int, len(candidates))
	for _, candidate := range candidates {
		fashionID, ok := candidateFashionItemID(candidate)
		if !ok {
			deduped = append(deduped, candidate)
			continue
		}
		position, seen := positionByFashionID[fashionID]
		if !seen {
			positionByFashionID[fashionID] = len(deduped)
			deduped = append(deduped, candidate)
			continue
		}
		if candidate.Source == types.CandidateSourceStale && deduped[position].Source != types.CandidateSourceStale {
			deduped[position] = candidate
		}
	}
	return deduped
}

// candidateFashionItemID resolves the fashion item id of a ranking candidate.
// Brand candidates carry their fashion item id in Item.FashionItemID, so the same
// helper covers user wardrobe rows and brand mock rows.
func candidateFashionItemID(candidate types.CandidateForRanking) (uuid.UUID, bool) {
	if candidate.Item == nil || candidate.Item.FashionItemID == nil {
		return uuid.Nil, false
	}
	return *candidate.Item.FashionItemID, true
}

// fashionItemIDString returns the fashion item id of a wardrobe row as a string,
// falling back to the wardrobe row id when the fusion is missing.
func fashionItemIDString(item *wardrobeDto.WardrobeItemContractDTO) string {
	if item == nil {
		return ""
	}
	if item.FashionItemID != nil {
		return item.FashionItemID.String()
	}
	return item.ID.String()
}
