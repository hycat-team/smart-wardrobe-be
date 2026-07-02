package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/contract"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/branditem/branditemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/branditem/branditemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/branditem/votetype"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

var activeStatus = branditemstatus.Active

type BrandItemUseCase struct {
	brandRepo       repositories.IBrandRepository
	memberRepo      repositories.IBrandMemberRepository
	customerRepo    repositories.IBrandCustomerRepository
	brandItemRepo   repositories.IBrandItemRepository
	feedbackRepo    repositories.IDigitalSampleResponseRepository
	userRepo        identity_repos.IUserRepository
	fashionContract fashion_contract.IFashionContract
	mediaService    media.IMediaService
	benefitUC       uc_interfaces.IBrandBenefitUseCase
}

func NewBrandItemUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	brandItemRepo repositories.IBrandItemRepository,
	feedbackRepo repositories.IDigitalSampleResponseRepository,
	userRepo identity_repos.IUserRepository,
	fashionContract fashion_contract.IFashionContract,
	mediaService media.IMediaService,
	benefitUC uc_interfaces.IBrandBenefitUseCase,
) uc_interfaces.IBrandItemUseCase {
	return &BrandItemUseCase{
		brandRepo:       brandRepo,
		memberRepo:      memberRepo,
		customerRepo:    customerRepo,
		brandItemRepo:   brandItemRepo,
		feedbackRepo:    feedbackRepo,
		userRepo:        userRepo,
		fashionContract: fashionContract,
		mediaService:    mediaService,
		benefitUC:       benefitUC,
	}
}

func (uc *BrandItemUseCase) GetBrandItemUploadSignature(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, userID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: fmt.Sprintf("brands/%s/items", brandID.String()),
	})
}

func (uc *BrandItemUseCase) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemStylingDTO, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Status != userstatus.Active {
		return []*dto.BrandItemStylingDTO{}, nil
	}

	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	activeBrandIDs := make([]uuid.UUID, 0, len(customers))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active || customer.Brand == nil || customer.Brand.Status != brandstatus.Active {
			continue
		}
		activeBrandIDs = append(activeBrandIDs, customer.BrandID)
	}

	brandItems, err := uc.brandItemRepo.GetByBrandIDs(ctx, activeBrandIDs)
	if err != nil {
		return nil, err
	}
	brandItemsByBrandID := make(map[uuid.UUID][]*entities.BrandItem, len(activeBrandIDs))
	for _, item := range brandItems {
		brandItemsByBrandID[item.BrandID] = append(brandItemsByBrandID[item.BrandID], item)
	}

	eligibleBrandItems := make([]*dto.BrandItemStylingDTO, 0, len(brandItems))
	sampleAccessByBrandID := make(map[uuid.UUID]bool, len(activeBrandIDs))
	sampleAccessChecked := make(map[uuid.UUID]struct{}, len(activeBrandIDs))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active || customer.Brand == nil || customer.Brand.Status != brandstatus.Active {
			continue
		}

		if _, checked := sampleAccessChecked[customer.BrandID]; !checked {
			hasSampleAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, customer.BrandID, benefitfeaturecode.SampleMixAccess)
			if err != nil {
				return nil, err
			}
			sampleAccessByBrandID[customer.BrandID] = hasSampleAccess
			sampleAccessChecked[customer.BrandID] = struct{}{}
		}
		for _, item := range brandItemsByBrandID[customer.BrandID] {
			if item.Status != branditemstatus.Active {
				continue
			}

			canUseItem := item.ItemType == branditemtype.Product ||
				(item.ItemType == branditemtype.Sample && sampleAccessByBrandID[item.BrandID])

			if canUseItem {
				eligibleBrandItems = append(eligibleBrandItems, mapper.MapToBrandItemStylingDTO(item))
			}
		}
	}

	return eligibleBrandItems, nil
}

func (uc *BrandItemUseCase) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, nil, err
	}
	if user == nil || user.Status != userstatus.Active {
		return false, nil, nil
	}

	brandItem, err := uc.brandItemRepo.GetByFashionItemID(ctx, fashionItemID)
	if err != nil {
		return false, nil, err
	}
	if brandItem == nil || brandItem.Status != branditemstatus.Active {
		return false, nil, nil
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandItem.BrandID)
	if err != nil {
		return false, nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandItem.BrandID, userID)
	if err != nil {
		return false, nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil, nil
	}

	if brandItem.ItemType == branditemtype.Sample {
		hasSampleAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, brandItem.BrandID, benefitfeaturecode.SampleMixAccess)
		if err != nil {
			return false, nil, err
		}
		if !hasSampleAccess {
			return false, nil, nil
		}
	}

	return true, mapper.MapToBrandItemRes(brandItem), nil
}

func (uc *BrandItemUseCase) CreateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandItemReq) (*dto.BrandItemRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

	if input.ProductCode != nil && *input.ProductCode != "" {
		existing, err := uc.brandItemRepo.GetByProductCode(ctx, brandID, *input.ProductCode)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, branderrors.ErrProductCodeExists()
		}
	}

	brandItemID := uuid.New()
	fashionItemID, err := uc.fashionContract.CreateFashionItem(
		ctx,
		staffUserID,
		brandItemID,
		"brand",
		input.CategoryID,
		input.ImageUrl,
		input.ImagePublicID,
	)
	if err != nil {
		return nil, err
	}

	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID:        brandItemID,
				CreatedAt: time.Now().UTC(),
			},
			UpdatedAt: time.Now().UTC(),
		},
		BrandID:       brandID,
		FashionItemID: fashionItemID,
		ProductCode:   input.ProductCode,
		Name:          input.Name,
		Description:   input.Description,
		Price:         input.Price,
		ItemType:      branditemtype.BrandItemType(input.ItemType),
		Status:        branditemstatus.BrandItemStatus(input.Status),
	}
	if item.Status == "" {
		item.Status = branditemstatus.Draft
	}

	if err := uc.brandItemRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	fashionItem, _ := uc.fashionContract.GetFashionItem(ctx, fashionItemID)
	item.FashionItem = fashionItem

	return mapper.MapToBrandItemRes(item), nil
}

func (uc *BrandItemUseCase) GetBrandItemsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	items, err := uc.brandItemRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.BrandItemRes, len(items))
	for i, item := range items {
		res[i] = mapper.MapToBrandItemRes(item)
	}
	return res, nil
}

func (uc *BrandItemUseCase) GetBrandItemForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	fashionItem, _ := uc.fashionContract.GetFashionItem(ctx, item.FashionItemID)
	item.FashionItem = fashionItem
	return mapper.MapToBrandItemRes(item), nil
}

func (uc *BrandItemUseCase) GetBrandItemForUser(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.Status != branditemstatus.Active {
		return nil, branderrors.ErrBrandNotFound()
	}
	brand, err := uc.brandRepo.GetByID(ctx, item.BrandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}
	if item.ItemType == branditemtype.Sample {
		if userID == uuid.Nil {
			return nil, branderrors.ErrBrandPortalForbidden()
		}
		hasAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, item.BrandID, benefitfeaturecode.SampleMixAccess)
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, branderrors.ErrBrandPortalForbidden()
		}
	}
	fashionItem, _ := uc.fashionContract.GetFashionItem(ctx, item.FashionItemID)
	item.FashionItem = fashionItem
	return mapper.MapToBrandItemRes(item), nil
}

func (uc *BrandItemUseCase) UpdateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.UpdateBrandItemReq) (*dto.BrandItemRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	item.Name = input.Name
	item.Description = input.Description
	item.Price = input.Price
	item.Status = branditemstatus.BrandItemStatus(input.Status)
	item.UpdatedAt = time.Now().UTC()

	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	fashionItem, _ := uc.fashionContract.GetFashionItem(ctx, item.FashionItemID)
	item.FashionItem = fashionItem

	return mapper.MapToBrandItemRes(item), nil
}

func (uc *BrandItemUseCase) UpdateBrandItemStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, status string) (*dto.BrandItemRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	nextStatus := branditemstatus.BrandItemStatus(strings.ToLower(strings.TrimSpace(status)))
	if nextStatus != branditemstatus.Draft && nextStatus != branditemstatus.Active && nextStatus != branditemstatus.Archived {
		return nil, branderrors.ErrBenefitInvalidStatus()
	}
	item.Status = nextStatus
	item.UpdatedAt = time.Now().UTC()
	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	fashionItem, _ := uc.fashionContract.GetFashionItem(ctx, item.FashionItemID)
	item.FashionItem = fashionItem
	return mapper.MapToBrandItemRes(item), nil
}

func (uc *BrandItemUseCase) GetBrandItemFeedbacks(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) ([]*dto.DigitalSampleResponseRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	feedbacks, err := uc.feedbackRepo.GetByBrandItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.DigitalSampleResponseRes, len(feedbacks))
	for i, f := range feedbacks {
		res[i] = mapper.MapToDigitalSampleResponseRes(f)
	}
	return res, nil
}

func (uc *BrandItemUseCase) ListBrandProductsForCustomer(ctx context.Context, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) (*dto.BrandItemListRes, error) {
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
	items := make([]*dto.BrandItemRes, len(result.Items))
	for i, item := range result.Items {
		items[i] = mapper.MapToBrandItemRes(item)
	}
	return &dto.BrandItemListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandItemUseCase) ListBrandSamplesForCustomer(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetBrandItemsQueryReq) (*dto.BrandItemListRes, error) {
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
	items := make([]*dto.BrandItemRes, len(result.Items))
	for i, item := range result.Items {
		items[i] = mapper.MapToBrandItemRes(item)
	}
	return &dto.BrandItemListRes{
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

func isValidVoteType(vote votetype.VoteType) bool {
	switch vote {
	case votetype.Like, votetype.Dislike, votetype.WouldBuy, votetype.NotInterested:
		return true
	default:
		return false
	}
}
