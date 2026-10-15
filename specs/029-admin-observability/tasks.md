# Tasks: Khả năng quan sát cho admin (029)

**Input**: Design documents từ `specs/029-admin-observability/` (spec.md, plan.md, research.md R1–R12, data-model.md, contracts/admin-api-delta.md, contracts/grpc-contract-delta.md, quickstart.md)

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: FR-022 yêu cầu kiểm thử tự động → mỗi user story có test task đi kèm; viết test TRƯỚC, đảm bảo FAIL (kể cả fail biên dịch khi chữ ký chưa đổi) trước khi implement.

**Organization**: Tasks nhóm theo user story để implement/test độc lập từng story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Chạy song song được (khác file, không phụ thuộc task chưa xong)
- **[Story]**: Story nào (US1..US5)
- Mọi task ghi đường dẫn file chính xác

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Chốt baseline xanh + xác nhận index trước khi sửa

- [x] T001 [P] Chạy baseline trước khi sửa: `make lint` + `go test -race -count=1` scoped `./services/account/... ./services/commerce/... ./services/ai/... ./shared/...` — ghi lại số package FAIL (kỳ vọng 0); nếu có FAIL sẵn thì báo dev trước khi bắt đầu (kết quả 2026-10-15: lint xanh; shared 28 ok / account 33 ok / commerce 31 ok / ai 6 ok — 0 FAIL)
- [x] T002 [P] Kiểm tra index dev DB cho các truy vấn kỳ mới (`\d identity.user_activity_events`, `\d subscription.deposit_transactions`, `\d subscription.subscription_renewal_attempts`, `\d aiexecutor.ai_usage_events`); nếu truy vấn bounded-theo-kỳ không có index phù hợp → ghi vào `specs/029-admin-observability/deferred.md` (mục "index bổ sung"), KHÔNG thêm index trong feature này (kết quả: thiếu index `created_at` trên deposit_transactions/renewal_attempts; ai_usage_events chỉ có composite — đã ghi D-01 vào deferred.md)

**Checkpoint**: baseline xanh + biết rõ tình trạng index

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Helper kỳ VN dùng chung + proto additive — chặn US2/US3/US4

**⚠️ CRITICAL**: Không task US3/US4 nào bắt đầu trước khi phase này xong; US1 và US2 chỉ cần T003 (helper kỳ) — có thể chạy song song ngay sau T003, không cần chờ proto T004–T006

- [x] T003 [P] Viết helper `ParseVNDateRangeQuery(fromDate, toDate string) (from, to time.Time, err error)` + unit test trong `shared/pkg/utils/validation/date_range.go` và `shared/pkg/utils/validation/date_range_test.go`: trả cặp nửa mở `[from, to)` với `from = StartOfDayInVN`, `to = EndOfDayInVN`; đầu thiếu trả zero-time (repo bỏ điều kiện tương ứng — gửi một đầu thì đầu còn lại để mở); 400 `apperror` tiếng Việt chỉ khi định dạng sai hoặc khi gửi cả hai mà `fromDate > toDate`; case biên: rỗng cả hai, chỉ một đầu, mùng 1, cuối tháng (R1, data-model §3)
- [x] T004 [P] Mở rộng `contracts/subscription/v1/subscription.proto` (additive): `AdminSubscriptionStats` + `string total_payos_success_value = 7;`; `GetAdminSubscriptionStatsRequest` + `from_date = 2; to_date = 3;` (giữ `snapshot_date = 1`); `GetSubscriptionDistributionRequest` + `from_date = 1; to_date = 2;`; `GetRenewalStatsRequest` + `from_date = 1; to_date = 2;` (grpc-contract-delta §1)
- [x] T005 [P] Mở rộng `contracts/ai/v1/ai.proto` (additive): `GetAIErrorBreakdownRequest` + `from_date = 1; to_date = 2;` (grpc-contract-delta §2)
- [x] T006 Chạy `make buf-gen` regen stub cho cả hai hợp đồng; xác nhận build xanh (chưa map field mới ở bước này) — phụ thuộc T004, T005

**Checkpoint**: helper kỳ có test; proto regen xanh, chưa consumer nào đọc field mới

---

## Phase 3: User Story 1 — Admin tra cứu nhật ký hoạt động người dùng (Priority: P1) 🎯 MVP

**Goal**: `GET /api/v1/admin/activity-events` phân trang + lọc userId/activityType/kỳ, kèm email + tên người dùng

**Independent Test**: Kịch bản 1 `quickstart.md` (login/refresh 2 tài khoản → gọi API có/không lọc, đối chiếu; 400/403 đúng)

### Tests cho User Story 1 ⚠️

> Viết test TRƯỚC, đảm bảo FAIL trước khi implement

- [x] T007 [P] [US1] Repo test trong `services/account/internal/modules/identity/infrastructure/persistence/user_activity_repo_impl_test.go` (mới): SQL pinning theo pattern `admin_stats_query_sql_test.go` cho `ListAdmin` — `LEFT JOIN users`, `ORDER BY occurred_at DESC, id DESC`, đủ 3 điều kiện lọc, COUNT cùng điều kiện; behavior (sqlite nếu chạy được): user soft-deleted vẫn hiện danh tính
- [x] T008 [P] [US1] Usecase test trong `services/account/internal/modules/identity/application/usecase/activity_uc_test.go` (mới): `userId` sai UUID → 400; `activityType` ngoài `{login, refresh}` → 400; kỳ sai/đảo → 400 (qua T003); phân trang chuẩn hoá; map row → DTO (firstName/lastName nil thì vắng mặt)
- [x] T009 [P] [US1] Handler/router test trong `services/account/internal/modules/identity/presentation/handler/activity_handler_test.go` (mới, pattern `campaign_admin_router_test.go`): route `/api/v1/admin/activity-events` yêu cầu admin (401/403), 200 đúng shape `PaginationResult`, query truyền xuống usecase; assert FR-005 chỉ đọc — route chỉ đăng ký GET, không có method ghi

### Implementation cho User Story 1

- [x] T010 [P] [US1] DTO mới `services/account/dto/identity/activity.go`: `GetActivityEventsQueryReq` (nhúng `shared_dto.PaginationQuery` + `userID`, `activityType`, `fromDate`, `toDate`) + `ActivityEventRes` (`id, userId, activityType, occurredAt, createdAt, userEmail, userFirstName?, userLastName?`) + `ActivityEventListRes = shared_dto.PaginationResult[*ActivityEventRes]`
- [x] T011 [P] [US1] Mở rộng repo interface + readmodel: thêm `ListAdmin(ctx, filter, page, limit)` vào `services/account/internal/modules/identity/domain/repositories/user_activity_repo.go` và `AdminActivityEventRow` vào `.../domain/repositories/readmodels.go` (data-model §1)
- [x] T012 [US1] Implement `ListAdmin` trong `services/account/internal/modules/identity/infrastructure/persistence/user_activity_repo_impl.go` — phụ thuộc T007, T011
- [x] T013 [US1] Usecase: thêm `ListActivityEvents` vào `services/account/internal/modules/identity/application/interface/usecase/user_uc.go` + implement method trên `UserUseCase` trong `.../application/usecase/activity_uc.go` (mới) + mapper `.../application/mapper/activity_mapper.go` (mới) — phụ thuộc T010, T012
- [x] T014 [US1] Handler + route: method `ListActivityEvents` trên `AdminHandler` trong `services/account/internal/modules/identity/presentation/handler/activity_handler.go` (mới, swagger annotation) + group `/admin/activity-events` trong `.../presentation/router/router.go` (auth + `RolesAuthorize(roleslug.Admin)`) — phụ thuộc T013
- [x] T015 [US1] Docs + swagger: tạo `docs/api/identity/admin-activity-api.md` + cập nhật mục lục `docs/api/README.md` + chạy `make swagger svc=account` — phụ thuộc T014
- [ ] T016 [US1] Verify Kịch bản 1 `quickstart.md` live trên dev (login/refresh 2 tài khoản; lọc đủ 3 điều kiện; 400/403) + gate: `make lint` + `go test -race -count=3` scoped `./services/account/... ./shared/...`

**Checkpoint**: US1 chạy độc lập — admin tra được lịch sử hoạt động kèm danh tính, không cần tra chéo

---

## Phase 4: User Story 2 — Admin xem và xử lý giao dịch payOS qua contract dùng được (Priority: P1)

**Goal**: `GET /transactions/{id}` trả DTO camelCase; `status` là chuỗi 11 giá trị (khứ hồi filter); kỳ ngày của list chuẩn VN + validate 400

**Independent Test**: Kịch bản 2 `quickstart.md` (status chuỗi, detail cùng shape list, `failed_legacy` khứ hồi, biên 23:30 VN, 400)

### Tests cho User Story 2 ⚠️

- [x] T017 [P] [US2] Mapper test trong `services/account/internal/modules/subscription/application/mapper/deposit_status_test.go` + `deposit_transaction_test.go` (mới): round-trip 11 giá trị `DepositStatusToString` ↔ `ParseDepositStatus` (cả hai sống trong package mapper — T020); `DepositTransactionToRes` đủ bộ trường camelCase, `status` là string
- [x] T018 [P] [US2] SQL pinning trong `services/account/internal/modules/subscription/infrastructure/persistence/deposit_transaction_query_sql_test.go` (mới, pattern unreachable dialector): `ListAdmin` so `created_at >= from AND created_at < to` (cận nửa mở), `status` filter không đổi, `ORDER BY created_at DESC`
- [x] T019 [P] [US2] Handler test trong `services/account/internal/modules/subscription/presentation/handler/admin_billing_handler_test.go` (mới): detail trả DTO camelCase (không còn `OrderCode`/`UserID` Go fields) + `status` chuỗi; list parse kỳ VN → filter nhận cận `[from, to)` đúng (case 23:30 VN vẫn trong kỳ); ngày sai/đảo → 400 tiếng Việt; `status=failed_legacy` parse được

### Implementation cho User Story 2

- [x] T020 [P] [US2] Mapper mới `services/account/internal/modules/subscription/application/mapper/deposit_status.go` (`DepositStatusToString` đủ 11 giá trị + `ParseDepositStatus` chuyển từ handler package sang, export) + `.../mapper/deposit_transaction.go` (`DepositTransactionToRes`)
- [x] T021 [US2] Đổi DTO: `Status` trong `DepositTransactionRes` thành `string` trong `services/account/dto/subscription/billing_dto.go` — KHÔNG đụng `PaymentLinkDTO.PaymentStatus` (user-facing, giữ số; R4) — phụ thuộc T020
- [x] T022 [US2] Handler `services/account/internal/modules/subscription/presentation/handler/admin_billing_handler.go`: detail map qua `DepositTransactionToRes`; list dùng cùng mapper (bỏ khối build inline); parse `fromDate/toDate` qua `validation.ParseVNDateRangeQuery` (T003) → 400; bỏ hàm local `parseDepositStatus`, gọi `mapper.ParseDepositStatus` (đã nhận thêm `failed_legacy`) — phụ thuộc T003, T020, T021
- [x] T023 [US2] Repo `services/account/internal/modules/subscription/infrastructure/persistence/deposit_transaction_repo_impl.go`: `ListAdmin` đổi `created_at <=` → `<` với `ToDate` là cận nửa mở; ghi chú semantics trên `AdminDepositTransactionFilter` (`domain/repositories/deposit_transaction_repo.go`) — phụ thuộc T018
- [x] T024 [US2] Docs + swagger: cập nhật `docs/api/subscription/admin-billing-api.md` before→after (status chuỗi 11 giá trị, detail camelCase, kỳ VN + 400) + `make swagger svc=account` — phụ thuộc T022
- [ ] T025 [US2] Verify Kịch bản 2 `quickstart.md` live trên dev + gate: `make lint` + `go test -race -count=3` scoped `./services/account/...`

**Checkpoint**: US2 xong — trang giao dịch có contract dùng được; US1 không hồi quy

---

## Phase 5: User Story 3 — Kỳ dữ liệu dashboard nhất quán cho daily/monthly (Priority: P2)

**Goal**: Mọi panel số liệu theo kỳ phản ánh đúng kỳ chọn (VN, `[from, to)`, 400 khi sai); distribution = giao cửa sổ gói; renewal theo `created_at`; error-breakdown cùng kỳ; payOS tổng hợp theo kỳ

**Independent Test**: Kịch bản 3 `quickstart.md` (đổi kỳ → số liệu đổi; 400; mặc định giữ nguyên theo bảng data-model §3)

### Tests cho User Story 3 ⚠️

- [x] T026 [P] [US3] SQL pinning (mở rộng `services/account/internal/modules/subscription/infrastructure/persistence/admin_stats_query_sql_test.go`): stats range `created_at >= from AND < to`; distribution overlap `started_at < to AND (expires_at IS NULL OR expires_at >= from)`; renewal `created_at >= from AND < to` giữ `status <> 'processing'`
- [x] T027 [P] [US3] Commerce uc test (mở rộng `services/commerce/internal/modules/dashboard/application/usecase/admin_dashboard_uc_test.go`): chữ ký `(from, to)` cho distribution/renewal/error-breakdown/payos; mặc định khi zero (payos = hôm nay VN; error-breakdown = 30 ngày; revenue-breakdown = 6 tháng VN); error-rate = E/R cùng kỳ, R=0 → 0, ≤ 100% (đồng thời thỏa US4 AC1); `overview` truyền zero → mặc định
- [x] T028 [P] [US3] AI repo test (mở rộng `services/ai/internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl_test.go`): `GetAIErrorBreakdown` nhận kỳ; `GetAIUsageByOperation`/`GetTotalAIRequestCount` chuẩn hoá `< to`
- [x] T029 [P] [US3] Commerce handler test (mới `services/commerce/internal/modules/dashboard/presentation/handler/admin_dashboard_handler_test.go`): parse kỳ dùng chung cho 7 endpoint nhận ngày (revenue-breakdown, distribution, renewal-stats, ai/error-breakdown, payos-reconciliation, ai-margin, ai/usage-by-operation) — 400 khi sai định dạng/đảo; tham số truyền đúng xuống usecase; endpoint không kỳ không đổi

### Implementation cho User Story 3

- [x] T030 [US3] Account SQL + usecase + interface: `GetAdminSubscriptionStats(ctx, from, to)` (range, giữ scope hiện tại), `GetDistributionByPlanCode(ctx, from, to)` (overlap), `GetRenewalStats(ctx, from, to)` (created_at) trong `deposit_transaction_repo_impl.go`, `user_subscription_repo_impl.go`, `subscription_renewal_attempt_repo_impl.go`; `GetRevenueBreakdown` bỏ `AddDate(0,0,1)` nội bộ (cận nửa mở); cập nhật interface domain (`domain/repositories/deposit_transaction_repo.go`, `user_subscription_repo.go`, `subscription_audit_repo.go`) + `admin_stats_uc.go` + `application/interface/usecase/admin_stats_usecase.go`; cập nhật 8 file test/fake implement chữ ký cũ (`purchase/checkout_test.go`, `reconciliation/payment_reconciliation_usecase_test.go`, `subscription/renewal_test.go`, `quota/quota_test.go`, `subscription/admin_grant_batch_uc_test.go`, `campaign/campaign_fakes_test.go`, `admin_stats_query_sql_test.go`) — phụ thuộc T026
- [x] T031 [US3] Account contract chain: `services/account/contract/subscription/dashboard_reader.go` (interface + in-process), `services/account/internal/modules/subscription/presentation/grpc/server.go` (đọc `from_date/to_date`, fallback `snapshot_date`), `contracts/subscription/v1/client/dashboard_reader_client.go` — phụ thuộc T004, T006, T030
- [x] T032 [US3] Worker snapshot: `services/commerce/internal/modules/dashboard/application/usecase/aggregator_uc_impl.go` gọi `GetAdminSubscriptionStats(ctx, from, to)` với `from, to := VNDayRange(today)` — hành vi ngày không đổi — phụ thuộc T031
- [x] T033 [US3] AI chain: `ai_cost_repo_impl.go` (breakdown nhận kỳ + chuẩn hoá `<`), `application/usecase/cost/usecase.go` + `application/interface/usecase/cost_uc.go`, `services/ai/contract/aiexecutor/dashboard_reader.go`, `services/ai/internal/modules/aiexecutor/presentation/grpc/server.go`, `contracts/ai/v1/client/dashboard_reader_client.go` — phụ thuộc T005, T006, T028
- [x] T034 [P] [US3] Commerce handler: `services/commerce/internal/modules/dashboard/presentation/handler/admin_dashboard_handler.go` — thay 3 khối parse lặp bằng helper chung (T003); thêm tham số kỳ cho distribution/renewal/error-breakdown/payos; cập nhật swagger annotation — phụ thuộc T003
- [x] T035 [US3] Commerce usecase: `services/commerce/internal/modules/dashboard/application/usecase/admin_dashboard_uc_impl.go` — chữ ký `(from, to)`; payos mặc định hôm nay + gọi reader theo kỳ; error-breakdown mặc định 30 ngày VN + `totalErrors = Σ breakdown` + rate cùng kỳ; distribution/renewal passthrough; revenue-breakdown mặc định 6 tháng theo biên ngày VN; sửa call site KPI fallback (dòng ~65: lấy `from, to := VNDayRange(now)` rồi truyền cặp — giữ hành vi ngày); rà `GetAdminSnapshots` so `snapshot_date` với cận ngày VN (date column) — phụ thuộc T027, T031, T033, T034
- [ ] T036 [US3] Verify Kịch bản 3 `quickstart.md` live trên dev + gate: `make lint` + `go test -race -count=3` scoped `./services/account/... ./services/ai/... ./services/commerce/...` (gate đỏ nếu còn fake/test chữ ký cũ chưa cập nhật ở T030)

**Checkpoint**: US3 xong — đổi kỳ làm số liệu đổi; mặc định giữ nguyên; 400 nhất quán

---

## Phase 6: User Story 4 — Chỉ số đúng ngữ nghĩa: tỷ lệ lỗi AI và tổng giá trị giao dịch thành công (Priority: P2)

**Goal**: payOS gồm cả `wallet_topup`; `totalSuccessValue` = tổng tiền giao dịch thành công thật; tỷ lệ lỗi ≤ 100% (đã đạt ở US3, khóa bằng test)

**Independent Test**: Kịch bản 4 `quickstart.md` (đối chiếu `totalSuccessValue` với SQL; nạp ví làm số đếm tăng; error-rate ≤ 100%)

### Tests cho User Story 4 ⚠️

- [x] T037 [P] [US4] SQL pinning (mở rộng `admin_stats_query_sql_test.go`): 5 số đếm payOS không còn filter `transaction_type`; `total_payos_success_value` = `SUM(amount) FILTER (status = success)`; `total_subscription_revenue` VẪN `transaction_type = direct_purchase`
- [x] T038 [P] [US4] Commerce uc test (mở rộng `admin_dashboard_uc_test.go`): `TotalSuccessValue` lấy từ field mới (khác `SubscriptionRevenueVND` khi có topup); assert error-rate ≤ 100% sau US3

### Implementation cho User Story 4

- [x] T039 [US4] Account SQL + readmodel: `GetAdminSubscriptionStats` trong `deposit_transaction_repo_impl.go` — bỏ filter type cho 5 số đếm, thêm `total_payos_success_value`; thêm `TotalPayOSSuccessValue decimal.Decimal` vào `domain/repositories/readmodels.go` — phụ thuộc T037
- [x] T040 [US4] Transport mapper field 7: `services/account/contract/subscription/transport/mapper.go` + `to_proto.go` (string decimal ↔ decimal) + cập nhật `transport/mapper_test.go` (round-trip field 7, giữ test cũ xanh) — phụ thuộc T039
- [x] T041 [US4] Commerce: `GetPayOSReconciliation` trong `admin_dashboard_uc_impl.go` lấy `TotalSuccessValue` từ `TotalPayOSSuccessValue` (bỏ mượn `SubscriptionRevenueVND`) — phụ thuộc T040
- [x] T042 [US4] Docs + swagger: cập nhật `docs/api/dashboard/dashboard-admin-api.md` before→after (tham số kỳ mọi endpoint; payOS scope 2 loại + giá trị thật; error-rate cùng kỳ) + `make swagger svc=commerce` — phụ thuộc T041
- [ ] T043 [US4] Verify Kịch bản 4 `quickstart.md` live trên dev (đối chiếu SQL 0 đồng; topup tăng đếm) + gate: `make lint` + `go test -race -count=3` scoped `./services/account/... ./services/commerce/...`

**Checkpoint**: US4 xong — hai con số tin được; US1–US3 không hồi quy

---

## Phase 7: User Story 5 — Frontend nhận tài liệu bàn giao đầy đủ (Priority: P3)

**Goal**: `frontend-guide.md` đủ để FE dựng 2 trang + sửa dashboard, không cần đọc code BE

**Independent Test**: Đối chiếu guide với Swagger/mã nguồn bằng đếm; dựng thử mapping từ ví dụ JSON trong guide

- [ ] T044 [US5] Viết `specs/029-admin-observability/frontend-guide.md`: (a) nhóm API MỚI activity-events + hướng dẫn tích hợp trang "Hoạt động người dùng"; (b) nhóm CHANGED payOS transactions before→after + hướng dẫn tích hợp trang "Giao dịch payOS" (list + detail + 4 hành động đối soát, phản hồi `data: null`); (c) nhóm CHANGED dashboard admin — before→after cho 7 endpoint nhận kỳ + `overview` (payosReconcile ngữ nghĩa mới) + `kpi-cards` (`payOSSuccessCount` đổi giá trị) + ngữ nghĩa payOS/error-rate; (d) mục đánh giá UX dashboard FE — tối thiểu 13 vấn đề FR-019, mỗi vấn đề kèm nguyên nhân gốc + phân loại BE-đã-sửa-029 / FE-cần-làm / deferred (repo FE `smart-wardrobe-fe`, ngoài workspace BE) — phụ thuộc T015, T024, T042
- [ ] T045 [US5] Bổ sung vào guide: bảng mã HTTP cho từng endpoint, ví dụ JSON request/response thật (đối chiếu với response dev), checklist việc FE cần làm — phụ thuộc T044

**Checkpoint**: US5 xong — dev FE đọc guide là triển khai được

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Đồng bộ guide có đếm, deferred, re-validate checklist, verify cuối

- [ ] T046 **Đồng bộ guide có đếm** (hiến pháp §Governance): đọc lại `specs/029-admin-observability/frontend-guide.md`, đối chiếu với mã nguồn và **đếm** — mọi trường JSON của DTO các nhóm API mới/thay đổi (activity-events, giao dịch payOS, dashboard admin), đủ 11 giá trị status, mọi route guide liệt kê (đối chiếu chéo danh sách route trong `contracts/admin-api-delta.md` — gồm cả `overview` + `kpi-cards` + 4 hành động), mọi mã HTTP guide khẳng định; sửa mọi chỗ lệch; ghi kết quả đếm vào cuối guide — phụ thuộc T045
- [ ] T047 Ghi `specs/029-admin-observability/deferred.md`: lịch sử đổi gói cho distribution (R7); device/IP trong activity events (bảng không lưu); index bổ sung nếu T002 phát hiện; fix UX dashboard phía FE (danh sách trong guide — ngoài repo BE); mỗi mục kèm "là gì / vì sao ngoài phạm vi / điều gì gỡ được"
- [ ] T048 Re-validate `specs/029-admin-observability/checklists/requirements.md` sau mọi chỉnh sửa spec trong quá trình implement (kỳ vọng giữ 21/21; nếu có mục đổi trạng thái thì ghi rõ)
- [ ] T049 Verify cuối: chạy đủ 5 kịch bản `quickstart.md` trên dev; `make lint`; `go test -race -count=3` scoped 3 service + shared; `make swagger-all` + `make buf-gen` sạch diff; **`make check` full trước push/tag** (theo bảng gate AGENTS.md)

### Frontend Integration Guide *(required — hiến pháp §Governance)*

**Có đổi bề mặt client** → đã trả lời: guide viết ở T044/T045 (liệt kê từng endpoint mới/đổi kèm before→after — gồm cả nhóm dashboard), đồng bộ có đếm ở T046. Nhánh "không đổi bề mặt client" không áp dụng.

### Phát hiện ngoài phạm vi *(hiến pháp §Governance)*

**Đã trả lời**: hạng mục ngoài phạm vi ghi vào `specs/029-admin-observability/deferred.md` ở T047 — không để `- [ ]` treo ở file này.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Không phụ thuộc — làm ngay
- **Foundational (Phase 2)**: Sau Setup — BLOCK US2/US3/US4; US1 chỉ cần T003 xong là có thể chạy song song
- **US1 (Phase 3)**: Sau T003 (helper không dùng cho US1 nhưng phase 2 ngắn); độc lập hoàn toàn với US2–US4
- **US2 (Phase 4)**: Sau T003 — độc lập với US1/US3/US4
- **US3 (Phase 5)**: Sau Foundational — độc lập với US1/US2
- **US4 (Phase 6)**: Sau US3 (cùng câu SQL stats + cùng endpoint payOS)
- **US5 (Phase 7)**: Sau T015/T024/T042 (API đã ổn định)
- **Polish (Phase 8)**: Sau mọi story mong muốn; T049 cuối cùng

### User Story Dependencies

- **US1 (P1)**: Độc lập — MVP
- **US2 (P1)**: Độc lập — có thể làm song song US1 (khác module account: identity vs subscription)
- **US3 (P2)**: Sau Foundational; không phụ thuộc US1/US2
- **US4 (P2)**: Sau US3 (chung `GetAdminSubscriptionStats` + `GetPayOSReconciliation`)
- **US5 (P3)**: Sau khi API ổn định (T042)

### Within Each User Story

- Test FAIL trước (kể cả fail biên dịch khi chữ ký chưa đổi), implement sau
- Interface/readmodel → impl → usecase → handler → docs → verify + gate
- Story xong mới sang story ưu tiên tiếp theo

### Parallel Opportunities

- T001, T002 song song; T003, T004, T005 song song (khác file); T006 sau T004/T005
- US1: T007, T008, T009 song song; T010, T011 song song; T012 sau; T013, T014, T015 tuần tự
- US2: T017, T018, T019 song song; T020 → T021 → T022 tuần tự; T023 độc lập; T024 sau T022
- US3: T026–T029 song song; T030→T031→T032; T033 độc lập T030 (khác service); T034 song song T030; T035 sau T031/T033/T034
- US4: T037, T038 song song; T039→T040→T041→T042
- US1 và US2 có thể chạy song song bởi 2 người sau Phase 2

---

## Parallel Example: User Story 3

```bash
# Test trước (song song, khác file):
Task: "SQL pinning stats range + distribution overlap + renewal (T026)"
Task: "Commerce uc test chữ ký kỳ + mặc định + error-rate (T027)"
Task: "AI repo test breakdown range (T028)"
Task: "Commerce handler test parse kỳ 400 (T029)"

# Implement song song theo service (khác file):
Task: "Account SQL + usecase chữ ký kỳ (T030)"
Task: "AI chain range + chuẩn hoá < (T033)"
Task: "Commerce handler helper kỳ (T034)"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 + Phase 2 (T001–T006)
2. Phase 3: US1 (T007–T016)
3. **STOP và VALIDATE**: Kịch bản 1 quickstart
4. Deploy/demo nếu sẵn sàng

### Incremental Delivery

1. Setup + Foundational → helper + proto sẵn sàng
2. + US1 → tra cứu hoạt động (MVP!)
3. + US2 → trang giao dịch dùng được
4. + US3 → kỳ dữ liệu nhất quán
5. + US4 → chỉ số đúng ngữ nghĩa
6. + US5 + Polish → guide + deferred + gate cuối

### Ghi chú triển khai

- Deploy theo thứ tự account/ai trước commerce (proto additive; degradation tạm ghi trong `contracts/grpc-contract-delta.md` §4)
- Commit sau mỗi task hoặc nhóm logic; không commit secret
- `make check` full chỉ chạy trước push/tag (T049), không chạy trong vòng lặp từng task

---

## Notes

- [P] = khác file, không phụ thuộc task chưa xong
- [Story] trace về user story trong spec.md
- US3 sở hữu công thức error-rate cùng kỳ (thỏa US4 AC1); US4 khóa bằng test — không làm lại
- Không migration; `dashboard_admin_daily_snapshots` giữ nguyên cấu trúc (R12)

---

## Format Validation

- Tổng 49 task (T001–T049), 100% theo format `- [ ] [ID] [P?] [Story?] mô tả + đường dẫn file`
- Setup: 2 • Foundational: 4 • US1: 10 (T007–T016) • US2: 9 (T017–T025) • US3: 11 (T026–T036) • US4: 7 (T037–T043) • US5: 2 (T044–T045) • Polish: 4 (T046–T049)
