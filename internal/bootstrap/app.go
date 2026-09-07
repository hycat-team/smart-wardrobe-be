package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"smart-wardrobe-be/internal/config"
	adminWorker "smart-wardrobe-be/internal/modules/admin/presentation/worker"
	aiexecutorWorker "smart-wardrobe-be/internal/modules/aiexecutor/presentation/worker"
	brandWorker "smart-wardrobe-be/internal/modules/brand/presentation/worker"
	dashboardWorker "smart-wardrobe-be/internal/modules/dashboard/presentation/worker"
	fashionWorker "smart-wardrobe-be/internal/modules/fashion/presentation/worker"
	notificationWorker "smart-wardrobe-be/internal/modules/notification/presentation/worker"
	subWorker "smart-wardrobe-be/internal/modules/subscription/presentation/worker"
	wardrobeWorker "smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	sharedWorker "smart-wardrobe-be/internal/shared/presentation/worker"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AppWorkers struct {
	RenewalWorker                          *subWorker.SubscriptionRenewalWorker
	PaymentReconciliationWorker            *subWorker.PaymentReconciliationWorker
	WebhookInboxWorker                     *subWorker.WebhookInboxWorker
	AIUsageReconciliationWorker            *aiexecutorWorker.AIUsageReconciliationWorker
	LoyaltyPointExpiryWorker               *brandWorker.LoyaltyPointExpiryWorker
	ESAsyncWorker                          *wardrobeWorker.SearchSyncWorker
	FashionInitWorker                      *fashionWorker.FashionInitWorker
	FailedItemsCleanupWorker               *fashionWorker.FailedItemsCleanupWorker
	ProcessingRecoveryWorker               *fashionWorker.ProcessingRecoveryWorker
	FashionAnalyzeWorker                   *fashionWorker.FashionAnalyzeWorker
	PlanChangedWorker                      *aiexecutorWorker.PlanChangedWorker
	DashboardAggregatorWorker              *dashboardWorker.DashboardAggregatorWorker
	OutboxWorker                           *sharedWorker.OutboxWorker
	EmailWorker                            *notificationWorker.EmailWorker
	AuditLogConsumerWorker                 *adminWorker.AuditLogConsumerWorker
	AuditCleanupWorker                     *adminWorker.AuditCleanupWorker
	WardrobeFashionEventWorker             *wardrobeWorker.WardrobeFashionEventWorker
	BrandFashionEventWorker                *brandWorker.BrandFashionEventWorker
	WardrobeCleanupWorker                  *wardrobeWorker.WardrobeCleanupWorker
	WardrobeProcessingExhaustedWorker      *wardrobeWorker.WardrobeProcessingExhaustedWorker
	WardrobeFashionBindWorker              *wardrobeWorker.WardrobeFashionBindWorker
	ElasticsearchProcessingExhaustedWorker *wardrobeWorker.ElasticsearchProcessingExhaustedWorker
	BrandFashionBindWorker                 *brandWorker.BrandFashionBindWorker
}

type App struct {
	Config    *config.Config
	Server    *gin.Engine
	Workers   *AppWorkers
	Validator StartupValidator
	DB        *gorm.DB
	Log       logger.Interface
}

func NewApp(
	cfg *config.Config,
	server *gin.Engine,
	workers *AppWorkers,
	validator StartupValidator,
	dbConn *gorm.DB,
	log logger.Interface,
) *App {
	return &App{
		Config:    cfg,
		Server:    server,
		Workers:   workers,
		Validator: validator,
		DB:        dbConn,
		Log:       log,
	}
}

func (a *App) Run() error {
	if err := ValidateServerHardening(a.Config); err != nil {
		return fmt.Errorf("server hardening config validation failed: %w", err)
	}

	if err := db.RunMigrations(context.Background(), a.DB, a.Log); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	if err := a.Validator.Validate(context.Background()); err != nil {
		return fmt.Errorf("startup validation failed: %w", err)
	}
	port := a.Config.Server.Port

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      a.Server,
		ReadTimeout:  time.Duration(a.Config.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(a.Config.Server.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:  time.Duration(a.Config.Server.IdleTimeoutSeconds) * time.Second,
	}

	a.Log.Info("SmartWardrobe BE is running",
		zap.String("port", port),
		zap.String("swagger", "http://localhost:"+port+"/swagger"),
	)

	ctxWorker := context.Background()

	// Health check đọc outbox backlog.
	a.Workers.OutboxWorker.SetHealthCheck(func(ctx context.Context) error {
		const outboxBacklogThreshold = 10000
		metrics := shared_msg.GetMetrics()
		if pending := metrics["outbox_pending"]; pending > outboxBacklogThreshold {
			return fmt.Errorf("outbox backlog high: %d pending records", pending)
		}
		return nil
	})

	a.Workers.RenewalWorker.Start(ctxWorker)
	a.Workers.PaymentReconciliationWorker.Start(ctxWorker)
	a.Workers.WebhookInboxWorker.Start(ctxWorker)
	a.Workers.AIUsageReconciliationWorker.Start(ctxWorker)
	a.Workers.LoyaltyPointExpiryWorker.Start(ctxWorker)
	a.Workers.FailedItemsCleanupWorker.Start(ctxWorker)
	a.Workers.ProcessingRecoveryWorker.Start(ctxWorker)
	a.Workers.ESAsyncWorker.Start(ctxWorker)
	a.Workers.FashionInitWorker.Start(ctxWorker)
	a.Workers.FashionAnalyzeWorker.Start(ctxWorker)
	a.Workers.PlanChangedWorker.Start(ctxWorker)
	a.Workers.DashboardAggregatorWorker.Start(ctxWorker)
	a.Workers.OutboxWorker.Start(ctxWorker)
	a.Workers.EmailWorker.Start(ctxWorker)
	a.Workers.AuditLogConsumerWorker.Start(ctxWorker)
	a.Workers.AuditCleanupWorker.Start(ctxWorker)
	a.Workers.WardrobeFashionEventWorker.Start(ctxWorker)
	a.Workers.BrandFashionEventWorker.Start(ctxWorker)
	a.Workers.WardrobeCleanupWorker.Start(ctxWorker)
	a.Workers.WardrobeProcessingExhaustedWorker.Start(ctxWorker)
	a.Workers.WardrobeFashionBindWorker.Start(ctxWorker)
	a.Workers.ElasticsearchProcessingExhaustedWorker.Start(ctxWorker)
	a.Workers.BrandFashionBindWorker.Start(ctxWorker)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Log.Error("HTTP listen failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	a.Log.Info("Received shutdown signal")

	a.Workers.RenewalWorker.Stop()
	a.Workers.PaymentReconciliationWorker.Stop()
	a.Workers.WebhookInboxWorker.Stop()
	a.Workers.AIUsageReconciliationWorker.Stop()
	a.Workers.LoyaltyPointExpiryWorker.Stop()
	a.Workers.FailedItemsCleanupWorker.Stop()
	a.Workers.ProcessingRecoveryWorker.Stop()
	a.Workers.ESAsyncWorker.Stop()
	a.Workers.FashionInitWorker.Stop()
	a.Workers.FashionAnalyzeWorker.Stop()
	a.Workers.PlanChangedWorker.Stop()
	a.Workers.DashboardAggregatorWorker.Stop()
	a.Workers.OutboxWorker.Stop()
	a.Workers.EmailWorker.Stop()
	a.Workers.AuditLogConsumerWorker.Stop()
	a.Workers.AuditCleanupWorker.Stop()
	a.Workers.WardrobeFashionEventWorker.Stop()
	a.Workers.BrandFashionEventWorker.Stop()
	a.Workers.WardrobeCleanupWorker.Stop()
	a.Workers.WardrobeProcessingExhaustedWorker.Stop()
	a.Workers.WardrobeFashionBindWorker.Stop()
	a.Workers.ElasticsearchProcessingExhaustedWorker.Stop()
	a.Workers.BrandFashionBindWorker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a.Log.Info("Shutting down Gin server")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("Server forced to shutdown: %w", err)
	}

	return nil
}
