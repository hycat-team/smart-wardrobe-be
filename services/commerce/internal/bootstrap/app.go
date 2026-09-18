// Package bootstrap owns the Commerce Core service app lifecycle: routes,
// background workers and graceful shutdown, selected by APP_MODE (http |
// worker | all) per FR-044. Commerce hosts no gRPC service.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ServiceName identifies the service in logs and health payloads.
const ServiceName = "commerce"

const shutdownGrace = 30 * time.Second

// App is the composed Commerce Core service runtime.
type App struct {
	Config       *sharedconfig.Config
	Log          logger.Interface
	Engine       *gin.Engine
	Workers      *Workers
	DB           *gorm.DB
	Validator    sharedbootstrap.StartupValidator
	MigrationsFS fs.FS
}

// NewApp assembles the runtime from the Wire composition root.
func NewApp(
	cfg *sharedconfig.Config,
	log logger.Interface,
	engine *gin.Engine,
	workers *Workers,
	dbConn *gorm.DB,
	validator sharedbootstrap.StartupValidator,
	migrationsFS fs.FS,
) *App {
	return &App{
		Config:       cfg,
		Log:          log,
		Engine:       engine,
		Workers:      workers,
		DB:           dbConn,
		Validator:    validator,
		MigrationsFS: migrationsFS,
	}
}

// Run boots the service, blocks until SIGINT/SIGTERM, then shuts down.
func (a *App) Run() error {
	ctx := context.Background()

	if a.DB != nil && a.MigrationsFS != nil {
		if err := db.RunMigrationsIfEnabled(ctx, a.Config, a.DB, a.MigrationsFS, a.Log); err != nil {
			return fmt.Errorf("database migration failed: %w", err)
		}
	}
	if a.Validator != nil {
		if err := a.Validator.Validate(ctx); err != nil {
			return fmt.Errorf("startup validation failed: %w", err)
		}
	}

	mode := Mode(a.Config)
	started := a.startWorkers(ctx, mode)

	server := &http.Server{
		Addr:         ":" + a.Config.Server.Port,
		Handler:      a.Engine,
		ReadTimeout:  time.Duration(a.Config.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(a.Config.Server.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:  time.Duration(a.Config.Server.IdleTimeoutSeconds) * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		a.Log.Info("service listening",
			zap.String("service", ServiceName),
			zap.String("port", a.Config.Server.Port),
			zap.String("app_mode", mode),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		a.stopWorkers(started)
		return fmt.Errorf("http server failed: %w", err)
	case sig := <-stop:
		a.Log.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		a.Log.Warn("http shutdown did not complete cleanly", zap.Error(err))
	}
	a.stopWorkers(started)
	return nil
}

// Mode returns the normalized APP_MODE, defaulting to `all` so self-hosted
// runs keep the previous behaviour (FR-044).
func Mode(cfg *sharedconfig.Config) string {
	mode := "all"
	if cfg != nil && cfg.App.Mode != "" {
		mode = strings.ToLower(strings.TrimSpace(cfg.App.Mode))
	}
	switch mode {
	case "http", "worker", "all":
		return mode
	default:
		return "all"
	}
}

// startWorkers starts consumers/outbox/cron unless the mode is `http`
// (FR-044/SC-018). It returns the workers that actually started.
func (a *App) startWorkers(ctx context.Context, mode string) []Worker {
	if mode == "http" {
		a.Log.Info("APP_MODE=http: consumers/outbox/cron are not started")
		return nil
	}

	var started []Worker
	for _, w := range a.Workers.All() {
		if w == nil {
			continue
		}
		w.Start(ctx)
		started = append(started, w)
	}
	a.Log.Info("background workers started", zap.Int("count", len(started)), zap.String("app_mode", mode))
	return started
}

func (a *App) stopWorkers(started []Worker) {
	for i := len(started) - 1; i >= 0; i-- {
		started[i].Stop()
	}
}
