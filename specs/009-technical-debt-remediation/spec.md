# Feature Specification: Technical Debt & Business Logic Remediation

**Feature Branch**: `009-technical-debt-remediation`

**Created**: 2026-07-26 | **Last Updated**: 2026-07-26

**Status**: Draft

**Input**: User description: "Thực hiện scan toàn bộ dự án, phát hiện và lên kế hoạch xử lý nợ kỹ thuật gồm: N+1 queries, performance bottlenecks, production readiness, business logic issues, cross-module vi phạm hiến pháp (qua presentation/contract, không dùng use case module khác, không join entity module khác), và các tính năng làm cho có không phù hợp thị trường."

## User Scenarios & Testing

### Sprint 1 — Foundation Security & Payment Operations (Priority: P1)

**Why this priority**: Payment và auth là những vấn đề có thể gây mất tiền hoặc lộ dữ liệu nếu không được xử lý. Worker lifecycle ảnh hưởng đến stability của production.

#### User Story 1.1 — Xử lý hoàn tiền tự động qua wallet (Priority: P1)

Người dùng có thể tắt auto-renew để không bị gia hạn tự động. Khi payment thành công nhưng subscription apply thất bại, hệ thống tái sử dụng bảng `provider_webhook_inbox` sẵn có (với `ProcessingAttempts`, `NextProcessingAt`, `ProcessingToken`) để retry. Worker `WebhookInboxWorker` (đã có sẵn) claim và xử lý retry. Sau max retries, hệ thống **tự động** hoàn tiền vào wallet nội bộ với wallet statement ghi rõ lý do. **Không** tạo bảng retry_queue mới.

**Independent Test**: Payment success nhưng subscription apply fail → inbox record được retry bởi WebhookInboxWorker → retry fail → system tự refund vào wallet.

**Acceptance Scenarios**:

1. **Given** payment thành công nhưng subscription apply thất bại, **When** webhook handler xử lý, **Then** ghi record vào `provider_webhook_inbox` với status PENDING, không block webhook response
2. **Given** `WebhookInboxWorker` claim inbox record, **When** retry subscription apply, **Then** mỗi lần tăng `ProcessingAttempts`, cập nhật `NextProcessingAt` với exponential backoff, dùng `ProcessingToken` để locking tránh duplicate
3. **Given** retry đạt max attempts (mặc định 5), **When** vẫn thất bại, **Then** system tự động refund vào wallet, deposit transaction status REFUNDED, wallet statement ghi rõ lý do
4. **Given** user muốn không bị gia hạn, **When** user tắt auto-renew, **Then** subscription vẫn active đến hết kỳ, không bị renew

#### User Story 1.2 — Password policy (Priority: P1)

Hệ thống áp dụng password policy (tối thiểu 8 ký tự, có chữ hoa, chữ thường, số, ký tự đặc biệt). Rate limiting đã có global middleware xử lý. **Không** có account lockout, concurrent session limit, login-specific rate limit, hay 2FA ở giai đoạn này.

**Independent Test**: Đăng ký với password yếu bị reject. Change password với password cũ sai bị reject.

**Acceptance Scenarios**:

1. **Given** user đăng ký với password "123", **When** validate, **Then** trả về lỗi "Mật khẩu phải có tối thiểu 8 ký tự, bao gồm chữ hoa, chữ thường, số và ký tự đặc biệt"
2. **Given** user change password, **When** password không đạt yêu cầu, **Then** trả về lỗi tương tự
3. **Given** user nhập sai password liên tục, **When** lần thứ 100, **Then** không có lockout — global rate limit trên IP xử lý

#### User Story 1.3 — Worker lifecycle và graceful shutdown (Priority: P1)

Tất cả background workers phải được quản lý lifecycle đúng cách: khởi động qua `Start(ctx)`, dừng qua `Stop()`, sử dụng context có thể cancel, và được đăng ký trong registry của `AppWorkers` để graceful shutdown. Không còn goroutine leaks.

**Independent Test**: Start app, gửi SIGTERM, verify tất cả workers dừng trong vòng timeout, không goroutine nào rò rỉ.

**Acceptance Scenarios**:

1. **Given** app khởi động, **When** kiểm tra worker registry, **Then** tất cả workers đều được khởi tạo qua `Start(ctx)`, không có goroutine auto-spawn trong constructor
2. **Given** app shutdown, **When** context được cancel, **Then** tất cả background loops (health check, reconnect, cleanup) dừng trong vòng 5 giây
3. **Given** FashionInitWorker subscriber, **When** delivery channel đóng, **Then** goroutine thoát, không leak

---

### Sprint 2 — Cross-Module Compliance & N+1 Fixes (Priority: P1)

**Why this priority**: 2 use case cross-module violations và 6 N+1 patterns ảnh hưởng đến maintainability và performance. Domain constants cross-module là allowed, nhưng hardcoded values cần được sửa.

#### User Story 2.1 — Kiểm tra và chuẩn hóa việc dùng domain constants (Priority: P1)

Domain constants cross-module là được phép theo hiến pháp. Tuy nhiên, cần kiểm tra module nào đang **hardcode giá trị constant** (string literal, số) thay vì import và dùng biến constant từ package sở hữu. Việc hardcode gây rủi ro typo và khó bảo trì.

**Independent Test**: Scan codebase — không còn file nào hardcode giá trị của domain constant từ module khác.

**Acceptance Scenarios**:

1. **Given** fashion module cần dùng `wardrobe/domain/constants/wardrobestatus`, **When** kiểm tra, **Then** không có file nào trong fashion module ghi `"active"` hay `0` thay vì import `wardrobestatus.Active`
2. **Given** dashboard module dùng subscription deposit status, **When** kiểm tra, **Then** tất cả đều import từ `subscription/domain/constants/depositstatus`
3. **Given** phát hiện file hardcode, **When** sửa, **Then** thay bằng import và dùng biến constant

#### User Story 2.2 — Cross-module use case calls qua presentation/contract (Priority: P1)

Các trường hợp module này dùng use case của module khác (fashion → wardrobe use case) phải được refactor: module sở hữu (wardrobe) implement contract function trong `presentation/contract/`, module cần dùng (fashion) gọi qua contract interface, không import use case trực tiếp.

**Independent Test**: Module fashion không còn import `wardrobe/application/usecase/`. Mọi cross-module data access đều qua contract.

**Acceptance Scenarios**:

1. **Given** `fashion/application/usecase/ai/recommendation/parser/normalization.go` import `wardrobe/application/usecase/wardrobe/shared`, **When** refactor, **Then** wardrobe module expose function qua `presentation/contract/`, fashion gọi qua contract
2. **Given** `fashion/application/usecase/ai/chat/helper.go` import `wardrobe/application/usecase/wardrobe/shared`, **When** refactor, **Then** tương tự qua contract

#### User Story 2.3 — Fix N+1 queries & missing indexes (Priority: P2)

Tất cả các N+1 query patterns được sửa: dùng batch UPDATE, batch transaction, thêm Preload đúng chỗ. Tất cả phải tuân thủ module boundaries — không cross-module join/preload. Module A không được preload entity của module B; phải gọi qua contract.

**Independent Test**: Chạy từng query với 1000+ records, verify số lượng database calls giảm và không có cross-module access.

**Acceptance Scenarios**:

1. **Given** `ExpireUnknown()` với 100 records, **When** sửa batch, **Then** chỉ 2 queries (SELECT + UPDATE) thay vì 300
2. **Given** `BulkUpdate()` với 100 brand members, **When** dùng 1 transaction, **Then** chỉ 1 transaction
3. **Given** `GetByUserID()` outfit, **When** sửa Preload Items, **Then** 1 query (JOIN) thay vì N+1
4. **Given** hybrid search query, **When** thay correlated subqueries bằng JOIN, **Then** 8 subqueries/row giảm xuống 1 JOIN
5. **Given** outfit listing, **When** fashion_item_id được Preload, **Then** không cross-module join (fashion module tự query riêng)

---

### Sprint 3 — Production Hardening (Priority: P2)

**Why this priority**: Performance bottlenecks và infrastructure issues sẽ gây sự cố khi có user thực tế.

> **Ghi chú**: US3.1 per-provider config đã được loại bỏ vì overengineering. Thay vào đó, Sprint 3 sẽ bao gồm 1 sub-task nhỏ: cấu hình `http.Transport` với `MaxIdleConnsPerHost=50` dùng chung cho tất cả HTTP client (sửa trong `shared/infrastructure/ai/core.go` và các chỗ tạo HTTP client).

#### User Story 3.1 — RabbitMQ reconnect & outbox pattern (Priority: P2)

Sau khi reconnect, tất cả consumers tự động re-register. Reconnect dùng exponential backoff, context-aware. **Bổ sung outbox pattern**: các event published qua RabbitMQ được ghi vào database outbox table trước, worker riêng đọc outbox và publish, đảm bảo at-least-once delivery.

**Independent Test**: Restart RabbitMQ, verify consumers tự động kết nối lại. Tắt RabbitMQ trước khi publish, verify event được lưu trong outbox và publish sau.

**Acceptance Scenarios**:

1. **Given** RabbitMQ restart, **When** reconnect, **Then** tất cả queues re-declared, consumers re-registered, messages mới được consume
2. **Given** app publish event trong khi RabbitMQ down, **When** event được ghi vào outbox, **Then** khi RMQ online, outbox worker publish event thành công
3. **Given** outbox record được publish, **When** worker xử lý, **Then** record được đánh dấu published, không publish lại

---

### Sprint 4 — Notification Module & Business Logic Maturation (Priority: P3)

**Why this priority**: Email notification đang là shared infrastructure nằm trong identity module. Cần tách thành module riêng `notification` với kiến trúc async (RMQ + outbox) trước khi thêm các notification mới. Các cải tiến UX khác cho retention.

#### User Story 4.0 — Extract notification module riêng (Priority: P3 — Foundation)

Email sender (`IEmailService`, `GmailSmtpService`, templates) hiện nằm trong `identity` module. Cần tách thành module `notification` riêng với kiến trúc async: gửi email qua RMQ + outbox, có worker riêng xử lý queue. Module khác không import trực tiếp — gọi qua `presentation/contract/`.

Module `notification` sẽ phụ trách tất cả email notification: OTP, payment, top-up, renewal, warnings. Identity module chuyển sang inject `INotificationContract`.

**Independent Test**: Tạo module notification, move SMTP + templates, test gửi email qua contract. Identity vẫn gửi được OTP email.

**Acceptance Scenarios**:

1. **Given** module notification đã được tạo, **When** identity module gửi OTP qua `INotificationContract`, **Then** email được ghi vào outbox, worker gửi SMTP thành công, worker log "email sent"
2. **Given** SMTP server tạm thời lỗi, **When** worker gửi fail, **Then** message nack → RMQ redeliver → vào DLQ sau max retries, không mất email (at-least-once delivery)
3. **Given** module notification có queue email, **When** các module khác gọi contract, **Then** non-blocking, không ảnh hưởng response time

#### User Story 4.1 — Auto-renew notification & wallet balance warning (Priority: P3)

Khi subscription sắp hết hạn và auto-renew đang bật, nếu wallet không đủ tiền, hệ thống gửi notification cảnh báo cho user. User có thể nạp tiền vào wallet trước khi hết hạn. **Không** có grace period, proration, hay trial period.

**Independent Test**: Tạo subscription sắp hết hạn với auto-renew, wallet thiếu tiền → verify user nhận được notification.

**Acceptance Scenarios**:

1. **Given** subscription sắp hết hạn trong 3 ngày, auto-renew bật, wallet không đủ, **When** worker kiểm tra, **Then** gửi email cảnh báo qua `INotificationContract`
2. **Given** user nhận notification, **When** user nạp đủ tiền vào wallet trước khi hết hạn, **Then** renewal thành công
3. **Given** user không nạp tiền, **When** subscription hết hạn, **Then** subscription tự động downgrade về free plan

#### User Story 4.2 — Email notification cho payment & top-up (Priority: P3)

Khi user nạp tiền vào wallet thành công hoặc thanh toán gói subscription thành công, hệ thống tự động gửi email xác nhận qua module notification (async RMQ). Email gồm thông tin: số tiền, gói dịch vụ, ngày hết hạn, số dư wallet mới.

**Independent Test**: Top-up thành công → user nhận email xác nhận nạp tiền. Payment subscription thành công → user nhận email xác nhận thanh toán.

**Acceptance Scenarios**:

1. **Given** user nạp 200,000đ vào wallet thành công, **When** transaction hoàn tất, **Then** gửi email "Nạp tiền thành công — Số dư mới: 200,000đ" qua module notification
2. **Given** user thanh toán gói Premium 100,000đ thành công, **When** subscription activated, **Then** gửi email "Thanh toán thành công — Gói Premium active đến 30/08/2026"
3. **Given** subscription renewal thành công, **When** wallet bị trừ tiền, **Then** gửi email "Gia hạn thành công — Gói Premium"
4. **Given** subscription renewal thất bại vì wallet không đủ, **When** worker xử lý, **Then** gửi email "Gia hạn thất bại — Vui lòng nạp thêm tiền"

#### User Story 4.3 — [MOVED TO docs/temp/technical-dept/] AI pipeline hardening

Chuyển toàn bộ user story về AI pipeline hardening (local NLP parser, magic string token `[ACTION:REDIRECT_OUTFIT]`, local route dead end) vào `docs/temp/technical-dept/` để tham khảo sau. Không triển khai trong spec này.

---

### Edge Cases

- **Payment**: Refund khi wallet balance đã dùng; overlapping giữa refund và renewal schedule
- **Auth**: Rate limit ảnh hưởng đến API health check; user bị rate limit khi forgot password
- **Worker**: Worker crash giữa chừng; outbox ghi duplicate; RabbitMQ network partition
- **Auto-renew**: User toggle auto-renew OFF ngay trước khi renewal worker đọc list — worker dùng `WHERE auto_renew = true AND id = ?` trong UPDATE transaction để tránh race, không renew nếu user vừa tắt
- **N+1**: Batch processing timeout với 10k+ records; deadlock khi UPDATE với FOR UPDATE
- **Cross-module contract**: Circular dependency nếu contract implement sai; backward compatibility khi thay đổi contract interface


## Requirements

### Functional Requirements

**Sprint 1 — Foundation Security & Payment Operations:**

- **FR-001**: Hệ thống tái sử dụng `provider_webhook_inbox` cho retry subscription apply — không tạo bảng retry_queue mới
- **FR-002**: Hết max retries → tự động refund vào wallet, deposit transaction status REFUNDED, wallet statement ghi rõ lý do
- **FR-003**: User có thể bật/tắt auto-renew subscription
- **FR-004**: DepositTransaction entity thêm trường `refunded_at`, `refund_amount`, `refund_reason`
- **FR-005**: Password policy enforced: tối thiểu 8 ký tự, ≥1 uppercase, ≥1 lowercase, ≥1 digit, ≥1 special character
- **FR-006**: Tất cả workers có `Start(ctx context.Context)` và `Stop()` methods, registry trong `AppWorkers`
- **FR-007**: Tất cả background goroutines (health check, cleanup, reconnect) có stop channel/context

**Sprint 2 — Cross-Module Compliance & N+1:**

- **FR-008**: Scan toàn bộ codebase tìm hardcoded domain constant values, thay bằng import và biến constant
- **FR-009**: Refactor 2 cross-module use case imports: wardrobe module implement contract function, fashion module gọi qua contract
- **FR-010**: `ai_cost_repo_impl.go` - batch UPDATE thay vì loop per transaction, không join entity ngoài module
- **FR-011**: `brand_member_repo.go` - gom các Save vào 1 transaction
- **FR-012**: `outfit_repo.go` - GetByUserID, GetByUserIDPaginated dùng Preload đúng, không cross-module join (outfit item chỉ preload trong wardrobe module)
- **FR-013**: `fashion_item_repo_hybrid*.go` - correlated subqueries thay bằng JOIN trong cùng module
- **FR-014**: Thêm composite indexes: `(user_id, is_deleted, status)`, `(fashion_item_id)`, `(item_type, is_deleted)`

**Sprint 3 — Production Hardening:**

- **FR-015**: Cấu hình `http.Transport` với `MaxIdleConnsPerHost=50` dùng chung cho tất cả HTTP client (ai, payos), giải quyết connection starvation
- **FR-016**: RabbitMQ reconnect: auto re-register consumers, exponential backoff (1s, 2s, 4s, ..., max 30s), context-aware stop
- **FR-017**: Database outbox table + outbox worker: event → outbox → publish to RMQ → mark published. **Bắt buộc cho mọi event publish** — không có direct RMQ publish path. `RabbitMQClient.Publish()` tự động ghi outbox; module có transaction sẵn ghi outbox trực tiếp. Cả 2 luồng đều qua outbox worker

**Sprint 4 — Notification Module & Business Logic Maturation:**

- **FR-022**: Tách email sender khỏi identity module thành `notification` module riêng — move `GmailSmtpService` (→ `SmtpSender`), templates, `IEmailService` interface. Delivery tracking qua worker log + RMQ DLQ, không tạo bảng `email_messages`
- **FR-023**: Module `notification` có kiến trúc async: outbox → RMQ queue → worker xử lý → gửi SMTP. Không block caller
- **FR-024**: Module `notification` expose `INotificationContract` qua `presentation/contract/` — interface generic `Send(channel ChannelEnum, template TemplateEnum, params map[string]any, to string)` để các module khác gửi notification mà không import trực tiếp. ChannelEnum = {Email}, template = {Otp, PaymentSuccess, TopupSuccess, RenewalSuccess, RenewalFailed, WalletWarning}
- **FR-025**: Identity module chuyển từ dùng `IEmailService` sang gọi `INotificationContract`
- **FR-026**: Gửi email xác nhận khi top-up thành công: số tiền, số dư mới
- **FR-027**: Gửi email xác nhận khi payment subscription thành công: gói, ngày hết hạn
- **FR-028**: Gửi email xác nhận khi renewal thành công/thất bại
- **FR-029**: Hệ thống kiểm tra wallet balance N ngày trước khi subscription expires, gửi notification nếu không đủ tiền


### Key Entities

- **DepositTransaction** (subscription): thêm refund tracking fields
- **OutboxRecord** (shared/messaging): mới — event outbox cho RabbitMQ
- **SubscriptionRenewalAttempt** (subscription): tracking renewal attempts
- **EmailTemplate** (notification): code constants trong Go files — không phải DB entity. Mỗi loại email (OTP, payment, top-up, renewal, warning) là 1 HTML template string. Delivery tracking qua worker log + RMQ DLQ, không có DB table cho email delivery
- **NotificationContract** (notification): mới — generic contract interface `Send(channel ChannelEnum, template TemplateEnum, params map[string]any, to string)` — extensible cho push/SMS sau này

## Success Criteria

### Measurable Outcomes

- **SC-001**: Payment success + subscription apply fail → system tự retry tối đa 5 lần, sau cùng tự refund vào wallet với statement rõ ràng
- **SC-002**: Password policy block 100% password yếu khi register, change-password, reset-password — login không check password policy
- **SC-003**: Worker shutdown < 5 giây khi nhận SIGTERM, 0 goroutine leak
- **SC-004**: 2 cross-module use case violations giảm xuống 0
- **SC-005**: ExpireUnknown(100) giảm từ 300 queries xuống 2
- **SC-006**: HTTP transport config `MaxIdleConnsPerHost=50`, 100 concurrent requests không bị `http: connection reset` hoặc timeout > 5s
- **SC-007**: RabbitMQ reconnect + outbox đảm bảo at-least-once delivery — không mất event khi RMQ restart (có thể publish duplicate, không loss)
- **SC-008**: Email sender được tách khỏi identity module, module notification hoạt động độc lập qua RMQ + outbox
- **SC-009**: User nhận email OTP sau khi module identity chuyển sang dùng contract (regression test)
- **SC-010**: User nhận email xác nhận khi top-up thành công
- **SC-011**: User nhận email xác nhận khi payment subscription thành công
- **SC-012**: User nhận email khi subscription sắp hết hạn và wallet không đủ

## Clarifications

### Session 2026-07-26

- Q: EmailTemplate lưu dưới dạng DB table hay code constants? → A: Code constants (string trong Go files). Deploy là thay đổi template.
- Q: Module notification chỉ support email hay cần extensible cho push/SMS sau này? → A: Generic interface `INotificationContract.Send(channel, template, params, to)` với channel enum, extensible cho push/SMS.
- Q: Admin có cần UI xem email history? → A: Chỉ worker log. Không cần admin API/UI cho email history ở MVP.
- Q: Observability cho refund, outbox, email — có cần metrics infrastructure (Prometheus) mới? → A: Không. Dùng logging hiện tại (`workerlog` + `zap`). Worker log đã có sẵn counter tracking (TotalCount, SuccessCount, ErrorCount). Chỉ cần log business events với structured fields là đủ cho MVP. Không thêm dependency metrics nào.

## Assumptions

- Refund tự động chỉ xảy ra khi payment thành công nhưng subscription apply thất bại sau retry, không refund cho user đổi ý
- Password policy không áp dụng cho user đã tồn tại (chỉ user mới hoặc change password)
- Global rate limit middleware đã đủ cho login protection
- Outbox pattern dùng PostgreSQL table + polling worker, không cần thêm infrastructure
- Email history chỉ cần worker log — không cần admin API hay UI
- EmailTemplate là code constants — không cần DB table cho template
- Template rendering dùng `html/template` (Go stdlib) — params được escape HTML-safe trước khi render. Các hàm cũ dùng `strings.ReplaceAll` sẽ được migrate sang `html/template`
- Thời gian mỗi sprint: 1-2 tuần
- SC-006 (HTTP perf) và SC-007 (outbox at-least-once) được verify qua implementation correctness, không có task load test riêng

## Sprints & Phases

### Sprint 1: Foundation Security & Payment Operations (P1)

| Phase | Scope | Files affected |
|-------|-------|---------------|
| 1.1 | Refund nội bộ (tái sử dụng inbox) + auto-renew | `subscription/domain/entities/payment.go`, `subscription/domain/constants/depositstatus.go`, `subscription/application/usecase/webhook/`, `subscription/infrastructure/persistence/webhook_inbox_repo_impl.go`, `subscription/presentation/handler/billing_handler.go` |
| 1.2 | Password policy | `identity/application/usecase/register_uc.go`, `identity/presentation/handler/auth_handler.go` |
| 1.3 | Worker Lifecycle | `bootstrap/app.go`, `modules/*/presentation/worker/*`, `shared/infrastructure/` background goroutines |

### Sprint 2: Cross-Module Compliance & N+1 Fixes (P1)

| Phase | Scope | Files affected |
|-------|-------|---------------|
| 2.1 | Hardcoded constants scan + fix | Toàn bộ modules — scan và thay thế hardcode value bằng import constant |
| 2.2 | Cross-module use case → contract | `fashion/application/usecase/ai/`, `wardrobe/presentation/contract/` |
| 2.3 | N+1 queries + indexes | `ai_cost_repo_impl.go`, `brand_member_repo.go`, `outfit_repo.go`, `fashion_item_repo_hybrid*.go`, migration files |

### Sprint 3: Production Hardening (P2)

| Phase | Scope | Files affected |
|-------|-------|---------------|
| 3.1 | HTTP transport fix + RMQ reconnect + outbox | `shared/infrastructure/ai/core.go`, `shared/infrastructure/messaging/rabbitmq*.go`, `shared/infrastructure/messaging/outbox.go` (mới), `configs/config.yaml`, migration cho bảng outbox |
| 3.2 | Caller restructuring — UoW compliance | `item_uc_write.go`, `catalog_uc.go`, `worker_uc.go`, `init_uc.go`, `subscription_purchase_uc.go` |

### Sprint 4: Notification Module & Business Logic Maturation (P3)

| Phase | Scope | Files affected |
|-------|-------|---------------|
| 4.0 | Foundation — Extract notification module | `internal/modules/notification/` (mới — contract, communication, worker), `internal/modules/identity/infrastructure/communication/` (move → smtp_sender.go), `internal/modules/identity/infrastructure/communication/templates/` (move), DI wiring |
| 4.1 | Foundational — RMQ + outbox integration | `internal/modules/notification/presentation/worker/email_worker.go`, tái sử dụng outbox và RMQ từ Sprint 3. **Không** tạo email_repo, email entity, email table |
| 4.2 | Auto-renew notification | `subscription/application/usecase/subscription/subscription_renewal_workflow.go`, subscription worker |
| 4.4 | Payment & top-up email | `subscription/application/usecase/wallet/wallet_uc.go`, `subscription/application/usecase/webhook/payment_webhook_uc.go`, `subscription/application/usecase/subscription/subscription_renewal_workflow.go` |

> **Note**: Mapping spec → tasks: US4.0 (Foundation: extract module) → tasks-sprint4.md US0; US4.0 (Foundation: RMQ+outbox integration) → tasks-sprint4.md US1; US4.1 (Auto-renew notif) → US2; US4.2 (Payment & top-up email) → US3. US4.3 (AI pipeline) đã move, không có task.
