package ai

import (
	"context"
	"net/http"
	"time"

	"smart-wardrobe-be/config"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/logger"

	"golang.org/x/time/rate"
)

const (
	ProviderOpenAI = "openai"
	ProviderGemini = "google"
)

type AIService struct {
	cfg     *config.Config
	cli     *http.Client
	limiter *rate.Limiter
	logger  logger.Interface
}

func NewAIService(cfg *config.Config, logger logger.Interface) app_ai.IAIService {
	rpm := cfg.AI.RpmLimit
	if rpm <= 0 {
		rpm = 5
	}

	return &AIService{
		cfg: cfg,
		cli: &http.Client{
			Timeout: 15 * time.Second,
		},
		limiter: rate.NewLimiter(rate.Limit(float64(rpm)/60.0), 1),
		logger:  logger,
	}
}

func (s *AIService) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	if err := s.limiter.Wait(ctx); err != nil {
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

	if err := s.limiter.Wait(ctx); err != nil {
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

func (s *AIService) GenerateText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return "", err
	}

	return executeWithFallback(
		s.logger,
		"GenerateText",
		s.cfg.AI.TextFallback,
		func() (string, error) {
			return s.tryTextProvider(ctx, s.cfg.AI.TextPrimary, systemPrompt, userPrompt)
		},
		func() (string, error) {
			return s.tryTextProvider(ctx, s.cfg.AI.TextFallback, systemPrompt, userPrompt)
		},
	)
}

func (s *AIService) tryVisionProvider(ctx context.Context, provider config.APIProviderConfig, imageUrl string, prompt string) (string, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIVision(ctx, provider, imageUrl, prompt)
	case ProviderGemini:
		return s.callGoogleVision(ctx, provider, imageUrl, prompt)
	}

	return "", apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ trí tuệ nhân tạo này.")
}

func (s *AIService) tryEmbeddingProviderBatch(ctx context.Context, provider config.APIProviderConfig, chunks []string) ([][]float32, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIEmbeddingBatch(ctx, provider, chunks)
	case ProviderGemini:
		return s.callGoogleEmbeddingBatch(ctx, provider, chunks)
	}

	return nil, apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phân tích dữ liệu này.")
}

func (s *AIService) tryTextProvider(ctx context.Context, provider config.APIProviderConfig, systemPrompt string, userPrompt string) (string, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIText(ctx, provider, systemPrompt, userPrompt)
	case ProviderGemini:
		return s.callGoogleText(ctx, provider, systemPrompt, userPrompt)
	}

	return "", apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phản hồi văn bản này.")
}
