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

- [x] T026 Triển khai Aggregation UseCase & Ticker Worker chuẩn Clean Architecture.
  - **Application Layer** (`internal/modules/dashboard/application/`):
    - Interface `IDashboardAggregatorUseCase` tại `application/interface/usecase/aggregator_uc.go`.
    - Implementation `DashboardAggregatorUseCase` tại `application/usecase/aggregator_uc_impl.go`.
    - **Hàm UseCase**:
      - `AggregateAdminSnapshots(ctx context.Context) error`: Thu thập số liệu từ Reader Contracts (`userReader`, `brandReader`, `aiCostReader`) và lưu DB qua `dashboardRepo.UpsertAdminSnapshot`.
      - `AggregateBrandSnapshots(ctx context.Context) error`: Duyệt qua các nhãn hàng active, thu thập chỉ số và lưu DB qua `dashboardRepo.UpsertBrandSnapshot`.
      - `AggregateUserWardrobeSummaries(ctx context.Context) error`: Thu thập tổng quan tủ đồ cá nhân từ `wardrobeReader` và lưu DB qua `dashboardRepo.UpsertUserWardrobeSummary`.
  - **Presentation Layer** (`internal/modules/dashboard/presentation/worker/`):
    - Struct `DashboardAggregatorWorker` tại `presentation/worker/dashboard_aggregator_worker.go`.
    - **Hàm Worker**:
      - `NewDashboardAggregatorWorker(logger logger.Interface, aggregatorUC uc_interfaces.IDashboardAggregatorUseCase) *DashboardAggregatorWorker`
      - `(w *DashboardAggregatorWorker) Start(ctx context.Context)`: Khởi chạy `time.NewTicker(15 * time.Minute)` trong goroutine riêng biệt và gọi `w.aggregatorUC`.
      - `(w *DashboardAggregatorWorker) Stop()`: Hủy context và đóng ticker khi ứng dụng graceful shutdown.
- [x] T027 Cấu hình Wire ProviderSet tại `internal/modules/dashboard/provider.go` và đăng ký wire injector trong `internal/di/wire.go`.
  - **Layer**: Infrastructure DI Layer
  - **Hàm/Set cần triển khai**:
    - Khai báo `DashboardProviderSet`: Đăng ký `NewDashboardRepository`, `NewDashboardAggregatorUseCase`, `NewAdminDashboardUseCase`, `NewBrandDashboardUseCase`, `NewUserDashboardUseCase`, `NewDynamicQueryUseCase`, `NewDrillDownUseCase`, `NewAdminDashboardHandler`, `NewBrandDashboardHandler`, `NewUserDashboardHandler`, `NewAnalyticsHandler`, `NewDashboardAggregatorWorker`.
    - Đăng ký `dashboard.DashboardProviderSet` vào `wire.Build(...)` tại `internal/di/wire.go`.
    - Chạy command `make wire` sinh mã DI tự động tại `internal/di/wire_gen.go`.
- [x] T028 Đăng ký Route Groups cho tất cả 11 Dashboard APIs tại `internal/api/routes/dashboard_routes.go`.
  - **Layer**: Presentation Router Layer
  - **Hàm cần triển khai**:
    - `RegisterDashboardRoutes(r *gin.RouterGroup, adminHandler *handler.AdminDashboardHandler, brandHandler *handler.BrandDashboardHandler, userHandler *handler.UserDashboardHandler, analyticsHandler *handler.AnalyticsHandler, authMiddleware gin.HandlerFunc, rbacMiddleware func(roles ...string) gin.HandlerFunc)`
    - Đăng ký routes:
      - Admin Group: `/api/v1/admin/dashboard/kpi-cards` $\rightarrow$ `adminHandler.GetKPICards`, `/api/v1/admin/dashboard/ai-margin` $\rightarrow$ `adminHandler.GetAIMargin`, `/api/v1/admin/dashboard/payos-reconciliation` $\rightarrow$ `adminHandler.GetPayOSReconciliation`, `/api/v1/admin/dashboard/overview` $\rightarrow$ `adminHandler.GetOverview`, `/api/v1/admin/dashboard/drilldown/ai-events` $\rightarrow$ `analyticsHandler.GetAIEventsDrillDown`
      - Brand Group: `/api/v1/brand/dashboard/kpi-cards` $\rightarrow$ `brandHandler.GetKPICards`, `/api/v1/brand/dashboard/financials/points-liability` $\rightarrow$ `brandHandler.GetPointsLiability`, `/api/v1/brand/dashboard/operations/digital-sample-lab/:itemId` $\rightarrow$ `brandHandler.GetDigitalSampleLabAnalytics`, `/api/v1/brand/dashboard/drilldown/sample-lab-feedbacks` $\rightarrow$ `analyticsHandler.GetSampleLabFeedbacksDrillDown`
      - User Group: `/api/v1/user/dashboard/wardrobe-insights` $\rightarrow$ `userHandler.GetWardrobeInsights`
      - Analytics Group: `/api/v1/dashboard/analytics/query` $\rightarrow$ `analyticsHandler.ExecuteDynamicQuery`
- [x] T029 Cập nhật Swagger Annotations cho tất cả Handlers và sinh Swagger Spec.
  - **Layer**: Presentation Documentation Layer
  - **Thao tác & Command**:
    - Rà soát các `@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router` annotations trên 11 handler methods trong `internal/modules/dashboard/presentation/handler/`.
    - Chạy command `make swag` tạo tài liệu OpenAPI tại `docs/swagger/`.

🛑 **CHECKPOINT PHASE 7**: TỔNG HOÀN THÀNH. Báo cáo nghiệm thu toàn bộ tính năng Dashboard & Analytics System.

---

## Phase 8: Convergence — Bug Fixes, Missing Fields & Edge Cases

**Goal**: Sửa các khoảng trống còn lại sau khi review toàn bộ codebase: fix bug `compare_with_previous`, bổ sung missing fields trong DTO response, xử lý edge case large date range, và thêm unit tests cho dashboard module.

**Independent Test**: Chạy `go build ./...` thành công, gọi từng API endpoint kiểm tra response chứa đủ các field theo spec, gọi Dynamic Query với `compare_with_previous=true` kiểm tra delta % chính xác, chạy `go test ./internal/modules/dashboard/...` pass.

---

- [x] T030 [FIX] Sửa `compare_with_previous` trong `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go` để query dữ liệu kỳ trước từ DB thay vì hardcode `prevVal = 0.0`.
  - **Hướng giải quyết**:
    1. Tính window kỳ trước: `prevFromDate`, `prevToDate` (đã có sẵn).
    2. Gọi `uc.dashboardRepo.GetAdminSnapshots` (hoặc `GetBrandSnapshots` tuỳ `target`) với khoảng kỳ trước.
    3. Sum các metric từ snapshots kỳ trước → `prevVal` thực tế.
    4. Tính delta % = `((currVal - prevVal) / prevVal) * 100`, xử lý division by zero.
  - **Source**: `spec.md:97` (FR-002), `plan.md:17` (Dynamic Aggregation Engine), `tasks.md:T023`.
  - **Test**: Gửi `POST /api/v1/dashboard/analytics/query` với `compareWithPrevious=true`, kiểm tra `comparison.previousPeriodFrom`/`previousPeriodTo` khớp và `metricDeltaPercentages` ra số hợp lý.
  - **Gap type**: `partial`

- [x] T031 [US3] Bổ sung field `aiQuotaRemaining` và `expiringVouchersCount` vào `UserWardrobeInsightsRes` DTO tại `internal/modules/dashboard/application/dto/user_dashboard_dto.go` và populate chúng trong `GetWardrobeInsights` tại `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Thêm 2 field vào struct `UserWardrobeInsightsRes` với JSON tags `aiQuotaRemaining` và `expiringVouchersCount`.
    2. Trong `GetWardrobeInsights`, khi cache hit từ snapshot: gán `summary.AIQuotaRemaining` và `summary.ExpiringVouchersCount`.
    3. Khi cache miss (on-demand compute): query thêm AI quota còn lại từ Contract `subscription` hoặc `aiexecutor`, query voucher sắp hết hạn từ `brand` reader → gán vào response và lưu vào snapshot.
  - **Source**: `spec.md:78` (US3/AC1: "AI Stylist Quota & Loyalty Alert"), `data-model.md`: entity `dashboard_user_wardrobe_summaries` đã có 2 cột này.
  - **Test**: Gọi `GET /api/v1/user/dashboard/wardrobe-insights` kiểm tra response có chứa `aiQuotaRemaining` và `expiringVouchersCount` khác `null`.
  - **Gap type**: `partial`

- [x] T032 [US2] Bổ sung CS SLA stats (`openCsTickets`, `csFirstResponseAvgSeconds`) và Digital Sample stats (`digitalSampleVotesCount`, `digitalSampleAvgRating`) vào `BrandKPICardsRes` DTO tại `internal/modules/dashboard/application/dto/brand_dashboard_dto.go` và populate trong `GetBrandKPICards` tại `internal/modules/dashboard/application/usecase/brand_dashboard_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Thêm 4 field vào `BrandKPICardsRes`.
    2. Trong `GetBrandKPICards`, gán từ `latest.DigitalSampleVotesCount`, `latest.DigitalSampleAvgRating`, `latest.OpenCSTickets`, `latest.CSFirstResponseAvgSeconds` (snapshot entity đã có sẵn).
    3. Fallback từ `stats` (live reader) cũng gán tương tự.
  - **Source**: `spec.md:61` (US2/AC1: "CS SLA Analytics", "Digital Sample Lab Style Context").
  - **Test**: Gọi `GET /api/v1/brand/dashboard/kpi-cards` kiểm tra response có 4 field mới.
  - **Gap type**: `partial`

- [x] T033 [FR-002] Triển khai xử lý edge case large date range trong `ExecuteDynamicQuery` tại `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Sau khi parse `fromDate`, `toDate`, tính số ngày chênh lệch: `days := toDate.Sub(fromDate).Hours() / 24`.
    2. Nếu `days > 365 && gran == granularity.Hour`: tự động set `gran = granularity.Day` (hoặc `Month`).
    3. Nếu `days > 730 && (gran == granularity.Hour || gran == granularity.Day)`: set `gran = granularity.Month`.
    4. Log warning khi granularity bị auto-downgrade.
  - **Source**: `spec.md:84` (Edge Cases: "Dynamic Query Abuse / Large Date Range").
  - **Test**: Gửi POST request với date range 400 ngày + `granularity=hour`, kiểm tra response `granularity` trả về `day` hoặc `month`.
  - **Gap type**: `missing`

- [x] T034 Viết unit tests cho dashboard module tại `internal/modules/dashboard/`.
  - **Hướng giải quyết**:
    1. Tạo file `internal/modules/dashboard/application/usecase/admin_dashboard_uc_test.go`: test `GetAdminKPICards` (mock repository + readers), test `GetAIMarginAnalytics` với snapshot có sẵn và fallback live reader.
    2. Tạo file `internal/modules/dashboard/application/usecase/brand_dashboard_uc_test.go`: test `GetBrandKPICards`, `GetPointsLiability` (kiểm tra redaction Staff vs Owner).
    3. Tạo file `internal/modules/dashboard/application/usecase/user_dashboard_uc_test.go`: test `GetWardrobeInsights` với cache hit (snapshot <1h) và cache miss.
    4. Tạo file `internal/modules/dashboard/application/usecase/dynamic_query_uc_test.go`: test `ExecuteDynamicQuery` với các target khác nhau, test `compareWithPrevious=true`.
    5. Tạo file `internal/modules/dashboard/application/usecase/drilldown_uc_test.go`: test `GetAIEventsDrillDown`, `GetSampleLabFeedbacksDrillDown`.
    6. Sử dụng `golang/mock` hoặc interface mock thủ công cho Repository và Reader interfaces.
    7. Chạy `go test ./internal/modules/dashboard/... -v -count=1` để verify.
  - **Source**: `plan.md:28` (Testing: Go standard `testing` package), thiếu test coverage cho toàn bộ module dashboard.
  - **Test**: `go test ./internal/modules/dashboard/... -v -count=1` pass ít nhất 80% coverage.
  - **Gap type**: `missing`

**Lưu ý**: Sau khi hoàn thành Phase 8, chạy `go build ./...` và `go test ./internal/modules/dashboard/...` để nghiệm thu. Sau đó chạy `/speckit-converge` để kiểm tra lại không còn gap.

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
       │
       ▼
Phase 8 (Convergence — Bug Fixes & Tests)
       │
       ▼
Phase 9 (Convergence — Zero Values, Boundaries & Hardcoded Values)
```

---

## Phase 9: Convergence — Zero Values, Boundaries & Hardcoded Values

**Goal**: Xử lý triệt để các vấn đề zero value, error handling sai, hardcoded magic numbers, và boundary issues trong toàn bộ dashboard module.

**Independent Test**: `go build ./...` + `go test ./internal/modules/dashboard/... -count=1` pass, đồng thời review code từng task đảm bảo không còn zero value sai ngữ cảnh.

---

### 🔴 CRITICAL — Zero Values & Core Logic

- [x] T035 [FIX] Populate `SubscriptionRevenueVND`, `PayOSTransactionCount`, `PayOSSuccessCount` trong `AggregateAdminSnapshots` tại `internal/modules/dashboard/application/usecase/aggregator_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Thêm dependency `subscriptionContract` hoặc reader mới để đọc doanh thu subscription và thống kê PayOS.
    2. Trong `AggregateAdminSnapshots`, query dữ liệu revenue từ reader và gán vào `snapshot.SubscriptionRevenueVND`, `snapshot.PayOSTransactionCount`, `snapshot.PayOSSuccessCount`.
    3. Cập nhật `ProviderSet` trong `provider.go` và `wire_gen.go` nếu cần dependency mới.
  - **Source**: `aggregator_uc_impl.go:49-69` — cả 3 field luôn = 0.
  - **Gap type**: `missing`

- [x] T036 [FIX] Xử lý reader errors trong `AggregateAdminSnapshots` và `AggregateBrandSnapshots` tại `aggregator_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Thay vì `userStats, _ := uc.userReader.GetAdminUserStats(...)`, kiểm tra `err != nil` → log error + `continue` hoặc return error (không ghi snapshot với zero values).
    2. Áp dụng tương tự cho `brandReader` và `aiCostReader`.
    3. Thêm `context.WithTimeout` cho mỗi reader call để tránh goroutine leak.
  - **Source**: `aggregator_uc_impl.go:45-47` — errors silently discarded.
  - **Gap type**: `partial`

- [x] T037 [FIX] Trả error thay vì zero values khi fallback reader trả về nil trong `GetDigitalSampleLabAnalytics` tại `internal/modules/dashboard/application/usecase/brand_dashboard_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Kiểm tra `stats == nil` sau fallback reader call (dòng 136-142): nếu nil → return `ErrBrandAnalyticsNotFound()`.
    2. Không cho phép response với votes=0, rating=0.0, sentiment="positive" mặc định.
  - **Source**: `brand_dashboard_uc_impl.go:122-151` — zero values trên fallback failure.
  - **Gap type**: `partial`

- [x] T038 [FIX] Preserve `AIQuotaRemaining` và `ExpiringVouchersCount` từ stale snapshot khi cache miss trong `GetWardrobeInsights` tại `internal/modules/dashboard/application/usecase/user_dashboard_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Kiểm tra: nếu `summary != nil && time.Since(summary.UpdatedAt) >= 1*time.Hour` (stale), vẫn giữ `summary.AIQuotaRemaining` và `summary.ExpiringVouchersCount`.
    2. Không set `aiQuota = 0` và `expiringVouchers = 0` khi `summary == nil` — thay vào đó query từ subscription/brand contracts nếu có thể.
    3. Viết vào write-back snapshot với giá trị đã giữ lại.
  - **Source**: `user_dashboard_uc_impl.go:103-104,117-118` — zero value corrupts cached data.
  - **Gap type**: `partial`

- [x] T039 [FIX] Phân biệt system error (DB/network) vs authorization error trong `ExecuteDynamicQuery` tại `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go`.
  - **Hướng giải quyết**:
    1. Tách `err` và `role == ""` thành 2 nhánh riêng:
       - `err != nil` → return `apperror.NewInternalServerError(...)` (HTTP 500).
       - `role == ""` → return `apperror.NewForbidden(...)` (HTTP 403).
    2. Sửa tương tự cho các chỗ khác có pattern `if err != nil || condition`.
  - **Source**: `dynamic_query_uc_impl.go:59-62` — DB error trả về 403.
  - **Gap type**: `contradicts`

---

### 🟡 MEDIUM — Error Handling & Boundary

- [x] T040 [FIX] Propagate JSON marshal error và DB upsert error thay vì silently discard trong `GetWardrobeInsights` tại `user_dashboard_uc_impl.go`.
  - **Hướng giải quyết**: Kiểm tra lỗi từ `json.Marshal` và `uc.dashboardRepo.UpsertUserWardrobeSummary`, log error nếu không thể return (non-critical). Ít nhất không discard hoàn toàn.
  - **Source**: `user_dashboard_uc_impl.go:97,121`.
  - **Gap type**: `partial`

- [x] T041 [FIX] Định nghĩa typed constant cho transaction context key `"TxKey"` tại `internal/modules/dashboard/infrastructure/persistence/dashboard_repository_impl.go`.
  - **Hướng giải quyết**: Tạo `type contextKey string` constant `TxKey contextKey = "dashboard_tx"` trong package persistence và sử dụng thay cho raw string.
  - **Source**: `dashboard_repository_impl.go:23`.
  - **Gap type**: `partial`

- [x] T042 [FIX] Sửa GORM default value `'POSITIVE'` thành `'positive'` trong entity `DashboardBrandItemDailySnapshot.FeedbackSentiment` tại `internal/modules/dashboard/domain/entities/dashboard_snapshots.go` để khớp với SQL migration và Go constant.
  - **Hướng giải quyết**: Đổi `default:'POSITIVE'` → `default:'positive'` trong GORM tag.
  - **Source**: `dashboard_snapshots.go:62` — mismatch giữa GORM default, SQL default, và Go constant.
  - **Gap type**: `contradicts`

- [x] T043 [FIX] Thêm upper bound cho pagination limit (VD: max 100) trong `Normalize()` tại `internal/shared/application/dto/pagination.go`.
  - **Hướng giải quyết**: Thêm `if q.Limit > 100 { q.Limit = 100 }` sau normalize.
  - **Source**: `shared/dto/pagination.go:10-16` — không bound, risk memory exhaustion.
  - **Gap type**: `missing`

- [x] T044 [FIX] Phân biệt "zero transactions" vs "no data" trong `GetPayOSReconciliation` tại `admin_dashboard_uc_impl.go`.
  - **Hướng giải quyết**: Nếu `latest != nil && latest.PayOSTransactionCount == 0`, trả về response hợp lệ với các field = 0 thay vì `ErrDashboardDataNotFound()`.
  - **Source**: `admin_dashboard_uc_impl.go:171` — zero transaction bị error.
  - **Gap type**: `partial`

- [x] T045 [FIX] Xử lý nil rating/votes khi không có snapshot và stats cũng nil trong `GetDigitalSampleLabAnalytics` tại `brand_dashboard_uc_impl.go`.
  - **Hướng giải quyết**: Trả về error rõ ràng thay vì response với `AvgRating: 0.0` không phân biệt được "no data".
  - **Source**: `brand_dashboard_uc_impl.go:123,148` — 0.0 vs "no data".
  - **Gap type**: `partial`

- [x] T046 [FIX] Thêm context timeout cho các reader calls trong `AggregateAdminSnapshots` và `AggregateBrandSnapshots` tại `aggregator_uc_impl.go`.
  - **Hướng giải quyết**: Sử dụng `context.WithTimeout(ctx, 30*time.Second)` cho mỗi reader call để tránh worker goroutine bị block vô hạn.
  - **Source**: `aggregator_uc_impl.go:45-47` — risk goroutine leak.
  - **Gap type**: `partial`

- [x] T047 [FIX] Sửa false negative detection khi `AIRequestsCount == 0 && AIActualCostMicroVND == 0` trong `GetAIMarginAnalytics` tại `admin_dashboard_uc_impl.go`.
  - **Hướng giải quyết**: Chỉ treat là "no data" khi `aiStats == nil`, không check giá trị zero nữa.
  - **Source**: `admin_dashboard_uc_impl.go:107-108` — zero requests + zero cost bị error.
  - **Gap type**: `partial`

---

### 🟢 LOW — Magic Numbers, Hardcoded Configs & Consistency

- [x] T048 [FIX] Extract magic number `1000000.0` (micro VND conversion) thành named constant tại `admin_dashboard_uc_impl.go` và `aggregator_uc_impl.go`.
  - **Source**: 3 occurrences (admin_dashboard_uc_impl.go:74,115; aggregator_uc_impl.go:69).
  - **Gap type**: `partial`

- [x] T049 [FIX] Extract hardcoded config values thành config struct hoặc constants:
  - Cache TTL `1*time.Hour` tại `user_dashboard_uc_impl.go:40`.
  - Aggregation interval `15*time.Minute` tại `dashboard_aggregator_worker.go:32`.
  - Default lookback `30 ngày` tại `admin_dashboard_uc_impl.go:94`.
  - Underutilized items limit `10` tại `user_dashboard_uc_impl.go:62`.
  - Granularity boundaries `365`, `730` tại `dynamic_query_uc_impl.go:286,289`.
  - **Source**: Nhiều file.
  - **Gap type**: `partial`

- [x] T050 [FIX] Chuẩn hóa JSON acronym casing trong `AdminKPICardsRes` tại `internal/modules/dashboard/application/dto/admin_dashboard_dto.go`.
  - **Hướng giải quyết**: Thống nhất convention: `"activeUsersDau"` → `"activeUsersDau"` (giữ nguyên), `"payosSuccessCount"` → `"payOSSuccessCount"`, `"netAiMarginVnd"` → `"netAIMarginVND"`.
  - **Source**: `admin_dashboard_dto.go:11-18`.
  - **Gap type**: `partial`

- [x] T051 [FIX] Thêm `CreatedAt` field vào `DashboardUserWardrobeSummary` entity và migration để đồng bộ với các sibling entities.
  - **Hướng giải quyết**: Thêm `CreatedAt time.Time \`gorm:"type:timestamp with time zone;not null;default:now()"\`` vào struct, và ALTER TABLE migration.
  - **Source**: `dashboard_snapshots.go:70-84`.
  - **Gap type**: `partial`

- [x] T052 [FIX] Return validation error cho invalid date format thay vì silent fallback trong `ExecuteDynamicQuery` tại `dynamic_query_uc_impl.go` và handler.
  - **Hướng giải quyết**: Khi `time.Parse` fail, return `apperror.NewBadRequest("Định dạng ngày không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")`.
  - **Source**: `dynamic_query_uc_impl.go:69-77`, `admin_dashboard_handler.go:55-60`.
  - **Gap type**: `partial`

- [x] T053 [FIX] Thêm max date range guard trong `ExecuteDynamicQuery` để tránh query quá nhiều snapshots.
  - **Hướng giải quyết**: Nếu `days > 730`, tự động giới hạn `fromDate = toDate.AddDate(0, 0, -730)` và log warning.
  - **Source**: `dynamic_query_uc_impl.go:281`.
  - **Gap type**: `partial`

- [x] T054 [FIX] Đổi `int` thành `int64` cho accumulators `totalRequests` và `totalErrors` trong `GetAIMarginAnalytics` tại `admin_dashboard_uc_impl.go`.
  - **Hướng giải quyết**: Đổi kiểu khai báo từ `int` → `int64` để an toàn trên 32-bit.
  - **Source**: `admin_dashboard_uc_impl.go:134-135`.
  - **Gap type**: `partial`

- [x] T055 [FIX] Xoá redundant `req.ItemID == ""` check sau `uuid.Parse` trong `GetSampleLabFeedbacksDrillDown` tại `drilldown_uc_impl.go`.
  - **Hướng giải quyết**: Chỉ giữ `if err != nil { return ... }` vì uuid.Parse đã validate string.
  - **Source**: `drilldown_uc_impl.go:66`.
  - **Gap type**: `partial`

