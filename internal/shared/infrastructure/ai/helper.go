package ai

import (
	"bufio"
	"io"
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

// parseSSEStream reads SSE stream data from reader and invokes callback for each line starting with "data:"
func parseSSEStream(reader io.Reader, onData func(data string) error) error {
	bufReader := bufio.NewReader(reader)
	for {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)
		if data == "" {
			continue
		}

		if err := onData(data); err != nil {
			return err
		}
	}
}
