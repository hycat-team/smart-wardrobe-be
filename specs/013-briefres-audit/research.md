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

## 5. Open questions (chuyển cho plan/dev quyết định)

> Mỗi câu hỏi kèm **Default Answer** — dev có thể xác nhận hoặc thay đổi tại T011-T015 (Phase 2). Nếu không phản đối, implement theo default để tránh rework.

1. **Breaking change contract identity**: đổi IUserContract.GetByID về UserBriefRes hay thêm method GetBriefByIDs giữ UserRes? (ảnh hưởng scope brand)
    - **Default**: Thêm method mới `GetBriefByIDs` trả `*UserBriefRes`, GIỮ `GetByID` trả `*UserRes` (vì brand admin vẫn cần full UserRes). Áp dụng cho các lookup display (member/customer/chat).
    - **✅ QUYẾT ĐỊNH (T011)**: Chọn Option A — thêm `GetBriefByIDs`/`GetBriefByUsernamesOrEmails` trả `*UserBriefRes`, giữ nguyên `GetByID` trả `*UserRes`. **Bổ sung**: rà soát tất cả nơi gọi hàm cũ (brand, dashboard, identity) để xác định nơi nào nên chuyển sang `GetBrief...` (display context) và nơi nào giữ nguyên (admin/full context).
2. **SSE completed**: đổi nguồn phát để data là brief (bỏ `any`)? — cross-module wardrobe contract
    - **Default**: Đổi nguồn phát SSE `completed` sang struct brief (id, imageUrl, colorHex, category, price, status); bỏ `any`.
    - **✅ QUYẾT ĐỊNH (T012)**: Giữ `Data any` trong `SSEEventPayload` (do event dùng chung cho cả wardrobe item lẫn brand item — cấu trúc khác nhau, không ép 1 struct). Đổi NỘI DUNG tại nguồn phát: wardrobe item truyền `WardrobeItemBriefRes` (giữ format như cũ: nested brandItem/fashionItem, chỉ bỏ field thừa userId/isDeleted/taskId/createdAt + HSL/material/pattern/fit/seasonality/description/timestamps), brand item truyền `BrandItemBriefRes`. Frontend parse như cũ, chỉ ít field hơn. Không dùng generics (SSEEventPayload là DTO, `any` đủ cho trường hợp 2 loại payload).
3. **WardrobeItemRes là contract nội bộ** wardrobe↔fashion — xác nhận không co nhỏ, chỉ thêm WardrobeItemBriefRes cho HTTP?
    - **Default**: Không co nhỏ `WardrobeItemRes` (AI pipeline cần full fields); thêm `WardrobeItemBriefRes` cho HTTP display.
    - **✅ QUYẾT ĐỊNH (T013)**: Chọn Option A — KHÔNG co nhỏ `WardrobeItemRes`/`FashionItemRes` (AI pipeline ranking HSL + chat prompt dùng full fields). Thêm `WardrobeItemBriefRes` riêng cho HTTP display (grid/list). Lưu ý: wardrobe có bản copy `FashionItemRes` (wardrobe.go:24) trùng fashion canonical — giữ nguyên trong scope này (dọn nợ riêng, không thuộc phạm vi BriefRes).
4. **Batch-upload**: thêm taskId vào WardrobeItemBriefRes hay struct riêng?
    - **Default**: Thêm `TaskID *uuid.UUID json:"taskId,omitempty"` vào `WardrobeItemBriefRes` (frontend cần subscribe SSE cho batch-upload).
    - **✅ QUYẾT ĐỊNH (T014)**: Chọn Option A — thêm `TaskID *uuid.UUID json:"taskId,omitempty"` vào `WardrobeItemBriefRes`. Dùng chung 1 struct; `taskId` chỉ xuất hiện ở batch-upload, các list/grid thường trả null (omitempty).
5. **FashionItemRes canonical**: chỉnh ở mapper display hay để nguyên (chuyển cho consumer)?
    - **Default**: Giữ nguyên struct canonical; consumer list (wardrobe/brand) dùng `FashionItemBriefRes` ở mapper display — không sửa struct chung.
    - **✅ QUYẾT ĐỊNH (T015)**: 5a — giữ nguyên `FashionItemRes` canonical, consumer list dùng `FashionItemBriefRes` ở mapper. 5b — `OutfitBriefRes` giữ `createdAt` (UI hiển thị ngày tạo outfit list). 5c — `PUT /api/v1/me/avatar` trả `UserBriefRes` thay `UserRes`.
6. **GET /me/outfits**: OutfitBriefRes có cần createdAt?
    - **Default**: Có — giữ `createdAt` trong `OutfitBriefRes` (UI hiển thị ngày tạo outfit list).
7. **/me/avatar**: trả UserBriefRes hay giữ UserRes?
    - **Default**: Trả `UserBriefRes` (client vừa set avatar, không cần full profile).
