package worker

import (
	"context"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_worker "smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

// CampaignGrantWorker consumes `identity.user.created` and hands each message to
// the campaign use case (025 T054).
//
// A THIN ADAPTER, on purpose. It unmarshals, logs, delegates, and classifies
// nothing. Every rule — which campaign, whether the account is eligible, whether
// the budget allows it — lives in the use case, and the catch-up sweep of Phase 10
// will call the same use case. A second decision path in this file would be a
// second set of answers to the same question, which is how the two drift apart
// (FR-080).
//
// The queue is named for the RECEIVER, not for the topic and not for the campaign
// (R12, FR-090). That is what lets a future second consumer of the same topic be
// added as a new queue without touching this one — the whole point of FR-090 being
// that adding a listener does not mean editing the registration flow.
type CampaignGrantWorker struct {
	*shared_worker.BaseWorker
	messageClient event.IMessageConsumer
	campaignUC    uc_interfaces.ICampaignUseCase
	log           logger.Interface
}

func NewCampaignGrantWorker(
	messageClient event.IMessageConsumer,
	campaignUC uc_interfaces.ICampaignUseCase,
	l logger.Interface,
) *CampaignGrantWorker {
	return &CampaignGrantWorker{
		BaseWorker:    shared_worker.NewBaseWorker("campaign_grant"),
		messageClient: messageClient,
		campaignUC:    campaignUC,
		log:           l,
	}
}

func (w *CampaignGrantWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	// `ConsumerOpts{}` means "take Prefetch and Concurrency from the Topology
	// registry" (Prefetch 20, Concurrency 1 — FR-089). Passing them here instead
	// would create a second place where a queue's concurrency is configured, and
	// the two would disagree the first time somebody tuned one of them.
	//
	// The handler returns the use case's error UNCHANGED, so its classification
	// survives: a `PermanentError` is dead-lettered and a `RetryableError` is
	// requeued (FR-062). Wrapping it here would collapse the two and lose a
	// customer who was, in fact, eligible.
	if err := event.RegisterTypedConsumer(
		w.messageClient,
		eventconstants.QueueCampaignGrantUserCreated,
		w.handle,
		event.ConsumerOpts{},
	); err != nil {
		w.log.Error("Failed to register campaign grant consumer", zap.Error(err))
	}
}

// handle processes one account-creation message.
func (w *CampaignGrantWorker) handle(ctx context.Context, payload identitydto.UserCreatedEventPayload) error {
	run := workerlog.New("campaign_grant", workerlog.TriggerQueue)
	run.LogReceived(w.log,
		// FR-064: the correlation id ties this line to the line the registration
		// wrote, so a skipped account can be traced back to the request that
		// created it. It comes from the queue row, injected by the framework.
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("userId", payload.UserID.String()),
	)

	if err := w.campaignUC.ProcessUserCreated(ctx, payload); err != nil {
		run.LogFailure(w.log, err,
			zap.String("userId", payload.UserID.String()),
		)
		return err
	}

	run.AddTotal(1)
	run.AddSuccess(1)
	run.LogSuccessIfData(w.log, zap.String("userId", payload.UserID.String()))
	return nil
}

func (w *CampaignGrantWorker) Stop() {
	w.BaseWorker.Stop()
}
