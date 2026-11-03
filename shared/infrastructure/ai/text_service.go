package ai

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

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

// generateText runs one non-streaming text call following the resolved plan:
// exactly one primary attempt and, only for a fallback-eligible failure, one
// Gemini fallback. There is no third free failover any more (feature 032 R5), and
// the tiers never swap implicitly in the free direction.
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
	attempts := plan.textAttempts()
	if len(attempts) == 0 {
		err := apperror.NewInternalError("AI provider is not configured")
		s.finalizeProviderError(context.WithoutCancel(ctx), options.RequestID, err)
		return "", err
	}

	var lastErr error
	for index, call := range attempts {
		text, err := s.runTextAttempt(ctx, call, systemPrompt, userPrompt, operation, index+1, options)
		if err == nil {
			return text, nil
		}
		lastErr = err

		if index == len(attempts)-1 || !s.textFallbackEligible(err) {
			break
		}
		s.logger.Warn("Primary text provider failed, attempting fallback", zap.Error(err))
	}

	s.finalizeProviderError(context.WithoutCancel(ctx), options.RequestID, lastErr)
	return "", lastErr
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
	attempts := plan.textAttempts()
	if len(attempts) == 0 {
		outErrChan <- apperror.NewInternalError("AI provider is not configured")
		close(outTextChan)
		close(outErrChan)
		return outTextChan, outErrChan
	}
	s.logTextGeneration("GenerateChatTextStream", systemPrompt, userPrompt, options)

	go func() {
		defer close(outTextChan)
		defer close(outErrChan)

		var lastErr error
		produced := false
		for index, call := range attempts {
			streamed, err := s.runStreamAttempt(ctx, call, systemPrompt, userPrompt, app_ai.AIOperationChat, index+1, options, outTextChan)
			produced = produced || streamed
			if err == nil {
				return
			}
			lastErr = err

			// Falling back after the first chunk would splice two answers
			// together, so a produced stream is always terminal.
			if produced || index == len(attempts)-1 || !s.textFallbackEligible(err) {
				break
			}
			s.logger.Warn("Chat primary stream failed before producing content, attempting fallback", zap.Error(err))
		}

		s.finalizeProviderError(context.WithoutCancel(ctx), options.RequestID, lastErr)
		var blocked *attemptBlockedError
		if !produced && ctx.Err() == nil && !errors.As(lastErr, &blocked) {
			outTextChan <- "Xin lỗi, stylist AI đang tạm thời bận. Bạn hãy thử lại sau ít phút nhé."
			return
		}
		outErrChanSend(outErrChan, lastErr)
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

// textFallbackEligible reports whether a failed attempt may be retried on the
// next provider. Classification lives in text_attempt.go so the streaming and
// non-streaming paths cannot disagree.
func (s *AIService) textFallbackEligible(err error) bool {
	_, _, eligible := classifyTextAttempt(err)
	return eligible
}

// forwardProviderStream forwards provider chunks to the caller, reports whether
// any content was produced and measures time-to-first-token from started.
func (s *AIService) forwardProviderStream(
	ctx context.Context,
	started time.Time,
	inText <-chan string,
	inErr <-chan error,
	outText chan<- string,
) (bool, time.Duration, error) {
	produced := false
	var ttft time.Duration

	for {
		select {
		case <-ctx.Done():
			return produced, ttft, ctx.Err()
		case text, ok := <-inText:
			if ok {
				if !produced {
					ttft = time.Since(started)
				}
				produced = true
				select {
				case <-ctx.Done():
					return produced, ttft, ctx.Err()
				case outText <- text:
				}
				continue
			}

			if err, ok := <-inErr; ok && err != nil {
				return produced, ttft, err
			}
			return produced, ttft, nil
		case err, ok := <-inErr:
			if ok && err != nil {
				return produced, ttft, err
			}
			if !ok {
				for text := range inText {
					if !produced {
						ttft = time.Since(started)
					}
					produced = true
					select {
					case <-ctx.Done():
						return produced, ttft, ctx.Err()
					case outText <- text:
					}
				}
				return produced, ttft, nil
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
