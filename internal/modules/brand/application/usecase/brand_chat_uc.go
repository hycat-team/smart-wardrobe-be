package usecase

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandchat/conversationstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandchat/senderrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandChatUseCase struct {
	brandRepo    repositories.IBrandRepository
	memberRepo   repositories.IBrandMemberRepository
	customerRepo repositories.IBrandCustomerRepository
	userRepo     identity_repos.IUserRepository
	convRepo     repositories.IBrandConversationRepository
	msgRepo      repositories.IBrandConversationMessageRepository
	uow          shared_repos.IUnitOfWork
}

func NewBrandChatUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	userRepo identity_repos.IUserRepository,
	convRepo repositories.IBrandConversationRepository,
	msgRepo repositories.IBrandConversationMessageRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandChatUseCase {
	return &BrandChatUseCase{
		brandRepo:    brandRepo,
		memberRepo:   memberRepo,
		customerRepo: customerRepo,
		userRepo:     userRepo,
		convRepo:     convRepo,
		msgRepo:      msgRepo,
		uow:          uow,
	}
}

func (uc *BrandChatUseCase) GetUserConversation(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandConversationRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	conv, err := uc.convRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, branderrors.ErrBrandNotFound()
	}

	user, _ := uc.userRepo.GetByID(ctx, userID)
	userDisp := getUserDisplayName(user)
	return uc.mapBrandConversationWithUnread(ctx, conv, customer.CustomerName, &userDisp)
}

func (uc *BrandChatUseCase) SendUserMessage(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	var messageRes *dto.BrandConversationMessageRes
	now := time.Now().UTC()

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		conv, err := uc.convRepo.GetByBrandAndUser(txCtx, brandID, userID)
		if err != nil {
			return err
		}

		if conv == nil {
			conv = &entities.BrandConversation{
				BrandID:       brandID,
				UserID:        userID,
				Status:        conversationstatus.Open,
				LastMessageAt: &now,
			}
			if err := uc.convRepo.Create(txCtx, conv); err != nil {
				return err
			}
		} else {
			conv.Status = conversationstatus.Open
			conv.LastMessageAt = &now
			conv.ClosedAt = nil
			conv.ClosedByUserID = nil
			if err := uc.convRepo.Update(txCtx, conv); err != nil {
				return err
			}
		}

		msg := &entities.BrandConversationMessage{
			ConversationID: conv.ID,
			SenderUserID:   &userID,
			SenderRole:     senderrole.Customer,
			Message:        strings.TrimSpace(input.Message),
			CreatedAt:      now,
		}
		if err := uc.msgRepo.Create(txCtx, msg); err != nil {
			return err
		}

		messageRes = mapper.MapBrandConversationMessage(msg)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return messageRes, nil
}

func (uc *BrandChatUseCase) ListBrandConversations(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandConversationRes, error) {
	if err := uc.requireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

	conversations, err := uc.convRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	customers, err := uc.customerRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	customerByUserID := make(map[uuid.UUID]*entities.BrandCustomer, len(customers))
	for _, customer := range customers {
		if customer.UserID != nil {
			customerByUserID[*customer.UserID] = customer
		}
	}

	var res []*dto.BrandConversationRes
	for _, conv := range conversations {
		customer := customerByUserID[conv.UserID]
		userDisp := getUserDisplayName(conv.User)
		var custName *string
		if customer != nil {
			custName = customer.CustomerName
		}
		mapped, err := uc.mapBrandConversationWithUnread(ctx, conv, custName, &userDisp)
		if err != nil {
			return nil, err
		}
		res = append(res, mapped)
	}

	return res, nil
}

func (uc *BrandChatUseCase) ListConversationMessages(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) ([]*dto.BrandConversationMessageRes, error) {
	if err := uc.requireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}

	messages, err := uc.msgRepo.GetByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	return mapper.MapBrandConversationMessages(messages), nil
}

func (uc *BrandChatUseCase) SendStaffMessage(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error) {
	if err := uc.requireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}

	var messageRes *dto.BrandConversationMessageRes
	now := time.Now().UTC()

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		lockedConv, err := uc.convRepo.GetByIDForUpdate(txCtx, conversationID)
		if err != nil {
			return err
		}
		if lockedConv == nil {
			return branderrors.ErrBrandNotFound()
		}

		lockedConv.Status = conversationstatus.Open
		lockedConv.LastMessageAt = &now
		lockedConv.ClosedAt = nil
		lockedConv.ClosedByUserID = nil
		if err := uc.convRepo.Update(txCtx, lockedConv); err != nil {
			return err
		}

		msg := &entities.BrandConversationMessage{
			ConversationID: lockedConv.ID,
			SenderUserID:   &staffUserID,
			SenderRole:     senderrole.BrandStaff,
			Message:        strings.TrimSpace(input.Message),
			CreatedAt:      now,
		}
		if err := uc.msgRepo.Create(txCtx, msg); err != nil {
			return err
		}

		messageRes = mapper.MapBrandConversationMessage(msg)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return messageRes, nil
}

func (uc *BrandChatUseCase) MarkUserConversationRead(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandConversationRes, error) {
	conv, err := uc.convRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, branderrors.ErrBrandNotFound()
	}
	now := time.Now().UTC()
	conv.UserLastReadAt = &now
	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, err
	}
	customer, _ := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	user, _ := uc.userRepo.GetByID(ctx, userID)
	userDisp := getUserDisplayName(user)
	var custName *string
	if customer != nil {
		custName = customer.CustomerName
	}
	return uc.mapBrandConversationWithUnread(ctx, conv, custName, &userDisp)
}

func (uc *BrandChatUseCase) MarkStaffConversationRead(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) (*dto.BrandConversationRes, error) {
	if err := uc.requireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	now := time.Now().UTC()
	conv.StaffLastReadAt = &now
	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, err
	}
	return uc.mapStaffConversation(ctx, conv)
}

func (uc *BrandChatUseCase) CloseBrandConversation(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) (*dto.BrandConversationRes, error) {
	if err := uc.requireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	now := time.Now().UTC()
	conv.Status = conversationstatus.Closed
	conv.ClosedAt = &now
	conv.ClosedByUserID = &staffUserID
	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, err
	}
	return uc.mapStaffConversation(ctx, conv)
}

func (uc *BrandChatUseCase) ReopenBrandConversation(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) (*dto.BrandConversationRes, error) {
	if err := uc.requireBrandRole(ctx, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}
	conv.Status = conversationstatus.Open
	conv.ClosedAt = nil
	conv.ClosedByUserID = nil
	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, err
	}
	return uc.mapStaffConversation(ctx, conv)
}

func (uc *BrandChatUseCase) mapStaffConversation(ctx context.Context, conv *entities.BrandConversation) (*dto.BrandConversationRes, error) {
	customer, _ := uc.customerRepo.GetByBrandAndUser(ctx, conv.BrandID, conv.UserID)
	user, _ := uc.userRepo.GetByID(ctx, conv.UserID)
	userDisp := getUserDisplayName(user)
	var custName *string
	if customer != nil {
		custName = customer.CustomerName
	}
	return uc.mapBrandConversationWithUnread(ctx, conv, custName, &userDisp)
}

func (uc *BrandChatUseCase) mapBrandConversationWithUnread(ctx context.Context, conv *entities.BrandConversation, customerName *string, userDisplayName *string) (*dto.BrandConversationRes, error) {
	res := mapper.MapBrandConversation(conv, customerName, userDisplayName)
	if res == nil {
		return nil, nil
	}
	userUnread, err := uc.msgRepo.CountUnread(ctx, conv.ID, string(senderrole.BrandStaff), conv.UserLastReadAt)
	if err != nil {
		return nil, err
	}
	staffUnread, err := uc.msgRepo.CountUnread(ctx, conv.ID, string(senderrole.Customer), conv.StaffLastReadAt)
	if err != nil {
		return nil, err
	}
	res.UserUnreadCount = userUnread
	res.StaffUnreadCount = staffUnread
	return res, nil
}

func (uc *BrandChatUseCase) requireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return err
	}
	if brand == nil {
		return branderrors.ErrBrandNotFound()
	}
	if brand.Status != brandstatus.Active {
		return branderrors.ErrBrandNotActive()
	}
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return err
	}
	if member == nil || member.Status != brandmemberstatus.Active {
		return branderrors.ErrBrandPortalForbidden()
	}
	for _, allowedRole := range allowedRoles {
		if member.Role == allowedRole {
			return nil
		}
	}
	return branderrors.ErrBrandPortalForbidden()
}

func getUserDisplayName(u *entities.User) string {
	if u == nil {
		return ""
	}
	var parts []string
	if u.FirstName != nil && *u.FirstName != "" {
		parts = append(parts, *u.FirstName)
	}
	if u.LastName != nil && *u.LastName != "" {
		parts = append(parts, *u.LastName)
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return u.Username
	}
	return name
}
