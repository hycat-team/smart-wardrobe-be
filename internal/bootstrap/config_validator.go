package bootstrap

import (
	"fmt"
	"net/http"
	"strings"

	"smart-wardrobe-be/shared/config"
)

var bodyLimitFreeMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// ValidateServerHardening enforces that connection timeouts and body size
// limits are explicitly configured before the server accepts traffic in
// production — zero-value (unlimited) settings must never reach the internet.
// Non-production environments stay permissive so local dev works with minimal config.
func ValidateServerHardening(cfg *config.Config) error {
	if cfg.Server.Env != "production" {
		return nil
	}

	if cfg.Server.TimeoutSeconds <= 0 {
		// GlobalTimeoutMiddleware wraps every request in context.WithTimeout;
		// a zero value would expire each request context immediately.
		return fmt.Errorf("server.timeout_seconds must be > 0 in production")
	}
	if cfg.Server.ReadTimeoutSeconds <= 0 {
		return fmt.Errorf("server.read_timeout_seconds must be > 0 in production")
	}
	if cfg.Server.WriteTimeoutSeconds <= 0 {
		return fmt.Errorf("server.write_timeout_seconds must be > 0 in production")
	}
	if cfg.Server.IdleTimeoutSeconds <= 0 {
		return fmt.Errorf("server.idle_timeout_seconds must be > 0 in production")
	}

	if cfg.Server.BodyLimit.DefaultMB <= 0 {
		return fmt.Errorf("server.body_limit.default_mb must be > 0 in production")
	}
	if cfg.Server.BodyLimit.ElevatedMB < cfg.Server.BodyLimit.DefaultMB {
		return fmt.Errorf("server.body_limit.elevated_mb (%d) must be >= default_mb (%d)",
			cfg.Server.BodyLimit.ElevatedMB, cfg.Server.BodyLimit.DefaultMB)
	}
	for i, route := range cfg.Server.BodyLimit.FreeRoutes {
		parts := strings.Fields(route)
		if len(parts) != 2 || !bodyLimitFreeMethods[parts[0]] || !strings.HasPrefix(parts[1], "/") {
			return fmt.Errorf("server.body_limit.free_routes[%d] must follow the `METHOD /path-prefix` format, got %q", i, route)
		}
	}
	return nil
}
