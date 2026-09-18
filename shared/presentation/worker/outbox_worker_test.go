package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	usecase "smart-wardrobe-be/shared/application/interface/usecase"
	"smart-wardrobe-be/shared/config"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubLogger satisfies logger.Interface without emitting anything.
type stubLogger struct{}

func (stubLogger) Debug(string, ...zap.Field) {}
func (stubLogger) Info(string, ...zap.Field)  {}
func (stubLogger) Warn(string, ...zap.Field)  {}
func (stubLogger) Error(string, ...zap.Field) {}
func (stubLogger) Fatal(string, ...zap.Field) {}

// quietListener never signals, like a pooler-degraded OutboxListener.
type quietListener struct {
	ch chan struct{}
}

func (q quietListener) SignalChan() <-chan struct{} { return q.ch }

// countingOutboxUC records PublishBatch invocations.
type countingOutboxUC struct {
	usecase.IOutboxUseCase
	published atomic.Int32
}

func (m *countingOutboxUC) PublishBatch(ctx context.Context, batchSize int, retryMax int, retryAfter time.Duration) (int, error) {
	m.published.Add(1)
	return 0, nil
}

func (m *countingOutboxUC) CleanupPublished(ctx context.Context, retentionDays int) (int64, error) {
	return 0, nil
}

func (m *countingOutboxUC) RefreshGauges(ctx context.Context) {}

// TestOutboxWorkerPollsWithoutSignals proves FR-010: with a signal-less
// listener (LISTEN unavailable), the 2s poll ticker alone keeps publishing.
func TestOutboxWorkerPollsWithoutSignals(t *testing.T) {
	uc := &countingOutboxUC{}
	cfg := &config.Config{}
	cfg.Outbox.BatchSize = 50
	cfg.Outbox.RetryIntervalMinutes = 1
	cfg.Outbox.CleanupIntervalMinutes = 60

	w := NewOutboxWorker(uc, quietListener{ch: make(chan struct{}, 1)}, cfg, stubLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	deadline := time.Now().Add(6 * time.Second)
	for uc.published.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	require.Greater(t, uc.published.Load(), int32(0), "poll ticker must drive PublishBatch without LISTEN signals")
}
