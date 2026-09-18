package bootstrap

import (
	aigrpc "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/grpc"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"

	"google.golang.org/grpc"
)

// NewGRPCServer registers the AI-hosted gRPC services (AIOrchestrator,
// AIDashboard) on one server shared through the single service port
// (FR-044/FR-048).
func NewGRPCServer(
	orchestrator *aigrpc.AIOrchestratorServer,
	dashboard *aigrpc.AIDashboardServer,
) *grpc.Server {
	server := grpc.NewServer()
	aiv1.RegisterAIOrchestratorServiceServer(server, orchestrator)
	aiv1.RegisterAIDashboardServiceServer(server, dashboard)
	return server
}
