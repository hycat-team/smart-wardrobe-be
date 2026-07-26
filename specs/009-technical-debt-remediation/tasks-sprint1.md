# Tasks: Technical Debt & Business Logic Remediation — Sprint 1

**Input**: Design documents from `specs/009-technical-debt-remediation/`

**Prerequisites**: plan.md, spec.md

**Organization**: Tasks are grouped by user story for independent implementation and testing of each story.

**Scope**: Sprint 1 — Foundation Security & Payment Operations (Priority: P1)

| US | Description | Priority |
|----|-------------|----------|
| US1 | Xử lý hoàn tiền tự động qua wallet + auto-renew toggle | P1 |
| US2 | Password policy | P1 |
| US3 | Worker lifecycle & graceful shutdown | P1 |

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Maps task to user story (US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Database Migrations)

**Purpose**: Thêm các trường và status cần thiết cho US1.1 (refund + retry tracking)

- [ ] T001 Create Goose migration adding `refunded_at TIMESTAMPTZ`, `refund_amount NUMERIC(12,2)`, `refund_reason VARCHAR(255)` to `deposit_transactions` in `database/migrations/`
- [ ] T002 Create Goose migration adding `failed` status to `provider_webhook_inbox.processing_status` — nếu DB không có CHECK constraint thì bỏ qua, chỉ add nếu cần trong `database/migrations/`

**Checkpoint**: DB schema sẵn sàng cho US1.1

---

## Phase 2: User Story 1 — Xử lý hoàn tiền tự động qua wallet (Priority: P1) 🎯 MVP

**Goal**: Khi payment thành công nhưng subscription apply thất bại, hệ thống tái sử dụng `provider_webhook_inbox` để retry với exponential backoff. Hết max attempts (mặc định 5) → tự động refund vào wallet nội bộ. User có thể bật/tắt auto-renew.

**Independent Test**: Payment success + subscription apply fail → inbox record retry bởi WebhookInboxWorker → retry lần thứ 5 vẫn fail → system tự refund, wallet balance tăng, deposit transaction status REFUNDED.

### Implementation for User Story 1

- [ ] T003 [P] [US1] Add `Refunded DepositStatus = 10` to `internal/modules/subscription/domain/constants/depositstatus/depositstatus.go`
- [ ] T004 [P] [US1] Add `AutoRefund WalletStatementType = "auto_refund"` to `internal/modules/subscription/domain/constants/walletstatementtype/walletstatementtype.go`
- [ ] T005 [P] [US1] Add `MarkFailed(ctx, id uuid.UUID, at time.Time, reason string) error` to `IWebhookInboxRepository` interface in `internal/modules/subscription/domain/repositories/payment_event_repo.go`
- [ ] T006 [P] [US1] Implement `MarkFailed` in `internal/modules/subscription/infrastructure/persistence/webhook_inbox_repo_impl.go` — set `processing_status = "failed"`, `next_processing_at = NULL`, `processing_error = reason`, clear `processing_token` and `processing_lease_until`
- [ ] T007 [P] [US1] Add `MaxRetryAttempts int` field to subscription config section in `internal/config/config.go` (default 5)
- [ ] T008 [US1] Modify `MarkRetry` in `internal/modules/subscription/infrastructure/persistence/webhook_inbox_repo_impl.go` — set `next_processing_at = NOW() + min(1 minute * 2^processing_attempts, 30 minutes)` for exponential backoff
- [ ] T009 [US1] Modify `ProcessInbox` in `internal/modules/subscription/application/usecase/webhook_inbox/webhook_inbox_usecase.go` — after `MarkRetry` check if `ProcessingAttempts >= maxRetryAttempts`, if so call `MarkFailed` + trigger auto-refund
- [ ] T010 [US1] Implement auto-refund logic: create wallet statement with type `AutoRefund`, credit wallet balance, set `DepositTransaction.status = Refunded` with `refunded_at`, `refund_amount`, `refund_reason` in `internal/modules/subscription/application/usecase/webhook/payment_webhook_uc.go`
- [ ] T011 [US1] Add API `POST /api/v1/subscriptions/auto-renew` — toggle auto-renew for current user subscription in `internal/modules/subscription/presentation/handler/billing_handler.go`
- [ ] T012 [US1] Implement auto-renew toggle use case in `internal/modules/subscription/application/usecase/subscription/` — read current `IsAutoRenewEnabled`, invert and save

**Checkpoint**: US1 hoàn chỉnh — payment fail → retry → auto-refund hoạt động, user có thể toggle auto-renew

---

## Phase 3: User Story 2 — Password policy (Priority: P1)

**Goal**: Áp dụng password complexity validation cho register, reset password, change password. Validator `password_complexity` đã có sẵn trong `internal/pkg/utils/validation/validation.go`.

**Independent Test**: Đăng ký với password "123" → reject với message tiếng Việt. Đăng ký với "Abcd@123" → thành công.

### Implementation for User Story 2

- [ ] T013 [P] [US2] Add `validate:"required,password_complexity"` tag to `Password` field in `RegisterReq` DTO at `internal/modules/identity/application/dto/auth_dto.go`
- [ ] T014 [P] [US2] Add `validate:"required,password_complexity"` tag to `Password` field in `ResetPasswordReq` DTO at `internal/modules/identity/application/dto/auth_dto.go`
- [ ] T015 [P] [US2] Add `validate:"required,password_complexity"` tag to `NewPassword` field in change password DTO (nếu có) — tìm trong `internal/modules/identity/application/dto/`
- [ ] T016 [US2] Ensure all auth handlers (register, reset-password, change-password) use `validation.BindJSON(c, &input)` — confirm in `internal/modules/identity/presentation/handler/auth_handler.go`

**Checkpoint**: US2 hoàn chỉnh — mọi endpoint auth đều validate password complexity

---

## Phase 4: User Story 3 — Worker lifecycle & graceful shutdown (Priority: P1)

**Goal**: Tất cả background workers quản lý lifecycle qua `Start(ctx)`/`Stop()`. BaseWorker chung. Background goroutines có stop mechanism. App graceful shutdown trong < 5 giây.

**Independent Test**: Start app → gửi SIGTERM → verify tất cả workers dừng, không goroutine leak.

### Implementation for User Story 3

- [ ] T017 [P] [US3] Create `shared/presentation/worker/base_worker.go` — `BaseWorker` struct với `Start(ctx)`, `Stop()`, `IsRunning()`, built-in `context.WithCancel`, health check callback slot, restart tracking counter
- [ ] T018 [US3] Refactor `FashionInitWorker` in `internal/modules/fashion/presentation/worker/fashion_init_worker.go` — remove `go w.startConsume()` từ constructor; implement `Start(ctx)` spawns consumer goroutine; `Stop()` cancels ctx + drains channel
- [ ] T019 [P] [US3] Refactor `FashionAnalyzeWorker` in `internal/modules/fashion/presentation/worker/fashion_analyze_worker.go` — same pattern as T018
- [ ] T020 [P] [US3] Refactor `SearchSyncWorker` in `internal/modules/wardrobe/presentation/worker/search_sync_worker.go` — move `go manageInitialSyncAndRecovery()` và `go startConsume()` vào `Start(ctx)`; `Stop()` cancels ctx
- [ ] T021 [P] [US3] Add stop mechanism to `ElasticsearchClient.startHealthCheck()` in `internal/shared/infrastructure/search/elasticsearch.go` — accept stopChan (chan struct{}), exit on channel close. **Note**: infrastructure layer — dùng channel primitive, KHÔNG import `BaseWorker` từ presentation layer (vi phạm Principle IV — strict layer dependency).
- [ ] T022 [P] [US3] Add stopChan to `RateLimitMiddleware.startCleanupJob()` in `internal/api/middleware/ratelimit.go` — select on ticker vs stopChan
- [ ] T023 [P] [US3] Add context-aware stop to `RabbitMQClient.handleReconnect()` in `internal/shared/infrastructure/messaging/rabbitmq_connection.go` — select on ctx.Done() vs errChan, break reconnect loop. **Note**: infrastructure layer — dùng context primitive, KHÔNG import `BaseWorker` từ presentation (vi phạm Principle IV).
- [ ] T024 [P] [US3] Update `DashboardAggregatorWorker` in `internal/modules/dashboard/presentation/worker/dashboard_aggregator_worker.go` — consolidate to use shared BaseWorker pattern nếu phù hợp
- [ ] T025 [US3] Update `bootstrap/app.go` — change `AppWorkers` interface to use `Start(ctx)` instead of `Start()`/`Start(ctxWorker)`; create root context with cancel; pass to all workers; call all `Stop()` in shutdown sequence

**Checkpoint**: US3 hoàn chỉnh — SIGTERM → tất cả workers + goroutines dừng trong 5 giây

---

## Dependencies & Execution Order

### Phase Dependencies

```
T001 ───── T002      (Setup — Phase 1)
                  │
                  ├── T003 ───── T004 ───── T005 ───── T006 ───── T007
                  │                                                   │
                  │                                           T008 ───┤
                  │                                                  │
                  │                                           T009 ───┤
                  │                                                  │
                  │                                           T010 ───┤
                  │                                                  │
                  │                                           T011 ───┤
                  │                                                  │
                  │                                           T012 ───┤ (US1 — Phase 2)
                  │                                                  │
                  ├── T013 ───── T014 ───── T015 ───── T016          │ (US2 — Phase 3)
                  │                                                  │
                  └── T017 ───── T018 ───── T019 ───── T020          │ (US3 — Phase 4)
                               ├── T021 ───── T022
                               ├── T023
                               └── T024 ───── T025
```

### User Story Dependencies

- **US1 (Phase 2)**: Depends on T001-T002 (setup). No dependency on other stories
- **US2 (Phase 3)**: Independent — can run in parallel with US1 and US3 after T001-T002
- **US3 (Phase 4)**: T017 (BaseWorker) là prerequisite cho T018-T025. Can run in parallel with US1, US2

### Parallel Opportunities

| Tasks | Reason |
|-------|--------|
| T003, T004, T005, T006, T007 | Khác file, không phụ thuộc — chạy song song |
| T013, T014, T015 | Khác DTO, không phụ thuộc — chạy song song |
| T018, T019, T020, T021, T022, T023, T024 | Sau T017 (BaseWorker), khác file — chạy song song |

---

## Implementation Strategy

### MVP Scope

US1 (T001-T012) là P1 và có thể launch độc lập:

1. Complete Phase 1: T001-T002 (DB setup)
2. Complete Phase 2: T003-T012 (US1 — refund + auto-renew)
3. **STOP and VALIDATE**: Test US1 independently
4. Deploy/demo nếu cần

### Incremental Delivery

1. **US1 done** → MVP: payment refund + auto-renew
2. Add **US2** → Password policy enforced
3. Add **US3** → Graceful shutdown, no goroutine leak

### Parallel Execution Example

```bash
# Launch US1 parallel tasks:
# T003 - deposit status constant
# T004 - wallet statement type
# T005 - MarkFailed interface
# T006 - MarkFailed implementation
# T007 - config field

# Launch US2 parallel tasks (after T001-T002):
# T013 - RegisterReq validator tag
# T014 - ResetPasswordReq validator tag
# T015 - ChangePassword DTO validator tag
```

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story independently completable and testable
- US1 and US3 share no overlapping files — can be done in parallel with 2 devs
- T007 (config) should be added to existing config struct, not break existing loaders
- All Goose migrations must be reversible (`--down` migration included)
