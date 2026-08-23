# Frontend Note — Feature 013: BriefRes Response Optimization

**Ngày**: 2026-08-23 | **Loại thay đổi**: Breaking Change

> Breaking change: field cũ bị bỏ, thay thế bằng BriefRes structs mới. Frontend cần cập nhật
> theo danh sách bên dưới. KHÔNG có thay đổi database schema hay request body — chỉ response.

---

## 1. Tổng hợp thay đổi

| Endpoint                                               | Method | Context | Thay đổi thực tế                                                                             | Ưu tiên    |
| ------------------------------------------------------ | ------ | ------- | -------------------------------------------------------------------------------------------- | ---------- |
| `/api/v1/brands`                                       | GET    | Public  | `BrandRes` (13f) → `BrandBriefRes` (6f)                                                      | Cao        |
| `/api/v1/brands/:brandId`                              | GET    | Public  | **KHÔNG đổi** — vẫn `BrandRes` (detail giữ full)                                              | —          |
| `/api/v1/brands/:brandId/items`                        | GET    | Public  | `BrandItemRes` + `FashionItemRes` → `BrandItemBriefRes` + `FashionItemBriefRes`              | Cao        |
| `/api/v1/brands/:brandId/items/samples`                | GET    | User    | `BrandItemRes` + `FashionItemRes` → `BrandItemBriefRes` + `FashionItemBriefRes`              | Cao        |
| `/api/v1/brand-items/:itemId`                          | GET    | Public  | **KHÔNG đổi** — vẫn `BrandItemRes` + `FashionItemRes` (detail giữ full)                      | —          |
| `/api/v1/brands/:brandId/benefits`                     | GET    | Public  | **KHÔNG đổi** — vẫn `BrandBenefitRes`                                                        | —          |
| `/api/v1/brand-benefits/:benefitId`                    | GET    | User    | **KHÔNG đổi**                                                                                | —          |
| `POST /brand-benefits/:benefitId/redeem`               | POST   | User    | Thêm `benefit *BenefitBriefRes` (giữ `benefitId`)                                            | Cao        |
| `GET /api/v1/me/benefit-redemptions`                   | GET    | User    | Thêm `benefit *BenefitBriefRes` (giữ `benefitId`)                                            | Cao        |
| `POST /api/v1/brands/:brandId/join-loyalty`            | POST   | User    | `BrandCustomerRes` (14f) → `BrandCustomerBriefRes` (7f)                                      | Trung bình |
| `POST /api/v1/brands/claim`                            | POST   | User    | `BrandCustomerRes` → `BrandCustomerBriefRes`                                                 | Trung bình |
| `GET /api/v1/brands/:brandId/conversation`             | GET    | User    | `BrandConversationDetailRes` → `BrandConversationUserRes` + msgs (sender brief)              | Trung bình |
| `POST /api/v1/brands/:brandId/conversation/read`       | POST   | User    | **KHÔNG đổi** — vẫn `BrandConversationRes`                                                   | —          |
| `GET /api/v1/me/brand-loyalties`                       | GET    | User    | **KHÔNG đổi**                                                                                | —          |
| `GET /api/v1/me/brand-loyalties/:brandId`              | GET    | User    | **KHÔNG đổi**                                                                                | —          |
| `GET /me/brand-loyalties/:brandId/transactions`        | GET    | User    | `LoyaltyPointTransactionDetailRes` (16f) → `LoyaltyPointTransactionBriefRes` (10f)           | Cao        |
| `GET /api/v1/me/brand-loyalties/:brandId/lots`         | GET    | User    | **KHÔNG đổi**                                                                                | —          |
| `GET /brands/:brandId/loyalty/tiers/:tierId`           | GET    | Public  | **KHÔNG đổi**                                                                                | —          |
| `GET /brand-portal/.../feedbacks`                      | GET    | Portal  | **KHÔNG đổi**                                                                                | —          |
| `PUT /api/v1/me/avatar`                                | PUT    | User    | `UserRes` (full) → `UserBriefRes` (7f)                                                       | Trung bình |
| `GET /api/v1/me/wardrobe-items`                        | GET    | User    | `WardrobeItemRes` (12+16f) → `WardrobeItemBriefRes` (~8f)                                    | Cao        |
| `POST /api/v1/wardrobe-items/{id}/clone`               | POST   | User    | `[]WardrobeItemRes` → `[]WardrobeItemBriefRes`                                               | Cao        |
| `POST /api/v1/wardrobe-items/catalog-init`             | POST   | User    | `[]WardrobeItemRes` → `[]WardrobeItemBriefRes`                                               | Cao        |
| `POST /api/v1/wardrobe-items/batch-upload`             | POST   | User    | `[]WardrobeItemRes` → `[]WardrobeItemBriefRes` + giữ `taskId`                                | Cao        |
| `PATCH /api/v1/wardrobe-items/{id}`                    | PATCH  | User    | **KHÔNG đổi** — vẫn `WardrobeItemRes` (userId/isDeleted vẫn hiện)                            | —          |
| `POST /wardrobe-items/{id}/retry-analysis`             | POST   | User    | **KHÔNG đổi**                                                                                | —          |
| `GET /api/v1/wardrobe-items/{id}`                      | GET    | User    | **KHÔNG đổi cấu trúc** — mapper bỏ populate HSL, timestamps vẫn còn                          | —          |
| `GET /system-catalog/wardrobe-items`                   | GET    | Public  | `fashionItem` đổi sang `FashionItemBriefRes`                                                 | Trung bình |
| `GET /api/v1/me/outfits`                               | GET    | User    | `OutfitRes` → `OutfitBriefRes` (5-6f)                                                        | Trung bình |
| `POST/PUT/GET /api/v1/outfits[/{id}]`                  | \*     | User    | **KHÔNG đổi**                                                                                | —          |
| `POST /ai/outfit-recommendations`                      | POST   | User    | `RecommendedFashionItemRes` (8f) → `RecommendedFashionItemBriefRes` (6f)                     | Cao        |
| SSE batch upload completed                             | SSE    | User    | `data` = full item → brief item                                                              | Cao        |
| `GET /api/v1/ai/chat/sessions`                         | GET    | User    | **KHÔNG đổi** — giữ `ChatSessionRes` (skip T062)                                             | —          |
| `GET /api/v1/subscriptions/me`                         | GET    | User    | Bỏ 3 field `fallback*` + `planKind`/`tierRank`                                               | Cao        |
| `GET /subscriptions/me/daily-quota`                    | GET    | User    | Bỏ `fallback*` + `planKind`/`tierRank` (giữ `maxWardrobeItems`/`maxOutfits`)                 | Cao        |
| `GET /api/v1/subscriptions/plans`                      | GET    | Public  | `SubscriptionPlanDTO` (12f) → `SubscriptionPlanBriefRes` (10f)                               | Trung bình |
| `GET /subscriptions/me/wallet`                         | GET    | User    | **KHÔNG đổi**                                                                                | —          |
| `GET /brand/dashboard/.../digital-sample-lab/{itemId}` | GET    | Brand   | Thêm `brandItemId`/`brandItemName`/`brandItemImageUrl` (đã populate)                       | Trung bình |
| `GET /brand/dashboard/drilldown/sample-lab-feedbacks`  | GET    | Brand   | Thêm `user`/`brandItemName`/`brandItemImageUrl` (đã populate)                               | Trung bình |
| `GET /api/v1/brand/dashboard/kpi-cards`                | GET    | Brand   | Bỏ `BrandID`, `UpdatedAt`                                                                    | Thấp       |
| `GET /api/v1/admin/ai/grants`                          | GET    | Admin   | Thêm `aiCostPolicy *AICostPolicyBriefRes` (giữ `aiCostPolicyId`)                             | Trung bình |

---

## 2. Chi tiết theo Module

### 2.1 Brand Module

#### `GET /api/v1/brands`

- **Cũ**: `BrandRes` — 13 fields (`id`, `slug`, `name`, `description`, `logoUrl`, `backgroundUrl`, `status`, `totalCustomer`, `createdByUserId`, `approvedByUserId`, `approvedAt`, `createdAt`, `updatedAt`)
- **Mới**: `BrandBriefRes` — 6 fields (`id`, `slug`, `name`, `description`, `logoUrl`, `backgroundUrl`)
- **Bỏ**: `status`, `totalCustomer`, `createdByUserId`, `approvedByUserId`, `approvedAt`, `createdAt`, `updatedAt`

> **Lưu ý**: `GET /api/v1/brands/:brandId` (detail) **KHÔNG đổi** — vẫn trả `BrandRes` đầy đủ.

```json
// Cũ
{ "id": "...", "slug": "...", "name": "...", "status": "active", "totalCustomer": 120, "createdByUserId": "...", ... }

// Mới (chỉ list)
{ "id": "...", "slug": "...", "name": "...", "description": "...", "logoUrl": "...", "backgroundUrl": "..." }
```

#### `GET /api/v1/brands/:brandId/items` và `/items/samples`

- **Cũ**: `BrandItemRes` (14f) + nested `FashionItemRes` (16f) = ~30 serialized fields
- **Mới**: `BrandItemBriefRes` + `FashionItemBriefRes` (6f)
- **BrandItemBriefRes giữ**: `id`, `brand` (nested `BrandBriefRes`), `name`, `price`, `itemType`, `status`, `fashionItem` (nested `FashionItemBriefRes`)
- **FashionItemBriefRes giữ**: `id`, `category`, `imageUrl`, `color`, `colorHex`, `style`
- **Bỏ**: `brandId`/`brandName` (thay bằng `brand` nested), `fashionItemId`, `productCode`, `description`, `taskId`, `createdAt`, `updatedAt`, `colorHue`, `colorSaturation`, `colorLightness`, `material`, `pattern`, `fit`, `seasonality`

#### `GET /api/v1/brand-items/:itemId` (detail)

- **KHÔNG ĐỔI** — vẫn trả `BrandItemRes` + `FashionItemRes` (full).

#### `GET /api/v1/brands/:brandId/benefits` và `GET /brand-benefits/:benefitId`

- **KHÔNG ĐỔI** — vẫn trả `BrandBenefitRes` full.

#### `POST /brand-benefits/:benefitId/redeem` và `GET /me/benefit-redemptions`

- **Cũ**: `BenefitRedemptionRes` — chỉ trả `benefitId` (UUID thuần)
- **Mới**: **Thêm** `benefit *BenefitBriefRes` (có `id`, `name`, `description`, `benefitType`, `unlockType`, `requiredPoints`, `featureCode`, `featureConfig`, `status`)
- **Giữ nguyên**: `benefitId`, `brandCustomerId`, `userId`, `createdAt`, `updatedAt` — chỉ THÊM `benefit`, không bỏ gì.

#### `POST /brands/:brandId/join-loyalty` và `POST /brands/claim`

- **Cũ**: `BrandCustomerRes` (14f, có `phoneE164`)
- **Mới**: `BrandCustomerBriefRes` (7f)
- **Giữ**: `id`, `customerName`, `joinedSource`, `status`, `joinedAt`, `user` (UserBriefRes), `loyaltyAccount`
- **Bỏ**: `phoneE164` (PII), `externalCustomerCode`, `claimedAt`, `createdByMemberId`, `createdAt`, `updatedAt`, `brandId`

#### `GET /brands/:brandId/conversation` (User)

- **Cũ**: `BrandConversationDetailRes` (14f)
- **Mới**: `BrandConversationUserRes` — nhúng `brand *BrandBriefRes`
- **Giữ**: `id`, `brand`, `customerName`, `userDisplayName`, `status`, `lastMessageAt`, `userLastReadAt`, `userUnreadCount`
- **Bỏ**: `brandId`, `userId`, `customerId`, `staffLastReadAt`, `staffUnreadCount`, `createdAt`, `updatedAt`
- **Messages**: sender dùng `sender *UserBriefRes` (thêm), vẫn giữ `senderUserId`

> **Lưu ý**: `POST /brands/:brandId/conversation/read` (staff mark-read) **KHÔNG đổi** — vẫn `BrandConversationRes`.

#### `GET /me/brand-loyalties` và `/:brandId`

- **KHÔNG ĐỔI** — vẫn trả `BrandLoyaltyRes` (có `brandCustomerId`, `loyaltyAccountId`).

#### `GET /me/brand-loyalties/:brandId/transactions`

- **Cũ**: `LoyaltyPointTransactionDetailRes` (16f, rò `idempotencyKey`)
- **Mới**: `LoyaltyPointTransactionBriefRes` (10f)
- **Giữ**: `id`, `pointsDelta`, `balanceAfter`, `transactionType`, `reason`, `spendAmount`, `referenceType`, `referenceId`, `expiresAt`, `createdAt`
- **Bỏ**: `loyaltyAccountId`, `brandCustomerId`, `brandId`, `userId`, `createdByUserId`, `idempotencyKey`

#### `GET /me/brand-loyalties/:brandId/lots`

- **KHÔNG ĐỔI** — vẫn giữ `earnTransactionId`.

#### `GET /brands/:brandId/loyalty/tiers/:tierId`

- **KHÔNG ĐỔI** — vẫn trả `LoyaltyTierDetailRes` full + `Benefits []BrandBenefitRes`.

---

### 2.2 Identity Module

#### `PUT /api/v1/me/avatar`

- **Cũ**: `UserRes` (14f + subscription 8f + bodyProfile 6f = ~26 leaf fields)
- **Mới**: `UserBriefRes` (7f)
- **Giữ**: `id`, `username`, `email` (omitempty), `firstName`, `lastName`, `gender`, `avatarUrl`
- **Bỏ**: `roleSlug`, `dateOfBirth`, `address`, `status`, `createdAt`, `subscription`, `bodyProfile`

```json
// Mới
{
    "id": "...",
    "username": "...",
    "firstName": "...",
    "lastName": "...",
    "gender": "male",
    "avatarUrl": "..."
}
```

> **Lưu ý**: `GET /api/v1/me` vẫn giữ nguyên `UserRes` đầy đủ — context profile cần full data.

---

### 2.3 Wardrobe Module

#### `GET /api/v1/me/wardrobe-items` (grid/list)

- **Cũ**: `PaginationResult[WardrobeItemRes]` — mỗi item gồm 12f + `FashionItemRes` 16f = ~28 serialized fields
- **Mới**: `PaginationResult[WardrobeItemBriefRes]` — ~8f + `FashionItemBriefRes` 6f = ~14 fields
- **WardrobeItemBriefRes giữ**: `id`, `price`, `status`, `isLocked`, `lastUsedAt`, `fashionItem` (FashionItemBriefRes), `brandItem` (BrandItemBriefRes), `taskId`
- **Bỏ khỏi WardrobeItemRes**: `userId`, `isDeleted`, `createdAt`
- **Bỏ khỏi FashionItemRes** (khi nhúng trong brief): `colorHue`, `colorSaturation`, `colorLightness`, `material`, `pattern`, `fit`, `seasonality`, `description`, `createdAt`, `updatedAt`

```json
// Cũ (mỗi item trong list)
{ "id": "...", "userId": "...", "isDeleted": false, "price": 100000, "status": "active", "fashionItem": { "id": "...", "imageUrl": "...", "colorHue": 210, "colorSaturation": 0.8, "colorLightness": 0.5, "createdAt": "...", ... }, ... }

// Mới
{ "id": "...", "price": 100000, "status": "active", "isLocked": false, "fashionItem": { "id": "...", "imageUrl": "...", "colorHex": "#4A90D9", "category": { "id": "...", "name": "Áo sơ mi", "slug": "ao-so-mi" } }, "brandItem": null }
```

#### `POST /wardrobe-items/{id}/clone`, `/catalog-init`, `/batch-upload`

- Giống trên: `[]WardrobeItemRes` → `[]WardrobeItemBriefRes`
- **Riêng batch-upload**: response giữ lại `taskId` (cần cho frontend subscribe SSE)

#### `PATCH /wardrobe-items/{id}` và `POST .../retry-analysis`

- **KHÔNG ĐỔI** — vẫn trả `WardrobeItemRes` (userId/isDeleted vẫn xuất hiện).

#### `GET /wardrobe-items/{id}` (detail)

- **KHÔNG ĐỔI cấu trúc** — mapper không còn populate `colorHue`/`colorSaturation`/`colorLightness` (trả null/zero). `createdAt`/`updatedAt` của fashionItem VẪN còn.

#### `GET /system-catalog/wardrobe-items` (public search)

- `fashionItem` đổi từ `*FashionItemRes` (16f) sang `*FashionItemBriefRes` (6f: id, category, imageUrl, color, colorHex, style)

#### `GET /me/outfits` (list)

- **Cũ**: `PaginationResult[OutfitRes]` (10f, `items` = nil)
- **Mới**: `PaginationResult[OutfitBriefRes]` (5-6f)
- **Giữ**: `id`, `name`, `description`, `coverImageUrl`, `status`, `createdAt`
- **Bỏ**: `userId`, `updatedAt`, `items`

#### `POST/PUT/GET /outfits[/{id}]` (detail)

- **KHÔNG ĐỔI** — vẫn trả `OutfitRes` full (có `userId`).

---

### 2.4 Fashion Module

#### `POST /api/v1/ai/outfit-recommendations`

- **Cũ**: `RecommendedFashionItemRes` (8f: `id`, `category`, `imageUrl`, `color`, `colorHex`, `style`, `material`, `pattern`, `fit`)
- **Mới**: `RecommendedFashionItemBriefRes` (6f: bỏ `material`, `pattern`, `fit`)

#### SSE Batch Upload Completed

- **Cũ**: `data` chứa full `WardrobeItemRes` (kèm `FashionItemRes` 16f, HSL, timestamps)
- **Mới**: `data` chứa brief item (id, price, status, isLocked, lastUsedAt, fashionItem brief, brandItem, taskId)

#### `GET /api/v1/ai/chat/sessions`

- **KHÔNG ĐỔI** — giữ `ChatSessionRes` (skip T062: `contextSummary` cần cho UI preview).

---

### 2.5 Subscription Module

#### `GET /api/v1/subscriptions/me`

- **Cũ**: `UserSubscriptionOverviewDTO` (14f)
- **Mới**: 11 fields — bỏ `fallbackPlanCode`, `fallbackTierRank`, `fallbackPlanKind`, `planKind`, `tierRank`

#### `GET /api/v1/subscriptions/me/daily-quota`

- **Cũ**: `UserSubscriptionDTO` (17f)
- **Mới**: 15 fields — bỏ `fallbackPlanCode`, `fallbackTierRank`, `fallbackPlanKind`, `planKind`, `tierRank`
- **Giữ nguyên**: `maxWardrobeItems`, `maxOutfits`, `aiOutfitDailyQuota`, `aiChatDailyQuota`, `outfitRecommendCount`, `aiUsageCount`, `lastResetDate`

```json
// Cũ
{ "planID": "...", "planName": "Pro", "planSlug": "pro", "fallbackPlanCode": "free", "fallbackTierRank": 0, "fallbackPlanKind": "free", "planKind": "premium", "tierRank": 2, "maxWardrobeItems": 100, "maxOutfits": 50, "aiOutfitDailyQuota": 10, ... }

// Mới
{ "planID": "...", "planName": "Pro", "planSlug": "pro", "expiresAt": "...", "isAutoRenewEnabled": true, "maxWardrobeItems": 100, "maxOutfits": 50, "aiOutfitDailyQuota": 10, "aiChatDailyQuota": 20, "outfitRecommendCount": 3, "aiUsageCount": 1, "lastResetDate": "..." }
```

#### `GET /api/v1/subscriptions/plans` (public)

- **Cũ**: `[]SubscriptionPlanDTO` (12f)
- **Mới**: `[]SubscriptionPlanBriefRes` (10f)
- **Bỏ**: `id` (frontend mua theo `slug`), `isActive` (field admin)

#### `GET /api/v1/subscriptions/me/wallet`

- **KHÔNG ĐỔI** — vẫn trả `WalletDTO` full.

---

### 2.6 Dashboard Module

#### `GET /brand/dashboard/.../digital-sample-lab/{itemId}`

- Thêm field `brandItemId`, `brandItemName`, `brandItemImageUrl` vào response
- `PairedCategoryStat` thêm `CategoryID`

#### `GET /brand/dashboard/drilldown/sample-lab-feedbacks`

- Thêm field `user` (`UserBriefRes`), `brandItemName`, `brandItemImageUrl` vào response

#### `GET /api/v1/brand/dashboard/kpi-cards`

- Bỏ `BrandID` và `UpdatedAt` (giữ 8 metric KPI)

---

### 2.7 AI Executor Module

#### `GET /api/v1/admin/ai/grants`

- **Cũ**: `AICostPolicyID` (UUID thuần)
- **Mới**: `AICostPolicy *AICostPolicyBriefRes` (có `id`, `code`, `name`, `version`)

---

## 3. Hướng dẫn Migration cho Frontend

### 3.1 Pattern chung

1. **Tìm tất cả nơi sử dụng field bị bỏ** trong codebase frontend (search theo tên field).
2. **Thêm nested BriefRes bên cạnh UUID còn giữ** — ví dụ `item.benefitId` (giữ) + THÊM `item.benefit.id`, `item.benefit.name`.
3. **Xóa tham chiếu đến field bị bỏ** — không có fallback, field bị xóa hoàn toàn.

### 3.2 Các pattern cần sửa cụ thể

| Pattern cũ                                    | Pattern mới                                                                   |
| --------------------------------------------- | ----------------------------------------------------------------------------- |
| `brand.createdByUserId` (list)                | ❌ Xóa — list giờ dùng `BrandBriefRes`                                         |
| `brand.totalCustomer` (list)                  | ❌ Xóa                                                                        |
| `brand.status` (list)                         | ❌ Xóa — `GET /brands/:brandId` (detail) vẫn có                                |
| `item.fashionItem.colorHue` (brand list)      | ❌ Xóa — dùng `item.fashionItem.colorHex`                                      |
| `item.fashionItem.material` (brand list)      | ❌ Xóa                                                                        |
| `redemption.benefitId`                        | Giữ nguyên `redemption.benefitId` + THÊM `redemption.benefit.name`            |
| `conversation.brandId`                        | `conversation.brand.id` + `conversation.brand.name`                           |
| `message.senderUserId`                        | Giữ `senderUserId` + THÊM `message.sender.id`/`.firstName`/`.avatarUrl`       |
| `loyaltyTransaction.idempotencyKey`           | ❌ Xóa — không hiển thị                                                       |
| `customer.phoneE164`                          | ❌ Xóa — không trả lại cho user                                               |
| `subscription.fallbackPlanCode`               | ❌ Xóa                                                                        |
| `subscription.planKind`/`tierRank`            | ❌ Xóa (subscription overview + daily-quota)                                  |
| `plan.id` (plans list)                        | Dùng `plan.slug` để mua                                                       |
| `plan.isActive` (plans list)                  | ❌ Xóa — field admin                                                          |
| `outfit.userId` (list)                        | ❌ Xóa — list dùng `OutfitBriefRes`                                           |
| `wardrobeItem.userId`/`isDeleted` (list)      | ❌ Xóa — list dùng `WardrobeItemBriefRes` (detail vẫn có)                     |

### 3.3 Các struct mới cần import

| Struct mới                        | Nơi import            | Dùng cho                                             |
| --------------------------------- | --------------------- | ---------------------------------------------------- |
| `UserBriefRes`                    | identity module       | avatar PUT, conversation sender, dashboard drilldown |
| `BrandBriefRes`                   | brand module (đã có)  | brand items list, conversation                           |
| `BenefitBriefRes`                 | brand module          | redemption nested (redeem, benefit-redemptions)          |
| `BrandItemBriefRes`               | brand/wardrobe module | brand items list, wardrobe list brandItem                |
| `BrandCustomerBriefRes`           | brand module          | join-loyalty, claim                                  |
| `BrandConversationUserRes`        | brand module          | conversation endpoints                               |
| `LoyaltyPointTransactionBriefRes` | brand module          | transactions list                                    |
| `WardrobeItemBriefRes`            | wardrobe module       | wardrobe list, clone, batch                          |
| `OutfitBriefRes`                  | wardrobe module       | outfit list                                          |
| `RecommendedFashionItemBriefRes`  | fashion module        | outfit recommendations                               |
| `SubscriptionPlanBriefRes`        | subscription module   | plans list                                           |
| `AICostPolicyBriefRes`            | aiexecutor module     | grants list                                          |

### 3.4 Lưu ý đặc biệt

- **Batch-upload**: response mới có `taskId` trong `WardrobeItemBriefRes` (optional) — frontend vẫn dùng được để subscribe SSE.
- **GET /wardrobe-items/{id}** (detail): vẫn trả `WardrobeItemRes` — `colorHue/Saturation/Lightness` trả null (không populate), `createdAt`/`updatedAt` của fashionItem VẪN còn.
- **GET /brand-items/:itemId** (detail): KHÔNG đổi — vẫn `BrandItemRes` + `FashionItemRes` full.
- **PUT /me/avatar**: response giờ là `UserBriefRes` (7f thay vì 26f) — frontend cần kiểm tra `data.user` hay `data` trực tiếp tùy cách wrap response hiện tại.

---

## 4. Endpoints KHÔNG thay đổi

- **Toàn bộ admin endpoints**: `/api/v1/admin/*` — giữ full fields cho context quản trị.
- **Portal endpoints**: `/brand-portal/*` — giữ nguyên hoàn toàn.
- **Brand public detail**: `GET /api/v1/brands/:brandId` — vẫn `BrandRes` full.
- **Brand item detail**: `GET /api/v1/brand-items/:itemId` — vẫn `BrandItemRes` + `FashionItemRes` full.
- **Brand benefits**: `GET /brands/:id/benefits`, `GET /brand-benefits/:id` — vẫn `BrandBenefitRes`.
- **Brand loyalty lots**: `GET /me/brand-loyalties/:id/lots` — vẫn `LoyaltyPointLotRes`.
- **Brand loyalty tiers**: `GET /brands/:id/loyalty/tiers/:id` — vẫn full.
- **Brand chat staff**: `POST /brands/:id/conversation/read` — vẫn `BrandConversationRes`.
- **Module notification**: service-only, không có API response.
- **User dashboard endpoints**: `GET /me/dashboard/*` — giữ nguyên.
- **Auth endpoints**: `POST /auth/login`, `POST /auth/refresh-token` — không đổi.
- **Identity profile**: `GET /me`, `PUT /me`, `PUT /me/body-profile` — giữ `UserRes` đầy đủ.
- **Payment endpoints**: `POST /me/wallet/topup`, `POST /me/purchase` — không đổi.
- **Category endpoints**: `GET /categories`, admin categories — không đổi.
- **Chat messages**: `GET /ai/chat/sessions/{id}/messages` — không đổi.
- **Wardrobe item detail**: `PATCH /wardrobe-items/{id}`, `POST .../retry-analysis`, `GET /wardrobe-items/{id}` — vẫn `WardrobeItemRes` (chỉ HSL không populate).

---

## 5. Thời gian triển khai

Feature 013 implement theo thứ tự ưu tiên:

1. **Brand module** (ưu tiên cao nhất — nhiều thay đổi nhất, có vấn đề bảo mật `idempotencyKey`)
2. **Identity + Wardrobe module** (tác động lớn đến grid/list performance)
3. **Subscription module** (payload reduction cho daily-quota)
4. **Fashion + Dashboard module** (outfit recommendation + analytics)
5. **AI Executor module** (admin, ít khẩn)
