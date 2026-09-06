package item

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/votetype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

func (uc *BrandItemUseCase) GetBrandItemFeedbacks(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) ([]*dto.DigitalSampleResponseRes, error) {
	feedbacks, err := uc.feedbackRepo.GetByBrandItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uuid.UUID, 0, len(feedbacks))
	for _, f := range feedbacks {
		userIDs = append(userIDs, f.UserID)
	}
	userBriefs := make([]*identity_dto.UserBriefContractDTO, 0)
	if uc.userBriefContract != nil {
		userBriefs, _ = uc.userBriefContract.GetByIDs(ctx, userIDs)
	}
	userBriefMap := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(userBriefs))
	for _, ub := range userBriefs {
		userBriefMap[ub.ID] = ub
	}
	res := make([]*dto.DigitalSampleResponseRes, len(feedbacks))
	for i, f := range feedbacks {
		res[i] = mapper.MapDigitalSampleResponseWithUser(f, userBriefMap[f.UserID])
	}
	return res, nil
}

func (uc *BrandItemUseCase) ListBrandProductsForCustomer(ctx context.Context, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) (*dto.BrandItemListBriefRes, error) {
	productType := branditemtype.Product
	filter := repositories.BrandItemFilter{
		BrandID:  brandID,
		ItemType: &productType,
		Status:   &activeStatus,
		Page:     query.Page,
		Limit:    query.Limit,
	}
	result, err := uc.brandItemRepo.GetByBrandIDPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}
	items := uc.mapSliceToBriefRes(ctx, result.Items)
	return &dto.BrandItemListBriefRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandItemUseCase) ListBrandSamplesForCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) (*dto.BrandItemListBriefRes, error) {
	hasAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, brandID, benefitfeaturecode.SampleMixAccess)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
	sampleType := branditemtype.Sample
	filter := repositories.BrandItemFilter{
		BrandID:  brandID,
		ItemType: &sampleType,
		Status:   &activeStatus,
		Page:     query.Page,
		Limit:    query.Limit,
	}
	result, err := uc.brandItemRepo.GetByBrandIDPaginated(ctx, filter)
	if err != nil {
		return nil, err
	}
	items := uc.mapSliceToBriefRes(ctx, result.Items)
	return &dto.BrandItemListBriefRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandItemUseCase) SubmitSampleFeedback(ctx context.Context, userID uuid.UUID, brandItemID uuid.UUID, input dto.SubmitSampleFeedbackReq) (*dto.DigitalSampleResponseRes, error) {
	brandItem, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return nil, err
	}
	if brandItem == nil || brandItem.Status != branditemstatus.Active {
		return nil, branderrors.ErrBrandNotFound()
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandItem.BrandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	var vote *votetype.VoteType
	if input.VoteType != nil {
		v := votetype.VoteType(strings.ToLower(strings.TrimSpace(*input.VoteType)))
		if !isValidVoteType(v) {
			return nil, branderrors.ErrInvalidVoteType(*input.VoteType)
		}
		vote = &v
	}

	feedback := &entities.DigitalSampleResponse{
		BrandItemID:  brandItemID,
		UserID:       userID,
		OutfitID:     input.OutfitID,
		VoteType:     vote,
		Rating:       input.Rating,
		FeedbackText: input.FeedbackText,
		CreatedAt:    time.Now().UTC(),
	}

	if err := uc.feedbackRepo.Create(ctx, feedback); err != nil {
		return nil, err
	}

	return mapper.MapToDigitalSampleResponseRes(feedback), nil
}
