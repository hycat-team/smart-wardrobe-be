package ai

import (
	"context"
	"net/http"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
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
	freeTextClient            *http.Client
	chatTextLimiter           *rate.Limiter
	recommendationTextLimiter *rate.Limiter
	visionLimiter             *rate.Limiter
	embeddingLimiter          *rate.Limiter
	freeTextLimiter           *rate.Limiter
	costPolicy                contract.IAICostPolicyContract
	logger                    logger.Interface
}

func NewAIService(cfg *config.Config, logger logger.Interface, costPolicy contract.IAICostPolicyContract) app_ai.IAIService {
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
		freeTextClient:            &http.Client{Timeout: time.Duration(cfg.AI.FreeTextTimeoutSeconds) * time.Second},
		chatTextLimiter:           newRPMLimiter(cfg.AI.ChatTextRPMLimit, 5),
		recommendationTextLimiter: newRPMLimiter(cfg.AI.RecommendationTextRPMLimit, 5),
		visionLimiter:             newRPMLimiter(cfg.AI.VisionRPMLimit, 5),
		embeddingLimiter:          newRPMLimiter(cfg.AI.EmbeddingRPMLimit, 5),
		freeTextLimiter:           newRPMLimiter(cfg.AI.FreeTextRPMLimit, 5),
		costPolicy:                costPolicy,
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

func (s *AIService) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	s.logTextGeneration("GenerateChatText", systemPrompt, userPrompt, options)
	provider, client, limiter, prepared, err := s.prepareText(ctx, s.cfg.AI.ChatTextPrimary, systemPrompt, userPrompt, options, contract.AIOperationChat)
	if err != nil {
		return "", err
	}
	if err = limiter.Wait(ctx); err != nil {
		_ = s.costPolicy.Release(context.WithoutCancel(ctx), prepared.RequestID, "rate_limiter_wait_failed")
		return "", err
	}
	_ = s.costPolicy.MarkInFlight(ctx, prepared.RequestID)
	text, err := s.tryTextProvider(ctx, client, provider, systemPrompt, userPrompt, prepared)
	if err != nil {
		if s.cfg.AI.ChatTextFallback.Provider != "" {
			s.logger.Warn("Chat primary failed, attempting chat fallback", zap.Error(err))
			text, err = s.tryTextProvider(ctx, client, s.cfg.AI.ChatTextFallback, systemPrompt, userPrompt, prepared)
		}
		if err != nil {
			s.finalizeProviderError(context.WithoutCancel(ctx), prepared.RequestID, err)
			if !s.isFreeProvider(provider) {
				return s.tryFreeText(ctx, systemPrompt, userPrompt, prepared)
			}
			return "", err
		}
	}
	return text, nil
}

func (s *AIService) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	s.logTextGeneration("GenerateRecommendationText", systemPrompt, userPrompt, options)
	provider, client, limiter, prepared, err := s.prepareText(ctx, s.cfg.AI.RecommendationTextPrimary, systemPrompt, userPrompt, options, contract.AIOperationOutfit)
	if err != nil {
		return "", err
	}
	if err = limiter.Wait(ctx); err != nil {
		_ = s.costPolicy.Release(context.WithoutCancel(ctx), prepared.RequestID, "rate_limiter_wait_failed")
		return "", err
	}
	_ = s.costPolicy.MarkInFlight(ctx, prepared.RequestID)
	text, err := s.tryTextProvider(ctx, client, provider, systemPrompt, userPrompt, prepared)
	if err != nil {
		if s.cfg.AI.RecommendationTextFallback.Provider != "" {
			s.logger.Warn("Recommendation primary failed, attempting recommendation fallback", zap.Error(err))
			text, err = s.tryTextProvider(ctx, client, s.cfg.AI.RecommendationTextFallback, systemPrompt, userPrompt, prepared)
		}
		if err != nil {
			s.finalizeProviderError(context.WithoutCancel(ctx), prepared.RequestID, err)
			if !s.isFreeProvider(provider) {
				return s.tryFreeText(ctx, systemPrompt, userPrompt, prepared)
			}
			return "", err
		}
	}
	return text, nil
}

func (s *AIService) GenerateChatTextStream(
	ctx context.Context,
	systemPrompt string,
	userPrompt string,
	options app_ai.TextGenerationOptions,
) (<-chan string, <-chan error) {
	outTextChan := make(chan string, 100)
	outErrChan := make(chan error, 1)

	provider, client, limiter, prepared, err := s.prepareText(ctx, s.cfg.AI.ChatTextPrimary, systemPrompt, userPrompt, options, contract.AIOperationChat)
	if err != nil {
		outErrChan <- err
		close(outTextChan)
		close(outErrChan)
		return outTextChan, outErrChan
	}
	s.logTextGeneration("GenerateChatTextStream", systemPrompt, userPrompt, prepared)
	if err := limiter.Wait(ctx); err != nil {
		_ = s.costPolicy.Release(context.WithoutCancel(ctx), prepared.RequestID, "rate_limiter_wait_failed")
		outErrChan <- err
		close(outTextChan)
		close(outErrChan)
		return outTextChan, outErrChan
	}

	go func() {
		defer close(outTextChan)
		defer close(outErrChan)

		_ = s.costPolicy.MarkInFlight(ctx, prepared.RequestID)
		produced, err := s.streamTextSingle(ctx, client, provider, systemPrompt, userPrompt, prepared, outTextChan)
		if err != nil {
			if !produced && s.cfg.AI.ChatTextFallback.Provider != "" {
				s.logger.Warn("Chat primary stream failed before producing content, attempting chat fallback", zap.Error(err))
				produced, err = s.streamTextSingle(ctx, client, s.cfg.AI.ChatTextFallback, systemPrompt, userPrompt, prepared, outTextChan)
			}
			if err != nil {
				s.finalizeProviderError(context.WithoutCancel(ctx), prepared.RequestID, err)
				if !produced && !s.isFreeProvider(provider) {
					if fallback, prepErr := s.prepareFreeOptions(ctx, systemPrompt, userPrompt, prepared); prepErr == nil {
						if waitErr := s.freeTextLimiter.Wait(ctx); waitErr == nil {
							_ = s.costPolicy.MarkInFlight(ctx, fallback.RequestID)
							_, fallbackErr := s.streamTextSingle(ctx, s.freeTextClient, s.cfg.AI.FreeTextPrimary, systemPrompt, userPrompt, fallback, outTextChan)
							if fallbackErr == nil {
								return
							}
							s.finalizeProviderError(context.WithoutCancel(ctx), fallback.RequestID, fallbackErr)
						}
					}
				}
				if !produced && ctx.Err() == nil {
					local := "Xin lỗi, stylist AI đang tạm thời bận. Bạn hãy thử lại sau ít phút nhé."
					outTextChan <- local
					return
				}
				outErrChanSend(outErrChan, err)
			}
		}
	}()

	return outTextChan, outErrChan
}

func (s *AIService) finalizeProviderError(ctx context.Context, id uuid.UUID, err error) {
	s.logger.Error("AI provider stream error", zap.String("request_id", id.String()), zap.Error(err))
	text := strings.ToLower(err.Error())
	definite := strings.Contains(text, "http 400") || strings.Contains(text, "http 401") || strings.Contains(text, "http 403") || strings.Contains(text, "http 404") || strings.Contains(text, "http 429") || strings.Contains(text, "chưa hỗ trợ") || strings.Contains(text, "not configured")
	if definite {
		_ = s.costPolicy.Release(ctx, id, "provider_rejected")
		return
	}
	_ = s.costPolicy.MarkUnknown(ctx, id, "provider_error")
}

func (s *AIService) prepareText(ctx context.Context, paid config.APIProviderConfig, systemPrompt, userPrompt string, options app_ai.TextGenerationOptions, defaultOperation string) (config.APIProviderConfig, *http.Client, *rate.Limiter, app_ai.TextGenerationOptions, error) {
	operation := options.Operation
	if operation == "" {
		operation = defaultOperation
	}
	if options.UserID == uuid.Nil {
		return paid, s.chatTextClient, s.chatTextLimiter, options, nil
	}
	promptTokens := int64(len([]byte(systemPrompt)) + len([]byte(userPrompt)))
	if paid.Provider == ProviderGemini {
		if count, err := s.countGoogleTokens(ctx, paid, systemPrompt, userPrompt); err == nil && count > 0 {
			promptTokens = count
		}
	}
	decision, err := s.costPolicy.Prepare(ctx, options.UserID, operation, promptTokens)
	if err != nil {
		return paid, nil, nil, options, err
	}
	options.RequestID = decision.RequestID
	if decision.MaxOutputTokens > 0 {
		options.MaxOutputTokens = decision.MaxOutputTokens
	}
	if decision.MaxInputTokens > 0 && promptTokens > int64(decision.MaxInputTokens) {
		_ = s.costPolicy.Release(ctx, decision.RequestID, "input_token_budget_exceeded")
		return paid, nil, nil, options, apperror.NewInternalError("AI input exceeds policy token budget")
	}
	if decision.Route == contract.AIRouteFree {
		return s.cfg.AI.FreeTextPrimary, s.freeTextClient, s.freeTextLimiter, options, nil
	}
	if decision.Route == contract.AIRouteLocal {
		return paid, nil, nil, options, apperror.NewInternalError("AI local fallback requested")
	}
	if defaultOperation == contract.AIOperationOutfit {
		return paid, s.recommendationTextClient, s.recommendationTextLimiter, options, nil
	}
	return paid, s.chatTextClient, s.chatTextLimiter, options, nil
}

func (s *AIService) logTextGeneration(operation, systemPrompt, userPrompt string, options app_ai.TextGenerationOptions) {
	s.logger.Info("AI text generation request",
		zap.String("operation", operation),
		zap.Int("input_characters", len([]rune(systemPrompt))+len([]rune(userPrompt))),
		zap.Int("max_output_tokens", options.MaxOutputTokens),
	)
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

func (s *AIService) streamTextSingle(ctx context.Context, client *http.Client, provider config.APIProviderConfig, systemPrompt, userPrompt string, options app_ai.TextGenerationOptions, outText chan<- string) (bool, error) {
	text, errs := s.tryTextProviderStream(ctx, client, provider, systemPrompt, userPrompt, options)
	return s.forwardProviderStream(ctx, text, errs, outText)
}

func (s *AIService) isFreeProvider(provider config.APIProviderConfig) bool {
	return provider.Provider == s.cfg.AI.FreeTextPrimary.Provider && provider.Model == s.cfg.AI.FreeTextPrimary.Model && provider.Endpoint == s.cfg.AI.FreeTextPrimary.Endpoint
}
func (s *AIService) tryFreeText(ctx context.Context, systemPrompt, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	if s.cfg.AI.FreeTextPrimary.Provider == "" {
		return "", apperror.NewInternalError("free AI provider is not configured")
	}
	prepared, err := s.prepareFreeOptions(ctx, systemPrompt, userPrompt, options)
	if err != nil {
		return "", err
	}
	if err := s.freeTextLimiter.Wait(ctx); err != nil {
		_ = s.costPolicy.Release(context.WithoutCancel(ctx), prepared.RequestID, "rate_limiter_wait_failed")
		return "", err
	}
	_ = s.costPolicy.MarkInFlight(ctx, prepared.RequestID)
	text, err := s.tryTextProvider(ctx, s.freeTextClient, s.cfg.AI.FreeTextPrimary, systemPrompt, userPrompt, prepared)
	if err != nil {
		s.finalizeProviderError(context.WithoutCancel(ctx), prepared.RequestID, err)
	}
	return text, err
}

func (s *AIService) prepareFreeOptions(ctx context.Context, systemPrompt, userPrompt string, options app_ai.TextGenerationOptions) (app_ai.TextGenerationOptions, error) {
	if options.UserID == uuid.Nil {
		options.RequestID = uuid.Nil
		return options, nil
	}
	operation := options.Operation
	if operation == "" {
		operation = contract.AIOperationChat
	}
	tokens := int64(len([]byte(systemPrompt)) + len([]byte(userPrompt)))
	if s.cfg.AI.FreeTextPrimary.Provider == ProviderGemini {
		if count, err := s.countGoogleTokens(ctx, s.cfg.AI.FreeTextPrimary, systemPrompt, userPrompt); err == nil && count > 0 {
			tokens = count
		}
	}
	decision, err := s.costPolicy.PrepareFree(ctx, options.UserID, operation, tokens, options.MaxOutputTokens)
	if err != nil {
		return options, err
	}
	if decision.Route == contract.AIRouteLocal {
		return options, apperror.NewInternalError("AI local fallback requested")
	}
	options.RequestID = decision.RequestID
	return options, nil
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
	options app_ai.TextGenerationOptions,
) (<-chan string, <-chan error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAITextStream(ctx, client, provider, systemPrompt, userPrompt, options)
	case ProviderGemini:
		return s.callGoogleTextStream(ctx, client, provider, systemPrompt, userPrompt, options)
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

func (s *AIService) tryTextProvider(ctx context.Context, client *http.Client, provider config.APIProviderConfig, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return s.callOpenAIText(ctx, client, provider, systemPrompt, userPrompt, options)
	case ProviderGemini:
		return s.callGoogleText(ctx, client, provider, systemPrompt, userPrompt, options)
	}

	return "", apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phản hồi văn bản này.")
}
