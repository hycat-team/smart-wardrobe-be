// Command account is the entry point of the Account service.
//
// Supported modes (FR-044): APP_MODE=http|worker|all (default all).
// The service listens on PORT (Cloud Run injects it), falling back to
// SERVER_PORT (local default lives in configs/config.yaml server.port),
// and always serves GET /api/v1/health for startup/liveness probes.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	_ "time/tzdata" // FR-050: embed tzdata so distroless static nonroot images resolve Asia/Ho_Chi_Minh.

	"smart-wardrobe-be/services/account/internal/di"
	sharedconfig "smart-wardrobe-be/shared/config"
	sharedlogger "smart-wardrobe-be/shared/pkg/logger"
)

// @title           Smart Wardrobe - Account Service API
// @version         1.0
// @description     API docs for the Account service (identity, subscription, admin, notification).
func main() {
	healthcheck := flag.Bool("healthcheck", false, "Probe /api/v1/health and exit (used by Dockerfile HEALTHCHECK).")
	flag.Parse()

	port := listenPort()
	if port == "" {
		log.Fatal("PORT or SERVER_PORT must be set (local default lives in configs/config.yaml server.port)")
	}

	if *healthcheck {
		probeHealth(port)
		return
	}

	cfg := sharedconfig.LoadConfig()
	cfg.Server.Port = port
	appLogger := sharedlogger.New(cfg.Server.Env, cfg.Logger.FilePath, cfg.Logger.LogLevel, cfg.Logger.LogToFile)

	app, cleanup, err := di.InitializeApp(cfg, appLogger)
	if err != nil {
		log.Fatalf("failed to initialize account service: %v", err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		log.Fatalf("account service stopped: %v", err)
	}
}

// listenPort resolves the platform port: PORT wins (Cloud Run), then
// SERVER_PORT for local/VPS (FR-049/SC-023).
func listenPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return os.Getenv("SERVER_PORT")
}

func probeHealth(port string) {
	resp, err := http.Get("http://localhost:" + port + "/api/v1/health") //nolint:noctx,bodyclose
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}
