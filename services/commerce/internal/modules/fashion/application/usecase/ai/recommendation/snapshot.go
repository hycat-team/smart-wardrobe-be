package recommendation

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_caching "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/caching"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// recordRecommendationSnapshot stores the server-side snapshot that the
// feedback endpoint later uses to authenticate the owner and recover the item
// ids without trusting the client (033 US6/T071, plan D8, data-model §5).
//
// The write is best-effort: a cache failure is logged and never fails the
// recommendation, because the snapshot only enables feedback, it is not part of
// the response (contracts §2.1). A recommendation whose outfit carries no
// fashion item id is not stored at all, so feedback on it correctly answers
// "snapshot not found" instead of learning an empty signal.
func (uc *OutfitRecommendationUseCase) recordRecommendationSnapshot(
	ctx context.Context,
	userID uuid.UUID,
	recommendationID uuid.UUID,
	finalRes *dto.RecommendedOutfitRes,
	filter *candidateFilterResult,
	now time.Time,
) {
	if uc == nil || uc.snapshotCache == nil || finalRes == nil || recommendationID == uuid.Nil {
		return
	}

	fashionItemIDs := recommendationFashionItemIDs(finalRes)
	if len(fashionItemIDs) == 0 {
		return
	}

	snapshot := &uc_caching.RecommendationSnapshot{
		UserID:         userID,
		FashionItemIDs: fashionItemIDs,
		Styles:         recommendationStyles(filter),
		CreatedAt:      now.UTC(),
	}
	if err := uc.snapshotCache.Set(ctx, recommendationID, snapshot); err != nil && uc.logger != nil {
		uc.logger.Warn("Store recommendation snapshot failed",
			zap.String("recommendation_id", recommendationID.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
	}
}

// recommendationFashionItemIDs collects the fashion item ids of the returned
// outfit: the primary plus the alternatives of every group, deduplicated and
// without the zero value. Brand items carry no fashion item and are skipped.
func recommendationFashionItemIDs(res *dto.RecommendedOutfitRes) []uuid.UUID {
	if res == nil {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(res.Items)*2)
	seen := make(map[uuid.UUID]struct{}, len(res.Items)*2)
	appendItem := func(item *dto.RecommendedItemRes) {
		if item == nil || item.FashionItem == nil || item.FashionItem.ID == uuid.Nil {
			return
		}
		if _, ok := seen[item.FashionItem.ID]; ok {
			return
		}
		seen[item.FashionItem.ID] = struct{}{}
		ids = append(ids, item.FashionItem.ID)
	}

	for _, group := range res.Items {
		if group == nil {
			continue
		}
		appendItem(group.Primary)
		for _, alternative := range group.Alternatives {
			appendItem(alternative)
		}
	}
	return ids
}

// recommendationStyles records the style labels that shaped this
// recommendation for audit: the explicit personal styles plus the learned
// labels, deduplicated case-insensitively while keeping the strongest first.
func recommendationStyles(filter *candidateFilterResult) []string {
	if filter == nil {
		return nil
	}

	labels := make([]string, 0,
		len(filter.Personalization.ExplicitStyles)+len(filter.Personalization.LearnedStyles))
	seen := make(map[string]struct{},
		len(filter.Personalization.ExplicitStyles)+len(filter.Personalization.LearnedStyles))
	appendLabels := func(values []string) {
		for _, value := range values {
			label := strings.TrimSpace(value)
			if label == "" {
				continue
			}
			key := strings.ToLower(label)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			labels = append(labels, label)
		}
	}
	appendLabels(filter.Personalization.ExplicitStyles)
	appendLabels(filter.Personalization.LearnedStyles)
	if len(labels) == 0 {
		return nil
	}
	return labels
}
