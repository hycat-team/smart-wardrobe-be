package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"smart-wardrobe-be/config"
	communityWorker "smart-wardrobe-be/internal/modules/community/presentation/worker"
	subWorker "smart-wardrobe-be/internal/modules/subscription/presentation/worker"
	wardrobeWorker "smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type AppWorkers struct {
	RenewalWorker             subWorker.ISubscriptionRenewalWorker
	PostHotnessWorker         communityWorker.IPostHotnessWorker
	WardrobeBatchUploadWorker *wardrobeWorker.WardrobeBatchUploadWorker
	ESAsyncWorker             *wardrobeWorker.SearchSyncWorker
	FailedItemsCleanupWorker  wardrobeWorker.IFailedItemsCleanupWorker
}

type App struct {
	Config  *config.Config
	Server  *gin.Engine
	Workers *AppWorkers
}

func NewApp(
	cfg *config.Config,
	server *gin.Engine,
	workers *AppWorkers,
) *App {
	return &App{
		Config:  cfg,
		Server:  server,
		Workers: workers,
	}
}

func (a *App) Run() error {
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
	a.Workers.PostHotnessWorker.Start()
	a.Workers.FailedItemsCleanupWorker.Start()

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
	a.Workers.PostHotnessWorker.Stop()
	a.Workers.FailedItemsCleanupWorker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Shutting down Gin server...")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("Server forced to shutdown: %w", err)
	}

	return nil
}
