// Package grpc provides inbound gRPC adapters for the subscription contract.
// The servers depend on the contract interfaces — never on use cases directly
// (Constitution II/IV, T039).
package grpc

import (
	"context"
	"time"

	grpcx "smart-wardrobe-be/contracts/grpcx"
	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	subscriptioncontract "smart-wardrobe-be/services/account/contract/subscription"
	subscriptiontransport "smart-wardrobe-be/services/account/contract/subscription/transport"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
)

// SubscriptionServer is the inbound adapter for SubscriptionService.
type SubscriptionServer struct {
	subscriptionv1.UnimplementedSubscriptionServiceServer
	overview subscriptioncontract.IUserSubscriptionContract
	quota    subscriptioncontract.IUserQuotaContract
}

var (
	_ subscriptionv1.SubscriptionServiceServer          = (*SubscriptionServer)(nil)
	_ subscriptionv1.SubscriptionDashboardServiceServer = (*SubscriptionDashboardServer)(nil)
)

// NewSubscriptionServer builds the subscription service bound to the contract ports.
func NewSubscriptionServer(overview subscriptioncontract.IUserSubscriptionContract, quota subscriptioncontract.IUserQuotaContract) *SubscriptionServer {
	return &SubscriptionServer{overview: overview, quota: quota}
}

func (s *SubscriptionServer) GetUserSubscriptionOverview(ctx context.Context, req *subscriptionv1.GetUserSubscriptionOverviewRequest) (*subscriptionv1.UserSubscriptionOverview, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, grpcx.ToGRPCError(apperror.From(err))
	}
	out, err := s.overview.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return subscriptiontransport.OverviewDTOToProto(out), nil
}

func (s *SubscriptionServer) GetUserSubscriptionOverviews(ctx context.Context, req *subscriptionv1.GetUserSubscriptionOverviewsRequest) (*subscriptionv1.UserSubscriptionOverviewMap, error) {
	ids := make([]uuid.UUID, 0, len(req.GetUserIds()))
	for _, raw := range req.GetUserIds() {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, grpcx.ToGRPCError(apperror.From(err))
		}
		ids = append(ids, id)
	}
	out, err := s.overview.GetUserSubscriptionOverviews(ctx, ids)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &subscriptionv1.UserSubscriptionOverviewMap{
		Overviews: subscriptiontransport.OverviewDTOsToProtoMap(out),
	}, nil
}

func (s *SubscriptionServer) GetAndResetDailyQuota(ctx context.Context, req *subscriptionv1.GetAndResetDailyQuotaRequest) (*subscriptionv1.UserSubscription, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, grpcx.ToGRPCError(apperror.From(err))
	}
	out, err := s.quota.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return subscriptiontransport.UserSubscriptionDTOToProto(out), nil
}

func (s *SubscriptionServer) UpdateOutfitQuota(ctx context.Context, req *subscriptionv1.UpdateQuotaRequest) (*subscriptionv1.UpdateQuotaResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, grpcx.ToGRPCError(apperror.From(err))
	}
	if err := s.quota.UpdateOutfitQuota(ctx, userID, int(req.GetCount())); err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &subscriptionv1.UpdateQuotaResponse{}, nil
}

func (s *SubscriptionServer) UpdateAiChatQuota(ctx context.Context, req *subscriptionv1.UpdateQuotaRequest) (*subscriptionv1.UpdateQuotaResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, grpcx.ToGRPCError(apperror.From(err))
	}
	if err := s.quota.UpdateAiChatQuota(ctx, userID, int(req.GetCount())); err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &subscriptionv1.UpdateQuotaResponse{}, nil
}

// SubscriptionDashboardServer is the inbound adapter for SubscriptionDashboardService.
type SubscriptionDashboardServer struct {
	subscriptionv1.UnimplementedSubscriptionDashboardServiceServer
	reader subscriptioncontract.ISubscriptionDashboardReader
}

// NewSubscriptionDashboardServer builds the subscription dashboard service bound to the reader port.
func NewSubscriptionDashboardServer(reader subscriptioncontract.ISubscriptionDashboardReader) *SubscriptionDashboardServer {
	return &SubscriptionDashboardServer{reader: reader}
}

// GetAdminSubscriptionStats reads the new from_date/to_date pair when present
// (029 grpc-delta §1). When both are absent it keeps the old wire contract:
// the request's snapshot_date feeds VNDayRange, so a pre-029 client still gets
// the day it asked for.
func (s *SubscriptionDashboardServer) GetAdminSubscriptionStats(ctx context.Context, req *subscriptionv1.GetAdminSubscriptionStatsRequest) (*subscriptionv1.AdminSubscriptionStats, error) {
	var from, to time.Time
	if ts := req.GetFromDate(); ts != nil {
		from = ts.AsTime()
	}
	if ts := req.GetToDate(); ts != nil {
		to = ts.AsTime()
	}
	if from.IsZero() && to.IsZero() {
		var snapshotDate time.Time
		if ts := req.GetSnapshotDate(); ts != nil {
			snapshotDate = ts.AsTime()
		}
		from, to = timeutils.VNDayRange(snapshotDate)
	}
	out, err := s.reader.GetAdminSubscriptionStats(ctx, from, to)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return subscriptiontransport.AdminStatsToProto(out), nil
}

// GetSubscriptionDistribution passes the request's day pair through; absent
// fields stay zero (unbounded, meaning the live snapshot when both are zero).
// The response carries the interval-based buckets plus the cutover date and
// completeness of the requested period (031 contract delta §3).
func (s *SubscriptionDashboardServer) GetSubscriptionDistribution(ctx context.Context, req *subscriptionv1.GetSubscriptionDistributionRequest) (*subscriptionv1.SubscriptionDistributionList, error) {
	var from, to time.Time
	if ts := req.GetFromDate(); ts != nil {
		from = ts.AsTime()
	}
	if ts := req.GetToDate(); ts != nil {
		to = ts.AsTime()
	}
	out, err := s.reader.GetSubscriptionDistribution(ctx, from, to)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return subscriptiontransport.DistributionResultToProto(out), nil
}

func (s *SubscriptionDashboardServer) GetRevenueBreakdown(ctx context.Context, req *subscriptionv1.GetRevenueBreakdownRequest) (*subscriptionv1.RevenueBreakdownList, error) {
	var fromDate, toDate time.Time
	if ts := req.GetFromDate(); ts != nil {
		fromDate = ts.AsTime()
	}
	if ts := req.GetToDate(); ts != nil {
		toDate = ts.AsTime()
	}
	out, err := s.reader.GetRevenueBreakdown(ctx, fromDate, toDate)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &subscriptionv1.RevenueBreakdownList{
		Items: subscriptiontransport.RevenueBreakdownToProto(out),
	}, nil
}

func (s *SubscriptionDashboardServer) GetRenewalStats(ctx context.Context, req *subscriptionv1.GetRenewalStatsRequest) (*subscriptionv1.RenewalStats, error) {
	var from, to time.Time
	if ts := req.GetFromDate(); ts != nil {
		from = ts.AsTime()
	}
	if ts := req.GetToDate(); ts != nil {
		to = ts.AsTime()
	}
	out, err := s.reader.GetRenewalStats(ctx, from, to)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return subscriptiontransport.RenewalStatsToProto(out), nil
}
