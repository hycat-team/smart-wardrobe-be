package ai

import (
	"errors"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"
	"strings"

	"go.uber.org/zap"
)

// rootCause returns the deepest error in the chain.
//
// apperror keeps the provider failure in Cause while Error() returns only the
// client-safe detail ("Dịch vụ đang bận, vui lòng thử lại sau ít phút."). A log
// built from err.Error() alone therefore hides the real reason, which is how a
// retired Gemini model (HTTP 404 "no longer available") looked like a busy
// service. Logging the deepest cause keeps that reason visible without changing
// what the client receives.
//
// The depth cap is a guard against a self-referencing chain, which a hand-built
// error could produce; real chains are shallow.
func rootCause(err error) error {
	deepest := err
	for depth := 0; depth < 8; depth++ {
		next := errors.Unwrap(deepest)
		if next == nil {
			break
		}
		deepest = next
	}
	return deepest
}

func executeWithFallback[T any](
	logger logger.Interface,
	operation string,
	fallbackCfg config.APIProviderConfig,
	primaryFn func() (T, error),
	fallbackFn func() (T, error),
) (T, error) {
	result, err := primaryFn()
	if err != nil {
		isBadRequest := strings.Contains(err.Error(), "HTTP 400") || strings.Contains(err.Error(), "400 Bad Request")
		hasFallback := fallbackCfg.Provider != ""

		if !isBadRequest && hasFallback {
			logger.Warn("Primary provider failed, switching to fallback",
				zap.String("operation", operation),
				zap.Error(err),
				zap.NamedError("cause", rootCause(err)),
			)

			fallbackResult, fallbackErr := fallbackFn()
			if fallbackErr != nil {
				logger.Error("Both primary and fallback providers failed",
					zap.String("operation", operation),
					zap.Error(fallbackErr),
					zap.NamedError("cause", rootCause(fallbackErr)),
				)
				var zero T
				return zero, fallbackErr
			}

			return fallbackResult, nil
		}

		logger.Error("Provider failed permanently without fallback",
			zap.String("operation", operation),
			zap.Error(err),
			zap.NamedError("cause", rootCause(err)),
		)
		var zero T
		return zero, err
	}

	return result, nil
}
