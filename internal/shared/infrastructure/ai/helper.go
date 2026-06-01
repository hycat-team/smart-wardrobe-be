package ai

import (
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/pkg/logger"
	"strings"

	"go.uber.org/zap"
)

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
			)

			fallbackResult, fallbackErr := fallbackFn()
			if fallbackErr != nil {
				logger.Error("Both primary and fallback providers failed",
					zap.String("operation", operation),
					zap.Error(fallbackErr),
				)
				var zero T
				return zero, fallbackErr
			}

			return fallbackResult, nil
		}

		logger.Error("Provider failed permanently without fallback",
			zap.String("operation", operation),
			zap.Error(err),
		)
		var zero T
		return zero, err
	}

	return result, nil
}
