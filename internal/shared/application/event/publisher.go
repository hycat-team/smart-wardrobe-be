package event

import "context"

type IEventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}
