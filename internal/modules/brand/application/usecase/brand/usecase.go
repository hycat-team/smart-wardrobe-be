package brand

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	brand_caching "smart-wardrobe-be/internal/modules/brand/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandUseCase struct {
	brandRepo             repositories.IBrandRepository
	memberRepo            repositories.IBrandMemberRepository
	customerRepo          repositories.IBrandCustomerRepository
	feedbackRepo          repositories.IDigitalSampleResponseRepository
	benefitRedemptionRepo repositories.IBenefitRedemptionRepository
	userContract          identity_contract.IUserContract
	mediaService          media.IMediaService
	uow                   shared_repos.IUnitOfWork
	cfg                   *config.Config
	membershipCache       brand_caching.IBrandMembershipCache
}

func NewBrandUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	feedbackRepo repositories.IDigitalSampleResponseRepository,
	benefitRedemptionRepo repositories.IBenefitRedemptionRepository,
	userContract identity_contract.IUserContract,
	mediaService media.IMediaService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
	membershipCache brand_caching.IBrandMembershipCache,
) uc_interfaces.IBrandUseCase {
	return &BrandUseCase{
		brandRepo:             brandRepo,
		memberRepo:            memberRepo,
		customerRepo:          customerRepo,
		feedbackRepo:          feedbackRepo,
		benefitRedemptionRepo: benefitRedemptionRepo,
		userContract:          userContract,
		mediaService:          mediaService,
		uow:                   uow,
		cfg:                   cfg,
		membershipCache:       membershipCache,
	}
}

func (uc *BrandUseCase) CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	return uc.createBrandWithOwner(ctx, userID, input, brandstatus.PendingReview, nil, nil)
}

func (uc *BrandUseCase) CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	now := time.Now().UTC()
	return uc.createBrandWithOwner(ctx, adminID, input, brandstatus.Active, &adminID, &now)
}

func (uc *BrandUseCase) createBrandWithOwner(ctx context.Context, creatorID uuid.UUID, input dto.CreateBrandReq, status brandstatus.BrandStatus, approverID *uuid.UUID, approvedAt *time.Time) (*dto.BrandRes, error) {
	slug := strings.TrimSpace(strings.ToLower(input.Slug))
	if existing, err := uc.brandRepo.GetBySlug(ctx, slug); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, branderrors.ErrBrandSlugExists()
	}

	brand := &entities.Brand{
		Slug:             slug,
		Name:             strings.TrimSpace(input.Name),
		Description:      input.Description,
		LogoURL:          input.LogoURL,
		LogoPublicID:     input.LogoPublicID,
		Status:           status,
		CreatedByUserID:  creatorID,
		ApprovedByUserID: approverID,
		ApprovedAt:       approvedAt,
	}
	member := &entities.BrandMember{
		UserID: creatorID,
		Role:   brandmemberrole.Owner,
		Status: brandmemberstatus.Active,
	}

	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.brandRepo.Create(txCtx, brand); err != nil {
			return err
		}
		member.BrandID = brand.ID
		return uc.memberRepo.Create(txCtx, member)
	}); err != nil {
		return nil, err
	}

	return mapper.MapBrand(brand, -1), nil
}

func (uc *BrandUseCase) UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error) {
	if !isValidBrandStatus(input.Status) {
		return nil, branderrors.ErrInvalidBrandStatus(input.Status)
	}
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}

	brand.Status = input.Status
	if input.Status == brandstatus.Active {
		now := time.Now().UTC()
		brand.ApprovedByUserID = &adminID
		brand.ApprovedAt = &now
	}
	if err := uc.brandRepo.Update(ctx, brand); err != nil {
		return nil, err
	}
	// Changing a brand's status changes its members' effective authorization,
	// so invalidate their membership cache immediately.
	if err := uc.InvalidateBrandMembersCache(ctx, brandID); err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand, -1), nil
}

func (uc *BrandUseCase) GetBrandsForAdmin(ctx context.Context, query dto.GetBrandsAdminQueryReq) (*dto.AdminBrandListRes, error) {
	filter := repositories.BrandFilter{
		Status: query.Status,
		Query:  query.Query,
		Page:   query.Page,
		Limit:  query.Limit,
	}

	result, err := uc.brandRepo.GetBrandsForAdmin(ctx, filter)
	if err != nil {
		return nil, err
	}

	brandIDs := make([]uuid.UUID, len(result.Brands))
	for i, b := range result.Brands {
		brandIDs[i] = b.ID
	}
	counts, err := uc.customerRepo.CountByBrandIDs(ctx, brandIDs)
	if err != nil {
		return nil, err
	}
	totalCustomerByBrandID := make(map[uuid.UUID]int, len(counts))
	for id, cnt := range counts {
		totalCustomerByBrandID[id] = int(cnt)
	}

	mappedItems := mapper.MapBrands(result.Brands, totalCustomerByBrandID)
	metadata := shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount)

	return &shared_dto.PaginationResult[*dto.BrandRes]{
		Items:    mappedItems,
		Metadata: metadata,
	}, nil
}

func (uc *BrandUseCase) GetActiveBrands(ctx context.Context, query dto.GetActiveBrandsQueryReq) (*dto.PublicBrandBriefListRes, error) {
	filter := repositories.BrandFilter{
		Query: query.Query,
		Page:  query.Page,
		Limit: query.Limit,
	}

	result, err := uc.brandRepo.GetActiveFiltered(ctx, filter)
	if err != nil {
		return nil, err
	}

	mappedItems := mapper.MapBrandBriefs(result.Brands)
	metadata := shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount)

	return &shared_dto.PaginationResult[*dto.BrandBriefRes]{
		Items:    mappedItems,
		Metadata: metadata,
	}, nil
}

func (uc *BrandUseCase) GetActiveBrand(ctx context.Context, brandID uuid.UUID) (*dto.BrandRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}
	if brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}
	totalCustomer, err := uc.customerRepo.CountByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand, int(totalCustomer)), nil
}

func (uc *BrandUseCase) GetBrandsForPortalUser(ctx context.Context, userID uuid.UUID) ([]*dto.PortalBrandRes, error) {
	members, err := uc.memberRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	brandIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		brandIDs[i] = m.BrandID
	}
	counts, err := uc.customerRepo.CountByBrandIDs(ctx, brandIDs)
	if err != nil {
		return nil, err
	}
	totalCustomerByBrandID := make(map[uuid.UUID]int, len(counts))
	for id, cnt := range counts {
		totalCustomerByBrandID[id] = int(cnt)
	}
	return mapper.MapPortalBrands(members, totalCustomerByBrandID), nil
}

func (uc *BrandUseCase) GetBrandLogoUploadSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: fmt.Sprintf("%s/%s", uc.cfg.Cloudinary.BrandsFolder, userID.String()),
	})
}

func (uc *BrandUseCase) UpdateBrandImages(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandImagesReq) (*dto.BrandRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, branderrors.ErrBrandNotFound()
	}
	if input.LogoURL != nil {
		brand.LogoURL = input.LogoURL
	}
	if input.LogoPublicID != nil {
		brand.LogoPublicID = input.LogoPublicID
	}
	if input.BackgroundURL != nil {
		brand.BackgroundURL = input.BackgroundURL
	}
	if input.BackgroundPublicID != nil {
		brand.BackgroundPublicID = input.BackgroundPublicID
	}
	brand.UpdatedAt = time.Now().UTC()
	if err := uc.brandRepo.Update(ctx, brand); err != nil {
		return nil, err
	}
	return mapper.MapBrand(brand, -1), nil
}

func (uc *BrandUseCase) GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.PortalBrandRes, error) {
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.Brand == nil {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
	totalCustomer, err := uc.customerRepo.CountByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapPortalBrand(member, int(totalCustomer)), nil
}

func (uc *BrandUseCase) AddBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMembersReq) (*dto.AddBrandMembersRes, error) {
	result := &dto.AddBrandMembersRes{
		Created: []dto.AddBrandMemberItemResult{},
		Updated: []dto.AddBrandMemberItemResult{},
		Failed:  []dto.AddBrandMemberItemResult{},
	}

	identifiersToFetch := make([]string, 0, len(input.Members))
	seenIdentifiers := make(map[string]struct{})
	for _, m := range input.Members {
		idStr := strings.TrimSpace(m.EmailOrUsername)
		norm := strings.ToLower(idStr)
		if norm != "" {
			if _, exists := seenIdentifiers[norm]; !exists {
				seenIdentifiers[norm] = struct{}{}
				identifiersToFetch = append(identifiersToFetch, idStr)
			}
		}
	}

	fetchedUsers, err := uc.userContract.GetByUsernamesOrEmails(ctx, identifiersToFetch)
	if err != nil {
		return nil, err
	}

	userByIdentifier := make(map[string]*identity_dto.UserRes, len(fetchedUsers)*2)
	for _, u := range fetchedUsers {
		if u == nil {
			continue
		}
		if u.Username != "" {
			userByIdentifier[strings.ToLower(u.Username)] = u
		}
		if u.Email != "" {
			userByIdentifier[strings.ToLower(u.Email)] = u
		}
	}

	resolvedUsersByID := make(map[uuid.UUID]struct {
		emailOrUsername string
		role            brandmemberrole.BrandMemberRole
	})
	resolvedUserOrder := make([]uuid.UUID, 0, len(input.Members))
	seenInputs := make(map[string]struct{})

	for _, memberInput := range input.Members {
		identifier := strings.TrimSpace(memberInput.EmailOrUsername)
		normalizedIdentifier := strings.ToLower(identifier)
		if _, exists := seenInputs[normalizedIdentifier]; exists {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "duplicate_input",
				Message:         "Email hoặc tên đăng nhập bị trùng trong danh sách gửi lên.",
			})
			continue
		}
		seenInputs[normalizedIdentifier] = struct{}{}

		if memberInput.Role != brandmemberrole.Staff {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "invalid_role",
				Message:         "API thêm thành viên chỉ cho phép vai trò staff.",
			})
			continue
		}

		user := userByIdentifier[normalizedIdentifier]
		if user == nil {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "user_not_found_or_inactive",
				Message:         "Không tìm thấy user đang hoạt động theo email hoặc tên đăng nhập.",
			})
			continue
		}
		if _, exists := resolvedUsersByID[user.ID]; exists {
			result.Failed = append(result.Failed, dto.AddBrandMemberItemResult{
				EmailOrUsername: identifier,
				ReasonCode:      "duplicate_user",
				Message:         "Email hoặc tên đăng nhập trỏ đến user đã có trong danh sách gửi lên.",
			})
			continue
		}
		resolvedUsersByID[user.ID] = struct {
			emailOrUsername string
			role            brandmemberrole.BrandMemberRole
		}{emailOrUsername: identifier, role: memberInput.Role}
		resolvedUserOrder = append(resolvedUserOrder, user.ID)
	}

	existingMembers, err := uc.memberRepo.GetByBrandAndUserIDs(ctx, brandID, resolvedUserOrder)
	if err != nil {
		return nil, err
	}
	existingByUserID := make(map[uuid.UUID]*entities.BrandMember, len(existingMembers))
	for _, existing := range existingMembers {
		existingByUserID[existing.UserID] = existing
	}

	type pendingResultItem struct {
		emailOrUsername string
		member          *entities.BrandMember
	}
	var toUpdate []*entities.BrandMember
	var pendingUpdates []pendingResultItem
	var toCreate []*entities.BrandMember
	var pendingCreates []pendingResultItem

	for _, resolvedUserID := range resolvedUserOrder {
		resolved := resolvedUsersByID[resolvedUserID]
		if existing := existingByUserID[resolvedUserID]; existing != nil {
			existing.Role = resolved.role
			existing.Status = brandmemberstatus.Active
			toUpdate = append(toUpdate, existing)
			pendingUpdates = append(pendingUpdates, pendingResultItem{
				emailOrUsername: resolved.emailOrUsername,
				member:          existing,
			})
		} else {
			member := &entities.BrandMember{
				BrandID: brandID,
				UserID:  resolvedUserID,
				Role:    resolved.role,
				Status:  brandmemberstatus.Active,
			}
			toCreate = append(toCreate, member)
			pendingCreates = append(pendingCreates, pendingResultItem{
				emailOrUsername: resolved.emailOrUsername,
				member:          member,
			})
		}
	}

	if err := uc.memberRepo.BatchUpdate(ctx, toUpdate); err != nil {
		return nil, err
	}
	for _, item := range pendingUpdates {
		result.Updated = append(result.Updated, dto.AddBrandMemberItemResult{
			EmailOrUsername: item.emailOrUsername,
			Member:          mapper.MapBrandMember(item.member),
		})
	}

	if err := uc.memberRepo.BulkCreate(ctx, toCreate); err != nil {
		return nil, err
	}
	for _, item := range pendingCreates {
		result.Created = append(result.Created, dto.AddBrandMemberItemResult{
			EmailOrUsername: item.emailOrUsername,
			Member:          mapper.MapBrandMember(item.member),
		})
	}

	// Invalidate the membership cache of every affected user so role/status
	// changes take effect immediately (not after the cache TTL).
	for _, userID := range resolvedUserOrder {
		if err := uc.InvalidateUserMembershipCache(ctx, userID); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (uc *BrandUseCase) GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error) {
	members, err := uc.memberRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	userMap := make(map[uuid.UUID]*identity_dto.UserBriefRes)
	if len(members) > 0 {
		userIDs := make([]uuid.UUID, len(members))
		for i, m := range members {
			userIDs[i] = m.UserID
		}
		users, err := uc.userContract.GetBriefByIDs(ctx, userIDs)
		if err == nil {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	return mapper.MapBrandMembersWithUsers(members, userMap), nil
}

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
