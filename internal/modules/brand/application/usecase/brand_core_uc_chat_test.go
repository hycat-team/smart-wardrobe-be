package usecase

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandchat/conversationstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandchat/senderrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

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

type mockUserRepo struct{}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return &entities.User{
		Username:  "testuser",
		FirstName: ptr("Test"),
		LastName:  ptr("User"),
	}, nil
}
func (m *mockUserRepo) GetAll(ctx context.Context) ([]*entities.User, error) { return nil, nil }
func (m *mockUserRepo) Create(ctx context.Context, u *entities.User) error   { return nil }
func (m *mockUserRepo) Update(ctx context.Context, u *entities.User) error   { return nil }
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error       { return nil }
func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetStyleProfile(ctx context.Context, id uuid.UUID) (*entities.UserStyleProfile, error) {
	return nil, nil
}
func (m *mockUserRepo) GetByIDWithBodyAndStyle(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return nil, nil
}
func (m *mockUserRepo) CreateStyleProfile(ctx context.Context, profile *entities.UserStyleProfile) error {
	return nil
}
func (m *mockUserRepo) UpdateStyleProfile(ctx context.Context, profile *entities.UserStyleProfile) error {
	return nil
}
func (m *mockUserRepo) CreateAIPolicyGrant(ctx context.Context, grant *entities.UserAIPolicyGrant) error {
	return nil
}
func (m *mockUserRepo) GetAIPolicyGrantByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserAIPolicyGrant, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateAIPolicyGrant(ctx context.Context, grant *entities.UserAIPolicyGrant) error {
	return nil
}
func (m *mockUserRepo) List(ctx context.Context, q string, page int, limit int) ([]*entities.User, int, error) {
	return nil, 0, nil
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

	uc := &BrandCoreUseCase{
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
	otherStaffID := uuid.New()
	convID := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID: {
			Slug:   "closy",
			Name:   "Closy Brand",
			Status: brandstatus.Active,
		},
	}}

	memberRepo := &mockMemberRepo{members: map[string]*entities.BrandMember{
		brandID.String() + "_" + staffID.String(): {
			BrandID: brandID,
			UserID:  staffID,
			Role:    brandmemberrole.Owner,
			Status:  brandmemberstatus.Active,
		},
	}}

	convRepo := &mockConvRepo{conversations: map[string]*entities.BrandConversation{
		brandID.String() + "_" + userID.String(): {
			AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: convID}},
			BrandID:         brandID,
			UserID:          userID,
			Status:          conversationstatus.Open,
		},
	}}
	msgRepo := &mockMsgRepo{messages: make(map[uuid.UUID]*entities.BrandConversationMessage)}

	uc := &BrandCoreUseCase{
		brandRepo:  brandRepo,
		memberRepo: memberRepo,
		convRepo:   convRepo,
		msgRepo:    msgRepo,
		uow:        loyaltyLotsTestUOW{},
	}

	// 1. Valid staff replies
	res, err := uc.SendStaffMessage(context.Background(), staffID, brandID, convID, dto.SendBrandChatMessageReq{Message: "Hello customer"})
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if res.SenderRole != string(senderrole.BrandStaff) {
		t.Errorf("Expected BRAND_STAFF role, got %s", res.SenderRole)
	}

	// 2. Other brand staff replies -> forbidden
	_, err = uc.SendStaffMessage(context.Background(), otherStaffID, brandID, convID, dto.SendBrandChatMessageReq{Message: "Intruder"})
	if err == nil {
		t.Fatalf("Expected forbidden err, got nil")
	}
}
