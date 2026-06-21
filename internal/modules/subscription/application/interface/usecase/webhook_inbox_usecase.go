package usecase

import (
	"context"
)

type IWebhookInboxUseCase interface {
	ProcessInbox(ctx context.Context) error
}
