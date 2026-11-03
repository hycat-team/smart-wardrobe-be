// Package completion owns the outfit-completion use case of feature 033 (US7).
//
// Given a set of fashion items the user already owns, it validates ownership,
// derives the wearing roles the set already covers and asks the shared
// recommendation suggestion builder for catalog items that fill the rest. It
// never runs the LLM, never synthesizes an outfit, never writes anything and
// never consumes the daily outfit quota (FR-024,
// contracts/personalization-api.md §2.3).
package completion

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashioerrors "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// maxCompletionItems mirrors the 1-10 range of the request body
	// (contracts §2.3). The HTTP binding enforces it; the use case keeps the
	// bound so a direct caller cannot bypass it.
	maxCompletionItems = 10
	// activeItemLookupLimit is 0, which the wardrobe contract treats as "no
	// limit": ownership must be provable for any requested item and the wardrobe
	// size is already bounded by the subscription plan.
	activeItemLookupLimit = 0
)

// OutfitCompletionUseCase implements IOutfitCompletionUseCase.
//
// Item metadata and roles come from the fashion repository; ownership comes
// from the wardrobe contract. Both are in-process module boundaries, so no
// cross-module DB join is involved (Constitution §II).
type OutfitCompletionUseCase struct {
	logger            logger.Interface
	wardrobeContract  wardrobe_contract.IWardrobeContract
	fashionItemRepo   repositories.IFashionItemRepository
	suggestionBuilder uc_interfaces.IOutfitSuggestionBuilder
}

func NewOutfitCompletionUseCase(
	l logger.Interface,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	fashionItemRepo repositories.IFashionItemRepository,
	suggestionBuilder uc_interfaces.IOutfitSuggestionBuilder,
) uc_interfaces.IOutfitCompletionUseCase {
	return &OutfitCompletionUseCase{
		logger:            l,
		wardrobeContract:  wardrobeContract,
		fashionItemRepo:   fashionItemRepo,
		suggestionBuilder: suggestionBuilder,
	}
}

// CompleteOutfit validates the requested item set then returns the catalog
// suggestions for the roles it is missing.
//
// Order of operations:
//  1. dedupe and bound the id list (the HTTP binding already enforces 1-10);
//  2. prove that every id belongs to an active wardrobe item of this user;
//  3. resolve the wearing role of each item from its fashion category;
//  4. compute the gap roles (core structure plus outerwear when the context
//     asks for a layer);
//  5. hand the gaps and the request context to the shared suggestion builder.
//
// A complete structure yields an empty (non-nil) `suggestions` array, never an
// error (FR-025). Suggestions are not outfit items and no quota is touched.
func (uc *OutfitCompletionUseCase) CompleteOutfit(
	ctx context.Context,
	userID uuid.UUID,
	input dto.CompleteOutfitReq,
) (*dto.CompleteOutfitRes, error) {
	fashionItemIDs := sliceutils.UniqueUUIDs(input.FashionItemIDs)
	if len(fashionItemIDs) == 0 || len(fashionItemIDs) > maxCompletionItems {
		return nil, fashioerrors.ErrOutfitCompletionItemsInvalid(maxCompletionItems)
	}
	if err := uc.validateOwnership(ctx, userID, fashionItemIDs); err != nil {
		return nil, err
	}

	present, err := uc.presentRoles(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}

	context := completionContext(input)
	res := &dto.CompleteOutfitRes{Suggestions: []dto.OutfitSuggestionRes{}}
	gapRoles := completionGapRoles(present, context)
	if len(gapRoles) == 0 || uc.suggestionBuilder == nil {
		return res, nil
	}

	res.Suggestions = uc.suggestionBuilder.BuildOutfitCompletionSuggestions(ctx, userID, context, gapRoles)
	if res.Suggestions == nil {
		res.Suggestions = []dto.OutfitSuggestionRes{}
	}
	return res, nil
}

// validateOwnership proves every request id belongs to an active wardrobe item of
// the user. The contract reads active items (InWardrobe, not deleted) only, so a
// sold or soft-deleted item is refused exactly like a missing one.
func (uc *OutfitCompletionUseCase) validateOwnership(
	ctx context.Context,
	userID uuid.UUID,
	fashionItemIDs []uuid.UUID,
) error {
	if uc.wardrobeContract == nil {
		// Without the contract ownership cannot be proven. Failing closed is the
		// only safe answer, and it matches the feedback use case's treatment of a
		// missing dependency.
		return fashioerrors.ErrOutfitCompletionItemForbidden(fashionItemIDs[0])
	}

	items, err := uc.wardrobeContract.GetActiveWardrobeItemsByUserID(ctx, userID, activeItemLookupLimit)
	if err != nil {
		return err
	}

	owned := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if item == nil || item.FashionItemID == nil {
			continue
		}
		owned[*item.FashionItemID] = struct{}{}
	}
	for _, fashionItemID := range fashionItemIDs {
		if _, ok := owned[fashionItemID]; !ok {
			return fashioerrors.ErrOutfitCompletionItemForbidden(fashionItemID)
		}
	}
	return nil
}

// presentRoles maps the requested items to the wearing roles the set already
// covers. An item without category metadata contributes no role; it is still a
// valid, owned item, so the request is not refused (fail-soft).
func (uc *OutfitCompletionUseCase) presentRoles(
	ctx context.Context,
	fashionItemIDs []uuid.UUID,
) (map[string]bool, error) {
	present := map[string]bool{}
	if uc.fashionItemRepo == nil {
		return present, nil
	}

	items, err := uc.fashionItemRepo.GetByIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}

	skipped := 0
	for _, item := range items {
		if item == nil || item.Category == nil {
			skipped++
			continue
		}
		present[mapper.CategorySlugToWearingRole(item.Category.Slug)] = true
	}
	if skipped > 0 && uc.logger != nil {
		uc.logger.Warn("Outfit completion skipped items without category metadata",
			zap.String("reason", "missing_category"),
			zap.Int("skipped", skipped),
		)
	}
	return present, nil
}
