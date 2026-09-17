package client

import (
	"context"
	"time"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	aicontract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	aitransport "smart-wardrobe-be/services/ai/contract/aiexecutor/transport"
	aidto "smart-wardrobe-be/services/ai/dto/aiexecutor"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AICostDashboardReaderClient is the remote adapter for IAICostDashboardReader.
type AICostDashboardReaderClient struct {
	svc aiv1.AIDashboardServiceClient
}

var _ aicontract.IAICostDashboardReader = (*AICostDashboardReaderClient)(nil)

// NewAICostDashboardReaderClient builds the AI cost dashboard reader over gRPC.
func NewAICostDashboardReaderClient(cc grpc.ClientConnInterface) *AICostDashboardReaderClient {
	return &AICostDashboardReaderClient{svc: aiv1.NewAIDashboardServiceClient(cc)}
}

func (c *AICostDashboardReaderClient) GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*aidto.AIUsageAdminStatsDTO, error) {
	res, err := c.svc.GetAdminAIUsageStats(ctx, &aiv1.GetAdminAIUsageStatsRequest{
		SnapshotDate: timestamppb.New(snapshotDate),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return aitransport.AdminStatsToDTO(res), nil
}

func (c *AICostDashboardReaderClient) GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]aidto.AIEventDrillDownDTO, int64, error) {
	res, err := c.svc.GetAIEventsDrillDown(ctx, &aiv1.GetAIEventsDrillDownRequest{
		Status: status,
		Page:   int32(page),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, 0, grpcx.ToAppError(err)
	}
	items, total := aitransport.EventsToDTOs(res)
	return items, total, nil
}

func (c *AICostDashboardReaderClient) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]aidto.AIUsageByOperationDTO, error) {
	res, err := c.svc.GetAIUsageByOperation(ctx, &aiv1.GetAIUsageByOperationRequest{
		FromDate: timestamppb.New(fromDate),
		ToDate:   timestamppb.New(toDate),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return aitransport.UsageByOperationToDTOs(res), nil
}

func (c *AICostDashboardReaderClient) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	res, err := c.svc.GetTotalAIRequestCount(ctx, &aiv1.GetTotalAIRequestCountRequest{
		FromDate: timestamppb.New(fromDate),
		ToDate:   timestamppb.New(toDate),
	})
	if err != nil {
		return 0, grpcx.ToAppError(err)
	}
	return res.GetTotal(), nil
}

func (c *AICostDashboardReaderClient) GetAIErrorBreakdown(ctx context.Context) ([]aidto.AIErrorBreakdownDTO, error) {
	res, err := c.svc.GetAIErrorBreakdown(ctx, &aiv1.GetAIErrorBreakdownRequest{})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return aitransport.ErrorBreakdownToDTOs(res), nil
}
