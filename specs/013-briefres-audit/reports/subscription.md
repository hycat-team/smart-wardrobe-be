# Audit Báo cáo — Module Subscription

**Module**: `internal/modules/subscription`
**Scope**: `application/dto/*` + `presentation/handler/*` + `presentation/router/router.go`
**Giai đoạn**: Audit/analysis — KHÔNG sửa code
**Ngày**: 2026-08-22
**Spec gốc**: `specs/013-briefres-audit/spec.md`

---

## 1. Tổng quan

Module subscription có **7 response struct dùng cho frontend (user-facing)** và **4 struct dùng cho admin/dashboard (không đề xuất refactor)**.

Nhận định chung: **Module đã khá gọn** — các response user-facing gần như không nhúng entity lớn, dữ liệu plan đã được flatten inline (planName, planSlug, planKind, tierRank) nên **không có underdelivery nghiêm trọng** (frontend không phải gọi thêm API để hiển thị). Các vấn đề chủ yếu thuộc nhóm **overdelivery nhẹ**: rò rỉ field hành vi nội bộ (nhóm `fallbackPlan*`) và field hạn mức không đúng context (maxWardrobeItems/maxOutfits ở màn hình daily-quota), field quản trị (`isActive`) xuất hiện ở endpoint public.

Không tồn tại BriefRes nào trong module hiện tại. Đề xuất duy nhất cần tạo mới: `SubscriptionPlanBriefRes`.

---

## 2. Danh sách struct

### User-facing (response cho frontend)

| # | Struct | Nguồn file | Số fields | Endpoint(s) |
|---|--------|-----------|-----------|-------------|
| 1 | `UserSubscriptionOverviewDTO` | `quota_dtos.go` | 14 | `GET /api/v1/subscriptions/me` |
| 2 | `UserSubscriptionDTO` | `quota_dtos.go` | 17 | `GET /api/v1/subscriptions/me/daily-quota` |
| 3 | `SubscriptionPlanDTO` | `billing_dto.go` | 12 | `GET /api/v1/subscriptions/plans` (public) + admin plan endpoints |
| 4 | `WalletDTO` | `billing_dto.go` | 4 | `GET /api/v1/subscriptions/me/wallet` |
| 5 | `WalletStatementDTO` | `billing_dto.go` | 8 | `GET /api/v1/subscriptions/me/wallet/statements` |
| 6 | `PaymentLinkDTO` | `billing_dto.go` | 5 | `POST /me/wallet/topup`, `POST /me/purchase` |
| 7 | `SetAutoRenewRes` (gin.H) | `subscription_handler.go` | 1 | `PUT /me/auto-renew` |

### Admin / Dashboard (KHÔNG đề xuất refactor — giữ full fields)

| # | Struct | Nguồn file | Số fields | Endpoint(s) |
|---|--------|-----------|-----------|-------------|
| 8 | `AdminUserSubscriptionItemDTO` + `AdminUserSubscriptionListRes` | `quota_dtos.go` | 11 | `GET /api/v1/admin/subscriptions` |
| 9 | `DepositTransactionRes` + `AdminDepositTransactionListRes` | `billing_dto.go` | 16 | `GET /admin/subscriptions/transactions`, `GET /:id` |
| 10 | `AdminSubscriptionStatsDTO` | `admin_stats_dto.go` | 3 | internal — feed dashboard module |
| 11 | `SubscriptionDistributionDTO`, `RevenueBreakdownDTO`, `RenewalStatsDTO` | `admin_stats_dto.go` | 2–4 | internal — feed dashboard module |

> Ghi chú: các struct ở `admin_stats_dto.go` là DTO truyền dữ liệu nội bộ sang module dashboard (không phải response API trực tiếp), đồng thời phục vụ context quản trị → không trong scope refactor.

---

## 3. Overdelivery

### 3.1 `UserSubscriptionDTO` — 17 fields → đề xuất 12 fields — **Ưu tiên: cao**

Context: màn hình **daily-quota** (chỉ cần quota AI hôm nay + danh tính gói).

- **Giữ (12)**: `planID`, `planName`, `planSlug`, `planKind`, `tierRank`, `expiresAt`, `isAutoRenewEnabled`, `aiOutfitDailyQuota`, `aiChatDailyQuota`, `outfitRecommendCount`, `aiUsageCount`, `lastResetDate`
- **Bỏ (5)**:
  - `fallbackPlanCode`, `fallbackTierRank`, `fallbackPlanKind` — hành vi nội bộ (gói fallback khi hết hạn), không màn hình nào hiển thị; rò rỉ logic internal ra API.
  - `maxWardrobeItems`, `maxOutfits` — hạn mức tủ đồ/trang phục thuộc context "gói hiện tại" (overview), không liên quan quota hằng ngày.

### 3.2 `UserSubscriptionOverviewDTO` — 14 fields → đề xuất 11 fields — **Ưu tiên: cao**

Context: màn hình **gói hội viên hiện tại** (cần hầu hết field hạn mức).

- **Giữ (11)**: `planID`, `planName`, `planSlug`, `planKind`, `tierRank`, `expiresAt`, `isAutoRenewEnabled`, `maxWardrobeItems`, `maxOutfits`, `aiOutfitDailyQuota`, `aiChatDailyQuota`
- **Bỏ (3)**: `fallbackPlanCode`, `fallbackTierRank`, `fallbackPlanKind` — nội bộ, không hiển thị.

### 3.3 `SubscriptionPlanDTO` — 12 fields (endpoint user-facing) — **Ưu tiên: trung bình**

Context: màn hình **mua gói** (pricing page). Struct dùng chung giữa endpoint public `GET /plans` và 4 endpoint admin quản trị gói → **không sửa struct gốc**, tạo `SubscriptionPlanBriefRes` riêng cho context user (xem §5).

- `SubscriptionPlanBriefRes` (10 fields): bỏ `id` (mua theo `planSlug`), `isActive` (field quản trị nội bộ, rò rỉ ra endpoint **public**).
- `SubscriptionPlanDTO` giữ nguyên cho admin (ListPlans, CreatePlan, UpdatePlan, SetPlanActive).

### 3.4 `WalletDTO` — 4 fields — **Ưu tiên: thấp**

Context: hiển thị số dư ví của chính user đang gọi.
- `userID` dư thừa (ví của chính caller), `updatedAt` không bắt buộc hiển thị.
- Tuy nhiên struct chỉ 4 fields, payload nhỏ → **không đáng tạo BriefRes**; chỉ ghi nhận optional trim.

### 3.5 `WalletStatementDTO` — 8 fields — **Ưu tiên: thấp**

Context: danh sách biến động số dư ví.
- `userID` dư thừa (statement của chính caller). Các field còn lại (amount, transactionType, previousBalance, newBalance, description, createdAt, id) đều cần cho hiển thị.
- 8 fields vẫn ở ngưỡng chấp nhận → optional trim, không bắt buộc.

### 3.6 `PaymentLinkDTO` — 5 fields — Không có vấn đề

Đủ field cho flow thanh toán (paymentUrl, orderCode, paymentStatus, expiresAt, nextReconciliationAt). **Không đề xuất thay đổi.**

---

## 4. Underdelivery

### User-facing: **không có vấn đề nghiêm trọng**

- `UserSubscriptionDTO` / `UserSubscriptionOverviewDTO` đã flatten sẵn plan info (planName, planSlug, planKind, tierRank) → frontend **không phải gọi thêm** `GET /plans` để render. Không bare-FK N+1.
- `WalletStatementDTO.transactionType` là enum + có `description` riêng → đủ để frontend render label/mô tả.
- `PaymentLinkDTO` đủ thông tin polling (orderCode) + link.

### Admin (chỉ ghi nhận — KHÔNG đề xuất refactor)

- `DepositTransactionRes` trả `userId` và `targetPlanId` là bare FK, không kèm tên user / tên gói → admin phải tra cứu thêm để hiển thị tên. Ghi nhận nhưng giữ nguyên full fields theo chính sách admin.
- `AdminUserSubscriptionItemDTO` đã flatten planName/planSlug/planKind → OK, không thiếu field hiển thị quan trọng.

### Gợi ý (tuỳ chọn, ưu tiên thấp)

- Nếu muốn thống nhất convention, có thể nest `SubscriptionPlanBriefRes` vào `UserSubscriptionOverviewDTO`/`UserSubscriptionDTO` thay cho các field plan flatten (planName, planSlug, planKind, tierRank). Đây là tái cấu trúc thuần tuý, **không giải quyết N+1** (dữ liệu đã có sẵn) → chỉ làm nếu team muốn đồng bộ pattern, không bắt buộc.

---

## 5. Đề xuất BriefRes mới

### `SubscriptionPlanBriefRes` — ưu tiên trung bình

Dùng cho **endpoint user-facing** `GET /api/v1/subscriptions/plans` (thay thế `SubscriptionPlanDTO` trong response public). Các endpoint admin giữ `SubscriptionPlanDTO` đầy đủ.

```go
type SubscriptionPlanBriefRes struct {
    Slug               string            `json:"slug"`
    Name               string            `json:"name"`
    Price              float64           `json:"price"`
    MaxWardrobeItems   int               `json:"maxWardrobeItems"`
    MaxOutfits         int               `json:"maxOutfits"`
    AiOutfitDailyQuota int               `json:"aiOutfitDailyQuota"`
    AiChatDailyQuota   int               `json:"aiChatDailyQuota"`
    DurationDays       *int              `json:"durationDays,omitempty"`
    PlanKind           plankind.PlanKind `json:"planKind"`
    TierRank           int               `json:"tierRank"`
}
```

- **Bỏ** so với `SubscriptionPlanDTO`: `id` (mua theo slug), `isActive` (nội bộ quản trị).
- Vị trí đề xuất: `internal/modules/subscription/application/dto/` (cùng file `billing_dto.go` hoặc file mới `plan_dto.go`).
- **Lưu ý**: nên **nghiên cứu trước mapper nguồn plan** — hiện `GetPlans` trả trực tiếp entity plan; cần mapper `MapPlanBrief` (theo pattern `MapBrandBrief`).

### Không cần tạo các BriefRes khác

- Không tạo `UserBriefRes` / `SubscriptionBriefRes` cho admin (ngoài scope refactor).
- Không tạo `PlanBriefRes` nested cho overview/quota (dữ liệu đã flatten, không lợi ích thực tế).

---

## 6. Ưu tiên

| Ưu tiên | Mục | Lý do |
|---------|-----|-------|
| **Cao** | Trim `fallbackPlanCode/TierRank/Kind` khỏi `UserSubscriptionDTO` + `UserSubscriptionOverviewDTO` | Field hành vi nội bộ (gói fallback khi hết hạn) rò rỉ ra API user; không màn hình nào hiển thị; dễ bỏ, không ảnh hưởng UX |
| **Cao** | Trim `maxWardrobeItems`/`maxOutfits` khỏi `UserSubscriptionDTO` | Field hạn mức thuộc context overview, không liên quan daily-quota — giảm payload + giữ đúng ngữ nghĩa màn hình |
| **Trung bình** | Tạo `SubscriptionPlanBriefRes` cho `GET /plans` | Loại `isActive` khỏi endpoint **public** (field quản trị nội bộ); payload giảm ít nhưng đúng separation admin/user |
| **Thấp** | Optional trim `WalletDTO.userID` / `WalletStatementDTO.userID` | Dư thừa nhỏ (ví/statement của chính caller); struct nhỏ nên không tạo BriefRes riêng |
| **Thấp** | Optional nest `SubscriptionPlanBriefRes` vào overview/quota | Đồng bộ convention, không giải quyết N+1 — chỉ làm nếu team muốn |
| — | Admin structs (AdminUserSubscriptionItemDTO, DepositTransactionRes, admin_stats_dto.go) | **Không đề xuất refactor** — giữ full fields cho context quản trị |

---

## 7. API bị ảnh hưởng

| Endpoint | Response hiện tại | Response đề xuất | Thay đổi |
|----------|-------------------|------------------|----------|
| `GET /api/v1/subscriptions/me` | `UserSubscriptionOverviewDTO` (14 fields) | `UserSubscriptionOverviewDTO` (11 fields) | Bỏ 3 field `fallback*` |
| `GET /api/v1/subscriptions/me/daily-quota` | `UserSubscriptionDTO` (17 fields) | `UserSubscriptionDTO` (12 fields) | Bỏ `fallback*` (3) + `maxWardrobeItems`/`maxOutfits` (2) |
| `GET /api/v1/subscriptions/plans` | `[]SubscriptionPlanDTO` (12 fields) | `[]SubscriptionPlanBriefRes` (10 fields) | Bỏ `id`, `isActive` |
| `GET /api/v1/subscriptions/me/wallet` | `WalletDTO` | `WalletDTO` | (Optional) bỏ `userID`, `updatedAt` |
| `GET /api/v1/subscriptions/me/wallet/statements` | `PaginationResult[WalletStatementDTO]` | `PaginationResult[WalletStatementDTO]` | (Optional) bỏ `userID` |
| `POST /api/v1/subscriptions/me/wallet/topup` | `PaymentLinkDTO` | `PaymentLinkDTO` | Không đổi |
| `POST /api/v1/subscriptions/me/purchase` | `PaymentLinkDTO` | `PaymentLinkDTO` | Không đổi |
| `PUT /api/v1/subscriptions/me/auto-renew` | `{isAutoRenewEnabled}` | — | Không đổi |
| `POST /api/v1/subscriptions/me/purchase-with-wallet` | `nil` | — | Không đổi |
| Admin plan/subscription/transaction endpoints | full fields | — | **Không đề xuất refactor** |

> Policy backward compat (spec §FR-008): **breaking change** — bỏ field cũ, thay bằng BriefRes; mọi thay đổi response ở bảng trên phải được ghi nhận vào `frontend-note.md` khi implement xong.