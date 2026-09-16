// Command commerce is the entry point of the Commerce Core service.
//
// Supported modes (see FR-044): APP_MODE=http|worker|all (default all).
// The service listens on PORT (Cloud Run injects it), default 8080,
// and always serves GET /api/v1/health for startup/liveness probes.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	_ "time/tzdata" // FR-050: embed tzdata so distroless static nonroot images resolve Asia/Ho_Chi_Minh.
)

const serviceName = "commerce"

func main() {
	healthcheck := flag.Bool("healthcheck", false, "Probe /api/v1/health and exit (used by Dockerfile HEALTHCHECK).")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if *healthcheck {
		resp, err := http.Get("http://localhost:" + port + "/api/v1/health") //nolint:noctx,bodyclose
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": serviceName})
	})

	addr := ":" + port
	fmt.Printf("%s service listening on %s (APP_MODE=%s)\n", serviceName, addr, appMode())
	log.Fatal(http.ListenAndServe(addr, mux))
}

func appMode() string {
	if m := os.Getenv("APP_MODE"); m != "" {
		return m
	}
	return "all"
}
