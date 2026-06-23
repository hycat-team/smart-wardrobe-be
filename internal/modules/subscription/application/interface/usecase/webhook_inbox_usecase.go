package usecase

import (
	"context"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
)

type IWebhookInboxUseCase interface {
	ProcessInbox(ctx context.Context, run *workerlog.Run) error
}
