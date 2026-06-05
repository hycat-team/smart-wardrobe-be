package main

import (
	"fmt"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/di"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

// @title           Smart Wardrobe API Document
// @version         1.0
// @description     API docs for Smart Wardrobe.
func main() {
	cfg := config.LoadConfig()
	l := logger.New("dev", cfg.Logger.FilePath, cfg.Logger.LogLevel, cfg.Logger.LogToFile)

	app, cleanup, err := di.InitializeApp(cfg, l)
	if err != nil {
		l.Error("Application bootstrap failed", zap.Error(err))
		return
	}

	defer func() {
		fmt.Println("Cleaning up resources...")
		cleanup()
		fmt.Println("Graceful shutdown completed successfully.")
	}()

	if err := app.Run(); err != nil {
		l.Fatal("Application crashed while running", zap.Error(err))
	}
}
