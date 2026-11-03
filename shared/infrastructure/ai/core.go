package ai

import (
	"context"
	"net/http"
	"time"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/httptransport"
	"smart-wardrobe-be/shared/pkg/logger"

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
	// textSemaphore is the shared concurrency gate for every text attempt. It
	// sits in front of the per-bucket RPM limiters so a whole request holds at
	// most one slot while it waits for a provider.
	textSemaphore chan struct{}
	estimator     *LocalTokenEstimator
	logger        logger.Interface
}

func NewAIService(cfg *config.Config, logger logger.Interface, estimator *LocalTokenEstimator) app_ai.IAIService {
	transport := httptransport.DefaultTransport()
	return &AIService{
		cfg: cfg,
		// Text clients carry no global Timeout: each attempt sets its own
		// deadline, so an Ollama attempt is not cut at the Gemini capability
		// timeout and a fallback still has a bounded budget.
		chatTextClient:            &http.Client{Transport: transport},
		recommendationTextClient:  &http.Client{Transport: transport},
		freeTextClient:            &http.Client{Transport: transport},
		visionClient:              &http.Client{Transport: transport, Timeout: time.Duration(cfg.AI.VisionTimeoutSeconds) * time.Second},
		embeddingClient:           &http.Client{Transport: transport, Timeout: time.Duration(cfg.AI.EmbeddingTimeoutSeconds) * time.Second},
		chatTextLimiter:           newRPMLimiter(cfg.AI.ChatTextRPMLimit, 5),
		recommendationTextLimiter: newRPMLimiter(cfg.AI.RecommendationTextRPMLimit, 5),
		visionLimiter:             newRPMLimiter(cfg.AI.VisionRPMLimit, 5),
		embeddingLimiter:          newRPMLimiter(cfg.AI.EmbeddingRPMLimit, 5),
		freeTextLimiter:           newRPMLimiter(cfg.AI.FreeTextRPMLimit, 5),
		textSemaphore:             make(chan struct{}, textConcurrency(cfg.AI.TextMaxConcurrency)),
		estimator:                 estimator,
		logger:                    logger,
	}
}

// textConcurrency resolves the shared text semaphore size, defaulting to 2 so a
// configuration written before feature 032 behaves like the trial default.
func textConcurrency(configured int) int {
	if configured < 1 {
		return config.DefaultTextMaxConcurrency
	}
	return configured
}

// acquireTextSlot blocks until a text attempt slot is free or ctx is cancelled.
// A nil semaphore (tests constructing AIService directly) is a no-op.
func (s *AIService) acquireTextSlot(ctx context.Context) error {
	if s.textSemaphore == nil {
		return nil
	}
	select {
	case s.textSemaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseTextSlot returns a slot acquired by acquireTextSlot. It is safe to call
// on a nil/empty semaphore because tests build AIService without one.
func (s *AIService) releaseTextSlot() {
	if s.textSemaphore == nil {
		return
	}
	select {
	case <-s.textSemaphore:
	default:
	}
}

// textAttemptTimeout returns the deadline for one text attempt. openai (the
// Ollama trial) uses the primary attempt timeout; a google fallback uses the
// fallback timeout; a google primary keeps its per-capability timeout so Gemini
// behaviour outside the trial is unchanged.
func (s *AIService) textAttemptTimeout(provider config.APIProviderConfig, capability string, fallback bool) time.Duration {
	seconds := 0
	switch {
	case provider.Provider == ProviderOpenAI:
		seconds = s.cfg.AI.TextPrimaryTimeoutSeconds
	case fallback:
		seconds = s.cfg.AI.TextFallbackTimeoutSeconds
	default:
		switch capability {
		case config.CapabilityRecommendationText:
			seconds = s.cfg.AI.RecommendationTextTimeoutSeconds
		case config.CapabilityFreeText:
			seconds = s.cfg.AI.FreeTextTimeoutSeconds
		default:
			seconds = s.cfg.AI.ChatTextTimeoutSeconds
		}
	}
	if seconds <= 0 {
		seconds = config.DefaultTextPrimaryTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

// withTextAttemptDeadline bounds one provider attempt. Callers must invoke the
// cancel function when the attempt finishes.
func (s *AIService) withTextAttemptDeadline(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}
