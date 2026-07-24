# Phase 0 Research & Tech Stack Decisions: Dashboard & Analytics System

**Feature**: `006-dashboard-analytics`  
**Date**: 2026-07-23  
**Target Platform**: Go (v1.25+) / Gin Web Framework / GORM / PostgreSQL / Redis / Wire DI

---

## 1. Technical Context & Codebase Alignment

### Current System Architecture
- **Monolith Paradigm**: Pragmatic Modular Monolith in Go (`cmd/server/main.go`).
- **Existing Modules**: `identity`, `brand`, `wardrobe`, `fashion`, `subscription`, `aiexecutor`.
- **Inter-Module Communication**: Via `presentation/contract/` interfaces using DTOs. Cross-module entity references or direct SQL cross-joins are strictly prohibited by the Constitution.
- **Infrastructure**: GORM + PostgreSQL, Redis (`internal/shared/infrastructure/caching`), Wire DI (`internal/di/wire.go`), Goose SQL migrations (`database/migrations/`).
- **Handler Convention**: `func (h *Handler) Method(c *gin.Context) error` wrapped via `shared_pres.WrapHandler`.

---

## 2. Research Questions & Technology Decisions

### R-001: Data Aggregation & Storage Strategy (Phù hợp với Codebase Hiện tại)

- **Decision**: Sử dụng **PostgreSQL Aggregation Tables (Snapshots)** kết hợp **Redis Caching** và **Background Ticker Worker** trong `internal/modules/dashboard/`.
- **Rationale**: 
  - Tránh vẽ quá mức (không dùng Kafka/ClickHouse/Complex Event Sourcing) vì dự án hiện dùng PostgreSQL + Redis + Goose Migrations.
  - Tạo 3 bảng Snapshot bằng Goose Migration (`dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_user_wardrobe_summaries`).
  - Worker chạy định kỳ (Ticker 15-30m) gọi Read Contracts của các domain module để tổng hợp số liệu và ghi vào Snapshot Tables + Redis.
- **Alternatives Considered**:
  - *Direct Query OLTP*: Chạy câu lệnh `COUNT()`, `SUM()` trực tiếp trên các bảng chính (`ai_usage_events`, `loyalty_point_lots`) mỗi lần user load dashboard. $\rightarrow$ **Loại vì gây slow query và nghẽn DB**.
  - *External OLAP (ClickHouse / TimescaleDB)*: $\rightarrow$ **Loại vì làm tăng độ phức tạp hạ tầng không cần thiết ở giai đoạn hiện tại**.

---

### R-002: Domain Boundaries & Inter-Module Contracts

- **Decision**: Mỗi Domain Module (`identity`, `brand`, `wardrobe`, `subscription`, `aiexecutor`) bổ sung một **Read Contract** chuyên biệt tại `presentation/contract/` (ví dụ: `UserDashboardReader`, `BrandDashboardReader`, `AICostDashboardReader`).
- **Rationale**:
  - Tuân thủ tuyệt đối **Constitution Principle I & II** (Strict Module Boundaries & DTO Contracts).
  - Dashboard Module không sở hữu DB tables của domain khác. Chỉ truy xuất dữ liệu tổng hợp dạng DTO qua các Read Contracts này.
- **Alternatives Considered**:
  - *Dashboard query DB trực tiếp*: Vi phạm nghiêm trọng Constitution.

---

### R-003: Dynamic Query Engine & API Granularity

- **Decision**:
  1. Cung cấp API `POST /api/v1/dashboard/analytics/query` tiếp nhận DTO filter động (`metric`, `timeframe`, `granularity`, `group_by`, `compare_with_previous`).
  2. Cung cấp các endpoints Granular Widgets (`/kpis`, `/ai-margin`, `/points-liability`, `/wardrobe-insights`) cho giao diện đơn giản.
  3. Cung cấp Drill-down endpoints (`/drilldown/*`) trả về danh sách bản ghi chi tiết có phân trang.
- **Rationale**: Đáp ứng tiêu chuẩn Backend Dashboard hiện đại, hỗ trợ Progressive UI rendering và cho phép frontend vẽ biểu đồ linh hoạt không cần thay đổi Backend code.

---

### R-004: Multi-Tenancy & Column-Level Security

- **Decision**: 
  1. Tự động kiểm tra `brand_id` lấy từ JWT Claims context cho các API Brand Dashboard.
  2. Tầng Mapper DTO kiểm tra Role: `brandmemberrole.Owner` được xem đầy đủ thông tin tài chính (`cost_vnd`, `margin_percent`), `brandmemberrole.Staff` bị ẩn/null các trường nhạy cảm này.
- **Rationale**: Đảm bảo an toàn thông tin multi-tenant và tuân thủ phân quyền RBAC dự án.

---

## 3. Summary of Decisions & Trade-offs

| Quyết định Kiến trúc | Lựa chọn Thực thi | Lý do & Khả năng Triển khai |
| :--- | :--- | :--- |
| **Storage & Aggregation** | Postgres Snapshot Tables + Redis + Ticker Worker | Thực tế, tận dụng DB & Redis hiện có, không vẽ thêm hạ tầng mới. |
| **Inter-Module Boundary** | Domain Read Contracts DTOs | Tuân thủ 100% Constitution, dễ mở rộng và kiểm thử. |
| **API Contract** | Dynamic Aggregator + Granular Widgets + Drill-down | Đạt chuẩn Dashboard hiện đại, hỗ trợ Progressive UI. |
| **Security** | Multi-tenant JWT Scoping + Role-based DTO Redaction | Đảm bảo an toàn thông tin 100%. |
