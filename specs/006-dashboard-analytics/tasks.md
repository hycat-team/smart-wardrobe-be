# Tasks: Dashboard & Analytics System

**Input**: Design documents from `/specs/006-dashboard-analytics/`  
**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/api-schema.md`, `quickstart.md`

---

## ⚠️ STRICT PHASE EXECUTION RULE (RÀNG BUỘC THỰC HIỆN SECTOR BY SECTOR)

1. **Thực hiện đơn phase**: Mỗi lần chỉ thực hiện duy nhất 1 Phase (không tự ý nhảy phase hoặc làm gộp nhiều phase cùng lúc).
2. **Nghiệm thu & Báo cáo**: Sau khi hoàn thành tất cả các task trong 1 Phase, Agent **MUST STOP** ngay lập tức, xuất báo cáo kết quả kiểm thử (Completion Report) của Phase đó.
3. **Chờ xác nhận**: Chờ người dùng (User) review và gõ lệnh phê duyệt (verify) mới được phép tiếp tục chuyển sang Phase kế tiếp.

---

## Task Format Index

- `[P]`: Task có thể làm song song (khác file, không phụ thuộc task chưa xong).
- `[US1]`, `[US2]`, `[US3]`: Gắn nhãn User Story tương ứng từ `spec.md`.

---

## Phase 1: Database Schema & Goose Migrations (Foundational Data Layer)

**Goal**: Tạo 3 bảng Read Model Snapshots trong PostgreSQL để phục vụ lưu trữ số liệu tổng hợp cho Dashboard mà không đụng tới OLTP tables chính.

**Independent Test**: Chạy `make migration-up` thành công, kiểm tra DB có 3 bảng snapshot với đúng kiểu dữ liệu và indexes.

- [x] T001 Tạo Goose SQL Migration file `database/migrations/20260723000001_create_dashboard_snapshots_tables.sql` bổ sung 3 bảng: `dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_user_wardrobe_summaries` kèm đầy đủ indexes.
- [x] T002 Tạo Go GORM Entity structs cho 3 bảng snapshots tại `internal/modules/dashboard/domain/entities/dashboard_snapshots.go`.
- [x] T003 [P] Tạo Repository Interfaces cho các bảng Snapshots tại `internal/modules/dashboard/domain/repositories/dashboard_repository.go`.
- [x] T004 Triển khai GORM Repositories tại `internal/modules/dashboard/infrastructure/persistence/dashboard_repository_impl.go`.

🛑 **CHECKPOINT PHASE 1**: Dừng lại, báo cáo kết quả migration và chờ User verify trước khi sang Phase 2.

---

## Phase 2: Domain Read Contracts & Data Mappers (Bounded Context Isolation)

**Goal**: Xây dựng các Read Contracts DTO nội bộ tại tầng presentation/contract của các module domain để cung cấp dữ liệu cho Dashboard mà không vi phạm Bounded Context.

**Independent Test**: Unit test từng Read Contract trả về đúng DTO aggregate từ domain database tương ứng.

- [x] T005 [P] Tạo Contract Interface `UserDashboardReader` và DTOs tại `internal/modules/identity/presentation/contract/dashboard_reader.go`.
- [x] T006 [P] Tạo Contract Interface `BrandDashboardReader` và DTOs tại `internal/modules/brand/presentation/contract/dashboard_reader.go`.
- [x] T007 [P] Tạo Contract Interface `WardrobeDashboardReader` và DTOs tại `internal/modules/wardrobe/presentation/contract/dashboard_reader.go`.
- [x] T008 [P] Tạo Contract Interface `AICostDashboardReader` và DTOs tại `internal/modules/aiexecutor/presentation/contract/dashboard_reader.go`.
- [x] T009 Triển khai implementations cho các Dashboard Read Contracts trên tại từng module tương ứng.

🛑 **CHECKPOINT PHASE 2**: Dừng lại, báo cáo kết quả thử nghiệm Read Contracts và chờ User verify trước khi sang Phase 3.

---

## Phase 3: User Story 1 - System Admin Dashboard & AI Margin Analytics (Priority: P1) 🎯 MVP

**Goal**: Triển khai các UseCase và Handlers phục vụ System Admin giám sát người dùng, tải AI, đối soát PayOS và tính Biên lợi nhuận AI Token (`ai_usage_period_ledgers`).

**Independent Test**: Gửi HTTP GET request tới `/api/v1/admin/dashboard/kpi-cards` và `/api/v1/admin/dashboard/ai-margin` kiểm tra response 200 OK với JWT token của Admin.

- [x] T010 [US1] Tạo DTOs cho Admin Dashboard tại `internal/modules/dashboard/application/dto/admin_dashboard_dto.go`.
- [x] T011 [US1] Tạo Interface `IAdminDashboardUseCase` tại `internal/modules/dashboard/application/interface/usecase/admin_dashboard_uc.go`.
- [x] T012 [US1] Triển khai `AdminDashboardUseCase` tại `internal/modules/dashboard/application/usecase/admin_dashboard_uc_impl.go` tổng hợp số liệu từ snapshot tables & Read Contracts.
- [x] T013 [US1] Đăng ký Admin Dashboard UseCase, Handler và ProviderSet vào Wire DI (`internal/di/wire.go`).
- [x] T014 [US1] Triển khai Gin Handler `AdminDashboardHandler` tại `internal/modules/dashboard/presentation/handler/admin_dashboard_handler.go` và đăng ký routes tại `internal/api/routes/admin/router.go`.

🛑 **CHECKPOINT PHASE 3**: Dừng lại, báo cáo kết quả test Admin Dashboard và chờ User verify trước khi sang Phase 4.

---

## Phase 4: User Story 2 - Brand Manager Dashboard & Points Liability (Priority: P1)

**Goal**: Triển khai UseCase và Handlers cho B2B Brand Portal: Thống kê Loyalty, Dự báo điểm hết hạn 30 ngày (`loyalty_point_lots`), Digital Sample Lab style context, CS SLA và phân quyền Column-level RBAC (`Owner` vs `Staff`).

**Independent Test**: Gửi HTTP GET request tới `/api/v1/brand/dashboard/kpi-cards` (Staff & Owner) và `/api/v1/brand/dashboard/financials/points-liability` (Owner OK, Staff 403 Forbidden).

- [x] T015 [US2] Tạo DTOs cho Brand Dashboard tại `internal/modules/dashboard/application/dto/brand_dashboard_dto.go`.
- [x] T016 [US2] Tạo Mapper `BrandDashboardMapper` với logic Redaction theo Role tại `internal/modules/dashboard/application/mapper/brand_mapper.go`.
- [x] T017 [US2] Tạo Interface & Implementation `BrandDashboardUseCase` tại `internal/modules/dashboard/application/usecase/brand_dashboard_uc_impl.go`.
- [x] T018 [US2] Triển khai Gin Handler `BrandDashboardHandler` tại `internal/modules/dashboard/presentation/handler/brand_dashboard_handler.go` và đăng ký routes tại `internal/api/routes/brand/router.go`.

🛑 **CHECKPOINT PHASE 4**: Dừng lại, báo cáo kết quả test Brand Portal Dashboard và chờ User verify trước khi sang Phase 5.

---

## Phase 5: User Story 3 - End User Smart Wardrobe Utility & CPW (Priority: P2)

**Goal**: Triển khai UseCase và Handlers cho B2C End User: Thống kê tủ đồ cá nhân, giá trị tài chính, tính chỉ số Cost Per Wear ($CPW = \text{purchase\_price} / \text{wear\_count}$) và danh sách đồ lãng phí.

**Independent Test**: Gửi HTTP GET request tới `/api/v1/user/dashboard/wardrobe-insights` với JWT User kiểm tra phản hồi chỉ số CPW và danh sách item chưa mặc > 30 ngày.

- [x] T019 [US3] Tạo DTOs cho User Wardrobe Insights tại `internal/modules/dashboard/application/dto/user_dashboard_dto.go`.
- [x] T020 [US3] Tạo Interface `IUserDashboardUseCase` tại `internal/modules/dashboard/application/interface/usecase/user_dashboard_uc.go`.
- [x] T021 [US3] Triển khai `UserDashboardUseCase` tại `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go`, Gin Handler `UserDashboardHandler` tại `internal/modules/dashboard/presentation/handler/user_dashboard_handler.go` và đăng ký routes tại `internal/api/routes/wardrobe/router.go`.

🛑 **CHECKPOINT PHASE 5**: Dừng lại, báo cáo kết quả test User Wardrobe Dashboard và chờ User verify trước khi sang Phase 6.

---

## Phase 6: Dynamic Query Engine & Drill-Down API Composition

**Goal**: Triển khai endpoint linh hoạt `POST /api/v1/dashboard/analytics/query` hỗ trợ tham số động (`metric`, `timeframe`, `granularity`, `group_by`, `compare_with_previous`) và các API Drill-down xem chi tiết log lỗi / feedback.

**Independent Test**: Gửi HTTP POST request tới `/api/v1/dashboard/analytics/query` với bộ lọc đa chiều và kiểm tra dữ liệu Time-Series trả về dưới 150ms.

- [x] T022 [P] Tạo Dynamic Query DTOs & Request Builder tại `internal/modules/dashboard/application/dto/dynamic_query_dto.go`.
- [x] T023 Triển khai Dynamic Aggregator Query Engine trong `internal/modules/dashboard/application/usecase/dynamic_query_uc.go`.
- [x] T024 Triển khai Drill-Down Traceability UseCases cho Admin & Brand tại `internal/modules/dashboard/application/usecase/drilldown_uc.go`.
- [x] T025 Triển khai Handlers cho Dynamic Query & Drill-down tại `internal/modules/dashboard/presentation/handler/analytics_handler.go`.

🛑 **CHECKPOINT PHASE 6**: Dừng lại, báo cáo kết quả Dynamic Query Engine & Drill-down và chờ User verify trước khi sang Phase 7.

---

## Phase 7: Aggregation Worker, Wire DI & Route Integration (Production Readiness)

**Goal**: Thiết lập Ticker Worker tự động snapshot dữ liệu định kỳ 15-30m, tích hợp Wire DI Provider, đăng ký HTTP Routes và hoàn thiện tài liệu Swagger.

**Independent Test**: Chạy `make dev`, kiểm tra Worker tự động log snapshot dữ liệu, tất cả API phản hồi 200 OK trên Swagger UI (`/swagger/index.html`).

- [ ] T026 Triển khai Aggregation Ticker Worker tại `internal/modules/dashboard/presentation/worker/dashboard_aggregator_worker.go`.
- [ ] T027 Cấu hình Wire ProviderSet tại `internal/modules/dashboard/provider.go` và đăng ký wire injector trong `internal/di/wire.go` (`make wire`).
- [ ] T028 Đăng ký Route Group `/api/v1/admin/dashboard`, `/api/v1/brand/dashboard`, `/api/v1/user/dashboard`, `/api/v1/dashboard/analytics` tại `internal/api/routes/dashboard_routes.go`.
- [ ] T029 Cập nhật Swagger Annotations cho tất cả Handlers và chạy `make swag` tạo OpenAPI spec.

🛑 **CHECKPOINT PHASE 7**: TỔNG HOÀN THÀNH. Báo cáo nghiệm thu toàn bộ tính năng Dashboard & Analytics System.

---

## Dependencies & Execution Strategy

```
Phase 1 (DB Snapshots)
       │
       ▼
Phase 2 (Domain Read Contracts)
       │
       ▼
Phase 3 (US1: Admin Dashboard & AI Margin)  ◄── MVP Checkpoint
       │
       ▼
Phase 4 (US2: Brand Portal & Liability)
       │
       ▼
Phase 5 (US3: User Smart Wardrobe CPW)
       │
       ▼
Phase 6 (Dynamic Query & Drill-down)
       │
       ▼
Phase 7 (Aggregator Worker & Wire DI)
```
