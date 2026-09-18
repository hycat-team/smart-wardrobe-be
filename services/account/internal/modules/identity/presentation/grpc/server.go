// Package grpc provides inbound gRPC adapters for the identity contract. The
// servers depend on the contract interface — never on use cases directly
// (Constitution II/IV, T039).
package grpc

import (
	"context"
	"time"

	grpcx "smart-wardrobe-be/contracts/grpcx"
	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identitytransport "smart-wardrobe-be/services/account/contract/identity/transport"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
)

// IdentityServer is the inbound adapter for IdentityService.
type IdentityServer struct {
	identityv1.UnimplementedIdentityServiceServer
	brief identitycontract.IUserBriefContract
}

var (
	_ identityv1.IdentityServiceServer          = (*IdentityServer)(nil)
	_ identityv1.IdentityDashboardServiceServer = (*IdentityDashboardServer)(nil)
)

// NewIdentityServer builds the identity service bound to the contract port.
func NewIdentityServer(brief identitycontract.IUserBriefContract) *IdentityServer {
	return &IdentityServer{brief: brief}
}

func (s *IdentityServer) GetUserBrief(ctx context.Context, req *identityv1.GetUserBriefRequest) (*identityv1.UserBrief, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, grpcx.ToGRPCError(apperror.From(err))
	}
	out, err := s.brief.GetByID(ctx, userID)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return identitytransport.ContractDTOToUserBrief(out), nil
}

func (s *IdentityServer) GetUserBriefs(ctx context.Context, req *identityv1.GetUserBriefsRequest) (*identityv1.UserBriefList, error) {
	ids := make([]uuid.UUID, 0, len(req.GetUserIds()))
	for _, raw := range req.GetUserIds() {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, grpcx.ToGRPCError(apperror.From(err))
		}
		ids = append(ids, id)
	}
	out, err := s.brief.GetByIDs(ctx, ids)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &identityv1.UserBriefList{Users: identitytransport.ContractDTOsToUserBriefs(out)}, nil
}

func (s *IdentityServer) GetUserBriefsByIdentifiers(ctx context.Context, req *identityv1.GetUserBriefsByIdentifiersRequest) (*identityv1.UserBriefList, error) {
	out, err := s.brief.GetByUsernamesOrEmails(ctx, req.GetIdentifiers())
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &identityv1.UserBriefList{Users: identitytransport.ContractDTOsToUserBriefs(out)}, nil
}

// IdentityDashboardServer is the inbound adapter for IdentityDashboardService.
type IdentityDashboardServer struct {
	identityv1.UnimplementedIdentityDashboardServiceServer
	reader identitycontract.IUserDashboardReader
}

// NewIdentityDashboardServer builds the identity dashboard service bound to the reader port.
func NewIdentityDashboardServer(reader identitycontract.IUserDashboardReader) *IdentityDashboardServer {
	return &IdentityDashboardServer{reader: reader}
}

func (s *IdentityDashboardServer) GetAdminUserStats(ctx context.Context, req *identityv1.GetAdminUserStatsRequest) (*identityv1.UserAdminStats, error) {
	var snapshotDate time.Time
	if ts := req.GetSnapshotDate(); ts != nil {
		snapshotDate = ts.AsTime()
	}
	out, err := s.reader.GetAdminUserStats(ctx, snapshotDate)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return identitytransport.UserAdminStatsToProto(out), nil
}

func (s *IdentityDashboardServer) GetActiveUserIDs(ctx context.Context, req *identityv1.GetActiveUserIDsRequest) (*identityv1.UserIDList, error) {
	out, err := s.reader.GetActiveUserIDs(ctx)
	if err != nil {
		return nil, grpcx.ToGRPCError(err)
	}
	return &identityv1.UserIDList{UserIds: identitytransport.UserIDsToProtoStrings(out)}, nil
}
