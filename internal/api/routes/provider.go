package routes

import (
	admin_router "smart-wardrobe-be/internal/modules/admin/presentation/router"
	aiexecutor_router "smart-wardrobe-be/internal/modules/aiexecutor/presentation/router"
	brand_router "smart-wardrobe-be/internal/modules/brand/presentation/router"
	dashboard_router "smart-wardrobe-be/internal/modules/dashboard/presentation/router"
	fashion_router "smart-wardrobe-be/internal/modules/fashion/presentation/router"
	identity_router "smart-wardrobe-be/internal/modules/identity/presentation/router"
	subscription_router "smart-wardrobe-be/internal/modules/subscription/presentation/router"
	wardrobe_router "smart-wardrobe-be/internal/modules/wardrobe/presentation/router"

	"github.com/google/wire"
)

type AppRouter struct {
	IdentityRouter     *identity_router.Router
	SubscriptionRouter *subscription_router.Router
	WardrobeRouter     *wardrobe_router.Router
	DashboardRouter    *dashboard_router.Router
	BrandRouter        *brand_router.Router
	FashionRouter      *fashion_router.Router
	AIExecutorRouter   *aiexecutor_router.Router
	AdminRouter        *admin_router.Router
}

var RouterSet = wire.NewSet(
	wire.Struct(new(AppRouter), "*"),
)
