# Contract Specification: Other Modules (Phase 2 Standardization)

**Feature**: `018-contract-dto-refactor`
**Date**: 2026-09-03
**Status**: Draft

---

## 1. Identity Module (`IUserBriefContract`)
**Contract Interface**: `internal/modules/identity/presentation/contract/user_brief_contract.go`
**DTO Location**: `internal/modules/identity/application/dto/contract_dto.go`

```go
type IUserBriefContract interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefRes, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefRes, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefRes, error)
}
```
*Ghi chú*: Bản thân `UserBriefRes` đã hoàn toàn pure, trong Phase 2 chỉ cần chuyển định nghĩa struct này hoặc alias sang `contract_dto.go` để bảo đảm chuẩn hóa cấu trúc thư mục toàn dự án.

---

## 2. Subscription Module (`IUserSubscriptionContract`, `IUserQuotaContract`)
**Contract Interface**: `internal/modules/subscription/presentation/contract/subscription_contract.go`
**DTO Location**: `internal/modules/subscription/application/dto/contract_dto.go`

```go
type IUserSubscriptionContract interface {
	GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionOverviewDTO, error)
	GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionOverviewDTO, error)
}

type IUserQuotaContract interface {
	GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error
	UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error
}
```
*Ghi chú*: Di chuyển `UserSubscriptionOverviewDTO` và `UserSubscriptionDTO` vào `contract_dto.go`.

---

## 3. Fashion Module (`IFashionContract`, `ICategoryContract`)
**Contract Interface**: `internal/modules/fashion/presentation/contract/fashion_contract.go`
**DTO Location**: `internal/modules/fashion/application/dto/contract_dto.go`

Tập trung các DTO phục vụ contract vào `contract_dto.go`:
- `FashionItemRes`
- `FashionItemBriefRes`
- `CreateFashionItemInput`
- `FashionProcessingInfoDTO`
- `CategoryRes`

---

## 4. Admin & Notification Modules
- `admin`: `internal/modules/admin/application/dto/contract_dto.go` (`CreateAuditLogReq`)
- `notification`: `internal/modules/notification/application/dto/contract_dto.go` (`SendEmailInput`, `SendTemplateEmailInput`)
