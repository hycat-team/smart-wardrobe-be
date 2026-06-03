package ai

import (
	"context"
	"net/http"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/application/dto"
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

func NewAIService(cfg *config.Config, logger logger.Interface) ai.IAIService {
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

func (s *AIService) AnalyzeFashionImage(ctx context.Context, imageUrl string, categories []dto.AICategoryRef) (*dto.FashionMetadataResult, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	return executeWithFallback(
		s.logger,
		"AnalyzeFashionImage",
		s.cfg.AI.VisionFallback,
		func() (*dto.FashionMetadataResult, error) {
			return s.tryVisionProvider(ctx, s.cfg.AI.VisionPrimary, imageUrl, categories)
		},
		func() (*dto.FashionMetadataResult, error) {
			return s.tryVisionProvider(ctx, s.cfg.AI.VisionFallback, imageUrl, categories)
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

func (s *AIService) tryVisionProvider(ctx context.Context, provider config.APIProviderConfig, imageUrl string, categories []dto.AICategoryRef) (*dto.FashionMetadataResult, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIVision(ctx, provider, imageUrl, categories)
	case ProviderGemini:
		return s.callGoogleVision(ctx, provider, imageUrl, categories)
	}
	return nil, errorcode.NewInternalError("Nhà cung cấp dịch vụ trí tuệ nhân tạo không được hỗ trợ.")
}

func (s *AIService) tryEmbeddingProviderBatch(ctx context.Context, provider config.APIProviderConfig, chunks []string) ([][]float32, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIEmbeddingBatch(ctx, provider, chunks)
	case ProviderGemini:
		return s.callGoogleEmbeddingBatch(ctx, provider, chunks)
	}
	return nil, errorcode.NewInternalError("Nhà cung cấp dịch vụ mã hóa không được hỗ trợ.")
}
