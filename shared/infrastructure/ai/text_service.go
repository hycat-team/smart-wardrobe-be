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
	return s.generateText(ctx, systemPrompt, userPrompt, options, app_ai.AIOperationChat, "GenerateChatText")
}

func (s *AIService) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options app_ai.TextGenerationOptions) (string, error) {
	return s.generateText(ctx, systemPrompt, userPrompt, options, app_ai.AIOperationOutfit, "GenerateRecommendationText")
}

// generateText runs one non-streaming text call following the resolved tier:
// paid primary -> paid fallback -> free failover, or free primary -> free
// fallback. The tiers never swap implicitly in the free direction, because a
// free key is a separate quota, not a cheaper copy of the paid one.
func (s *AIService) generateText(
	ctx context.Context,
	systemPrompt string,
	userPrompt string,
	options app_ai.TextGenerationOptions,
	operation string,
	label string,
) (string, error) {
	s.logTextGeneration(label, systemPrompt, userPrompt, options)

	plan := s.resolveTextCallPlan(options, operation)

	var text string
	var err error
	if plan.primary.provider.Provider != "" {
		if waitErr := plan.primary.limiter.Wait(ctx); waitErr != nil {
			return "", waitErr
		}
		text, err = s.tryTextProvider(ctx, plan.primary.client, plan.primary.provider, systemPrompt, userPrompt, s.withUsageStamp(options, plan.primary))
		if err == nil {
			return text, nil
		}
	}

	if plan.fallback.provider.Provider != "" {
		s.logger.Warn("Primary text provider failed, attempting fallback", zap.Error(err))
		text, err = s.tryTextProvider(ctx, plan.fallback.client, plan.fallback.provider, systemPrompt, userPrompt, s.withUsageStamp(options, plan.fallback))
		if err == nil {
			return text, nil
		}
	}

	if plan.freeFailover != nil && plan.freeFailover.provider.Provider != "" {
		if waitErr := plan.freeFailover.limiter.Wait(ctx); waitErr == nil {
			s.logger.Warn("Text providers failed, attempting free provider", zap.Error(err))
			text, err = s.tryTextProvider(ctx, plan.freeFailover.client, plan.freeFailover.provider, systemPrompt, userPrompt, s.withUsageStamp(options, *plan.freeFailover))
			if err == nil {
				return text, nil
			}
		}
	}

	if err == nil {
		err = apperror.NewInternalError("AI provider is not configured")
	}
	s.finalizeProviderError(context.WithoutCancel(ctx), options.RequestID, err)
	return "", err
}

func (s *AIService) GenerateChatTextStream(
	ctx context.Context,
	systemPrompt string,
	userPrompt string,
	options app_ai.TextGenerationOptions,
) (<-chan string, <-chan error) {
	outTextChan := make(chan string, 100)
	outErrChan := make(chan error, 1)

	plan := s.resolveTextCallPlan(options, app_ai.AIOperationChat)
	if plan.primary.provider.Provider == "" && plan.fallback.provider.Provider == "" && plan.freeFailover == nil {
		outErrChan <- apperror.NewInternalError("AI provider is not configured")
		close(outTextChan)
		close(outErrChan)
		return outTextChan, outErrChan
	}
	s.logTextGeneration("GenerateChatTextStream", systemPrompt, userPrompt, options)
	if plan.primary.provider.Provider != "" {
		if err := plan.primary.limiter.Wait(ctx); err != nil {
			outErrChan <- err
			close(outTextChan)
			close(outErrChan)
			return outTextChan, outErrChan
		}
	}

	go func() {
		defer close(outTextChan)
		defer close(outErrChan)

		produced := false
		var err error
		if plan.primary.provider.Provider != "" {
			produced, err = s.streamTextSingle(ctx, plan.primary.client, plan.primary.provider, systemPrompt, userPrompt, s.withUsageStamp(options, plan.primary), outTextChan)
			if err == nil {
				return
			}
		}

		if !produced && plan.fallback.provider.Provider != "" {
			s.logger.Warn("Chat primary stream failed before producing content, attempting fallback", zap.Error(err))
			produced, err = s.streamTextSingle(ctx, plan.fallback.client, plan.fallback.provider, systemPrompt, userPrompt, s.withUsageStamp(options, plan.fallback), outTextChan)
			if err == nil {
				return
			}
		}

		if !produced && plan.freeFailover != nil && plan.freeFailover.provider.Provider != "" {
			if waitErr := plan.freeFailover.limiter.Wait(ctx); waitErr == nil {
				_, fallbackErr := s.streamTextSingle(ctx, plan.freeFailover.client, plan.freeFailover.provider, systemPrompt, userPrompt, s.withUsageStamp(options, *plan.freeFailover), outTextChan)
				if fallbackErr == nil {
					return
				}
				err = fallbackErr
			}
		}

		if err == nil {
			err = apperror.NewInternalError("AI provider is not configured")
		}
		s.finalizeProviderError(context.WithoutCancel(ctx), options.RequestID, err)
		if !produced && ctx.Err() == nil {
			outTextChan <- "Xin lỗi, stylist AI đang tạm thời bận. Bạn hãy thử lại sau ít phút nhé."
			return
		}
		outErrChanSend(outErrChan, err)
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
