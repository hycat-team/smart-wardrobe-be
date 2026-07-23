# Implementation Plan: Sửa Lỗi Dashboard

**Branch**: `007-fix-dashboard-issues` | **Date**: 2026-07-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/007-fix-dashboard-issues/spec.md`

---

## Summary

Sửa 7 lỗi trong Dashboard APIs (Feature 006) được phát hiện qua audit: granularity không hoạt động, filters/groupBy chết, fallback thiếu data, MetricKey không đầy đủ, error handling sai, field semantic mismatch. Đây là các bugfix trong module dashboard hiện tại — không thêm entity hay endpoint mới.

---

## Technical Context

**Language/Version**: Golang 1.25+ (compiler constraint từ constitution)

**Primary Dependencies**: Gin (web framework), GORM (ORM), Google Wire (DI), Postgres (DB)

**Storage**: PostgreSQL — chỉ đọc từ bảng `dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_brand_item_daily_snapshots`, `dashboard_user_wardrobe_summaries`

**Testing**: Go standard testing (`go test`) — đã có test files cho use cases: `admin_dashboard_uc_test.go`, `dynamic_query_uc_test.go`, `drilldown_uc_test.go`, `brand_dashboard_uc_test.go`, `user_dashboard_uc_test.go`

**Target Platform**: Linux server (existing backend)

**Project Type**: Web service (Gin-based HTTP API, modular monolith)

**Performance Goals**: Không thay đổi performance profile — thêm in-memory aggregation, không IO mới ngoại trừ fallback reader calls

**Constraints**:
- Module boundaries: không preload entity của module khác
- Tuân thủ Clean Architecture: handler thin, business logic trong use case
- Error messages bằng tiếng Việt
- GORM column tags đã có sẵn

**Scale/Scope**: Feature này chỉ sửa code có sẵn — không thêm endpoint, entity, migration, hay seed data

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| I. Modular Monolith — không preload entity chéo module | ✅ Pass | Fallback dùng reader interface (đã có pattern), không join entity |
| II. Strict Module Boundaries & DTO Contracts | ✅ Pass | Thay đổi nội bộ DTO trong dashboard module; cross-module qua contract interfaces |
| III. Asynchronous Event-Driven Processing | ✅ Không ảnh hưởng | Không thay đổi worker/event logic |
| IV. Clean Architecture and Thin Controllers | ✅ Pass | Handler giữ nguyên thin — logic thêm vào use case |
| V. Standardized Transaction & Error Handling | ✅ Pass | Dùng `apperror.NewInternalError` (đã có pattern) |
| VI. AI Pipeline, Cost Control & Local Fallbacks | ✅ Không ảnh hưởng | Không thay đổi AI pipeline |
| Technology Stack & Database Constraints | ✅ Pass | Không migration mới, không thay đổi DB |
| Code Style, Naming & Language Standards | ✅ Pass | Error messages Vietnamese, code identifiers English |
| Access Control (RBAC) & Data Privacy | ✅ Pass | RBAC redact pattern giữ nguyên (FR-007 xoá field, không ảnh hưởng RBAC) |

**Kết luận**: Không có vi phạm constitution. Gate PASS.

---

## Project Structure

### Documentation (this feature)

```text
specs/007-fix-dashboard-issues/
├── plan.md              # File này
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output — DTO changes detail
├── quickstart.md        # Phase 1 output — validation guide
├── contracts/           # Phase 1 output — interface definitions
└── tasks.md             # Do speckit-tasks tạo sau
```

### Source Code (repository root)

```text
internal/modules/dashboard/
├── provider.go                          # [Sửa] Inject subscription reader mới
├── application/
│   ├── app.go                           # [Sửa] App-level constants (nếu cần)
│   ├── dto/
│   │   ├── admin_dashboard_dto.go       # [Không đổi]
│   │   ├── brand_dashboard_dto.go       # [Không đổi]
│   │   ├── dynamic_query_dto.go         # [Sửa] FR-002: xoá Filters/GroupBy
│   │   └── user_dashboard_dto.go        # [Sửa] FR-007: xoá averageItemValueVnd
│   ├── interface/usecase/               # [Không đổi]
│   ├── mapper/                          # [Không đổi]
│   └── usecase/
│       ├── admin_dashboard_uc_impl.go   # [Sửa] FR-003: fallback subscription; FR-006: error propagate
│       ├── aggregator_uc_impl.go        # [Không đổi]
│       ├── brand_dashboard_uc_impl.go   # [Sửa] FR-004: fallback sentiment/paired
│       ├── drilldown_uc_impl.go         # [Không đổi]
│       ├── dynamic_query_uc_impl.go     # [Sửa] FR-001: granularity aggregation; FR-005: new MetricKeys
│       └── user_dashboard_uc_impl.go    # [Sửa] FR-007: xoá averageItemValueVnd mapping
├── domain/
│   ├── constants/metrickey/metric_key.go # [Sửa] FR-005/FR-007: thêm 7 brand keys, xoá average_cost_per_wear_vnd
│   └── entities/dashboard_snapshots.go   # [Không đổi]
└── presentation/handler/
    ├── admin_dashboard_handler.go       # [Không đổi] (error tự propagate)
    ├── analytics_handler.go             # [Không đổi]
    ├── brand_dashboard_handler.go       # [Không đổi]
    └── user_dashboard_handler.go        # [Không đổi]

internal/modules/brand/presentation/contract/
└── dashboard_reader.go                  # [Sửa] Thêm method GetItemFeedbackByItemID nếu chưa có

internal/modules/aiexecutor/presentation/contract/
└── ...                                  # [Không đổi]

internal/shared/application/dto/
└── pagination.go                        # [Không đổi]

docs/api/
├── dashboard-api.md                     # [Đồng bộ] Sau Phase 1
└── constants/dashboard.md               # [Đồng bộ] Sau Phase 1
```

**Structure Decision**: Single project — dashboard module internal. No new modules created.

---

## Complexity Tracking

> Không có vi phạm constitution — section để trống.

---

## Phase 0: Research & Unknown Resolution

### NEEDS CLARIFICATION từ spec

Tất cả đã được resolve qua `/speckit-clarify`:

1. **Filters/GroupBy** → Xoá khỏi DTO (FR-002)
2. **AdminOverview error** → `apperror.NewInternalError` trả 500 (FR-006)
3. **averageItemValueVnd** → Xoá khỏi DTO + xoá MetricKey (FR-007)

### Research tasks

| Unknown | Context | Method |
|---------|---------|--------|
| Granularity aggregation pattern | Cần xác định cách gộp daily → monthly/weekly trong Go | In-memory: đọc daily snapshots, group bằng `map[yearMonth]`, tính SUM/AVG/LAST_VALUE |
| Subscription reader cho fallback | Cần reader cho subscription revenue + PayOS stats | Kiểm tra các module hiện tại: không có subscription reader. Cần inject qua Wire. Tạo interface mới `depositContract.IDepositDashboardReader` nếu có sẵn, hoặc thêm method vào userReader |
| Brand item feedback/paired fallback | Cần reader cho per-item sentiment + paired categories | `brandReader.GetBrandAnalyticsStats` trả về brand-level, không per-item. Cần thêm method `GetItemFeedbackSummary` vào `IBrandDashboardReader` |
| MetricKey cho brand fields | 7 fields mới | Tạo constants trong `metric_key.go`, thêm switch cases trong `dynamic_query_uc_impl.go` |

### Decisions

Xem [research.md](research.md) cho chi tiết.

---

## Phase 1: Design & Contracts

### 1. DTO Changes — data-model.md

Xem [data-model.md](data-model.md) cho chi tiết từng field thay đổi.

Tóm tắt:
- `DynamicQueryReq`: xoá `Filters`, `GroupBy`
- `UserWardrobeInsightsRes`: xoá `AverageItemValueVND`
- Thêm 7 MetricKey constants mới
- Xoá MetricKey `AverageCostPerWearVND`

### 2. Contract Interfaces

Xem [contracts/](contracts/) cho chi tiết.

- **New**: `ISubscriptionDashboardReader` interface trong `internal/modules/deposit/presentation/contract/` hoặc mở rộng `userReader`
- **Update**: `IBrandDashboardReader` thêm `GetItemFeedbackSummary(ctx, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error)`

### 3. Quickstart Validation

Xem [quickstart.md](quickstart.md).

---

## Post-Design Constitution Re-check

| Gate | Status | Notes |
|------|--------|-------|
| I. Modular Monolith — không preload entity chéo module | ✅ Pass | `ISubscriptionDashboardReader` là contract interface mới (DTO-based), không preload entity |
| II. Strict Module Boundaries & DTO Contracts | ✅ Pass | Mọi cross-module communication qua contract interfaces |
| V. Standardized Transaction & Error Handling | ✅ Pass | `apperror.NewInternalError` cho FR-006 |
| Code Style, Naming & Language Standards | ✅ Pass | Error messages Vietnamese, code English |

**Kết luận**: Design pass constitution. Ready for `/speckit-tasks`.
