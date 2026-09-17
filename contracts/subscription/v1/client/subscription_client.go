// Package client provides the outbound gRPC adapter for the subscription
// contract, implementing the same ports as the in-process implementation so
// callers do not change when the service boundary is crossed (FR-019).
package client

import (
	"context"

	grpcx "smart-wardrobe-be/contracts/grpcx"
	subscriptionv1 "smart-wardrobe-be/contracts/subscription/v1"
	subscriptioncontract "smart-wardrobe-be/services/account/contract/subscription"
	subscriptiontransport "smart-wardrobe-be/services/account/contract/subscription/transport"
	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// SubscriptionClient is the remote adapter for IUserSubscriptionContract and
// IUserQuotaContract.
type SubscriptionClient struct {
	svc subscriptionv1.SubscriptionServiceClient
}

var (
	_ subscriptioncontract.IUserSubscriptionContract = (*SubscriptionClient)(nil)
	_ subscriptioncontract.IUserQuotaContract        = (*SubscriptionClient)(nil)
)

// NewSubscriptionClient builds the subscription contract over gRPC.
func NewSubscriptionClient(cc grpc.ClientConnInterface) *SubscriptionClient {
	return &SubscriptionClient{svc: subscriptionv1.NewSubscriptionServiceClient(cc)}
}

func (c *SubscriptionClient) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*subscriptiondto.UserSubscriptionContractDTO, error) {
	res, err := c.svc.GetUserSubscriptionOverview(ctx, &subscriptionv1.GetUserSubscriptionOverviewRequest{UserId: userID.String()})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.OverviewToDTO(res)
}

func (c *SubscriptionClient) GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*subscriptiondto.UserSubscriptionContractDTO, error) {
	ids := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		ids = append(ids, id.String())
	}
	res, err := c.svc.GetUserSubscriptionOverviews(ctx, &subscriptionv1.GetUserSubscriptionOverviewsRequest{UserIds: ids})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.OverviewMapToDTOs(res.GetOverviews())
}

func (c *SubscriptionClient) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*subscriptiondto.UserSubscriptionDTO, error) {
	res, err := c.svc.GetAndResetDailyQuota(ctx, &subscriptionv1.GetAndResetDailyQuotaRequest{UserId: userID.String()})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return subscriptiontransport.UserSubscriptionToDTO(res)
}

func (c *SubscriptionClient) UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error {
	_, err := c.svc.UpdateOutfitQuota(ctx, &subscriptionv1.UpdateQuotaRequest{UserId: userID.String(), Count: int32(count)})
	return grpcx.ToAppError(err)
}

func (c *SubscriptionClient) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error {
	_, err := c.svc.UpdateAiChatQuota(ctx, &subscriptionv1.UpdateQuotaRequest{UserId: userID.String(), Count: int32(count)})
	return grpcx.ToAppError(err)
}
