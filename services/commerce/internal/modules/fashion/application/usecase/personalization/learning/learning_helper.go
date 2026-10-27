// Package learning owns the style learning use case of feature 033 (US4): the
// worker handler that turns a wardrobe.outfit.saved event into a style signal
// and the decayed update of learned_styles/learned_embedding. The feedback
// endpoint (US6) reuses the same ApplySignal entry point, so a like, a dislike
// and a "saved" feedback share one implementation.
package learning

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

// Signal sources carried by dto.StyleSignalInput.Source. They are the public
// vocabulary of the learning pipeline: the outfit worker publishes
// "outfit_saved" and the feedback endpoint uses one of the "feedback_*" values.
const (
	SignalSourceOutfitSaved     = "outfit_saved"
	SignalSourceFeedbackLike    = "feedback_like"
	SignalSourceFeedbackDislike = "feedback_dislike"
	SignalSourceFeedbackSaved   = "feedback_saved"
)

// defaultLearningDecay is used only when personalization.learning_decay is not
// configured; the shipped config sets 0.9.
const defaultLearningDecay = 0.9

// topLearnedStyles is K of the top-K rule of design decision D7: learned
// styles beyond the ten strongest labels are dropped so the learned
// representation cannot describe the whole vocabulary at once.
const topLearnedStyles = 10

// ErrInvalidSignal marks a signal that can never be learned (missing ids, no
// fashion item, unknown source). The worker turns it into a permanent error so
// a malformed message is dead-lettered instead of retried forever.
var ErrInvalidSignal = errors.New("style signal requires a signal id, a user id and at least one fashion item id")

// signalSource maps the string vocabulary of a signal to the smallint stored in
// fashion.style_signals.source and to the weight of the decay formula
// (data-model §3, D7). A "saved" feedback is deliberately a light signal (0.25)
// because the same outfit very likely already produced an "outfit_saved" one.
type signalSource struct {
	code   int16
	weight float64
}

var signalSources = map[string]signalSource{
	SignalSourceOutfitSaved:     {code: entities.StyleSignalSourceOutfitSaved, weight: 1.0},
	SignalSourceFeedbackLike:    {code: entities.StyleSignalSourceFeedbackLike, weight: 0.5},
	SignalSourceFeedbackDislike: {code: entities.StyleSignalSourceFeedbackDislike, weight: -0.5},
	SignalSourceFeedbackSaved:   {code: entities.StyleSignalSourceFeedbackSaved, weight: 0.25},
}

// buildStyleSignal validates the input and resolves it to the row that is
// inserted with ON CONFLICT DO NOTHING. It returns the weight the signal applies
// to the profile.
//
// An empty Source is read as "outfit_saved": the only topic that carries a
// signal without an explicit source is wardrobe.outfit.saved, and the field is
// additive for older records. An unknown non-empty source is rejected because
// learning a signal whose weight is not defined would silently corrupt the
// profile.
func buildStyleSignal(input dto.StyleSignalInput) (*entities.StyleSignal, float64, error) {
	if input.SignalID == uuid.Nil || input.UserID == uuid.Nil {
		return nil, 0, ErrInvalidSignal
	}

	fashionItemIDs := make([]uuid.UUID, 0, len(input.FashionItemIDs))
	for _, id := range sliceutils.UniqueUUIDs(input.FashionItemIDs) {
		if id == uuid.Nil {
			continue
		}
		fashionItemIDs = append(fashionItemIDs, id)
	}
	if len(fashionItemIDs) == 0 {
		return nil, 0, ErrInvalidSignal
	}

	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = SignalSourceOutfitSaved
	}
	mapped, ok := signalSources[source]
	if !ok {
		return nil, 0, fmt.Errorf("%w: unknown source %q", ErrInvalidSignal, source)
	}

	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return &entities.StyleSignal{
		ID:               input.SignalID,
		UserID:           input.UserID,
		Source:           mapped.code,
		OutfitID:         input.OutfitID,
		RecommendationID: input.RecommendationID,
		FashionItemIDs:   entities.UUIDList(fashionItemIDs),
		CreatedAt:        occurredAt,
	}, mapped.weight, nil
}

// collectStyleObservations turns the metadata of the signalled items into the
// two inputs of the profile update: the normalized style counts and the item
// embeddings. Items without a style or without an embedding are skipped; an
// item still being analysed therefore neither contributes a label nor blocks the
// signal (spec Assumptions).
func collectStyleObservations(items []*entities.FashionItem) (map[string]float64, [][]float32) {
	labels := make([]string, 0, len(items))
	embeddings := make([][]float32, 0, len(items))

	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Style != nil {
			if normalized := shared.NormalizeStyleLabels([]string{*item.Style}); len(normalized) > 0 {
				labels = append(labels, normalized[0])
			}
		}
		if len(item.Embedding) > 0 {
			embeddings = append(embeddings, []float32(item.Embedding))
		}
	}

	return styleShares(labels), embeddings
}

// styleShares normalises the per-item style counts into fractions that sum to 1
// (a one-item outfit contributes 1.0 to its style, a two-style outfit 0.5 each).
// Normalising keeps the contribution of one signal bounded no matter how many
// items the outfit has.
func styleShares(labels []string) map[string]float64 {
	if len(labels) == 0 {
		return nil
	}

	counts := make(map[string]int, len(labels))
	for _, label := range labels {
		counts[label]++
	}

	total := float64(len(labels))
	shares := make(map[string]float64, len(counts))
	for label, count := range counts {
		shares[label] = float64(count) / total
	}
	return shares
}

// applyStyleSignal implements
// learned_styles = topK(decay·old + sourceWeight·normalizedStyleCounts):
//
//   - every previously learned style is multiplied by the decay, so recent
//     outfits weigh more than older ones without any style disappearing
//     abruptly (FR-039);
//   - the observed styles are incremented by the source weight; a dislike
//     carries a negative weight;
//   - negative results are clamped back to 0 and dropped, so a dislike lowers a
//     style's priority without ever excluding it (FR-038/FR-050);
//   - only the ten strongest labels survive (D7).
func applyStyleSignal(
	existing entities.StyleWeightMap,
	observations map[string]float64,
	sourceWeight, decay float64,
) entities.StyleWeightMap {
	merged := make(entities.StyleWeightMap, len(existing)+len(observations))
	for style, weight := range existing {
		merged[style] = weight * decay
	}
	for style, share := range observations {
		merged[style] += sourceWeight * share
	}
	for style, weight := range merged {
		if weight <= 0 {
			delete(merged, style)
		}
	}
	return topWeightedStyles(merged, topLearnedStyles)
}

// topWeightedStyles keeps the strongest labels and returns a non-nil copy, so
// an emptied map is still persisted as a JSON object ({}), never JSON null.
func topWeightedStyles(weights entities.StyleWeightMap, limit int) entities.StyleWeightMap {
	labels := make([]string, 0, len(weights))
	for label := range weights {
		labels = append(labels, label)
	}
	// Strongest first; ties break by label so the stored map - and therefore
	// every test and log line - is deterministic.
	sort.Slice(labels, func(i, j int) bool {
		if weights[labels[i]] != weights[labels[j]] {
			return weights[labels[i]] > weights[labels[j]]
		}
		return labels[i] < labels[j]
	})
	if limit > 0 && len(labels) > limit {
		labels = labels[:limit]
	}

	kept := make(entities.StyleWeightMap, len(labels))
	for _, label := range labels {
		kept[label] = weights[label]
	}
	return kept
}
