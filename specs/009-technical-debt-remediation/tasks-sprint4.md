# Tasks: Technical Debt & Business Logic Remediation — Sprint 4

**Input**: Design documents from `specs/009-technical-debt-remediation/`

**Scope**: Sprint 4 — Notification Module & Business Logic Maturation (Priority: P3)

| US | Description | Priority |
|----|-------------|----------|
| US0 | Foundation — Extract notification module riêng | P3 |
| US1 | Foundational — RMQ + outbox integration | P3 |
| US2 | Auto-renew notification & wallet balance warning | P3 |
| US3 | Email notification cho payment & top-up | P3 |

---

## Phase 1: Foundation — Extract notification module (Priority: P3)

**Goal**: Tách email sender khỏi identity module thành module `notification` riêng. Move `GmailSmtpService`, templates, interface. Module khác gọi qua `presentation/contract/`.

**Independent Test**: Identity module vẫn gửi được OTP email qua contract. Module notification có thể gửi email độc lập.

- [ ] T001 Create module structure: `internal/modules/notification/domain/entities/email.go` — `EmailMessage` struct (to, subject, body, status, attempts, lastError, createdAt, sentAt, channel) với GORM tags + table name
- [ ] T002 Create email status constants: `internal/modules/notification/domain/constants/emailstatus/emailstatus.go` — `Pending`, `Sending`, `Sent`, `Failed`
- [ ] T003 Create email repository interface: `internal/modules/notification/domain/repositories/interfaces.go` — `IEmailRepository` with `Create`, `UpdateStatus`, `GetByID`
- [ ] T004 Create email repository GORM impl: `internal/modules/notification/infrastructure/persistence/email_repo.go`
- [ ] T005 Create notification contract interface: `internal/modules/notification/presentation/contract/notification_contract.go` — `INotificationContract` with generic method `Send(ctx, channel ChannelEnum, template TemplateEnum, params map[string]any, to string) error`. Define `ChannelEnum` = {Email}, `TemplateEnum` = {Otp, PaymentSuccess, TopupSuccess, RenewalSuccess, RenewalFailed, WalletWarning}. Interface extensible — thêm channel/template sau không break contract.
- [ ] T006 Create contract implementation: `internal/modules/notification/presentation/contract/notification_contract_impl.go` — nhận generic params, lookup template constants theo `TemplateEnum`, render HTML với `params`, tạo `EmailMessage(channel=ChannelEnum)`, gọi `IEmailRepository.Create`, publish to RMQ via outbox
- [ ] T007 Move `GmailSmtpService` from `internal/modules/identity/infrastructure/communication/gmail_smtp_service.go` to `internal/modules/notification/infrastructure/communication/smtp_sender.go` — refactor thành `SmtpSender`
- [ ] T008 Move email templates from `internal/modules/identity/infrastructure/communication/templates/` to `internal/modules/notification/infrastructure/communication/templates/`. Templates là code constants (HTML string trong Go files), không phải DB entity.
- [ ] T009 Create send email use case: `internal/modules/notification/application/usecase/send_email_uc.go` — nhận contract request, validate, save EmailMessage (status: pending), ghi outbox + publish to RMQ queue `notification_email`
- [ ] T010 Create `INotificationUseCase` interface: `internal/modules/notification/application/interface/usecase/interfaces.go`
- [ ] T011 Create module provider: `internal/modules/notification/provider.go` — Wire ProviderSet
- [ ] T012 Update identity module: replace `IEmailService` injection with `INotificationContract` in `identity/application/usecase/register_uc.go` and `password_recovery_uc.go`; update DI wiring. Identity sẽ gọi `contract.Send(ctx, EmailChannel, OtpTemplate, params, email)` thay vì `emailService.SendRegistrationOtpEmail(...)`
- [ ] T013 Remove old `IEmailService` interface at `identity/application/interface/communication/email_service.go` (no longer needed)
- [ ] T014 Register `notification` module in DI: update `internal/di/wire.go` (không cần route registration — notification module không có handler/public endpoints, chỉ có worker)

**Checkpoint**: Module notification hoạt động, identity gửi OTP qua contract, SMTP sending hoạt động

---

## Phase 2: Foundational — RMQ + outbox integration (Priority: P3)

**Goal**: Email sending là async qua RMQ queue + outbox pattern (tái sử dụng outbox từ Sprint 3). Worker riêng consume queue và gửi SMTP.

**Independent Test**: Gọi contract → outbox record created → worker consume → SMTP sent → email status updated.

- [ ] T015 [P] [US1] Create email queue constants: `internal/modules/notification/infrastructure/messaging/queue_constants.go` — queue name `notification_email_queue`, routing key `notification.email.*`
- [ ] T016 [P] [US1] Modify `send_email_uc.go` to integrate with outbox: save `EmailMessage` + outbox record in same DB transaction, outbox worker (từ Sprint 3) publish to RMQ queue `notification_email_queue`
- [ ] T017 [US1] Create email worker: `internal/modules/notification/presentation/worker/email_worker.go` — consume từ `notification_email_queue`, gọi `SmtpSender` để gửi email, update `EmailMessage.status` = `sent` hoặc `failed`
- [ ] T018 [US1] Implement retry logic in email worker: nếu SMTP fail, `EmailMessage.attempts++`, nếu < max retry (3) thì ghi lại outbox + publish lại RMQ; nếu >= max retry → status = `failed`
- [ ] T019 [US1] Register email worker in `internal/bootstrap/app.go` — add `EmailWorker` to `AppWorkers`, start/stop với lifecycle

**Checkpoint**: Email queue hoạt động, worker gửi SMTP async, retry khi fail

---

## Phase 3: User Story 2 — Auto-renew notification & wallet balance warning (Priority: P3)

**Goal**: Khi subscription sắp hết hạn và auto-renew đang bật, nếu wallet không đủ tiền, hệ thống gửi email cảnh báo.

**Independent Test**: Tạo subscription sắp hết hạn trong 3 ngày, auto-renew bật, wallet thiếu → user nhận email cảnh báo.

- [ ] T020 [P] [US2] Add `wallet_balance_warning_days int` config to subscription config section in `internal/config/config.go` (default 3)
- [ ] T021 [US2] Implement use case: `internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go` — add `ProcessUpcomingExpiryNotifications(ctx)` query các subscription: expires trong N ngày, auto-renew enabled, wallet balance < plan price → gọi `contract.Send(ctx, EmailChannel, WalletWarningTemplate, {daysRemaining, balance, requiredAmount}, userEmail)`
- [ ] T022 [US2] Modify `SubscriptionRenewalWorker.executeRenewalJob()` in `internal/modules/subscription/presentation/worker/subscription_renewal_worker.go` — gọi `ProcessUpcomingExpiryNotifications` trước khi xử lý renewals

**Checkpoint**: User nhận email cảnh báo trước khi subscription hết hạn nếu wallet không đủ

---

## Phase 4: User Story 3 — Payment & top-up email notification (Priority: P3)

**Goal**: Gửi email xác nhận khi user top-up hoặc payment subscription thành công.

**Independent Test**: Top-up → email "Nạp tiền thành công". Payment → email "Thanh toán thành công".

- [ ] T023 [US3] Modify wallet top-up use case: `internal/modules/subscription/application/usecase/wallet/wallet_uc.go` — after top-up success, inject `INotificationContract` và gọi `contract.Send(ctx, EmailChannel, TopupSuccessTemplate, {amount, newBalance}, userEmail)`
- [ ] T024 [US3] Modify payment webhook use case: `internal/modules/subscription/application/usecase/webhook/payment_webhook_uc.go` — after payment completed + subscription activated, gọi `contract.Send(ctx, EmailChannel, PaymentSuccessTemplate, {planName, amount, expiresAt}, userEmail)`
- [ ] T025 [US3] Modify renewal workflow: `internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go` — after renewal success, gọi `contract.Send(ctx, EmailChannel, RenewalSuccessTemplate, ...)`, after renewal fail, gọi `contract.Send(ctx, EmailChannel, RenewalFailedTemplate, ...)`

**Checkpoint**: User nhận email cho mọi giao dịch tài chính quan trọng

---

## Dependencies & Execution Order

```
Phase 1 (Foundation): T001 → T002 → T003 → T004 → T005 → T006 → T009 → T010
                             T007 ─── T008 (move từ identity)
                             T011 ─── T012 ─── T013 ─── T014 (identity update + DI)

Phase 2 (RMQ+outbox): T015 ───── T016 ───── T017 ───── T018 ───── T019
                  (phụ thuộc Phase 1 và outbox từ Sprint 3)

Phase 3 (Auto-renew): T020 ───── T021 ───── T022
                  (phụ thuộc Phase 2 — cần contract + worker)

Phase 4 (Payment notif): T023 ───── T024 ───── T025
                  (phụ thuộc Phase 2 — cần contract)
```

### Parallel Opportunities

| Tasks | Reason |
|-------|--------|
| T007, T008 | Move file thuần, không phụ thuộc code logic — song song |
| T015, T020, T023 | Có thể bắt đầu sau Phase 1, không phụ thuộc nhau |
| Phase 3 & Phase 4 | Có thể chạy song song sau Phase 2 (cùng dùng contract) |

### Implementation Strategy

1. **Phase 1** (Foundation — 14 tasks): Tạo module notification, move SMTP + templates, update identity → nặng nhất
2. **Phase 2** (Foundational — 5 tasks): Worker + outbox integration → sau Phase 1
3. **Phase 3** (Auto-renew — 3 tasks): US2 → sau Phase 2
4. **Phase 4** (Payment notif — 3 tasks): US3 → sau Phase 2
