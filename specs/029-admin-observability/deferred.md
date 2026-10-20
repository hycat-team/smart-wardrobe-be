# Deferred — Khả năng quan sát cho admin (029)

Work discovered while building 029 that is **not this feature's job**.
Nothing here blocks closing 029; nothing here is a task of 029.

## Entries

### D-01 — Index bổ sung cho các truy vấn kỳ dữ liệu (T002)

- **Vấn đề**: Không có index `created_at` đơn lẻ trên `subscription.deposit_transactions`
  (chỉ có partial index theo trạng thái/reconciliation), `subscription.subscription_renewal_attempts`
  (chỉ có unique key), và `aiexecutor.ai_usage_events` chỉ có composite `(user_id, created_at, status)`
  — không dùng được cho truy vấn lọc theo `created_at` đơn thuần. Các truy vấn mới của 029
  (payOS theo kỳ, renewal theo `created_at`, error-breakdown theo kỳ) và các truy vấn cũ
  (`ListAdmin`, revenue-breakdown, worker snapshot 15 phút/lần) đều bounded theo kỳ nên hiện
  chấp nhận được ở quy mô hiện tại, nhưng sẽ chậm dần khi bảng lớn lên.
- **Vì sao ngoài phạm vi**: 029 tuyên bố 0 migration (R12 — toàn bộ thay đổi là read-side);
  thêm index là một migration riêng cần cân nhắc chi phí ghi và kế hoạch migration của account/AI.
  Đây là quyết định hạ tầng vận hành, không phải yêu cầu chức năng của 029.
- **Gỡ bằng cách nào**: Khi `deposit_transactions` / `subscription_renewal_attempts` /
  `ai_usage_events` đủ lớn để truy vấn kỳ chậm (đo bằng `EXPLAIN ANALYZE` trên môi trường thật),
  thêm migration Goose tại service tương ứng: `(created_at)` hoặc composite theo pattern truy vấn
  phổ biến nhất (ví dụ `(status, created_at)` cho deposit). Ghi vào spec vận hành/performance tiếp theo.
- **Ảnh hưởng tới task đã tick**: không — T002 chính là task ghi nhận phát hiện này; các task
  truy vấn (T030, T033, T035) vẫn đúng vì truy vấn bounded theo kỳ.

- **TRẠNG THÁI (2026-10-20 — RESOLVED, đo trước khi thêm)**: đã đo trên database tổng hợp
  (200k `deposit_transactions`, 100k `subscription_renewal_attempts`, 500k `ai_usage_events`,
  cửa sổ 30 ngày) bằng `EXPLAIN (ANALYZE, BUFFERS)`, rồi chỉ thêm index có thay đổi plan:
  - `subscription.deposit_transactions (created_at DESC)` — list không lọc status: **17.5 → 0.10 ms**;
    list có `status IN (...)`: **24.1 → 0.20 ms**; revenue theo kỳ: **12.9 → 4.2 ms**.
  - `subscription.subscription_renewal_attempts (created_at)` — stats theo kỳ: **16.1 → 2.6 ms**.
  - `aiexecutor.ai_usage_events (created_at)` — aggregate theo kỳ: **59.0 → 18.5 ms**.
  - `aiexecutor.ai_usage_events (status, created_at)` — error-breakdown: **36.9 → 2.9 ms**
    (Index Only Scan; bỏ composite thì trở lại 15.5 ms).
  - **Loại có bằng chứng**: `deposit_transactions (status, created_at)` chỉ cải thiện revenue
    aggregate từ 4.2 → 2.6 ms và không giúp gì cho list, nên không thêm để tránh chi phí ghi trên
    đường ghi nóng nhất.
  Migration: `services/account/.../20261020130000_add_period_indexes.sql` và
  `services/ai/.../20261020130000_add_ai_usage_period_indexes.sql` (có `Down` drop index; đã kiểm
  tra vòng down→up trên DB tạm). `make migration-check` xanh (27 file, 0 vi phạm),
  `make test-db-integration` xanh.

### D-02 — Trần `limit` của list giao dịch lệch giữa repo (200) và metadata (100)

- **Vấn đề**: `GET /admin/subscriptions/transactions` truyền `limit` thô xuống repo — repo cap 200 —
  trong khi `PaginationQuery.Normalize` (dùng cho `metadata`) cap 100. Gửi `limit=150` trả 150 item
  nhưng `metadata.limit` ghi 100. Lỗi tồn tại trước 029.
- **Vì sao ngoài phạm vi**: FR-010 giữ nguyên hành vi phân trang của list; sửa trần là thay đổi
  hành vi ngoài phạm vi tính năng (và cần rà các caller khác của repo).
- **Gỡ bằng cách nào**: thống nhất một trần duy nhất (khuyến nghị 100 theo `PaginationQuery`) trong
  handler hoặc repo, kèm test; guide 029 đã khuyến nghị FE gửi ≤ 100.
- **Ảnh hưởng tới task đã tick**: không (T019/T022/T023 giữ nguyên semantics phân trang cũ theo FR-010).
- **TRẠNG THÁI (2026-10-20, đợt quick-win — RESOLVED)**: handler `ListTransactions` nay
  `Normalize()` một lần rồi dùng đúng giá trị đó cho cả truy vấn và metadata, nên trần là 100 ở
  cả hai nơi. Có test `TestAdminBillingListAppliesTheMetadataLimit` khoá 3 ca: `limit=150 → 100`,
  `limit=50 → 50`, không truyền → 20.

### D-03 — Ví dụ envelope cũ (`"success": true`) trong `docs/api/dashboard/dashboard-admin-api.md`

- **Vấn đề**: ví dụ `kpi-cards` trong doc dashboard còn dùng envelope thời monolith
  (`{"success": true, "data": {...}}`); envelope thật là `{message, data}` (`shared/presentation/response.go`).
  Tồn tại trước 029 (xác nhận bằng `git show cd920cfe`).
- **Vì sao ngoài phạm vi**: T042 chỉ cập nhật before→after của các thay đổi thuộc 029; sửa ví dụ cũ
  là rà soát doc tổng thể, không thuộc hợp đồng delta.
- **Gỡ bằng cách nào**: rà `docs/api/dashboard/*.md` thay mọi ví dụ `"success": true` bằng envelope
  thật; cân nhắc thêm bước lint doc.
- **Ảnh hưởng tới task đã tick**: không.
- **TRẠNG THÁI (2026-10-20, đợt quick-win — RESOLVED)**: đã rà toàn bộ `docs/api/**` và thay
  mọi ví dụ `"success": true` bằng envelope thật `{message, data}` — 6 chỗ trong 4 file
  (`dashboard-admin-api.md`, `admin-brand-api.md`, `brand-item-api.md`, `brand-portal-api.md`).
  Grep `"success": true` trong `docs/api` = 0.

### D-04 — Message 401 là chuỗi mã nội bộ ("4011")

- **Vấn đề**: lỗi `ErrInvalidAccessToken` trả `message: "4011"` — không thân thiện nếu FE hiển thị thô.
  Tồn tại trước 029.
- **Vì sao ngoài phạm vi**: thuộc tầng auth dùng chung mọi endpoint, không thuộc bề mặt 029; sửa
  message là thay đổi client-facing ngoài phạm vi tính năng.
- **Gỡ bằng cách nào**: đổi message sang tiếng Việt thân thiện trong `apperror` auth (kèm rà FE có
  phụ thuộc chuỗi mã không); guide 029 đã khuyến nghị FE không hiển thị thô `message` của 401.
- **Ảnh hưởng tới task đã tick**: không.

### D-05 — 13 lỗi UX dashboard phía FE (danh sách đầy đủ trong `frontend-guide.md` §5)

- **Vấn đề**: các lỗi UX thuộc repo `smart-wardrobe-fe` (mặc định 30 ngày cuốn, thiếu preset,
  nhãn trùng, filter AI events lỗi giá trị mặc định, `console.log` sót, lệch múi giờ hiển thị,
  granularity hardcode, trộn ngôn ngữ, cột "Loại" sai trường, dữ liệu chết KPI...).
- **Vì sao ngoài phạm vi**: repo BE không chứa code FE; 029 chỉ bàn giao đánh giá qua guide (FR-019).
- **Gỡ bằng cách nào**: FE triển khai theo checklist `frontend-guide.md` §7; theo dõi ở backlog
  `smart-wardrobe-fe`. Các fix BE liên quan (kỳ VN, 400, error-rate, payOS value) đã xong trong 029.
- **Ảnh hưởng tới task đã tick**: không (guide §5 phân loại rõ BE-đã-sửa vs FE-cần-làm).

### D-06 — Distribution không tái dựng được lịch sử đổi gói (R7)

- **Vấn đề**: `user_subscriptions` chỉ lưu trạng thái gói hiện tại; phân bổ theo kỳ đếm theo giao
  cửa sổ gói hiện tại với kỳ lọc — user đã đổi gói chỉ được tính theo gói hiện tại, gói cũ trong
  kỳ không xuất hiện.
- **Vì sao ngoài phạm vi**: đã chốt là giới hạn chấp nhận khi clarify (Q1); tái dựng lịch sử cần
  nguồn dữ liệu mới (audit/renewal không đảm bảo đủ) — thiết kế khác, không phải sửa nhỏ.
- **Gỡ bằng cách nào**: khi có yêu cầu chính xác lịch sử phân bổ, thiết kế bảng snapshot phân bổ
  định kỳ hoặc dựng lại từ `subscription_audit` + `deposit_transactions` trong một spec riêng.
- **Ảnh hưởng tới task đã tick**: không — T027/T030/T035 khớp quyết định clarify và có test khóa.
- **TRẠNG THÁI (2026-10-20 — RESOLVED bởi feature 031)**: chọn **bảng khoảng hiệu lực ghi cùng
  transaction** thay cho hai gợi ý cũ (snapshot định kỳ / dựng lại từ audit + giao dịch):
  - **Vì sao khác gợi ý cũ**: snapshot định kỳ chỉ đúng tại đúng thời điểm chụp; một kỳ bất kỳ vẫn
    phải nội suy giữa hai lần chụp và phụ thuộc job chạy đúng giờ — job trễ hoặc chết là mất dữ liệu
    vĩnh viễn. Dựng lại từ `subscription_audit` + `deposit_transactions` không đủ chính xác: sự kiện
    gia hạn không ghi lại thời điểm hết hạn kết quả, còn giao dịch chỉ chứng minh việc mua, không
    chứng minh khoảng thời gian người dùng thực sự giữ gói (spec 031 §Bối cảnh). Bảng interval trả
    lời chính xác **mọi kỳ**, không phụ thuộc job, và không cần suy đoán.
  - **Hook ghi**: `PlanChangePublisher.PublishPlanChange`
    (`services/account/.../plan_change_publish_helper.go`) gọi `PlanPeriodRecorder.Record` **sau**
    khi insert `user_subscription_events` thành công, trong **cùng transaction** của thao tác đổi
    gói. Event trùng `event_key` return sớm nên retry không tạo khoảng thứ hai; recorder lỗi làm
    rollback cả event lẫn khoảng. Event metadata `source=admin_restore` là no-op tuyệt đối.
  - **Backfill + cutover**: migration `services/account/database/migrations/20261020150000_add_user_plan_periods.sql`
    tạo `subscription.user_plan_periods` + `subscription.plan_history_meta`, backfill đúng cửa sổ
    hiện tại của mỗi `user_subscriptions` (`source_event_key = 'BACKFILL:<user_id>'`), và ghi row
    `plan_history_meta('cutover', now())`. Preflight `RAISE EXCEPTION` nếu có row
    `current_plan_kind = 1 AND expires_at <= started_at`; không bỏ qua row vì sẽ làm mất user khỏi
    báo cáo.
  - **Giới hạn trước cutover**: lịch sử trước mốc `cutover` **không** được tái dựng (FR-013). Báo
    cáo kỳ trước mốc trả `dataCompleteness = unavailable` (distribution rỗng, totalUsers 0); kỳ cắt
    ngang mốc trả `partial` và chỉ tính phần từ mốc trở đi. Đây là hành vi đúng, không phải lỗi.
  - **Task/test của 031**: T003 (migration + backfill + preflight), T006 (repo interval), T007
    (recorder), T008 (hook publisher + DI), T010–T020 (báo cáo, DTO, proto, swagger), T027–T031
    (transition/idempotency/rollback/concurrency). Test khóa:
    `TestIntegrationUserPlanPeriodsBackfillAndCutover`, `...PreflightAborts`, `...OverlapGuard`,
    `...FallbackWithoutExtensionPrivilege`, `TestIntegrationPlanPeriodRecorderLifecycle`,
    `TestIntegrationPlanPeriodRecorderConcurrentChanges`,
    `TestIntegrationPlanPeriodRecorderFallbackConcurrentChanges`,
    `TestIntegrationPlanPeriodRecorderPublishIsIdempotent`,
    `TestIntegrationPlanPeriodRecorderFailureRollsBackBothRows`,
    `TestIntegrationUserPlanPeriodDistributionLifecycles`, và bộ unit test
    `distribution_completeness_test.go` / `plan_period_recorder_test.go`.
  - **Hệ quả cho 029**: D-06 đóng; không còn cảnh báo "không tái dựng được lịch sử" trong guide 029
    — đã thay bằng "đầy đủ từ mốc cutover" và trỏ sang `specs/031-user-plan-periods/frontend-guide.md`.

### D-07 — Activity events không có thiết bị/IP

- **Vấn đề**: bảng `user_activity_events` không lưu `deviceName`/`deviceIP`/`userAgent` (những
  trường này nằm ở bảng sessions). Admin điều tra "ai đăng nhập từ đâu" chỉ thấy user + thời điểm.
- **Vì sao ngoài phạm vi**: scope FR-001 chỉ đọc bảng activity-events hiện có; thêm trường thiết bị
  cần migration + thay đổi write-path của session (và quyết định privacy).
- **Gỡ bằng cách nào**: spec riêng bổ sung cột/nguồn dữ liệu thiết bị cho sự kiện login (hoặc join
  sessions theo thời điểm) kèm cân nhắc PII.
- **Ảnh hưởng tới task đã tick**: không.

- **TRẠNG THÁI (2026-10-20 — RESOLVED)**: chọn hướng "ghi cùng sự kiện" (không join session
  theo thời điểm):
  - Migration `services/account/.../20261020131000_add_activity_device_columns.sql` thêm
    `device_name varchar(255)`, `device_ip varchar(45)`, `user_agent varchar(512)` vào
    `identity.user_activity_events` (cùng kiểu `identity.refresh_tokens`), có `Down`; đã kiểm tra
    vòng up→down→up.
  - Write path: `SessionUseCase.recordActivity` nhận device; login lấy từ input, refresh lấy từ
    token cũ hoặc input ghi đè (đúng nguồn session đang dùng).
  - Read path: entity → repo SELECT/`AdminActivityEventRow` → `ActivityEventRes` → mapper; cả ba
    field là con trỏ nên JSON **vắng mặt** với hàng ghi trước migration hoặc khi client không gửi.
  - Tài liệu client-facing: `docs/api/identity/admin-activity-api.md` đã thêm 3 field + mục
    before→after, nêu rõ "vắng mặt = không ghi nhận".
  - Test khoá: login ghi đủ 3 field (`TestIssueSessionRecordsLoginActivity`), mapper giữ field và
    giữ `nil` cho hàng cũ (`activity_mapper_test.go`), SQL pin cột mới
    (`TestListAdminActivityEventsSQLPinsJoinOrderAndMatchingCount`).
  - **Quyết định PII (owner)**: lưu đầy đủ, admin thấy nguyên văn, **không purge** — vòng đời
    bằng vòng đời bản ghi activity. Ghi ở đây để lần sau không phải suy lại.
  - Phụ: harness `usecase/subscription` được làm cho **rerunnable** (email user theo id, dọn slug
    fixture trước khi seed) vì lần chạy lại trên cùng DB đã lộ va chạm unique key; suite nay chạy
    hai lần liên tiếp trên cùng DB đều xanh.

### D-08 — `GET /admin/ai/usage` (feature 012) vẫn dùng cận `created_at <=`

- **Vấn đề**: endpoint usage cũ (`ai_cost_repo_impl.go`, feature 012) so `created_at <= toDate` với
  mốc UTC-midnight — cùng lớp lỗi lệch ngày mà 029 chuẩn hoá cho 7 endpoint theo kỳ (FR-014).
  Endpoint này KHÔNG nằm trong danh sách FR-011 nên 029 giữ nguyên.
- **Vì sao ngoài phạm vi**: không thuộc bề mặt 029 (không có trong plan/guide); sửa là thay đổi
  hành vi của feature 012, cần rà caller riêng.
- **Gỡ bằng cách nào**: một spec/task nhỏ chuẩn hoá endpoint này theo `ParseVNDateRangeQuery` +
  cận nửa mở, kèm test pin kỳ.
- **Ảnh hưởng tới task đã tick**: không — phát hiện bởi converge vòng 2, ghi nhận ngoài phạm vi.
- **TRẠNG THÁI (2026-10-20, đợt quick-win — CLOSED, không cần code)**: mục này đã trở nên lỗi
  thời. Endpoint `GET /admin/ai/usage` của feature 012 **không còn tồn tại** trong router hiện tại
  (`services/commerce/.../dashboard/presentation/router/router.go` chỉ còn
  `/ai/usage-by-operation` và `/ai/error-breakdown`), và repo `ai_cost_repo_impl.go` đã dùng cận nửa
  mở `created_at < ?` với comment tham chiếu 029 R1. Grep `created_at <=` toàn repo = 0.
