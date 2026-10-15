// Package grpc provides inbound gRPC adapters for the AI orchestrator and cost
// contracts. The servers depend on the contract interfaces — never on use cases
// directly (Constitution II/IV, T039).
package grpc

import (
	"context"
	"time"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	aicontract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	aitransport "smart-wardrobe-be/services/ai/contract/aiexecutor/transport"

	"google.golang.org/grpc"
)

// AIOrchestratorServer is the inbound adapter for AIOrchestratorService.
type AIOrchestratorServer struct {
	aiv1.UnimplementedAIOrchestratorServiceServer
	orchestrator aicontract.IAIOrchestrator
}

var (
	_ aiv1.AIOrchestratorServiceServer = (*AIOrchestratorServer)(nil)
	_ aiv1.AIDashboardServiceServer    = (*AIDashboardServer)(nil)
)

// NewAIOrchestratorServer builds the AI orchestrator service bound to the contract port.
func NewAIOrchestratorServer(orchestrator aicontract.IAIOrchestrator) *AIOrchestratorServer {
	return &AIOrchestratorServer{orchestrator: orchestrator}
}

func (s *AIOrchestratorServer) AnalyzeImage(ctx context.Context, req *aiv1.AnalyzeImageRequest) (*aiv1.AnalyzeImageResponse, error) {
	out, err := s.orchestrator.AnalyzeImage(ctx, req.GetImageUrl(), req.GetPrompt())
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &aiv1.AnalyzeImageResponse{Result: out}, nil
}

func (s *AIOrchestratorServer) GenerateEmbeddings(ctx context.Context, req *aiv1.GenerateEmbeddingsRequest) (*aiv1.GenerateEmbeddingsResponse, error) {
	matrices, err := s.orchestrator.GenerateEmbeddings(ctx, req.GetChunks())
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	vectors := make([]*aiv1.EmbeddingVector, 0, len(matrices))
	for _, values := range matrices {
		vectors = append(vectors, &aiv1.EmbeddingVector{Values: values})
	}
	return &aiv1.GenerateEmbeddingsResponse{Vectors: vectors}, nil
}

func (s *AIOrchestratorServer) GenerateChatText(ctx context.Context, req *aiv1.GenerateChatTextRequest) (*aiv1.GenerateChatTextResponse, error) {
	options, err := aitransport.OptionsFromProto(req.GetOptions())
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	out, err := s.orchestrator.GenerateChatText(ctx, req.GetSystemPrompt(), req.GetUserPrompt(), options)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &aiv1.GenerateChatTextResponse{Result: out}, nil
}

func (s *AIOrchestratorServer) GenerateRecommendationText(ctx context.Context, req *aiv1.GenerateRecommendationTextRequest) (*aiv1.GenerateRecommendationTextResponse, error) {
	options, err := aitransport.OptionsFromProto(req.GetOptions())
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	out, err := s.orchestrator.GenerateRecommendationText(ctx, req.GetSystemPrompt(), req.GetUserPrompt(), options)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &aiv1.GenerateRecommendationTextResponse{Result: out}, nil
}

// GenerateChatTextStream bridges the contract's channel pair to the wire
// server stream. It terminates when both channels are drained, an error
// arrives, or the stream context is cancelled.
func (s *AIOrchestratorServer) GenerateChatTextStream(req *aiv1.GenerateChatTextRequest, stream grpc.ServerStreamingServer[aiv1.ChatChunk]) error {
	options, err := aitransport.OptionsFromProto(req.GetOptions())
	if err != nil {
		return grpcx.ToGRPCError(err)
	}
	ctx := stream.Context()
	deltas, errc := s.orchestrator.GenerateChatTextStream(ctx, req.GetSystemPrompt(), req.GetUserPrompt(), options)
	for deltas != nil || errc != nil {
		select {
		case delta, ok := <-deltas:
			if !ok {
				deltas = nil
				continue
			}
			if delta != "" {
				if err := stream.Send(&aiv1.ChatChunk{Delta: delta}); err != nil {
					return grpcx.ToGRPCError(err)
				}
			}
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			if err != nil {
				return grpcx.ToGRPCError(err)
			}
		case <-ctx.Done():
			return grpcx.ToGRPCError(ctx.Err())
		}
	}
	return stream.Send(&aiv1.ChatChunk{Done: true})
}

// AIDashboardServer is the inbound adapter for AIDashboardService.
type AIDashboardServer struct {
	aiv1.UnimplementedAIDashboardServiceServer
	reader aicontract.IAICostDashboardReader
}

// NewAIDashboardServer builds the AI dashboard service bound to the reader port.
func NewAIDashboardServer(reader aicontract.IAICostDashboardReader) *AIDashboardServer {
	return &AIDashboardServer{reader: reader}
}

func (s *AIDashboardServer) GetAdminAIUsageStats(ctx context.Context, req *aiv1.GetAdminAIUsageStatsRequest) (*aiv1.AIUsageAdminStats, error) {
	var snapshotDate time.Time
	if ts := req.GetSnapshotDate(); ts != nil {
		snapshotDate = ts.AsTime()
	}
	out, err := s.reader.GetAdminAIUsageStats(ctx, snapshotDate)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return aitransport.StatsFromDTO(out), nil
}

func (s *AIDashboardServer) GetAIEventsDrillDown(ctx context.Context, req *aiv1.GetAIEventsDrillDownRequest) (*aiv1.AIEventDrillDownList, error) {
	items, total, err := s.reader.GetAIEventsDrillDown(ctx, req.GetStatus(), int(req.GetPage()), int(req.GetLimit()))
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &aiv1.AIEventDrillDownList{
		Items: aitransport.EventsFromDTOs(items),
		Total: total,
	}, nil
}

func (s *AIDashboardServer) GetAIUsageByOperation(ctx context.Context, req *aiv1.GetAIUsageByOperationRequest) (*aiv1.AIUsageByOperationList, error) {
	var fromDate, toDate time.Time
	if ts := req.GetFromDate(); ts != nil {
		fromDate = ts.AsTime()
	}
	if ts := req.GetToDate(); ts != nil {
		toDate = ts.AsTime()
	}
	items, err := s.reader.GetAIUsageByOperation(ctx, fromDate, toDate)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &aiv1.AIUsageByOperationList{
		Items: aitransport.UsageFromDTOs(items),
	}, nil
}

func (s *AIDashboardServer) GetTotalAIRequestCount(ctx context.Context, req *aiv1.GetTotalAIRequestCountRequest) (*aiv1.TotalAIRequestCount, error) {
	var fromDate, toDate time.Time
	if ts := req.GetFromDate(); ts != nil {
		fromDate = ts.AsTime()
	}
	if ts := req.GetToDate(); ts != nil {
		toDate = ts.AsTime()
	}
	total, err := s.reader.GetTotalAIRequestCount(ctx, fromDate, toDate)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &aiv1.TotalAIRequestCount{Total: total}, nil
}

// GetAIErrorBreakdown reads the new from_date/to_date pair (029 grpc-delta
// §2). Absent fields stay zero, which the reader treats as "all-time" — the
// pre-029 behaviour.
func (s *AIDashboardServer) GetAIErrorBreakdown(ctx context.Context, req *aiv1.GetAIErrorBreakdownRequest) (*aiv1.AIErrorBreakdownList, error) {
	var from, to time.Time
	if ts := req.GetFromDate(); ts != nil {
		from = ts.AsTime()
	}
	if ts := req.GetToDate(); ts != nil {
		to = ts.AsTime()
	}
	items, err := s.reader.GetAIErrorBreakdown(ctx, from, to)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &aiv1.AIErrorBreakdownList{
		Items: aitransport.ErrorBreakdownFromDTOs(items),
	}, nil
}
