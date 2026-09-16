package notification

import (
	"smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/notification/application/usecase/email"
	"smart-wardrobe-be/services/account/internal/modules/notification/infrastructure/communication"
	"smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/notification/presentation/worker"

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
