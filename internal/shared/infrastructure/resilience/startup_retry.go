package resilience

import (
	"fmt"
	"smart-wardrobe-be/config"
	"time"

	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type RetryAttempt struct {
	Timeout   time.Duration
	WaitAfter time.Duration
}

func RunStartupRetry(cfg *config.Config, l logger.Interface, dependencyName string, operation func(timeout time.Duration) error) error {
	startupRetryPlan := []RetryAttempt{
		{
			Timeout:   time.Duration(cfg.Startup.RetryAttempt1Seconds) * time.Second,
			WaitAfter: time.Duration(cfg.Startup.RetryAttempt1Seconds) * time.Second,
		},
		{
			Timeout:   time.Duration(cfg.Startup.RetryAttempt2Seconds) * time.Second,
			WaitAfter: time.Duration(cfg.Startup.RetryAttempt2Seconds) * time.Second,
		},
		{
			Timeout:   time.Duration(cfg.Startup.RetryAttempt3Seconds) * time.Second,
			WaitAfter: 0,
		},
	}

	var lastErr error

	for idx, attempt := range startupRetryPlan {
		attemptNumber := idx + 1

		l.Info("Initializing dependency connection",
			zap.String("dependency", dependencyName),
			zap.Int("attempt", attemptNumber),
			zap.Int("max_attempts", len(startupRetryPlan)),
			zap.Duration("timeout", attempt.Timeout),
		)

		if err := operation(attempt.Timeout); err == nil {
			if attemptNumber > 1 {
				l.Info("Dependency connection recovered successfully",
					zap.String("dependency", dependencyName),
					zap.Int("attempt", attemptNumber),
				)
			}
			return nil
		} else {
			lastErr = err
			l.Warn("Dependency connection attempt failed",
				zap.String("dependency", dependencyName),
				zap.Int("attempt", attemptNumber),
				zap.Int("max_attempts", len(startupRetryPlan)),
				zap.Duration("timeout", attempt.Timeout),
				zap.Error(err),
			)
		}

		if attempt.WaitAfter > 0 {
			time.Sleep(attempt.WaitAfter)
		}
	}

	return fmt.Errorf("%s initialization failed after %d attempts: %w", dependencyName, len(startupRetryPlan), lastErr)
}
