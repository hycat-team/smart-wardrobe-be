package routes

import (
	// 020: router types via per-service providers (monolith composition
	// until cmd/server leaves main at T034).
	accountproviders "smart-wardrobe-be/services/account/providers"
	aiproviders "smart-wardrobe-be/services/ai/providers"
	commerceproviders "smart-wardrobe-be/services/commerce/providers"
	shared_search "smart-wardrobe-be/shared/infrastructure/search"

	"github.com/google/wire"
)

type AppRouter struct {
	IdentityRouter     *accountproviders.IdentityRouter
	SubscriptionRouter *accountproviders.SubscriptionRouter
	WardrobeRouter     *commerceproviders.WardrobeRouter
	DashboardRouter    *commerceproviders.DashboardRouter
	BrandRouter        *commerceproviders.BrandRouter
	FashionRouter      *commerceproviders.FashionRouter
	AIExecutorRouter   *aiproviders.AIExecutorRouter
	AdminRouter        *accountproviders.AdminRouter
	// SearchClient exposes engine status on the health endpoint (020, T018).
	SearchClient *shared_search.ElasticsearchClient
}

var RouterSet = wire.NewSet(
	wire.Struct(new(AppRouter), "*"),
)
