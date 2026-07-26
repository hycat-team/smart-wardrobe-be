package notification

import (
	"smart-wardrobe-be/internal/modules/notification/infrastructure/communication"
	"smart-wardrobe-be/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/internal/modules/notification/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	communication.NewSmtpSender,
	contract.NewNotificationContract,
	worker.NewEmailWorker,
)
