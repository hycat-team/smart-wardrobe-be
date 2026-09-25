package bootstrap

import (
	"context"

	brandworker "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/worker"
	communityworker "smart-wardrobe-be/services/commerce/internal/modules/community/presentation/worker"
	dashboardworker "smart-wardrobe-be/services/commerce/internal/modules/dashboard/presentation/worker"
	fashionworker "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/worker"
	wardrobeworker "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/worker"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"
)

// Worker is the lifecycle contract every background worker satisfies through
// the shared BaseWorker (FR-044/FR-045).
type Worker interface {
	Start(ctx context.Context)
	Stop()
}

// Workers groups the Commerce Core background workers. They run only in
// APP_MODE=worker|all; APP_MODE=http never starts them (FR-044, SC-018).
// Commerce hosts no gRPC service.
type Workers struct {
	LoyaltyPointExpiry               *brandworker.LoyaltyPointExpiryWorker
	SearchSync                       *wardrobeworker.SearchSyncWorker
	FashionInit                      *fashionworker.FashionInitWorker
	FailedItemsCleanup               *fashionworker.FailedItemsCleanupWorker
	ProcessingRecovery               *fashionworker.ProcessingRecoveryWorker
	FashionAnalyze                   *fashionworker.FashionAnalyzeWorker
	DashboardAggregator              *dashboardworker.DashboardAggregatorWorker
	WardrobeFashionEvent             *wardrobeworker.WardrobeFashionEventWorker
	BrandFashionEvent                *brandworker.BrandFashionEventWorker
	WardrobeCleanup                  *wardrobeworker.WardrobeCleanupWorker
	WardrobeProcessingExhausted      *wardrobeworker.WardrobeProcessingExhaustedWorker
	WardrobeFashionBind              *wardrobeworker.WardrobeFashionBindWorker
	ElasticsearchProcessingExhausted *wardrobeworker.ElasticsearchProcessingExhaustedWorker
	BrandFashionBind                 *brandworker.BrandFashionBindWorker
	CommunityOutfitCascade           *communityworker.OutfitCascadeWorker
	CommunityUserStatus              *communityworker.UserStatusWorker
	CommunityFollowStatsReconcile    *communityworker.FollowStatsReconcileWorker
	Outbox                           *sharedWorker.OutboxWorker
}

// All returns the workers in a stable start/stop order.
func (w *Workers) All() []Worker {
	if w == nil {
		return nil
	}
	return []Worker{
		w.LoyaltyPointExpiry,
		w.SearchSync,
		w.FashionInit,
		w.FailedItemsCleanup,
		w.ProcessingRecovery,
		w.FashionAnalyze,
		w.DashboardAggregator,
		w.WardrobeFashionEvent,
		w.BrandFashionEvent,
		w.WardrobeCleanup,
		w.WardrobeProcessingExhausted,
		w.WardrobeFashionBind,
		w.ElasticsearchProcessingExhausted,
		w.BrandFashionBind,
		w.CommunityOutfitCascade,
		w.CommunityUserStatus,
		w.CommunityFollowStatsReconcile,
		w.Outbox,
	}
}
