package profile

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashioerrors "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/text/unicode/norm"
)

const (
	// styleValueMaxLength is the per-value cap of contracts §1.2 and
	// data-model §7. It mirrors the binding tag on UpdateStyleProfileReq.
	styleValueMaxLength = 50
	// defaultMaxStyles is used only when personalization.max_styles is not
	// configured; the shipped config sets 4.
	defaultMaxStyles = 4
	// defaultEmbeddingTimeoutSeconds is used only when
	// personalization.embedding_timeout_seconds is not configured.
	defaultEmbeddingTimeoutSeconds = 2
	// fallbackEmbeddingDimension matches the vector(768) columns of
	// fashion.user_style_profiles and is used only when the RAG dimension is
	// not configured.
	fallbackEmbeddingDimension = 768
	// embeddingModelMaxLength matches embedding_model varchar(100).
	embeddingModelMaxLength = 100
)

// mergeStyleProfileInput resolves the partial update against the stored profile.
// An absent field keeps the stored value; an empty array clears the list. A new
// profile starts enabled (data-model §2) unless the request explicitly turns it
// off.
func mergeStyleProfileInput(
	existing *entities.UserStyleProfile,
	input dto.UpdateStyleProfileReq,
	maxStyles int,
) (styles, excludedStyles []string, enabled bool, err error) {
	styles = []string{}
	excludedStyles = []string{}
	enabled = true

	if existing != nil {
		styles = append(styles, existing.Styles...)
		excludedStyles = append(excludedStyles, existing.ExcludedStyles...)
		enabled = existing.Enabled
	}

	if input.Styles != nil {
		styles, err = normalizeStyleValues(*input.Styles, maxStyles)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if input.ExcludedStyles != nil {
		excludedStyles, err = normalizeStyleValues(*input.ExcludedStyles, maxStyles)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	return styles, excludedStyles, enabled, nil
}

// normalizeStyleValues trims and NFC-normalizes every value, drops empties,
// removes case-insensitive duplicates (keeping the first occurrence), refuses a
// value longer than styleValueMaxLength and finally enforces the configured
// per-list cap (FR-008, FR-009, FR-005).
func normalizeStyleValues(values []string, maxStyles int) ([]string, error) {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(norm.NFC.String(value))
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, duplicated := seen[key]; duplicated {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	for _, value := range normalized {
		if utf8.RuneCountInString(value) > styleValueMaxLength {
			return nil, fashioerrors.ErrStyleValueInvalid(styleValueMaxLength)
		}
	}

	if len(normalized) > maxStyles {
		return nil, fashioerrors.ErrStyleLimitExceeded(maxStyles)
	}
	return normalized, nil
}

// applyEmbedding fills the embedding columns of the profile that is about to be
// persisted (D3).
//
//   - nothing to embed: the columns stay NULL;
//   - the embedded text and model are unchanged: the stored vector is reused
//     without calling the provider;
//   - otherwise the provider is called once with a bounded timeout. On any
//     failure the columns stay NULL so the next PUT retries, while the style
//     lists are still saved (FR-011).
func (uc *StyleProfileUseCase) applyEmbedding(
	ctx context.Context,
	profile *entities.UserStyleProfile,
	existing *entities.UserStyleProfile,
	styles, excludedStyles []string,
) {
	embeddingText := shared.BuildStyleEmbeddingText(styles, excludedStyles)
	if embeddingText == "" {
		return
	}

	model := uc.embeddingModel()
	hash := shared.HashStyleProfile(styles, excludedStyles, model)
	if existing != nil && len(existing.StyleEmbedding) > 0 &&
		existing.EmbeddingHash != nil && *existing.EmbeddingHash == hash {
		profile.StyleEmbedding = existing.StyleEmbedding
		profile.EmbeddingHash = existing.EmbeddingHash
		profile.EmbeddingModel = existing.EmbeddingModel
		return
	}

	vector, ok := uc.generateStyleEmbedding(ctx, embeddingText)
	if !ok {
		return
	}

	profile.StyleEmbedding = vector
	profile.EmbeddingHash = &hash
	if model != "" {
		profile.EmbeddingModel = &model
	}
}

// generateStyleEmbedding asks the AI orchestrator for one vector and validates
// its dimension. It reports false (never an error) so the caller keeps the
// profile save fail-soft.
func (uc *StyleProfileUseCase) generateStyleEmbedding(ctx context.Context, text string) (entities.Vector, bool) {
	embedCtx, cancel := context.WithTimeout(ctx, uc.embeddingTimeout())
	defer cancel()

	vectors, err := uc.aiService.GenerateEmbeddings(embedCtx, []string{text})
	if err != nil {
		uc.logger.Warn("Style profile embedding failed",
			zap.String("reason", "provider_error"),
			zap.Error(err),
		)
		return nil, false
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		uc.logger.Warn("Style profile embedding failed",
			zap.String("reason", "empty_vector_response"),
		)
		return nil, false
	}

	expected := uc.embeddingDimension()
	if len(vectors[0]) != expected {
		uc.logger.Warn("Style profile embedding failed",
			zap.String("reason", "unexpected_dimension"),
			zap.Int("dimension", len(vectors[0])),
			zap.Int("expected_dimension", expected),
		)
		return nil, false
	}
	return entities.Vector(vectors[0]), true
}

func (uc *StyleProfileUseCase) maxStyles() int {
	if uc.cfg.Personalization.MaxStyles > 0 {
		return uc.cfg.Personalization.MaxStyles
	}
	return defaultMaxStyles
}

func (uc *StyleProfileUseCase) embeddingTimeout() time.Duration {
	seconds := uc.cfg.Personalization.EmbeddingTimeoutSeconds
	if seconds <= 0 {
		seconds = defaultEmbeddingTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

// embeddingDimension mirrors the dimension used for item and query embeddings,
// because the style vector is blended with the query vector at recommendation
// time. The fallback matches the vector(768) columns.
func (uc *StyleProfileUseCase) embeddingDimension() int {
	if dimension := uc.cfg.RAG.RecommendationEmbeddingDimension; dimension > 0 {
		return dimension
	}
	return fallbackEmbeddingDimension
}

// embeddingModel records which model produced the vector so a model change
// forces a re-embed (the hash includes it). The provider name stands in when no
// model is configured explicitly.
func (uc *StyleProfileUseCase) embeddingModel() string {
	model := strings.TrimSpace(uc.cfg.AI.EmbeddingPrimary.Model)
	if model == "" {
		model = strings.TrimSpace(uc.cfg.AI.EmbeddingPrimary.Provider)
	}
	if utf8.RuneCountInString(model) <= embeddingModelMaxLength {
		return model
	}
	return string([]rune(model)[:embeddingModelMaxLength])
}

// suggestionRatioDecimals is the rounding used by contracts §1.4
// ("ratio = itemCount / sampleSize, làm tròn 2 chữ số").
const suggestionRatioDecimals = 2

// emptyStyleSuggestionRes is the "not enough data" payload of contracts §1.4.
// Styles is an empty (non-nil) slice so the JSON carries [] instead of null.
func emptyStyleSuggestionRes() *dto.StyleSuggestionRes {
	return &dto.StyleSuggestionRes{Styles: []dto.StyleSuggestionItem{}}
}

// distinctFashionItemIDs collects the distinct fashion item references of the
// active wardrobe rows. Brand items (and rows whose fashion item is not yet
// created) carry no FashionItemID and are skipped.
//
// A wardrobe row is per owned item, so the same fashion item cannot normally
// appear twice; de-duplicating keeps sampleSize and the per-label counts
// consistent if one ever does.
func distinctFashionItemIDs(items []*wardrobe_dto.WardrobeItemContractDTO) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item == nil || item.FashionItemID == nil {
			continue
		}
		ids = append(ids, *item.FashionItemID)
	}
	return sliceutils.UniqueUUIDs(ids)
}

// buildStyleSuggestions turns the metadata of the owned items into the
// cold-start suggestion payload (contracts §1.4).
//
// Normalisation goes through the shared lexicon, so a known synonym (for
// example "Tối giản") counts as its canonical label ("minimalist") while an
// unknown value is counted under its trimmed spelling. sampleSize counts the
// items that actually carry a style: an item without one is neither counted in
// sampleSize nor in any label, so every ratio stays within [0, 1].
//
// Results are sorted by itemCount descending, then by value ascending, which
// keeps the output stable for equal counts.
func buildStyleSuggestions(items []*entities.FashionItem) *dto.StyleSuggestionRes {
	res := emptyStyleSuggestionRes()

	counts := make(map[string]int, len(items))
	for _, item := range items {
		if item == nil || item.Style == nil {
			continue
		}
		// One item carries one style column, so the normalised slice has at
		// most one element; an empty style normalises to nothing.
		labels := shared.NormalizeStyleLabels([]string{*item.Style})
		if len(labels) == 0 {
			continue
		}
		res.SampleSize++
		counts[labels[0]]++
	}

	if res.SampleSize == 0 {
		return res
	}

	res.Styles = make([]dto.StyleSuggestionItem, 0, len(counts))
	for value, count := range counts {
		res.Styles = append(res.Styles, dto.StyleSuggestionItem{
			Value:     value,
			ItemCount: count,
			Ratio:     roundRatio(float64(count) / float64(res.SampleSize)),
		})
	}

	sort.Slice(res.Styles, func(i, j int) bool {
		if res.Styles[i].ItemCount != res.Styles[j].ItemCount {
			return res.Styles[i].ItemCount > res.Styles[j].ItemCount
		}
		return res.Styles[i].Value < res.Styles[j].Value
	})
	return res
}

// roundRatio rounds ratio to suggestionRatioDecimals decimal places.
func roundRatio(ratio float64) float64 {
	factor := math.Pow(10, suggestionRatioDecimals)
	return math.Round(ratio*factor) / factor
}
