# Implementation Plan: Khả năng quan sát cho admin (029)

**Branch**: `029-admin-observability` | **Date**: 2026-10-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification từ `specs/029-admin-observability/spec.md` + 5 clarify đã chốt (session 2026-10-15) + kết quả khảo sát code ngày 2026-10-15 (đã ghi trong `research.md`).

## Summary

Bốn nhóm thay đổi, không migration:

1. **API mới** `GET /api/v1/admin/activity-events` (account/identity): phân trang + lọc
   userId/activityType/kỳ ngày VN, `LEFT JOIN users` trả kèm email/tên; route riêng để tránh xung
   đột wildcard gin với `/admin/users/:id`.
2. **Chuẩn hoá contract giao dịch payOS** (account/subscription): `GET /transactions/{id}` trả DTO
   camelCase; `status` serialize thành chuỗi đủ 11 giá trị (gồm `failed_legacy`, khứ hồi filter);
   kỳ ngày của list sửa theo chuẩn VN `[from, to)` + validate 400 (trước đây lệch 7h và nuốt lỗi).
3. **Kỳ dữ liệu dashboard nhất quán** (commerce + account + ai): một chuẩn kỳ duy nhất
   (ngày lịch VN, cận nửa mở `[from, to)`, helper dùng chung, 400 khi sai); `distribution` đếm
   user có gói hiệu lực giao với kỳ; `renewal-stats` lọc theo `created_at`; `error-breakdown` cùng
   kỳ cho tử/mẫu/phân nhóm; `payos-reconciliation` chuyển từ ảnh chụp sang truy vấn theo kỳ, phạm
   vi cả `direct_purchase` + `wallet_topup`, `totalSuccessValue` là tổng tiền giao dịch thành công
   thật. Proto mở rộng additive (2 file), không đổi field cũ.
4. **Bàn giao**: `frontend-guide.md` (mọi nhóm API mới/thay đổi — activity-events, giao dịch payOS,
   dashboard admin — kèm before→after + hướng dẫn tích hợp 2 trang + đánh giá UX
   dashboard FE), `deferred.md`, docs API (3 file) + Swagger + regen proto.

## Technical Context

**Language/Version**: Go 1.25+, multi-module workspace (`go.work`, mỗi service một `go.mod`)

**Primary Dependencies**: GORM, Gin, gRPC/protobuf (`contracts/subscription/v1`, `contracts/ai/v1`),
Wire DI, buf (`make buf-gen`), Swag (`make swagger-all`); Goose **không dùng** (0 migration)

**Storage**: PostgreSQL — đọc `identity.user_activity_events` (account), `subscription.deposit_transactions`,
`subscription.user_subscriptions`, `subscription.subscription_renewal_attempts` (account),
`aiexecutor.ai_usage_events` (ai), `dashboard_admin_daily_snapshots` (commerce, chỉ worker ghi).
Không thêm bảng/cột.

**Testing**: `go test -race -count=3` scoped theo service + `make lint` (gate mỗi phase);
`make check` trước push/tag. Pattern SQL pinning sẵn có
(`services/account/.../persistence/admin_stats_query_sql_test.go`) dùng cho các truy vấn kỳ mới.

**Target Platform**: 3-service topology (Account / Commerce / AI Engine), chạy cả VPS Compose lẫn
managed — cùng image, chỉ khác config.

**Project Type**: backend web-service (REST admin + gRPC liên service)

**Performance Goals**: Truy vấn mới đều bounded theo kỳ và dùng index sẵn có
(`user_activity_events(occurred_at)`, `(user_id, occurred_at)`; `deposit_transactions(created_at)`;
`subscription_renewal_attempts(created_at)`; `ai_usage_events(created_at)`) — đối chiếu lại schema
khi implement, không thêm index nếu truy vấn hiện tại đã dùng. Panel payOS chuyển từ đọc 1 dòng
snapshot sang 1 câu aggregate theo kỳ (thêm 1 gRPC call — chấp nhận được cho màn admin).

**Constraints**: Proto additive (service cũ bỏ qua field mới; deploy account/ai trước commerce);
kỳ dữ liệu cố định múi giờ VN bằng `timeutils` (không phụ thuộc tzdata container); DTO hướng
frontend camelCase; không đụng DTO user-facing `PaymentLinkDTO.paymentStatus` (giữ số).

**Scale/Scope**: 2 proto + regen; ~10 file account, ~6 file commerce, ~4 file ai, ~4 file contract
client/transport, 1 helper shared; 3 docs API + guide + deferred; 0 migration.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- §I (deploy độc lập): PASS — proto additive, không phá wire; thứ tự deploy account/ai → commerce;
  degradation tạm khi client mới gặp server cũ đã ghi trong `contracts/grpc-contract-delta.md` §4.
- §II (module boundary/contract): PASS — activity-events thuộc account identity (chủ sở hữu bảng,
  không join chéo service); commerce đọc subscription/AI qua đúng contract gRPC + dual adapter sẵn có.
- §III (async/outbox): N/A — không thêm event/worker mới.
- §IV (layering): PASS — handler mỏng (parse + gọi usecase), mapper mới nằm `application/mapper`,
  repo implement domain interface; helper parse kỳ đặt ở `shared/pkg/utils/validation` (shared, không
  phải cross-module implementation import).
- §V (error/transaction): PASS — 400/404 qua `apperror` với thông báo tiếng Việt; không thêm transaction.
- §VI (AI cost control): N/A — chỉ đọc số liệu lỗi.
- §Code Style: PASS — JSON camelCase (FR-004/007/008), tài liệu tiếng Việt.
- §Governance — frontend guide: PASS — FR-018/019/020; guide + sync đối chiếu có đếm ở phase cuối.
- §Governance — technical debt: PASS — FR-021 `deferred.md`.
- Không thay đổi kiến trúc, không cần bump version constitution.

Re-check sau Phase 1: không phát sinh vi phạm mới — không migration, không endpoint ghi mới,
thay đổi contract đều additive hoặc đã ghi before→after cho guide.

## Project Structure

### Documentation (this feature)

```text
specs/029-admin-observability/
├── plan.md                      # File này
├── research.md                  # Phase 0: R1–R12
├── data-model.md                # Phase 1: ngữ nghĩa đọc + validation
├── quickstart.md                # Phase 1: 5 kịch bản kiểm chứng + lệnh test
├── contracts/
│   ├── admin-api-delta.md       # Phase 1: HTTP contract (mới/thay đổi, before→after)
│   └── grpc-contract-delta.md   # Phase 1: proto additive + chuỗi adapter
├── checklists/
│   └── requirements.md          # Spec quality (21/21)
├── frontend-guide.md            # Phase cuối: bàn giao FE (FR-018..020)
├── deferred.md                  # Phase cuối: hạng mục ngoài phạm vi (FR-021)
└── tasks.md                     # Phase 2 (speckit-tasks)
```

### Source Code (repository root)

```text
shared/pkg/utils/validation/
└── date_range.go                    # MỚI: ParseVNDateRangeQuery (400 tiếng Việt)

services/account/dto/identity/
└── activity.go                      # MỚI: ActivityEventRes + query req (camelCase)

services/account/internal/modules/identity/
├── domain/repositories/user_activity_repo.go        # + ListAdmin + readmodel
├── domain/repositories/readmodels.go                # + AdminActivityEventRow
├── infrastructure/persistence/user_activity_repo_impl.go  # LEFT JOIN users, ORDER, COUNT
├── application/interface/usecase/user_uc.go         # + ListActivityEvents
├── application/usecase/activity_uc.go               # MỚI (method trên UserUseCase)
├── application/mapper/activity_mapper.go            # MỚI (row → DTO)
├── presentation/handler/activity_handler.go         # MỚI (method trên AdminHandler)
└── presentation/router/router.go                    # + group /admin/activity-events

services/account/internal/modules/subscription/
├── application/mapper/deposit_status.go             # MỚI: DepositStatusToString (11 giá trị)
├── application/mapper/deposit_transaction.go        # MỚI: entity → DepositTransactionRes
├── presentation/handler/admin_billing_handler.go    # detail→DTO; parse kỳ VN + 400; + failed_legacy
├── infrastructure/persistence/deposit_transaction_repo_impl.go  # ListAdmin `<`; stats SQL (2 type + value)
├── domain/repositories/{deposit_transaction_repo,readmodels}.go # filter semantics; + TotalPayOSSuccessValue
├── application/usecase/admin_stats/admin_stats_uc.go + interface # chữ ký (from, to)
└── presentation/grpc/server.go                      # request mới → usecase

services/account/contract/subscription/
├── dashboard_reader.go                              # interface + in-process (from, to)
└── transport/{mapper,to_proto}.go                   # field 7 + cặp ngày

contracts/subscription/v1/{subscription.proto, *.pb.go}   # additive; regen make buf-gen
contracts/subscription/v1/client/dashboard_reader_client.go

services/ai/internal/modules/aiexecutor/
├── domain/repositories/ai_cost_repo.go              # GetAIErrorBreakdown(ctx, from, to)
├── infrastructure/persistence/ai_cost_repo_impl.go  # + range; usage/request `<` chuẩn VN
├── application/usecase/cost/usecase.go + interface   # chữ ký mới
└── presentation/grpc/server.go                      # request mới
services/ai/contract/aiexecutor/{dashboard_reader.go, transport/*}
contracts/ai/v1/{ai.proto, *.pb.go, client/dashboard_reader_client.go}

services/commerce/internal/modules/dashboard/
├── presentation/handler/admin_dashboard_handler.go  # helper kỳ dùng chung; tham số mới
├── presentation/router/router.go                    # (không đổi path)
├── application/usecase/admin_dashboard_uc_impl.go   # payos kỳ+phạm vi; error-rate cùng kỳ; distribution/renewal kỳ
├── application/dto/admin_dashboard_dto.go           # (shape giữ; ghi chú ngữ nghĩa)
└── application/usecase/aggregator_uc_impl.go        # worker: VNDayRange (không đổi hành vi)

docs/api/
├── identity/admin-activity-api.md                   # MỚI
├── subscription/admin-billing-api.md                # cập nhật before→after
├── dashboard/dashboard-admin-api.md                 # cập nhật tham số kỳ + ngữ nghĩa
└── README.md                                        # cập nhật mục lục
```

**Structure Decision**: Sửa tại chỗ theo module sở hữu dữ liệu (identity/subscription/account,
dashboard/commerce, aiexecutor/ai); bề mặt liên service mở rộng đúng chuỗi contract + dual adapter
sẵn có; 1 helper dùng chung ở `shared/pkg/utils/validation` để hai service không lặp quy tắc kỳ.

## Complexity Tracking

> Không có vi phạm constitution cần biện minh — bảng này để trống theo template.
