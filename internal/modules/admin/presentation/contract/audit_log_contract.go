package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/admin/application/dto"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
)

type IAuditLogContract interface {
	Record(ctx context.Context, input dto.RecordAuditLogInput) error
}

type AuditLogContract struct {
	publisher event.IEventPublisher
}

func NewAuditLogContract(publisher event.IEventPublisher) IAuditLogContract {
	return &AuditLogContract{
		publisher: publisher,
	}
}

func (c *AuditLogContract) Record(ctx context.Context, input dto.RecordAuditLogInput) error {
	// Publish asynchronous event via transactional outbox
	return c.publisher.Publish(ctx, eventconstants.TopicAuditLog, input)
}
