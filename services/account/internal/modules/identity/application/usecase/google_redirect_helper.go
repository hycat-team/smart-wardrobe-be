package usecase

import (
	"net/url"
	"strings"

	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	"smart-wardrobe-be/shared/config"
)

// validateGoogleRedirectURL ensures raw points at one of the configured frontend
// origins. It parses the URL and compares scheme + hostname exactly (no
// subdomains), which rejects userinfo tricks (https://origin@evil.com) and
// suffix spoofing (origin.hehe). The return target is preserved verbatim
// (path/query/fragment allowed).
func validateGoogleRedirectURL(cfg *config.Config, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", identityerrors.ErrGoogleRedirectInvalid()
	}

	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return "", identityerrors.ErrGoogleRedirectInvalid()
	}

	host := normalizeHostname(u.Hostname())
	if _, ok := allowedOriginHosts(cfg.Server.FrontEndOrigins)[host]; !ok {
		return "", identityerrors.ErrGoogleRedirectInvalid()
	}

	isDev := !config.IsProductionEnv(cfg.Server.Env)
	switch strings.ToLower(u.Scheme) {
	case "https":
	case "http":
		if !isDev || (host != "localhost" && host != "127.0.0.1") {
			return "", identityerrors.ErrGoogleRedirectInvalid()
		}
	default:
		return "", identityerrors.ErrGoogleRedirectInvalid()
	}

	return u.String(), nil
}

func normalizeHostname(host string) string {
	return strings.TrimRight(strings.ToLower(strings.TrimSpace(host)), ".")
}

func allowedOriginHosts(origins []string) map[string]struct{} {
	set := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || parsed.Host == "" {
			continue
		}
		set[normalizeHostname(parsed.Hostname())] = struct{}{}
	}
	return set
}
