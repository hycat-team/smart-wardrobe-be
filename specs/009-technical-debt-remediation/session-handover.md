# Session Handover

**Project**: SmartWardrobe Backend — Technical Debt & Business Logic Remediation

**Spec branch**: `feat/009-technical-debt-remediation`

**Spec directory**: `specs/009-technical-debt-remediation/`

**Date**: 2026-07-26

---

## Architecture Decisions

### 1. Refund Mechanism — Reuse Existing `provider_webhook_inbox` Table

| Field | Detail |
|-------|--------|
| **Decision** | Không tạo bảng `retry_queue` mới. Tái sử dụng `provider_webhook_inbox` + `WebhookInboxWorker` có sẵn |
| **Why** | Bảng inbox đã có `ProcessingAttempts`, `NextProcessingAt`, `ProcessingToken` (locking), `ProcessingStatus`. Chỉ cần mở rộng logic use case, không cần migration mới cho retry |
| **Trade-off** | Worker `WebhookInboxWorker` hiện chạy cron 30s/lần — có latency 30s giữa retry. Chấp nhận được cho MVP |
| **Đừng** | Đừng tạo bảng/worker mới cho retry |

### 2. HTTP Client Transport — Shared Transport, Không Per-Provider Config

| Field | Detail |
|-------|--------|
| **Decision** | Một shared `http.Transport` với `MaxIdleConnsPerHost=50`. Không config per-provider |
| **Why** | Vấn đề thực sự là default transport chỉ 2 idle conn/host. Per-provider rate limit là overengineering cho MVP |
| **Trade-off** | AI provider Google/OpenAI không được config rate limit riêng — phải dùng API key quota từ provider |
| **Đừng** | Đừng tách config riêng cho từng HTTP provider ở giai đoạn này |

### 3. Outbox Pattern — Bắt Buộc Cho Mọi RMQ Publish

| Field | Detail |
|-------|--------|
| **Decision** | 1 shared `outbox_records` table cho tất cả module. `topic` field quyết định RMQ routing key. Polling worker 1s/lần. **Tất cả event publish bắt buộc qua outbox** — không có direct RMQ publish |
| **Why** | Đảm bảo at-least-once delivery. `RabbitMQClient.Publish()` được refactor để inject `OutboxRepository` và ghi outbox record qua `db.GetTx(ctx)`. Outbox worker publish lên RMQ async |
| **Trade-off** | 1s polling latency giữa outbox write và RMQ publish. Transaction atomicity yêu cầu caller phải publish bên trong UoW (xem Caller Restructuring bên dưới) |
| **Đừng** | Đừng tạo riêng outbox table per module ở MVP. Đừng publish trực tiếp qua `ch.PublishWithContext`. Đừng gọi `Publish(ctx)` ngoài UoW — phải dùng `Publish(txCtx)` |

### 4. Notification Email — Module Riêng, Không Shared Package

| Field | Detail |
|-------|--------|
| **Decision** | `internal/modules/notification/` — module riêng với kiến trúc async (outbox + RMQ + worker). **Không** đặt trong `shared/`. **Không** tạo `email_messages` table |
| **Why** | Email tracking qua worker log + RMQ DLQ là đủ cho MVP. `EmailMessage` entity + repository + retry logic là overengineering — RMQ DLQ handle retry tốt hơn. SMTP deferred bounce không thể bắt bằng Go `net/smtp` thuần, nên `email_messages` table không giải quyết được vấn đề thực sự |
| **Trade-off** | Không có audit trail email trong DB — chỉ worker log. Nếu cần audit sau này, thêm sau MVP |
| **Đừng** | Đừng tạo bảng `email_messages`. Đừng implement custom email retry logic (attempts, last_error) — dùng RMQ ack/nack/DLQ |

### 5. Notification Contract Interface — Generic, Extensible

| Field | Detail |
|-------|--------|
| **Decision** | `INotificationContract.Send(ctx, channel ChannelEnum, template TemplateEnum, params map[string]any, to string) error`. Không specific methods per email type |
| **Why** | Extensible cho push/SMS sau này mà không break contract. Thêm channel mới = thêm enum value |
| **Trade-off** | Hơi abstract hơn specific methods. `params map[string]any` mất type safety |
| **Đừng** | Đừng tạo interface kiểu `SendOtpEmail`, `SendPaymentSuccessEmail` riêng lẻ |

### 6. EmailTemplate — Code Constants + html/template Render

| Field | Detail |
|-------|--------|
| **Decision** | HTML string constants trong Go files. Dùng `html/template` (Go stdlib) để render — params được escape HTML-safe. Deploy là thay đổi template |
| **Why** | `strings.ReplaceAll` tiềm ẩn XSS risk nếu params chứa user input. `html/template` auto-escape. Các template cũ (`RegistrationOtp.html`, `ForgotPasswordOtp.html`) đã migrate từ `{{OTP_CODE}}` syntax sang `{{.OTPCode}}` |
| **Trade-off** | Template thay đổi phải deploy lại code. Nhưng template thay đổi rất ít |
| **Đừng** | Đừng dùng `strings.ReplaceAll` cho template rendering. Đừng tạo DB table cho template. Đừng đặt template string trong use case — để trong file constants riêng |

### 7. Email History — Worker Log Only, Không Admin UI

| Field | Detail |
|-------|--------|
| **Decision** | Không admin API/UI cho email history. Chỉ worker log |
| **Why** | Giảm scope cho MVP. Worker log đủ debug |
| **Trade-off** | Không thể tra cứu email history từ UI |

### 8. Elasticsearch — Loại Bỏ Khỏi Hệ Thống

| Field | Detail |
|-------|--------|
| **Decision** | ES đang được lên kế hoạch loại bỏ. Không có US nào cho ES trong spec này |
| **Đừng** | Đừng tạo US cho ES hardening. ES sẽ được xử lý riêng ngoài spec này |

### 9. RMQ Constants — Tập Trung Trong Shared Files

| Field | Detail |
|-------|--------|
| **Decision** | Queue name, routing key constants tập trung tại `shared/infrastructure/messaging/rabbitmq_constants.go`. Topic strings (event names) tập trung tại `shared/application/constants/eventconstants/eventconstants.go`. Không tạo file constants riêng trong từng module |
| **Why** | Dễ maintain, tránh duplicate, dễ dò tìm khi debug flow event |
| **Đừng** | Đừng hardcode topic/routing key trong use case hay worker. Đừng tạo file constants riêng trong module (trừ khi là domain constants) |

### 10. Queue Binding — Worker Tự Khai Báo, Không Trong `connect()`

| Field | Detail |
|-------|--------|
| **Decision** | Worker tự gọi `DeclareAndBindQueue()` trong `Start()` trước khi Consume. Không khai báo cứng trong `rabbitmq_connection.go`. `DeclareAndBindQueue` là idempotent |
| **Why** | Worker biết queue của nó cần binding gì. Tránh phải sửa file infra mỗi khi thêm worker mới |
| **Đừng** | Đừng thêm queue binding vào `rabbitmq_connection.go` cho worker mới |

### 11. Outbox Write — Direct Table Write, Không Qua `Publish()` Khi Có Transaction

| Field | Detail |
|-------|--------|
| **Decision** | Module có sẵn DB transaction (S4 notification contract impl) ghi outbox record **trực tiếp** qua `outboxRepo.Create(ctx, record)` với `db.GetTx(ctx)`. Module publish độc lập gọi `RabbitMQClient.Publish()` (sau Sprint 3, `Publish()` tự ghi outbox). Không module nào publish trực tiếp lên RMQ |
| **Why** | Outbox record phải atomic với business transaction → phải dùng `db.GetTx(ctx)` từ caller. `RabbitMQClient.Publish()` không share được transaction của caller |
| **Đừng** | Đừng gọi `RabbitMQClient.Publish()` bên trong UoW callback — ghi outbox trực tiếp. Đừng publish ra RMQ mà không qua outbox |

### 12. Observability — Logging-Only, Không Metrics Infrastructure

| Field | Detail |
|-------|--------|
| **Decision** | Dùng `workerlog` + `zap` hiện tại. Không thêm Prometheus, expvar, OpenTelemetry. Worker log đã có sẵn counter tracking (`TotalCount`, `SuccessCount`, `ErrorCount`) |
| **Why** | Codebase chưa có metrics infrastructure — thêm Prometheus tốn dependency, endpoint, middleware cho chỉ 3-5 counters. Overengineering cho MVP |
| **Đừng** | Đừng thêm Prometheus/expvar trong sprint này |

### 13. Password Policy — Chỉ 3 Endpoints

| Field | Detail |
|-------|--------|
| **Decision** | Password policy enforced cho 3 endpoints: `Register`, `ChangePassword`, `ResetPassword`. **Login không check** password policy (chỉ check đúng/sai) |
| **Why** | Endpoints còn lại (ForgotPassword, ConfirmForgotPasswordOtp, Login) không tạo/thay đổi password |
| **Đừng** | Đừng thêm password policy vào Login handler |

---

## Design Constraints

### Task File Structure
- **Mỗi sprint 1 file task riêng**: `tasks-sprint1.md` → `tasks-sprint4.md`
- **Hoàn thành tất cả task files trước khi bắt đầu implement**
- **Implement tuần tự theo sprint**: Sprint 1 → 2 → 3 → 4 (Sprint 3 là prerequisite cho Sprint 4)

### Infrastructure Layer — Không Import Presentation
- Infrastructure (ES, RMQ, HTTP clients, outbox) KHÔNG được import `BaseWorker` từ `shared/presentation/worker/`
- Infrastructure chỉ dùng Go primitives: `chan struct{}`, `context.Context` để stop/cleanup
- `BaseWorker` chỉ dùng cho business workers ở presentation layer

### Outbox — Bắt Buộc Cho Mọi RMQ Publish
- ALL event publishing phải qua transactional outbox. Không fallback
- `RabbitMQClient.Publish()` luôn ghi OutboxRecord (qua OutboxRepository inject)
- Module có transaction sẵn ghi outbox trực tiếp, không qua Publish()
- Outbox worker đọc từ outbox table và publish lên RMQ

### Caller Restructuring — UoW + txCtx
- Mọi `eventPublisher.Publish(ctx, topic, payload)` phải được gọi bên trong `uow.Execute(func(txCtx) error { ... })` với `txCtx`
- 12 call sites đã được xác định cần restructure (Sprint 3 T015-T017)
- Mục tiêu: outbox record atomic với business data

### Queue Binding
- Worker tự gọi `DeclareAndBindQueue(queueName, routingKey)` trong `Start()` — không khai báo cứng trong `connect()`
- `DeclareAndBindQueue` idempotent — an toàn gọi lại khi restart

### Transaction Boundary
- `RabbitMQClient.Publish()` khi nhận `ctx` có transaction (từ `db.GetTx(ctx)`), ghi outbox trong transaction đó
- Khi không có transaction, `Publish()` tự tạo transaction riêng
- Caller restructuring (T015-T017) đảm bảo mọi publish đều có txCtx

### Không Mở Rộng Không Cần Thiết
- `SubscriptionRenewalAttempt`: entity ĐÃ tồn tại (repo, interface, migration, DI) — không cần task mới
- EmailMessage: KHÔNG tạo entity, repository, table
- Metrics: KHÔNG thêm Prometheus/expvar
- Per-provider HTTP config: Đã loại bỏ

### Goose Migrations
- Database changes qua Goose SQL migrations files mới. Không sửa migration cũ. Không dùng GORM AutoMigrate
- 5 migrations tổng cộng: S1:T001 (refund fields), S1:T002 (inbox status), S2:T001 (indexes), S3:T001 (outbox table)

---

## Domain Conventions

### Cross-Module Domain Constants
- Domain constants của module khác **được phép import trực tiếp**
- **Không** được hardcode giá trị — luôn import và dùng biến constant

### RMQ Constants Convention
- Queue names, routing keys → `shared/infrastructure/messaging/rabbitmq_constants.go`
- Event topic strings → `shared/application/constants/eventconstants/eventconstants.go`
- Worker queue declaration → worker tự gọi `DeclareAndBindQueue()` trong `Start()`

### Template Rendering Convention
- Dùng `html/template` (Go stdlib) với struct data — không dùng `strings.ReplaceAll`
- Template syntax: `{{.FieldName}}` (camelCase)
- Template là code constants, không phải DB entity
- Template string để trong file constants riêng, không trong use case

### Password Policy Convention
- Áp dụng cho: Register, ChangePassword, ResetPassword
- Không apply cho user cũ (chỉ user mới hoặc change password)
- Login không check password policy

### Auto-renew Race Condition Convention
- Worker chỉ update subscription `WHERE auto_renew = true AND id = ?` trong UPDATE transaction
- Nếu user vừa tắt auto-renew, row không match → worker không renew

### SC Verification Convention
- SC-006 (HTTP perf) và SC-007 (outbox at-least-once) verify qua implementation correctness — không có task load test riêng

---

## Important Reasoning

### Tại sao `email_messages` table là overengineering
- Bảng này chỉ để track delivery status (sent/failed/retry)
- Nhưng SMTP có deferred bounce: server báo "250 OK" rồi gửi bounce email sau 1h — Go `net/smtp` không bắt được
- Vậy `email_messages` table chỉ track được immediate error (server down, auth fail)
- Immediate error đã được handle bởi RMQ DLQ (nack → redeliver → DLQ)
- Worker log + zap đủ để debug delivery issue
- Nếu sau này cần audit trail email thật, thêm table khi có requirement cụ thể

### Tại sao không dùng Retry Queue Mới
- `provider_webhook_inbox` đã có đủ: `ProcessingAttempts`, `NextProcessingAt`, `ProcessingToken` (SKIP LOCKED)
- Thiếu: cap on retries + exponential backoff + terminal auto-refund — đây là mở rộng logic use case

### Tại sao Notification Module Riêng (Không Shared)
- Email có retry, failed queue, template rendering — business module, không phải shared utility
- Đặt trong `shared/` → module khác import trực tiếp → vi phạm module boundaries

### Tại sao Phải Restructure Caller (Direction A)
- Constitution Principle III yêu cầu outbox record trong cùng DB transaction với business operation
- 11/14 publish call hiện tại ở ngoài UoW — nếu chỉ refactor `Publish()` mà không restructure caller, outbox record không atomic
- Restructure caller là công việc nhiều hơn (~5 task mới) nhưng đúng constitution

### Tại sao Contract Impl Ghi Outbox Trực Tiếp (Không Qua Publish)
- Contract `Send()` được gọi từ trong business transaction (top-up, payment, renewal)
- Outbox record phải atomic với transaction đó
- `RabbitMQClient.Publish()` không share được transaction
- Solution: contract impl gọi `outboxRepo.Create(ctx, record)` với `db.GetTx(ctx)` từ caller

---

## Hidden Context

1. **PayOS "Chi hộ" (payout API)**: PayOS có API `POST /v1/payouts` nhưng đang ở giai đoạn quảng bá mới ra mắt. Dev quyết định **không dùng chi hộ**. Refund hoàn toàn qua wallet nội bộ.

2. **Task numbering jumps**: Sprint 4 FR bắt đầu từ FR-022 (không phải FR-018). FR-018 và FR-019 không tồn tại (bỏ qua do renumbering). FR-020/FR-021 đã được renumber thành FR-029/FR-030, sau đó FR-030 bị xóa vì duplicate với FR-003.

3. **Config struct**: `MaxRetryAttempts` (S1 T007) và `wallet_balance_warning_days` (S4 T013) là config mới cần thêm vào `internal/config/config.go` ở subscription section.

4. **Database migration files**: Cần tạo file `.sql` mới dưới `database/migrations/` cho mỗi migration task (T001 của mỗi sprint). Không sửa file migration cũ.

5. **DI Wiring**: Sprint 3: `RabbitMQClient` nhận thêm `OutboxRepository` trong constructor. Sprint 4: notification module cần Wire ProviderSet mới, identity module chuyển từ `IEmailService` sang `INotificationContract`.

6. **GmailSmtpService đã migrate**: Code dùng `strings.ReplaceAll` đã được chuyển sang `html/template`. Template syntax từ `{{OTP_CODE}}` → `{{.OTPCode}}`. Nếu tạo template mới (payment, top-up, renewal), dùng Go template syntax.

7. **Phase 4.3 → 4.4**: Sprint 4 Sprints table dùng Phase 4.4 thay vì 4.3 vì US4.3 (AI pipeline) đã move. Mapping note ở cuối Sprint 4 table giải thích spec US → tasks US.

8. **Outbox topic routing**: S3 outbox worker publish lên RMQ dựa trên `topic` field. Exchange `smart_wardrobe_exchange` (topic type). Worker tự declare queue + binding qua `DeclareAndBindQueue()`.

9. **SubscriptionRenewalAttempt entity đã tồn tại**: Entity, repository, interface, migration, DI wiring đều có sẵn. Không cần task tạo mới.

10. **Template language**: Email templates hiện chỉ support tiếng Việt. Multi-language support chưa có quyết định.

---

## TODO

### Sprint 1 — Foundation Security & Payment Operations (25 tasks)
**File**: `tasks-sprint1.md` — US1 (refund), US2 (password), US3 (worker lifecycle)

1. **Phase 1** (T001-T002): DB migrations (refund fields + inbox status)
2. **Phase 2** (T003-T012): Auto-refund + auto-renew toggle → **ưu tiên cao nhất**
3. **Phase 3** (T013-T016): Password policy
4. **Phase 4** (T017-T025): Worker lifecycle + BaseWorker

**⚠️ Note**: Khi implement T011 (auto-renew API), nhớ rằng FR-030 đã bị xóa — auto-renew chỉ implement 1 lần ở Sprint 1.

### Sprint 2 — Cross-Module Compliance & N+1 (20 tasks)
**File**: `tasks-sprint2.md` — US1 (constants), US2 (contract), US3 (N+1)

1. **Phase 1** (T001): DB migration (indexes)
2. **Phase 2** (T002-T007): Scan hardcoded constants
3. **Phase 3** (T008-T013): Cross-module use case → contract
4. **Phase 4** (T014-T020): N+1 queries + indexes

### Sprint 3 — Production Hardening (19 tasks)
**File**: `tasks-sprint3.md` — US1 (HTTP + RMQ + outbox + caller restructuring)

1. **Phase 1** (T001): DB migration (outbox table)
2. **Phase 2** (T002-T014): HTTP transport fix, RMQ reconnect, outbox infrastructure (entity, repo, worker, Publish refactor)
3. **Phase 3** (T015-T019): Caller restructuring — wrap 12 publish call sites trong UoW. **Rủi ro regression cao nhất** — chạy `go test ./...` + `make test` sau mỗi task.

**⚠️ Note**: T011: inject `OutboxRepository` vào `RabbitMQClient`. `Publish()` dùng `db.GetTx(ctx)` để ghi outbox. T014 verify Publish() đã clean. T015-T017 restructure từng module. T019 verify cuối.

### Sprint 4 — Notification Module & Business Logic Maturation (18 tasks)
**File**: `tasks-sprint4.md` — US0 (module foundation), US1 (RMQ+outbox), US2 (auto-renew notif), US3 (payment/top-up email)

1. **Phase 1** (T001-T008): Foundation — tạo module notification, contract, move SMTP + templates, DI
2. **Phase 2** (T009-T012): RMQ + outbox integration — constants vào shared files, worker + queue binding
3. **Phase 3** (T013-T015): Auto-renew notification & wallet warning
4. **Phase 4** (T016-T018): Payment & top-up email notification

**⚠️ Note**: Phụ thuộc outbox infrastructure từ Sprint 3. `contract.Send(txCtx, ...)` gọi bên trong UoW callback. Notification worker tự `DeclareAndBindQueue()` trong `Start()`.

---

## Risks

1. **Sprint 3 là prerequisite cho Sprint 4**: Outbox infrastructure + caller restructuring phải hoàn thành trước Sprint 4. Nếu làm Sprint 4 trước, notification module phải gọi SMTP sync (tạm thời).

2. **Caller restructuring (T015-T019) rủi ro regression cao nhất**: 12 call sites được restructure để dùng UoW + txCtx. Ảnh hưởng đến wardrobe CRUD, fashion worker, subscription purchase. Cần chạy `go test ./...` + `make test` sau mỗi task. Nếu không có test coverage cho các use case này, cần manual test.

3. **WebhookInboxWorker cron 30s/lần**: Retry latency 30s giữa các lần retry. Nếu không chấp nhận được, cần giảm interval nhưng tăng DB load. Chưa có quyết định.

4. **Template rendering migration**: Code cũ dùng `strings.ReplaceAll` đã migrate sang `html/template`. Nếu có chỗ nào còn dùng pattern cũ, cần migrate.

5. **Notification module phụ thuộc identity module hiện tại**: Move `GmailSmtpService` + templates từ identity sang notification có thể gây regression. Cần test OTP flow sau khi chuyển.

6. **Elasticsearch deprecation không có migration plan**: ES đang được loại bỏ nhưng chưa có strategy cho data hiện tại. Có thể gây mất dữ liệu search nếu không có plan.

---

## Open Questions

1. **Worker intervals**: WebhookInboxWorker cron 30s/lần, outbox worker poll 1s/lần. Có thể cần tuning dựa trên production load, nhưng chưa có config. Để mặc định cho MVP.

2. **Template localization**: Hiện tại email templates chỉ support tiếng Việt. Chưa có quyết định về multi-language support.

3. **Audit trail**: Chưa có audit trail cho refund/payment/email. Dev chọn không implement ở MVP. Có thể thêm sau khi có nhu cầu thực tế.
