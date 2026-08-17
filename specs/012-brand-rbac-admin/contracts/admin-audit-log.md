# Contract: Admin Audit Log (IAuditLogContract)

**Date**: 2026-08-17 | **Feature**: [spec.md](../spec.md)

Contract ghi nhật ký hành động quản trị; module `admin` cung cấp, các module khác inject để ghi. **Ghi bất đồng bộ qua outbox + RabbitMQ** (đúng hiến pháp §III): `Record` chỉ publish outbox record, consumer module `admin` ghi vào bảng.

---

## Interface (admin/presentation/contract/audit_log_contract.go)

```go
package contract

import "context"

type RecordAuditInput struct {
    ActorUserID string         `json:"actorUserId"`
    Action      string         `json:"action"`      // snake_case, vd "user.deactivate"
    TargetType  string         `json:"targetType"`  // user | brand | brand_member | deposit_transaction | ...
    TargetID    *string        `json:"targetId,omitempty"`
    Payload     map[string]any `json:"payload"`     // không chứa secret/token/password
    RequestIP   string         `json:"requestIp,omitempty"`
}

type IAuditLogContract interface {
    Record(ctx context.Context, input RecordAuditInput) error
}
```

- Triển khai: `admin/presentation/contract/audit_log_contract_impl.go` → `event.IEventPublisher.Publish(ctx, eventconstants.TopicAuditLog, input)` (ghi `outbox_records`).
- **Không ghi DB trực tiếp**: việc ghi bảng `admin_audit_logs` do consumer thực hiện.
- Fire-and-forget: lỗi publish không làm fail nghiệp vụ chính (log error).

## Message flow

1. Handler admin (identity/subscription/brand/aiexecutor) gọi `IAuditLogContract.Record` → ghi `outbox_records` (topic `admin.audit.log`, payload = `RecordAuditInput`).
2. `OutboxWorker` (shared) claim outbox → publish tới topic exchange với routing key `TopicAuditLog`.
3. Consumer module `admin` (`presentation/worker/audit_log_consumer.go`) đăng ký queue `QueueAuditLog` → unmarshal payload → `IAuditLogUseCase.Record` → INSERT `admin_audit_logs`.
4. `GET /admin/audit-logs` đọc từ bảng (bản ghi có thể trễ vài nhịp consumer so với hành động).

**Hằng số mới**:
- `eventconstants.TopicAuditLog = "admin.audit.log"` (`internal/shared/application/constants/eventconstants/eventconstants.go`)
- `QueueAuditLog = "audit_log_queue"` + `RoutingKeyAuditLog = eventconstants.TopicAuditLog` (`internal/shared/infrastructure/messaging/rabbitmq_constants.go`)
- Bind queue trong `rabbitmq_connection.go` (`DeclareAndBindQueue(QueueAuditLog, RoutingKeyAuditLog)`)

## Bảng (schema `shared`)

`admin_audit_logs` — xem chi tiết tại [data-model.md](../data-model.md).

## Danh sách action chuẩn

| Action | TargetType | Ghi từ |
|--------|-----------|--------|
| `user.create` | user | identity admin |
| `user.role_change` | user | identity admin |
| `user.deactivate` | user | identity admin |
| `user.activate` | user | identity admin (route hiện có) |
| `user.sessions_revoke` | user | identity admin |
| `brand.create_admin` | brand | brand admin |
| `brand.approve` / `brand.suspend` / `brand.archive` | brand | brand admin (route hiện có UpdateBrandStatusAdmin) |
| `brand.member_add` / `member_role_change` / `member_remove` | brand_member | brand admin mới |
| `brand.ownership_transfer` | brand_member | brand admin mới |
| `payment.retry_investigation` / `resolve_paid` / `resolve_cancelled` / `resolve_creation_failed` | deposit_transaction | subscription admin |
| `subscription.plan_create` / `plan_update` / `plan_deactivate` | subscription_plan | subscription admin |
| `subscription.cancel` / `subscription.grant` | user_subscription | subscription admin |
| `ai.pricing_update` | ai_model_config | aiexecutor admin |
| `ai.grant_create` | ai_policy_grant | aiexecutor admin |

## Payload khuyến nghị

- `user.deactivate`: `{oldStatus, newStatus, reason}`
- `brand.approve`: `{oldStatus, newStatus, reason}`
- `payment.resolve_paid`: `{orderCode, status, reference, reason}`
- `subscription.grant`: `{planId, planCode, userId}`
- `ownership_transfer`: `{oldOwnerUserId, newOwnerUserId, oldOwnerHandling}`

## Endpoint xem audit (chỉ admin)

- `GET /admin/audit-logs` — filter `actorUserId`, `action`, `targetType`, `targetId`, `fromDate`, `toDate`, `page`, `limit`; response phân trang chuẩn.
- Dữ liệu giữ 12 tháng; cleanup worker (`admin/presentation/worker/audit_cleanup_worker.go`) xóa bản ghi cũ định kỳ.

## Wire

- Admin provider: `usecase.NewAuditLogUseCase`, `contract.NewAuditLogContract` (publish outbox), handler, router, `worker.NewAuditLogConsumer` (consume + ghi bảng), `worker.NewAuditCleanupWorker`.
- Module khác: inject `contract.IAuditLogContract` vào handler admin (identity, subscription, brand, aiexecutor).
- Đăng ký `AdminRouter` vào AppRouter + NewEngine + wire.go; đăng ký consumer + cleanup worker vào bootstrap worker.

## Quy tắc

- Không ghi payload chứa mật khẩu, access/refresh token, secret.
- Ghi audit bất đồng bộ: bản ghi xuất hiện sau khi consumer xử lý (at-least-once qua outbox; không chặn nghiệp vụ chính).