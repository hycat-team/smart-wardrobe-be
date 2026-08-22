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
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	sharedWorker "smart-wardrobe-be/internal/shared/presentation/worker"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AppWorkers struct {
	RenewalWorker               *subWorker.SubscriptionRenewalWorker
	PaymentReconciliationWorker *subWorker.PaymentReconciliationWorker
	WebhookInboxWorker          *subWorker.WebhookInboxWorker
	AIUsageReconciliationWorker *aiexecutorWorker.AIUsageReconciliationWorker
	LoyaltyPointExpiryWorker    *brandWorker.LoyaltyPointExpiryWorker
	ESAsyncWorker               *wardrobeWorker.SearchSyncWorker
	FashionInitWorker           *fashionWorker.FashionInitWorker
	FailedItemsCleanupWorker    *fashionWorker.FailedItemsCleanupWorker
	ProcessingRecoveryWorker    *fashionWorker.ProcessingRecoveryWorker
	FashionAnalyzeWorker        *fashionWorker.FashionAnalyzeWorker
	PlanChangedWorker           *aiexecutorWorker.PlanChangedWorker
	DashboardAggregatorWorker   *dashboardWorker.DashboardAggregatorWorker
	OutboxWorker                *sharedWorker.OutboxWorker
	EmailWorker                 *notificationWorker.EmailWorker
	AuditLogConsumerWorker      *adminWorker.AuditLogConsumerWorker
	AuditCleanupWorker          *adminWorker.AuditCleanupWorker
}

type App struct {
	Config    *config.Config
	Server    *gin.Engine
	Workers   *AppWorkers
	Validator StartupValidator
	DB        *gorm.DB
}

func NewApp(
	cfg *config.Config,
	server *gin.Engine,
	workers *AppWorkers,
	validator StartupValidator,
	dbConn *gorm.DB,
) *App {
	return &App{
		Config:    cfg,
		Server:    server,
		Workers:   workers,
		Validator: validator,
		DB:        dbConn,
	}
}

func (a *App) Run() error {
	if err := db.RunMigrations(context.Background(), a.DB); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	if err := a.Validator.Validate(context.Background()); err != nil {
		return fmt.Errorf("startup validation failed: %w", err)
	}
	port := a.Config.Server.Port

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: a.Server,
	}

	fmt.Println("==========================================================")
	fmt.Printf("SmartWardrobe BE is running on port: %s\n", port)
	fmt.Printf("Swagger UI is available at: http://localhost:%s/swagger\n", port)
	fmt.Println("==========================================================")

	ctxWorker := context.Background()

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

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Listen error: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	fmt.Println("\nReceived shutdown signal...")

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Shutting down Gin server...")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("Server forced to shutdown: %w", err)
	}

	return nil
}
