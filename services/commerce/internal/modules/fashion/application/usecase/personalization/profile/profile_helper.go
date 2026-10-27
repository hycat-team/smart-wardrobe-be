package profile

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashioerrors "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"

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
