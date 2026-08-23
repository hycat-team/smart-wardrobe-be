package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandchat/conversationstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandchat/senderrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

type loyaltyLotsTestUOW struct{}

func (u loyaltyLotsTestUOW) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type mockBrandRepo struct {
	brands map[uuid.UUID]*entities.Brand
}

func (m *mockBrandRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Brand, error) {
	return m.brands[id], nil
}
func (m *mockBrandRepo) GetAll(ctx context.Context) ([]*entities.Brand, error)   { return nil, nil }
func (m *mockBrandRepo) Create(ctx context.Context, brand *entities.Brand) error { return nil }
func (m *mockBrandRepo) Update(ctx context.Context, brand *entities.Brand) error { return nil }
func (m *mockBrandRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *mockBrandRepo) GetBySlug(ctx context.Context, slug string) (*entities.Brand, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetActive(ctx context.Context) ([]*entities.Brand, error) { return nil, nil }
func (m *mockBrandRepo) GetActiveFiltered(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	activeStatus := brandstatus.Active
	filter.Status = &activeStatus
	return m.GetBrandsForAdmin(ctx, filter)
}
func (m *mockBrandRepo) GetBrandsForAdmin(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	var list []*entities.Brand
	for _, b := range m.brands {
		if filter.Status != nil && b.Status != *filter.Status {
			continue
		}
		if filter.Query != nil {
			q := strings.ToLower(*filter.Query)
			if !strings.Contains(strings.ToLower(b.Name), q) && !strings.Contains(strings.ToLower(b.Slug), q) {
				continue
			}
		}
		list = append(list, b)
	}
	return &repositories.BrandListResult{
		Brands:     list,
		TotalCount: int64(len(list)),
	}, nil
}
func (m *mockBrandRepo) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error) {
	return nil, 0, nil
}
func (m *mockBrandRepo) BatchUpdate(ctx context.Context, entities []*entities.Brand) error {
	return nil
}
func (m *mockBrandRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (m *mockBrandRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Brand, error) {
	return nil, nil
}

type mockMemberRepo struct {
	members map[string]*entities.BrandMember
}

func (m *mockMemberRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) GetAll(ctx context.Context) ([]*entities.BrandMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) Create(ctx context.Context, member *entities.BrandMember) error { return nil }
func (m *mockMemberRepo) Update(ctx context.Context, member *entities.BrandMember) error { return nil }
func (m *mockMemberRepo) Delete(ctx context.Context, id uuid.UUID) error                 { return nil }
func (m *mockMemberRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error) {
	return m.members[brandID.String()+"_"+userID.String()], nil
}
func (m *mockMemberRepo) GetByBrandAndUserIDs(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	for _, userID := range userIDs {
		if member := m.members[brandID.String()+"_"+userID.String()]; member != nil {
			members = append(members, member)
		}
	}
	return members, nil
}
func (m *mockMemberRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	for _, member := range m.members {
		if member.UserID == userID && member.Status == brandmemberstatus.Active {
			members = append(members, member)
		}
	}
	return members, nil
}
func (m *mockMemberRepo) BulkCreate(ctx context.Context, members []*entities.BrandMember) error {
	return nil
}
func (m *mockMemberRepo) BatchUpdate(ctx context.Context, members []*entities.BrandMember) error {
	return nil
}
func (m *mockMemberRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (m *mockMemberRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.BrandMember, error) {
	return nil, nil
}

type mockCustomerRepo struct {
	customers map[string]*entities.BrandCustomer
}

func (m *mockCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetAll(ctx context.Context) ([]*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) Create(ctx context.Context, customer *entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) Update(ctx context.Context, customer *entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockCustomerRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error) {
	return m.customers[brandID.String()+"_"+userID.String()], nil
}
func (m *mockCustomerRepo) GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandAndExternalCode(ctx context.Context, brandID uuid.UUID, externalCustomerCode string) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandCustomer, error) {
	var list []*entities.BrandCustomer
	for _, cust := range m.customers {
		if cust.UserID != nil && *cust.UserID == userID {
			list = append(list, cust)
		}
	}
	return list, nil
}
func (m *mockCustomerRepo) GetByBrandIDPaginated(ctx context.Context, filter repositories.BrandCustomerFilter) (*repositories.BrandCustomerListResult, error) {
	return nil, nil
}
func (m *mockCustomerRepo) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) CountByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	return map[uuid.UUID]int64{}, nil
}
func (m *mockCustomerRepo) CountByJoinedSource(ctx context.Context, brandID uuid.UUID) ([]repositories.CustomerSourceCount, error) {
	return nil, nil
}
func (m *mockCustomerRepo) CountNewByBrandID(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) CountClaimedByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) BatchUpdate(ctx context.Context, entities []*entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (m *mockCustomerRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.BrandCustomer, error) {
	return nil, nil
}

type mockConvRepo struct {
	conversations map[string]*entities.BrandConversation
}

func (m *mockConvRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandConversation, error) {
	for _, c := range m.conversations {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, nil
}
func (m *mockConvRepo) GetAll(ctx context.Context) ([]*entities.BrandConversation, error) {
	return nil, nil
}
func (m *mockConvRepo) Create(ctx context.Context, conv *entities.BrandConversation) error {
	conv.ID = uuid.New()
	conv.CreatedAt = time.Now().UTC()
	conv.UpdatedAt = time.Now().UTC()
	m.conversations[conv.BrandID.String()+"_"+conv.UserID.String()] = conv
	return nil
}
func (m *mockConvRepo) Update(ctx context.Context, conv *entities.BrandConversation) error {
	m.conversations[conv.BrandID.String()+"_"+conv.UserID.String()] = conv
	return nil
}
func (m *mockConvRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockConvRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandConversation, error) {
	return m.conversations[brandID.String()+"_"+userID.String()], nil
}
func (m *mockConvRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandConversation, error) {
	var list []*entities.BrandConversation
	for _, c := range m.conversations {
		if c.BrandID == brandID {
			list = append(list, c)
		}
	}
	return list, nil
}
func (m *mockConvRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.BrandConversation, error) {
	return m.GetByID(ctx, id)
}
func (m *mockConvRepo) BatchUpdate(ctx context.Context, entities []*entities.BrandConversation) error {
	return nil
}
func (m *mockConvRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (m *mockConvRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.BrandConversation, error) {
	return nil, nil
}

type mockMsgRepo struct {
	messages map[uuid.UUID]*entities.BrandConversationMessage
}

func (m *mockMsgRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandConversationMessage, error) {
	return nil, nil
}
func (m *mockMsgRepo) GetAll(ctx context.Context) ([]*entities.BrandConversationMessage, error) {
	return nil, nil
}
func (m *mockMsgRepo) Create(ctx context.Context, msg *entities.BrandConversationMessage) error {
	msg.ID = uuid.New()
	msg.CreatedAt = time.Now().UTC()
	m.messages[msg.ID] = msg
	return nil
}
func (m *mockMsgRepo) Update(ctx context.Context, msg *entities.BrandConversationMessage) error {
	return nil
}
func (m *mockMsgRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockMsgRepo) GetByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entities.BrandConversationMessage, error) {
	var list []*entities.BrandConversationMessage
	for _, msg := range m.messages {
		if msg.ConversationID == conversationID {
			list = append(list, msg)
		}
	}
	return list, nil
}
func (m *mockMsgRepo) GetByConversationIDPaginated(ctx context.Context, filter repositories.BrandConversationMessageFilter) (*repositories.BrandConversationMessageListResult, error) {
	var list []*entities.BrandConversationMessage
	for _, msg := range m.messages {
		if msg.ConversationID == filter.ConversationID {
			list = append(list, msg)
		}
	}
	return &repositories.BrandConversationMessageListResult{
		Messages:   list,
		TotalCount: int64(len(list)),
	}, nil
}

func (m *mockMsgRepo) CountUnread(ctx context.Context, conversationID uuid.UUID, senderRole string, since *time.Time) (int, error) {
	count := 0
	for _, msg := range m.messages {
		if msg.ConversationID != conversationID || string(msg.SenderRole) != senderRole {
			continue
		}
		if since == nil || msg.CreatedAt.After(*since) {
			count++
		}
	}
	return count, nil
}
func (m *mockMsgRepo) BatchUpdate(ctx context.Context, entities []*entities.BrandConversationMessage) error {
	return nil
}
func (m *mockMsgRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (m *mockMsgRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.BrandConversationMessage, error) {
	return nil, nil
}
func (m *mockMsgRepo) CountUnreadBatch(ctx context.Context, conversationIDs []uuid.UUID, senderRole string, sinceByConv map[uuid.UUID]*time.Time) (map[uuid.UUID]int, error) {
	return nil, nil
}

func TestSendUserMessage_AutoCreateAndReopen(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID: {
			Slug:   "closy",
			Name:   "Closy Brand",
			Status: brandstatus.Active,
		},
	}}

	customerRepo := &mockCustomerRepo{customers: map[string]*entities.BrandCustomer{
		brandID.String() + "_" + userID.String(): {
			BrandID: brandID,
			UserID:  &userID,
			Status:  brandcustomerstatus.Active,
		},
	}}

	convRepo := &mockConvRepo{conversations: make(map[string]*entities.BrandConversation)}
	msgRepo := &mockMsgRepo{messages: make(map[uuid.UUID]*entities.BrandConversationMessage)}

	uc := &BrandChatUseCase{
		brandRepo:    brandRepo,
		customerRepo: customerRepo,
		convRepo:     convRepo,
		msgRepo:      msgRepo,
		uow:          loyaltyLotsTestUOW{},
	}

	input := dto.SendBrandChatMessageReq{
		Message: "Hello Brand!",
	}

	// 1. Send first message -> should create conversation
	res, err := uc.SendUserMessage(context.Background(), userID, brandID, input)
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}

	if res.Message != "Hello Brand!" {
		t.Errorf("Expected message Hello Brand!, got %s", res.Message)
	}
	if res.SenderRole != string(senderrole.Customer) {
		t.Errorf("Expected role CUSTOMER, got %s", res.SenderRole)
	}

	conv, _ := convRepo.GetByBrandAndUser(context.Background(), brandID, userID)
	if conv == nil {
		t.Fatalf("Expected conversation to be created")
	}
	if conv.Status != conversationstatus.Open {
		t.Errorf("Expected status OPEN, got %s", conv.Status)
	}

	// 2. Close conversation manually in mock and send another message -> should reopen
	conv.Status = conversationstatus.Closed
	_ = convRepo.Update(context.Background(), conv)

	res2, err := uc.SendUserMessage(context.Background(), userID, brandID, dto.SendBrandChatMessageReq{Message: "Reopen please"})
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if res2.Message != "Reopen please" {
		t.Errorf("Expected Reopen please, got %s", res2.Message)
	}

	conv2, _ := convRepo.GetByBrandAndUser(context.Background(), brandID, userID)
	if conv2.Status != conversationstatus.Open {
		t.Errorf("Expected status OPEN after reopen, got %s", conv2.Status)
	}
}

func TestSendStaffMessage_Authorization(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()
	staffID := uuid.New()
	convID := uuid.New()

	convRepo := &mockConvRepo{conversations: map[string]*entities.BrandConversation{
		brandID.String() + "_" + userID.String(): {
			AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: convID}},
			BrandID:         brandID,
			UserID:          userID,
			Status:          conversationstatus.Open,
		},
	}}
	msgRepo := &mockMsgRepo{messages: make(map[uuid.UUID]*entities.BrandConversationMessage)}

	uc := &BrandChatUseCase{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		uow:      loyaltyLotsTestUOW{},
	}

	// 1. Valid staff replies
	res, err := uc.SendStaffMessage(context.Background(), staffID, brandID, convID, dto.SendBrandChatMessageReq{Message: "Hello customer"})
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if res.SenderRole != string(senderrole.BrandStaff) {
		t.Errorf("Expected BRAND_STAFF role, got %s", res.SenderRole)
	}
}

func TestSendStaffMessage_ConversationBelongsToBrand(t *testing.T) {
	brandID := uuid.New()
	otherBrandID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()

	convRepo := &mockConvRepo{conversations: map[string]*entities.BrandConversation{
		otherBrandID.String() + "_" + userID.String(): {
			AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: convID}},
			BrandID:         otherBrandID,
			UserID:          userID,
			Status:          conversationstatus.Open,
		},
	}}
	msgRepo := &mockMsgRepo{messages: make(map[uuid.UUID]*entities.BrandConversationMessage)}

	uc := &BrandChatUseCase{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		uow:      loyaltyLotsTestUOW{},
	}

	// Replying to a conversation that belongs to a different brand is rejected
	// at the use case level (business validation), regardless of staff identity.
	_, err := uc.SendStaffMessage(context.Background(), uuid.New(), brandID, convID, dto.SendBrandChatMessageReq{Message: "Intruder"})
	if err == nil {
		t.Fatalf("Expected error for conversation not belonging to brand, got nil")
	}
}
