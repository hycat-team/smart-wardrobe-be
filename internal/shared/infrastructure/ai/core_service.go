package ai

import (
	"context"
	"net/http"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/application/dto"

	"golang.org/x/time/rate"
)

const (
	ProviderOpenAI = "openai"
	ProviderGemini = "google" // Google Gemini API
)

type AIService struct {
	cfg     *config.Config
	cli     *http.Client
	limiter *rate.Limiter
}

func NewAIService(cfg *config.Config) ai.IAIService {
	rpm := cfg.AI.RpmLimit
	if rpm <= 0 {
		rpm = 5 // Mặc định 5 RPM
	}

	return &AIService{
		cfg: cfg,
		cli: &http.Client{
			Timeout: 15 * time.Second,
		},
		limiter: rate.NewLimiter(rate.Limit(float64(rpm)/60.0), 1),
	}
}

func (s *AIService) AnalyzeFashionImage(ctx context.Context, imageUrl string) (*dto.FashionMetadataResult, error) {
	// Hệ thống tự động xếp hàng nếu vượt quá giới hạn RPM cấu hình toàn cục
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	result, err := s.tryVisionProvider(ctx, s.cfg.AI.VisionPrimary, imageUrl)
	if err != nil {
		if strings.Contains(err.Error(), "429") && s.cfg.AI.VisionFallback.Provider != "" {
			return s.tryVisionProvider(ctx, s.cfg.AI.VisionFallback, imageUrl)
		}
		return nil, err
	}
	return result, nil
}

func (s *AIService) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	// Hệ thống tự động xếp hàng nếu vượt quá giới hạn RPM cấu hình toàn cục
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	result, err := s.tryEmbeddingProviderBatch(ctx, s.cfg.AI.EmbeddingPrimary, chunks)
	if err != nil {
		if strings.Contains(err.Error(), "429") && s.cfg.AI.EmbeddingFallback.Provider != "" {
			return s.tryEmbeddingProviderBatch(ctx, s.cfg.AI.EmbeddingFallback, chunks)
		}
		return nil, err
	}
	return result, nil
}

func (s *AIService) tryVisionProvider(ctx context.Context, provider config.APIProviderConfig, imageUrl string) (*dto.FashionMetadataResult, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIVision(ctx, provider, imageUrl)
	case ProviderGemini:
		return s.callGoogleVision(ctx, provider, imageUrl)
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
