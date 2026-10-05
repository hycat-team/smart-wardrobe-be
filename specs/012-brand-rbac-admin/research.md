# Research: Brand RBAC Tập Trung & Bổ Sung Nghiệp Vụ Quản Trị Admin

**Date**: 2026-08-17 | **Feature**: [spec.md](./spec.md)

Nghiên cứu này tổng hợp các quyết định thiết kế và lựa chọn giải pháp cho feature, dựa trên khảo sát codebase (modules, docs, migration) và các câu hỏi đã chốt trong vòng clarify.

---

## R1. Cache kiểm tra membership (thay vì query DB mỗi request)

- **Decision**: Redis cache theo `userID` — giá trị là map `{brandID: role}` cho các membership **active** của các brand **active**. Read-through trong `BrandUseCase.GetUserBrandRole`; TTL mặc định 5 phút (configurable); fallback về DB khi Redis lỗi; invalidation tại các điểm thay đổi membership/brand.
- **Rationale**: 1 user thuộc nhiều brand → cache theo user (không theo cặp user-brand) để 1 lần load dùng cho nhiều request portal/dashboard. Chỉ cache membership hợp lệ (member active + brand active) nên middleware chỉ cần tra map: brandID không có trong map ⇒ không được phép. Giảm tải DB trên hot path (SC-003 ≥90% cache hit). Mẫu có sẵn trong dự án: `aiexecutor/infrastructure/caching/redis_pricing_cache.go` (JSON + Redis, nil-safe khi client nil).
- **Alternatives considered**:
  - *Không cache (giữ `RequireBrandRoleShared`)*: bị loại vì vẫn query DB mỗi request (~2 query: member + brand), trái yêu cầu giảm tải.
  - *Cache in-memory (map + mutex)*: bị loại vì không dùng chung giữa các replica (dự án chạy Docker multi-instance), Redis đã có sẵn.
  - *Cache theo brand*: bị loại vì brand thay đổi ảnh hưởng nhiều user → fan-out invalidation lớn; cache theo user dễ invalidation chính xác.
- **Chi tiết kỹ thuật**:
  - Key: `brand:members:{userID}`; value JSON `{"<brandID>":"owner"|"staff",...}`.
  - `GetUserBrandRole(ctx, userID, brandID)`: đọc cache → hit thì trả role; miss thì `memberRepo.GetActiveByUserID` + filter brand active → build map → set cache → tra cứu. Không có trong map ⇒ `ErrBrandPortalForbidden` (sửa theo FR-008, không còn trả `staff` mặc định).
  - Invalidation (nằm trong module brand): `AddBrandMembers` → `InvalidateByUserID` từng user; `UpdateBrandStatus` (admin) → `InvalidateByBrandID` (fan-out qua `GetByBrandID`); các endpoint admin member mới (thêm/đổi role/gỡ/transfer) → `InvalidateByUserID`. identity `UpdateUserStatus` Inactive không cần gọi invalidation (đã blacklist token ở tầng auth + TTL cache) — tránh coupling identity→brand.
  - Khi Redis lỗi: `Get` trả nil (miss) → đọc DB như bình thường (không chặn user hợp lệ).

## R2. Thiết kế middleware Brand RBAC

- **Decision**: `BrandRoleMiddleware` trong `brand/presentation/middleware/brand_role.go`, expose qua `IBrandRoleMiddleware` (brand/presentation/contract) để dashboard dùng. Hai factory: `PathAuthorize(brandIDParam, allowedRoles...)` (portal, brandId từ path) và `QueryAuthorize(brandIDParam, allowedRoles...)` (dashboard, brandId từ query). Cả hai nhận functional option `AllowAdmin(bool)` — portal=false, dashboard=true.
- **Rationale**: Đưa toàn bộ check 403 về middleware (A4 chốt khả thi: mọi route portal có `:brandId` trong path; worker nền không đi qua các method đang check). Dashboard dùng query variant vì `brandId` nằm ở query param. Expose qua contract để tuân thủ G1 (dashboard không import file middleware trực tiếp). Admin bypass ở dashboard giữ hiện trạng (đã chốt).
- **Alternatives considered**:
  - *Giữ check trong usecase + chỉ thêm middleware cho dashboard*: bị loại vì trái yêu cầu "tập trung về middleware" và không gỡ được ~30 check.
  - *Place middleware trong shared*: bị loại vì shared không được phụ thuộc business module (G2); middleware phải ở brand module.
- **Chi tiết**:
  - Flow: lấy `userID` từ context → resolve brandID (path/query) → nếu `AllowAdmin && role==admin` thì set context (brandId, brandRole=admin) và `Next()` → ngược lại gọi `IBrandUseCase.GetUserBrandRole` → role không nằm trong `allowedRoles` ⇒ `apperror.NewForbidden` → set context `CtxBrandId`, `CtxBrandRole` (chuẩn `contextutils`) → `Next()`.
  - Context: thêm hằng số `CtxBrandId`, `CtxBrandRole` vào `internal/pkg/utils/contextutils/key.go` và getter `GetBrandID`, `GetBrandRole` vào `user.go` — không hardcode (A2).
  - Router portal: nhóm `member` (Owner+Staff) cho toàn bộ route portal (trừ 3 route tạo brand/me/profile-signature không có `:brandId`); nhóm `owner` (Owner) cho `PUT /loyalty/program`, `POST /loyalty/tiers`, `PUT /loyalty/tiers/:tierId` (hợp nghiệp vụ: cấu hình loyalty, A3).
  - Gỡ `RequireBrandRoleShared` khỏi ~30 usecase HTTP (loyalty/benefit/item/claim/chat/brand); giữ helper cho luồng nội bộ nếu cần.

## R3. Theo dõi phiên đăng nhập (session tracking)

- **Decision**: Thêm cột `device_name`, `device_ip`, `user_agent`, `last_active_at` vào `identity.refresh_tokens` qua migration; ghi nhận tại thời điểm login và refresh.
- **Rationale**: Bảng `refresh_tokens` hiện chỉ có user_id/token/expires/is_revoked/created_at — không đủ để hiển thị phiên hữu ích (thiết bị/IP) cho admin quản lý phiên (FR-011, Q4 chốt Option A). `last_active_at` hỗ trợ phát hiện phiên bất thường.
- **Alternatives considered**: *Không đổi schema, chỉ liệt kê token* (Option B): bị loại vì không đáp ứng nhu cầu vận hành; *bỏ list chi tiết* (Option C): bị loại.
- **Chi tiết**: header `User-Agent` + `X-Forwarded-For`/RemoteAddr tại `Login` và `RefreshToken` trong `session_uc.go` (hoặc handler) → lưu vào refresh token row. Không lưu raw token. Thay đổi DB này gộp chung **1 file migration** với bảng `admin_audit_logs` để hạn chế file nhỏ lẻ.

## R4. Audit log (module `admin` mới, ghi bất đồng bộ qua outbox + RabbitMQ)

- **Decision**: Tạo module mới `internal/modules/admin` sở hữu entity/repo/usecase/contract/consumer worker/cleanup worker cho `admin_audit_logs` (bảng ở schema `shared`). Ghi audit **bất đồng bộ qua outbox + RabbitMQ**: handler các module gọi `IAuditLogContract.Record` → ghi `outbox_records` (outbox) → `OutboxWorker` publish tới topic exchange → consumer module `admin` (`audit_log_consumer`) ghi `admin_audit_logs`. Giữ 12 tháng + cleanup worker; chỉ admin xem qua endpoint.
- **Rationale**: Đáp ứng hiến pháp §III (write side-effect xuyên module ưu tiên event/outbox; cấu hình §III yêu cầu outbox transactional). Ghi audit không chặn nghiệp vụ chính, at-least-once (outbox bền vững kể cả broker sự cố). C1 chốt: ưu tiên cơ chế bất đồng bộ qua outbox + RabbitMQ.
- **Alternatives considered**:
  - *Ghi đồng bộ qua `IAuditLogContract.Record` (direct DB write)*: bị loại — trái tinh thần hiến pháp §III về cross-module write side-effect.
  - *Đặt entity audit trong `internal/shared`*: bị loại vì shared không nên chứa nghiệp vụ.
  - *Gộp endpoint xem audit vào dashboard/identity*: bị loại vì đặt nghiệp vụ audit vào module không liên quan.
- **Chi tiết**: bảng `shared.admin_audit_logs` (id, actor_user_id FK→identity.users, action, target_type, target_id, payload jsonb, request_ip, created_at) + index (actor_user_id), (target_type,target_id), (created_at). Topic/queue mới: `eventconstants.TopicAuditLog = "admin.audit.log"`, `QueueAuditLog = "audit_log_queue"`, `RoutingKeyAuditLog` (bind trong `rabbitmq_connection.go`). Contract `Record` chỉ publish outbox (không ghi DB); consumer `audit_log_consumer` unmarshal payload → `IAuditLogUseCase.Record` → ghi bảng. Cleanup worker xóa bản ghi `created_at < now()-12mo`. Handler: `GET /admin/audit-logs` (lọc actor/action/target/date + phân trang).

## R5. Tầng HTTP mới cho module aiexecutor

- **Decision**: Dựng `aiexecutor/presentation/handler/admin_handler.go` + `aiexecutor/presentation/router/router.go` (group `/admin/ai/*`, auth + RolesAuthorize(Admin)); mở rộng `IAICostRepository` (+ `GetUsageByUser`, `ListGrantsByUser`) và `IAIModelConfigRepository` (+ `ListAdmin`); thêm `AIAdminUseCase` (application) với ListPricing/UpdatePricing/GetUserUsage/ListGrants/CreateGrant (create grant kèm expire grant cũ qua `ExpireGrantsByUserID`). Đăng ký handler/router vào provider aiexecutor + AppRouter + wire.go + NewEngine.
- **Rationale**: Module hiện chỉ có contract + worker, không route nào cho admin quản trị pricing/usage/grant (lỗ hổng vận hành). Thêm tầng presentation là chuẩn kiến trúc module (G2), không vi phạm ranh giới vì chỉ dùng interface nội bộ + DTO.
- **Alternatives considered**: *Bỏ qua AI executor admin (chỉ dashboard)*: bị loại vì dashboard không cho sửa pricing/grant; *đưa endpoint vào module khác*: bị loại vì vi phạm module ownership.
- **Chi tiết**: `PUT /admin/ai/pricing/:id` cập nhật `price_per_million_tokens`, `is_active`; cache pricing Redis cần invalidate hoặc để TTL ngắn.

## R6. Placement endpoint admin theo module sở hữu

- **Decision**: Endpoint admin mới nằm ở module sở hữu nghiệp vụ: payment/plan/subscription → subscription; user/session → identity; brand member/ownership → brand; AI pricing/usage/grant → aiexecutor; audit log → admin module. Không tạo 1 "admin module" tổng gộp tất cả route admin (tránh refactor route admin hiện có).
- **Rationale**: Nhất quán với module ownership đã áp dụng (spec 010) — mỗi module tự đăng ký endpoint của nó; không làm breaking change route admin hiện có.
- **Alternatives considered**: *Tạo module `admin` gộp mọi endpoint admin*: bị loại vì phải di chuyển route admin hiện có (identity/wardrobe/brand/dashboard) → breaking + ngoài phạm vi; chỉ dùng module `admin` riêng cho audit log (cross-cutting mới).

## R7. Đồng bộ docs-code (lấy code làm chuẩn)

- **Decision**: Sửa 4 mâu thuẫn: (1) `dashboard-api-v2.md` `DIRECT_PURCHASE`→`direct_purchase`; (2) plan code trong `dashboard-api-v2.md` → khớp `free`/`premium-monthly`; (3) `architecture.md` khái niệm "B2C Roles" → "hạng theo gói subscription (PlanKind)"; (4) `brand-api.md:480` lỗi liệt kê role cũ + thống nhất ma trận owner/staff (brand-api, loyalty-api, constants/brand.md reference về 1 ma trận tại `docs/system-design/brand-permission-matrix.md`).
- **Rationale**: Code là nguồn sự thật (seed plan `03-seed.sql`, constants package `deposittransactiontype`, `roleslug`). Một ma trận duy nhất tránh tái lặp mâu thuẫn (B3: đặt ở `docs/system-design/`).
- **Alternatives considered**: *Sửa code theo docs*: bị loại vì code là chuẩn đang chạy (người dùng yêu cầu); *giữ 2 nguồn*: bị loại vì tiếp tục mâu thuẫn.

## R8. Ghi out-of-scope vào docs

- **Decision**: ADR mới `docs/decisions/004-out-of-scope-2026-08.md` liệt kê 9 mục (RBAC toàn hệ thống giữ `users.role_slug`, campaign, support ticket, virtual trial, notification đa kênh + quản lý, swap/re-roll AI, AI OOTD generator, body-shape validation, garment decay) kèm lý do; cập nhật `docs/product/roadmap.md` + `docs/README.md` trỏ tới ADR.
- **Rationale**: Người dùng yêu cầu đưa out-scope vào docs; ADR là đúng quy ước `docs/decisions/`.
- **Alternatives considered**: *Chỉ ghi ở roadmap*: bị loại vì roadmap mô tả tương lai, ADR ghi quyết định rõ ràng hơn.

## R9. Quy tắc bảo mật admin (kết quả clarify)

- **Decision**: (1) Chỉ admin seed tạo/phong admin; endpoint đổi role không cho đổi thành admin; admin thường chặn thao tác lên admin khác/chính mình. (2) `resolve-as-paid` bắt buộc bằng chứng (Reference + PaymentLinkID), không override. (3) Transfer ownership: admin chọn hạ staff hoặc gỡ hoàn toàn người chủ cũ; chặn chuyển khi chỉ còn 1 owner. (4) Audit log 12 tháng + cleanup, admin-only.
- **Rationale**: Cân bằng giữa vận hành và bảo mật; giữ RBAC hệ thống đơn giản (2 role) nhưng có "admin seed" làm điểm kiểm soát phong admin.

## R10. Chuẩn codegen

- **Decision**: Sau khi thêm provider/handler/usecase mới: chạy `make wire` (sinh `wire_gen.go`) và `make swagger` (cập nhật spec API + module split). Kiểm soát bằng `make generate-check` trong CI.
- **Rationale**: Dự án dùng Wire + Swag như cơ chế sinh chuẩn (Makefile, spec 010); bất kỳ provider/handler mới đều phải đi qua generate.