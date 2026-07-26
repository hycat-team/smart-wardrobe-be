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

### 3. Outbox Pattern — Shared Table + Polling Worker

| Field | Detail |
|-------|--------|
| **Decision** | 1 shared `outbox_records` table cho tất cả module. `topic` field quyết định RMQ routing key. Polling worker 1s/lần |
| **Why** | Đơn giản, 1 worker quét 1 bảng, đủ cho MVP. Khi scale (10k+ records/min) mới tách per-module table |
| **Trade-off** | 1s polling latency giữa outbox write và RMQ publish. Khi 1 module produce quá nhiều, worker có thể bị nghẽn |
| **Đừng** | Đừng tạo riêng outbox table per module ở MVP. Nếu sau này cần, chỉ cần `CREATE TABLE module_x_outbox (LIKE outbox_records)` |

### 4. Notification Email — Module Riêng, Không Shared Package

| Field | Detail |
|-------|--------|
| **Decision** | `internal/modules/notification/` — module riêng với kiến trúc async (RMQ + outbox + worker). **Không** đặt trong `shared/` |
| **Why** | Email có retry + failed queue + template + history tracking — đầy đủ đặc điểm business module. Module khác gọi qua `presentation/contract/` |
| **Trade-off** | Nặng hơn shared package: cần migration riêng, worker riêng, DI wiring riêng. Nhưng đúng kiến trúc modular monolith |
| **Đừng** | Đừng đặt trong `shared/` — notification không phải shared infrastructure, là business module |

### 5. Notification Contract Interface — Generic, Extensible

| Field | Detail |
|-------|--------|
| **Decision** | `INotificationContract.Send(ctx, channel ChannelEnum, template TemplateEnum, params map[string]any, to string) error`. Không specific methods per email type |
| **Why** | Extensible cho push/SMS sau này mà không break contract. Thêm channel mới = thêm enum value |
| **Trade-off** | Hơi abstract hơn specific methods (`SendOtpEmail`, `SendPaymentEmail`...). Tham số `params map[string]any` mất type safety |
| **Đừng** | Đừng tạo interface kiểu `SendOtpEmail`, `SendPaymentSuccessEmail` riêng lẻ |

### 6. EmailTemplate — Code Constants, Không DB Table

| Field | Detail |
|-------|--------|
| **Decision** | HTML string constants trong Go files. Deploy là thay đổi template |
| **Why** | Đơn giản, type-safe, review được content trong code review. DB table là overengineering — template thay đổi rất ít |
| **Trade-off** | Muốn thay đổi template phải deploy lại code (không thể hotfix qua DB) |
| **Đừng** | Đừng tạo bảng email_template. Cũng đừng hardcode template trong use case — để trong file constants riêng |

### 7. Email History — Worker Log Only, Không Admin UI

| Field | Detail |
|-------|--------|
| **Decision** | Không admin API/UI cho email history. Chỉ worker log |
| **Why** | Giảm scope cho MVP. Worker log đủ debug |
| **Trade-off** | Không thể tra cứu email history từ UI nếu không có admin |

### 8. Elasticsearch — Loại Bỏ Khỏi Hệ Thống

| Field | Detail |
|-------|--------|
| **Decision** | ES đang được lên kế hoạch loại bỏ. Không có US nào cho ES trong spec này |
| **Trade-off** | Nếu dev cần ES debug, có thể bật bằng CLI flag — nhưng feature này đã bị remove khỏi spec |
| **Đừng** | Đừng tạo US cho ES hardening. ES sẽ được xử lý riêng ngoài spec này |

---

## Design Constraints

### Task File Structure
- **Mỗi sprint 1 file task riêng**: `tasks-sprint1.md` → `tasks-sprint4.md`
- **Hoàn thành tất cả task files trước khi bắt đầu implement**
- **Implement tuần tự theo sprint**: Sprint 1 → 2 → 3 → 4

### Infrastructure Layer — Không Import Presentation
- Infrastructure (ES, RMQ, HTTP clients) KHÔNG được import `BaseWorker` từ `shared/presentation/worker/`
- Infrastructure chỉ dùng Go primitives: `chan struct{}`, `context.Context` để stop/cleanup
- `BaseWorker` chỉ dùng cho business workers ở presentation layer (FashionInitWorker, DashboardAggregator, EmailWorker...)

### Outbox là Primary Path
- Tất cả event publishing phải qua transactional outbox. **Không fallback** — outbox là primary
- `RabbitMQClient.Publish()` luôn ghi OutboxRecord, không gửi trực tiếp đến RMQ
- Outbox worker đọc từ outbox table và publish lên RMQ

---

## Domain Conventions

### Cross-Module Domain Constants
- Domain constants (`domain/constants/`) của module khác **được phép import trực tiếp**
- **Nhưng**: Không được hardcode giá trị (viết `"active"` thay vì `wardrobestatus.Active`)
- Lý do: constants là pure enum values, không behavior. Duplicate gây maintenance overhead

### Payment Refund Flow
```
Payment success → subscription apply fail
  → ghi vào provider_webhook_inbox (status: PENDING)
  → WebhookInboxWorker claim + retry (exponential backoff: 1m * 2^attempt)
  → attempt >= maxRetryAttempts (default 5)
  → MarkFailed + auto-refund vào wallet
  → deposit_transaction status = REFUNDED
  → wallet statement type = "auto_refund"
```

### Notification Email Flow
```
Module A (identity/subscription)
  → INotificationContract.Send(channel=Email, template=XxxTemplate, params, to)
  → notification module save EmailMessage (status: pending)
  → ghi outbox record (cùng DB transaction)
  → outbox worker publish to RMQ queue notification_email_queue
  → EmailWorker consume → SmtpSender.send() → update status = sent/failed
```

---

## Important Reasoning

### Tại sao không dùng Retry Queue Mới
`provider_webhook_inbox` đã có đủ: `ProcessingAttempts`, `NextProcessingAt`, `ProcessingToken` (SKIP LOCKED), `ProcessingStatus`. Thiếu: cap on retries + exponential backoff + terminal auto-refund. Đây là mở rộng logic use case, không cần migration hay bảng mới.

### Tại sao Notification Module Riêng (Không Shared)
Email có retry logic, failed queue, template rendering, delivery history — giống business module hơn là shared utility. Nếu đặt trong `shared/`, các module khác import trực tiếp → vi phạm module boundaries. Contract pattern (`presentation/contract/`) giải quyết cross-module coupling.

### Tại sao không Audit Trail DB Table
Dev chọn chưa implement audit trail. Lý do: chưa có yêu cầu thực tế, ELK stack là overkill cho MVP. Có thể thêm sau khi có nhu cầu.

### Tại sao không Proration / Trial / Grace Period
Theo nghiệp vụ: hệ thống không support extend ngày dùng của tháng. Marketing chưa lên kế hoạch trial. Scope giới hạn ở auto-renew notification + wallet balance warning.

---

## Hidden Context

1. **PayOS "Chi hộ" (payout API)**: PayOS có API `POST /v1/payouts` nhưng đang ở giai đoạn quảng bá mới ra mắt. Dev quyết định **không dùng chi hộ**. Refund hoàn toàn qua wallet nội bộ.

2. **Task numbering jumps**: Sprint 4 FR bắt đầu từ FR-022 (không phải FR-018). FR-018 và FR-019 không tồn tại (bỏ qua do renumbering). FR-020/FR-021 đã được renumber thành FR-029/FR-030, sau đó FR-030 bị xóa vì duplicate với FR-003.

3. **Config struct**: `MaxRetryAttempts` và `wallet_balance_warning_days` là config mới cần thêm vào `internal/config/config.go` ở subscription section.

4. **Database migration files**: Cần tạo file `.sql` mới dưới `database/migrations/` cho mỗi migration task (T001 của mỗi sprint). Không sửa file migration cũ.

5. **DI Wiring**: Mỗi sprint thay đổi DI wiring. Notification module (Sprint 4) cần Wire ProviderSet mới. Identity module cần thay đổi inject từ `IEmailService` sang `INotificationContract`.

---

## TODO

### Sprint 1 — Foundation Security & Payment Operations (25 tasks)
**File**: `tasks-sprint1.md` — US1 (refund), US2 (password), US3 (worker lifecycle)

1. **Phase 1** (T001-T002): DB migrations (refund fields + inbox status)
2. **Phase 2** (T003-T012): Auto-refund + auto-renew toggle → **ưu tiên cao nhất**
3. **Phase 3** (T013-T016): Password policy (dễ, chỉ thêm validator tag)
4. **Phase 4** (T017-T025): Worker lifecycle + BaseWorker

**⚠️ Note**: Khi implement T011 (auto-renew API), nhớ rằng FR-030 đã bị xóa — auto-renew chỉ implement 1 lần ở Sprint 1.

### Sprint 2 — Cross-Module Compliance & N+1 (20 tasks)
**File**: `tasks-sprint2.md` — US1 (constants), US2 (contract), US3 (N+1)

1. **Phase 1** (T001): DB migration (indexes)
2. **Phase 2** (T002-T007): Scan hardcoded constants → **US1 và US2 có thể chạy song song**
3. **Phase 3** (T008-T013): Cross-module use case → contract
4. **Phase 4** (T014-T020): N+1 queries + indexes

### Sprint 3 — Production Hardening (13 tasks)
**File**: `tasks-sprint3.md` — US1 (HTTP + RMQ + outbox)

1. **Phase 1** (T001): DB migration (outbox table)
2. **Phase 2** (T002-T013): HTTP transport, RMQ reconnect, outbox

**⚠️ Note**: T011: outbox là PRIMARY path, không fallback.

### Sprint 4 — Notification Module (25 tasks)
**File**: `tasks-sprint4.md` — US0 (module), US1 (RMQ+outbox), US2 (auto-renew), US3 (payment)

1. **Phase 1** (T001-T014): Extract notification module — **move SMTP/templates từ identity**
2. **Phase 2** (T015-T019): RMQ + outbox integration
3. **Phase 3** (T020-T022): Auto-renew notification
4. **Phase 4** (T023-T025): Payment & top-up email

**⚠️ Note**: Phase 1 và Phase 2 phụ thuộc outbox infrastructure từ Sprint 3. Nếu chưa có Sprint 3, notification module có thể sync gửi SMTP tạm thời.

---

## Risks

1. **Sprint 3 là prerequisite cho Sprint 4**: Outbox infrastructure + RMQ reconnect phải hoàn thành trước khi notification module (Sprint 4) có thể dùng outbox. Nếu implement Sprint 4 trước Sprint 3, notification module phải gọi SMTP sync.

2. **WebhookInboxWorker hiện cron 30s/lần**: Retry latency 30s. Nếu không chấp nhận được, cần giảm interval nhưng sẽ tăng DB load. Chưa có quyết định.

3. **Notification module phụ thuộc identity module hiện tại**: Move `GmailSmtpService` + templates từ identity sang notification có thể gây regression nếu identity còn dùng interface cũ. Cần test OTP flow sau khi chuyển.

4. **Elasticsearch deprecation không có migration plan**: ES đang được lên kế hoạch loại bỏ nhưng chưa có strategy cho data hiện tại. Có thể gây mất dữ liệu search nếu không có plan.

---

## Open Questions

1. **Worker interval cho retry và outbox**: Hiện tại WebhookInboxWorker chạy cron 30s/lần, outbox worker poll 1s/lần. Con số này có thể cần tuning dựa trên production load, nhưng chưa có config.

2. **Sprint 3 → Sprint 4 dependency**: Nếu notification module cần outbox từ Sprint 3, thứ tự implement bắt buộc phải Sprint 3 trước Sprint 4. Nếu muốn Sprint 4 độc lập, notification module cần fallback sync SMTP.

3. **Template localization**: Hiện tại email templates chỉ support tiếng Việt. Chưa có quyết định về multi-language support trong tương lai.
