package ai

import (
	"context"
	"math"
	"net/http"
	"strings"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/ai/google"
	"smart-wardrobe-be/shared/infrastructure/ai/openai"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func (s *AIService) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	s.logTextGeneration("GenerateChatText", systemPrompt, userPrompt, options)
	provider, client, limiter, prepared, err := s.prepareText(ctx, s.cfg.AI.ChatTextPrimary, systemPrompt, userPrompt, options, app_ai.AIOperationChat)
	if err != nil {
		return "", err
	}
	if err = limiter.Wait(ctx); err != nil {
		return "", err
	}
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
	provider, client, limiter, prepared, err := s.prepareText(ctx, s.cfg.AI.RecommendationTextPrimary, systemPrompt, userPrompt, options, app_ai.AIOperationOutfit)
	if err != nil {
		return "", err
	}
	if err = limiter.Wait(ctx); err != nil {
		return "", err
	}
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

	provider, client, limiter, prepared, err := s.prepareText(ctx, s.cfg.AI.ChatTextPrimary, systemPrompt, userPrompt, options, app_ai.AIOperationChat)
	if err != nil {
		outErrChan <- err
		close(outTextChan)
		close(outErrChan)
		return outTextChan, outErrChan
	}
	s.logTextGeneration("GenerateChatTextStream", systemPrompt, userPrompt, prepared)
	if err := limiter.Wait(ctx); err != nil {
		outErrChan <- err
		close(outTextChan)
		close(outErrChan)
		return outTextChan, outErrChan
	}

	go func() {
		defer close(outTextChan)
		defer close(outErrChan)

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
		return
	}
}

func (s *AIService) prepareText(ctx context.Context, paid config.APIProviderConfig, systemPrompt, userPrompt string, options app_ai.TextGenerationOptions, defaultOperation string) (config.APIProviderConfig, *http.Client, *rate.Limiter, app_ai.TextGenerationOptions, error) {
	var geminiReq google.PreparedGeminiRequest
	isGemini := paid.Provider == ProviderGemini
	if isGemini {
		geminiReq = google.PreparedGeminiRequest{
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
				GenerationConfig: google.GoogleGenerationConfig(options),
			},
		}
		if strings.TrimSpace(systemPrompt) != "" {
			geminiReq.Body.SystemInstruction = &google.GeminiContent{
				Parts: []google.GeminiPart{
					{Text: systemPrompt},
				},
			}
		}
	}

	if defaultOperation == app_ai.AIOperationOutfit {
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

// newRPMLimiter creates a per-minute limiter sized to absorb one second of traffic.
func newRPMLimiter(rpm int, fallback int) *rate.Limiter {
	if rpm <= 0 {
		rpm = fallback
	}
	burst := int(math.Ceil(float64(rpm) / 60.0))
	if burst < 1 {
		burst = 1
	}

	return rate.NewLimiter(rate.Limit(float64(rpm)/60.0), burst)
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
		return "", err
	}
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
		return openai.CallTextStream(ctx, client, provider, systemPrompt, userPrompt, options)
	case ProviderGemini:
		return google.CallTextStream(ctx, client, provider, systemPrompt, userPrompt, options)
	}

	errChan := make(chan error, 1)
	errChan <- apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phản hồi văn bản này.")
	close(errChan)
	textChan := make(chan string)
	close(textChan)
	return textChan, errChan
}

func (s *AIService) tryTextProvider(ctx context.Context, client *http.Client, provider config.APIProviderConfig, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	switch provider.Provider {
	case ProviderOpenAI:
		return openai.CallText(ctx, client, provider, systemPrompt, userPrompt, options)
	case ProviderGemini:
		return google.CallText(ctx, client, provider, systemPrompt, userPrompt, options)
	}

	return "", apperror.NewInternalError("Hệ thống chưa hỗ trợ nhà cung cấp dịch vụ phản hồi văn bản này.")
}
