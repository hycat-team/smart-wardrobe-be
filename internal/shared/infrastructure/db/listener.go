package db

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/pkg/logger"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const outboxChannel = "outbox_channel"
const reconnectBackoff = 2 * time.Second
const maxReconnectAttempts = 3

type OutboxListener struct {
	connStr    string
	logger     logger.Interface
	signalChan chan struct{}
	cancel     context.CancelFunc
}

func NewOutboxListener(cfg *config.Config, l logger.Interface) (*OutboxListener, error) {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DbName,
		cfg.Database.Port,
		cfg.Database.SslMode,
		cfg.Database.TimeZone,
	)

	ol := &OutboxListener{
		connStr:    connStr,
		logger:     l,
		signalChan: make(chan struct{}, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	ol.cancel = cancel

	if err := ol.connectAndListen(ctx); err != nil {
		return nil, fmt.Errorf("failed to establish outbox listener: %w", err)
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
	for i := 0; i < maxReconnectAttempts; i++ {
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectBackoff):
		}

		if err := ol.connectAndListen(ctx); err == nil {
			ol.logger.Info("Outbox listener reconnected")
			return
		}
	}
	ol.logger.Error("Outbox listener: all reconnect attempts exhausted")
}
