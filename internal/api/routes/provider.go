package routes

import (
	"smart-wardrobe-be/internal/api/routes/auth"
	"smart-wardrobe-be/internal/api/routes/me"
	"smart-wardrobe-be/internal/api/routes/subscription"

	"github.com/google/wire"
)

type AppRouter struct {
	AuthRouter         *auth.AuthRouter
	MeRouter           *me.MeRouter
	SubscriptionRouter *subscription.SubscriptionRouter
}

var RouterSet = wire.NewSet(
	auth.NewRouter,
	me.NewRouter,
	subscription.NewRouter,
)
