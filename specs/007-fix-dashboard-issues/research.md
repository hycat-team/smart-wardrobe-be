# Research: Sửa Lỗi Dashboard

## Granularity Aggregation Strategy

| Decision | In-memory aggregation từ daily snapshots |
|----------|----------------------------------------|
| **Context** | Dynamic Query Engine cần aggregate daily snapshot data theo week/month |
| **Alternatives** | (1) GROUP BY trong SQL — phức tạp vì repository pattern dùng GORM query builder; (2) Bảng monthly snapshot riêng — out of scope; (3) In-memory aggregation — đơn giản, tận dụng dữ liệu đã có |
| **Why** | Không thay đổi DB schema, không cần migration, dễ test, code nằm gọn trong use case |
| **Implementation** | Đọc daily snapshots → group bằng map[yearMonth]/map[yearWeek] → aggregate: cumulative metrics = last value, non-cumulative = SUM, rate = AVG |

## Subscription Reader for Fallback (FR-003)

| Decision | Inject reader interface qua Wire, không phải contract từ module mới |
|----------|---------------------------------------------------------------------|
| **Context** | Fallback path của `GetAdminKPICards` cần subscription revenue + PayOS stats từ live data |
| **Alternatives** | (1) Mở rộng `IUserDashboardReader` — userReader không có subscription data; (2) Tạo contract mới trong deposit module — pattern clean nhưng thêm module phụ thuộc; (3) **Inline data từ aggregator** — không khả thi vì aggregator đọc từ snapshot worker |
| **Why** | Chọn đọc từ contract đã có sẵn (deposit module hoặc subscription module) vì đúng clean architecture pattern |
| **Current state** | Check các module: `internal/modules/deposit/` — kiểm tra xem có presentation/contract/ không. Nếu chưa có, tạo interface `IDepositDashboardReader` |

## Brand Item Feedback Fallback (FR-004)

| Decision | Thêm method `GetItemFeedbackSummary` vào `IBrandDashboardReader` |
|----------|-----------------------------------------------------------------|
| **Context** | Fallback của `GetDigitalSampleLabAnalytics` cần feedback sentiment + paired categories cho từng item |
| **Alternatives** | (1) Đọc từ DB trực tiếp — vi phạm module boundary; (2) Query từ item snapshot repository — không có item-level fallback |
| **Why** | BrandReader đã có pattern cho brand-level stats; thêm per-item method là extension tự nhiên |

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
