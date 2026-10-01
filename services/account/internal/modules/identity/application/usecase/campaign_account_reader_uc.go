package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
)

// CampaignAccountReader serves the campaign catch-up sweep's account listing
// (025 T024, FR-079).
//
// It is a separate type from `UserBriefUseCase` rather than another method on it.
// `IUserBriefUseCase` is the shape of the gRPC-served contract; adding a method
// there would force a change to `identity.proto` — a cross-service contract change
// for a need that exists only inside this service, which FR-058 forbids
// (contracts/identity-contract.md §2).
type CampaignAccountReader struct {
	userRepo repositories.IUserRepository
}

func NewCampaignAccountReader(userRepo repositories.IUserRepository) uc_interfaces.ICampaignAccountReader {
	return &CampaignAccountReader{userRepo: userRepo}
}

// ListCreatedInWindow returns one batch of accounts created inside the window,
// oldest first.
//
// An empty result is an EMPTY SLICE with `TotalCount` from the window, never nil
// and never a partially-built page: the sweep ranges over `Items` and uses
// `TotalCount` to decide how many batches are left, so a nil slice would read as
// "not implemented" where it means "nothing in this window".
func (uc *CampaignAccountReader) ListCreatedInWindow(ctx context.Context, filter repositories.CampaignAccountFilter) (*uc_interfaces.CampaignAccountPageResult, error) {
	page, err := uc.userRepo.ListCreatedInWindow(ctx, filter)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return &uc_interfaces.CampaignAccountPageResult{Items: []*dto.UserBriefContractDTO{}}, nil
	}

	items := make([]*dto.UserBriefContractDTO, 0, len(page.Items))
	for _, user := range page.Items {
		if user == nil || user.IsDeleted {
			continue
		}
		items = append(items, mapper.MapToUserBrief(user))
	}

	return &uc_interfaces.CampaignAccountPageResult{
		Items:      items,
		TotalCount: page.TotalCount,
	}, nil
}

var _ uc_interfaces.ICampaignAccountReader = (*CampaignAccountReader)(nil)
