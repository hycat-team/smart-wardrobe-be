# Research: Tổng hợp Audit Response DTOs toàn dự án

**Feature**: 013-briefres-audit | **Ngày**: 2026-08-22 | **Giai đoạn**: Phase 0 research

Phương pháp: 9 sub-agents (1/module) đã quét internal/modules/\*/application/dto/ + handlers. Báo cáo chi tiết tại
eports/. Nội dung dưới đây tổng hợp quyết định và phát hiện chính.

---

## 1. Kết quả phủ sóng theo module

| Module       | Report                  | Kết luận chính                                                                            | Có vấn đề?         |
| ------------ | ----------------------- | ----------------------------------------------------------------------------------------- | ------------------ |
| brand        | reports/brand.md        | 10 overdelivery, 5 underdelivery, đề xuất 7 BriefRes                                      | CÓ (nhiều nhất)    |
| identity     | reports/identity.md     | 1 overdelivery trọng yếu (UserRes qua contract), đề xuất UserBriefRes                     | CÓ                 |
| wardrobe     | reports/wardrobe.md     | WardrobeItemRes nhúng FashionItemRes (16f), đề xuất WardrobeItemBriefRes + OutfitBriefRes | CÓ                 |
| fashion      | reports/fashion.md      | RecommendedFashionItemBriefRes + SSE completed brief                                      | CÓ (2 điểm)        |
| subscription | reports/subscription.md | Trim fallback\*, tạo SubscriptionPlanBriefRes                                             | CÓ (nhẹ)           |
| dashboard    | reports/dashboard.md    | Underdelivery: thiếu Item/User brief ở 2 struct                                           | CÓ (underdelivery) |
| aiexecutor   | reports/aiexecutor.md   | AICostPolicyBriefRes cho grant list                                                       | CÓ (nhẹ)           |
| admin        | reports/admin.md        | Không refactor (policy admin)                                                             | KHÔNG              |
| notification | reports/notification.md | Service-only, không có response DTO                                                       | KHÔNG              |

---

## 2. Quyết định chính (Decision)

### D1. Chính sách tách user-variant thay vì sửa struct dùng chung

- **Decision**: Struct dùng chung 2 context (admin/portal + user/public) → TẠO BriefRes variant cho context user, giữ nguyên struct gốc cho admin/portal.
- **Rationale**: Tránh break admin endpoints; theo đúng pattern BrandBriefRes đã làm ở feature 012.
- **Alternatives**: Sửa struct gốc (loại bỏ vì break admin), tạo generic (không hợp Go).

### D2. UserBriefRes đặt ở identity module

- **Decision**: UserBriefRes (ID, username, firstName, lastName, gender, avatarUrl, email) đặt tại identity/application/dto/, các module khác import qua allowed cross-module import (application/dto).
- **Rationale**: Identity sở hữu entity User; tránh duplicate struct (BrandCustomerUserRes hiện đang tự duy trì ở brand).
- **Alternatives**: Đặt ở brand (sai ownership), duplicate mỗi nơi (phá constitution).

### D3. FashionItemRes canonical KHÔNG trim — xử lý ở consumer

- **Decision**: FashionItemRes (17f) giữ nguyên vì AI pipeline (ranking HSL, chat prompt) tiêu thụ full fields. Consumer list (wardrobe/brand) dùng FashionItemBriefRes (có sẵn).
- **Rationale**: Ràng buộc kiến trúc AI; tránh phá pipeline.
- **Alternatives**: Trim struct chung (break AI), duplicate struct (không cần thiết).

### D4. Breach compat policy

- **Decision**: Breaking change — bỏ field cũ, thay BriefRes. Thông báo qua frontend-note.md ở cuối implement.
- **Rationale**: Theo chốt clarify Q2.

### D5. Ưu tiên triển khai theo module

- **Decision**: brand (P1) → identity+wardrobe (P1) → subscription (P2) → fashion+dashboard (P2) → aiexecutor (P3).
- **Rationale**: brand có nhiều vấn đề bảo mật (idempotencyKey, PII phoneE164, audit fields public); identity/wardrobe ảnh hưởng payload lớn nhất.

---

## 3. Danh sách BriefRes mới đề xuất (tổng hợp 9 reports)

| #   | Struct                          | Module       | Fields                                                                                                                  | Ưu tiên        | Nguồn                    |
| --- | ------------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------------- | -------------- | ------------------------ |
| B1  | BenefitBriefRes                 | brand        | id, name, benefitType, unlockType, requiredPoints, requiredTierName, featureCode, featureConfig, status                 | Cao            | brand                    |
| B2  | BrandItemBriefRes               | brand        | id, brandId, brandName, name, price, itemType, status, imageUrl, colorHex, category                                     | Cao            | brand                    |
| B3  | UserBriefRes                    | identity     | id, username, email, firstName, lastName, gender, avatarUrl                                                             | Cao            | identity/dashboard/brand |
| B4  | BrandCustomerBriefRes           | brand        | id, customerName, joinedSource, status, joinedAt, loyaltyAccount, user                                                  | TB             | brand                    |
| B5  | BrandConversationUserRes        | brand        | id, brand, customerName, userDisplayName, status, lastMessageAt, userLastReadAt, userUnreadCount                        | TB             | brand                    |
| B6  | LoyaltyPointTransactionBriefRes | brand        | id, pointsDelta, balanceAfter, transactionType, reason, spendAmount, referenceType, referenceId, expiresAt, createdAt   | Cao            | brand                    |
| B7  | WardrobeItemBriefRes            | wardrobe     | id, price, status, isLocked, lastUsedAt, fashionItem, brandItem                                                         | Cao            | wardrobe                 |
| B8  | OutfitBriefRes                  | wardrobe     | id, name, description, coverImageUrl, status, createdAt                                                                 | TB             | wardrobe                 |
| B9  | RecommendedFashionItemBriefRes  | fashion      | id, category, imageUrl, color, colorHex, style                                                                          | Cao            | fashion                  |
| B10 | ChatSessionBriefRes             | fashion      | id, title, isArchived, updatedAt                                                                                        | TB (điều kiện) | fashion                  |
| B11 | SubscriptionPlanBriefRes        | subscription | slug, name, price, maxWardrobeItems, maxOutfits, aiOutfitDailyQuota, aiChatDailyQuota, durationDays, planKind, tierRank | TB             | subscription             |
| B12 | AICostPolicyBriefRes            | aiexecutor   | id, code, name, version                                                                                                 | TB             | aiexecutor               |

### Struct BriefRes sẵn có — đánh giá (FR-009)

| Struct                       | Đánh giá                | Hành động                                             |
| ---------------------------- | ----------------------- | ----------------------------------------------------- |
| BrandBriefRes                | Tốt                     | Có thể rút gọn (bỏ description, backgroundUrl) — Thấp |
| LoyaltyTierBriefRes          | Tốt                     | Giữ nguyên                                            |
| CategoryBriefRes (fashion)   | Tốt, trùng với wardrobe | Ghi nhận, dọn nợ riêng                                |
| FashionItemBriefRes          | Tốt                     | Optional thêm color (tên màu) — Thấp                  |
| BrandItemBriefRes (wardrobe) | Tốt                     | Optional thêm brandLogoUrl — Thấp                     |

---

## 4. Các điểm rò rỉ dữ liệu cần xử lý gấp (bảo mật)

| #   | Vấn đề                                                              | Module       | Mức độ |
| --- | ------------------------------------------------------------------- | ------------ | ------ |
| S1  | LoyaltyPointTransactionDetailRes.idempotencyKey rò ra endpoint user | brand        | CAO    |
| S2  | BrandCustomerRes.phoneE164 (PII) trả ngược cho user                 | brand        | CAO    |
| S3  | BrandRes public lộ createdByUserId/approvedByUserId/approvedAt      | brand        | CAO    |
| S4  | BrandConversationRes.staffLastReadAt/staffUnreadCount cho user      | brand        | TB     |
| S5  | UserSubscriptionDTO.fallbackPlan\* rò logic internal                | subscription | TB     |
| S6  | WardrobeItemRes.isDeleted (soft-delete flag) lộ ra API              | wardrobe     | CAO    |

---

## 5. Open questions — ĐÃ GIẢI QUYẾT (Phase 2 + Phase 3-9)

> Tất cả open questions đã được resolve trong Phase 2 (T011-T015) và implement trong Phase 3-9.

1. **Breaking change contract identity** → **✅ ĐÃ IMPLEMENT (T011, T016-T018)**
   Thêm `GetBriefByIDs`/`GetBriefByUsernamesOrEmails` trả `*UserBriefRes`, giữ `GetByID` trả `*UserRes`. Brand member/customer/chat dùng `GetBriefByIDs`.

2. **SSE completed brief** → **✅ ĐÃ IMPLEMENT (T061, Option A)**
   Giữ `Data any` trong SSEEventPayload. Nguồn phát map sang brief trước khi publish:
   - Brand item: `mapBrandItemToBrief()` (worker_uc.go)
   - Wardrobe item: `mapWardrobeItemToBrief()` (worker_uc.go)

3. **WardrobeItemRes contract** → **✅ ĐÃ IMPLEMENT (T013, Phase 4)**
   Không co nhỏ `WardrobeItemRes`/`FashionItemRes` (AI pipeline). Thêm `WardrobeItemBriefRes` cho HTTP list/grid.

4. **Batch-upload taskId** → **✅ ĐÃ IMPLEMENT (T014, Phase 4)**
   Thêm `TaskID *uuid.UUID json:"taskId,omitempty"` vào `WardrobeItemBriefRes`.

5. **FashionItemRes canonical** → **✅ ĐÃ IMPLEMENT (T015, Phase 4)**
   Giữ nguyên canonical, consumer list dùng `FashionItemBriefRes`. `OutfitBriefRes` giữ `createdAt`. `PUT /me/avatar` trả `UserBriefRes`.

6. **GET /me/outfits createdAt** → **✅ ĐÃ IMPLEMENT (T043)**
   `OutfitBriefRes` giữ `createdAt`.

7. **/me/avatar** → **✅ ĐÃ IMPLEMENT (T047)**
   Trả `UserBriefRes` thay `UserRes`.
