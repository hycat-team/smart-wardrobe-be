// Package providers exposes the Commerce Core service's composition surface
// for the transitional monolith (internal/di, internal/bootstrap, internal/api).
//
// It re-exports the service's wire ProviderSets plus the concrete router and
// worker types the monolith composes. It MUST NOT import implementation
// packages of other services. Removed/replaced by per-service composition
// (T041) when cmd/server leaves main at T034.
package providers

import (
	brandmodule "smart-wardrobe-be/services/commerce/internal/modules/brand"
	brandrouter "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/router"
	brandworker "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/worker"
	dashboardmodule "smart-wardrobe-be/services/commerce/internal/modules/dashboard"
	dashboardrouter "smart-wardrobe-be/services/commerce/internal/modules/dashboard/presentation/router"
	dashboardworker "smart-wardrobe-be/services/commerce/internal/modules/dashboard/presentation/worker"
	fashionmodule "smart-wardrobe-be/services/commerce/internal/modules/fashion"
	fashionrouter "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/router"
	fashionworker "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/worker"
	wardrobemodule "smart-wardrobe-be/services/commerce/internal/modules/wardrobe"
	wardroberouter "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/router"
	wardrobeworker "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/worker"

	"github.com/google/wire"
)

// ProviderSet aggregates the Commerce Core modules (wardrobe, fashion,
// brand, dashboard).
var ProviderSet = wire.NewSet(
	wardrobemodule.ProviderSet,
	brandmodule.ProviderSet,
	fashionmodule.ProviderSet,
	dashboardmodule.ProviderSet,
)

// Router aliases for monolith composition.
type (
	WardrobeRouter  = wardroberouter.Router
	DashboardRouter = dashboardrouter.Router
	BrandRouter     = brandrouter.Router
	FashionRouter   = fashionrouter.Router
)

// Worker aliases for monolith composition.
type (
	LoyaltyPointExpiryWorker               = brandworker.LoyaltyPointExpiryWorker
	SearchSyncWorker                       = wardrobeworker.SearchSyncWorker
	FashionInitWorker                      = fashionworker.FashionInitWorker
	FailedItemsCleanupWorker               = fashionworker.FailedItemsCleanupWorker
	ProcessingRecoveryWorker               = fashionworker.ProcessingRecoveryWorker
	FashionAnalyzeWorker                   = fashionworker.FashionAnalyzeWorker
	DashboardAggregatorWorker              = dashboardworker.DashboardAggregatorWorker
	WardrobeFashionEventWorker             = wardrobeworker.WardrobeFashionEventWorker
	BrandFashionEventWorker                = brandworker.BrandFashionEventWorker
	WardrobeCleanupWorker                  = wardrobeworker.WardrobeCleanupWorker
	WardrobeProcessingExhaustedWorker      = wardrobeworker.WardrobeProcessingExhaustedWorker
	WardrobeFashionBindWorker              = wardrobeworker.WardrobeFashionBindWorker
	ElasticsearchProcessingExhaustedWorker = wardrobeworker.ElasticsearchProcessingExhaustedWorker
	BrandFashionBindWorker                 = brandworker.BrandFashionBindWorker
)

// CommerceBundle is the composed product of the Commerce Core service.
type CommerceBundle struct {
	WardrobeRouter  *WardrobeRouter
	DashboardRouter *DashboardRouter
	BrandRouter     *BrandRouter
	FashionRouter   *FashionRouter

	LoyaltyPointExpiryWorker               *LoyaltyPointExpiryWorker
	SearchSyncWorker                       *SearchSyncWorker
	FashionInitWorker                      *FashionInitWorker
	FailedItemsCleanupWorker               *FailedItemsCleanupWorker
	ProcessingRecoveryWorker               *ProcessingRecoveryWorker
	FashionAnalyzeWorker                   *FashionAnalyzeWorker
	DashboardAggregatorWorker              *DashboardAggregatorWorker
	WardrobeFashionEventWorker             *WardrobeFashionEventWorker
	BrandFashionEventWorker                *BrandFashionEventWorker
	WardrobeCleanupWorker                  *WardrobeCleanupWorker
	WardrobeProcessingExhaustedWorker      *WardrobeProcessingExhaustedWorker
	WardrobeFashionBindWorker              *WardrobeFashionBindWorker
	ElasticsearchProcessingExhaustedWorker *ElasticsearchProcessingExhaustedWorker
	BrandFashionBindWorker                 *BrandFashionBindWorker
}
