package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
)

// ICampaignAccountReader is the in-process read the campaign mechanism needs from
// the identity module (025 T024, FR-079).
//
// It is NOT a method on `IUserBriefUseCase`. That interface is the shape of the
// gRPC-served contract, and widening it would change the wire format — a
// cross-service contract change for a need that only exists inside this service,
// which FR-058 forbids. See contracts/identity-contract.md §2.
type ICampaignAccountReader interface {
	// ListCreatedInWindow returns one batch of non-deleted accounts created inside
	// [filter.From, filter.To), oldest first, paginated by the composite cursor
	// (filter.CursorTime, filter.CursorID).
	//
	// Items is an EMPTY slice when nothing matches, never nil.
	ListCreatedInWindow(ctx context.Context, filter repositories.CampaignAccountFilter) (*CampaignAccountPageResult, error)
}

// CampaignAccountPageResult is the use-case-level shape of one batch.
//
// It carries DTOs, never `entities.User`: constitution §II — "using DTOs, never
// domain entities". The entity-to-DTO translation happens here so the module
// boundary only ever sees the contract payload.
type CampaignAccountPageResult struct {
	Items      []*dto.UserBriefContractDTO
	TotalCount int64
}
