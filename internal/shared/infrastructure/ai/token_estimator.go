package ai

import (
	"math"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/infrastructure/ai/google"
)

// LocalTokenEstimator estimates token count from text using Unicode rune count.
// Formula: ceil(rune_count / chars_per_token * safety_multiplier)
// Only estimates text input. Multimodal tokens (image/audio/video) must be counted
// by the provider (e.g., via countTokens). For non-Gemini providers, callers extract
// text from the provider's own final request and call EstimateFromText directly.
type LocalTokenEstimator struct {
	charsPerToken         float64
	localSafetyMultiplier float64
}

// NewLocalTokenEstimator creates a new instance of LocalTokenEstimator.
func NewLocalTokenEstimator(charsPerToken, localSafetyMultiplier float64) *LocalTokenEstimator {
	return &LocalTokenEstimator{
		charsPerToken:         charsPerToken,
		localSafetyMultiplier: localSafetyMultiplier,
	}
}

// NewLocalTokenEstimatorFromConfig creates a LocalTokenEstimator using configuration parameters.
func NewLocalTokenEstimatorFromConfig(cfg *config.Config) *LocalTokenEstimator {
	chars := cfg.AI.TokenEstimation.CharsPerToken
	if chars <= 0 {
		chars = 4.0
	}
	mult := cfg.AI.TokenEstimation.LocalSafetyMultiplier
	if mult < 1.0 {
		mult = 1.25
	}
	return NewLocalTokenEstimator(chars, mult)
}

// EstimateFromText returns estimated token count for arbitrary text strings.
// Protects against potential integer overflow.
func (e *LocalTokenEstimator) EstimateFromText(texts ...string) int64 {
	var totalRunes int64
	for _, text := range texts {
		runes := int64(len([]rune(text)))
		// Protect against overflow before addition
		if runes > math.MaxInt64-totalRunes {
			totalRunes = math.MaxInt64
			break
		}
		totalRunes += runes
	}

	if totalRunes <= 0 {
		return 0
	}

	rawEstimate := (float64(totalRunes) / e.charsPerToken) * e.localSafetyMultiplier
	ceilValue := math.Ceil(rawEstimate)
	
	if ceilValue >= float64(math.MaxInt64) {
		return math.MaxInt64
	}

	return int64(ceilValue)
}

// EstimateFromRequest returns estimated token count for a google.PreparedGeminiRequest
// by calling AllInputText() to extract all relevant text fields.
func (e *LocalTokenEstimator) EstimateFromRequest(req google.PreparedGeminiRequest) int64 {
	return e.EstimateFromText(req.AllInputText())
}
