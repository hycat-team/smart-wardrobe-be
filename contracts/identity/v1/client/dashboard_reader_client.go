package client

import (
	"context"
	"time"

	grpcx "smart-wardrobe-be/contracts/grpcx"
	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identitytransport "smart-wardrobe-be/services/account/contract/identity/transport"
	identitydto "smart-wardrobe-be/services/account/dto/identity"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserDashboardReaderClient is the remote adapter for IUserDashboardReader.
type UserDashboardReaderClient struct {
	svc identityv1.IdentityDashboardServiceClient
}

var _ identitycontract.IUserDashboardReader = (*UserDashboardReaderClient)(nil)

// NewUserDashboardReaderClient builds the identity dashboard reader over gRPC.
func NewUserDashboardReaderClient(cc grpc.ClientConnInterface) *UserDashboardReaderClient {
	return &UserDashboardReaderClient{svc: identityv1.NewIdentityDashboardServiceClient(cc)}
}

func (c *UserDashboardReaderClient) GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*identitydto.UserAdminStatsDTO, error) {
	res, err := c.svc.GetAdminUserStats(ctx, &identityv1.GetAdminUserStatsRequest{
		SnapshotDate: timestamppb.New(snapshotDate),
	})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return identitytransport.UserAdminStatsToDTO(res), nil
}

func (c *UserDashboardReaderClient) GetActiveUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	res, err := c.svc.GetActiveUserIDs(ctx, &identityv1.GetActiveUserIDsRequest{})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return identitytransport.UserIDsFromProto(res.GetUserIds()), nil
}
