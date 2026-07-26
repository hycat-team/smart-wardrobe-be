# Implementation Plan: Technical Debt & Business Logic Remediation

**Branch**: `009-technical-debt-remediation` | **Date**: 2026-07-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/009-technical-debt-remediation/spec.md`

## Summary

Remediate 4 sprints of technical debt: (1) Payment retry/auto-refund + password policy + worker lifecycle, (2) Cross-module contract compliance + N+1 fixes + missing indexes, (3) HTTP per-provider config + RabbitMQ reconnect + outbox pattern, (4) Auto-renew notification. Tất cả tuân thủ module boundaries, không cross-module join/preload, gọi qua contract.

## Technical Context

**Language/Version**: Go 1.25+, GORM v1.25.12, Gin v1.11.0

**Primary Dependencies**:
- `gorm.io/gorm` — database
- `github.com/rabbitmq/amqp091-go` — messaging
- `github.com/redis/go-redis/v9` — caching
- `github.com/go-playground/validator/v10` — validation (password policy)
- PayOS REST API — payment gateway

**Storage**: PostgreSQL (migrations via Goose), Redis (caching/rate-limit)

**Testing**: `go test ./...` + `make test` (existing convention)

**Target Platform**: Linux server (VPS/docker), Windows dev

**Project Type**: Go modular monolith (Gin web service)

**Performance Goals**: Worker shutdown < 5s, outbox latency < 1s, N+1 elimination

**Constraints**:
- No cross-module entity joins/preloads
- Module A → Module B via `presentation/contract/` only
- Domain constants cross-module là allowed
- Database changes via Goose SQL migrations only

**Scale/Scope**: 4 sprints, ~12 phases, ~20 files affected

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status |
|------|--------|
| I. Pragmatic Modular Monolith — không cross-module preload/join entities | ✅ Sẽ kiểm tra ở Sprint 2 |
| II. Strict Module Boundaries & DTO Contracts — module gọi nhau qua contract | ✅ Sprint 2.2 sửa 2 violations |
| III. Async Event-Driven — RabbitMQ cho slow tasks | ✅ Sprint 3.2 outbox pattern |
| IV. Clean Architecture — handlers thin, delegate to use cases | ✅ Worker lifecycle Sprint 1.3 |
| V. Standardized Error Handling — apperror + UoW | ✅ Refund + retry dùng UoW |
| VI. AI Pipeline Cost Control — không liên quan | N/A |
| Goose migrations — không sửa init-db | ✅ Thêm migration file mới |

**Gate Evaluation**: ✅ PASS — không có vi phạm hiến pháp nào không thể giải quyết

## Project Structure

### Documentation (this feature)

```text
specs/009-technical-debt-remediation/
├── spec.md              # Feature specification
├── plan.md              # This file
├── tasks-sprint1.md     # Sprint 1 tasks (25 tasks)
├── tasks-sprint2.md     # Sprint 2 tasks (20 tasks)
├── tasks-sprint3.md     # Sprint 3 tasks (13 tasks)
├── tasks-sprint4.md     # Sprint 4 tasks (25 tasks)
└── checklists/          # Quality checklists
```

### Source Code (repository root)

```text
internal/
├── modules/
│   ├── identity/
│   │   ├── application/usecase/register_uc.go        # Sprint 1.2
│   │   └── presentation/handler/auth_handler.go       # Sprint 1.2
│   ├── subscription/
│   │   ├── domain/entities/payment.go                 # Sprint 1.1
│   │   ├── domain/constants/depositstatus.go          # Sprint 1.1
│   │   ├── application/usecase/webhook/               # Sprint 1.1
│   │   ├── application/usecase/subscription/           # Sprint 4.1
│   │   └── presentation/handler/billing_handler.go     # Sprint 1.1
│   ├── wardrobe/
│   │   ├── infrastructure/persistence/outfit_repo.go   # Sprint 2.3
│   │   └── presentation/contract/                      # Sprint 2.2 (mới)
│   ├── fashion/
│   │   ├── infrastructure/persistence/fashion_item_repo_hybrid*.go # Sprint 2.3
│   │   └── application/usecase/ai/                     # Sprint 2.2
│   ├── brand/
│   │   └── infrastructure/persistence/brand_member_repo.go # Sprint 2.3
│   ├── aiexecutor/
│   │   └── infrastructure/persistence/ai_cost_repo_impl.go # Sprint 2.3
│   └── dashboard/
│       └── application/usecase/                       # Sprint 2.2
│   └── notification/
│       ├── domain/entities/email.go                   # Sprint 4
│       ├── domain/constants/emailstatus/
│       ├── domain/repositories/
│       ├── infrastructure/persistence/email_repo.go
│       ├── infrastructure/communication/smtp_sender.go    # Move từ identity
│       ├── infrastructure/communication/templates/        # Move từ identity
│       ├── application/usecase/send_email_uc.go
│       ├── presentation/contract/notification_contract.go # INotificationContract
│       ├── presentation/worker/email_worker.go            # RMQ consumer
│       └── provider.go
├── bootstrap/
│   └── app.go                                         # Sprint 1.3
└── shared/
    ├── presentation/worker/
    │   └── base_worker.go                             # Sprint 1.3 (mới) — BaseWorker struct
    └── infrastructure/
        ├── messaging/rabbitmq*.go                     # Sprint 3.1
        ├── messaging/outbox.go                        # Sprint 3.1 (mới)
        └── ai/core.go                                 # Sprint 3.1 — fix http.Transport
```

## Complexity Tracking

Không có vi phạm cần justification.

---

## Phase 0: Research — Đã giải quyết

**Mục tiêu**: Giải quyết các NEEDS CLARIFICATION trước khi thiết kế chi tiết.

### Quyết định kỹ thuật (sau khi thảo luận với dev)

| STT | US | Vấn đề | Quyết định | Lý do |
|-----|-----|--------|------------|-------|
| 1 | US1.1 | Retry pattern | **Tái sử dụng `provider_webhook_inbox`** — Webhook handler save record vào inbox với status PENDING. `WebhookInboxWorker` (đã có sẵn) claim record qua `ProcessingToken` (locking), retry subscription apply, tăng `ProcessingAttempts` mỗi lần, dùng `NextProcessingAt` cho exponential backoff. Hết max attempts → auto-refund vào wallet. **Không tạo bảng mới** | Bảng inbox đã có sẵn cơ chế retry, locking, attempts tracking. Không cần reinvent |
| 2 | US3.1 | HTTP per-provider config | **Loại bỏ US3.1** — Gộp thành sub-task nhỏ trong Sprint 3.1: cấu hình 1 shared `http.Transport` với `MaxIdleConnsPerHost=50` cho tất cả HTTP client | Per-provider config là overengineering. Vấn đề thực sự chỉ là default transport 2 conn/host |
| 3 | US3.2 | Outbox mechanism | **Polling worker 1s/lần** + transactional outbox table | Đơn giản, đủ cho MVP. Nâng cấp LISTEN/NOTIFY sau khi scale |
| 4 | US1.3 | Worker lifecycle | **`shared.BaseWorker`** struct chung với Start(ctx), Stop(), health check, restart tracking | Đồng bộ, dễ bảo trì, tránh duplicate code |
| 5 | US4.0 | Notification contract design | **Generic interface**: `INotificationContract.Send(channel, template, params, to)` với channel enum + template enum — không specific methods per email type | Extensible cho push/SMS sau này, không break contract |
| 6 | US4.0 | EmailTemplate | **Code constants** — HTML string trong Go files, không cần DB table cho template | Đơn giản, deploy là thay đổi template |
| 7 | US4.2 | Email history | **Worker log** — không admin API/UI cho email history ở MVP | Giảm scope |

### Cập nhật cấu trúc source code

```text
# Thay đổi so với dự thảo ban đầu:
# ❌ US3.1 per-provider config → Đã loại bỏ
# ❌ Bảng retry_queue → Không tạo, tái sử dụng provider_webhook_inbox
# ✅ shared.BaseWorker → Thêm mới tại internal/shared/presentation/worker/
# ✅ OutboxRecord → Thêm mới tại shared/infrastructure/messaging/outbox.go
# ✅ Notification module → Thêm mới tại internal/modules/notification/ (thay vì shared)
#     - Kiến trúc: module riêng với RMQ + outbox + worker
#     - move GmailSmtpService, templates từ identity sang
#     - expose INotificationContract qua presentation/contract/
```
