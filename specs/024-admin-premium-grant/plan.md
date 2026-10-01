# Implementation Plan: Một chủ sở hữu mức quyền tính phí AI và công cụ cấp gói Premium cho quản trị viên

**Branch**: `024-admin-premium-grant` | **Date**: 2026-09-29 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification từ `/specs/024-admin-premium-grant/spec.md`

## Summary

Chuyển bảng `aiexecutor.ai_policy_grants` về **một chủ sở hữu duy nhất là hệ thống AI**. Hệ thống Account hiện ghi thẳng vào bảng này bằng SQL thô trong cùng giao dịch (vi phạm hiến pháp §I/§II, không bị trình biên dịch bắt), song song với một đường hàng đỗi **hỏng** — bản ghi tạo ra thiếu `policy_snapshot` bắt buộc nên ghi vào cơ sở dữ liệu thất bại mà không có dấu vết. Song song với đó, ba đường đổi gói (cấp thủ công, thu hồi thủ công, cấp phát gói mặc định) không sinh bản ghi lịch sử nên không ai nhận được tín hiệu.

Kỹ thuật: hệ thống Account chỉ ghi bản ghi lịch sử + bản ghi outbox trong một giao dịch; hệ thống AI là bên **duy nhất** đóng bản ghi cũ và tạo bản ghi mới, tự dựng ảnh chụp chính sách từ bảng chính sách của chính nó, chống xử lý trùng bằng `sourceEventId`. Trước khi cắt đường ghi cũ, dựng hai tầng suy giảm để hệ thống không bao giờ báo lỗi vì thiếu mức quyền tính phí. Cuối cùng thay thế thao tác cấp gói đơn lẻ bằng **một** giao diện lập trình nhận danh sách 1–500 tài khoản, tự nhận diện mã định danh (UUID / tên đăng nhập / email).

Triển khai theo đúng thứ tự GĐ0 → GĐ8; GĐ5 là ranh giới không đảo ngược được nên phải kiểm chứng trước khi gộp.

## Technical Context

**Language/Version**: Go 1.26.0 — `go.work`, `services/ai/go.mod` và `services/account/go.mod` đều khai báo `go 1.26.0`, đã khớp nên `go work sync` không cảnh báo.

**Primary Dependencies**: Gin, Google Wire (DI), GORM, RabbitMQ (transactional outbox), Redis, gRPC/protobuf, Swag (Swagger), Goose (migrations)

**Storage**: PostgreSQL 17.10 (schema-per-module: `identity`, `subscription`, `aiexecutor`, `shared`…), Redis (cache + rate limit), RabbitMQ (topic exchange `smart_wardrobe_exchange`)

**Testing**: `go test -race ./...` theo từng module qua `make test-race`. `testify` (assert + mock) — **riêng `services/ai/go.mod` chưa có testify, phải thêm**. Không có gomock/sqlmock/miniredis/testcontainers; mock viết tay theo hai mẫu: fake nhúng interface, hoặc `testify/mock`. Test hạ tầng thật tự `t.Skip` khi thiếu `TEST_DATABASE_URL` / `TEST_RABBITMQ_URL`.

**Target Platform**: Linux server / Docker Compose (self-hosted) và nền tảng quản lý (Cloud Run) — mọi khác biệt môi trường chỉ qua biến môi trường (§VIII)

**Project Type**: multi-service web-service — 3 dịch vụ Go độc lập (Account, Commerce Core, AI Engine) trong một Go workspace

**Performance Goals**:
- Cấp gói: 1 tài khoản p95 < 5 giây; 500 tài khoản < 60 giây (FR-076)
- Mức quyền tính phí có hiệu lực trong 60 giây kể từ lúc đổi gói (SC-014)
- Không AI thao tác nào bị từ chối trong lúc tín hiệu đang trên đường tới (SC-008)

**Constraints**:
- ~~`server.write_timeout_seconds: 60`~~ — **đã nâng lên `120` ở T001**; ràng buộc này đã được xử lý, không còn là xung đột với FR-076 (xem R9)
- `server.body_limit.default_mb: 2` — 500 phần tử ≈ 40 KB, không cần nới
- Bộ dò đường đọc không được ghi bất kỳ dữ liệu nào (FR-042)
- Một hàng đỗi không được ngăn mất tín hiệu; tín hiệu tạm không phân phối được phải giữ lại (FR-017)

**Scale/Scope**: 82 tài khoản, 16 bản ghi mức quyền tính phí, 13 dòng sổ tính phí, 2 chính sách phí, 82 tài khoản (68 chưa có bản ghi gói). Quy mô nhỏ — thiết kế ưu tiên **đơn giản và có thể kiểm chứng**, không tối ưu hiệu năng.

## Constitution Check

**GATE — pass trước Phase 0.** Đánh giá lại sau Phase 1: pass, không có vi phạm mới.

| Mục | Yêu cầu hiến pháp | Căn cứ | Kết luận |
|---|---|---|---|
| §I Ranh giới dịch vụ | Một dịch vụ không được ghi bảng của dịch vụ khác | `projectAIPolicyGrant` ghi `ai_policy_grants` bằng SQL thô; trình biên dịch **không** bắt được vì không phải lời gọi hàm | **VI phạm (đang tồn tại)** → GĐ5 gỡ; GĐ4 làm AI thành chủ sở hữu hợp lệ |
| §II Giao tiếp xuyên dịch vụ | Truy vấn đồng bộ qua gRPC; tác dụng phụ bất đồng bộ qua RabbitMQ + outbox | Tín hiệu `subscription.plan.changed` đã có; cơ chế outbox đã có | Tuân thủ — sửa nội dung tín hiệu, không đổi kiến trúc |
| §II Hợp đồng module | Chỉ import `presentation/contract/`, `application/dto`, `domain/constants` | Sẽ dùng `IUserBriefContract` cho tra cứu người dùng; sẽ thêm package constant dùng chung cho loại sự kiện | Tuân thủ |
| §III Outbox | Mọi phát hành sự kiện qua outbox, cùng giao dịch với thao tác nghiệp vụ | `IEventPublisher.Publish` ghi `outbox_records`; đã dùng đúng trong `uow.Execute` | Tuân thủ |
| §IV Phân tầng | `presentation → application → domain ← infrastructure`; không bỏ qua tầng | Bản ghi lịch sử + phát tín hiệu nằm ở tầng application | Tuân thủ |
| §V Giao dịch & lỗi | Unit of Work; lỗi qua factory `apperror` | Đóng bản ghi cũ + tạo bản ghi mới phải là **một** giao dịch (FR-031) | Tuân thủ — `GrantUseCase` phải nhận `IUnitOfWork` |
| §VI Nghị định tính phí | Dựa trên **đặt trước** (reservation-first), khoá dòng `FOR UPDATE` trên sổ tính phí; vượt hạn mức thì hạ xuống mô hình miễn phí | Suy giảm GĐ1 khi thiếu mức quyền tính phí dùng **chính** sổ tính phí `FOR UPDATE`, chỉ đổi vị trí khoá chống trùng cho dòng không có bản ghi (R8) | Tuân thủ — **không có ngoại lệ** |
| §VII Quy ước | Tuân thủ `docs/system-design/design-pattern.md` | Đã đối chiếu: tên hàm `Get...`, mapper ở `application/mapper`, hằng số thông điệp ở đầu file handler, hỗ trợ `_helper.go` | Tuân thủ |
| §VIII Không phụ thuộc nhà cung cấp | Không hardcode host; chuyển môi trường chỉ qua cấu hình | Chính sách phí mặc định lấy theo **mã** cấu hình qua biến môi trường, không hardcode | Tuân thủ |
| Migration | Goose, mỗi dịch vụ sở hữu cây migration của nó | Bảng `ai_policy_grants` có DDL trong cây migration của **Account**; dịch vụ AI **không có** cây migration | Xem R6 |
| Ngôn ngữ | Thông báo client bằng tiếng Việt; chú thích, log, định danh bằng tiếng Anh | Giữ nguyên quy ước; FR-002 yêu cầu chú thích tiếng Anh cho tệp kịch bản SQL | Tuân thủ |
| Cổng xanh | `make wire && make lint && make test-race` | Bắt buộc trước mỗi đề xuất commit | Tuân thủ |

## Project Structure

### Documentation (this feature)

```text
specs/024-admin-premium-grant/
├── plan.md              # File này
├── spec.md              # 105 FR, 30 SC, 38 dòng làm rõ
├── research.md          # Phase 0 — 14 quyết định thiết kế
├── data-model.md        # Phase 1 — thực thể, trạng thái, chuyển trạng thái
├── quickstart.md        # Phase 1 — kịch bản kiểm chứng end-to-end
├── contracts/
│   ├── subscription-plan-changed-event.md   # Hợp đồng sự kiện RabbitMQ
│   └── admin-grant-api.md                  # Hợp đồng REST cấp gói hàng loạt
├── checklists/
│   └── requirements.md  # 16/16 đạt
└── tasks.md             # Phase 2 — /speckit-tasks (đã tạo, 213 task)
```

### Source Code (repository root)

```text
shared/                                    # Mô-đun dùng chung cho cả 3 dịch vụ
├── domain/constants/
│   ├── subscriptioneventtype/             # MỚI — loại thay đổi gói, dùng chung
│   ├── aifallbackreason/                  # MỚI — mã lý do suy giảm (FR-008, FR-010)
│   └── adminaudit/action.go               # + SubscriptionGrantRestore (T110), + AICostPolicyUpdate
├── config/config.go                       # + admin_grant_rate_limit, ai.default_cost_policy_code
├── infrastructure/messaging/topology.go    # subscription_plan_changed_queue → quorum
└── presentation/middleware/
    ├── ratelimit_admin_grant.go            # MỚI — hai mức hạn ngạch theo quản trị viên
    └── ratelimit.go                       # tham chiếu mẫu LoginRateLimitMiddleware

services/account/                          # Chủ sở hữu gói, lịch sử, outbox
├── dto/subscription/billing_dto.go        # + AdminGrantBatchReq/Res
├── internal/modules/subscription/
│   ├── domain/entities/
│   │   ├── subscription_behavior.go       # dùng constant thay chuỗi viết tay
│   │   └── table_name.go
│   ├── application/
│   │   ├── mapper/                        # MỚI — entity → DTO
│   │   │   ├── plan_changed_payload.go    # MỚI — 14 trường phía phát
│   │   │   └── admin_grant_message.go  # MỚI — thông điệp đáp lời phía client (FR-085)
│   │   └── usecase/
│   │       ├── subscription/
│   │       │   ├── subscription_uc.go     # bỏ AdminGrant, giữ AdminCancel
│   │       │   ├── admin_grant_batch_uc.go  # MỚI — AdminGrantBatch, một tệp riêng
│   │       │   ├── admin_user_resolver.go   # MỚI — nhận diện mã định danh
│   │       │   ├── internal_subscription_apply_workflow.go  # + ghi ảnh chụp quyền lợi
│   │       │   ├── plan_change_publish_helper.go   # MỚI — nơi phát tín hiệu DUY NHẤT
│   │       │   └── subscription_state_helper.go    # cấp phát mặc định + lịch sử + tín hiệu
│   │       ├── purchase/wallet.go         # dùng helper, bỏ payload viết tay
│   │       └── webhook/purchase_workflow.go
│   ├── infrastructure/persistence/
│   │   ├── subscription_audit_repo_impl.go      # GĐ5: gỡ projectAIPolicyGrant
│   │   └── user_subscription_repo_impl.go       # FR-014: ProvisionDefault từng tài khoản; BulkCreate đã gỡ
│   └── presentation/
│       ├── handler/admin_subscription_handler.go   # + GrantSubscriptionsBatch, RestorePolicyGrants
│       └── router/router.go               # + /admin/subscription-grants, gỡ /:userId/grant
└── database/migrations/                   # DDL của ai_policy_grants + sổ tính phí (nợ sở hữu)

services/ai/                               # Chủ sở hữu duy nhất mức quyền tính phí
├── dto/aiexecutor/event_payload.go        # 6 → 14 trường, có eventVersion
├── internal/modules/aiexecutor/
│   ├── domain/repositories/ai_cost_repo.go # ResolvePolicy bỏ tham số planID
│   ├── infrastructure/persistence/
│   │   ├── ai_cost_repo_impl.go           # bỏ mutate; + GetPolicyByCode; Reserve nhận grant nil
│   │   └── ai_cost_policy_reader.go       # MỚI — tách phần đọc bảng chính sách (tệp hiện tại 432 dòng)
│   └── application/usecase/
│       ├── grant/grant_uc.go              # dựng grant + UoW + idempotency
│       ├── grant/grant_entitlements.go    # MỚI — đọc ảnh chụp (quyền lợi) + cấu hình sống (chống lỗ)
│       ├── grant/grant_fallback_resolver.go  # MỚI — hai tầng suy giảm
│       ├── grant/grant_payload_parser.go  # MỚI — kiểm tra nội dung tín hiệu
│       ├── grant/grant_window.go          # MỚI — quy tắc mốc bắt đầu
│       ├── orchestrator/plan.go           # nối suy giảm, bỏ trả lỗi cứng
│       ├── cost/usecase.go                # Reserve nhận grant == nil (R8)
│       ├── cost/cost_fallback_helper.go    # MỚI — tầng 2
│       └── admin/ai_admin_uc.go           # XOÁ CreateGrant (FR-035)
└── internal/di/wire_gen.go                # sinh lại

database/migrations (nằm trong services/account/database/migrations/):
└── ..._admin_premium_grant_schema.sql            # MỚI — gộp cả ba thay đổi lược đồ:
                                                  #   1. chỉ mục source_event_id (R6)
                                                  #   2. grant_id nullable + chỉ mục riêng (R8)
                                                  #   3. max_paid_attempts_per_day nullable + NULL cho Premium

database/scripts/
├── 03-dry_run_update_ai_cost_limits.sql     # MỚI — in giá trị, không ghi
├── 04-update_ai_cost_limits.sql             # MỚI — BEGIN/COMMIT, chống ghi lặp
├── 05-dry_run_cleanup_policy_grants.sql     # MỚI
├── 06-cleanup_policy_grants.sql             # MỚI
└── 07-verify_policy_grants.sql              # MỚI — chỉ đọc
```

**Structure Decision**: giữ nguyên cấu trúc multi-service hiện có. Feature này **không** tạo dịch vụ hay mô-đun mới — nó sửa mã trong `services/account` và `services/ai`, thêm một package constant dùng chung trong `shared/`, và thêm tệp kịch bản trong `database/scripts/`. Ranh giới dịch vụ là ranh giới **sở hữu dữ liệu**, không phải ranh giới thư mục mới.

**Giao diện quản trị cổng 2**: hai route HTTP mới đặt trong nhóm `/admin/ai` **sẵn có** của dịch vụ AI, kế thừa `r.auth.Handle()` và `RolesAuthorize(roleslug.Admin)` — không tạo đường xác thực riêng và không thêm hợp đồng gRPC nào. Đây là lời gọi từ **bảng điều khiển web** vào dịch vụ (client ↔ service), **không phải** trao đổi giữa hai dịch vụ backend, nên không thuộc phạm vi ràng buộc gRPC của §II. Pattern này đã có sẵn và được chứng minh hoạt động: module `brand` của Commerce dùng `/admin/brands`, và chính mô-đun này đã có 4 route `/admin/ai/*` trong `Router.Init` với Swagger sinh tự động từ chú thích `@Router` và Nginx ánh xạ sẵn tại `deployments/nginx/local.conf`. Bản ghi kiểm toán nằm ở bảng **của AI** (`aiexecutor.ai_cost_policy_audit_logs`), không dùng `shared.admin_audit_logs` vì bảng đó thuộc mô-đun `admin` của Account và ghi vào từ AI sẽ vi phạm §I. Nhật ký này phát `admin.audit.log` qua outbox **cùng giao dịch** với thao tác cập nhật (`IUnitOfWork.Execute` → `db.InjectTx`) để Account nhận vào `shared.admin_audit_logs` — tức là chủ đề thứ hai sau `subscription.plan.changed` (FR-004c).

## Phase 0 — Research

Kết quả nghiên cứu và 14 quyết định thiết kế nằm ở [research.md](./research.md). Tóm tắt những quyết định có ảnh hưởng lớn nhất tới hình dạng mã nguồn:

| # | Quyết định | Ảnh hưởng |
|---|---|---|
| R1 | Loại sự kiện dùng chung qua `shared/domain/constants/subscriptioneventtype` | Diệt gốc bug: phía AI đang xử lý chuỗi `"RENEWED"`/`"EXPIRED"` mà Account **không bao giờ phát ra** |
| R2 | Hàng đỗi thành quorum + bọc lỗi tạm bằng `RetryableError` | Không có hai thay đổi này, FR-017 không thể đạt: lỗi tạm hiện đi thẳng vào hàng chết, **không giao lại lần nào** |
| R3 | Một điểm phát tín hiệu duy nhất, dùng lại bởi cả 9 đường đổi gói | FR-011/FR-015 trở thành điều kiện kỹ thuật chứ không phải lời nhắc |
| R4 | Xoá hẳn `POST /api/v1/admin/ai/grants` | Luận cứ GĐ5: bỏ hết đường ghi chéo dịch vụ, không giữ đường ghi thứ hai chỉ vì nó tồn tại |
| R5 | Endpoint hàng loạt đặt ở `/admin/subscription-grants` | Bắt buộc: thêm `POST /admin/subscriptions/grants` sẽ làm **Gin panic** vì xung đột với `:userId` |
| R6 | Thêm chỉ mục `source_event_id` bằng Goose trong cây của Account | Không có chỉ mục thì kiểm tra trùng mỗi tín hiệu là quét toàn bảng |
| R7 | Suy giảm tầng 2 dùng lại đường bỏ qua đã có (ghi sự kiện, ledger rỗng) | Không sinh bản ghi trong đường đọc (FR-042) mà vẫn có số liệu dùng để kiểm chứng |
| R8 | Suy giảm tầng 1 vẫn dùng sổ tính phí `FOR UPDATE`; thêm chỉ mục riêng cho dòng không có bản ghi | `free-default` có `normal_route = paid`; giữ §VI nguyên vẹn và chỉ có **một** cơ chế chống vượt hạn mứm. Đã kiểm chứng trên PG 17.10 |
| R9 | Nâng `write_timeout_seconds` 60 → 120 ở cả hai tệp YAML (**đã chốt**) | Không có env ghi đè cho khoá này và `UnmarshalExact` chặn khoá lạ → sửa tay cả hai tệp |
| R10 | Dùng `user_subscriptions.version` làm chống cộng dồn kép | `version` đã tồn tại và đã được tăng; không cần cột mới |
| R11 | Chống gửi lại bằng `event_key` dựa trên khoá bất biến của yêu cầu | `event_key` đã `UNIQUE` và `GetByKey` đã có sẵn — dùng lại, không xây mới |
| R12 | Thêm mẫu email `admin_grant` | GĐ7 yêu cầu thông báo; cần 4 chỗ sửa dồn (hằng số, HTML, embed, bản đồ) |
| R13 | Ảnh chụp cho **quyền lợi**, cấu hình **sống** cho **chống lỗ tài chính** | Hai nhóm trường có nghĩa đối lập: đóng băng trần chi phí khiến việc hạ trần vô hiệu với bản ghi không thời hạn; đọc sống hạn mức sẽ âm thầm lấy đi thứ khách đã mua. Ngành dùng cùng công thức này (Chargebee grandfathering, GCP spend cap). **Không cần migration DDL**. Đã chốt |
| R14 | Nhật ký chỉnh cổng 2 nằm ở **bảng của AI**, phát `admin.audit.log` **cùng giao dịch** | `shared.admin_audit_logs` thuộc mô-đun `admin` của Account — AI ghi vào sẽ vi phạm §I. Hạ tầng phát/nhận đã có sẵn (`TopicAuditLog`, `QueueAuditLog`, `AuditLogConsumerWorker`) nên chỉ bổ sung **phía phát** |

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md) — 5 thực thể liên quan, vòng đời bản ghi mức quyền tính phí, quy tắc chuyển trạng thái
- [contracts/subscription-plan-changed-event.md](./contracts/subscription-plan-changed-event.md) — 14 trường, bảng tương thích ngược, ma trận loại sự kiện
- [contracts/admin-grant-api.md](./contracts/admin-grant-api.md) — yêu cầu/đáp lời, 3 trạng thái mỗi phần tử, giới hạn tần suất
- [quickstart.md](./quickstart.md) — kịch bản kiểm chứng từng giai đoạn GĐ0 → GĐ8

### Đánh giá lại Constitution Check sau thiết kế

Không có vi phạm mới, và **không còn ngoại lệ nào**. Suy giảm GĐ1 dùng chính sổ tính phí `FOR UPDATE` của §VI — chỉ đổi vị trí khoá chống trùng cho các dòng không có bản ghi (R8), nên vẫn chỉ có **một** cơ chế chống vượt hạn mức. Một điểm được xác nhận thêm: dịch vụ AI không có cây migration riêng, nên thay đổi DDL trên `ai_policy_grants` và `ai_usage_period_ledgers` phải đặt trong cây migration của Account — điều này **không** vi phạm hiến pháp (hiến pháp quy định mỗi dịch vụ chạy cây migration của chính nó, không quy định mỗi bảng phải nằm trong cây của chủ sở hữu), nhưng là **nợ kỹ thuật** cần ghi lại.

## Complexity Tracking

> **Trống — không có vi phạm hiến pháp nào cần biện minh.**

Bản đầu tiên của plan này ghi một ngoại lệ §VI: suy giảm GĐ1 chặn lượt trả phí bằng bộ đếm Redis thay vì sổ tính phí `FOR UPDATE`. R8 đã được thiết kế lại để **tuân thủ** §VI — sổ tính phí vẫn là điểm chống vượt hạn mức duy nhất, chỉ đổi vị trí khoá chống trùng cho các dòng không có bản ghi. Cách làm đã được kiểm chứng trên PostgreSQL 17.10. Xem [research.md](./research.md) R8.

Hệ quả là §VI giữ nguyên toàn vẹn: **một** cơ chế đặt trước, **một** nơi quyết định có cho chạy tuyến trả phí hay không, và không cơ chế thứ hai nào phải đồng bộ.

## Định mức

Áp dụng ngưỡng trong `docs/development/code-hygiene.md`: tệp sản xuất cảnh báo ở 500 dòng / lỗi ở 800; hàm cảnh báo ở 80 dòng; tối đa 2 kiểu có hành vi trong một tệp.

Các tệp sẵn có **đã vượt ngưỡng** và sẽ bị đụng tới, nên phải tách khi sửa:

| Tệp | Hiện tại | Việc |
|------|----------|------|
| `services/ai/internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go` | 432 | Đang dưới ngưỡng lỗi nhưng sẽ vượt cảnh báo → tách phần đọc bảng chính sách sang tệp riêng khi thêm `GetPolicyByCode` |
| `services/ai/internal/modules/aiexecutor/application/usecase/cost/usecase.go` | ~170 | Thêm nhánh suy giảm → tách sang `cost_fallback_helper.go` |
| `services/account/internal/modules/subscription/application/usecase/subscription/subscription_uc.go` | 354 | Thay `AdminGrant` bằng `AdminGrantBatch` + `AdminCancel` → đặt vào tệp riêng `admin_grant_batch_uc.go` |
| `services/account/internal/modules/subscription/infrastructure/persistence/subscription_audit_repo_impl.go` | 79 | `Create` rút còn 4 dòng sau khi gỡ `projectAIPolicyGrant` |

## Lệnh kiểm chứng

```bash
# Bắt buộc trước mỗi đề xuất commit
make wire-all && make lint && make test-race

# Sinh lại mã sinh tự động (Wire + Swagger + nginx) sau khi đổi DI hoặc route
make generate svc=account
make generate-check
```

`make hygiene-scan` **không** chạy tự động — chỉ khi được phép rõ ràng, vì nó ghi tệp báo cáo vào cây nguồn.
