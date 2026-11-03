package notification

import (
	"smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/notification/application/usecase/email"
	"smart-wardrobe-be/services/account/internal/modules/notification/infrastructure/caching"
	"smart-wardrobe-be/services/account/internal/modules/notification/infrastructure/communication"
	"smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/notification/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	communication.NewSmtpSender,
	wire.Bind(new(usecase.IEmailSender), new(*communication.SmtpSender)),
	// Redis-backed dedupe for the email worker. ICacheService is provided by the
	// service composition root; event.IRawMessagePublisher is bound there too.
	caching.NewRedisDeliveryDeduplicator,
	wire.Bind(new(usecase.IDeliveryDeduplicator), new(*caching.RedisDeliveryDeduplicator)),
	contract.NewNotificationContract,
	email.NewEmailUseCase,
	email.NewOutboxEmailUseCase,
	worker.NewEmailWorker,
)
