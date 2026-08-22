# Audit Response DTOs — Module `brand`

> Giai đoạn audit — KHÔNG sửa code. Chỉ ghi nhận phát hiện và đề xuất.
> Phạm vi: `internal/modules/brand/application/dto/` + `internal/modules/brand/presentation/handler/` + `presentation/router/`.

## Tổng quan

Module `brand` là module giàu response DTO nhất hệ thống (23 struct response). Đặc điểm chính:

- **Dùng chung 1 struct cho nhiều context**: `BrandRes`, `BrandItemRes`, `BrandBenefitRes`, `BrandConversationRes`, `LoyaltyPointTransactionDetailRes`, `BrandCustomerRes` đều phục vụ đồng thời admin/portal (cần đầy đủ) **và** endpoint public/user B2C (chỉ cần 3–5 fields). Đây là nguồn overdelivery chính.
- **Endpoints public trả luôn internal/audit fields**: `GET /brands` (public) trả `createdByUserId`, `approvedByUserId`, `approvedAt`, `createdAt`, `updatedAt`; user xem lịch sử điểm nhận cả `idempotencyKey`.
- **Nested thiếu related info**: `BenefitRedemptionRes` chỉ trả `benefitId` (không có tên/loại/featureConfig của benefit), `DigitalSampleResponseRes` chỉ trả `userId` (staff không biết ai feedback), `BrandConversationMessageRes` chỉ trả `senderUserId`.
- **Đã có sẵn pattern BriefRes tốt**: `BrandBriefRes`, `LoyaltyTierBriefRes`, `BrandCustomerUserRes` (kèm avatarUrl). Fashion module đã có `FashionItemBriefRes` (4 fields) — nên tái sử dụng thay vì nhúng `FashionItemRes` (16 fields). Lưu ý module `wardrobe` đã tự định nghĩa `BrandItemBriefRes` riêng (6 fields) — có thể dùng làm tham chiếu shape.

Phân loại endpoint (từ `presentation/router/router.go`):

| Nhóm | Prefix | Xử lý |
|---|---|---|
| Public (không auth) | `/brands`, `/brand-items` (opt-auth) | Đề xuất refactor |
| User B2C (auth) | `/me/*`, `/brands/*`, `/brand-benefits/*` | Đề xuất refactor |
| Portal (staff/owner) | `/brand-portal/*` | Ghi nhận, KHÔNG refactor |
| Admin | `/admin/brands*` | Ghi nhận, KHÔNG refactor |

## Danh sách struct

| # | Struct | Fields | Dùng ở context | Phân loại |
|---|---|---|---|---|
| 1 | `BrandBriefRes` | 6 | Nested trong redemption/loyalty | Brief đã có — có thể rút gọn |
| 2 | `BrandRes` | 13 | Public + Admin + (embed trong PortalBrandRes) | **Overdelivery** (public) |
| 3 | `PortalBrandRes` | 16 | Portal | Portal — không refactor |
| 4 | `BrandMemberRes` | 10 | Portal + Admin | Portal/Admin — không refactor |
| 5 | `AddBrandMembersRes` / `AddBrandMemberItemResult` | 3 / 4 | Portal | Portal — không refactor |
| 6 | `BrandBenefitRes` | 14 | Portal staff + User/Public | **Overdelivery + Underdelivery** (user) |
| 7 | `BenefitRedemptionRes` | 12 | User | **Overdelivery + Underdelivery** |
| 8 | `BrandItemRes` (+ nested `FashionItemRes`) | 14 + 16 | Portal + Public + User | **Overdelivery** (public/user) |
| 9 | `DigitalSampleResponseRes` | 8 | Portal staff + User (submit) | **Underdelivery** (staff) |
| 10 | `BrandCustomerRes` | 14 | Portal staff + User | **Overdelivery** (user) |
| 11 | `BrandCustomerUserRes` | 7 | Nested (đã có avatarUrl) | Brief đã tốt |
| 12 | `CustomerLoyaltyAccountRes` | 5 | Nested | Brief đã tốt |
| 13 | `LoyaltyTierBriefRes` | 2 | Nested | Brief đã tốt |
| 14 | `ClaimTokenRes` | 10 | Portal | Portal — không refactor |
| 15 | `BrandConversationRes` | 14 | User + Portal staff | **Overdelivery** (user) |
| 16 | `BrandConversationMessageRes` | 6 | User + Portal staff | **Underdelivery** (sender) |
| 17 | `BrandConversationDetailRes` | 14 + messages | User + Portal | Đi theo #15 |
| 18 | `LoyaltyPointsTransactionRes` | 9 | Portal staff | Portal — không refactor |
| 19 | `LoyaltyProgramRes` | 9 | Portal | Portal — không refactor |
| 20 | `LoyaltyTierRes` / `LoyaltyTierDetailRes` | 8 / 8+benefits | Portal staff + User | **Overdelivery** (user detail) |
| 21 | `LoyaltyPointLotRes` | 7 | User + Portal staff | Minor (user) |
| 22 | `BrandLoyaltyRes` | 9 | User | Minor (user) |
| 23 | `LoyaltyPointTransactionDetailRes` | 16 | User + Portal staff | **Overdelivery** (user) |

*Các DTO analytics/dashboard (`CustomerAcquisitionDTO`, `LoyaltyTierDistributionDTO`, `PointExpiryForecastDTO`, `BenefitRedemptionDTO`, `CatalogStatsDTO`, `CatalogStatGroupDTO`, `SpendSegmentDTO`, `BrandAdminStatsDTO`, `BrandAnalyticsStatsDTO`, `SampleFeedbackDrillDownDTO`) là aggregate dữ liệu dashboard cho admin/portal → không nằm phạm vi BriefRes, không refactor.*
## Overdelivery

> Ký hiệu: fields giữ (✔) / fields bỏ (✘) cho context public/user. Portal/Admin vẫn giữ nguyên struct hiện tại.

### 1. `BrandRes` (13 fields) — trả cho PUBLIC
Endpoints: `GET /api/v1/brands`, `GET /api/v1/brands/:brandId`.
Context hiển thị (card brand / trang brand B2C): id, slug, name, description, logoUrl, backgroundUrl.
- ✔ Giữ: `id`, `slug`, `name`, `description`, `logoUrl`, `backgroundUrl`
- ✘ Bỏ: `status`, `totalCustomer`, `createdByUserId`, `approvedByUserId`, `approvedAt`, `createdAt`, `updatedAt` (audit/internal, không cần cho B2C display)
- Đề xuất: public list/detail trả `BrandBriefRes` (đã có). `BrandRes` giữ nguyên cho admin + PortalBrandRes.

### 2. `BrandItemRes` (14 fields + nested `FashionItemRes` 16 fields = 30 serialized) — trả cho PUBLIC/USER
Endpoints: `GET /api/v1/brands/:brandId/items`, `GET /api/v1/brands/:brandId/items/samples`, `GET /api/v1/brand-items/:itemId`.
Context hiển thị (product/sample card): ảnh, tên, giá, itemType, brandName, màu, category.
- ✔ Giữ (top-level): `id`, `brandId`, `brandName`, `name`, `price`, `itemType`, `status`
- ✘ Bỏ (top-level): `fashionItemId` (FK), `productCode`, `description`, `taskId` (internal async), `createdAt`, `updatedAt`
- ✘ Nested `FashionItemRes` (16 fields) quá nặng cho card → thay bằng `FashionItemBriefRes` (4 fields: id, imageUrl, colorHex, category) — **đã tồn tại sẵn** ở fashion module.
- Đề xuất: thêm `BrandItemBriefRes` cho list public/user; giữ `BrandItemRes` cho portal staff detail.

### 3. `BrandBenefitRes` (14 fields) — trả cho USER/PUBLIC
Endpoints: `GET /api/v1/brands/:brandId/benefits`, `GET /api/v1/brand-benefits/:benefitId`, nested trong tier detail user.
Context hiển thị (benefit card): tên, mô tả, loại, unlock type, điểm yêu cầu, featureCode/config.
- ✔ Giữ: `id`, `name`, `description`, `benefitType`, `unlockType`, `requiredPoints`, `requiredTierName`, `featureCode`, `featureConfig`
- ✘ Bỏ (list user): `brandId` (có thể giữ hoặc nhúng brand brief), `status` (list đã lọc active), `createdAt`, `updatedAt`
- Đề xuất: `BrandBenefitBriefRes` cho list; giữ `BrandBenefitRes` cho portal staff + detail user.

### 4. `BenefitRedemptionRes` (12 fields) — trả cho USER
Endpoints: `POST /api/v1/brand-benefits/:benefitId/redeem`, `GET /api/v1/me/benefit-redemptions`.
- ✔ Giữ: `id`, `benefit` (BriefRes mới), `pointsSpent`, `status`, `redeemedAt`, `usedAt`, `expiresAt`
- ✘ Bỏ: `benefitId` (thay bằng `benefit` brief), `brandCustomerId` (FK nội bộ), `userId` (context người dùng hiện tại), `createdAt`, `updatedAt`

### 5. `BrandCustomerRes` (14 fields) — trả cho USER
Endpoints: `POST /api/v1/brands/:brandId/join-loyalty`, `POST /api/v1/brands/claim`.
Context: user vừa join/claim chỉ cần trạng thái loyalty + điểm + tên KH.
- ✔ Giữ: `id`, `customerName`, `joinedSource`, `status`, `joinedAt`, `user` (brief có avatar), `loyaltyAccount`
- ✘ Bỏ: `phoneE164` (PII, không cần trả ngược cho user), `externalCustomerCode`, `claimedAt`, `createdByMemberId`, `createdAt`, `updatedAt`, `brandId` (brand đã nằm trong path/context)
- Đề xuất: `BrandCustomerBriefRes` (hoặc tái sử dụng shape `BrandLoyaltyRes`) cho response user; giữ `BrandCustomerRes` cho portal staff.

### 6. `BrandConversationRes` (14 fields) — trả cho USER
Endpoints: `GET /api/v1/brands/:brandId/conversation`, `POST /api/v1/brands/:brandId/conversation/read`.
Context: user xem hội thoại của mình với brand — header chat cần tên/logo brand, không cần data staff.
- ✔ Giữ: `id`, `brand` (BriefRes), `customerName`, `userDisplayName`, `status`, `lastMessageAt`, `userUnreadCount`, `userLastReadAt`
- ✘ Bỏ: `brandId` (nhúng brand brief), `userId`, `customerId`, `staffLastReadAt`, `staffUnreadCount` (data nội bộ staff), `createdAt`, `updatedAt`
- Đề xuất: `BrandConversationUserRes` (user variant) nhúng `Brand *BrandBriefRes`; giữ `BrandConversationRes` cho portal staff.

### 7. `LoyaltyPointTransactionDetailRes` (16 fields) — trả cho USER
Endpoints: `GET /api/v1/me/brand-loyalties/:brandId/transactions`.
Context: user xem lịch sử điểm — cần số điểm, loại gd, lý do, thời gian.
- ✔ Giữ: `id`, `pointsDelta`, `balanceAfter`, `transactionType`, `reason`, `spendAmount`, `referenceType`, `referenceId`, `expiresAt`, `createdAt`
- ✘ Bỏ: `loyaltyAccountId`, `brandCustomerId`, `brandId` (FK nội bộ), `userId`, `createdByUserId`, `idempotencyKey` (**rò rỉ key nội bộ**) — mức ưu tiên cao vì đây là internal token.
- Đề xuất: `LoyaltyPointTransactionBriefRes` cho user; giữ struct đầy đủ cho portal staff.

### 8. `LoyaltyTierDetailRes` (LoyaltyTierRes 8 fields + Benefits) — trả cho USER
Endpoint: `GET /api/v1/brands/:brandId/loyalty/tiers/:tierId` (public, không auth).
- ✘ Bỏ: `createdAt`, `updatedAt` của tier; nested `Benefits` nên dùng `BrandBenefitBriefRes` thay vì `BrandBenefitRes` đầy đủ (status, createdAt, updatedAt không cần).

### 9. `BrandLoyaltyRes` (9 fields) — trả cho USER (minor)
Endpoints: `GET /api/v1/me/brand-loyalties`, `GET /api/v1/me/brand-loyalties/:brandId`.
- ✘ Bỏ (minor): `brandCustomerId`, `loyaltyAccountId` (FK nội bộ, context user). Pattern brief đã tốt (nhúng `BrandBriefRes` + `LoyaltyTierBriefRes`).

### 10. `LoyaltyPointLotRes` (7 fields) — trả cho USER (minor)
Endpoint: `GET /api/v1/me/brand-loyalties/:brandId/lots`.
- ✘ Bỏ (minor): `earnTransactionId` (FK linkage nội bộ) cho user context.

## Underdelivery

### 1. `BenefitRedemptionRes` — thiếu thông tin benefit (CAO)
Chỉ trả `benefitId`. User sau khi redeem/đổi quyền lợi phải gọi thêm `GET /brand-benefits/:benefitId` mới biết mình nhận được gì (tên, loại, voucher/config).
→ Cần nested `Benefit *BenefitBriefRes`.

### 2. `DigitalSampleResponseRes` — thiếu thông tin user (CAO)
Staff xem danh sách feedback (`GET /brand-portal/.../items/:itemId/feedbacks`) chỉ thấy `userId` — không biết ai vote/đánh giá. Màn hình cần tên + avatar.
→ Cần nested `User *UserBriefRes` (id, firstName, lastName, avatarUrl).

### 3. `BrandConversationMessageRes` — thiếu sender brief (TRUNG BÌNH)
Staff xem tin nhắn chỉ thấy `senderUserId` + `senderRole`, không render được avatar/tên người gửi trong giao diện chat.
→ Cần nested `Sender *UserBriefRes`.

### 4. `BrandConversationRes` — thiếu brand brief ở user context (TRUNG BÌNH)
User chat với brand không có tên/logo brand trong response (`brandId` FK trần) → phải gọi thêm brand API cho header chat.
→ Nhúng `Brand *BrandBriefRes` (đề xuất trong user variant ở mục Overdelivery #6).

### 5. `BrandBenefitRes` / tier detail — thiếu brand brief (TRUNG BÌNH)
List benefit/chi tiết tier public không có thông tin brand (tên, logo) → card hiển thị cần gọi thêm.
→ Nhúng `Brand *BrandBriefRes` hoặc chấp nhận brand nằm trong path (tùy UX). Đề xuất nhúng cho detail.

### 6. `BrandMemberRes` — thiếu avatar user (THẤP, chỉ ghi nhận)
Có `userFullName`/`userEmail` nhưng không có `avatarUrl`. Portal không refactor nên chỉ ghi nhận, không đề xuất.

## Đề xuất BriefRes

> Convention đặt tên `<Entity>BriefRes`. Nested object dùng camelCase (constitution §IV). Đa số tái sử dụng DTO đã tồn tại.

| # | Tên mới | Nơi đặt | Fields đề xuất | Dùng cho |
|---|---|---|---|---|
| B1 | `BenefitBriefRes` | `brand/application/dto/benefit.go` | `id`, `name`, `benefitType`, `unlockType`, `requiredPoints`, `requiredTierName`, `featureCode`, `featureConfig`, `status` | `BenefitRedemptionRes.benefit`, `LoyaltyTierDetailRes.benefits`, list benefit user |
| B2 | `BrandItemBriefRes` | `brand/application/dto/brand_item.go` | `id`, `brandId`, `brandName`, `name`, `price`, `itemType`, `status`, `imageUrl`, `colorHex`, `category` (`CategoryBriefRes`) | List product/sample public & user; cần reconcile với `wardrobe.BrandItemBriefRes` (đang duplicate 6 fields) |
| B3 | `UserBriefRes` | `identity/application/dto/` (import cross-module hợp lệ) | `id`, `firstName`, `lastName`, `avatarUrl` | `DigitalSampleResponseRes.user`, `BrandConversationMessageRes.sender`, `BrandConversationRes` user variant |
| B4 | `BrandCustomerBriefRes` | `brand/application/dto/customer.go` | `id`, `customerName`, `joinedSource`, `status`, `joinedAt`, `loyaltyAccount` (`CustomerLoyaltyAccountRes`), `user` (`UserBriefRes`) | `JoinLoyalty`, `ClaimOfflineAccount` (user response) |
| B5 | `BrandConversationUserRes` | `brand/application/dto/chat.go` | `id`, `brand` (`BrandBriefRes`), `customerName`, `userDisplayName`, `status`, `lastMessageAt`, `userLastReadAt`, `userUnreadCount` | `GET /brands/:brandId/conversation`, mark-read user |
| B6 | `LoyaltyPointTransactionBriefRes` | `brand/application/dto/loyalty.go` | `id`, `pointsDelta`, `balanceAfter`, `transactionType`, `reason`, `spendAmount`, `referenceType`, `referenceId`, `expiresAt`, `createdAt` | `GET /me/brand-loyalties/:brandId/transactions` |
| B7 | (rút gọn) `BrandBriefRes` | sửa `brand/application/dto/brand.go` | Giữ `id`, `name`, `slug`, `logoUrl`; bỏ `description`, `backgroundUrl` (không cần cho card brand/avatar) | Mọi nested brand brief |

**Ghi chú reconcile**: `wardrobe/application/dto/wardrobe.go` tự định nghĩa `BrandItemBriefRes` (6 fields) và `FashionItemRes` (16 fields, trùng fashion). Theo constitution §II, được phép import `application/dto` cross-module — đề xuất wardrobe dùng `BrandItemBriefRes` của brand + `FashionItemBriefRes` của fashion để tránh 2 nguồn sự thật. (Ghi nhận, ngoài phạm vi module brand.)

## Ưu tiên

| Mức | Đề xuất | Lý do |
|---|---|---|
| **CAO** | B1 `BenefitBriefRes` cho `BenefitRedemptionRes` | User đổi quyền lợi không biết mình nhận gì → phải gọi thêm API; gãy trải nghiệm voucher/gift. |
| **CAO** | B2 `BrandItemBriefRes` + `FashionItemBriefRes` cho list public/user | Payload lớn nhất (30 fields) trên endpoint public, phí băng thông + lộ `taskId`/analytics AI nội bộ. |
| **CAO** | B6 `LoyaltyPointTransactionBriefRes` | Rò rỉ `idempotencyKey` (internal token) ra endpoint user — vấn đề bảo mật. |
| **CAO** | B3 `UserBriefRes` cho `DigitalSampleResponseRes` | Staff không thể quản lý feedback (ai vote/đánh giá) khi chỉ có `userId`. |
| **CAO** | `BrandRes` → `BrandBriefRes` cho 2 endpoint public | Public trả audit fields `createdByUserId`/`approvedByUserId` — lộ dữ liệu admin. |
| **TB** | B5 `BrandConversationUserRes` | Rò rỉ `staffLastReadAt`/`staffUnreadCount` cho user + thiếu brand header. |
| **TB** | B4 `BrandCustomerBriefRes` | Trả `phoneE164` (PII) ngược cho user sau join/claim. |
| **TB** | B3 cho `BrandConversationMessageRes.sender` | Staff chat không render được avatar/tên người gửi. |
| **TB** | `BrandBenefitRes` list user → `BrandBenefitBriefRes` + brand brief | Giảm payload + thêm brand info cho card benefit. |
| **TB** | `LoyaltyTierDetailRes` user → bỏ internal, benefits dùng `BrandBenefitBriefRes` | Endpoint public không auth trả internal timestamps + status. |
| **THẤP** | B7 rút gọn `BrandBriefRes` | Tối ưu nhỏ; `description`/`backgroundUrl` hiếm khi cần ở nested. |
| **THẤP** | `BrandLoyaltyRes`, `LoyaltyPointLotRes` bỏ FK nội bộ | Vệ sinh nhỏ, không gây hại hiện tại. |

## API bị ảnh hưởng

| Endpoint | Context | Response hiện tại | Response đề xuất |
|---|---|---|---|
| `GET /api/v1/brands` | Public | `BrandRes` (13f) | `BrandBriefRes` (4–6f) |
| `GET /api/v1/brands/:brandId` | Public | `BrandRes` (13f) | `BrandBriefRes` |
| `GET /api/v1/brands/:brandId/items` | Public | `BrandItemRes` + `FashionItemRes` (30f) | `BrandItemBriefRes` + `FashionItemBriefRes` |
| `GET /api/v1/brands/:brandId/items/samples` | User | `BrandItemRes` + `FashionItemRes` | `BrandItemBriefRes` + `FashionItemBriefRes` |
| `GET /api/v1/brand-items/:itemId` | Public/opt-auth | `BrandItemRes` + `FashionItemRes` | `BrandItemRes` (giữ detail, nested `FashionItemBriefRes`) |
| `GET /api/v1/brands/:brandId/benefits` | Public | `BrandBenefitRes` (14f) | `BrandBenefitBriefRes` + `Brand *BrandBriefRes` |
| `GET /api/v1/brand-benefits/:benefitId` | User | `BrandBenefitRes` | `BrandBenefitRes` (detail, thêm brand brief) |
| `POST /api/v1/brand-benefits/:benefitId/redeem` | User | `BenefitRedemptionRes` (thiếu benefit) | `BenefitRedemptionRes` + `Benefit *BenefitBriefRes` |
| `GET /api/v1/me/benefit-redemptions` | User | `BenefitRedemptionRes` | `BenefitRedemptionRes` + `Benefit *BenefitBriefRes` |
| `POST /api/v1/brands/:brandId/join-loyalty` | User | `BrandCustomerRes` (14f, có phone) | `BrandCustomerBriefRes` |
| `POST /api/v1/brands/claim` | User | `BrandCustomerRes` | `BrandCustomerBriefRes` |
| `GET /api/v1/brands/:brandId/conversation` | User | `BrandConversationDetailRes` (14f + msgs) | `BrandConversationUserRes` + msgs (có `Sender *UserBriefRes`) |
| `POST /api/v1/brands/:brandId/conversation/read` | User | `BrandConversationRes` | `BrandConversationUserRes` |
| `GET /api/v1/me/brand-loyalties` | User | `BrandLoyaltyRes` | `BrandLoyaltyRes` (bỏ FK nội bộ) |
| `GET /api/v1/me/brand-loyalties/:brandId` | User | `BrandLoyaltyRes` | `BrandLoyaltyRes` (bỏ FK nội bộ) |
| `GET /api/v1/me/brand-loyalties/:brandId/transactions` | User | `LoyaltyPointTransactionDetailRes` (16f, rò `idempotencyKey`) | `LoyaltyPointTransactionBriefRes` |
| `GET /api/v1/me/brand-loyalties/:brandId/lots` | User | `LoyaltyPointLotRes` | `LoyaltyPointLotRes` (bỏ `earnTransactionId`) |
| `GET /api/v1/brands/:brandId/loyalty/tiers/:tierId` | Public | `LoyaltyTierDetailRes` (full + `BrandBenefitRes`) | `LoyaltyTierRes` (bỏ internal) + `[]BrandBenefitBriefRes` |
| `GET /brand-portal/.../items/:itemId/feedbacks` | Portal staff | `DigitalSampleResponseRes` (chỉ userId) | `DigitalSampleResponseRes` + `User *UserBriefRes` (staff cần danh tính) |

**Không đề xuất refactor (giữ full fields)**: toàn bộ nhóm `/brand-portal/*` (member, customers, loyalty program/tiers, benefits quản trị, claim tokens, items staff, conversations staff, digital sample feedback staff) và `/admin/brands*` (brand list/create/status, members, transfer-ownership) — admin/portal cần đầy đủ data vận hành; chỉ riêng `feedbacks` staff được đề xuất **thêm** (không bớt) `UserBriefRes`.

## Tóm tắt phát hiện chính

1. **10 struct overdelivery** ở context public/user; 3 vấn đề CAO: `BrandRes` lộ audit fields public, `BrandItemRes` payload 30 fields, `LoyaltyPointTransactionDetailRes` rò `idempotencyKey`.
2. **5 điểm underdelivery**; 2 vấn đề CAO: `BenefitRedemptionRes` thiếu benefit brief, `DigitalSampleResponseRes` thiếu user brief cho staff.
3. Đề xuất **7 BriefRes** (B1–B7), tái sử dụng `FashionItemBriefRes` (có sẵn), `BrandBriefRes`, `LoyaltyTierBriefRes`; chú ý reconcile `BrandItemBriefRes` đang duplicate ở wardrobe.
4. 6 struct dùng riêng admin/portal giữ nguyên; cần tách user-variant thay vì sửa chung struct đang dùng chung hai context.
