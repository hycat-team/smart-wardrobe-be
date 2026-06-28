package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"smart-wardrobe-be/config"
	brandWorker "smart-wardrobe-be/internal/modules/brand/presentation/worker"
	subWorker "smart-wardrobe-be/internal/modules/subscription/presentation/worker"
	wardrobeWorker "smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AppWorkers struct {
	RenewalWorker               subWorker.ISubscriptionRenewalWorker
	PaymentReconciliationWorker subWorker.IPaymentReconciliationWorker
	WebhookInboxWorker          subWorker.IWebhookInboxWorker
	AIUsageReconciliationWorker subWorker.IAIUsageReconciliationWorker
	LoyaltyPointExpiryWorker    brandWorker.ILoyaltyPointExpiryWorker
	WardrobeBatchUploadWorker   *wardrobeWorker.WardrobeBatchUploadWorker
	ESAsyncWorker               *wardrobeWorker.SearchSyncWorker
	FailedItemsCleanupWorker    wardrobeWorker.IFailedItemsCleanupWorker
	ProcessingRecoveryWorker    wardrobeWorker.IProcessingRecoveryWorker
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
	// 1. Run database migrations first
	if err := db.RunMigrations(context.Background(), a.DB); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	// 2. Validate startup configurations
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

	a.Workers.RenewalWorker.Start()
	a.Workers.PaymentReconciliationWorker.Start()
	a.Workers.WebhookInboxWorker.Start()
	a.Workers.AIUsageReconciliationWorker.Start()
	a.Workers.LoyaltyPointExpiryWorker.Start()
	a.Workers.FailedItemsCleanupWorker.Start()
	a.Workers.ProcessingRecoveryWorker.Start()

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Shutting down Gin server...")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("Server forced to shutdown: %w", err)
	}

	return nil
}
