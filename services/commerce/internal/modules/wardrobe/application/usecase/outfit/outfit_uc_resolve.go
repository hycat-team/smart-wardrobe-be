package outfit

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/shared/application/constants/eventconstants"

	"github.com/google/uuid"
)

// Explicit item sources accepted by SaveOutfitItemReq.itemContext (feature 033).
const (
	itemContextUserWardrobe  = "user_wardrobe"
	itemContextBrandItem     = "brand_item"
	itemContextSystemCatalog = "system_catalog"

	// outfitSavedSignalSource is the value carried by OutfitSavedEventPayload.Source.
	outfitSavedSignalSource = "outfit_saved"
)

// outfitResolution is the outcome of resolving the requested outfit items to
// concrete sources: existing wardrobe rows that are reused, catalog rows that
// must be cloned, and the final outfit items (feature 033 D6).
type outfitResolution struct {
	items []*entities.OutfitItem
	// reuseItemIDs lists the existing wardrobe rows referenced by the outfit.
	reuseItemIDs []uuid.UUID
	// cloneItems lists the new user wardrobe rows to insert in the same
	// transaction as the outfit.
	cloneItems []*entities.WardrobeItem
	// wardrobeItemIDByFashion maps a fashion item id to the wardrobe row that
	// represents it for this outfit (reused or freshly cloned).
	wardrobeItemIDByFashion map[uuid.UUID]uuid.UUID
}

func (r *outfitResolution) allFashionItemIDs() []uuid.UUID {
	return fashionItemIDsOf(r.items)
}

func (r *outfitResolution) cloneFashionItemIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(r.cloneItems))
	for _, item := range r.cloneItems {
		if item.FashionItemID != nil {
			ids = append(ids, *item.FashionItemID)
		}
	}
	return ids
}

// touchIDsFor returns the wardrobe rows that must have last_used_at bumped.
// When only is nil every resolved row is touched (save path); otherwise only
// the fashion items present in the set are touched (update path, FR-062).
func (r *outfitResolution) touchIDsFor(only map[uuid.UUID]struct{}) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(r.wardrobeItemIDByFashion))
	for fashionID, wardrobeItemID := range r.wardrobeItemIDByFashion {
		if only != nil {
			if _, ok := only[fashionID]; !ok {
				continue
			}
		}
		ids = append(ids, wardrobeItemID)
	}
	return ids
}

func (r *outfitResolution) touchIDs() []uuid.UUID {
	return r.touchIDsFor(nil)
}

// ensureUniqueOutfitItems rejects a payload that references the same fashion
// item more than once (FR-033).
func ensureUniqueOutfitItems(items []dto.SaveOutfitItemReq) error {
	seen := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if _, exists := seen[item.FashionItemID]; exists {
			return wardrobeerrors.ErrDuplicateOutfitItem(item.FashionItemID)
		}
		seen[item.FashionItemID] = struct{}{}
	}
	return nil
}

// exceedsWardrobeLimit reports whether adding newCloneCount rows to the current
// active wardrobe count would exceed the plan limit.
func exceedsWardrobeLimit(currentCount, newCloneCount, maxItems int) bool {
	if newCloneCount <= 0 {
		return false
	}
	return currentCount+newCloneCount > maxItems
}

// fashionItemIDsOf returns the unique fashion item ids of the given outfit
// items, preserving first-seen order.
func fashionItemIDsOf(items []*entities.OutfitItem) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	seen := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if _, ok := seen[item.FashionItemID]; ok {
			continue
		}
		seen[item.FashionItemID] = struct{}{}
		ids = append(ids, item.FashionItemID)
	}
	return ids
}

// newlyAddedFashionItemIDs returns the ids present in next but not in previous,
// preserving the order of next.
func newlyAddedFashionItemIDs(previous, next []uuid.UUID) []uuid.UUID {
	if len(next) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(previous))
	for _, id := range previous {
		seen[id] = struct{}{}
	}
	added := make([]uuid.UUID, 0, len(next))
	addedSeen := make(map[uuid.UUID]struct{}, len(next))
	for _, id := range next {
		if _, ok := seen[id]; ok {
			continue
		}
		if _, ok := addedSeen[id]; ok {
			continue
		}
		addedSeen[id] = struct{}{}
		added = append(added, id)
	}
	return added
}

// pickOwnedWardrobeRow chooses the row to reuse when the user owns several
// wardrobe rows for the same fashion item. An InWardrobe row wins, otherwise a
// non-sold row, and a Sold row is only used when it is the sole option (the
// caller then rejects the outfit because a sold item cannot be worn).
func pickOwnedWardrobeRow(rows []*entities.WardrobeItem) *entities.WardrobeItem {
	var fallback *entities.WardrobeItem
	var sold *entities.WardrobeItem
	for _, row := range rows {
		if row == nil || row.FashionItemID == nil {
			continue
		}
		switch row.Status {
		case wardrobestatus.InWardrobe:
			return row
		case wardrobestatus.Sold:
			if sold == nil {
				sold = row
			}
		default:
			if fallback == nil {
				fallback = row
			}
		}
	}
	if fallback != nil {
		return fallback
	}
	return sold
}

// resolveOutfitItems applies feature 033 D6: explicit itemContext restricts the
// source, an omitted context falls back to wardrobe -> eligible brand item ->
// system catalog. Catalog items are cloned into wardrobe_items (or an existing
// owned row is reused) and outfit items always reference the fashion item with
// item_context = user_wardrobe.
func (uc *OutfitUseCase) resolveOutfitItems(
	ctx context.Context,
	userID uuid.UUID,
	outfitID uuid.UUID,
	input []dto.SaveOutfitItemReq,
	maxWardrobeItems int,
	lockedMap map[uuid.UUID]bool,
) (*outfitResolution, error) {
	fashionIDs := make([]uuid.UUID, 0, len(input))
	for _, item := range input {
		fashionIDs = append(fashionIDs, item.FashionItemID)
	}

	ownedRows, err := uc.wardrobeRepo.GetByUserIDAndFashionItemIDs(ctx, userID, fashionIDs)
	if err != nil {
		return nil, err
	}
	ownedByFashion := make(map[uuid.UUID][]*entities.WardrobeItem, len(ownedRows))
	for _, row := range ownedRows {
		if row.FashionItemID == nil {
			continue
		}
		ownedByFashion[*row.FashionItemID] = append(ownedByFashion[*row.FashionItemID], row)
	}

	catalogRows, err := uc.wardrobeRepo.GetActiveSystemCatalogItemsByIDs(ctx, fashionIDs)
	if err != nil {
		return nil, err
	}
	catalogByFashion := make(map[uuid.UUID]*entities.WardrobeItem, len(catalogRows))
	for _, row := range catalogRows {
		if row.FashionItemID == nil {
			continue
		}
		if _, exists := catalogByFashion[*row.FashionItemID]; !exists {
			catalogByFashion[*row.FashionItemID] = row
		}
	}

	resolution := &outfitResolution{
		items:                   make([]*entities.OutfitItem, 0, len(input)),
		reuseItemIDs:            make([]uuid.UUID, 0, len(input)),
		cloneItems:              make([]*entities.WardrobeItem, 0, len(input)),
		wardrobeItemIDByFashion: make(map[uuid.UUID]uuid.UUID, len(input)),
	}

	for _, req := range input {
		fashionID := req.FashionItemID
		owned := pickOwnedWardrobeRow(ownedByFashion[fashionID])

		reuseOwned := func(row *entities.WardrobeItem) error {
			if row.Status == wardrobestatus.Sold {
				return wardrobeerrors.ErrOutfitItemSold(fashionID)
			}
			if lockedMap[row.ID] {
				return wardrobeerrors.ErrItemLockedDueToLimit(maxWardrobeItems)
			}
			resolution.items = append(resolution.items, buildOutfitItem(outfitID, req, outfititemcontext.UserWardrobe))
			resolution.reuseItemIDs = append(resolution.reuseItemIDs, row.ID)
			resolution.wardrobeItemIDByFashion[fashionID] = row.ID
			return nil
		}

		cloneCatalog := func(template *entities.WardrobeItem) {
			resolution.items = append(resolution.items, buildOutfitItem(outfitID, req, outfititemcontext.UserWardrobe))
			clone := &entities.WardrobeItem{
				UserID:        userID,
				FashionItemID: template.FashionItemID,
				Price:         template.Price,
				Status:        wardrobestatus.InWardrobe,
				ItemType:      itemtype.UserItem,
			}
			clone.ID = uuid.New()
			resolution.cloneItems = append(resolution.cloneItems, clone)
			resolution.wardrobeItemIDByFashion[fashionID] = clone.ID
		}

		if req.ItemContext != nil && *req.ItemContext != "" {
			switch *req.ItemContext {
			case itemContextUserWardrobe:
				if owned == nil {
					return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(fashionID)
				}
				if err := reuseOwned(owned); err != nil {
					return nil, err
				}
			case itemContextBrandItem:
				eligible, err := uc.isBrandItemEligible(ctx, userID, fashionID)
				if err != nil {
					return nil, err
				}
				if !eligible {
					return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(fashionID)
				}
				resolution.items = append(resolution.items, buildOutfitItem(outfitID, req, outfititemcontext.BrandItem))
			case itemContextSystemCatalog:
				template, ok := catalogByFashion[fashionID]
				if !ok {
					return nil, wardrobeerrors.ErrCatalogItemUnavailable()
				}
				if owned != nil {
					if err := reuseOwned(owned); err != nil {
						return nil, err
					}
					continue
				}
				cloneCatalog(template)
			default:
				return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(fashionID)
			}
			continue
		}

		// Auto resolution: wardrobe -> eligible brand item -> system catalog.
		if owned != nil {
			if err := reuseOwned(owned); err != nil {
				return nil, err
			}
			continue
		}

		eligible, err := uc.isBrandItemEligible(ctx, userID, fashionID)
		if err != nil {
			return nil, err
		}
		if eligible {
			resolution.items = append(resolution.items, buildOutfitItem(outfitID, req, outfititemcontext.BrandItem))
			continue
		}

		template, ok := catalogByFashion[fashionID]
		if !ok {
			return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(fashionID)
		}
		cloneCatalog(template)
	}

	return resolution, nil
}

func (uc *OutfitUseCase) isBrandItemEligible(ctx context.Context, userID, fashionItemID uuid.UUID) (bool, error) {
	if uc.brandContract == nil {
		return false, nil
	}
	eligible, _, _, err := uc.brandContract.CheckBrandItemEligibility(ctx, userID, fashionItemID)
	if err != nil {
		return false, err
	}
	return eligible, nil
}

func buildOutfitItem(outfitID uuid.UUID, req dto.SaveOutfitItemReq, itemContext outfititemcontext.OutfitItemContext) *entities.OutfitItem {
	return &entities.OutfitItem{
		OutfitID:      outfitID,
		FashionItemID: req.FashionItemID,
		ItemContext:   itemContext,
		PositionX:     req.PositionX,
		PositionY:     req.PositionY,
		Scale:         req.Scale,
		LayerOrder:    req.LayerOrder,
	}
}

// publishOutfitSaved writes the wardrobe.outfit.saved signal through the outbox
// inside the caller's transaction (feature 033 D7). A publish failure aborts
// the transaction so the outfit and the learning signal stay consistent.
func (uc *OutfitUseCase) publishOutfitSaved(ctx context.Context, outfitID, userID uuid.UUID, fashionItemIDs []uuid.UUID) error {
	if uc.eventPublisher == nil {
		return nil
	}

	payload := dto.OutfitSavedEventPayload{
		SignalID:       uuid.New(),
		UserID:         userID,
		OutfitID:       outfitID,
		FashionItemIDs: fashionItemIDs,
		Source:         outfitSavedSignalSource,
		OccurredAt:     time.Now().UTC(),
	}
	if err := uc.eventPublisher.Publish(ctx, eventconstants.TopicOutfitSaved, payload); err != nil {
		return err
	}
	return nil
}

func (uc *OutfitUseCase) executeInUnitOfWork(ctx context.Context, fn func(context.Context) error) error {
	if uc.uow == nil {
		return fn(ctx)
	}
	return uc.uow.Execute(ctx, fn)
}
