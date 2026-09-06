package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/notification/domain/constants/channel"
	"smart-wardrobe-be/internal/modules/notification/domain/constants/template"

	"github.com/google/uuid"
)

// IOutboxEmailUseCase là boundary application layer cho việc soạn và ghi
// email vào transactional outbox (feature 017). Contract của module chỉ
// delegate xuống usecase này — không đụng infrastructure trực tiếp.
type IOutboxEmailUseCase interface {
	Send(ctx context.Context, channel channel.Channel, template template.Template, params map[string]any, to string) error
	SendByUserID(ctx context.Context, channel channel.Channel, template template.Template, params map[string]any, userID uuid.UUID) error
}
