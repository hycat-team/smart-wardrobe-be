package ai

import (
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
	estimator                 *LocalTokenEstimator
	logger                    logger.Interface
}

func NewAIService(cfg *config.Config, logger logger.Interface, estimator *LocalTokenEstimator) app_ai.IAIService {
	transport := httptransport.DefaultTransport()
	return &AIService{
		cfg: cfg,
		chatTextClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(cfg.AI.ChatTextTimeoutSeconds) * time.Second,
		},
		recommendationTextClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(cfg.AI.RecommendationTextTimeoutSeconds) * time.Second,
		},
		visionClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(cfg.AI.VisionTimeoutSeconds) * time.Second,
		},
		embeddingClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(cfg.AI.EmbeddingTimeoutSeconds) * time.Second,
		},
		freeTextClient:            &http.Client{Transport: transport, Timeout: time.Duration(cfg.AI.FreeTextTimeoutSeconds) * time.Second},
		chatTextLimiter:           newRPMLimiter(cfg.AI.ChatTextRPMLimit, 5),
		recommendationTextLimiter: newRPMLimiter(cfg.AI.RecommendationTextRPMLimit, 5),
		visionLimiter:             newRPMLimiter(cfg.AI.VisionRPMLimit, 5),
		embeddingLimiter:          newRPMLimiter(cfg.AI.EmbeddingRPMLimit, 5),
		freeTextLimiter:           newRPMLimiter(cfg.AI.FreeTextRPMLimit, 5),
		estimator:                 estimator,
		logger:                    logger,
	}
}
