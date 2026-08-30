package ai

import (
	"context"
	"net/http"
	"time"

	"smart-wardrobe-be/internal/config"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/tokenestimationmethod"
	"smart-wardrobe-be/internal/shared/infrastructure/ai/google"
	"smart-wardrobe-be/internal/shared/infrastructure/ai/openai"

	"go.uber.org/zap"
)

func (s *AIService) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	if err := s.visionLimiter.Wait(ctx); err != nil {
		return "", err
	}

	return executeWithFallback(
		s.logger,
		"AnalyzeImage",
		s.cfg.AI.VisionFallback,
		func() (string, error) {
			return s.tryVisionProvider(ctx, s.cfg.AI.VisionPrimary, imageUrl, prompt)
		},
		func() (string, error) {
			return s.tryVisionProvider(ctx, s.cfg.AI.VisionFallback, imageUrl, prompt)
		},
	)
}

func (s *AIService) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	if err := s.embeddingLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	return executeWithFallback(
		s.logger,
		"GenerateEmbeddings",
		s.cfg.AI.EmbeddingFallback,
		func() ([][]float32, error) {
			return s.tryEmbeddingProviderBatch(ctx, s.cfg.AI.EmbeddingPrimary, chunks)
		},
		func() ([][]float32, error) {
			return s.tryEmbeddingProviderBatch(ctx, s.cfg.AI.EmbeddingFallback, chunks)
		},
	)
}

func (s *AIService) tryVisionProvider(ctx context.Context, provider config.APIProviderConfig, imageUrl string, prompt string) (string, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return openai.CallVision(ctx, s.visionClient, provider, imageUrl, prompt)
	case ProviderGemini:
		return google.CallVision(ctx, s.visionClient, provider, imageUrl, prompt)
	}

	return "", apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ trí tuệ nhân tạo này.")
}

func (s *AIService) tryEmbeddingProviderBatch(ctx context.Context, provider config.APIProviderConfig, chunks []string) ([][]float32, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return openai.CallEmbeddingBatch(ctx, s.embeddingClient, provider, chunks)
	case ProviderGemini:
		return google.CallEmbeddingBatch(ctx, s.embeddingClient, provider, chunks)
	}

	return nil, apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phân tích dữ liệu này.")
}

func (s *AIService) EstimatePromptTokens(ctx context.Context, operation string, systemPrompt, userPrompt string) (app_ai.TokenEstimationMeta, error) {
	if s.estimator == nil {
		return app_ai.TokenEstimationMeta{
			EstimatedPromptTokens: 0,
			TokenEstimationMethod: tokenestimationmethod.Local,
		}, nil
	}

	var paid config.APIProviderConfig
	var client *http.Client
	if operation == app_ai.AIOperationOutfit {
		paid = s.cfg.AI.RecommendationTextPrimary
		client = s.recommendationTextClient
	} else {
		paid = s.cfg.AI.ChatTextPrimary
		client = s.chatTextClient
	}

	var promptTokens int64
	var countDuration time.Duration
	estimationMethod := tokenestimationmethod.Local

	isGemini := paid.Provider == ProviderGemini
	if isGemini {
		geminiReq := google.PreparedGeminiRequest{
			Model: paid.Model,
			Body: google.GeminiGenerateContentBody{
				Contents: []google.GeminiContent{
					{
						Role: "user",
						Parts: []google.GeminiPart{
							{Text: userPrompt},
						},
					},
				},
			},
		}
		if systemPrompt != "" {
			geminiReq.Body.SystemInstruction = &google.GeminiContent{
				Role: "system",
				Parts: []google.GeminiPart{
					{Text: systemPrompt},
				},
			}
		}
		promptTokens = s.estimator.EstimateFromRequest(geminiReq)

		shouldCallCountTokens := (operation == app_ai.AIOperationOutfit || operation == app_ai.AIOperationChat)
		if shouldCallCountTokens {
			const defaultMaxInputTokens = 12000
			ratio := s.cfg.AI.TokenEstimation.CountTokensThresholdRatio
			if ratio <= 0 {
				ratio = 0.70
			}
			threshold := int64(float64(defaultMaxInputTokens) * ratio)

			if promptTokens >= threshold {
				count, elapsed, err := google.CountGoogleTokensByRequest(ctx, client, s.cfg, paid, geminiReq)
				if err == nil && count > 0 {
					promptTokens = count
					countDuration = elapsed
					estimationMethod = tokenestimationmethod.ProviderCount
				} else {
					s.logger.Warn("Failed to count tokens using Gemini API, falling back to local estimation", zap.Error(err))
				}
			}
		}
	} else {
		promptTokens = s.estimator.EstimateFromText(systemPrompt, userPrompt)
	}

	var latencyMs *int64
	if estimationMethod == tokenestimationmethod.ProviderCount {
		l := countDuration.Milliseconds()
		latencyMs = &l
	}

	return app_ai.TokenEstimationMeta{
		EstimatedPromptTokens: promptTokens,
		TokenEstimationMethod: estimationMethod,
		TokenCountLatencyMs:   latencyMs,
	}, nil
}
