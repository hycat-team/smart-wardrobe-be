package notification

import (
	"smart-wardrobe-be/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/notification/application/usecase/email"
	"smart-wardrobe-be/internal/modules/notification/infrastructure/communication"
	"smart-wardrobe-be/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/internal/modules/notification/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	communication.NewSmtpSender,
	wire.Bind(new(usecase.IEmailSender), new(*communication.SmtpSender)),
	contract.NewNotificationContract,
	email.NewEmailUseCase,
	email.NewOutboxEmailUseCase,
	worker.NewEmailWorker,
)
