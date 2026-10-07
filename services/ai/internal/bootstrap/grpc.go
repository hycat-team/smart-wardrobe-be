package bootstrap

import (
	aigrpc "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/grpc"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	"smart-wardrobe-be/shared/pkg/logger"

	"google.golang.org/grpc"
)

// NewGRPCServer registers the AI-hosted gRPC services (AIOrchestrator,
// AIDashboard) on one server shared through the single service port
// (FR-044/FR-048).
//
// The recovery interceptors are not optional hardening here. This service was
// taken down by a single nil pointer dereference in one handler: a nil grant
// reached the entitlement reader, the reader dereferenced it while building a log
// field, and the container came back with RestartCount=6. A gRPC handler panic is
// unrecovered in the goroutine that owns it, so it removes the whole process and
// every other in-flight call — which is why the blast radius was the entire AI
// feature for all users rather than one failed request.
//
// The HTTP side of this service has had panic recovery through the shared error
// middleware all along. This is the gRPC side catching up, not a new policy.
func NewGRPCServer(
	orchestrator *aigrpc.AIOrchestratorServer,
	dashboard *aigrpc.AIDashboardServer,
	log logger.Interface,
) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcx.UnaryRecoveryInterceptor(log)),
		grpc.StreamInterceptor(grpcx.StreamRecoveryInterceptor(log)),
	)
	aiv1.RegisterAIOrchestratorServiceServer(server, orchestrator)
	aiv1.RegisterAIDashboardServiceServer(server, dashboard)
	return server
}
