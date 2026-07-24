# Feature Specification: Modern Dashboard & Analytics System

**Feature Branch**: `006-dashboard-analytics`

**Created**: 2026-07-23

**Status**: Draft (Enterprise Architecture Spec - Post Review)

**Input**: User description: "Triển khai Dashboard & Analytics chuẩn Backend SaaS Hiện Đại (System Admin, B2B Brand Manager, B2C End User) cho nền tảng Closy (Smart Wardrobe)"

---

## Executive Summary

Hệ thống Dashboard & Analytics phục vụ 3 nhóm đối tượng người dùng chính trên nền tảng B2B2C Closy:
1. **System Admin Dashboard**: Giám sát sức khỏe hệ thống, tăng trưởng B2C, hoạt động B2B, tải AI Engine, đối soát nạp tiền, **Biên lợi nhuận tài chính AI (AI Margin Analytics từ `ai_usage_period_ledgers`)**, và **Traceability Drill-Down vào từng event lỗi**.
2. **Brand Manager Dashboard (B2B Portal)**: Cung cấp insight kinh doanh cho Nhãn hiệu bao gồm hiệu quả chiến dịch (Campaign/Voucher), **Dự báo nợ điểm sắp hết hạn (Points Liability Forecast 30 ngày)**, báo cáo thị trường từ Digital Sample Lab (Vote, Feedback, Style Context), và **Báo cáo So sánh chu kỳ (MoM, WoW Comparison)**.
3. **End User Personal Overview (B2C Mobile/Web)**: Thống kê cá nhân hóa về tủ đồ số (**Giá trị tủ đồ & Cost Per Wear - CPW**), đồ ít mặc, hạn mức gọi AI Stylist còn lại, tổng hợp điểm thưởng Loyalty và Voucher sắp hết hạn từ các Nhãn hiệu.

Về mặt kiến trúc, hệ thống đáp ứng **100% Tiêu chuẩn Backend Dashboard SaaS Hiện Đại**:
- **CQRS Event-Driven Outbox & Read Model Projections**: Các sự kiện nghiệp vụ (Domain Events) được ghi nhận qua Transactional Outbox, được tiêu thụ bất đồng bộ bởi Aggregator Worker để đổ vào các bảng **Time-Series Read Models** (Postgres Aggregates / Redis Cache).
- **Universal Dynamic Aggregation Engine (API query linh hoạt)**: Hỗ trợ Filter, Group-by, Date-range, Granularity, và Compare-period động qua 1 Contract chuẩn化 mà không cần viết API mới cho từng Widget UI.
- **Drill-down Capability**: Cho phép click từ con số tổng quan (Aggregate KPI) xuống danh sách bản ghi chi tiết (Detail Records) có phân trang.
- **Multi-Tenant Data Scoping Interceptor & Column-Level RBAC**: Tự động inject Tenant ID vào Query Context và tự động ẩn (redact) các trường tài chính nhạy cảm theo role (`Owner` vs `Staff`).
- **Sẵn sàng tiến hóa Microservices**: Độc lập 100% về mặt dữ liệu, sẵn sàng tách thành Analytics Microservice chuyên biệt mà không cần refactor các service nghiệp vụ hiện tại.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - System Admin Monitoring, AI Margin & Drill-down Traceability (Priority: P1)

Là một System Admin, tôi muốn xem bức tranh toàn cảnh về hoạt động của hệ thống (User Growth, AI Cost Margin, Financial Reconciliation) đồng thời có khả năng **Drill-down (truy vết chi tiết)** vào từng bản ghi sự kiện khi phát hiện bất thường (như AI call lỗi tăng đột biến hoặc giao dịch nạp tiền thất bại).

**Why this priority**: Admin vừa cần số liệu tổng quan để điều hành, vừa cần năng lực truy vết nguyên nhân sự cố (Root Cause Analysis) ngay trên Dashboard mà không phải mở DB log thủ công.

**Independent Test**: Test bằng cách gọi API query tổng quan `POST /api/v1/admin/dashboard/analytics/query` (lấy KPI AI Error rate), sau đó click drill-down `GET /api/v1/admin/dashboard/drilldown/ai-events?status=FAILED` kiểm tra danh sách bản ghi lỗi trả về đúng trace ID.

**Acceptance Scenarios**:
1. **Given** Admin truy cập Dashboard Admin, **When** hệ thống tải dữ liệu, **Then** hiển thị:
   - **B2C User Growth & Activity**: DAU/MAU, User mới, tỷ lệ giữ chân (Retention rate).
   - **AI Cost & Margin Analytics**: Tổng chi phí AI thực tế (`actual_cost_micro_vnd` từ `ai_usage_period_ledgers`) so với Doanh thu gói Premium B2C (`subscription_plans` / `deposit_transactions`) để xác định biên lợi nhuận ròng.
   - **Comparative Analytics**: So sánh tăng trưởng và chi phí so với chu kỳ trước (WoW, MoM % Delta Change).
   - **Financial Reconciliation & Gateway Health**: Tỷ lệ giao dịch PayOS thành công/thất bại và danh sách webhook inbox bị nghẽn retry.
2. **Given** Admin thấy số lượng AI Error tăng đột biến trên biểu đồ, **When** click vào điểm lỗi trên biểu đồ (Drill-down), **Then** hệ thống chuyển hướng hiển thị bảng danh sách các `ai_usage_events` bị lỗi tương ứng với đầy đủ `error_code`, `provider`, `model` và timestamp.

---

### User Story 2 - Brand Manager Comparative Insights & Points Liability (Priority: P1)

Là một Brand Manager, tôi muốn xem báo cáo so sánh tăng trưởng qua các chu kỳ (WoW, MoM), dự báo nợ điểm Loyalty sắp hết hạn (30 ngày), phân tích phản hồi Digital Sample Lab và phân quyền xem dữ liệu theo chức vụ (`Owner` được xem doanh thu, `Staff` chỉ được xem vận hành).

**Why this priority**: Brand cần các công cụ phân tích nâng cao (Analytics capabilities) để đưa ra quyết định tiếp thị và quản trị rủi ro tài chính của chương trình tích điểm.

**Independent Test**: Đăng nhập bằng tài khoản Staff, kiểm tra API xem các trường doanh thu/chi phí có bị tự động ẩn (redact) hay không; đăng nhập bằng Owner để xem đầy đủ biểu đồ so sánh doanh thu và nợ điểm.

**Acceptance Scenarios**:
1. **Given** Brand Owner vào Brand Portal, **When** xem trang Dashboard, **Then** hệ thống hiển thị:
   - **Points Expiry & Liability Forecast**: Tổng số điểm thưởng và giá trị tiền quy đổi sắp **hết hạn trong 30 ngày tới** (từ `loyalty_point_lots`) kèm so sánh với tháng trước.
   - **Digital Sample Lab Style Context**: Tỷ lệ vote/rating của mẫu thử 3D kèm biểu đồ phân tích các món đồ thời trang thường được phối cùng.
   - **CS SLA Analytics**: Thời gian phản hồi tin nhắn đầu tiên (First Response Time) và tỷ lệ giải quyết cuộc hội thoại.
2. **Given** Brand Staff vào xem Dashboard, **When** API trả về dữ liệu, **Then** các widget chỉ số tài chính (Doanh thu voucher, Chi phí đổi điểm) bị ẩn hoặc trả về `null` nhờ cơ chế **Column-level RBAC**.

---

### User Story 3 - End User Smart Wardrobe Utility & Cost Per Wear (Priority: P2)

Là một Người dùng B2C, tôi muốn xem các thống kê thông minh về tủ đồ số của mình: Tổng giá trị tủ đồ, chỉ số Cost Per Wear ($CPW = \text{Giá mua} / \text{Số lần mặc}$), trang phục đắt giá nhưng ít mặc nhất, và cảnh báo ưu đãi sắp hết hạn.

**Why this priority**: Biến ứng dụng thành trợ lý thời trang cá nhân hóa (Smart Wardrobe), giúp người dùng chi tiêu thông minh và tăng tần suất quay lại ứng dụng.

**Independent Test**: Test API `GET /api/v1/user/dashboard/wardrobe-insights` kiểm tra con số CPW được tính toán chính xác từ `wardrobe_items` và lịch sử outfit.

**Acceptance Scenarios**:
1. **Given** Người dùng B2C mở ứng dụng, **When** truy cập Dashboard tủ đồ cá nhân, **Then** hiển thị:
   - **Wardrobe Financial Value & CPW**: Tổng giá trị tài chính của tủ đồ và chỉ số CPW trung bình.
   - **Smart Recommendations**: Danh sách "Top 5 trang phục bị lãng phí" (CPW cao + không mặc trong >30 ngày) kèm gợi ý phối đồ từ AI Stylist.
   - **AI Stylist Quota & Loyalty Alert**: Hạn mức AI còn lại trong ngày và cảnh báo điểm/voucher sắp hết hạn trong 7 ngày.

---

### Edge Cases

- **Dynamic Query Abuse / Large Date Range**: Khi client gửi query lọc khoảng thời gian quá dài (>1 năm với granularity=hour), Dynamic Aggregation Engine tự động điều chỉnh granularity thành `month` hoặc từ chối query để bảo vệ tài nguyên hệ thống.
- **Tenant Scope Bypass Attempt**: Nếu một request cố tình truyền `brand_id` của thương hiệu khác trong Payload, **Tenant Scoping Interceptor** sẽ ghi đè `brand_id` bằng giá trị lấy trực tiếp từ JWT Claims verified.
- **Microservices Outbox Lag**: Khi hệ thống gặp tải cao khiến Event Outbox Worker bị trễ (Lag 5s), Dashboard hiển thị nhãn "Dữ liệu cập nhật cách đây N phút" để người dùng nhận biết.

---

## Requirements *(mandatory)*

### Functional Requirements

#### 1. System Architecture & Modern Dashboard Engine
- **FR-001**: Hệ thống MUST triển khai module `internal/modules/dashboard/` theo mô hình **CQRS Event-Driven Outbox & Read Model Projections**.
- **FR-002**: Hệ thống MUST cung cấp một **Universal Dynamic Aggregation Engine** thông qua endpoint `POST /api/v1/dashboard/analytics/query` tiếp nhận các tham số động: `metric`, `timeframe` (`from`, `to`), `granularity` (`hour` | `day` | `week` | `month`), `filters`, `group_by`, và `compare_with_previous` (WoW, MoM, YoY).
- **FR-003**: Hệ thống MUST hỗ trợ tính năng **Drill-Down Query APIs** (`/api/v1/.../dashboard/drilldown/*`) cho phép truy vấn danh sách bản ghi chi tiết có phân trang từ bất kỳ chỉ số tổng quan nào trên Dashboard.

#### 2. Advanced Analytics & Predictive Capabilities
- **FR-004 (AI Margin Analytics)**: System Admin Dashboard MUST tổng hợp và tính toán chi phí token AI thực tế (`actual_cost_micro_vnd` từ `ai_usage_period_ledgers`) so với Doanh thu gói Premium B2C (`subscription_plans` & `deposit_transactions`) kèm tỷ lệ tăng trưởng so với chu kỳ trước (MoM % Change).
- **FR-005 (Points Liability Forecast)**: Brand Dashboard MUST dự báo tổng số điểm thưởng và giá trị quy đổi tài chính sắp hết hạn trong 30 ngày tới dựa trên `loyalty_point_lots`.
- **FR-006 (User CPW Metric)**: User Dashboard MUST tự động tính toán chỉ số Cost Per Wear ($CPW = \text{purchase\_price} / \text{wear\_count}$) và đưa ra danh sách gợi ý trang phục lãng phí.

#### 3. Security, Multi-Tenancy & Column-Level RBAC
- **FR-007**: Hệ thống MUST triển khai **Tenant Scoping Interceptor** ở tầng Repository/Query Engine để tự động inject `brand_id` hoặc `user_id` từ JWT Claims vào tất cả các câu truy vấn Dashboard, loại bỏ rủi ro rò rỉ dữ liệu multi-tenant.
- **FR-008**: Hệ thống MUST triển khai **Column-Level RBAC (Data Redaction)** tại tầng Response Mapper: Ẩn các trường tài chính nhạy cảm (`cost`, `revenue`, `margin`) đối với tài khoản `brandmemberrole.Staff`.

#### 4. Microservices Transition & Outbox Pattern
- **FR-009**: Tất cả các cập nhật dữ liệu vào Dashboard Read Models MUST thông qua việc tiêu thụ các **Domain Events** (như `UserRegisteredEvent`, `AICostRecordedEvent`, `PointRedeemedEvent`) được phát hành từ Transactional Outbox Pattern của từng domain module.

---

### Key Entities *(Non-implementation description)*

- **Dashboard Time-Series Read Models**: Các bảng aggregate lưu trữ theo chu kỳ thời gian (Hourly/Daily Buckets) hỗ trợ truy vấn so sánh MoM/WoW siêu tốc.
- **Event Outbox Ledger**: Bảng lưu trữ sự kiện sự cố nghiệp vụ phát sinh trong hệ thống phục vụ việc projection bất đồng bộ sang Dashboard Database.
- **Dynamic Query Contract**: Cấu trúc DTO quy định bộ lọc, nhóm dữ liệu, khoảng thời gian và phép so sánh chu kỳ chuẩn hóa cho toàn bộ Dashboard APIs.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: API Dynamic Query `POST /api/v1/dashboard/analytics/query` phản hồi dưới **150ms** đối với 95% các truy vấn phức tạp (bao gồm so sánh chu kỳ MoM/WoW) nhờ các bảng Time-Series Read Models.
- **SC-002**: 100% các con số aggregate KPI trên Dashboard Admin và Brand có khả năng Drill-down xuống danh sách bản ghi chi tiết trong vòng dưới 2 click của người dùng.
- **SC-003**: 100% rủi ro rò rỉ dữ liệu Multi-tenant được triệt tiêu nhờ Tenant Scoping Interceptor tự động ghi đè Scope từ JWT.
- **SC-004**: Khi ứng dụng tách thành Microservices độc lập, 0% code của các domain modules chính (`identity`, `brand`, `wardrobe`) phải viết lại ở tầng Dashboard.

---

## Assumptions

- **Hạ tầng Event & Time-Series DB**: Hệ thống hỗ trợ Transactional Outbox trên Postgresql và Redis Caching.
- **Lưu trữ dữ liệu lịch sử**: Dữ liệu Time-Series Snapshots được nén (Compress) và lưu trữ tối thiểu 24 tháng phục vụ so sánh YoY.
