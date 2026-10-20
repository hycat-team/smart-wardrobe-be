---
description: "Task list for feature 031 — validity intervals for subscription distribution"
---

# Tasks: Lịch sử gói theo khoảng hiệu lực

**Input**: Design documents from `specs/031-user-plan-periods/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/distribution-api-delta.md

**Tests**: Required. The spec's technical constraints demand recorder unit tests, Postgres integration
tests for five lifecycle scenarios, SQL pin tests, and completeness tests. Test tasks appear before the
implementation they cover within each story.

**Organization**: Tasks are grouped by user story. The write path (schema, repository, recorder) is
shared infrastructure and therefore foundational; each story then adds its own readable slice.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1, US2, US3 (see spec.md)
- Every task names the exact file(s) it touches.

## Path Conventions

Paths are repository-relative. Account service root: `services/account/`. Commerce service root:
`services/commerce/`. Contracts root: `contracts/`.

---

## Phase 1: Setup

**Purpose**: Prepare the verification environment and confirm the toolchain before touching schema.

- [x] T001 [P] Dựng database tạm `verify_plan031` (init `database/init/*.sql` + `make migrate-up` cho account/commerce/ai) theo `specs/031-user-plan-periods/quickstart.md`; chấp nhận **một trong hai** biến thể: role thường tạo được `btree_gist` + EXCLUDE, hoặc role bị chặn `CREATE EXTENSION` để migration rơi vào fallback partial unique; chuẩn bị cả hai role cho T009 — *Ghi chú: orchestrator xác nhận môi trường đã sẵn sàng; T009 tự tạo scratch DB khi role có `CREATEDB`, nếu không thì subtest migration tự skip.*
- [x] T002 [P] Xác nhận toolchain sạch trên nhánh hiện tại: `make buf-lint`, `make buf-gen`, `make swagger svc=commerce`, `make migration-check` (đọc target trong `Makefile`) — *Ghi chú: orchestrator xác nhận toolchain đã sạch; lượt này chạy thêm `make lint`, `make migration-check`, `go build ./...` và test scoped (không đổi proto/swagger).*

**Checkpoint**: Scratch DB và codegen chạy được; chưa có thay đổi mã nguồn.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Schema, entity, repository, recorder và DI — mọi user story đều đọc/ghi trên nền này.

**⚠️ CRITICAL**: Không bắt đầu user story nào trước khi phase này xong.

- [x] T003 Tạo migration `services/account/database/migrations/20261020150000_add_user_plan_periods.sql`: bảng `subscription.user_plan_periods` + `subscription.plan_history_meta` đúng DDL trong `specs/031-user-plan-periods/data-model.md` (CHECK, `plan_name`, 3 index gồm GiST range); `DO` block thử `CREATE EXTENSION btree_gist` + EXCLUDE, fallback partial unique `(user_id) WHERE valid_to IS NULL`; backfill `INSERT ... SELECT` join `subscription_plans` lấy `plan_name`, preflight `DO` block: nếu có row `current_plan_kind = 1 AND expires_at <= started_at` thì `RAISE EXCEPTION` kèm danh sách `user_id` để operator sửa dữ liệu trước khi migrate (không bỏ qua row, vì sẽ làm thiếu user trong báo cáo), `source_event_key = 'BACKFILL:<user_id>'`; chèn row `cutover`; `RAISE NOTICE` thời gian backfill; Down drop cả hai bảng (không drop extension)
- [x] T004 [P] Thêm entity `services/account/internal/modules/subscription/domain/entities/user_plan_period.go` (`UserPlanPeriod` gồm `PlanName`, `PlanHistoryMeta`) và đăng ký `TableName()` trong `services/account/internal/modules/subscription/domain/entities/table_name.go`
- [x] T005 [P] Thêm read model + interface `services/account/internal/modules/subscription/domain/repositories/user_plan_period_repo.go`: `SubscriptionDistributionResult`, `IUserPlanPeriodRepository` với `GetDistribution(ctx, from, to)`, `GetCurrentByUserIDWithLock`, `CloseCurrent`, `Create`, `UpdateValidTo`, `GetByUserID`, `GetHistoryCutover`
- [x] T006 Implement GORM repo `services/account/internal/modules/subscription/infrastructure/persistence/user_plan_period_repo_impl.go`: predicate giao khoảng theo `data-model.md §5` dùng `tstzrange(valid_from, valid_to, '[)')` để dùng GiST index; bound zero truyền `NULL`, riêng `to` zero thêm `valid_from <= now()`; `COUNT(DISTINCT user_id)`, `FOR UPDATE` cho current interval (row mới nhất theo alid_from), cập nhật `valid_to`/`updated_at`, truy vấn lịch sử theo `(user_id, valid_from)`, đọc cutover
- [x] T007 Thêm recorder `services/account/internal/modules/subscription/application/usecase/subscription/plan_period_recorder.go` định nghĩa interface `PlanPeriodRecorder` + implementation theo bảng `data-model.md §3`: chữ ký `Record(ctx, event, sub, plan)` để lấy `plan_name`; lấy `pg_advisory_xact_lock(hashtext(user_id))` trước khi đọc current interval (guard chính, hoạt động cả ở fallback); no-op **tuyệt đối** khi event metadata có `source=admin_restore` (kiểm tra trước mọi nhánh), và no-op khi admin restore đúng gói đang mở; thay thế row khi `closeAt <= current.valid_from`; `min(current.valid_to, effective_at)`; clamp `valid_from = max(effective_at, current.valid_from)`; `EXTENDED` chỉ dời `valid_to`; downgrade mở free tại `closeAt`; `AUTO_RENEW_UPDATED` no-op
- [x] T008 Nối recorder vào `PlanChangePublisher` (`application/usecase/subscription/plan_change_publish_helper.go`): thêm dependency, gọi **sau** khi insert event thành công và bỏ qua khi event đã tồn tại; đăng ký provider trong `application/provider.go` và `infrastructure/provider.go`; cập nhật **toàn bộ** constructor `NewPlanChangePublisher` trong test (grep `NewPlanChangePublisher`, hiện khoảng 17 chỗ như `campaign_harness_test.go`, `plan_changed_payload_validation_test.go`, `renewal_test.go`, `eager_default_provision_test.go`) bằng recorder no-op; chạy `make wire svc=account` và `cd services/account; go build ./...`
- [x] T009 [P] DB integration test nền cho migration/repo tại `services/account/internal/modules/subscription/infrastructure/persistence/user_plan_period_repo_impl_test.go`: backfill tạo đúng 1 interval/user kèm `plan_name`, cutover row tồn tại; với EXCLUDE: insert chồng lấn fail `23P01`; với fallback (role không có quyền `CREATE EXTENSION`): `ux_user_plan_periods_open` tồn tại và insert open thứ hai fail `23505`; thêm case row `expires_at <= started_at` làm migration fail với thông báo rõ; `GetDistribution` trả đúng `COUNT(DISTINCT)` cho user quay lại cùng gói; đổi tên plan sau kỳ không làm thay đổi `plan_name` đã snapshot — *Ghi chú: test đã viết và compile; chưa chạy trong lượt này vì không có `TEST_DATABASE_URL` — sẽ chạy ở DB gate của orchestrator.*

**Checkpoint**: `make wire svc=account` + `cd services/account; go build ./...` xanh; migration up/down/up xanh; recorder đã có nơi gọi duy nhất.

---

## Phase 3: User Story 1 — Báo cáo phân bố đúng theo kỳ (Priority: P1) 🎯 MVP

**Goal**: Admin chọn kỳ và nhận đúng số user từng gói, kể cả người đã đổi/hạ gói sau kỳ; phản hồi
giữ nguyên shape và thêm trạng thái đầy đủ dữ liệu.

**Independent Test**: Sau khi backdate cutover trong fixture, dựng 3 user (hạ gói sau kỳ, đổi gói giữa kỳ, free cả kỳ), gọi endpoint cho kỳ test,
đối chiếu bảng chuẩn; gọi kỳ trước cutover và nhận `unavailable`.

### Tests for User Story 1

- [x] T010 [P] [US1] Unit test tính completeness tại `services/account/internal/modules/subscription/application/usecase/admin_stats/distribution_completeness_test.go`: cả hai bound zero (không được coi là `to <= cutover`), chỉ `from`, chỉ `to`, `to` provided và `<= cutover`, `from < cutover < to`, `from >= cutover`, row cutover vắng mặt (kết quả có `dataCompleteness = unavailable` và không có `dataAvailableFrom`)
- [x] T011 [P] [US1] Integration test phân bố tại `services/account/internal/modules/subscription/infrastructure/persistence/user_plan_period_distribution_integration_test.go`: fixture đặt `plan_history_meta.started_at` lùi về trước kỳ test; hạ gói sau kỳ vẫn hiện gói cũ; đổi gói giữa kỳ hiện cả hai bucket; free/lifetime; kỳ một phía (chỉ `from` không tính interval tương lai); kỳ cắt ngang cutover chỉ tính phần sau mốc; tổng bucket khớp
- [x] T012 [P] [US1] Thay SQL pin cũ trong `services/account/internal/modules/subscription/infrastructure/persistence/admin_stats_query_sql_test.go` bằng pin cho câu query interval (`tstzrange ... &&`, `valid_from <= now()` cho kỳ chỉ có `from`, `COUNT(DISTINCT user_id)`)

### Implementation for User Story 1

- [x] T013 [US1] Cập nhật `services/account/internal/modules/subscription/application/usecase/admin_stats/admin_stats_uc.go` dùng `IUserPlanPeriodRepository`, tính `DataCompleteness` và cắt `from` lên cutover khi `partial`; thêm `SubscriptionDistributionResultDTO` trong `services/account/dto/subscription/dashboard_dto.go` và sửa interface `application/interface/usecase/admin_stats_usecase.go`
- [x] T014 [US1] Đổi contract `services/account/contract/subscription/dashboard_reader.go` trả `SubscriptionDistributionResultDTO` (in-process adapter + comment ngữ nghĩa mới)
- [x] T015 [US1] Thêm enum `DistributionDataCompleteness` và 2 field vào `SubscriptionDistributionList` trong `contracts/subscription/v1/subscription.proto`, chạy `make buf-gen` và commit stub trong `contracts/subscription/v1/*.pb.go`
- [x] T016 [US1] Map field mới hai chiều + test round-trip (`UNSPECIFIED` → bỏ field): `services/account/contract/subscription/transport/to_proto.go`, `.../transport/mapper.go`, `.../transport/mapper_test.go`, server `.../presentation/grpc/server.go`, client `contracts/subscription/v1/client/dashboard_reader_client.go`
- [x] T017 [US1] Cập nhật `services/commerce/internal/modules/dashboard/application/dto/admin_dashboard_dto.go` (`DataAvailableFrom *string` định dạng `YYYY-MM-DD`, `DataCompleteness string`, `omitempty`) và `application/usecase/admin_dashboard_uc_impl.go` (truyền metadata, giữ cách tính `%` và `totalUsers` theo tổng bucket)
- [x] T018 [US1] Cập nhật swagger annotation + response trong `services/commerce/internal/modules/dashboard/presentation/handler/admin_dashboard_handler.go`, chạy `make swagger svc=commerce` và `make swagger-combine`
- [x] T019 [US1] Xoá `GetDistributionByPlanCode` khỏi `services/account/internal/modules/subscription/domain/repositories/user_subscription_repo.go` và implementation trong `infrastructure/persistence/user_subscription_repo_impl.go`; sửa mọi test/compile phụ thuộc
- [x] T020 [US1] Sửa test use case Commerce `services/commerce/internal/modules/dashboard/application/usecase/admin_dashboard_uc_test.go` cho result DTO mới và assert 2 field optional
- [x] T021 [P] [US1] Cập nhật tài liệu API `docs/api/dashboard/dashboard-admin-api.md`: nguồn dữ liệu interval, kỳ một phía, `totalUsers` là tổng bucket, ý nghĩa 3 giá trị `dataCompleteness`, ví dụ before→after
- [x] T022 [P] [US1] Viết `specs/031-user-plan-periods/frontend-guide.md`: endpoint bị đổi (method/path), 2 field mới, hành vi trước→sau, cách render `partial`/`unavailable`
- [x] T023 [US1] Cập nhật cảnh báo lịch sử trong `specs/029-admin-observability/frontend-guide.md` từ "không tái dựng được" thành "đầy đủ từ mốc cutover" và trỏ sang guide 031

**Checkpoint**: Sau khi backdate cutover, gọi endpoint cho kỳ test với user đã hạ gói sau kỳ trả đúng gói cũ; kỳ trước cutover trả `unavailable`.

---

## Phase 4: User Story 2 — Đối chiếu lịch sử gói của một người dùng (Priority: P2)

**Goal**: Kỹ sư vận hành đọc được các khoảng gói của một user (thứ tự, mốc, không chồng lấn) qua công
cụ tra cứu nội bộ, không cần endpoint mới.

**Independent Test**: Chạy lifecycle 4 bước (activate → extend → upgrade → downgrade) rồi truy vấn
`user_plan_periods` theo `user_id`; kiểm tra thứ tự, mốc nối tiếp, không khoảng hở khi chuyển liên tục.

- [x] T024 [P] [US2] Bổ sung test truy vấn lịch sử theo user vào `services/account/internal/modules/subscription/infrastructure/persistence/user_plan_period_repo_impl_test.go`: thứ tự `valid_from`, khoảng sau nối khoảng trước, `EXTENDED` không tạo row, free/lifetime `valid_to IS NULL`
- [x] T025 [US2] Thêm mục tra cứu vận hành vào `docs/development/technical-notes.md`: mô tả bảng `subscription.user_plan_periods`, câu SQL theo `user_id`, cách đọc `valid_to NULL`, và cảnh báo không sửa tay
- [x] T026 [US2] Chạy `EXPLAIN (ANALYZE, BUFFERS)` cho truy vấn lịch sử trên DB tạm, xác nhận dùng `idx_user_plan_periods_user_valid_from`, ghi kết quả vào `docs/development/technical-notes.md`

**Checkpoint**: Một user bất kỳ có thể được đối chiếu lịch sử bằng SQL đã tài liệu hoá; index được chứng minh.

---

## Phase 5: User Story 3 — Hệ thống tự ghi lịch sử (Priority: P3)

**Goal**: Lịch sử được ghi tự động, đúng biên, idempotent, và rollback cùng giao dịch nghiệp vụ.

**Independent Test**: Thực hiện từng loại thay đổi gói và kiểm tra lịch sử; làm hỏng recorder để xác
nhận thao tác nghiệp vụ rollback; chạy lại cùng event để xác nhận không sinh khoảng trùng.

- [x] T027 [P] [US3] Unit test transition table tại `services/account/internal/modules/subscription/application/usecase/subscription/plan_period_recorder_test.go`: 6 event có tác động + `AUTO_RENEW_UPDATED` no-op, `min(valid_to, effective_at)`, clamp `valid_from`, trường hợp không có current interval
- [x] T028 [P] [US3] Integration test lifecycle tại `services/account/internal/modules/subscription/application/usecase/subscription/plan_period_recorder_integration_test.go`: activate → extend → upgrade → downgrade; downgrade muộn hơn expiry đóng đúng `expires_at`; overlay lifetime rồi restore; mọi khoảng liên tục và không chồng lấn
- [x] T029 [P] [US3] Test idempotency + rollback tại `services/account/internal/modules/subscription/application/usecase/subscription/plan_period_recorder_idempotency_test.go`: gọi `PublishPlanChange` hai lần cùng `event_key` cho một interval; recorder trả lỗi (fake repo) làm rollback cả event và interval
- [x] T030 [US3] Test đồng thời hai thay đổi cho cùng user tại `plan_period_recorder_integration_test.go` và `plan_period_recorder_fallback_integration_test.go`: advisory lock serialize hai transaction, cả hai commit, kết quả cuối có đúng một current interval, 0 cặp chồng lấn, kể cả khi EXCLUDE bị thay bằng partial unique
- [x] T031 [US3] Kiểm tra và bổ sung test cho các biên đã biết trong `plan_period_recorder_test.go`: admin restore cùng gói (no-op), admin restore chạy đồng thời với purchase (event metadata `source=admin_restore` phải no-op, không ghi đè interval mới), provision + grant cùng mốc (replace), legacy row không có current interval, lapse có khoảng hở, `valid_from` bị clock skew, downgrade/restore đúng mốc `current.valid_from` (fallback cũng replace, không tạo interval 0)

**Checkpoint**: 7 event type đều có test; retry/rollback/concurrency được chứng minh trên Postgres.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Tài liệu hệ thống, deferred, đo hiệu năng, gate cuối.

- [x] T032 [P] Cập nhật inventory trong `docs/system-design/architecture.md`: dòng `subscription` 10 → 12 bảng (thêm `user_plan_periods`, `plan_history_meta`), tổng bảng 52 → 54, tổng FK và số FK của subscription tăng tương ứng (kiểm lại sau migration) — *Thực tế đếm từ DB: 58 bảng, không phải 54, vì inventory cũ còn thiếu `identity.user_activity_events` (+1) và 3 bảng campaign của 025 (+3); tổng FK 85, users FK 41, subscription 17 FK (10 tới users).*
- [x] T033 [P] Đánh dấu `029/D-06` RESOLVED trong `specs/029-admin-observability/deferred.md`: ghi rõ chọn bảng interval thay cho snapshot định kỳ như gợi ý cũ và vì sao (chính xác theo mọi kỳ, không phụ thuộc job), hook publisher, backfill cutover, giới hạn trước cutover, task/test tương ứng
- [x] T034 [P] Seed 200.000 interval trên DB tạm, chạy 50 lần cho kỳ 30 ngày và snapshot để lấy p95 (kèm `EXPLAIN (ANALYZE, BUFFERS)` xác nhận GiST range index), ghi số đo (200k cho SC-004, thêm 1M để tham khảo) và kết luận đạt/không đạt vào `docs/development/technical-notes.md`; nếu trượt SC-004, thêm một task thật vào `tasks.md` cho phương án tối ưu (ví dụ pre-aggregate) và giữ SC-004 mở — không đẩy việc trong phạm vi sang `deferred.md` và không tick task khi chưa đạt
- [x] T035 Thêm mục "Kiểm chứng feature 031" vào `specs/031-user-plan-periods/quickstart.md` sau khi chạy hết: các kịch bản đã chạy, kết quả, và lệnh tái lập
- [x] T036 Chạy `make generate` để đồng bộ swagger + wire + nginx route map; xác nhận diff chỉ gồm thay đổi của feature
- [x] T037 Chạy gate đầy đủ: `make migration-check`, `make SHELL=C:/Windows/System32/cmd.exe lint`, `go test -race -count=3` cho `./services/account/internal/modules/subscription/...` và `./services/commerce/internal/modules/dashboard/...`, `make test-db-integration`, `make buf-lint`, kiểm tra drift bằng `make buf-gen` + `git diff --exit-code contracts/`, và cuối cùng `make check` (pre-push gate)
- [x] T038 Rà soát phát hiện ngoài phạm vi trong lúc làm; nếu có, ghi vào `specs/031-user-plan-periods/deferred.md` (là gì, vì sao ngoài phạm vi, gì sẽ gỡ); nếu không có, ghi rõ "không có phát hiện ngoài phạm vi"

- [x] T039 Thêm test tương thích client cũ tại `services/commerce/internal/modules/dashboard/application/usecase/admin_dashboard_uc_test.go`: decode JSON response cũ (không có 2 field) vẫn đọc đúng `distribution`/`totalUsers`; response mới có `dataAvailableFrom` đúng `YYYY-MM-DD`; proto decoding bỏ qua field lạ

### Frontend Integration Guide *(required — hiến pháp §Governance)*

- [x] T040 **Đồng bộ guide** sau khi mã ổn định: đọc lại `frontend-guide.md` và đối chiếu với `services/commerce/internal/modules/dashboard/application/dto/admin_dashboard_dto.go` + handler, **đếm** số field JSON, số giá trị enum, số route, số HTTP status; sửa mọi lệch và ghi rõ phần nào chưa kiểm được

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: bắt đầu ngay; chặn migration (T003) chỉ ở khía cạnh môi trường.
- **Foundational (Phase 2)**: sau Setup; **chặn toàn bộ user story**.
- **US1 (Phase 3)**: sau Foundational; không phụ thuộc US2/US3.
- **US2 (Phase 4)**: sau Foundational; độc lập với US1 (chỉ đọc repo + docs).
- **US3 (Phase 5)**: sau Foundational; kiểm chứng write path đã dựng ở T007/T008.
- **Polish (Phase 6)**: sau khi các story mong muốn xong.

### Within Each Story

- Test trước implement (T010–T012 trước T013–T020; T027–T029 trước/song song T031).
- Entity/repo trước use case; proto trước transport; DTO trước handler; swagger sau handler.
- US1 kết thúc bằng docs + frontend guide trước khi sang US2.

### Parallel Opportunities

- T001, T002.
- T004, T005 sau T003 (khác file).
- T010, T011, T012 (ba file test khác nhau) sau Foundational.
- T021, T022 (docs) song song với T013–T020.
- T024 độc lập với US1; T025/T026 sau T024.
- T027, T028, T029 (ba file/nhóm test) song song; T030 sau T007.
- T032, T033, T034 song song ở polish.

---

## Parallel Example: User Story 1

```text
T010  completeness unit tests        (admin_stats/distribution_completeness_test.go)
T011  distribution integration test  (persistence/user_plan_period_distribution_integration_test.go)
T012  SQL pin tests                  (persistence/admin_stats_query_sql_test.go)
```

```text
T021  docs/api/dashboard/dashboard-admin-api.md
T022  specs/031-user-plan-periods/frontend-guide.md
```

## Parallel Example: User Story 3

```text
T027  recorder transition unit tests
T028  lifecycle integration tests
T029  idempotency + rollback tests
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 + Phase 2 (schema, repo, recorder, DI) — đây là phần bắt buộc dù MVP chỉ có US1.
2. Phase 3 (US1) → chạy T010–T012, sau đó `make test-db-integration`.
3. Dừng và kiểm chứng (sau khi backdate cutover trong fixture): kỳ test với user hạ gói sau kỳ trả đúng; kỳ trước cutover trả `unavailable`.
4. Deploy/demo nếu sẵn sàng; US2/US3 có thể theo sau mà không đổi hợp đồng.

### Incremental Delivery

1. Foundation + US1 → giá trị chính (báo cáo đúng), có thể ship.
2. US2 → năng lực vận hành (tra cứu lịch sử + docs), không đổi API.
3. US3 → bằng chứng đầy đủ về write path (idempotency, rollback, concurrency) trước khi tuyên bố đóng nợ.
4. Polish → tài liệu hệ thống, deferred, hiệu năng, gate đầy đủ.

### Notes

- Không sửa `subscription.user_subscriptions`; mọi thay đổi là bảng mới.
- Không backfill lịch sử trước cutover; không tạo endpoint lịch sử từng user.
- `make hygiene-scan` chỉ chạy khi được yêu cầu rõ ràng.
- Commit theo phase; không push trước khi T037 xanh.
