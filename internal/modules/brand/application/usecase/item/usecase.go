package item

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/votetype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/constants/fashionitemtype"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/pkg/utils/sliceutils"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

var activeStatus = branditemstatus.Active

type BrandItemUseCase struct {
	brandRepo         repositories.IBrandRepository
	memberRepo        repositories.IBrandMemberRepository
	customerRepo      repositories.IBrandCustomerRepository
	brandItemRepo     repositories.IBrandItemRepository
	feedbackRepo      repositories.IDigitalSampleResponseRepository
	userBriefContract identity_contract.IUserBriefContract
	fashionContract   fashion_contract.IFashionContract
	eventPublisher    event.IEventPublisher
	mediaService      media.IMediaService
	benefitUC         uc_interfaces.IBrandBenefitUseCase
	uow               shared_repos.IUnitOfWork
}

func NewBrandItemUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	brandItemRepo repositories.IBrandItemRepository,
	feedbackRepo repositories.IDigitalSampleResponseRepository,
	userBriefContract identity_contract.IUserBriefContract,
	fashionContract fashion_contract.IFashionContract,
	eventPublisher event.IEventPublisher,
	mediaService media.IMediaService,
	benefitUC uc_interfaces.IBrandBenefitUseCase,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandItemUseCase {
	return &BrandItemUseCase{
		brandRepo:         brandRepo,
		memberRepo:        memberRepo,
		customerRepo:      customerRepo,
		brandItemRepo:     brandItemRepo,
		feedbackRepo:      feedbackRepo,
		userBriefContract: userBriefContract,
		fashionContract:   fashionContract,
		eventPublisher:    eventPublisher,
		mediaService:      mediaService,
		benefitUC:         benefitUC,
		uow:               uow,
	}
}

func (uc *BrandItemUseCase) GetBrandItemUploadSignature(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: fmt.Sprintf("brands/%s/items", brandID.String()),
	})
}

func (uc *BrandItemUseCase) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemRes, []*fashion_dto.FashionItemRes, error) {
	user, err := uc.userBriefContract.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return []*dto.BrandItemRes{}, []*fashion_dto.FashionItemRes{}, nil
	}

	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	brandItemsByBrandID := make(map[uuid.UUID][]*entities.BrandItem, len(activeBrandIDs))
	for _, item := range brandItems {
		brandItemsByBrandID[item.BrandID] = append(brandItemsByBrandID[item.BrandID], item)
	}

	eligibleDTOS := make([]*dto.BrandItemRes, 0, len(brandItems))
	eligibleItems := make([]*entities.BrandItem, 0, len(brandItems))
	sampleAccessByBrandID := make(map[uuid.UUID]bool, len(activeBrandIDs))
	sampleAccessChecked := make(map[uuid.UUID]struct{}, len(activeBrandIDs))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active || customer.Brand == nil || customer.Brand.Status != brandstatus.Active {
			continue
		}

		if _, checked := sampleAccessChecked[customer.BrandID]; !checked {
			hasSampleAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, customer.BrandID, benefitfeaturecode.SampleMixAccess)
			if err != nil {
				return nil, nil, err
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
				eligibleItems = append(eligibleItems, item)
			}
		}
	}

	fashionIDs := make([]uuid.UUID, 0, len(eligibleItems))
	for _, item := range eligibleItems {
		if item.FashionItemID != nil {
			fashionIDs = append(fashionIDs, *item.FashionItemID)
		}
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionItemMap := map[uuid.UUID]*fashion_dto.FashionItemRes{}
	if len(fashionIDs) > 0 {
		fashionContractItems, err := uc.fashionContract.ListFashionItems(ctx, fashionIDs)
		if err != nil {
			return nil, nil, err
		}
		fashionItemMap = sliceutils.IndexBy(fashionContractItems, func(f *fashion_dto.FashionItemRes) uuid.UUID { return f.ID })
	}

	for _, item := range eligibleItems {
		var fi *fashion_dto.FashionItemRes
		if item.FashionItemID != nil {
			fi = fashionItemMap[*item.FashionItemID]
		}
		eligibleDTOS = append(eligibleDTOS, mapper.MapToBrandItemRes(item, fi))
	}

	fashionItems := make([]*fashion_dto.FashionItemRes, len(eligibleItems))
	for i := range eligibleItems {
		if eligibleItems[i].FashionItemID != nil {
			if fi := fashionItemMap[*eligibleItems[i].FashionItemID]; fi != nil {
				fashionItems[i] = fi
			}
		}
	}

	return eligibleDTOS, fashionItems, nil
}

func (uc *BrandItemUseCase) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, *fashion_dto.FashionItemRes, error) {
	user, err := uc.userBriefContract.GetByID(ctx, userID)
	if err != nil {
		return false, nil, nil, err
	}
	if user == nil {
		return false, nil, nil, nil
	}

	brandItem, err := uc.brandItemRepo.GetByFashionItemID(ctx, fashionItemID)
	if err != nil {
		return false, nil, nil, err
	}
	if brandItem == nil || brandItem.Status != branditemstatus.Active {
		return false, nil, nil, nil
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandItem.BrandID)
	if err != nil {
		return false, nil, nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil, nil, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandItem.BrandID, userID)
	if err != nil {
		return false, nil, nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil, nil, nil
	}

	if brandItem.ItemType == branditemtype.Sample {
		hasSampleAccess, err := uc.benefitUC.CheckBrandFeatureAccess(ctx, userID, brandItem.BrandID, benefitfeaturecode.SampleMixAccess)
		if err != nil {
			return false, nil, nil, err
		}
		if !hasSampleAccess {
			return false, nil, nil, nil
		}
	}

	fi, _ := uc.fashionContract.GetFashionItem(ctx, *brandItem.FashionItemID)
	return true, mapper.MapToBrandItemRes(brandItem, fi), fi, nil
}

func (uc *BrandItemUseCase) ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(fashionItemIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}

	brandItems, err := uc.brandItemRepo.GetByFashionItemIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]bool, len(fashionItemIDs))
	for _, bi := range brandItems {
		if bi.FashionItemID != nil && bi.Status == branditemstatus.Active {
			result[*bi.FashionItemID] = true
		}
	}
	return result, nil
}

func (uc *BrandItemUseCase) CreateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandItemReq) (*dto.BrandItemRes, error) {
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
	fashionItemID := uuid.New()
	taskID := uuid.New()

	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID:        brandItemID,
				CreatedAt: time.Now().UTC(),
			},
			UpdatedAt: time.Now().UTC(),
		},
		BrandID:       brandID,
		FashionItemID: &fashionItemID,
		ProductCode:   input.ProductCode,
		Name:          input.Name,
		Description:   input.Description,
		Price:         input.Price,
		ItemType:      branditemtype.BrandItemType(input.ItemType),
		Status:        branditemstatus.BrandItemStatus(input.Status),
		TaskID:        &taskID,
	}
	if item.Status == "" {
		item.Status = branditemstatus.Draft
	}

	fashionInput := fashion_dto.CreateFashionItemInput{
		ID:                fashionItemID,
		ImageUrl:          input.ImageUrl,
		ImagePublicID:     input.ImagePublicID,
		CategoryID:        input.CategoryID,
		ProcessingVersion: 1,
	}

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.fashionContract.CreateFashionItem(txCtx, fashionInput); err != nil {
			return err
		}
		if err := uc.brandItemRepo.Create(txCtx, item); err != nil {
			return err
		}
		job := fashion_dto.FashionAnalyzeJobDTO{
			FashionItemID:     item.FashionItemID,
			ItemID:            brandItemID,
			ItemType:          fashionitemtype.Brand,
			UserID:            staffUserID,
			ImageUrl:          input.ImageUrl,
			ImagePublicID:     input.ImagePublicID,
			CategoryID:        input.CategoryID,
			ProcessingVersion: 1,
			TaskID:            taskID.String(),
			Total:             1,
			Index:             0,
		}
		if err := uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionAnalyze, job); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	fi := &fashion_dto.FashionItemRes{
		ID:            fashionItemID,
		ImageUrl:      input.ImageUrl,
		ImagePublicID: input.ImagePublicID,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	return mapper.MapToBrandItemRes(item, fi), nil
}

func (uc *BrandItemUseCase) GetBrandItemsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandItemRes, error) {
	items, err := uc.brandItemRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return uc.mapSliceToRes(ctx, items), nil
}

func (uc *BrandItemUseCase) GetBrandItemForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	return uc.mapToRes(ctx, item), nil
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
	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) UpdateBrandItem(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, input dto.UpdateBrandItemReq) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	if input.ProductCode != nil && (item.ProductCode == nil || *input.ProductCode != *item.ProductCode) {
		existing, err := uc.brandItemRepo.GetByProductCode(ctx, brandID, *input.ProductCode)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != itemID {
			return nil, branderrors.ErrProductCodeExists()
		}
	}
	item.ProductCode = input.ProductCode
	item.Name = input.Name
	item.Description = input.Description
	item.Price = input.Price
	item.Status = branditemstatus.BrandItemStatus(input.Status)
	item.UpdatedAt = time.Now().UTC()

	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) UpdateBrandItemStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID, status branditemstatus.BrandItemStatus) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	if status != branditemstatus.Draft && status != branditemstatus.Active && status != branditemstatus.Archived {
		return nil, branderrors.ErrBenefitInvalidStatus()
	}
	item.Status = status
	item.UpdatedAt = time.Now().UTC()
	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) GetBrandItemFeedbacks(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, itemID uuid.UUID) ([]*dto.DigitalSampleResponseRes, error) {
	feedbacks, err := uc.feedbackRepo.GetByBrandItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uuid.UUID, 0, len(feedbacks))
	for _, f := range feedbacks {
		userIDs = append(userIDs, f.UserID)
	}
	userBriefs := make([]*identity_dto.UserBriefRes, 0)
	if uc.userBriefContract != nil {
		userBriefs, _ = uc.userBriefContract.GetByIDs(ctx, userIDs)
	}
	userBriefMap := make(map[uuid.UUID]*identity_dto.UserBriefRes, len(userBriefs))
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

func (uc *BrandItemUseCase) UpdateBrandItemStatusDirect(ctx context.Context, brandItemID uuid.UUID, status branditemstatus.BrandItemStatus) error {
	item, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	item.Status = status
	return uc.brandItemRepo.Update(ctx, item)
}

func (uc *BrandItemUseCase) GetBrandItemByIDDirect(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemRes, error) {
	item, err := uc.brandItemRepo.GetByID(ctx, brandItemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return uc.mapToRes(ctx, item), nil
}

func (uc *BrandItemUseCase) GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemBriefRes, error) {
	if len(fashionItemIDs) == 0 {
		return []*dto.BrandItemBriefRes{}, nil
	}
	items, err := uc.brandItemRepo.GetByFashionItemIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}
	activeStatus := branditemstatus.Active
	filtered := make([]*entities.BrandItem, 0, len(items))
	for _, item := range items {
		if item.Status == activeStatus {
			filtered = append(filtered, item)
		}
	}
	return uc.mapSliceToBriefRes(ctx, filtered), nil
}

func (uc *BrandItemUseCase) GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemRes, error) {
	if len(brandItemIDs) == 0 {
		return []*dto.BrandItemRes{}, nil
	}
	items, err := uc.brandItemRepo.GetByIDs(ctx, brandItemIDs)
	if err != nil {
		return nil, err
	}
	return uc.mapSliceToRes(ctx, items), nil
}
