package bootstrap

import (
	"net/http"
	"time"

	airouter "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/router"
	"smart-wardrobe-be/shared/application/constants/apperror"
	sharedconfig "smart-wardrobe-be/shared/config"
	sharedsearch "smart-wardrobe-be/shared/infrastructure/search"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// NewEngine builds the AI HTTP engine: shared middleware stack, the module
// router and the health endpoint (FR-044 health on PORT).
func NewEngine(
	cfg *sharedconfig.Config,
	log logger.Interface,
	rateLimit *shared_middleware.RateLimitMiddleware,
	bodyLimit gin.HandlerFunc,
	aiRouter *airouter.Router,
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
			return c.Request.URL.Path == "/api/v1/health" && c.Writer.Status() == http.StatusOK
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
		api.GET("/health", func(c *gin.Context) {
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
		})

		aiRouter.Init(api)
	}

	return engine
}
