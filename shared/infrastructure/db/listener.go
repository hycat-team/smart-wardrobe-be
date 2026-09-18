package db

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const outboxChannel = "outbox_channel"
const reconnectBackoff = 2 * time.Second

type OutboxListener struct {
	connStr    string
	logger     logger.Interface
	signalChan chan struct{}
	cancel     context.CancelFunc
}

func NewOutboxListener(cfg *config.Config, l logger.Interface) (*OutboxListener, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// FR-009/FR-010: an unavailable LISTEN channel (pooler, network, or
	// explicit disable) must NEVER fail boot. The worker always polls every
	// 2s (see OutboxWorker.runLoop), so a signal-less listener only loses
	// wake-up latency, never events.
	if !cfg.Outbox.ListenerEnabled {
		l.Info("Outbox LISTEN disabled by configuration — running in poll mode")
		return &OutboxListener{
			logger:     l,
			signalChan: make(chan struct{}, 1),
			cancel:     cancel,
		}, nil
	}

	// Session-scoped direct DSN (FR-046): LISTEN/NOTIFY, migration and advisory
	// locks MUST NOT go through the transaction pooler. No search_path here:
	// LISTEN needs none, and poolers ignore/deny client startup parameters.
	connStr := DirectDSN(cfg)

	ol := &OutboxListener{
		connStr:    connStr,
		logger:     l,
		signalChan: make(chan struct{}, 1),
		cancel:     cancel,
	}

	if err := ol.connectAndListen(ctx); err != nil {
		l.Warn("Outbox LISTEN unavailable at boot — running in poll mode",
			zap.String("channel", outboxChannel),
			zap.Error(err),
		)
		return ol, nil
	}

	ol.logger.Info("Outbox listener started", zap.String("channel", outboxChannel))
	return ol, nil
}

func (ol *OutboxListener) SignalChan() <-chan struct{} {
	return ol.signalChan
}

func (ol *OutboxListener) Stop() {
	ol.cancel()
}

func (ol *OutboxListener) connectAndListen(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, ol.connStr)
	if err != nil {
		return fmt.Errorf("pgx connect: %w", err)
	}

	if _, err := conn.Exec(ctx, "LISTEN "+outboxChannel); err != nil {
		conn.Close(ctx)
		return fmt.Errorf("LISTEN %s: %w", outboxChannel, err)
	}

	go ol.notificationLoop(ctx, conn)
	return nil
}

func (ol *OutboxListener) notificationLoop(ctx context.Context, conn *pgx.Conn) {
	defer conn.Close(ctx)

	for {
		_, err := conn.WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			ol.logger.Warn("Outbox listener disconnected, reconnecting...", zap.Error(err))
			ol.reconnect(ctx)
			return
		}
		select {
		case ol.signalChan <- struct{}{}:
		default:
		}
	}
}

func (ol *OutboxListener) reconnect(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectBackoff):
		}

		if err := ol.connectAndListen(ctx); err == nil {
			ol.logger.Info("Outbox listener reconnected")
			return
		}
		ol.logger.Warn("Outbox listener reconnect failed, retrying with fallback poll")
	}
}
