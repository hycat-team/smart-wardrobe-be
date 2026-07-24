# Đặc Tả Tính Năng: Mở Rộng Dashboard (Dashboard Expansion)

**Feature**: `008-dashboard-expansion`

**Ngày tạo**: 2026-07-24

**Trạng thái**: Bản nháp

**Đầu vào**: Yêu cầu mở rộng Dashboard với 12 API mới và 8 metric mới cho Dynamic Query Engine, chia làm 3 nhóm: Admin (5 APIs + 3 metrics), Brand (6 APIs + 5 metrics), User (1 API).

---

## Tổng Quan

Tính năng này mở rộng hệ thống Dashboard & Analytics hiện có (Feature 006) bằng cách bổ sung các API thống kê chuyên sâu dựa trên các trường dữ liệu đã tồn tại trong database. Các API mới khai thác dữ liệu từ nhiều bảng nghiệp vụ khác nhau để cung cấp góc nhìn chi tiết hơn cho từng nhóm đối tượng.

**Phạm vi**:
1. **Admin Dashboard**: 5 API endpoints + 3 Dynamic Query metrics
2. **Brand Dashboard**: 6 API endpoints + 5 Dynamic Query metrics
3. **User Dashboard**: 1 API endpoint

---

## Kịch Bản Người Dùng & Kiểm Thử

### Câu Chuyện 1 — Admin Xem Phân Bổ Thuê Bao & Doanh Thu (P0)

Là một System Admin, tôi muốn xem số lượng người dùng phân bổ theo từng gói subscription và doanh thu chi tiết theo từng gói, để đánh giá hiệu quả kinh doanh của từng plan.

**Independent Test**: Gọi `GET /admin/dashboard/subscriptions/distribution` kiểm tra số user mỗi plan có khớp với dữ liệu trong `user_subscriptions` không. Gọi `GET /admin/dashboard/subscriptions/revenue-breakdown` kiểm tra tổng doanh thu có khớp với `deposit_transactions` không.

**Acceptance Scenarios**:
1. **Given** Admin vào Dashboard, **When** xem mục Subscription, **Then** thấy biểu đồ phân bổ user theo gói (FREE, PREMIUM_MONTHLY, PREMIUM_YEARLY, v.v.) và doanh thu theo từng gói theo tháng.
2. **Given** Admin muốn xem chi tiết, **When** chọn khoảng thời gian, **Then** dữ liệu doanh thu được lọc tương ứng.

---

### Câu Chuyện 2 — Admin Xem Thống Kê Sử Dụng AI Theo Tác Vụ (P1)

Là một System Admin, tôi muốn biết tính năng AI nào (outfit-recommend, chat, styling, v.v.) đang tiêu tốn nhiều tài nguyên nhất và tỷ lệ lỗi của từng provider/model, để tối ưu chi phí và debug sự cố.

**Independent Test**: Gọi `GET /admin/dashboard/ai/usage-by-operation?fromDate=2026-07-01&toDate=2026-07-24` kiểm tra output có phân bổ theo operation không. Gọi `GET /admin/dashboard/ai/error-breakdown` kiểm tra danh sách lỗi theo error_code.

**Acceptance Scenarios**:
1. **Given** Admin vào mục AI Analytics, **When** chọn khoảng thời gian, **Then** thấy số request và chi phí theo từng operation (outfit-recommend, chat, v.v.).
2. **Given** Admin muốn xem lỗi, **When** vào mục Error Breakdown, **Then** thấy số lỗi theo error_code, provider, model.

---

### Câu Chuyện 3 — Admin Xem Thống Kê Renewal Subscription (P1)

Là một System Admin, tôi muốn biết tỷ lệ gia hạn subscription thành công để đánh giá sức khỏe dòng doanh thu định kỳ.

**Independent Test**: Gọi `GET /admin/dashboard/subscriptions/renewal-stats` kiểm tra tỷ lệ success/fail có khớp với `subscription_renewal_attempts` không.

**Acceptance Scenarios**:
1. **Given** Admin vào mục Subscription Renewal, **When** xem thống kê, **Then** thấy tổng số lần renewal, số thành công, số thất bại, tỷ lệ thành công.

---

### Câu Chuyện 4 — Brand Xem Kênh Thu Hút Khách Hàng & Tier Distribution (P0)

Là một Brand Manager, tôi muốn biết khách hàng đến từ kênh nào (walk_in, referral, online, v.v.) và phân bổ hạng loyalty của họ, để điều chỉnh chiến lược marketing và loyalty program.

**Independent Test**: Gọi `GET /brand/dashboard/customers/acquisition?brandId=<uuid>` kiểm tra phân bổ joined_source. Gọi `GET /brand/dashboard/loyalty/tier-distribution?brandId=<uuid>` kiểm tra số member theo từng tier loyalty.

**Acceptance Scenarios**:
1. **Given** Brand Manager vào Dashboard, **When** xem mục Customers, **Then** thấy biểu đồ tròn phân bổ kênh thu hút.
2. **Given** Brand Manager xem mục Loyalty, **Then** thấy số member theo từng hạng (Bronze, Silver, Gold, v.v.).

---

### Câu Chuyện 5 — Brand Xem Dự Báo Điểm Hết Hạn & Redemption Analytics (P1)

Là một Brand Manager, tôi muốn biết điểm sắp hết hạn theo tháng và benefit nào được đổi nhiều nhất, để lên kế hoạch khuyến mãi và quản lý nợ điểm.

**Independent Test**: Gọi `GET /brand/dashboard/loyalty/point-expiry-forecast?brandId=<uuid>` kiểm tra forecast và `GET /brand/dashboard/benefits/redemption-analytics?brandId=<uuid>` kiểm tra top benefits.

**Acceptance Scenarios**:
1. **Given** Brand Manager xem mục Point Expiry, **Then** thấy biểu đồ dự báo điểm hết hạn theo từng tháng.
2. **Given** Brand Manager xem mục Benefits, **Then** thấy top benefit được đổi nhiều nhất, redemption trend.

---

### Câu Chuyện 6 — Brand Xem Thống Kê Catalog & Phân Khúc Chi Tiêu (P2)

Là một Brand Manager, tôi muốn biết catalog có bao nhiêu sản phẩm theo trạng thái/loại và khách hàng chi tiêu bao nhiêu, để quản lý hàng tồn và phân khúc khách hàng.

**Independent Test**: Gọi `GET /brand/dashboard/operations/catalog-stats?brandId=<uuid>` và `GET /brand/dashboard/customers/spend-segmentation?brandId=<uuid>`.

**Acceptance Scenarios**:
1. **Given** Brand Manager xem mục Catalog, **Then** thấy tổng sản phẩm, phân bổ theo status và item_type.
2. **Given** Brand Manager xem mục Spend Segmentation, **Then** thấy số customer theo nhóm chi tiêu (0-1tr, 1-5tr, >5tr).

---

### Câu Chuyện 7 — User Xem Phân Bổ Danh Mục Tủ Đồ (P2)

Là một người dùng B2C, tôi muốn xem biểu đồ tròn phân bổ quần áo trong tủ đồ của mình theo danh mục (Áo, Quần, Giày, v.v.), để biết mình đang thiếu/món đồ gì.

**Independent Test**: Gọi `GET /me/dashboard/wardrobe/category-distribution` kiểm tra phân bổ category từ `wardrobe_items` → `fashion_items` → `categories`.

**Acceptance Scenarios**:
1. **Given** User vào Dashboard cá nhân, **When** xem mục Wardrobe, **Then** thấy biểu đồ tròn tỷ lệ phần trăm từng danh mục (Áo Sơ Mi: 30%, Quần Jean: 25%, Giày: 15%, v.v.) kèm số lượng cụ thể.

---

### Edge Cases

- **Brand không có dữ liệu**: Các API Brand trả về mảng/dữ liệu rỗng thay vì lỗi 404.
- **User chưa có wardrobe item**: API User trả về danh sách rỗng, không lỗi.
- **Khoảng thời gian không hợp lệ**: Các API có query param ngày tháng validate định dạng, trả 400 nếu sai.
- **RBAC**: Các API Admin chỉ Admin mới truy cập được. API Brand kiểm tra quyền member. API User tự động scope theo user hiện tại.
- **Dữ liệu zero value**: Các API luôn trả về `0` hoặc `[]` khi không có dữ liệu (không trả về `null`). Tuy nhiên cần phân biệt:
  - **Real zero**: Dữ liệu tồn tại trong DB, giá trị thực sự là 0 (ví dụ: không có lỗi AI → `totalErrors: 0`).
  - **Zero value do lỗi type assertion**: Khi dùng pgx scan `SUM()` của `int8` / `numeric` vào `map[string]interface{}`, giá trị trả về là `float64`, không phải `int64`. Nếu type assertion sai (`.(int64)`), giá trị mặc định `0` được dùng thay vì data thật → **bug**. Tất cả aggregate numeric fields (`SUM`, `AVG`) của `int8`/`numeric` phải dùng `toFloat64()` helper để parse an toàn, sau đó cast về kiểu mong muốn.
  - **Mảng rỗng**: `[]` luôn là real data (không có bản ghi nào), không phải zero value.

---

## Yêu Cầu

### Yêu Cầu Chức Năng

#### Nhóm 1: Admin Dashboard APIs

- **FR-001 (Subscription Distribution)**: Hệ thống MUST cung cấp API `GET /api/v1/admin/dashboard/subscriptions/distribution` trả về số lượng user và tỷ lệ phần trăm theo từng `current_plan_code` (hoặc `current_tier_rank`), dựa trên bảng `user_subscriptions`.
- **FR-002 (Revenue Breakdown)**: Hệ thống MUST cung cấp API `GET /api/v1/admin/dashboard/subscriptions/revenue-breakdown` trả về tổng doanh thu theo từng `plan_code` và khoảng thời gian, dựa trên bảng `deposit_transactions` với `transaction_type='DIRECT_PURCHASE'` và `status=1 (Success)`.
- **FR-003 (AI Usage by Operation)**: Hệ thống MUST cung cấp API `GET /api/v1/admin/dashboard/ai/usage-by-operation` trả về số request, tổng chi phí (`actual_cost_micro_vnd`) và số token theo từng `operation` và `logical_route`, dựa trên bảng `ai_usage_events`. Hỗ trợ lọc `fromDate`/`toDate`.
- **FR-004 (AI Error Breakdown)**: Hệ thống MUST cung cấp API `GET /api/v1/admin/dashboard/ai/error-breakdown` trả về số lỗi, số request và tỷ lệ lỗi theo từng `error_code`, `provider`, `model`, dựa trên bảng `ai_usage_events` với `status` thuộc nhóm lỗi (dùng package `aiusageeventstatus` để xác định status nào là lỗi, không tự định nghĩa danh sách cứng).
- **FR-005 (Renewal Stats)**: Hệ thống MUST cung cấp API `GET /api/v1/admin/dashboard/subscriptions/renewal-stats` trả về tổng số lần renewal, số thành công, số thất bại, tỷ lệ thành công và số lần retry trung bình, dựa trên bảng `subscription_renewal_attempts`.
- **FR-006 (Admin Dynamic Query Metrics)**: Hệ thống MUST bổ sung 3 metric mới vào Dynamic Query Engine cho target=admin:
  - `new_users`: từ `snapshot.NewUsers` (field đã tồn tại trong entity)
  - `ai_error_count`: từ `snapshot.AIErrorCount` (field đã tồn tại trong entity)
  - `ai_error_rate`: tính toán = `ai_error_count / ai_requests_count * 100`

#### Nhóm 2: Brand Dashboard APIs

- **FR-007 (Customer Acquisition Source)**: Hệ thống MUST cung cấp API `GET /api/v1/brand/dashboard/customers/acquisition` trả về số lượng customer và tỷ lệ phần trăm theo từng `joined_source`, dựa trên bảng `brand_customers`. Yêu cầu `brandId` query param.
- **FR-008 (Loyalty Tier Distribution)**: Hệ thống MUST cung cấp API `GET /api/v1/brand/dashboard/loyalty/tier-distribution` trả về số lượng loyalty account theo từng `current_tier_id` (kèm tên tier từ `loyalty_tiers`), dựa trên bảng `loyalty_accounts`. Yêu cầu `brandId` query param.
- **FR-009 (Point Expiry Forecast)**: Hệ thống MUST cung cấp API `GET /api/v1/brand/dashboard/loyalty/point-expiry-forecast` trả về dự báo số điểm hết hạn theo từng tháng (tối thiểu 3 tháng tới), dựa trên bảng `loyalty_point_lots` với `status='ACTIVE'` và `remaining_points>0`. Yêu cầu `brandId` query param.
- **FR-010 (Benefit Redemption Analytics)**: Hệ thống MUST cung cấp API `GET /api/v1/brand/dashboard/benefits/redemption-analytics` trả về top 10 benefit được đổi nhiều nhất kèm số lượt đổi và xu hướng theo thời gian, dựa trên bảng `benefit_redemptions` và `brand_benefits`. Yêu cầu `brandId` query param.
- **FR-011 (Catalog Stats)**: Hệ thống MUST cung cấp API `GET /api/v1/brand/dashboard/operations/catalog-stats` trả về tổng số sản phẩm, số lượng theo từng `status`, số lượng theo từng `item_type`, dựa trên bảng `brand_items`. Yêu cầu `brandId` query param.
- **FR-012 (Customer Spend Segmentation)**: Hệ thống MUST cung cấp API `GET /api/v1/brand/dashboard/customers/spend-segmentation` trả về số lượng customer theo các nhóm chi tiêu (0-1tr, 1-5tr, 5-10tr, >10tr), dựa trên bảng `loyalty_accounts.total_spend`. Yêu cầu `brandId` query param.
- **FR-013 (Brand Dynamic Query Metrics)**: Hệ thống MUST bổ sung 5 metric mới vào Dynamic Query Engine cho target=brand:
  - `new_customers`: từ `brand_customers.created_at`
  - `customer_claim_rate`: từ `brand_customers` tỷ lệ `claimed_at IS NOT NULL` / tổng số
  - `benefit_redemption_count`: từ `benefit_redemptions.redeemed_at`
  - `brand_item_count`: từ `brand_items`
  - `brand_item_count_by_type`: từ `brand_items.item_type`

#### Nhóm 3: User Dashboard API

- **FR-014 (Wardrobe Category Distribution)**: Hệ thống MUST cung cấp API `GET /api/v1/me/dashboard/wardrobe/category-distribution` trả về số lượng và tỷ lệ phần trăm item trong tủ đồ theo từng danh mục (`categories`), dựa trên chuỗi join `wardrobe_items` → `fashion_items` → `categories`. Scope tự động theo user hiện tại.

#### Nhóm 4: Yêu Cầu Phi Chức Năng

- **FR-015 (RBAC)**: Các API Admin phải yêu cầu role Admin. API Brand phải kiểm tra quyền brand member. API User tự động scope theo JWT user.
- **FR-016 (Error Handling)**: Khi không có dữ liệu, API trả về dữ liệu rỗng/0 thay vì lỗi 404, trừ trường hợp brandId không hợp lệ.
- **FR-017 (Documentation)**: Kết thúc plan, file `docs/api/dashboard-api-v2.md` phải được tạo mới với đầy đủ thông tin các API cũ và mới.

---

## Tiêu Chí Thành Công

### Kết Quả Đo Lường Được

- **SC-001**: 100% API mới trả về dữ liệu chính xác, khớp với kết quả query trực tiếp từ database.
- **SC-002**: Các API mới phản hồi trong vòng dưới 2 giây cho bộ dữ liệu dưới 100,000 bản ghi.
- **SC-003**: 100% API kiểm tra RBAC đúng role trước khi trả dữ liệu.
- **SC-004**: 3 metric dynamic query mới (`new_users`, `ai_error_count`, `ai_error_rate`) hoạt động trong `POST /api/v1/dashboard/analytics/query` với target=admin.
- **SC-005**: 5 metric dynamic query mới hoạt động trong `POST /api/v1/dashboard/analytics/query` với target=brand.
- **SC-006**: File `docs/api/dashboard-api-v2.md` được tạo mới với đầy đủ spec cho tất cả API cũ và mới.

---

## Thực Thể Chính

- **`user_subscriptions`**: Thông tin subscription hiện tại của user (plan_code, tier_rank, expires_at).
- **`deposit_transactions`**: Lịch sử giao dịch nạp tiền mua subscription (plan_code, amount, status, created_at).
- **`ai_usage_events`**: Sự kiện sử dụng AI (operation, logical_route, provider, model, status, error_code, actual_cost_micro_vnd, prompt_tokens, output_tokens, created_at).
- **`subscription_renewal_attempts`**: Lịch sử các lần thử renewal (status, attempt_count, last_error_code, completed_at).
- **`brand_customers`**: Khách hàng của brand (joined_source, claimed_at, status, brand_id).
- **`loyalty_accounts`**: Tài khoản loyalty (current_tier_id, total_spend, brand_id).
- **`loyalty_tiers`**: Định nghĩa các hạng loyalty (name, rank, brand_id).
- **`loyalty_point_lots`**: Lô điểm chi tiết (earned_points, remaining_points, expires_at, status).
- **`benefit_redemptions`**: Lịch sử đổi benefit (benefit_id, status, redeemed_at, brand_id).
- **`brand_benefits`**: Định nghĩa benefit (name, benefit_type, brand_id).
- **`brand_items`**: Sản phẩm của brand (status, item_type, price, brand_id).
- **`wardrobe_items`**: Món đồ trong tủ (user_id, fashion_item_id, is_deleted).
- **`fashion_items`**: Item thời trang (category_id).
- **`categories`**: Danh mục thời trang (name, slug).

---

## Giả Định

- Các snapshot hiện tại (`dashboard_admin_daily_snapshots`) đã có sẵn trường `new_users` và `ai_error_count` — chỉ cần thêm mapping trong Dynamic Query Engine.
- Các API mới query trực tiếp từ bảng nghiệp vụ (không qua snapshot) để đảm bảo dữ liệu real-time.
- Các chỉ số tính toán (`ai_error_rate`, `customer_claim_rate`) được tính ở tầng use case, không lưu vào database.
- Phân khúc chi tiêu sử dụng mốc cố định: 0-1,000,000 VND, 1,000,000-5,000,000 VND, 5,000,000-10,000,000 VND, >10,000,000 VND.
- RBAC kế thừa cơ chế hiện tại: middleware auth + role check cho Admin, brand member check cho Brand, JWT user scope cho User.
