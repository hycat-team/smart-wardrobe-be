package chat

import (
	"context"
	"strings"
	"time"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandchat/conversationstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandchat/senderrole"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

type BrandChatUseCase struct {
	brandRepo         repositories.IBrandRepository
	memberRepo        repositories.IBrandMemberRepository
	customerRepo      repositories.IBrandCustomerRepository
	userBriefContract identity_contract.IUserBriefContract
	convRepo          repositories.IBrandConversationRepository
	msgRepo           repositories.IBrandConversationMessageRepository
	uow               shared_repos.IUnitOfWork
}

func NewBrandChatUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	userBriefContract identity_contract.IUserBriefContract,
	convRepo repositories.IBrandConversationRepository,
	msgRepo repositories.IBrandConversationMessageRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandChatUseCase {
	return &BrandChatUseCase{
		brandRepo:         brandRepo,
		memberRepo:        memberRepo,
		customerRepo:      customerRepo,
		userBriefContract: userBriefContract,
		convRepo:          convRepo,
		msgRepo:           msgRepo,
		uow:               uow,
	}
}

func (uc *BrandChatUseCase) GetUserConversation(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, query dto.GetConversationMessagesQueryReq) (*dto.BrandConversationUserDetailRes, error) {
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

	user, _ := uc.userBriefContract.GetByID(ctx, userID)
	userDisp := getUserDisplayName(user)
	convRes := mapper.MapBrandConversationUser(conv, brand, customer.ID, customer.CustomerName, &userDisp, 0)

	msgResult, err := uc.msgRepo.GetByConversationIDPaginated(ctx, repositories.BrandConversationMessageFilter{
		ConversationID: conv.ID,
		Page:           query.Page,
		Limit:          query.Limit,
	})
	if err != nil {
		return nil, err
	}

	senderIDs := make([]uuid.UUID, 0)
	for _, msg := range msgResult.Messages {
		if msg.SenderUserID != nil {
			senderIDs = append(senderIDs, *msg.SenderUserID)
		}
	}
	senderIDs = sliceutils.UniqueUUIDs(senderIDs)
	senderMap := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO)
	if len(senderIDs) > 0 {
		senderUsers, _ := uc.userBriefContract.GetByIDs(ctx, senderIDs)
		for _, u := range senderUsers {
			senderMap[u.ID] = u
		}
	}

	messages := make([]*dto.BrandConversationMessageRes, len(msgResult.Messages))
	for i, msg := range msgResult.Messages {
		var sender *identity_dto.UserBriefContractDTO
		if msg.SenderUserID != nil {
			sender = senderMap[*msg.SenderUserID]
		}
		messages[i] = mapper.MapBrandConversationMessageWithUser(msg, sender)
	}

	return &dto.BrandConversationUserDetailRes{
		BrandConversationUserRes: *convRes,
		Messages:                 messages,
		Metadata:                 shared_dto.BuildPaginationMetadata(query.PaginationQuery, msgResult.TotalCount),
	}, nil
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

	if messageRes == nil {
		return nil, branderrors.ErrBrandNotFound()
	}

	if uc.userBriefContract != nil {
		senderBrief, _ := uc.userBriefContract.GetByIDs(ctx, []uuid.UUID{userID})
		if len(senderBrief) > 0 {
			messageRes.Sender = mapper.MapIdentityUserBrief(senderBrief[0])
		}
	}
	return messageRes, nil
}

func (uc *BrandChatUseCase) ListBrandConversations(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandConversationRes, error) {
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

	userIDs := make([]uuid.UUID, 0, len(conversations))
	for _, conv := range conversations {
		if conv.UserID != uuid.Nil {
			userIDs = append(userIDs, conv.UserID)
		}
	}
	userMap := map[uuid.UUID]*identity_dto.UserBriefContractDTO{}
	if len(userIDs) > 0 {
		users, err := uc.userBriefContract.GetByIDs(ctx, userIDs)
		if err == nil {
			userMap = sliceutils.IndexBy(users, func(u *identity_dto.UserBriefContractDTO) uuid.UUID { return u.ID })
		}
	}

	convIDs := make([]uuid.UUID, 0, len(conversations))
	userLastReadByConv := make(map[uuid.UUID]*time.Time, len(conversations))
	staffLastReadByConv := make(map[uuid.UUID]*time.Time, len(conversations))
	for _, conv := range conversations {
		convIDs = append(convIDs, conv.ID)
		userLastReadByConv[conv.ID] = conv.UserLastReadAt
		staffLastReadByConv[conv.ID] = conv.StaffLastReadAt
	}

	userUnreadMap, err := uc.msgRepo.CountUnreadBatch(ctx, convIDs, string(senderrole.BrandStaff), userLastReadByConv)
	if err != nil {
		return nil, err
	}
	staffUnreadMap, err := uc.msgRepo.CountUnreadBatch(ctx, convIDs, string(senderrole.Customer), staffLastReadByConv)
	if err != nil {
		return nil, err
	}

	res := make([]*dto.BrandConversationRes, 0, len(conversations))
	for _, conv := range conversations {
		customer := customerByUserID[conv.UserID]
		if customer == nil {
			continue
		}
		user := userMap[conv.UserID]
		userDisp := getUserDisplayNameFromDTO(user)
		mapped := mapper.MapBrandConversation(conv, customer.ID, customer.CustomerName, &userDisp)
		if mapped == nil {
			continue
		}
		mapped.UserUnreadCount = userUnreadMap[conv.ID]
		mapped.StaffUnreadCount = staffUnreadMap[conv.ID]
		res = append(res, mapped)
	}

	return res, nil
}

func (uc *BrandChatUseCase) ListConversationMessages(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID, query dto.GetConversationMessagesQueryReq) (*dto.BrandConversationMessageListRes, error) {
	conv, err := uc.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || conv.BrandID != brandID {
		return nil, branderrors.ErrBrandNotFound()
	}

	result, err := uc.msgRepo.GetByConversationIDPaginated(ctx, repositories.BrandConversationMessageFilter{
		ConversationID: conversationID,
		Page:           query.Page,
		Limit:          query.Limit,
	})
	if err != nil {
		return nil, err
	}

	return &dto.BrandConversationMessageListRes{
		Items:    mapper.MapBrandConversationMessages(result.Messages),
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *BrandChatUseCase) SendStaffMessage(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID, input dto.SendBrandChatMessageReq) (*dto.BrandConversationMessageRes, error) {
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
	user, _ := uc.userBriefContract.GetByID(ctx, userID)
	userDisp := getUserDisplayName(user)
	var custName *string
	if customer != nil {
		custName = customer.CustomerName
	}
	return uc.mapBrandConversationWithUnread(ctx, conv, customer.ID, custName, &userDisp)
}

func (uc *BrandChatUseCase) MarkStaffConversationRead(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, conversationID uuid.UUID) (*dto.BrandConversationRes, error) {
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
	user, _ := uc.userBriefContract.GetByID(ctx, conv.UserID)
	userDisp := getUserDisplayName(user)
	var custName *string
	if customer != nil {
		custName = customer.CustomerName
	}
	return uc.mapBrandConversationWithUnread(ctx, conv, customer.ID, custName, &userDisp)
}

func (uc *BrandChatUseCase) mapBrandConversationWithUnread(ctx context.Context, conv *entities.BrandConversation, customerID uuid.UUID, customerName *string, userDisplayName *string) (*dto.BrandConversationRes, error) {
	res := mapper.MapBrandConversation(conv, customerID, customerName, userDisplayName)
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

func getUserDisplayName(u *identity_dto.UserBriefContractDTO) string {
	if u == nil {
		return ""
	}
	var parts []string
	if u.FirstName != "" {
		parts = append(parts, u.FirstName)
	}
	if u.LastName != "" {
		parts = append(parts, u.LastName)
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return u.Username
	}
	return name
}

func getUserDisplayNameFromDTO(u *identity_dto.UserBriefContractDTO) string {
	if u == nil {
		return ""
	}
	var parts []string
	if u.FirstName != "" {
		parts = append(parts, u.FirstName)
	}
	if u.LastName != "" {
		parts = append(parts, u.LastName)
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return u.Username
	}
	return name
}
