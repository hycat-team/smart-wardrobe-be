# Research: Sửa Lỗi Dashboard

## Granularity Aggregation Strategy

| Decision | In-memory aggregation từ daily snapshots |
|----------|----------------------------------------|
| **Context** | Dynamic Query Engine cần aggregate daily snapshot data theo week/month |
| **Alternatives** | (1) GROUP BY trong SQL — phức tạp vì repository pattern dùng GORM query builder; (2) Bảng monthly snapshot riêng — out of scope; (3) In-memory aggregation — đơn giản, tận dụng dữ liệu đã có |
| **Why** | Không thay đổi DB schema, không cần migration, dễ test, code nằm gọn trong use case |
| **Implementation** | Đọc daily snapshots → group bằng map[yearMonth]/map[yearWeek] → aggregate: cumulative metrics = last value, non-cumulative = SUM, rate = AVG |

## Subscription Reader for Fallback (FR-003)

| Decision | Mở rộng contract của subscription module: tạo `ISubscriptionDashboardReader` |
|----------|-----------------------------------------------------------------------------|
| **Context** | Fallback path của `GetAdminKPICards` cần subscription revenue + PayOS stats từ live data |
| **Alternatives** | (1) Mở rộng `IUserDashboardReader` — userReader không có subscription data; (2) Tạo module `deposit` mới — overkill; (3) **Mở rộng subscription module contract** — đúng clean architecture, subscription module đã tồn tại |
| **Why** | `deposit` module không tồn tại trong dự án. Subscription module (`internal/modules/subscription/`) có sẵn, mở rộng contract là tự nhiên nhất |
| **Current state** | Tạo `internal/modules/subscription/presentation/contract/dashboard_reader.go` với interface `ISubscriptionDashboardReader` |

## Brand Item Lazy Load (FR-004)

| Decision | Lazy load on-demand — bỏ snapshot, query live data qua 3 modules (brand + wardrobe + fashion) |
|----------|----------------------------------------------------------------------------------------------|
| **Context** | `GetDigitalSampleLabAnalytics` hiện tại dùng worker-based snapshot `DashboardBrandItemDailySnapshot`. Không đáng tin cậy và stale |
| **Alternatives** | (1) Fallback khi snapshot miss — spec cũ; (2) **Lazy load hoàn toàn** — tasks.md, đúng hướng hơn |
| **Why** | Snapshot worker có thể không kịp chạy. Lazy load đảm bảo data luôn fresh. Pattern phù hợp với tính chất real-time của digital sample lab |
| **Implementation** | BrandReader.GetItemFeedbackSummary → WardrobeReader.GetOutfitItemFashionItemIDs → FashionContract.ListFashionItems → aggregate top 5 paired categories |

## MetricKey Constants (FR-005 / FR-007)

| Decision | Xoá `average_cost_per_wear_vnd`; thêm 7 brand keys |
|----------|-----------------------------------------------------|
| **Context** | Dynamic Query Engine switch case thiếu metric resolution |
| **New keys** | `vouchers_redeemed`, `digital_sample_votes_count`, `digital_sample_avg_rating`, `open_cs_tickets`, `cs_first_response_avg_seconds`, `points_expiring_30d`, `points_liability_value_vnd` |
| **Deleted keys** | `average_cost_per_wear_vnd` (field bị xoá khỏi DTO) |

## Error Handling Pattern (FR-006)

| Decision | Dùng `apperror.NewInternalError` — đúng chuẩn dự án |
|----------|-----------------------------------------------------|
| **Context** | `GetSystemOverview` đang swallow error từ sub-calls |
| **Pattern** | `apperror.NewInternalError("Không thể truy vấn dữ liệu {component}. Vui lòng thử lại sau.")` — handler nhận error → WrapHandler trả 500 |
