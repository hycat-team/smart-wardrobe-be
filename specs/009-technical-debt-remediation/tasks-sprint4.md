# Tasks: Technical Debt & Business Logic Remediation — Sprint 4

**Input**: Design documents from `specs/009-technical-debt-remediation/`

**Scope**: Sprint 4 — Notification Module & Business Logic Maturation (Priority: P3)

| US | Description | Priority |
|----|-------------|----------|
| US0 | Foundation — Extract notification module riêng | P3 |
| US1 | Foundational — RMQ + outbox integration | P3 |
| US2 | Auto-renew notification & wallet balance warning | P3 |
| US3 | Email notification cho payment & top-up | P3 |

**Total**: 18 tasks

---

## Phase 1: Foundation — Extract notification module (Priority: P3)

**Goal**: Tách email sender khỏi identity module thành module `notification` riêng. Move `SmtpSender`, templates, contract interface. Module khác gọi qua `presentation/contract/`. Email delivery tracking dùng worker log + RMQ DLQ — không tạo bảng `email_messages`.

**Independent Test**: Identity module vẫn gửi được OTP email qua contract. Module notification có thể gửi email độc lập.

- [x] T001 Create notification contract interface: `internal/modules/notification/presentation/contract/notification_contract.go` — `INotificationContract` with generic method `Send(ctx, channel ChannelEnum, template TemplateEnum, params map[string]any, to string) error`. Define `ChannelEnum` = {Email}, `TemplateEnum` = {Otp, PaymentSuccess, TopupSuccess, RenewalSuccess, RenewalFailed, WalletWarning}. Interface extensible — thêm channel/template sau không break contract.
- [x] T002 Create contract implementation: `internal/modules/notification/presentation/contract/notification_contract_impl.go` — lookup template constants theo `TemplateEnum`, render HTML với `html/template` engine và `params`, ghi outbox record (topic=`notification.email.*`, payload=rendered body + to + subject). **Không** tạo EmailMessage entity, không gọi repository. Outbox record là nguồn duy nhất.
- [x] T003 Move `GmailSmtpService` from `internal/modules/identity/infrastructure/communication/gmail_smtp_service.go` to `internal/modules/notification/infrastructure/communication/smtp_sender.go` — refactor thành `SmtpSender`
- [x] T004 Move email templates from `internal/modules/identity/infrastructure/communication/templates/` to `internal/modules/notification/infrastructure/communication/templates/`. Templates là code constants (HTML string trong Go files), không phải DB entity.
- [x] T005 Create module provider: `internal/modules/notification/provider.go` — Wire ProviderSet (gồm SmtpSender, INotificationContract, worker dependencies). **Không** cần email_repo, email entity, hay email use case.
- [x] T006 Update identity module: replace `IEmailService` injection with `INotificationContract` in `identity/application/usecase/register_uc.go` and `password_recovery_uc.go`; update DI wiring. Identity sẽ gọi `contract.Send(ctx, EmailChannel, OtpTemplate, params, email)` thay vì `emailService.SendRegistrationOtpEmail(...)`
- [x] T007 Remove old `IEmailService` interface at `identity/application/interface/communication/email_service.go` (no longer needed)
- [x] T008 Register `notification` module in DI: update `internal/di/wire.go` (không cần route registration — notification module không có handler/public endpoints, chỉ có worker)

**Checkpoint**: Module notification hoạt động, identity gửi OTP qua contract, SMTP sending hoạt động

---

## Phase 2: Foundational — RMQ + outbox integration (Priority: P3)

**Goal**: Contract implementation ghi outbox record → outbox worker (Sprint 3) publish to RMQ queue `notification_email` → email worker consume → `SmtpSender.Send()`. Retry qua RMQ DLQ — không custom retry logic.

**Independent Test**: Gọi contract → outbox record created → outbox worker publish → email worker consume → SMTP sent.

- [x] T009 [P] [US1] Add notification queue constants to `internal/shared/infrastructure/messaging/rabbitmq_constants.go` (`QueueNotificationEmail = "notification_email_queue"`, `RoutingKeyNotificationEmail = "notification.email.*"`) and event topic to `internal/shared/application/constants/eventconstants/eventconstants.go` (`TopicNotificationEmail = "notification.email"`) — nhất quán với các constants RMQ hiện tại, không tạo file constants riêng trong notification module
- [x] T010 [US1] Modify contract implementation (T002) to integrate with outbox: khi `Send()` được gọi, ghi outbox record (`topic = TopicNotificationEmail`, `payload = {to, subject, body, channel}`) trong cùng DB transaction với business operation (top-up, payment, renewal). Dùng `db.GetTx(ctx)` từ caller để ghi cùng transaction. Outbox worker từ Sprint 3 publish lên RMQ. **Không** save EmailMessage riêng.
- [x] T011 [US1] Create email worker: `internal/modules/notification/presentation/worker/email_worker.go` — trong `Start()`, gọi `rmqClient.DeclareAndBindQueue(QueueNotificationEmail, RoutingKeyNotificationEmail)` (idempotent), sau đó `Consume(QueueNotificationEmail)`. Parse payload, gọi `SmtpSender.Send(ctx, to, subject, body)`. Nếu success: ack message, log "email sent". Nếu fail: nack message → RMQ redeliver → DLQ sau max retries (do RMQ config). **Không** update EmailMessage.status, không custom attempts tracking.
- [x] T012 [US1] Register email worker in `internal/bootstrap/app.go` — add `EmailWorker` to `AppWorkers`, start/stop với lifecycle

**Checkpoint**: Email queue hoạt động, worker gửi SMTP async, retry qua RMQ DLQ

---

## Phase 3: User Story 2 — Auto-renew notification & wallet balance warning (Priority: P3)

**Goal**: Khi subscription sắp hết hạn và auto-renew đang bật, nếu wallet không đủ tiền, hệ thống gửi email cảnh báo.

**Independent Test**: Tạo subscription sắp hết hạn trong 3 ngày, auto-renew bật, wallet thiếu → user nhận email cảnh báo.

- [x] T013 [P] [US2] Add `wallet_balance_warning_days int` config to subscription config section in `internal/config/config.go` (default 3)
- [x] T014 [US2] Implement use case: `internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go` — add `ProcessUpcomingExpiryNotifications(ctx)` query các subscription: expires trong N ngày, auto-renew enabled, wallet balance < plan price → gọi `contract.Send(ctx, EmailChannel, WalletWarningTemplate, {daysRemaining, balance, requiredAmount}, userEmail)`
- [x] T015 [US2] Modify `SubscriptionRenewalWorker.executeRenewalJob()` in `internal/modules/subscription/presentation/worker/subscription_renewal_worker.go` — gọi `ProcessUpcomingExpiryNotifications` trước khi xử lý renewals

**Checkpoint**: User nhận email cảnh báo trước khi subscription hết hạn nếu wallet không đủ

---

## Phase 4: User Story 3 — Payment & top-up email notification (Priority: P3)

**Goal**: Gửi email xác nhận khi user top-up hoặc payment subscription thành công.

**Independent Test**: Top-up → email "Nạp tiền thành công". Payment → email "Thanh toán thành công".

- [x] T016 [US3] Modify wallet top-up use case: `internal/modules/subscription/application/usecase/wallet/wallet_uc.go` — inject `INotificationContract`. Gọi `contract.Send(txCtx, EmailChannel, TopupSuccessTemplate, {amount, newBalance}, userEmail)` **bên trong** `uow.Execute` callback (đã có UoW cho top-up), dùng `txCtx` để outbox record atomic với wallet transaction.
- [x] T017 [US3] Modify payment webhook use case: `internal/modules/subscription/application/usecase/webhook/payment_webhook_uc.go` — inject `INotificationContract`. Gọi `contract.Send(txCtx, EmailChannel, PaymentSuccessTemplate, {planName, amount, expiresAt}, userEmail)` **bên trong** existing UoW callback, dùng `txCtx`.
- [x] T018 [US3] Modify renewal workflow: `internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go` — inject `INotificationContract`. Gọi `contract.Send(txCtx, EmailChannel, RenewalSuccessTemplate, ...)` và `contract.Send(txCtx, EmailChannel, RenewalFailedTemplate, ...)` **bên trong** existing UoW callback, dùng `txCtx`.

**Checkpoint**: User nhận email cho mọi giao dịch tài chính quan trọng

---

## Dependencies & Execution Order

```
Phase 1 (Foundation): T005 ───── T006 ───── T007 ───── T008 (identity update + DI)
                       T001 ───── T002 ───── T003 ───── T004 (move từ identity)

Phase 2 (RMQ+outbox): T009 ───── T010 ───── T011 ───── T012
                  (phụ thuộc Phase 1 và outbox từ Sprint 3)

Phase 3 (Auto-renew): T013 ───── T014 ───── T015
                  (phụ thuộc Phase 2 — cần contract + worker)

Phase 4 (Payment notif): T016 ───── T017 ───── T018
                  (phụ thuộc Phase 2 — cần contract)
```

### Parallel Opportunities

| Tasks | Reason |
|-------|--------|
| T001, T003, T004 | Khác file, không phụ thuộc — tạo contract, move SMTP, move templates song song |
| T009, T013, T016 | Có thể bắt đầu sau Phase 1, không phụ thuộc nhau |
| Phase 3 & Phase 4 | Có thể chạy song song sau Phase 2 (cùng dùng contract) |

### Implementation Strategy

1. **Phase 1** (Foundation — 8 tasks): Tạo module notification, move SMTP + templates, update identity
2. **Phase 2** (Foundational — 4 tasks): Worker + outbox integration → sau Phase 1. T009 thêm constants vào shared files, không tạo file riêng
3. **Phase 3** (Auto-renew — 3 tasks): US2 → sau Phase 2
4. **Phase 4** (Payment notif — 3 tasks): US3 → sau Phase 2

### Task Savings

So với thiết kế cũ (25 tasks), đã loại bỏ 7 tasks:
- **T001-T004**: EmailMessage entity, status constants, repository interface, GORM repo — không cần
- **T009-T010**: send_email_use_case, INotificationUseCase interface — logic dồn vào T002+T010
- **T018**: custom retry attempts — RMQ DLQ handle thay thế
