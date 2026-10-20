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

// GetAdminSubscriptionStats sends the half-open VN day pair [from, to)
// (029 grpc-delta §1/§3). Zero bounds are sent as zero timestamps: a 029
// server treats a both-zero pair as "no period" and falls back to
// snapshot_date (absent here), and a pre-029 server ignores the new fields.
func (c *SubscriptionDashboardReaderClient) GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*subscriptiondto.AdminSubscriptionStatsDTO, error) {
	res, err := c.svc.GetAdminSubscriptionStats(ctx, &subscriptionv1.GetAdminSubscriptionStatsRequest{
		FromDate: timestamppb.New(from),
		ToDate:   timestamppb.New(to),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.AdminStatsToDTO(res)
}

// GetSubscriptionDistribution returns the interval-based buckets plus the
// cutover date and completeness metadata (031 contract delta §3). An old
// Account server leaves both fields unset; DistributionListToDTO turns that
// into nil/empty values so the Commerce edge omits the optional JSON fields.
func (c *SubscriptionDashboardReaderClient) GetSubscriptionDistribution(ctx context.Context, from, to time.Time) (*subscriptiondto.SubscriptionDistributionResultDTO, error) {
	res, err := c.svc.GetSubscriptionDistribution(ctx, &subscriptionv1.GetSubscriptionDistributionRequest{
		FromDate: timestamppb.New(from),
		ToDate:   timestamppb.New(to),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.DistributionListToDTO(res), nil
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

func (c *SubscriptionDashboardReaderClient) GetRenewalStats(ctx context.Context, from, to time.Time) (*subscriptiondto.RenewalStatsDTO, error) {
	res, err := c.svc.GetRenewalStats(ctx, &subscriptionv1.GetRenewalStatsRequest{
		FromDate: timestamppb.New(from),
		ToDate:   timestamppb.New(to),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.RenewalStatsToDTO(res), nil
}
