package messaging

import (
	"context"

	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

// OutboxGaugeProbe implements event.IOutboxGaugeProbe on the infrastructure
// side: it fetches queue depths from the RabbitMQ management API and stores
// them together with the pending count into the operational gauges
// (feature 017).
type OutboxGaugeProbe struct {
	cfg    *config.Config
	logger logger.Interface
}

func NewOutboxGaugeProbe(cfg *config.Config, l logger.Interface) *OutboxGaugeProbe {
	return &OutboxGaugeProbe{cfg: cfg, logger: l}
}

func (p *OutboxGaugeProbe) ProbeGauges(ctx context.Context, pending int64) {
	mgmtPort := p.cfg.RabbitMQ.ManagementPort
	if mgmtPort <= 0 {
		mgmtPort = 15672
	}
	queueDepth, dlqDepth, err := FetchQueueDepths(ctx, p.cfg.RabbitMQ.Host, mgmtPort,
		p.cfg.RabbitMQ.User, p.cfg.RabbitMQ.Password, p.logger)
	if err != nil {
		p.logger.Warn("[OutboxGaugeProbe] Failed to fetch queue depths from management API", zap.Error(err))
	}
	SetGauges(pending, queueDepth, dlqDepth)
}
