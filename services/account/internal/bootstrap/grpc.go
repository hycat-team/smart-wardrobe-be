package bootstrap

import (
	admingrpc "smart-wardrobe-be/services/account/internal/modules/admin/presentation/grpc"
	identitygrpc "smart-wardrobe-be/services/account/internal/modules/identity/presentation/grpc"
	subscriptiongrpc "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/grpc"

	adminv1 "smart-wardrobe-be/contracts/admin/v1"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	"smart-wardrobe-be/shared/pkg/logger"

	"google.golang.org/grpc"
)

// NewGRPCServer registers the Account-hosted gRPC services (Identity,
// Subscription, Admin) on one server shared through the single service port
// (FR-044/FR-048).
//
// The recovery interceptors here are for the same reason they are in the AI
// service: a panic in a gRPC handler kills the process, not the request, and this
// service hosts every contract the other two call. Its HTTP side has had panic
// recovery through the shared error middleware all along; this is the gRPC side
// catching up rather than a new policy.
func NewGRPCServer(
	identity *identitygrpc.IdentityServer,
	identityDashboard *identitygrpc.IdentityDashboardServer,
	subscription *subscriptiongrpc.SubscriptionServer,
	subscriptionDashboard *subscriptiongrpc.SubscriptionDashboardServer,
	admin *admingrpc.AdminServer,
	log logger.Interface,
) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcx.UnaryRecoveryInterceptor(log)),
		grpc.StreamInterceptor(grpcx.StreamRecoveryInterceptor(log)),
	)
	identityv1.RegisterIdentityServiceServer(server, identity)
	identityv1.RegisterIdentityDashboardServiceServer(server, identityDashboard)
	subscriptionv1.RegisterSubscriptionServiceServer(server, subscription)
	subscriptionv1.RegisterSubscriptionDashboardServiceServer(server, subscriptionDashboard)
	adminv1.RegisterAdminServiceServer(server, admin)
	return server
}
