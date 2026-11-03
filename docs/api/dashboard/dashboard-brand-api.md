# Dashboard Brand API

> **Tham chiếu brand (`brandId` query):** chấp nhận **UUID hoặc slug** của brand. Dashboard frontend truyền slug từ URL `/brand-portal/{slug}`; UUID vẫn hợp lệ để tương thích ngược.

Tài liệu đặc tả các API Dashboard dành cho **Brand Manager (B2B Portal)**. Bao gồm các endpoints từ dashboard-api.md và dashboard-api-v2.md.

Xem thêm hằng số tại [constants.md](constants.md).

---

## 1. Lấy thẻ KPI Brand

- **Endpoint:** `GET /api/v1/brand/dashboard/kpi-cards`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Query Parameters:**
    - `brandId` (string, **required**) — UUID của nhãn hàng.
- **Response struct:** `BrandKPICardsRes`

| Field                     | JSON                        | Kiểu Go   | Ý nghĩa                                       |
| ------------------------- | --------------------------- | --------- | --------------------------------------------- |
| BrandID                   | `brandId`                   | uuid.UUID | ID của nhãn hàng                              |
| TotalMembers              | `totalMembers`              | int       | Tổng số hội viên loyalty                      |
| PointsIssued              | `pointsIssued`              | int       | Số điểm đã phát hành                          |
| PointsRedeemed            | `pointsRedeemed`            | int       | Số điểm đã quy đổi                            |
| ActiveVouchers            | `activeVouchers`            | int       | Số voucher đang hoạt động                     |
| VouchersRedeemed          | `vouchersRedeemed`          | int       | Số voucher đã quy đổi                         |
| DigitalSampleVotesCount   | `digitalSampleVotesCount`   | int       | Số lượt vote mẫu thử Digital Sample           |
| DigitalSampleAvgRating    | `digitalSampleAvgRating`    | float64   | Điểm đánh giá trung bình (thang 0–5)          |
| OpenCSTickets             | `openCsTickets`             | int       | Số ticket CS đang mở                          |
| CSFirstResponseAvgSeconds | `csFirstResponseAvgSeconds` | int       | Thời gian phản hồi đầu tiên trung bình (giây) |
| UpdatedAt                 | `updatedAt`                 | time.Time | Thời điểm cập nhật gần nhất                   |

---

## 2. Phân tích nợ điểm hết hạn

- **Endpoint:** `GET /api/v1/brand/dashboard/financials/points-liability`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Query Parameters:**
    - `brandId` (string, **required**) — UUID của nhãn hàng.
- **Cơ chế Column-Level RBAC:** Tài khoản `Staff` bị redact trường `pointsLiabilityValueVnd` (trả về `0`).
- **Response struct:** `BrandPointsLiabilityRes`

| Field                   | JSON                      | Kiểu Go   | Ý nghĩa                                        |
| ----------------------- | ------------------------- | --------- | ---------------------------------------------- |
| BrandID                 | `brandId`                 | uuid.UUID | ID của nhãn hàng                               |
| PointsExpiring30d       | `pointsExpiring30d`       | int       | Số điểm sắp hết hạn trong 30 ngày              |
| PointsLiabilityValueVND | `pointsLiabilityValueVnd` | float64   | Giá trị quy đổi (VND). **Staff luôn thấy `0`** |

---

## 3. Phân tích đánh giá mẫu thử Digital Sample Lab

- **Endpoint:** `GET /api/v1/brand/dashboard/operations/digital-sample-lab/:itemId`
- **Tác nhân (Actor):** Thành viên nhãn hàng (`brandmemberrole.Owner` / `Staff`).
- **Path Parameters:**
    - `itemId` (string, **required**) — UUID sản phẩm mẫu thử.
- **Query Parameters:**
    - `brandId` (string, **required**) — UUID của nhãn hàng.
- **Response struct:** `DigitalSampleLabAnalyticsRes`

| Field               | JSON                  | Kiểu Go              | Ý nghĩa                                            |
| ------------------- | --------------------- | -------------------- | -------------------------------------------------- |
| ItemID              | `itemId`              | uuid.UUID            | ID của mẫu thử                                     |
| VotesCount          | `votesCount`          | int                  | Số lượt vote                                       |
| AvgRating           | `avgRating`           | float64              | Điểm rating trung bình (0–5)                       |
| FeedbackSentiment   | `feedbackSentiment`   | FeedbackSentiment    | Sentiment: `"positive"`, `"neutral"`, `"negative"` |
| TopPairedCategories | `topPairedCategories` | []PairedCategoryStat | Danh sách danh mục thường phối kèm                 |

**Cấu trúc `PairedCategoryStat`:**

| Field        | JSON           | Kiểu Go | Ý nghĩa                                   |
| ------------ | -------------- | ------- | ----------------------------------------- |
| CategoryName | `categoryName` | string  | Tên danh mục thời trang (VD: "Quần Jean") |
| PairCount    | `pairCount`    | int     | Số lần xuất hiện trong các bộ phối đồ     |

---

## 4. Lấy thống kê kênh thu hút khách hàng

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

## 5. Lấy phân bổ hạng loyalty thành viên

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

## 6. Lấy dự báo điểm sắp hết hạn

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

## 7. Lấy thống kê đổi benefit

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

## 8. Lấy thống kê catalog sản phẩm

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

## 9. Lấy phân khúc chi tiêu khách hàng

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

## 10. Truy vết đánh giá mẫu thử (Brand)

- **Endpoint:** `GET /api/v1/brand/dashboard/drilldown/sample-lab-feedbacks`
- **Tác nhân (Actor):** Thành viên nhãn hàng / Quản trị viên.
- **Query Parameters:**
    - `itemId` (string, **required**) — UUID sản phẩm mẫu thử.
    - `page` (int, optional, default=1) — Số trang.
    - `limit` (int, optional, default=20, max=100) — Số bản ghi mỗi trang.
- **Response struct:** `DrillDownSampleFeedbackListRes` (alias của `PaginationResult[DrillDownSampleFeedbackDTO]`)

**Cấu trúc `DrillDownSampleFeedbackDTO`:**

| Field      | JSON         | Kiểu Go   | Ý nghĩa                    |
| ---------- | ------------ | --------- | -------------------------- |
| FeedbackID | `feedbackId` | string    | ID của feedback            |
| ItemID     | `itemId`     | uuid.UUID | ID sản phẩm mẫu thử        |
| UserID     | `userId`     | uuid.UUID | ID người dùng gửi feedback |
| Rating     | `rating`     | float64   | Điểm đánh giá (0–5)        |
| Comment    | `comment`    | string    | Nội dung nhận xét          |
| CreatedAt  | `createdAt`  | time.Time | Thời điểm gửi              |

---

## Phụ lục: Trace toàn bộ Metric — Brand

### Brand KPI Cards (`GET /api/v1/brand/dashboard/kpi-cards`)

| Field                       | Nguồn                        | Câu truy vấn                                                                                                                                           |
| --------------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `totalMembers`              | `brand_customers`            | `SELECT COUNT(*) FROM brand_customers WHERE brand_id=? AND status='ACTIVE'`                                                                            |
| `pointsIssued`              | `loyalty_point_transactions` | `SELECT COALESCE(SUM(points),0) FROM loyalty_point_transactions WHERE transaction_type IN ('EARN','CREDIT','ISSUED') AND brand_id=?`                   |
| `pointsRedeemed`            | `loyalty_point_transactions` | `SELECT COALESCE(SUM(points),0) FROM loyalty_point_transactions WHERE transaction_type IN ('REDEEM','DEBIT','REDEEMED') AND brand_id=?`                |
| `activeVouchers`            | `brand_benefits`             | `SELECT COUNT(*) FROM brand_benefits WHERE brand_id=? AND status IN ('ACTIVE','active')`                                                               |
| `vouchersRedeemed`          | `benefit_redemptions`        | `SELECT COUNT(*) FROM benefit_redemptions WHERE brand_id=?`                                                                                            |
| `digitalSampleVotesCount`   | `digital_sample_responses`   | `SELECT COUNT(*) FROM digital_sample_responses JOIN brand_items ON brand_items.id=digital_sample_responses.brand_item_id WHERE brand_items.brand_id=?` |
| `digitalSampleAvgRating`    | `digital_sample_responses`   | `SELECT COALESCE(AVG(rating),0) FROM digital_sample_responses [cùng join] WHERE rating IS NOT NULL`                                                    |
| `openCsTickets`             | `brand_conversations`        | `SELECT COUNT(*) FROM brand_conversations WHERE brand_id=? AND status='OPEN'`                                                                          |
| `csFirstResponseAvgSeconds` | `brand_conversations`        | `SELECT COALESCE(AVG(first_response_seconds),0) FROM brand_conversations WHERE first_response_seconds IS NOT NULL`                                     |

### Points Liability (`GET /api/v1/brand/dashboard/financials/points-liability`)

| Field                     | Nguồn                | Câu truy vấn                                                                                                                                      |
| ------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pointsExpiring30d`       | `loyalty_point_lots` | `SELECT COALESCE(SUM(remaining_points),0) FROM loyalty_point_lots WHERE brand_id=? AND status='ACTIVE' AND expires_at BETWEEN [today, today+30d]` |
| `pointsLiabilityValueVnd` | Tính toán            | `pointsExpiring30d * 1000` (giả định 1 điểm = 1000 VND)                                                                                           |

### Dynamic Query — Brand Metrics (`POST /api/v1/dashboard/analytics/query`)

| Metric                       | Mapped từ field                    | Kiểu Go gốc   |
| ---------------------------- | ---------------------------------- | ------------- |
| `"total_members"`            | `snapshot.TotalMembers`            | int → float64 |
| `"points_issued"`            | `snapshot.PointsIssued`            | int → float64 |
| `"points_redeemed"`          | `snapshot.PointsRedeemed`          | int → float64 |
| `"active_vouchers"`          | `snapshot.ActiveVouchers`          | int → float64 |
| `"vouchers_redeemed"`        | `snapshot.VouchersRedeemed`        | int → float64 |
| `"digital_sample_votes_count"` | `snapshot.DigitalSampleVotesCount` | int → float64 |
| `"digital_sample_avg_rating"`  | `snapshot.DigitalSampleAvgRating`  | float64       |
| `"open_cs_tickets"`          | `snapshot.OpenCSTickets`           | int → float64 |
| `"cs_first_response_avg_seconds"` | `snapshot.CSFirstResponseAvgSeconds` | int → float64 |
| `"points_expiring_30d"`      | `snapshot.PointsExpiring30d`       | int → float64 |
| `"points_liability_value_vnd"` | `snapshot.PointsLiabilityValueVND` | float64       |
| `"new_customers"`            | `brand_customers`                  | Live query    |
| `"customer_claim_rate"`      | `brand_customers`                  | Live query    |
| `"benefit_redemption_count"` | `benefit_redemptions`              | Live query    |
| `"brand_item_count"`         | `brand_items`                      | Live query    |
| `"brand_item_count_by_type"` | `brand_items`                      | Live query    |

### Dynamic Query — Brand New Metrics (Live Query)

| Metric                     | Nguồn                 | Cách tính                                                | Mô tả                               |
| -------------------------- | --------------------- | -------------------------------------------------------- | ----------------------------------- |
| `new_customers`            | `brand_customers`     | `COUNT(*) WHERE created_at BETWEEN fromDate AND toDate`  | Số khách hàng mới trong kỳ          |
| `customer_claim_rate`      | `brand_customers`     | `claimed_at IS NOT NULL / total * 100`                   | Tỷ lệ khách hàng đã claim tài khoản |
| `benefit_redemption_count` | `benefit_redemptions` | `COUNT(*) WHERE redeemed_at BETWEEN fromDate AND toDate` | Số lượt đổi benefit trong kỳ        |
| `brand_item_count`         | `brand_items`         | `COUNT(*) WHERE brand_id = ?`                            | Tổng số sản phẩm trong catalog      |
| `brand_item_count_by_type` | `brand_items`         | `SELECT item_type, COUNT(*) GROUP BY item_type`          | Số sản phẩm theo từng loại          |

---

## Tổng hợp Endpoint — Brand

| Phương thức | Endpoint                                                          | Vai trò      | Chức năng                                        |
| ----------- | ----------------------------------------------------------------- | ------------ | ------------------------------------------------ |
| GET         | `/api/v1/brand/dashboard/kpi-cards`                               | Brand Member | Thẻ KPI Brand                                    |
| GET         | `/api/v1/brand/dashboard/financials/points-liability`             | Brand Member | Nợ điểm hết hạn (RBAC)                           |
| GET         | `/api/v1/brand/dashboard/operations/digital-sample-lab/:itemId`   | Brand Member | Phân tích mẫu thử                                |
| GET         | `/api/v1/brand/dashboard/customers/acquisition`                   | Brand Member | Thống kê kênh thu hút khách hàng                 |
| GET         | `/api/v1/brand/dashboard/loyalty/tier-distribution`               | Brand Member | Phân bổ hạng loyalty thành viên                  |
| GET         | `/api/v1/brand/dashboard/loyalty/point-expiry-forecast`           | Brand Member | Dự báo điểm sắp hết hạn                          |
| GET         | `/api/v1/brand/dashboard/benefits/redemption-analytics`           | Brand Member | Thống kê top benefit được đổi nhiều nhất         |
| GET         | `/api/v1/brand/dashboard/operations/catalog-stats`                | Brand Member | Thống kê catalog sản phẩm                        |
| GET         | `/api/v1/brand/dashboard/customers/spend-segmentation`            | Brand Member | Phân khúc chi tiêu khách hàng                    |
| GET         | `/api/v1/brand/dashboard/drilldown/sample-lab-feedbacks`          | Brand Member | Drill-down feedback                               |

---

## Brand analytics dynamic query (slug support)

- `POST /api/v1/dashboard/analytics/query` nhận `brandId` là **UUID hoặc slug** trong body JSON.
- Response lỗi rõ ràng: slug không tồn tại → `404`; không phải thành viên brand → `403`.
- `GET /api/v1/brand/dashboard/drilldown/sample-lab-feedbacks?brandId=...&itemId=...`: `brandId` dùng để xác thực quyền và kiểm tra item thuộc brand (cross-brand trả `404`).
- `GET /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId}?brandId=...`: trả số liệu 0 thay vì `404` khi mẫu thử chưa có feedback.
