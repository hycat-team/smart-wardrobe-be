package ai

import (
	"context"
	"net/http"
	"time"

	"smart-wardrobe-be/config"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	ProviderOpenAI = "openai"
	ProviderGemini = "google"
)

type AIService struct {
	cfg                       *config.Config
	chatTextClient            *http.Client
	recommendationTextClient  *http.Client
	visionClient              *http.Client
	embeddingClient           *http.Client
	chatTextLimiter           *rate.Limiter
	recommendationTextLimiter *rate.Limiter
	visionLimiter             *rate.Limiter
	embeddingLimiter          *rate.Limiter
	logger                    logger.Interface
}

func NewAIService(cfg *config.Config, logger logger.Interface) app_ai.IAIService {
	return &AIService{
		cfg: cfg,
		chatTextClient: &http.Client{
			Timeout: time.Duration(cfg.AI.ChatTextTimeoutSeconds) * time.Second,
		},
		recommendationTextClient: &http.Client{
			Timeout: time.Duration(cfg.AI.RecommendationTextTimeoutSeconds) * time.Second,
		},
		visionClient: &http.Client{
			Timeout: time.Duration(cfg.AI.VisionTimeoutSeconds) * time.Second,
		},
		embeddingClient: &http.Client{
			Timeout: time.Duration(cfg.AI.EmbeddingTimeoutSeconds) * time.Second,
		},
		chatTextLimiter:           newRPMLimiter(cfg.AI.ChatTextRPMLimit, 5),
		recommendationTextLimiter: newRPMLimiter(cfg.AI.RecommendationTextRPMLimit, 5),
		visionLimiter:             newRPMLimiter(cfg.AI.VisionRPMLimit, 5),
		embeddingLimiter:          newRPMLimiter(cfg.AI.EmbeddingRPMLimit, 5),
		logger:                    logger,
	}
}

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

func (s *AIService) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if err := s.chatTextLimiter.Wait(ctx); err != nil {
		return "", err
	}

	return executeWithFallback(
		s.logger,
		"GenerateChatText",
		s.cfg.AI.ChatTextFallback,
		func() (string, error) {
			return s.tryTextProvider(ctx, s.chatTextClient, s.cfg.AI.ChatTextPrimary, systemPrompt, userPrompt)
		},
		func() (string, error) {
			return s.tryTextProvider(ctx, s.chatTextClient, s.cfg.AI.ChatTextFallback, systemPrompt, userPrompt)
		},
	)
}

func (s *AIService) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if err := s.recommendationTextLimiter.Wait(ctx); err != nil {
		return "", err
	}

	return executeWithFallback(
		s.logger,
		"GenerateRecommendationText",
		s.cfg.AI.RecommendationTextFallback,
		func() (string, error) {
			return s.tryTextProvider(ctx, s.recommendationTextClient, s.cfg.AI.RecommendationTextPrimary, systemPrompt, userPrompt)
		},
		func() (string, error) {
			return s.tryTextProvider(ctx, s.recommendationTextClient, s.cfg.AI.RecommendationTextFallback, systemPrompt, userPrompt)
		},
	)
}

func (s *AIService) GenerateChatTextStream(
	ctx context.Context,
	systemPrompt string,
	userPrompt string,
) (<-chan string, <-chan error) {
	outTextChan := make(chan string, 100)
	outErrChan := make(chan error, 1)

	if err := s.chatTextLimiter.Wait(ctx); err != nil {
		outErrChan <- err
		close(outTextChan)
		close(outErrChan)
		return outTextChan, outErrChan
	}

	go func() {
		defer close(outTextChan)
		defer close(outErrChan)

		if err := s.streamChatTextWithFallback(ctx, systemPrompt, userPrompt, outTextChan); err != nil {
			outErrChanSend(outErrChan, err)
		}
	}()

	return outTextChan, outErrChan
}

// newRPMLimiter creates a per-minute limiter with a single-request burst.
func newRPMLimiter(rpm int, fallback int) *rate.Limiter {
	if rpm <= 0 {
		rpm = fallback
	}

	return rate.NewLimiter(rate.Limit(float64(rpm)/60.0), 1)
}

func outErrChanSend(outErr chan<- error, err error) {
	select {
	case outErr <- err:
	default:
	}
}

// GenerateTextStream streams text from the primary provider and falls back only when the stream fails before producing content.
func (s *AIService) streamChatTextWithFallback(ctx context.Context, systemPrompt string, userPrompt string, outText chan<- string) error {
	primaryText, primaryErr := s.tryTextProviderStream(ctx, s.chatTextClient, s.cfg.AI.ChatTextPrimary, systemPrompt, userPrompt)
	produced, err := s.forwardProviderStream(ctx, primaryText, primaryErr, outText)
	if err == nil || ctx.Err() != nil {
		return err
	}
	if produced || s.cfg.AI.ChatTextFallback.Provider == "" {
		if !produced {
			s.logger.Error("Primary stream provider failed permanently without fallback",
				zap.String("operation", "GenerateTextStream"),
				zap.Error(err),
			)
		}
		return err
	}

	s.logger.Warn("Primary stream provider failed before first chunk, switching to fallback",
		zap.String("operation", "GenerateChatTextStream"),
		zap.Error(err),
	)

	fallbackText, fallbackErr := s.tryTextProviderStream(ctx, s.chatTextClient, s.cfg.AI.ChatTextFallback, systemPrompt, userPrompt)
	_, fallbackStreamErr := s.forwardProviderStream(ctx, fallbackText, fallbackErr, outText)
	return fallbackStreamErr
}

// forwardProviderStream forwards provider chunks to the caller and reports whether any content was produced.
func (s *AIService) forwardProviderStream(
	ctx context.Context,
	inText <-chan string,
	inErr <-chan error,
	outText chan<- string,
) (bool, error) {
	produced := false

	for {
		select {
		case <-ctx.Done():
			return produced, ctx.Err()
		case text, ok := <-inText:
			if ok {
				produced = true
				select {
				case <-ctx.Done():
					return produced, ctx.Err()
				case outText <- text:
				}
				continue
			}

			if err, ok := <-inErr; ok && err != nil {
				return produced, err
			}
			return produced, nil
		case err, ok := <-inErr:
			if ok && err != nil {
				return produced, err
			}
			if !ok {
				for text := range inText {
					produced = true
					select {
					case <-ctx.Done():
						return produced, ctx.Err()
					case outText <- text:
					}
				}
				return produced, nil
			}
		}
	}
}

func (s *AIService) tryTextProviderStream(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
) (<-chan string, <-chan error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAITextStream(ctx, client, provider, systemPrompt, userPrompt)
	case ProviderGemini:
		return s.callGoogleTextStream(ctx, client, provider, systemPrompt, userPrompt)
	}

	errChan := make(chan error, 1)
	errChan <- apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phản hồi văn bản này.")
	close(errChan)
	textChan := make(chan string)
	close(textChan)
	return textChan, errChan
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

func (s *AIService) tryTextProvider(ctx context.Context, client *http.Client, provider config.APIProviderConfig, systemPrompt string, userPrompt string) (string, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIText(ctx, client, provider, systemPrompt, userPrompt)
	case ProviderGemini:
		return s.callGoogleText(ctx, client, provider, systemPrompt, userPrompt)
	}

	return "", apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phản hồi văn bản này.")
}
