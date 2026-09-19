package bootstrap

import (
	"net/http"
	"time"

	brandrouter "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/router"
	dashboardrouter "smart-wardrobe-be/services/commerce/internal/modules/dashboard/presentation/router"
	fashionrouter "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/router"
	wardroberouter "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/router"
	"smart-wardrobe-be/shared/application/constants/apperror"
	sharedconfig "smart-wardrobe-be/shared/config"
	sharedsearch "smart-wardrobe-be/shared/infrastructure/search"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// healthHandler serves liveness probes for /health and the per-service alias
// (/commerce/health) used by deploy smoke tests and uptime monitors.
//
// @Summary Commerce service liveness probe
// @Description Liveness probe of the Commerce service.
// @Tags Commerce Health
// @Produce json
// @Success 200 {object} object "Liveness payload with status, service, time and mode"
// @Router /api/v1/commerce/health [get]
func healthHandler(cfg *sharedconfig.Config, searchClient *sharedsearch.ElasticsearchClient) gin.HandlerFunc {
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
		})
	}
}

// NewEngine builds the Commerce HTTP engine: shared middleware stack, the
// module routers and the health endpoint. Commerce hosts no gRPC service,
// so the engine serves plain HTTP (the Cloud Run --use-http2 flag in T073a
// applies to the platform, not the app).
func NewEngine(
	cfg *sharedconfig.Config,
	log logger.Interface,
	rateLimit *shared_middleware.RateLimitMiddleware,
	bodyLimit gin.HandlerFunc,
	wardrobeRouter *wardroberouter.Router,
	dashboardRouter *dashboardrouter.Router,
	brandRouter *brandrouter.Router,
	fashionRouter *fashionrouter.Router,
	searchClient *sharedsearch.ElasticsearchClient,
) *gin.Engine {
	if cfg.Server.Env == "production" {
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
			return (p == "/api/v1/health" || p == "/api/v1/commerce/health") && c.Writer.Status() == http.StatusOK
		},
	}))
	engine.Use(shared_middleware.GlobalErrorHandler(log, cfg.Server.Env))
	engine.Use(shared_middleware.CORSMiddleware(cfg.Server.FrontEndOrigin))
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
		health := healthHandler(cfg, searchClient)
		api.GET("/health", health)
		// Per-service alias: distinct path per service so the generated Nginx
		// route map routes each alias to its owner (shared /health cannot be
		// documented per service without a duplicate-path conflict).
		api.GET("/commerce/health", health)

		wardrobeRouter.Init(api)
		dashboardRouter.Init(api)
		brandRouter.Init(api)
		fashionRouter.Init(api)
	}

	return engine
}
