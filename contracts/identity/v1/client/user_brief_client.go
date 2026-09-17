// Package client provides the outbound gRPC adapter for the identity contract,
// implementing the same port as the in-process implementation so callers do not
// change when the service boundary is crossed (Constitution II, FR-019).
package client

import (
	"context"

	grpcx "smart-wardrobe-be/contracts/grpcx"
	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identitytransport "smart-wardrobe-be/services/account/contract/identity/transport"
	identitydto "smart-wardrobe-be/services/account/dto/identity"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// UserBriefClient is the remote adapter for IUserBriefContract.
type UserBriefClient struct {
	svc identityv1.IdentityServiceClient
}

var _ identitycontract.IUserBriefContract = (*UserBriefClient)(nil)

// NewUserBriefClient builds the identity user-brief contract over gRPC.
func NewUserBriefClient(cc grpc.ClientConnInterface) *UserBriefClient {
	return &UserBriefClient{svc: identityv1.NewIdentityServiceClient(cc)}
}

func (c *UserBriefClient) GetByID(ctx context.Context, userID uuid.UUID) (*identitydto.UserBriefContractDTO, error) {
	res, err := c.svc.GetUserBrief(ctx, &identityv1.GetUserBriefRequest{UserId: userID.String()})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return identitytransport.UserBriefToDTO(res)
}

func (c *UserBriefClient) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*identitydto.UserBriefContractDTO, error) {
	ids := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		ids = append(ids, id.String())
	}
	res, err := c.svc.GetUserBriefs(ctx, &identityv1.GetUserBriefsRequest{UserIds: ids})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return identitytransport.UserBriefsToDTOs(res.GetUsers())
}

func (c *UserBriefClient) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*identitydto.UserBriefContractDTO, error) {
	res, err := c.svc.GetUserBriefsByIdentifiers(ctx, &identityv1.GetUserBriefsByIdentifiersRequest{Identifiers: identifiers})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return identitytransport.UserBriefsToDTOs(res.GetUsers())
}
