# Contract: Admin API — Endpoint mới

**Date**: 2026-08-17 | **Feature**: [spec.md](../spec.md)

Tất cả endpoint admin: `auth.Handle()` + `RolesAuthorize(roleslug.Admin)`, prefix `/api/v1`, response theo chuẩn `APIResponse`, lỗi theo `apperror` (message tiếng Việt). Mọi thao tác ghi đều ghi audit log qua `IAuditLogContract`.

---

## Module: subscription — xử lý giao dịch thanh toán bất thường (3A)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/admin/subscriptions/transactions` | Danh sách giao dịch (filter `status`, `fromDate`, `toDate`, `page`, `limit`) |
| GET | `/admin/subscriptions/transactions/:id` | Chi tiết giao dịch |
| POST | `/admin/subscriptions/transactions/:orderCode/retry-investigation` | Chuyển về `reconciliation_required` để xử lý lại (body: `reason`) |
| POST | `/admin/subscriptions/transactions/:orderCode/resolve-paid` | Xác nhận đã thanh toán (body: `reference`, `paymentLinkId`, `reason` — **bắt buộc bằng chứng**) |
| POST | `/admin/subscriptions/transactions/:orderCode/resolve-cancelled` | Xác nhận đã hủy (body: `reason`) |
| POST | `/admin/subscriptions/transactions/:orderCode/resolve-creation-failed` | Đánh dấu tạo link thất bại (body: `reason`) |

- **Input trung tâm**: tái sử dụng `IPaymentResolutionUseCase` (đã wire trong DI) với `ManualPaymentResolutionInput{OrderCode, ExpectedStatus, ActorAdminID, Reason, Evidence}`. `ExpectedStatus` = trạng thái hiện tại đọc từ DB (nếu không khớp → lỗi). `ActorAdminID` = admin userID từ context.
- **validate**: `resolve-paid` thiếu `reference`/`paymentLinkId` → `apperror.NewBadRequest` (không cập nhật ví). Trạng thái không khớp → `ErrInvalidManualPaymentResolution` (message tiếng Việt).
- **Repo**: thêm `IDepositTransactionRepository.ListAdmin(ctx, filter, page, limit)` + `GetByID` (generic có sẵn).

## Module: subscription — plan & user subscription (3B)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/admin/subscriptions/plans` | Danh sách gói (bao gồm inactive) |
| POST | `/admin/subscriptions/plans` | Tạo gói (body: slug, name, price, max_wardrobe_items, max_outfits, ai quotas, duration_days, is_active) |
| PUT | `/admin/subscriptions/plans/:id` | Cập nhật gói |
| PATCH | `/admin/subscriptions/plans/:id/status` | Bật/tắt gói (body: `isActive`) |
| GET | `/admin/subscriptions` | Danh sách đăng ký user (filter `planCode`, `status`, `page`, `limit`) |
| POST | `/admin/subscriptions/:userId/cancel` | Hủy gói → downgrade về free (dùng state machine hiện có) |
| POST | `/admin/subscriptions/:userId/grant` | Cấp gói thủ công (body: `planId`) — đồng bộ grant AI qua luồng `PlanChanged` hiện có |

- **Usecase mở rộng**: `ISubscriptionPlanUseCase` + `CreatePlan/UpdatePlan/SetPlanActive`; `ISubscriptionUseCase` + `AdminList/AdminCancel/AdminGrant`.
- **Repo**: `ISubscriptionPlanRepository` (generic đã đủ) + thêm `ListAdmin`; `IUserSubscriptionRepository` + `ListAdmin(filter)`.
- **Ghi chú**: `AdminGrant` khi gói inactive → lỗi; không tạo mới plan trùng slug (`GetBySlug` kiểm tra).

## Module: identity — quản lý user (3C)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/admin/users/:id` | Chi tiết user + subscription overview |
| POST | `/admin/users` | Tạo user (body: username, email, password, role — role mặc định `user`; admin seed mới tạo được role admin) |
| PATCH | `/admin/users/:id/role` | Đổi role (chỉ role `user`; đổi thành admin chỉ admin seed) |
| DELETE | `/admin/users/:id` | Vô hiệu hóa (soft) — tái sử dụng luồng `UpdateUserStatus(Inactive)` |
| GET | `/admin/users/:id/sessions` | Danh sách phiên (thiết bị, IP, UA, last_active_at, expires_at, trạng thái) |
| POST | `/admin/users/:id/sessions/revoke` | Revoke toàn bộ phiên (RevokeAllByUserID) |

- **Bảo mật (FR-012)**: chặn thao tác lên chính admin đang thực hiện; chỉ admin seed đổi role thành admin; admin thường không thao tác lên tài khoản admin khác.
- **Repo**: `IUserRepository` (generic) + tạo user cần hash password (bcrypt) + đảm bảo username/email duy nhất (đã có luồng register). Session đọc từ `refresh_tokens` (sau migration cột thiết bị).
- **Audit**: `user.create`, `user.role_change`, `user.deactivate`, `user.sessions_revoke`.

## Module: brand — quản lý thành viên (3D)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/admin/brands/:brandId/members` | Danh sách member |
| POST | `/admin/brands/:brandId/members` | Thêm staff (body: `emailOrUsername`) |
| PATCH | `/admin/brands/:brandId/members/:memberId` | Đổi role/status member (body: `role`/`status`) |
| DELETE | `/admin/brands/:brandId/members/:memberId` | Gỡ member |
| POST | `/admin/brands/:brandId/transfer-ownership` | Chuyển ownership (body: `newOwnerUserId`, `oldOwnerHandling: staff|remove`) |

- **Ràng buộc (FR-013/014)**: không gỡ/chuyển khi chỉ còn 1 owner active; transfer chọn xử lý người chủ cũ (hạ staff hoặc gỡ hoàn toàn); không gỡ chính admin-thành-viên bất cẩn (nếu admin cũng là member).
- **Usecase mới**: `IBrandAdminUseCase` (brand module) — ListMembers, AddStaff, UpdateMember, RemoveMember, TransferOwnership (dùng chung repo `IBrandMemberRepository` + invalidate cache).
- **Audit**: `brand.member_add`, `brand.member_role_change`, `brand.member_remove`, `brand.ownership_transfer`.

## Module: aiexecutor — pricing/usage/grant (3E, tầng HTTP mới)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/admin/ai/pricing` | Danh sách cấu hình giá model |
| PUT | `/admin/ai/pricing/:id` | Cập nhật giá (body: `pricePerMillionTokens`, `isActive`) |
| GET | `/admin/ai/usage?userId=&fromDate=&toDate=` | Sử dụng AI theo user (số request, tổng token, chi phí) |
| GET | `/admin/ai/grants?userId=` | Danh sách grant chính sách của user |
| POST | `/admin/ai/grants` | Tạo grant mới (body: `userId`, `planId`/policy, `expiresAt`) — expire grant cũ nếu cần |

- **Repo**: `IAIModelConfigRepository` + `ListAdmin`; `IAICostRepository` + `GetUsageByUser`, `ListGrantsByUser`; tái sử dụng `CreateGrant`, `ExpireGrantsByUserID`.
- **Usecase mới**: `IAIAdminUseCase` (application/interface/usecase/admin_uc.go) + impl.
- **Wire**: thêm handler/router vào aiexecutor provider; đăng ký `AIExecutorRouter` vào AppRouter + NewEngine + wire.go.
- **Audit**: `ai.pricing_update`, `ai.grant_create`.

## Module: admin — audit log (3F)

| Method | Path | Mô tả |
|--------|------|-------|
| GET | `/admin/audit-logs` | Danh sách audit (filter `actorUserId`, `action`, `targetType`, `targetId`, `fromDate`, `toDate`, `page`, `limit`) |

- Chỉ admin truy cập; dữ liệu giữ 12 tháng (cleanup worker).
- Repo: `IAdminAuditLogRepository.Create/List/DeleteOlderThan`.

---

## Chuẩn chung

- **Auth**: `auth.Handle()` (JWT) + `RolesAuthorize(roleslug.Admin)` — `roleslug.Admin` từ shared.
- **Response**: `shared_pres.Success(c, "message tiếng Việt", data)`; lỗi `apperror` (mã 4xx, title/message tiếng Việt).
- **Swagger**: thêm `@Tags` theo module (vd `Admin Subscription`, `Admin AI`), `@Router`, `@Security` — sau đó `make swagger`.
- **DTO**: JSON camelCase; validation binding bằng `binding:"required"` + message tiếng Việt.
- **Wire**: mọi handler/usecase/provider mới phải nằm trong ProviderSet tương ứng → `make wire`.