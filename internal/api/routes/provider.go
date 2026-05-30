package routes

import (
	"smart-wardrobe-be/internal/api/routes/auth"
	"smart-wardrobe-be/internal/api/routes/me"
	"smart-wardrobe-be/internal/api/routes/outfit"
	"smart-wardrobe-be/internal/api/routes/subscription"
	"smart-wardrobe-be/internal/api/routes/wardrobe"

	"github.com/google/wire"
)

type AppRouter struct {
	AuthRouter         *auth.AuthRouter
	MeRouter           *me.MeRouter
	SubscriptionRouter *subscription.SubscriptionRouter
	WardrobeRouter     *wardrobe.WardrobeRouter
	OutfitRouter       *outfit.OutfitRouter
}

var RouterSet = wire.NewSet(
	auth.NewRouter,
	me.NewRouter,
	subscription.NewRouter,
	wardrobe.NewRouter,
	outfit.NewRouter,
)
