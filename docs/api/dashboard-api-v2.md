# Dashboard API Expansion (Spec 008)

Tài liệu bổ sung cho [dashboard-api.md](dashboard-api.md). Chỉ bao gồm các API và metric mới được thêm trong Feature 008.

---

## Flow 1: Admin Dashboard — APIs mở rộng

### 1. Lấy phân bổ người dùng theo gói subscription

- **Endpoint:** `GET /api/v1/admin/dashboard/subscriptions/distribution`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê số lượng người dùng đang hoạt động theo từng gói subscription (FREE, PREMIUM_MONTHLY, PREMIUM_YEARLY, v.v.), kèm tỷ lệ phần trăm.
- **Response struct:** `SubscriptionDistributionRes`
- **Nguồn dữ liệu:** Bảng `user_subscriptions`, query `SELECT current_plan_code, COUNT(*) GROUP BY current_plan_code ORDER BY COUNT(*) DESC`.

| Field        | JSON           | Kiểu                          | Ý nghĩa                                          |
| ------------ | -------------- | ----------------------------- | ------------------------------------------------ |
| Distribution | `distribution` | []SubscriptionDistributionDTO | Danh sách phân bổ theo plan_code                 |
| ├ PlanCode   | `planCode`     | string                        | Mã gói subscription (FREE, PREMIUM_MONTHLY, ...) |
| ├ UserCount  | `userCount`    | int                           | Số người dùng thuộc gói này                      |
| └ Percentage | `percentage`   | float64                       | Tỷ lệ phần trăm so với tổng user                 |
| TotalUsers   | `totalUsers`   | int                           | Tổng số người dùng có subscription               |

---

### 2. Lấy doanh thu subscription theo gói và thời gian

- **Endpoint:** `GET /api/v1/admin/dashboard/subscriptions/revenue-breakdown`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê doanh thu subscription theo từng plan_code và theo tháng, chỉ tính các giao dịch DIRECT_PURCHASE thành công. Hỗ trợ lọc khoảng thời gian.
- **Query Parameters:**
    - `fromDate` (string, optional) — Ngày bắt đầu, định dạng `YYYY-MM-DD`. Mặc định: 6 tháng trước.
    - `toDate` (string, optional) — Ngày kết thúc, định dạng `YYYY-MM-DD`. Mặc định: hôm nay.
- **Response struct:** `RevenueBreakdownRes`
- **Nguồn dữ liệu:** Bảng `deposit_transactions`, `WHERE status=1 AND transaction_type='DIRECT_PURCHASE'`, `GROUP BY plan_code, DATE_TRUNC('month', created_at)`.

| Field              | JSON               | Kiểu                  | Ý nghĩa                               |
| ------------------ | ------------------ | --------------------- | ------------------------------------- |
| FromDate           | `fromDate`         | string                | Ngày bắt đầu kỳ thống kê              |
| ToDate             | `toDate`           | string                | Ngày kết thúc kỳ thống kê             |
| Breakdown          | `breakdown`        | []RevenueBreakdownDTO | Danh sách doanh thu theo plan + tháng |
| ├ PlanCode         | `planCode`         | string                | Mã gói subscription                   |
| ├ Month            | `month`            | string                | Tháng (YYYY-MM)                       |
| ├ TransactionCount | `transactionCount` | int                   | Số giao dịch thành công               |
| └ RevenueVND       | `revenueVnd`       | float64               | Tổng doanh thu (VND)                  |
| TotalRevenueVND    | `totalRevenueVnd`  | float64               | Tổng doanh thu tất cả gói             |

---

### 3. Lấy thống kê sử dụng AI theo tác vụ

- **Endpoint:** `GET /api/v1/admin/dashboard/ai/usage-by-operation`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê số lượng request, chi phí (VND), và token AI theo từng loại tác vụ (operation) và tuyến xử lý (logical_route). Chi phí được quy đổi từ micro VND sang VND.
- **Query Parameters:**
    - `fromDate` (string, optional) — Ngày bắt đầu, định dạng `YYYY-MM-DD`. Mặc định: 30 ngày trước.
    - `toDate` (string, optional) — Ngày kết thúc, định dạng `YYYY-MM-DD`. Mặc định: hôm nay.
- **Response struct:** `AIUsageByOperationRes`
- **Nguồn dữ liệu:** Bảng `ai_usage_events`, `GROUP BY operation, logical_route`.

| Field               | JSON                | Kiểu                    | Ý nghĩa                                                |
| ------------------- | ------------------- | ----------------------- | ------------------------------------------------------ |
| FromDate            | `fromDate`          | string                  | Ngày bắt đầu kỳ thống kê                               |
| ToDate              | `toDate`            | string                  | Ngày kết thúc kỳ thống kê                              |
| Operations          | `operations`        | []AIUsageByOperationDTO | Danh sách thống kê theo operation                      |
| ├ Operation         | `operation`         | string                  | Loại tác vụ (outfit-recommend, chat, ...)              |
| ├ LogicalRoute      | `logicalRoute`      | string                  | Tuyến xử lý (normal, reduced, free)                    |
| ├ RequestCount      | `requestCount`      | int64                   | Số lượng request                                       |
| ├ TotalCostVND      | `totalCostVnd`      | float64                 | Tổng chi phí (VND) = actual_cost_micro_vnd / 1,000,000 |
| ├ TotalPromptTokens | `totalPromptTokens` | int64                   | Tổng token đầu vào                                     |
| └ TotalOutputTokens | `totalOutputTokens` | int64                   | Tổng token đầu ra                                      |

---

### 4. Lấy thống kê lỗi AI

- **Endpoint:** `GET /api/v1/admin/dashboard/ai/error-breakdown`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê số lượng lỗi AI, tổng số request trong 30 ngày qua, và tỷ lệ lỗi. Chi tiết lỗi được phân nhóm theo error_code, provider, và model.
- **Response struct:** `AIErrorBreakdownRes`
- **Nguồn dữ liệu:** Bảng `ai_usage_events`, lấy các status thuộc nhóm lỗi (tham khảo package `aiusageeventstatus`). Tổng request lấy từ `COUNT(*)` trong 30 ngày gần nhất.

| Field         | JSON            | Kiểu                  | Ý nghĩa                                        |
| ------------- | --------------- | --------------------- | ---------------------------------------------- |
| TotalErrors   | `totalErrors`   | int64                 | Tổng số request bị lỗi                         |
| TotalRequests | `totalRequests` | int64                 | Tổng số request (30 ngày qua)                  |
| ErrorRate     | `errorRate`     | float64               | Tỷ lệ lỗi = totalErrors / totalRequests \* 100 |
| Breakdown     | `breakdown`     | []AIErrorBreakdownDTO | Chi tiết lỗi theo nhóm                         |
| ├ ErrorCode   | `errorCode`     | string                | Mã lỗi (rate_limit_exceeded, timeout, ...)     |
| ├ Provider    | `provider`      | string                | Nhà cung cấp AI (openai, anthropic, ...)       |
| ├ Model       | `model`         | string                | Model AI (gpt-4, claude-3, ...)                |
| └ ErrorCount  | `errorCount`    | int64                 | Số lần xuất hiện lỗi này                       |

---

### 5. Lấy thống kê gia hạn subscription

- **Endpoint:** `GET /api/v1/admin/dashboard/subscriptions/renewal-stats`
- **Tác nhân (Actor):** Quản trị viên (`roleslug.Admin`).
- **Mô tả:** Thống kê tổng quan về tình trạng gia hạn subscription: tổng số lần gia hạn, số thành công, số thất bại, tỷ lệ thành công, và số lần thử lại trung bình.
- **Response struct:** `RenewalStatsRes`
- **Nguồn dữ liệu:** Bảng `subscription_renewal_attempts`.

| Field         | JSON            | Kiểu    | Ý nghĩa                                                |
| ------------- | --------------- | ------- | ------------------------------------------------------ |
| TotalAttempts | `totalAttempts` | int     | Tổng số lần gia hạn (đã thực hiện)                     |
| SuccessCount  | `successCount`  | int     | Số lần gia hạn thành công (status = 'SUCCESS')         |
| FailedCount   | `failedCount`   | int     | Số lần gia hạn thất bại (status = 'FAILED')            |
| SuccessRate   | `successRate`   | float64 | Tỷ lệ thành công = successCount / totalAttempts \* 100 |
| AvgRetryCount | `avgRetryCount` | float64 | Số lần thử lại trung bình (AVG của attempt_count)      |

---

## Flow 2: Brand Dashboard — APIs mở rộng

### 6. Lấy thống kê kênh thu hút khách hàng

- **Endpoint:** `GET /api/v1/brand/dashboard/customers/acquisition`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Mô tả:** Thống kê số lượng khách hàng và tỷ lệ phần trăm theo từng kênh thu hút (joined_source). Ví dụ: walk_in, referral, online, QR, ...
- **Query Parameters:** `brandId` (string, required) — UUID của nhãn hàng.
- **Response struct:** `CustomerAcquisitionRes`
- **Nguồn dữ liệu:** Bảng `brand_customers`, `GROUP BY joined_source WHERE brand_id = ?`.

| Field           | JSON             | Kiểu                     | Ý nghĩa                                     |
| --------------- | ---------------- | ------------------------ | ------------------------------------------- |
| BrandID         | `brandId`        | uuid.UUID                | UUID của nhãn hàng                          |
| TotalCustomers  | `totalCustomers` | int                      | Tổng số khách hàng                          |
| Sources         | `sources`        | []CustomerAcquisitionDTO | Phân bổ theo kênh thu hút                   |
| ├ Source        | `source`         | string                   | Tên kênh (ví dụ: walk_in, referral, online) |
| ├ CustomerCount | `customerCount`  | int                      | Số khách hàng từ kênh này                   |
| └ Percentage    | `percentage`     | float64                  | Tỷ lệ phần trăm so với tổng                 |

---

### 7. Lấy phân bổ hạng loyalty thành viên

- **Endpoint:** `GET /api/v1/brand/dashboard/loyalty/tier-distribution`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Mô tả:** Thống kê số lượng thành viên và tỷ lệ phần trăm theo từng hạng loyalty (Bronze, Silver, Gold, v.v.).
- **Query Parameters:** `brandId` (string, required) — UUID của nhãn hàng.
- **Response struct:** `TierDistributionRes`
- **Nguồn dữ liệu:** Bảng `loyalty_accounts` LEFT JOIN `loyalty_tiers`, `GROUP BY current_tier_id`.

| Field         | JSON           | Kiểu                  | Ý nghĩa                               |
| ------------- | -------------- | --------------------- | ------------------------------------- |
| BrandID       | `brandId`      | uuid.UUID             | UUID của nhãn hàng                    |
| TotalMembers  | `totalMembers` | int                   | Tổng số thành viên loyalty            |
| Tiers         | `tiers`        | []TierDistributionDTO | Phân bổ theo hạng                     |
| ├ TierID      | `tierId`       | uuid.UUID             | ID của hạng loyalty                   |
| ├ TierName    | `tierName`     | string                | Tên hạng (Bronze, Silver, Gold, v.v.) |
| ├ TierRank    | `tierRank`     | int                   | Thứ hạng (1, 2, 3, ...)               |
| ├ MemberCount | `memberCount`  | int                   | Số thành viên thuộc hạng này          |
| └ Percentage  | `percentage`   | float64               | Tỷ lệ phần trăm so với tổng           |

---

### 8. Lấy dự báo điểm sắp hết hạn

- **Endpoint:** `GET /api/v1/brand/dashboard/loyalty/point-expiry-forecast`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Mô tả:** Dự báo số điểm thưởng sắp hết hạn theo từng tháng (tối đa 6 tháng), dựa trên các lô điểm (loyalty_point_lots) đang ACTIVE và còn điểm.
- **Query Parameters:** `brandId` (string, required) — UUID của nhãn hàng.
- **Response struct:** `PointExpiryForecastRes`
- **Nguồn dữ liệu:** Bảng `loyalty_point_lots`, `WHERE status='ACTIVE' AND remaining_points>0 AND expires_at IS NOT NULL`, `GROUP BY DATE_TRUNC('month', expires_at)`.

| Field            | JSON             | Kiểu                     | Ý nghĩa                              |
| ---------------- | ---------------- | ------------------------ | ------------------------------------ |
| BrandID          | `brandId`        | uuid.UUID                | UUID của nhãn hàng                   |
| Forecast         | `forecast`       | []PointExpiryForecastDTO | Dự báo theo tháng                    |
| ├ Month          | `month`          | string                   | Tháng dự báo (YYYY-MM)               |
| └ PointsExpiring | `pointsExpiring` | int                      | Tổng số điểm sắp hết hạn trong tháng |

---

### 9. Lấy thống kê đổi benefit

- **Endpoint:** `GET /api/v1/brand/dashboard/benefits/redemption-analytics`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Mô tả:** Thống kê top 10 benefit được khách hàng đổi nhiều nhất kèm số lượt đổi và tỷ lệ phần trăm, cùng xu hướng đổi benefit theo thời gian.
- **Query Parameters:** `brandId` (string, required) — UUID của nhãn hàng.
- **Response struct:** `BenefitRedemptionAnalyticsRes`
- **Nguồn dữ liệu:** Bảng `benefit_redemptions` JOIN `brand_benefits`, `GROUP BY benefit_id`.

| Field             | JSON               | Kiểu                   | Ý nghĩa                                 |
| ----------------- | ------------------ | ---------------------- | --------------------------------------- |
| BrandID           | `brandId`          | uuid.UUID              | UUID của nhãn hàng                      |
| TotalRedemptions  | `totalRedemptions` | int                    | Tổng số lượt đổi benefit                |
| Trend             | `trend`            | []RedemptionTrendDTO   | Xu hướng đổi benefit theo tháng         |
| ├ Month           | `month`            | string                 | Tháng (YYYY-MM)                         |
| └ RedemptionCount | `redemptionCount`  | int                    | Số lượt đổi trong tháng                 |
| TopBenefits       | `topBenefits`      | []BenefitRedemptionDTO | Top 10 benefit được đổi nhiều nhất      |
| ├ BenefitID       | `benefitId`        | uuid.UUID              | UUID của benefit                        |
| ├ BenefitName     | `benefitName`      | string                 | Tên benefit (ví dụ: Giảm 10%, FreeShip) |
| ├ BenefitType     | `benefitType`      | string                 | Loại benefit (discount, shipping, ...)  |
| ├ RedemptionCount | `redemptionCount`  | int                    | Số lượt đổi                             |
| └ Percentage      | `percentage`       | float64                | Tỷ lệ phần trăm so với tổng             |

---

### 10. Lấy thống kê catalog sản phẩm

- **Endpoint:** `GET /api/v1/brand/dashboard/operations/catalog-stats`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Mô tả:** Thống kê tổng số sản phẩm trong catalog của brand, phân bổ theo trạng thái (ACTIVE, DRAFT, INACTIVE) và theo loại sản phẩm (clothing, accessory).
- **Query Parameters:** `brandId` (string, required) — UUID của nhãn hàng.
- **Response struct:** `CatalogStatsRes`
- **Nguồn dữ liệu:** Bảng `brand_items`, hai query `GROUP BY status` và `GROUP BY item_type`.

| Field        | JSON         | Kiểu                  | Ý nghĩa                                               |
| ------------ | ------------ | --------------------- | ----------------------------------------------------- |
| BrandID      | `brandId`    | uuid.UUID             | UUID của nhãn hàng                                    |
| TotalItems   | `totalItems` | int                   | Tổng số sản phẩm                                      |
| ByStatus     | `byStatus`   | []CatalogStatGroupDTO | Phân bổ theo trạng thái (ACTIVE, DRAFT, INACTIVE)     |
| ├ Label      | `label`      | string                | Giá trị trạng thái                                    |
| ├ Count      | `count`      | int                   | Số lượng                                              |
| └ Percentage | `percentage` | float64               | Tỷ lệ phần trăm                                       |
| ByType       | `byType`     | []CatalogStatGroupDTO | Phân bổ theo loại sản phẩm (clothing, accessory, ...) |

---

### 11. Lấy phân khúc chi tiêu khách hàng

- **Endpoint:** `GET /api/v1/brand/dashboard/customers/spend-segmentation`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Mô tả:** Phân khúc khách hàng loyalty theo tổng chi tiêu (total_spend) thành 4 nhóm: low (0-1tr), medium (1tr-5tr), high (5tr-10tr), premium (>10tr).
- **Query Parameters:** `brandId` (string, required) — UUID của nhãn hàng.
- **Response struct:** `SpendSegmentationRes`
- **Nguồn dữ liệu:** Bảng `loyalty_accounts`, dùng `CASE WHEN` để phân nhóm total_spend.

| Field           | JSON             | Kiểu              | Ý nghĩa                                            |
| --------------- | ---------------- | ----------------- | -------------------------------------------------- |
| BrandID         | `brandId`        | uuid.UUID         | UUID của nhãn hàng                                 |
| TotalCustomers  | `totalCustomers` | int               | Tổng số khách hàng loyalty                         |
| Segments        | `segments`       | []SpendSegmentDTO | Danh sách phân khúc                                |
| ├ Segment       | `segment`        | string            | Tên nhóm: low, medium, high, premium               |
| ├ MinVND        | `minVnd`         | float64           | Mức chi tiêu tối thiểu (VND)                       |
| ├ MaxVND        | `maxVnd`         | float64           | Mức chi tiêu tối đa (VND), null nếu không giới hạn |
| ├ CustomerCount | `customerCount`  | int               | Số khách hàng trong nhóm                           |
| └ Percentage    | `percentage`     | float64           | Tỷ lệ phần trăm so với tổng                        |

---

## Flow 3: User Dashboard — APIs mở rộng

### 12. Lấy phân bổ danh mục tủ đồ

- **Endpoint:** `GET /api/v1/me/dashboard/wardrobe/category-distribution`
- **Tác nhân (Actor):** Người dùng B2C (`roleslug.User`).
- **Mô tả:** Thống kê số lượng và tỷ lệ phần trăm các món đồ trong tủ của người dùng, phân nhóm theo danh mục thời trang (Áo Sơ Mi, Quần Jean, Giày, v.v.). Scope tự động theo user từ JWT.
- **Response struct:** `WardrobeCategoryDistributionRes`
- **Nguồn dữ liệu:** Join 3 bảng `wardrobe_items` → `fashion_items` → `categories`, `WHERE user_id=? AND is_deleted=false`, `GROUP BY category_id`.

| Field          | JSON           | Kiểu                      | Ý nghĩa                                   |
| -------------- | -------------- | ------------------------- | ----------------------------------------- |
| TotalItems     | `totalItems`   | int                       | Tổng số món đồ trong tủ                   |
| Categories     | `categories`   | []CategoryDistributionDTO | Phân bổ theo danh mục                     |
| ├ CategoryID   | `categoryId`   | uuid.UUID                 | UUID của danh mục                         |
| ├ CategoryName | `categoryName` | string                    | Tên danh mục (ví dụ: Áo Sơ Mi, Quần Jean) |
| ├ ItemCount    | `itemCount`    | int                       | Số lượng món đồ thuộc danh mục            |
| └ Percentage   | `percentage`   | float64                   | Tỷ lệ phần trăm so với tổng               |

---

## Dynamic Query Engine — Metrics mở rộng

### Admin metrics

Các metric này sử dụng field đã có sẵn trong `dashboard_admin_daily_snapshots`:

| Metric           | Field / Cách tính                          | Kiểu gốc        | Mô tả                                |
| ---------------- | ------------------------------------------ | --------------- | ------------------------------------ |
| `new_users`      | `snapshot.NewUsers`                        | int → float64   | Số người dùng mới đăng ký trong ngày |
| `ai_error_count` | `snapshot.AIErrorCount`                    | int64 → float64 | Số request AI bị lỗi trong ngày      |
| `ai_error_rate`  | `ai_error_count / ai_requests_count * 100` | float64         | Tỷ lệ lỗi AI (%), computed           |

### Brand metrics

Các metric brand mới query live từ bảng nguồn (không qua snapshot):

| Metric                     | Nguồn                 | Cách tính                                                | Mô tả                               |
| -------------------------- | --------------------- | -------------------------------------------------------- | ----------------------------------- |
| `new_customers`            | `brand_customers`     | `COUNT(*) WHERE created_at BETWEEN fromDate AND toDate`  | Số khách hàng mới trong kỳ          |
| `customer_claim_rate`      | `brand_customers`     | `claimed_at IS NOT NULL / total * 100`                   | Tỷ lệ khách hàng đã claim tài khoản |
| `benefit_redemption_count` | `benefit_redemptions` | `COUNT(*) WHERE redeemed_at BETWEEN fromDate AND toDate` | Số lượt đổi benefit trong kỳ        |
| `brand_item_count`         | `brand_items`         | `COUNT(*) WHERE brand_id = ?`                            | Tổng số sản phẩm trong catalog      |
| `brand_item_count_by_type` | `brand_items`         | `SELECT item_type, COUNT(*) GROUP BY item_type`          | Số sản phẩm theo từng loại          |

---

## Tổng hợp Endpoint mới

| Phương thức | Endpoint                                                  | Vai trò | Chức năng                                        |
| ----------- | --------------------------------------------------------- | ------- | ------------------------------------------------ |
| GET         | `/api/v1/admin/dashboard/subscriptions/distribution`      | Admin   | Phân bổ người dùng theo gói subscription         |
| GET         | `/api/v1/admin/dashboard/subscriptions/revenue-breakdown` | Admin   | Doanh thu subscription theo gói và thời gian     |
| GET         | `/api/v1/admin/dashboard/ai/usage-by-operation`           | Admin   | Thống kê sử dụng AI theo tác vụ                  |
| GET         | `/api/v1/admin/dashboard/ai/error-breakdown`              | Admin   | Thống kê lỗi AI theo error_code, provider, model |
| GET         | `/api/v1/admin/dashboard/subscriptions/renewal-stats`     | Admin   | Thống kê tỷ lệ gia hạn subscription              |
| GET         | `/api/v1/brand/dashboard/customers/acquisition`           | Brand   | Thống kê kênh thu hút khách hàng                 |
| GET         | `/api/v1/brand/dashboard/loyalty/tier-distribution`       | Brand   | Phân bổ hạng loyalty thành viên                  |
| GET         | `/api/v1/brand/dashboard/loyalty/point-expiry-forecast`   | Brand   | Dự báo điểm sắp hết hạn                          |
| GET         | `/api/v1/brand/dashboard/benefits/redemption-analytics`   | Brand   | Thống kê top benefit được đổi nhiều nhất         |
| GET         | `/api/v1/brand/dashboard/operations/catalog-stats`        | Brand   | Thống kê catalog sản phẩm                        |
| GET         | `/api/v1/brand/dashboard/customers/spend-segmentation`    | Brand   | Phân khúc chi tiêu khách hàng                    |
| GET         | `/api/v1/me/dashboard/wardrobe/category-distribution`     | User    | Phân bổ danh mục tủ đồ cá nhân                   |
