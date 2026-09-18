package client

import (
	"context"
	"time"

	grpcx "smart-wardrobe-be/contracts/grpcx"
	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	subscriptioncontract "smart-wardrobe-be/services/account/contract/subscription"
	subscriptiontransport "smart-wardrobe-be/services/account/contract/subscription/transport"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SubscriptionDashboardReaderClient is the remote adapter for
// ISubscriptionDashboardReader.
type SubscriptionDashboardReaderClient struct {
	svc subscriptionv1.SubscriptionDashboardServiceClient
}

var _ subscriptioncontract.ISubscriptionDashboardReader = (*SubscriptionDashboardReaderClient)(nil)

// NewSubscriptionDashboardReaderClient builds the subscription dashboard reader over gRPC.
func NewSubscriptionDashboardReaderClient(cc grpc.ClientConnInterface) *SubscriptionDashboardReaderClient {
	return &SubscriptionDashboardReaderClient{svc: subscriptionv1.NewSubscriptionDashboardServiceClient(cc)}
}

func (c *SubscriptionDashboardReaderClient) GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*subscriptiondto.AdminSubscriptionStatsDTO, error) {
	res, err := c.svc.GetAdminSubscriptionStats(ctx, &subscriptionv1.GetAdminSubscriptionStatsRequest{
		SnapshotDate: timestamppb.New(snapshotDate),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.AdminStatsToDTO(res)
}

func (c *SubscriptionDashboardReaderClient) GetSubscriptionDistribution(ctx context.Context) ([]subscriptiondto.SubscriptionDistributionDTO, error) {
	res, err := c.svc.GetSubscriptionDistribution(ctx, &subscriptionv1.GetSubscriptionDistributionRequest{})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.DistributionToDTOs(res.GetItems()), nil
}

func (c *SubscriptionDashboardReaderClient) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]subscriptiondto.RevenueBreakdownDTO, error) {
	res, err := c.svc.GetRevenueBreakdown(ctx, &subscriptionv1.GetRevenueBreakdownRequest{
		FromDate: timestamppb.New(fromDate),
		ToDate:   timestamppb.New(toDate),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.RevenueBreakdownToDTOs(res.GetItems())
}

func (c *SubscriptionDashboardReaderClient) GetRenewalStats(ctx context.Context) (*subscriptiondto.RenewalStatsDTO, error) {
	res, err := c.svc.GetRenewalStats(ctx, &subscriptionv1.GetRenewalStatsRequest{})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.RenewalStatsToDTO(res), nil
}
