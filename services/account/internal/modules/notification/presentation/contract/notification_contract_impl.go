package contract

import (
	"context"

	usecase_interfaces "smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"

	"github.com/google/uuid"
)

// notificationContractImpl delegates to the application outbox email use
// case; presentation must not touch infrastructure directly (§IV, feature 017).
type notificationContractImpl struct {
	emailUC usecase_interfaces.IOutboxEmailUseCase
}

func NewNotificationContract(
	emailUC usecase_interfaces.IOutboxEmailUseCase,
) INotificationContract {
	return &notificationContractImpl{
		emailUC: emailUC,
	}
}

func (c *notificationContractImpl) Send(ctx context.Context, channel ChannelEnum, tmplEnum TemplateEnum, params map[string]any, to string) error {
	return c.emailUC.Send(ctx, channel, tmplEnum, params, to)
}

func (c *notificationContractImpl) SendByUserID(ctx context.Context, channel ChannelEnum, tmplEnum TemplateEnum, params map[string]any, userID uuid.UUID) error {
	return c.emailUC.SendByUserID(ctx, channel, tmplEnum, params, userID)
}
