package brand

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"

	"github.com/google/uuid"
)

func (uc *BrandUseCase) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error) {
	return uc.brandRepo.GetAdminBrandStats(ctx, snapshotDate)
}

func (uc *BrandUseCase) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error) {
	return uc.brandRepo.GetBrandAnalyticsStats(ctx, brandID, snapshotDate)
}

// GetUserBrandRole returns the active member role of the user in the brand.
// It reads through a per-user cache to avoid hitting the database on every
// request; on a cache miss it loads the user's active memberships from the DB
// and stores them. It returns ErrBrandPortalForbidden when the user is not an
// active member of an active brand (no default "staff" fallback).
func (uc *BrandUseCase) GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error) {
	memberships, err := uc.getActiveBrandMemberships(ctx, userID)
	if err != nil {
		return "", err
	}
	role, ok := memberships[brandID]
	if !ok {
		return "", branderrors.ErrBrandPortalForbidden()
	}
	return role, nil
}

// InvalidateUserMembershipCache drops the cached memberships of a single user.
func (uc *BrandUseCase) InvalidateUserMembershipCache(ctx context.Context, userID uuid.UUID) error {
	if uc.membershipCache == nil {
		return nil
	}
	return uc.membershipCache.InvalidateByUserID(ctx, userID)
}

// InvalidateBrandMembersCache drops the cached memberships of every active
// member of a brand (used when a brand's status changes).
func (uc *BrandUseCase) InvalidateBrandMembersCache(ctx context.Context, brandID uuid.UUID) error {
	if uc.membershipCache == nil {
		return nil
	}
	members, err := uc.memberRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return err
	}
	userIDs := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	return uc.membershipCache.InvalidateByBrandID(ctx, brandID, userIDs)
}

// getActiveBrandMemberships returns a map of brandID -> role for the user's
// memberships where the member is active and the brand is active. It reads the
// cache first and falls back to the database on a miss.
func (uc *BrandUseCase) getActiveBrandMemberships(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]brandmemberrole.BrandMemberRole, error) {
	if uc.membershipCache != nil {
		if cached, err := uc.membershipCache.Get(ctx, userID); err == nil && cached != nil {
			return cached, nil
		}
	}

	members, err := uc.memberRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	memberships := make(map[uuid.UUID]brandmemberrole.BrandMemberRole, len(members))
	for _, m := range members {
		if m == nil || m.Status != brandmemberstatus.Active {
			continue
		}
		if m.Brand != nil && m.Brand.Status != brandstatus.Active {
			continue
		}
		memberships[m.BrandID] = m.Role
	}

	if uc.membershipCache != nil {
		if err := uc.membershipCache.Set(ctx, userID, memberships); err != nil {
			// Cache write failures must not break the authorization decision.
			return memberships, nil
		}
	}
	return memberships, nil
}

func (uc *BrandUseCase) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error) {
	return uc.brandRepo.GetSampleLabFeedbacksDrillDown(ctx, itemID, page, limit)
}

func (uc *BrandUseCase) GetItemFeedbackSummary(ctx context.Context, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error) {
	responses, err := uc.feedbackRepo.GetByBrandItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if len(responses) == 0 {
		return nil, branderrors.ErrItemFeedbackNotFound()
	}

	var totalRating float64
	outfitSet := map[uuid.UUID]bool{}
	for _, r := range responses {
		if r.Rating != nil {
			totalRating += float64(*r.Rating)
		}
		if r.OutfitID != nil {
			outfitSet[*r.OutfitID] = true
		}
	}

	avgRating := totalRating / float64(len(responses))

	var sentiment string
	if avgRating >= 4 {
		sentiment = "positive"
	} else if avgRating >= 3 {
		sentiment = "neutral"
	} else {
		sentiment = "negative"
	}

	outfitIDs := make([]uuid.UUID, 0, len(outfitSet))
	for id := range outfitSet {
		outfitIDs = append(outfitIDs, id)
	}

	return &dto.ItemFeedbackSummaryDTO{
		VotesCount:        len(responses),
		AvgRating:         avgRating,
		FeedbackSentiment: sentiment,
		OutfitIDs:         outfitIDs,
	}, nil
}

func (uc *BrandUseCase) GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return uc.benefitRedemptionRepo.GetUserExpiringBenefitsCount(ctx, userID)
}
