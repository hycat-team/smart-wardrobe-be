# Implementation Plan: Dashboard & Analytics System

**Branch**: `006-dashboard-analytics` | **Date**: 2026-07-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/006-dashboard-analytics/spec.md`

---

## Summary

Triển khai module **Dashboard & Analytics** (`internal/modules/dashboard/`) phục vụ 3 nhóm đối tượng (System Admin, Brand Manager B2B, End User B2C).

**Phương án triển khai thực tế (Tránh over-engineering)**:
- **Tận dụng hạ tầng hiện tại**: Sử dụng Go (v1.25), Gin Framework, GORM, PostgreSQL và Redis hiện có.
- **Mô hình CQRS Read Model Snapshots**: Sử dụng Goose Migration tạo 3 bảng Snapshots (`dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_user_wardrobe_summaries`).
- **Aggregation Worker**: Một Ticker Worker trong `internal/modules/dashboard/presentation/worker/` tổng hợp dữ liệu định kỳ (15-30 phút) từ Read Contracts của các domain module (`identity`, `brand`, `wardrobe`, `subscription`, `aiexecutor`) và đổ vào Snapshot Tables + Redis.
- **API Interfaces**: Cung cấp Dynamic Aggregation Query (`POST /api/v1/dashboard/analytics/query`), Widget Endpoints, và Drill-down Endpoints.
- **Bảo mật**: Multi-tenant JWT scoping + Column-level RBAC DTO mappers (`Owner` vs `Staff`).

---

## Technical Context

**Language/Version**: Golang v1.25+  
**Primary Dependencies**: Gin Web Framework, GORM, Google Wire (DI), Go-Redis, Goose DB Migrations  
**Storage**: PostgreSQL (main datastore & read snapshots), Redis (caching)  
**Testing**: Go standard `testing` package  
**Target Platform**: Linux server / Docker container  
**Project Type**: Modular Monolith Web Service Backend  
**Performance Goals**: < 150ms p95 response time cho tất cả API Dashboard  
**Constraints**: Tuân thủ 100% SmartWardrobe Constitution (Clean Architecture, Bounded Contexts, DTO Contracts, Vietnamese client error messages).  

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **Pragmatic Modular Monolith**: Module mới nằm ở `internal/modules/dashboard/`, DI qua Wire, không tạo microservice mới không cần thiết.
- [x] **Strict Module Boundaries**: Không import DB entities hay GORM structs của các module khác. Chỉ dùng DTOs qua `presentation/contract/`.
- [x] **Thin Controllers**: Handler tuân thủ signature `func (h *Handler) Method(c *gin.Context) error` và được bọc qua `shared_pres.WrapHandler`.
- [x] **Goose Migrations**: Mọi thay đổi bảng Read Snapshots đều dùng Goose SQL Migration.
- [x] **Vietnamese Client Errors / English Code**: Toàn bộ Swagger text và thông báo lỗi bằng tiếng Việt, code comment và identifiers bằng tiếng Anh.
- [x] **RBAC & Multi-tenancy**: Phân quyền `roleslug.Admin`, `brandmemberrole.Owner`, `brandmemberrole.Staff`, `roleslug.User` và kiểm tra `brand_id` từ JWT Claims.

---

## Project Structure

### Documentation (this feature)

```text
specs/006-dashboard-analytics/
├── spec.md              # Feature specification
├── plan.md              # Implementation Plan (this file)
├── research.md          # Phase 0 output (Tech decisions & trade-offs)
├── data-model.md        # Phase 1 output (DB schema & Go entities)
├── quickstart.md        # Phase 1 output (Runnable validation guide)
└── contracts/           # Phase 1 output (REST API & DTO contracts)
    └── api-schema.md
```

### Source Code Structure

```text
internal/modules/dashboard/
├── application/
│   ├── dto/                    # Request/Response DTOs (Dynamic Query, KPIs, Drilldown)
│   ├── mapper/                 # Entity/DTO mappers with Role Redaction logic
│   ├── interface/
│   │   └── usecase/            # Dashboard UseCase interfaces
│   └── usecase/                # Dashboard UseCase implementations
├── domain/
│   ├── entities/               # GORM entities for Snapshot tables
│   └── repositories/           # Repository interfaces
├── infrastructure/
│   ├── persistence/            # GORM repository implementations
│   └── caching/                # Redis caching handlers
├── presentation/
│   ├── handler/                # Gin HTTP handlers
│   └── worker/                 # Dashboard Aggregator Ticker Worker
└── provider.go                 # Wire ProviderSet
```

---

## Phases & Deliverables

### Phase 0: Research & Architecture Alignment (Completed)
- Artifact: `specs/006-dashboard-analytics/research.md`

### Phase 1: Design & Contracts (Completed)
- Artifacts: 
  - `specs/006-dashboard-analytics/data-model.md`
  - `specs/006-dashboard-analytics/contracts/api-schema.md`
  - `specs/006-dashboard-analytics/quickstart.md`

### Phase 2: Implementation Tasks (Next Step via `/speckit-tasks`)
- Run `/speckit-tasks` to generate detailed implementation tasks in `tasks.md`.
