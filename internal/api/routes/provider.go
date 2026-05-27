package routes

import (
	"smart-wardrobe-be/internal/api/routes/auth"
	"smart-wardrobe-be/internal/api/routes/me"

	"github.com/google/wire"
)

type AppRouter struct {
	AuthRouter *auth.AuthRouter
	MeRouter   *me.MeRouter
}

var RouterSet = wire.NewSet(
	auth.NewRouter,
	me.NewRouter,
)
