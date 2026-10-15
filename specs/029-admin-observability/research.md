# Research: Khả năng quan sát cho admin (029)

> Phase 0 của `/speckit-plan`. Mọi quyết định dưới đây đã đối chiếu trực tiếp với mã nguồn
> tại thời điểm lập kế hoạch (2026-10-15). Số liệu file:line dùng để dev kiểm chứng lại.

## R1 — Chuẩn kỳ dữ liệu duy nhất: ngày lịch VN, cận nửa mở `[from, to)`

**Quyết định**: Mọi tham số HTTP `fromDate`/`toDate` là ngày lịch Việt Nam (`YYYY-MM-DD`), bao gồm
trọn cả hai ngày. Ở tầng application/repo, kỳ được biểu diễn bằng cặp nửa mở
`[from, to)` với `from = StartOfDayInVN(fromDate)` và `to = EndOfDayInVN(toDate)` (00:00 ngày kế
tiếp theo giờ VN). Mọi câu SQL lọc `created_at >= from AND created_at < to`.

- Helper parse + validate dùng chung cho cả hai service: `shared/pkg/utils/validation/date_range.go`
  → `ParseVNDateRangeQuery(fromDate, toDate string) (from, to time.Time, err error)`; lỗi trả
  `apperror.NewBadRequest` với thông báo tiếng Việt (định dạng sai; fromDate sau toDate). Cả hai
  service đều đã import `shared/pkg/utils/validation` nên không phát sinh phụ thuộc mới.

**Rationale**: Đây chính là convention mà các truy vấn snapshot đang dùng đúng
(`timeutils.VNDayRange` + `GetAdminAIUsageStats` tại `ai_cost_repo_impl.go:330` dùng
`>= start AND < end`). Hiện có ba biến thể lọc ngày khác nhau gây lỗi:
`GetRevenueBreakdown` tự cộng `+1d` (`deposit_transaction_repo_impl.go:249`),
`GetAIUsageByOperation`/`GetTotalAIRequestCount` dùng `<=` với mốc nửa đêm UTC
(`ai_cost_repo_impl.go:431,449`), `ListAdmin` deposit dùng `<=` với `time.Parse` UTC
(`deposit_transaction_repo_impl.go:48`). Một quy tắc duy nhất loại bỏ cả lớp lỗi lệch 7h.

**Alternatives considered**:
- Giữ `<=` inclusive: không sửa được lỗi "toDate bỏ sót gần hết ngày cuối" (FR-010/FR-014).
- Truyền `toDate+1d` ở từng repo: chính là tình trạng hiện tại (ba biến thể), tiếp tục lệch nhau.
- Parse giữ UTC rồi đổi múi giờ trong SQL (`AT TIME ZONE`): đẩy logic múi giờ xuống DB, khó test
  và không khớp convention `timeutils` sẵn có.

**Hệ quả triển khai** (các chỗ phải sửa theo chuẩn):
- `GetRevenueBreakdown`: bỏ `toDate.AddDate(0, 0, 1)` nội bộ; nhận thẳng cận nửa mở.
- `GetAIUsageByOperation`, `GetTotalAIRequestCount`, `GetAIErrorBreakdown` (mới nhận kỳ): đổi sang
  `>= from AND < to`.
- `DepositTransactionRepository.ListAdmin`: đổi `<=` → `<` với `to` là cận nửa mở.
- Handler commerce: thay 3 khối parse lặp trong `admin_dashboard_handler.go` bằng helper chung.

## R2 — Mở rộng proto theo hướng additive + cập nhật cả hai adapter

**Quyết định**: Mở rộng 2 file proto bằng field mới, không đổi số field cũ:

| File | Message | Field thêm |
|---|---|---|
| `contracts/subscription/v1/subscription.proto` | `AdminSubscriptionStats` | `string total_payos_success_value = 7;` |
| | `GetAdminSubscriptionStatsRequest` | `from_date = 2; to_date = 3;` (giữ `snapshot_date = 1` cho tương thích wire) |
| | `GetSubscriptionDistributionRequest` | `from_date = 1; to_date = 2;` |
| | `GetRenewalStatsRequest` | `from_date = 1; to_date = 2;` |
| `contracts/ai/v1/ai.proto` | `GetAIErrorBreakdownRequest` | `from_date = 1; to_date = 2;` |

Server gRPC: nếu `from_date`/`to_date` có mặt thì dùng cận mới; nếu không thì giữ hành vi cũ
(`snapshot_date` → `VNDayRange`; breakdown → all-time). Regen bằng `make buf-gen`.

Chuỗi cập nhật đầy đủ cho mỗi interface (theo mô hình dual adapter đang có):
1. Proto (+ regen `contracts/*/v1`).
2. Interface Go + in-process adapter: `services/account/contract/subscription/dashboard_reader.go`,
   `services/ai/contract/aiexecutor/dashboard_reader.go`.
3. gRPC client: `contracts/subscription/v1/client/dashboard_reader_client.go`,
   `contracts/ai/v1/client/dashboard_reader_client.go`.
4. gRPC server: `services/account/.../subscription/presentation/grpc/server.go`,
   `services/ai/.../aiexecutor/presentation/grpc/server.go`.
5. Transport mapper: `services/account/contract/subscription/transport/{mapper,to_proto}.go`,
   `services/ai/contract/aiexecutor/transport/{mapper,to_proto}.go`.

**Rationale**: Đúng nguyên tắc §II (contract qua gRPC, dual adapter) và §I (deploy độc lập) —
field additive để service cũ bỏ qua được.

**Alternatives considered**: Thêm RPC mới (`GetPayOSStatsRange`) — nhân đôi bề mặt contract cho
cùng một truy vấn SQL; đổi số field/thay kiểu — phá tương thích wire.

**Ghi chú rolling deploy**: client mới + server cũ sẽ nhận kết quả sai phạm vi (server bỏ qua
field mới) — chấp nhận cho dashboard nội bộ; thứ tự deploy khuyến nghị: account/ai trước commerce.

## R3 — API nhật ký hoạt động: thiết kế endpoint, DTO, truy vấn

**Quyết định**:
- Route `GET /api/v1/admin/activity-events`, nhóm mới trong
  `services/account/internal/modules/identity/presentation/router/router.go` với
  `AuthMiddleware.Handle()` + `RolesAuthorize(roleslug.Admin)`. Không đặt dưới `/admin/users/...`
  vì gin xung đột wildcard với `/admin/users/:id`.
- Query: `userId` (UUID, tuỳ chọn — sai định dạng 400), `activityType` (chỉ `login`|`refresh` theo
  `useractivitytype` — giá trị khác 400), `fromDate`/`toDate` (R1), `page`/`limit`
  (`shared_dto.PaginationQuery`, mặc định 20, trần 100).
- DTO mới `services/account/dto/identity/activity.go`:
  `ActivityEventRes{id, userId, activityType, occurredAt, createdAt, userEmail, userFirstName, userLastName}`
  (camelCase; `userFirstName`/`userLastName` là con trỏ có `omitempty` khi NULL);
  `ActivityEventListRes = shared_dto.PaginationResult[*ActivityEventRes]`.
  Tên trường khớp quy ước `UserRes` (`email`, `firstName`, `lastName` — `dto/identity/profile.go:12`).
- Repo: thêm `ListAdmin(ctx, filter, page, limit) ([]AdminActivityEventRow, int64, error)` vào
  `IUserActivityRepository`; readmodel mới trong `identity/domain/repositories/readmodels.go`.
  SQL một câu SELECT + một câu COUNT cùng điều kiện:
  `LEFT JOIN users u ON u.id = e.user_id` — LEFT để sự kiện không biến mất nếu dòng user bị
  soft-delete; danh tính vẫn đọc được (soft delete giữ dòng). Sắp xếp `occurred_at DESC, id DESC`
  (tie-breaker ổn định cho phân trang).
- Usecase: thêm `ListActivityEvents` vào `IUserUseCase`, implement trong file mới
  `identity/application/usecase/activity_uc.go` (giữ `user_uc.go`/`admin_uc.go` dưới ngưỡng
  code-hygiene). Không đổi constructor → không phát sinh binding Wire mới.
- Handler: thêm method `ListActivityEvents` vào `AdminHandler` (không đổi constructor/DI); nếu
  `admin_handler.go` vượt ngưỡng độ dài thì đặt trong file mới `activity_handler.go` cùng package.
- Swagger annotation + tài liệu mới `docs/api/identity/admin-activity-api.md` + cập nhật
  `docs/api/README.md`.

**Rationale**: Bảng `identity.user_activity_events` thuộc identity module (§I/§II — không join
chéo service). Một endpoint duy nhất đủ cho US1; lọc ở SQL để phân trang đúng.

**Alternatives considered**:
- Nested user object trong response: FE phải xử lý object lồng; flat khớp `UserRes` hiện có.
- `INNER JOIN`: ẩn sự kiện của user soft-deleted — trái edge case "dữ liệu lịch sử không bị ẩn".
- Usecase/handler riêng: thêm binding Wire mà không thêm giá trị.

## R4 — Trạng thái giao dịch dạng chuỗi: KHÔNG MarshalJSON toàn cục

**Quyết định**: Không thêm `MarshalJSON` cho type `depositstatus.DepositStatus` vì type này còn
xuất hiện trong DTO user-facing `PaymentLinkDTO.PaymentStatus` (`services/account/dto/subscription/billing_dto.go:66`)
— đổi sẽ phá client user (ngoài phạm vi). Thay vào đó:
- Đổi `DepositTransactionRes.Status` (`billing_dto.go:157`) sang `string`.
- Mapper mới `services/account/internal/modules/subscription/application/mapper/deposit_status.go`:
  `DepositStatusToString(depositstatus.DepositStatus) string` với đủ 11 giá trị
  (`pending`, `success`, `failed_legacy`, `creating`, `reconciliation_required`, `reconciling`,
  `creation_failed`, `cancelled`, `expired`, `investigation_required`, `refunded`).
- `parseDepositStatus` trong `admin_billing_handler.go:370` mở rộng nhận `failed_legacy` (khứ hồi
  đủ 11 giá trị theo FR-008).

**Rationale**: Cô lập thay đổi trong DTO admin; `PaymentLinkDTO` giữ số nguyên như cũ.

**Alternatives considered**: `MarshalJSON` trên type — phá user-facing contract; mapper chuỗi ở
tầng handler — vi phạm quy ước "mapper nằm trong `application/mapper`".

## R5 — `GET /transactions/{id}` trả DTO camelCase dùng chung mapper với list

**Quyết định**: Giữ usecase trả entity (`IAdminBillingUseCase.GetTransaction` không đổi chữ ký),
handler map sang DTO qua mapper mới `mapper.DepositTransactionToRes(tx)`; refactor luôn khối build
DTO inline trong `ListTransactions` (`admin_billing_handler.go:99-127`) dùng cùng mapper này.

**Rationale**: Một nguồn mapping duy nhất → list và detail không thể lệch shape (FR-007);
không đổi interface usecase (ít rủi ro hơn).

**Alternatives considered**: Đổi usecase trả DTO — churn chữ ký + mock tests không cần thiết.

## R6 — payOS: phạm vi hai loại giao dịch + tổng giá trị thật + theo kỳ

**Quyết định**:
- `GetAdminSubscriptionStats` đổi chữ ký thành `(ctx, from, to time.Time)` (cận nửa mở R1). Ba call site:
  worker snapshot (`aggregator_uc_impl.go`, `VNDayRange(today)` — hành vi ngày không đổi); KPI
  fallback khi chưa có snapshot (`admin_dashboard_uc_impl.go:65`, đổi thành `VNDayRange(now)` —
  giữ hành vi ngày); payOS mới của commerce (kỳ người dùng chọn, mặc định hôm nay).
- SQL (`deposit_transaction_repo_impl.go:193-205`) sửa:
  - Doanh thu `total_subscription_revenue` GIỮ `transaction_type = direct_purchase`
    (doanh thu gói cước — không đổi ngữ nghĩa `NetAIMargin`).
  - Bỏ filter `transaction_type` cho 5 số đếm payOS (cả `direct_purchase` + `wallet_topup`).
  - Thêm `total_payos_success_value = COALESCE(SUM(amount) FILTER (WHERE status = success), 0)`
    trên cùng phạm vi hai loại.
- `AdminSubscriptionStatsDTO` (account readmodel) thêm `TotalPayOSSuccessValue decimal.Decimal`;
  proto thêm field 7 (R2); mapper transport parse/format string.
- Commerce: `GetPayOSReconciliation(ctx, from, to)` gọi reader với kỳ; handler parse ngày; mặc
  định khi không gửi = hôm nay VN (`VNDayRange(NowVN())`). `overview` truyền zero → mặc định hôm
  nay. `PayOSReconciliationRes.TotalSuccessValue` lấy từ field mới (KHÔNG còn mượn
  `SubscriptionRevenueVND` — `admin_dashboard_uc_impl.go:211`).
- Snapshot worker: luồng không đổi; `total_payos_success_value` không lưu vào bảng snapshot
  (không migration). Hệ quả: cột đếm payOS trong snapshot mới (và KPI card `payOSSuccessCount`)
  mang phạm vi hai loại — đây là dữ liệu chết trên FE, ghi chú trong guide.

**Rationale**: Đúng FR-017 + clarify Q2/Q3; giữ doanh thu subscription tách khỏi giá trị payOS.

**Alternatives considered**: Thêm bảng/cột snapshot cho giá trị — migration không cần thiết vì
panel chuyển sang truy vấn theo kỳ.

## R7 — Phân bổ theo gói: giao cửa sổ gói hiện tại với kỳ lọc

**Quyết định**: `GetDistributionByPlanCode(ctx, from, to)`:
- Zero cả hai → giữ nguyên query hiện trạng (`expires_at IS NULL OR expires_at > now()`).
- Có kỳ → `WHERE started_at < to AND (expires_at IS NULL OR expires_at >= from)` nhóm theo
  `current_plan_code`. Gói FREE (`expires_at IS NULL`) hiện diện mọi kỳ.

Chuỗi thay đổi: repo → usecase → contract → gRPC server → proto (R2) → client → commerce
usecase/handler.

**Rationale**: Đúng clarify Q1 (giao cửa sổ); dữ liệu sẵn có (`started_at`, `expires_at`).

**Giới hạn đã chấp nhận** (ghi vào guide + deferred nếu cần): lịch sử đổi gói không tái dựng
được từ `user_subscriptions` — user chỉ tính theo gói hiện tại.

## R8 — Hiệu quả gia hạn theo kỳ: lọc theo `created_at` của attempt

**Quyết định**: `GetRenewalStats(ctx, from, to)` thêm `created_at >= from AND created_at < to` khi
có kỳ (bảng `subscription_renewal_attempts` có `AuditableEntity.CreatedAt`); giữ điều kiện
`status <> 'processing'` và cách tính success rate hiện có.

**Rationale**: "Sự kiện phát sinh trong kỳ" = thời điểm tạo attempt. Chọn `created_at` (không
`completed_at`) vì attempt in-flight trong kỳ vẫn thuộc kỳ đó; attempt tạo trước kỳ hoàn tất
trong kỳ không tính — ghi rõ trong guide.

**Alternatives considered**: `completed_at` — bỏ sót attempt đang xử lý và attempt tạo trong kỳ
nhưng chưa xong (không đếm được); `COALESCE(completed_at, created_at)` — quy tắc khó giải thích.

## R9 — Tỷ lệ lỗi AI cùng kỳ + mặc định 30 ngày

**Quyết định**:
- AI `GetAIErrorBreakdown(ctx, from, to)`: SQL thêm range khi có (R1); zero → all-time (tương
  thích hành vi cũ cho caller cũ).
- Commerce `GetAIErrorBreakdown(ctx, from, to)`: mặc định khi zero = 30 ngày gần nhất VN
  (`StartOfDayInVN(NowVN().AddDate(0,0,-30))` → `EndOfDayInVN(NowVN())`); `totalErrors` =
  Σ `breakdown.errorCount`; `totalRequests` gọi `GetTotalAIRequestCount` cùng kỳ; `errorRate` =
  errors/requests×100, bằng 0 khi mẫu 0. Phân nhóm lỗi cùng kỳ ⇒ tổng khớp (FR-016).

**Rationale**: Sửa trực tiếp bug trộn all-time/30 ngày (`admin_dashboard_uc_impl.go:325-328`);
mặc định 30 ngày giữ trải nghiệm panel hiện tại.

**Alternatives considered**: Trả thêm trường riêng cho "lỗi all-time" — không ai cần.

## R10 — Danh sách giao dịch: sửa kỳ + nhận `failed_legacy`

**Quyết định**: `ListTransactions` parse qua helper R1 (400 khi sai/đảo); filter
`AdminDepositTransactionFilter` đổi nghĩa `ToDate` thành cận nửa mở; repo `ListAdmin` đổi
`<=` → `<`; `parseDepositStatus` nhận thêm `failed_legacy` (R4). Before→after ghi trong guide
(FR-010, SC-005).

## R11 — Tài liệu, Swagger, guide, deferred, test

**Quyết định**:
- Cập nhật `docs/api/subscription/admin-billing-api.md` (status chuỗi, kỳ chuẩn, failed_legacy),
  `docs/api/dashboard/dashboard-admin-api.md` (tham số kỳ + phạm vi payOS + error-rate),
  tạo `docs/api/identity/admin-activity-api.md`, cập nhật `docs/api/README.md`.
- `make swagger-all` + `make buf-gen` (regen là phần của task, không commit tay file sinh).
- `specs/029-admin-observability/frontend-guide.md` (FR-018..020) + `deferred.md` (FR-021).
- Test theo pattern sẵn có:
  - Account: SQL pinning theo `admin_stats_query_sql_test.go` (unreachable dialector) cho
    distribution overlap + payOS stats; behavior test cho activity repo/usecase/handler; mapper
    status string; parse range 400.
  - Commerce: `admin_dashboard_uc_test.go` (mocks) cho payOS kỳ/phạm vi, error-rate, distribution
    passthrough; handler validation.
  - AI: repo test cho range mới.
  - Transport mapper tests (`mapper_test.go` hai phía) cập nhật field mới.
- Gate: `make lint` + `go test -race -count=3` scoped theo phase; `make check` trước push.

## R12 — Không migration, không bảng/cột mới

**Quyết định**: Toàn bộ thay đổi là read-side (truy vấn) + serialization + proto. Không có
migration Goose nào. Bảng `dashboard_admin_daily_snapshots` giữ nguyên cấu trúc.

**Rationale**: FR-001..FR-022 đều đọc/đổi hợp đồng; không cần lưu thêm dữ liệu.
