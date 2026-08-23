# Data Model: Audit Response DTOs & BriefRes Pattern

**Feature**: 013-briefres-audit | **Ngày**: 2026-08-22

> Feature 013 gồm 2 pha: (1) AUDIT — xác định vấn đề, KHÔNG đổi database schema; (2) IMPLEMENT — tạo struct BriefRes, trim field, cập nhật mapper/handler/swagger theo ưu tiên. "Data model" ở đây mô tả các struct DTO mới đề xuất (BriefRes) và mối quan hệ nested — chỉ tác động response DTO + mapper, KHÔNG đổi DB schema/entity/request DTO.

## 1. Nguyên tắc chung

- KHÔNG thay đổi DB schema, entity, hoặc request DTO.
- Chỉ thay đổi: response DTO (tạo mới BriefRes / trim field) + mapper.
- Convention đặt tên: <Entity>BriefRes, đặt tại module sở hữu entity gốc.
- Cross-module: được import application/dto (constitution §II). UserBriefRes đặt ở identity, các module khác import.

## 2. Các struct BriefRes đề xuất mới

### brand module

**BenefitBriefRes** (B1)
`go
type BenefitBriefRes struct {
    ID               uuid.UUID  json:"id"
    Name             string     json:"name"
    Description      *string    json:"description"
    BenefitType      string     json:"benefitType"
    UnlockType       string     json:"unlockType"
    RequiredPoints   *int       json:"requiredPoints"
    RequiredTierName *string    json:"requiredTierName,omitempty"
    FeatureCode      *string    json:"featureCode"
    FeatureConfig    any        json:"featureConfig"
    Status           string     json:"status"
}
`

**BrandItemBriefRes** (B2) — reconcile với wardrobe.BrandItemBriefRes
`go
type BrandItemBriefRes struct {
    ID        uuid.UUID        json:"id"
    BrandID   uuid.UUID        json:"brandId"
    BrandName string           json:"brandName"
    Name      string           json:"name"
    Price     *float64         json:"price,omitempty"
    ItemType  string           json:"itemType"
    Status    string           json:"status"
    ImageURL  *string          json:"imageUrl,omitempty"
    ColorHex  *string          json:"colorHex,omitempty"
    Category  *CategoryBriefRes json:"category,omitempty"
}
`

**BrandCustomerBriefRes** (B4)
`go
type BrandCustomerBriefRes struct {
    ID            uuid.UUID                   json:"id"
    CustomerName  string                      json:"customerName"
    JoinedSource  string                      json:"joinedSource"
    Status        brandcustomerstatus.BrandCustomerStatus json:"status"
    JoinedAt      time.Time                   json:"joinedAt"
    LoyaltyAccount *CustomerLoyaltyAccountRes json:"loyaltyAccount,omitempty"
    User          *identity_dto.UserBriefRes  json:"user,omitempty"
}
`

**BrandConversationUserRes** (B5)
`go
type BrandConversationUserRes struct {
    ID               uuid.UUID            json:"id"
    Brand            *BrandBriefRes       json:"brand,omitempty"
    CustomerName     *string              json:"customerName,omitempty"
    UserDisplayName  *string              json:"userDisplayName,omitempty"
    Status           string               json:"status"
    LastMessageAt    *time.Time           json:"lastMessageAt,omitempty"
    UserLastReadAt   *time.Time           json:"userLastReadAt,omitempty"
    UserUnreadCount  int                  json:"userUnreadCount"
}
`

**LoyaltyPointTransactionBriefRes** (B6)
`go
type LoyaltyPointTransactionBriefRes struct {
    ID              uuid.UUID  json:"id"
    PointsDelta     int        json:"pointsDelta"
    BalanceAfter    int        json:"balanceAfter"
    TransactionType string     json:"transactionType"
    Reason          *string    json:"reason"
    SpendAmount     *float64   json:"spendAmount"
    ReferenceType   *string    json:"referenceType"
    ReferenceID     *uuid.UUID json:"referenceId"
    ExpiresAt       *time.Time json:"expiresAt"
    CreatedAt       time.Time  json:"createdAt"
}
`

### identity module

**UserBriefRes** (B3) — chuẩn hóa, thay BrandCustomerUserRes tự duy trì
`go
type UserBriefRes struct {
    ID        uuid.UUID     json:"id"
    Username  string        json:"username"
    Email     string        json:"email,omitempty"
    FirstName string        json:"firstName"
    LastName  string        json:"lastName,omitempty"
    Gender    gender.Gender json:"gender"
    AvatarUrl *string       json:"avatarUrl,omitempty"
}
`

### wardrobe module

**WardrobeItemBriefRes** (B7)
`go
type WardrobeItemBriefRes struct {
    ID          uuid.UUID                 json:"id"
    Price       *float64                  json:"price,omitempty"
    Status      wardrobestatus.WardrobeItemStatus json:"status"
    IsLocked    bool                      json:"isLocked"
    LastUsedAt  *time.Time                json:"lastUsedAt,omitempty"
    FashionItem *fashion_dto.FashionItemBriefRes json:"fashionItem,omitempty"
    BrandItem   *BrandItemBriefRes        json:"brandItem,omitempty"
    TaskID      *uuid.UUID                json:"taskId,omitempty" // optional: batch-upload
}
`

**OutfitBriefRes** (B8)
`go
type OutfitBriefRes struct {
    ID            uuid.UUID                 json:"id"
    Name          string                    json:"name"
    Description   *string                   json:"description"
    CoverImageUrl *string                   json:"coverImageUrl"
    Status        outfitstatus.OutfitStatus json:"status"
    CreatedAt     time.Time                 json:"createdAt"
}
`

### fashion module

**RecommendedFashionItemBriefRes** (B9)
`go
type RecommendedFashionItemBriefRes struct {
    ID       uuid.UUID         json:"id"
    Category *CategoryBriefRes json:"category,omitempty"
    ImageUrl string            json:"imageUrl,omitempty"
    Color    *string           json:"color,omitempty"
    ColorHex *string           json:"colorHex,omitempty"
    Style    *string           json:"style,omitempty"
}
`

**ChatSessionBriefRes** (B10) — chỉ khi chốt contextSummary không cần
`go
type ChatSessionBriefRes struct {
    ID         uuid.UUID json:"id"
    Title      string    json:"title"
    IsArchived bool      json:"isArchived"
    UpdatedAt  time.Time json:"updatedAt"
}
`

### subscription module

**SubscriptionPlanBriefRes** (B11)
`go
type SubscriptionPlanBriefRes struct {
    Slug               string            json:"slug"
    Name               string            json:"name"
    Price              float64           json:"price"
    MaxWardrobeItems   int               json:"maxWardrobeItems"
    MaxOutfits         int               json:"maxOutfits"
    AiOutfitDailyQuota int               json:"aiOutfitDailyQuota"
    AiChatDailyQuota   int               json:"aiChatDailyQuota"
    DurationDays       *int              json:"durationDays,omitempty"
    PlanKind           plankind.PlanKind json:"planKind"
    TierRank           int               json:"tierRank"
}
`

### aiexecutor module

**AICostPolicyBriefRes** (B12)
`go
type AICostPolicyBriefRes struct {
    ID      uuid.UUID json:"id"
    Code    string    json:"code"
    Name    string    json:"name"
    Version int64     json:"version"
}
`

## 3. Mối quan hệ nested (response trước → sau)

| Response hiện tại                 | Field hiện tại               | Field đề xuất                       |
| --------------------------------- | ---------------------------- | ----------------------------------- |
| BenefitRedemptionRes              | benefitId                    | benefit \*BenefitBriefRes           |
| BrandItemRes                      | fashionItem \*FashionItemRes | fashionItem \*FashionItemBriefRes   |
| BrandItemRes (list)               | full                         | BrandItemBriefRes                   |
| DigitalSampleResponseRes          | userId                       | user \*UserBriefRes                 |
| BrandConversationMessageRes       | senderUserId                 | sender \*UserBriefRes               |
| BrandConversationRes (user)       | brandId                      | brand \*BrandBriefRes               |
| BrandCustomerRes (join/claim)     | full (phoneE164)             | BrandCustomerBriefRes               |
| WardrobeItemRes.FashionItem       | \*FashionItemRes (17f)       | \*FashionItemBriefRes               |
| SearchWardrobeItemRes.FashionItem | \*FashionItemRes             | \*FashionItemBriefRes               |
| RecommendedItemRes.FashionItem    | RecommendedFashionItemRes    | RecommendedFashionItemBriefRes      |
| AIPolicyGrantDTO                  | AICostPolicyID               | AICostPolicy \*AICostPolicyBriefRes |
| DigitalSampleLabAnalyticsRes      | ItemID                       | Item \*BrandItemBriefRes            |
| DrillDownSampleFeedbackDTO        | UserID                       | User \*UserBriefRes                 |

## 4. Entities KHÔNG thay đổi

- Không migration DB, không sửa entity GORM.
- Các struct giữ nguyên: BrandRes (admin), PortalBrandRes, BrandMemberRes, ClaimTokenRes, LoyaltyProgramRes, LoyaltyTierRes (portal), FashionItemRes (AI pipeline), SearchDocumentDTO, WardrobeEventPayload, toàn bộ admin/dashboard DTO.
