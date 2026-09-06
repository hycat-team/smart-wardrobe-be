package brand

import (
	"context"
	"strings"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

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

	fetchedUsers, err := uc.userBriefContract.GetByUsernamesOrEmails(ctx, identifiersToFetch)
	if err != nil {
		return nil, err
	}

	userByIdentifier := make(map[string]*identity_dto.UserBriefContractDTO, len(fetchedUsers)*2)
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

	userMap := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO)
	if len(members) > 0 {
		userIDs := make([]uuid.UUID, len(members))
		for i, m := range members {
			userIDs[i] = m.UserID
		}
		users, err := uc.userBriefContract.GetByIDs(ctx, userIDs)
		if err == nil {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	return mapper.MapBrandMembersWithUsers(members, userMap), nil
}
