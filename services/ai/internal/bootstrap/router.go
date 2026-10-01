package bootstrap

import (
	"net/http"
	"time"

	grant "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	airouter "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/router"

	"smart-wardrobe-be/shared/application/constants/apperror"
	sharedconfig "smart-wardrobe-be/shared/config"
	sharedsearch "smart-wardrobe-be/shared/infrastructure/search"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// healthHandler serves liveness probes for /health and the per-service alias
// (/ai/health) used by deploy smoke tests and uptime monitors.
//
// @Summary AI service liveness probe
// @Description Liveness probe of the AI service, including the AI policy grant counters.
// @Tags AI Health
// @Produce json
// @Success 200 {object} object "Liveness payload with status, service, time, mode and grant counters"
// @Router /api/v1/ai/health [get]
//
// 024 T139: the grant counters live here because this is the endpoint an uptime
// monitor already polls, so FR-087's "cần phát hiện khi đếm tăng" becomes
// answerable by something an operator is already looking at.
//
// The counters used to exist in process and had no consumer at all: `Snapshot()`
// was called from tests only. A rejected event then produced a log line, a healthy
// consumer, a zero queue depth and an entitlement that never arrived - which is
// the exact combination FR-010 exists to make distinguishable, and which nothing
// could distinguish.
func healthHandler(
	cfg *sharedconfig.Config,
	searchClient *sharedsearch.ElasticsearchClient,
	grantMetrics *grant.GrantMetrics,
	fallbackMetrics *grant.FallbackMetrics,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": ServiceName,
			"time":    time.Now().Format(time.RFC3339),
			"mode":    cfg.App.Mode,
			"elasticsearch": gin.H{
				"enabled": searchClient.Enabled(),
				"healthy": searchClient.IsHealthy(),
			},
			// Rejected events, keyed by reason code (FR-087, SC-023).
			"grant_rejected_by_reason": grantMetrics.Snapshot(),
			// Degradations, keyed by fallback reason (FR-010).
			"cost_fallback_by_reason": fallbackMetrics.Snapshot(),
		})
	}
}

// NewEngine builds the AI HTTP engine: shared middleware stack, the module
// router and the health endpoint (FR-044 health on PORT).
func NewEngine(
	cfg *sharedconfig.Config,
	log logger.Interface,
	rateLimit *shared_middleware.RateLimitMiddleware,
	bodyLimit gin.HandlerFunc,
	aiRouter *airouter.Router,
	searchClient *sharedsearch.ElasticsearchClient,
	// 024 T139: injected so the health payload can publish the grant counters.
	grantMetrics *grant.GrantMetrics,
	fallbackMetrics *grant.FallbackMetrics,
) *gin.Engine {
	if sharedconfig.IsProductionEnv(cfg.Server.Env) {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	gin.DebugPrintRouteFunc = func(string, string, string, int) {}

	engine := gin.New()
	shared_pres.ConfigureSSEDeadlines(
		time.Duration(cfg.Server.SSE.ChunkDeadlineSeconds)*time.Second,
		time.Duration(cfg.Server.SSE.AIChatChunkDeadlineSeconds)*time.Second,
	)
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Skip: func(c *gin.Context) bool {
			p := c.Request.URL.Path
			return (p == "/api/v1/health" || p == "/api/v1/ai/health") && c.Writer.Status() == http.StatusOK
		},
	}))
	engine.Use(shared_middleware.GlobalErrorHandler(log, cfg.Server.Env))
	engine.Use(shared_middleware.CORSMiddleware(cfg.Server.FrontEndOrigins))
	engine.Use(bodyLimit)
	if cfg.Server.GzipEnabled {
		engine.Use(shared_middleware.GzipMinSizeMiddleware(cfg))
	}
	engine.Use(shared_middleware.GlobalTimeoutMiddleware(time.Duration(cfg.Server.TimeoutSeconds) * time.Second))
	engine.Use(rateLimit.Handle())

	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, apperror.NewNotFound("Đường dẫn API không tồn tại hoặc phương thức không được hỗ trợ."))
	})
	engine.HandleMethodNotAllowed = true
	engine.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, apperror.NewError(
			http.StatusMethodNotAllowed,
			"Phương thức không được hỗ trợ",
			"Phương thức HTTP yêu cầu không được hỗ trợ cho đường dẫn này.",
		))
	})

	swaggerGroup := engine.Group("")
	swaggerGroup.Use(shared_middleware.SwaggerLockMiddleware(cfg))
	{
		swaggerGroup.Static("/api-docs", "./api/swagger")
		swaggerGroup.StaticFile("/swagger", "./api/swagger/index.html")
		swaggerGroup.GET("/swagger/*any", func(c *gin.Context) {
			c.File("./api/swagger/index.html")
		})
	}

	api := engine.Group("/api/v1")
	{
		health := healthHandler(cfg, searchClient, grantMetrics, fallbackMetrics)
		api.GET("/health", health)
		// Per-service alias: distinct path per service so the generated Nginx
		// route map routes each alias to its owner (shared /health cannot be
		// documented per service without a duplicate-path conflict).
		api.GET("/ai/health", health)

		aiRouter.Init(api)
	}

	return engine
}
