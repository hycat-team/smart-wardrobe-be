package bootstrap

import (
	admingrpc "smart-wardrobe-be/services/account/internal/modules/admin/presentation/grpc"
	identitygrpc "smart-wardrobe-be/services/account/internal/modules/identity/presentation/grpc"
	subscriptiongrpc "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/grpc"

	adminv1 "smart-wardrobe-be/contracts/admin/v1"
	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"

	"google.golang.org/grpc"
)

// NewGRPCServer registers the Account-hosted gRPC services (Identity,
// Subscription, Admin) on one server shared through the single service port
// (FR-044/FR-048).
func NewGRPCServer(
	identity *identitygrpc.IdentityServer,
	identityDashboard *identitygrpc.IdentityDashboardServer,
	subscription *subscriptiongrpc.SubscriptionServer,
	subscriptionDashboard *subscriptiongrpc.SubscriptionDashboardServer,
	admin *admingrpc.AdminServer,
) *grpc.Server {
	server := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(server, identity)
	identityv1.RegisterIdentityDashboardServiceServer(server, identityDashboard)
	subscriptionv1.RegisterSubscriptionServiceServer(server, subscription)
	subscriptionv1.RegisterSubscriptionDashboardServiceServer(server, subscriptionDashboard)
	adminv1.RegisterAdminServiceServer(server, admin)
	return server
}
