# Tasks: Sổ đăng ký chiến dịch trong cơ sở dữ liệu

**Input**: Tài liệu thiết kế từ `specs/026-campaign-registry-db/`

**Prerequisites**: plan.md (bắt buộc), spec.md (bắt buộc cho user story), research.md, data-model.md, contracts/, quickstart.md

**Tests**: Có. Đặc tả yêu cầu điều kiện kiểm chứng chạy được (spec §11, quickstart V1–V7), nên mỗi câu chuyện có kiểm thử đơn vị và/hoặc tích hợp tương ứng.

**Organization**: Tác vụ nhóm theo user story để mỗi câu chuyện triển khai và kiểm chứng độc lập.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: chạy song song được — khác tệp **và** không phụ thuộc tác vụ chưa xong
- **[Story]**: câu chuyện (US1..US6) **chỉ** ở phase user story
- Phụ thuộc không đánh dấu `[P]` được nêu tường minh trong mô tả

## Path Conventions

- Go workspace đa mô-đun; tính năng chỉ chạm `services/account/` và `shared/`
- Migration: `services/account/database/migrations/`
- Module chiến dịch: `services/account/internal/modules/subscription/`
- Hợp đồng: `services/account/contract/admin/`

---

## Phase 1: Setup (Hạ tầng dùng chung)

**Purpose**: Hằng số và lỗi dùng chung cho mọi câu chuyện

- [x] T001 [P] Thêm `Campaign AuditTargetType = "campaign"` vào `shared/domain/constants/adminaudit/targettype.go`
- [x] T002 [P] Thêm `CampaignCreate`/`CampaignUpdate`/`CampaignClose` (`campaign.create`/`campaign.update`/`campaign.close`) vào `shared/domain/constants/adminaudit/action.go`
- [x] T003 [P] Thêm các hằng số lỗi `apperror` tiếng Việt cho sổ đăng ký (`ErrCampaignDuplicateCode`, `ErrCampaignInvalidField`, `ErrCampaignBudgetLocked`, `ErrCampaignVersionConflict`, `ErrCampaignRegistryWriteFailed`, `ErrCampaignAuditListFailed`, `ErrCampaignRegistryReadFailed`) vào `services/account/internal/modules/subscription/application/errors/errors.go`

---

## Phase 2: Foundational (Chặn mọi câu chuyện)

**Purpose**: Bảng sổ, seed, gỡ cấu hình, đổi nguồn đọc — mọi câu chuyện đều cần

**⚠️ CRITICAL**: Không câu chuyện nào bắt đầu được trước khi phase này xong

**⚠️ Đơn vị cut-over nguyên tử**: T007 (tạo bảng) → T008 (seed `launch-2026-10`) → T013 (gỡ khối YAML)
phải được triển khai **cùng nhau**. Gỡ YAML mà chưa seed thì production mất chiến dịch đang chạy.

- [x] T004 Tạo thực thể `Campaign` (code, plan_slug, quota, reserve, starts_at, ends_at, closed_at, version, created_at, updated_at) trong `services/account/internal/modules/subscription/domain/entities/campaign.go` (data-model.md §2.5)
- [x] T005 Khai báo `ICampaignRepository` trong `services/account/internal/modules/subscription/domain/repositories/campaign_repo.go` (`ListOpen` chỉ lọc `closed_at IS NULL` cho đường cấp gói và quét bù; `ListAll` cho API tra cứu để hiện được chiến dịch đã đóng; `GetByCode`, `GetByCodeForShare`, `Create`, `Update`, `Close`, `ExistsClaims`; **không** có `Delete`)
- [x] T006 Cài `campaign_repo_impl.go` trong `services/account/internal/modules/subscription/infrastructure/persistence/` (GORM: `SELECT ... FOR SHARE`, `UPDATE ... WHERE code=? AND version=?` trả số dòng, `EXISTS` trên `campaign_claims`) — phụ thuộc T004, T005
- [x] T007 Viết migration lược đồ `services/account/database/migrations/<ts>_add_campaigns.sql`: bảng + 5 CHECK + `PRIMARY KEY` + `ix_campaigns_starts_at` + hàm `campaigns_guard_immutable_budget()` + trigger `trg_campaigns_immutable_budget` + `-- +goose Down` (data-model.md §2.1–2.2)
- [x] T008 Viết migration seed `services/account/database/migrations/<ts>_seed_launch_campaign.sql`: `INSERT ... ON CONFLICT (code) DO NOTHING` (`launch-2026-10`, `premium-monthly`, 50, 5, `2026-10-02T00:00:00+07:00`, NULL) + `-- +goose Down` xoá hàng seed (data-model.md §2.3–2.4, FR-090). **Chạy ngay sau T007, trước T013**
- [x] T009 [P] Xoá `CampaignConfig` và `Config.Campaigns` khỏi `shared/config/config.go`; **giữ** `CampaignSweepConfig`
- [x] T010 [P] Xoá phần phân tích/ghi đè khối `campaigns` trong `shared/config/loader.go`; giữ `campaign_sweep`
- [x] T011 [P] Xoá `validateCampaigns`, `validateCampaign`, `campaignCodePattern` khỏi `shared/config/validation.go` và **xoá** `shared/config/campaign_validation_test.go`
- [x] T012 Viết lại `shared/config/campaign_config_files_test.go` **đảo chiều khẳng định** của 025 T106: cả hai tệp YAML phải **KHÔNG** có khoá `campaigns`, và **vẫn phải có** `campaign_sweep`. Bỏ các khẳng định "2 chiến dịch trong dev / 1 trong production". Bỏ lời gọi `validateCampaign` (đã bị T011 xoá). Chạy sau T011 và T013
- [x] T013 Xoá khối `campaigns` (kể cả `sample-2026-11`) khỏi `configs/config.yaml` và `configs/config.production.yaml`; giữ `campaign_sweep`. **Không** phải `[P]`: bước cuối của đơn vị cut-over T007 → T008 → T013, chạy sau T008
- [x] T014 [P] Xoá ghi chú về khối `campaigns` trong `.env.example` và `.env.production.example` (comment tiếng Anh)
- [x] T015 Đổi `campaignEvaluator` nhận `[]*entities.Campaign`; cập nhật `newCampaignEvaluator`/`evaluate`/`isAwake`/`selectCampaign` trong `services/account/internal/modules/subscription/application/usecase/campaign/admission_uc.go` (áp tie-break `code DESC` một lần, theo spec Câu 6) — phụ thuộc T004, T005
- [x] T016 Đổi nguồn chiến dịch sang repository và đổi kiểu tham số `campaign` từ `sharedconfig.CampaignConfig` sang `*entities.Campaign` trong **hai** tệp: `campaign_uc.go` (đọc từ `ICampaignRepository` thay `cfg.Campaigns`, giữ nguyên `plan_unavailable`) và `grant_uc.go` (`applyGrant` + `logGranted` vẫn nhận kiểu cũ và sẽ không biên dịch nếu bỏ sót), trong `services/account/internal/modules/subscription/application/usecase/campaign/` — phụ thuộc T015
- [x] T017 Đổi `CampaignQueryUseCase` đọc sổ từ repository thay `cfg.Campaigns` trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_query_uc.go` — phụ thuộc T015
- [x] T018 Đổi lời gọi `newCampaignEvaluator(uc.cfg.Campaigns, now)` trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_query_claims_uc.go` sang nguồn repository (endpoint `GET /campaigns/:code/claims`) — phụ thuộc T015, T017
- [x] T019 Đổi `CampaignSweepUseCase` đọc sổ từ repository (`ListOpen`) thay `cfg.Campaigns` trong `services/account/internal/modules/subscription/application/usecase/campaign/sweep_uc.go` — phụ thuộc T015
- [x] T020 Đăng ký `ICampaignRepository`/impl vào Wire: `services/account/internal/modules/subscription/infrastructure/provider.go`, `services/account/internal/modules/subscription/provider.go`, `services/account/internal/di/app.go`, `services/account/internal/di/wire_gen.go` (`make wire`) — phụ thuộc T006
- [x] T021 Đổi kiểu sang `entities.Campaign` trong **toàn bộ** fake/harness và kiểm thử của gói campaign, vì các tệp này đang dựng `[]sharedconfig.CampaignConfig{}` và sẽ không biên dịch sau T015: `campaign_fakes_test.go`, `campaign_harness_test.go`, `campaign_query_fakes_test.go`, `campaign_sweep_fakes_test.go`, `campaign_sweep_health_test.go`, `campaign_query_status_test.go`, `campaign_query_eligibility_test.go`, `campaign_sweep_test.go`, `campaign_admission_test.go`, `campaign_quota_test.go`, `campaign_admin_exclusion_test.go`, `campaign_grant_test.go`, `campaign_multi_campaign_test.go` — phụ thuộc T015
- [x] T022 Kiểm thử tích hợp repository trên CSDL thật (`GetByCode`, `GetByCodeForShare`, `Update` sai version trả 0 dòng, `ExistsClaims`, `ListOpen` loại `closed_at`, và **không** có đường xoá) trong `services/account/internal/modules/subscription/infrastructure/persistence/campaign_repo_integration_test.go` — phụ thuộc T006, T007, T008
- [x] T023 Kiểm thử đơn vị bộ đánh giá với `entities.Campaign`: chọn `starts_at` lớn nhất, tie-break `code DESC` tất định, loại chiến dịch `closed_at` khác NULL, cửa sổ theo `registered_at` trong `services/account/internal/modules/subscription/application/usecase/campaign/admission_uc_test.go` — phụ thuộc T021
- [x] T024 Kiểm thử hồi quy FR-042 và FR-043: chạy lại **toàn bộ** bộ kiểm thử chiến dịch của 025 (`campaign_grant_test.go`, `campaign_quota_test.go`, `campaign_multi_campaign_test.go`, `campaign_admission_test.go`, `campaign_sweep_test.go`) và chứng minh các điều kiện 1–9 của `data-model.md` §6 giữ nguyên hành vi — điều kiện chống trùng, `grant_bucket`, `fills_main_quota`, mốc ưu tiên, trần `quota + reserve` — sau khi đổi nguồn — phụ thuộc T021
- [x] T025 Thêm kiểm thử chống FR-003/SC-010: `shared/config` sau khi gỡ **không** import `gorm.io/gorm`, `database/sql` hay bất kỳ driver CSDL nào, và `Load` cấu hình không mở kết nối — phụ thuộc T009, T010
- [x] T026 Thêm kiểm thử SC-008: sổ rỗng ⇒ hệ thống khởi động, bộ nhận thông báo và worker quét bù vẫn lên lịch, **0** cảnh báo giả về chiến dịch — phụ thuộc T016, T019
- [x] T027 Thêm kiểm thử chống FR-091: `subscription.campaigns` **không** có khoá ngoại nào; `campaign_claims` và `campaign_sweep_runs` không đổi lược đồ (so số cột và chỉ mục trước/sau) trong `services/account/internal/modules/subscription/infrastructure/persistence/campaign_schema_guard_integration_test.go` — phụ thuộc T007, T008
- [x] T028 Thêm kiểm thử chống FR-015: mọi phép so sánh `now` trong gói chiến dịch đi qua `cfg.Database.TimeZone`; dựng một chiến dịch với mốc thời gian ngay cạnh biên và chứng minh kết quả không lệch theo múi giờ máy chủ, trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_timezone_test.go` — phụ thuộc T016

**Checkpoint**: `make wire && make lint && go test -race -count=3 ./services/account/internal/modules/subscription/... ./shared/config/...` xanh; `make migrate-up svc=account` chạy được; hệ thống biên dịch khi không còn khối `campaigns`

---

## Phase 3: User Story 1 — Mở chiến dịch mới ngay lập tức, không deploy (Priority: P1) 🎯 MVP

**Goal**: Tạo chiến dịch qua API quản trị; tài khoản đăng ký sau đó được cấp gói; không sửa mã nguồn, không deploy.

**Independent Test**: Gọi `POST /api/v1/admin/campaigns` với `starts_at` trong quá khứ, đăng ký tài khoản mới, kiểm tra tài khoản nhận đúng gói. Không đổi tệp mã nguồn nào.

- [x] T029 [P] [US1] Thêm DTO request `CampaignCreateReq` (code, planSlug, quota, reserve, startsAt, endsAt) vào `services/account/dto/subscription/campaign_dto.go`
- [x] T030 [P] [US1] Viết `registry_validate.go` trong `services/account/internal/modules/subscription/application/usecase/campaign/` theo bảng FR-054: `400` cho `code` sai định dạng và thời gian không phải RFC3339 có múi giờ; `422` cho `planSlug` rỗng, `quota < 1`, `reserve < 0`, `endsAt <= startsAt`
- [x] T031 [US1] Khai báo `ICampaignRegistryUseCase` (Create/Update/Close) trong `services/account/internal/modules/subscription/application/interface/usecase/campaign_registry_uc.go`
- [x] T032 [US1] Viết mapper `campaign_mapper.go` (entity → `CampaignSummaryRes`) trong `services/account/internal/modules/subscription/application/mapper/` — phụ thuộc T029
- [x] T033 [US1] Cài `registry_uc.go` — `Create`: kiểm hình dạng, insert, ghi audit `campaign.create` qua `IAuditLogContract` trong cùng transaction (`uow.Execute`), dùng **hằng** `adminaudit.Campaign`/`adminaudit.CampaignCreate` và `payload.campaignCode` — phụ thuộc T030, T031, T032
- [x] T034 [US1] Viết handler `campaign_registry_handler.go` — `Create` trong `services/account/internal/modules/subscription/presentation/handler/` (mỏng: bind, gọi use case, `shared_pres.Success`) — phụ thuộc T031
- [x] T035 [US1] Đăng ký route `POST /api/v1/admin/campaigns` và wire handler/use case vào `services/account/internal/modules/subscription/presentation/router/router.go`, `.../handler` provider, `services/account/internal/di/wire_gen.go` (`make wire`) — phụ thuộc T034
- [x] T036 [US1] Kiểm thử đơn vị `registry_uc.go` Create: hợp lệ ⇒ tạo + đúng **một** lệnh ghi audit; trùng mã ⇒ `409`; hình dạng sai ⇒ `400`/`422` đúng loại; lỗi ghi ⇒ rollback và **không** có dòng outbox audit (FR-061) trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_create_test.go` — phụ thuộc T033
- [x] T037 [US1] Kiểm thử hợp đồng `POST /campaigns` trả đủ mã `201`/`400`/`409`/`422`/`403`/`429` trong `services/account/internal/modules/subscription/presentation/handler/campaign_registry_router_test.go` — phụ thuộc T035
- [x] T038 [US1] Kiểm thử tích hợp quickstart V1 phần mở: tạo chiến dịch rồi mô phỏng thông điệp `identity.user.created` cho tài khoản mới ⇒ có lượt cấp, trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_db_grant_integration_test.go` — phụ thuộc T033

**Checkpoint**: Tạo chiến dịch qua API và tài khoản mới nhận gói — MVP chạy được, kiểm chứng độc lập.

---

## Phase 4: User Story 2 — Đóng chiến dịch sớm (Priority: P1)

**Goal**: Đóng cưỡng bức qua API; chiến dịch ngừng cấp ngay; tài khoản đã cấp giữ nguyên.

**Independent Test**: Đóng một chiến dịch đang chạy, đăng ký tài khoản mới, kiểm tra không nhận gói và `reason = campaign_sleeping`.

- [x] T039 [P] [US2] Thêm trường `ClosedAt *time.Time` (`closedAt`) vào `CampaignSummaryRes` và hằng `CampaignStatusClosed = "closed"` vào `services/account/dto/subscription/campaign_dto.go`
- [x] T040 [US2] Thêm nhánh `closed` (ưu tiên cao nhất, kiểm **trước** `expired`) vào `deriveCampaignStatus` trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_query_uc.go` — phụ thuộc T039
- [x] T041 [US2] Cài `Close` trong `registry_uc.go` theo đúng thứ tự: `GetByCode` trong transaction (không có ⇒ `404`) → đã đóng thì trả `200` idempotent, không ghi thêm audit, không so phiên bản → `UPDATE ... WHERE version=?` (0 dòng ⇒ `412`); ghi `closed_at`, tăng `version`, audit `campaign.close` kèm `payload.reason` — phụ thuộc T033
- [x] T042 [US2] Thêm DTO request `CampaignCloseReq` (version, reason) vào `services/account/dto/subscription/campaign_dto.go`
- [x] T043 [US2] Thêm handler `Close` (`reason` rỗng ⇒ `422`) vào `services/account/internal/modules/subscription/presentation/handler/campaign_registry_handler.go` và route `POST /:code/close` — phụ thuộc T041, T042
- [x] T044 [US2] Cài FR-045: đọc lại hàng chiến dịch `FOR SHARE` trong transaction cấp gói (`GetByCodeForShare`), bỏ qua không ghi nếu `closed_at != NULL`, trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_uc.go` và `quota_uc.go` — phụ thuộc T006, T016
- [x] T045 [US2] Kiểm thử đơn vị `Close`: idempotent không ghi audit lần hai; `reason` rỗng ⇒ `422`; mã lạ ⇒ `404`; version lệch ⇒ `412`; `version` tăng; audit có `payload.reason` trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_close_test.go` — phụ thuộc T041
- [x] T046 [US2] Kiểm thử trạng thái `closed` ưu tiên `expired`/`exhausted`/`compensation` trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_query_status_test.go` — phụ thuộc T040
- [x] T047 [US2] Kiểm thử chống FR-033: đóng một chiến dịch **đã cấp** ⇒ `user_subscriptions` của tài khoản đã cấp không đổi (`current_plan_code` và `expires_at` giữ nguyên) trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_db_close_granted_integration_test.go` — phụ thuộc T041
- [x] T048 [US2] Kiểm thử hợp đồng `POST /campaigns/:code/close` trả đủ mã `200`/`400`/`404`/`412`/`422`/`403`/`429` trong `services/account/internal/modules/subscription/presentation/handler/campaign_registry_router_test.go` — phụ thuộc T043
- [x] T049 [US2] Kiểm thử tích hợp quickstart V1/V2: sau đóng **0** lượt cấp mới; đọc lại `FOR SHARE` chặn lượt đang xử lý, trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_db_close_integration_test.go` — phụ thuộc T044

**Checkpoint**: US1 + US2 chạy độc lập; đóng dừng cấp tức thời, không cần deploy.

---

## Phase 5: User Story 3 — Bất biến ngân sách của chiến dịch đã cấp (Priority: P1)

**Goal**: `quota`/`reserve`/`plan_slug` sửa được khi chưa cấp, bị từ chối khi đã cấp; SQL trực tiếp bị trigger chặn.

**Independent Test**: Sửa `quota` khi chưa cấp → 200; cấp một lượt rồi sửa → 422; `UPDATE` thẳng → trigger lỗi.

- [x] T050 [US3] Thêm trường `BudgetLocked bool` (`budgetLocked`) và `Version int` (`version`) vào `CampaignSummaryRes` trong `services/account/dto/subscription/campaign_dto.go`, và tính `budgetLocked = grantedMainCount+grantedCompensationCount > 0` trong `services/account/internal/modules/subscription/application/mapper/campaign_mapper.go` — cùng đợi T051 vì cùng sửa một tệp DTO
- [x] T051 [US3] Thêm DTO `CampaignUpdateReq` (version, planSlug?, quota?, reserve?, startsAt?, endsAt?) vào `services/account/dto/subscription/campaign_dto.go` — cùng đợi T050 vì cùng sửa một tệp DTO
- [x] T052 [US3] Cài `Update` trong `registry_uc.go` theo thứ tự: `GetByCode` trong transaction (không có ⇒ `404`) → so từng trường trong thân với giá trị hiện có; **không trường nào đổi** ⇒ trả `200` no-op, **không** `UPDATE`, **không** tăng `version`, **không** ghi audit (FR-055/US3.4) → kiểm bất biến tầng ứng dụng (có lượt cấp + đổi một trong `plan_slug`/`quota`/`reserve` ⇒ `422` với thông báo hướng dẫn đóng và mở mã mới) → `UPDATE ... WHERE version=?` (0 dòng ⇒ `412`); audit `campaign.update` với `payload.before`/`payload.after` — phụ thuộc T033, T050
- [x] T053 [US3] Thêm handler `Update` vào `services/account/internal/modules/subscription/presentation/handler/campaign_registry_handler.go` và route `PATCH /:code` — phụ thuộc T052
- [x] T054 [US3] Kiểm thử chống FR-024: `PATCH` với thân chứa `code` không đổi được giá trị nào (mã bất biến) trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_code_immutable_test.go` — phụ thuộc T052
- [x] T055 [US3] Kiểm thử chống FR-005 và FR-034: `ICampaignRepository` không khai báo `Delete`/`Remove`, và trong `router.go` **không** có route xoá chiến dịch lẫn route mở lại chiến dịch đã đóng — kiểm bằng phản chiếu trên interface + liệt kê route, trong `services/account/internal/modules/subscription/presentation/handler/campaign_no_delete_route_test.go` — phụ thuộc T053
- [x] T056 [US3] Kiểm thử đơn vị `Update`: chưa cấp sửa được cả ba cột; đã cấp chặn ⇒ `422`; version lệch ⇒ `412`; mã lạ ⇒ `404`; audit có `before`/`after` đúng trường đổi; và **`PATCH` không trường nào đổi ⇒ `200`, `version` giữ nguyên, không có dòng outbox audit thứ hai** (FR-055, US3.4) trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_update_test.go` — phụ thuộc T052
- [x] T057 [US3] Kiểm thử tích hợp quickstart V3 trên CSDL thật: API trả `422`; `UPDATE subscription.campaigns SET quota = quota + 1 WHERE code = '...'` nâng `check_violation` và giá trị không đổi, trong `services/account/internal/modules/subscription/infrastructure/persistence/campaign_guard_integration_test.go` — phụ thuộc T052, T007
- [x] T058 [US3] Kiểm thử hợp đồng `PATCH /campaigns/{code}` trả đủ mã `200`/`400`/`404`/`412`/`422`/`403`/`429` trong `services/account/internal/modules/subscription/presentation/handler/campaign_registry_router_test.go` — phụ thuộc T053

**Checkpoint**: Bất biến ngân sách được giữ ở cả tầng ứng dụng lẫn tầng dữ liệu.

---

## Phase 6: User Story 6 — Chuyển `launch-2026-10` sang sổ không đổi nghĩa (Priority: P1)

**Goal**: Seed (T008, đã làm ở Phase 2) được chứng minh trung thực: số liệu tra cứu giống trước; rollback không mất `campaign_claims`.

**Independent Test**: So output 4 API đọc trước/sau migration; chạy migration hai lần; rollback rồi kiểm tra `campaign_claims` còn nguyên.

- [x] T059 [US6] Kiểm thử tích hợp seed: chạy `make migrate-up svc=account` hai lần không nhân đôi hàng; rollback bằng `make migrate-down svc=account` **hai lần** (`goose down` chỉ lùi **một** phiên bản mỗi lần: một lần cho migration seed, một lần cho migration lược đồ) ⇒ bảng `campaigns` biến mất nhưng `campaign_claims` giữ nguyên số dòng, trong `services/account/internal/modules/subscription/infrastructure/persistence/campaign_seed_integration_test.go` — phụ thuộc T008
- [x] T060 [US6] Kiểm thử quickstart V4: output 4 API đọc giống hệt trước/sau migration (đếm đã cấp, `watermarkAt`, `status`) trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_migration_fidelity_test.go` — phụ thuộc T059

**Checkpoint**: Cut-over production không đổi nghĩa; rollback an toàn.

---

## Phase 7: User Story 4 — Sửa thời gian linh hoạt (Priority: P2)

**Goal**: `starts_at`/`ends_at` sửa được bất kể đã cấp hay chưa, miễn `ends_at > starts_at`; dời `ends_at` về quá khứ làm chiến dịch ngừng cấp.

**Independent Test**: Sửa `ends_at` về tương lai → cấp lại; về quá khứ → ngừng cấp; `ends_at <= starts_at` → `422`.

- [x] T061 [US4] Hoàn thiện xử lý trường thời gian trong `registry_uc.go`: chỉ cập nhật trường có mặt trong thân; `endsAt: null` xoá ngày kết thúc; kiểm `ends_at > starts_at` **sau khi áp giá trị mới** (trường vắng mặt giữ giá trị cũ) — phụ thuộc T052
- [x] T062 [US4] Kiểm thử đơn vị sửa thời gian: dời `ends_at`, xoá `ends_at`, `ends_at <= starts_at` ⇒ `422`, không gửi `starts_at` thì `starts_at` giữ nguyên trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_time_test.go` — phụ thuộc T061
- [x] T063 [US4] Kiểm thử tích hợp: dời `ends_at` về quá khứ làm `deriveCampaignStatus` trả `expired` và điều kiện 2 trả `campaign_sleeping`; dời về tương lai thì cấp lại, trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_db_time_integration_test.go` — phụ thuộc T061

**Checkpoint**: Thời gian chiến dịch sửa linh hoạt, không phá lượt cấp đã ghi.

---

## Phase 8: User Story 5 — Nhật ký kiểm toán (Priority: P2)

**Goal**: Màn hình đọc nhật ký **ai / lúc nào / đổi gì** cho một chiến dịch.

**Independent Test**: Thực hiện tạo/sửa/đóng, đợi bộ phát hàng đỗi và consumer kiểm toán chạy xong (FR-060), rồi đọc `GET /api/v1/admin/campaigns/{code}/audit` và đối chiếu từng bản ghi.

- [x] T064 [P] [US5] Khai báo hợp đồng hẹp `ICampaignAuditLookup` (`ListByCampaignCode(ctx, code, page, limit)`) trong `services/account/contract/admin/campaign_audit_contract.go` (không đụng `IAuditLogContract` gRPC)
- [x] T065 [US5] Thêm `ListByCampaignCode` vào `services/account/internal/modules/admin/domain/repositories/admin_audit_log_repo.go` (lọc `target_type='campaign' AND payload->>'campaignCode' = ?`) và cài trong `services/account/internal/modules/admin/infrastructure/persistence/admin_audit_log_repo.go` — phụ thuộc T064
- [x] T066 [US5] Cài bộ chuyển đổi `ICampaignAuditLookup` trong module `admin` tại `services/account/internal/modules/admin/provider.go`, trả DTO `dto/admin.AdminAuditLogDTO` — phụ thuộc T065
- [x] T067 [US5] Thêm handler `GetCampaignAudit` vào `services/account/internal/modules/subscription/presentation/handler/campaign_admin_handler.go` và route `GET /:code/audit` (chỉ đọc, dùng `adminGrantRateLimit()`); handler tra `ICampaignRepository.GetByCode` **trước** và trả `404` bằng `ErrCampaignNotFound` nếu mã không tồn tại, **trước khi** gọi `ICampaignAuditLookup` (module `admin` không biết chiến dịch nào tồn tại và chỉ trả được trang rỗng); ánh xạ sang DTO `CampaignAuditLogRes` khai `targetId` **không** `omitempty` để phát `null` — phụ thuộc T066, T039
- [x] T068 [US5] Wire hợp đồng hẹp vào `CampaignAdminHandler` (`.../presentation/router/router.go`, `services/account/internal/di/wire_gen.go`, `make wire`) — phụ thuộc T067
- [x] T069 [US5] Thêm `CampaignAuditLogRes` vào `services/account/dto/subscription/campaign_dto.go` với `TargetID *uuid.UUID` khai `json:"targetId"` **không** `omitempty`, cùng mapper trong `services/account/internal/modules/subscription/application/mapper/campaign_mapper.go` — phụ thuộc T065
- [x] T070 [US5] Kiểm thử đơn vị `ListByCampaignCode` lọc đúng theo `target_type='campaign' AND payload->>'campaignCode' = ?`, trả `AdminAuditLogDTO` có `TargetID = nil`, và **không** có tác vụ nào trong `admin` tra cứu bảng `subscription.campaigns` — đặt trong `services/account/internal/modules/admin/infrastructure/persistence/admin_audit_log_campaign_test.go` — phụ thuộc T065
- [x] T071 [US5] Kiểm thử hợp đồng `GET /campaigns/{code}/audit` trả `200`/`403`/`404`/`429` trong `services/account/internal/modules/subscription/presentation/handler/campaign_audit_router_test.go` — phụ thuộc T068
- [x] T072 [US5] Kiểm thử tích hợp quickstart V7 (kịch bản đã có sẵn ở `spec.md` §11.8 và `quickstart.md`): tạo → sửa → đóng rồi đọc `GET /campaigns/{code}/audit`; kiểm `action` đúng `campaign.create`/`campaign.update`/`campaign.close`, **`targetId` phát ra `null` chứ không bị bỏ**, `payload.campaignCode`, `payload.before`/`after`, `payload.reason`, và `404` cho mã lạ — trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_db_audit_read_integration_test.go` — phụ thuộc T071

**Checkpoint**: Vòng đời chiến dịch truy vết được ai/lúc nào/đổi gì.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Tài liệu, swagger, và kiểm chứng đầu-cuối

- [x] T073 Cập nhật Swagger của dịch vụ `account`: thêm annotation cho 4 endpoint mới ở handler, rồi chạy `make swagger svc=account` + `make swagger-all` + `make swagger-combine` (hiến phán: union của các dịch vụ phải giữ mọi mục đã tài liệu) — phụ thuộc T043, T053, T067
- [x] T074 [P] Viết `specs/026-campaign-registry-db/frontend-guide.md`: 3 endpoint ghi + 1 endpoint đọc nhật ký (method, path, đánh dấu **mới**), trường bổ sung `closedAt`/`budgetLocked`/`version`, `status=closed`, tập `reason` 12 mã không đổi, 6 màn hình admin, và bảng **trước → sau** cho 4 API đọc
- [x] T075 Đồng bộ `frontend-guide.md` sau khi mã nguồn ổn định: đối chiếu **có đếm** mọi thẻ `json:"…"` mà DTO của 026 phát ra, mọi giá trị của mọi enum guide mô tả (so hai chiều với khối hằng Go), mọi route (dịch `:param` của gin sang `{param}`), mọi mã HTTP guide khẳng định; sửa mọi chỗ lệch — phụ thuộc T074
- [x] T076 Thêm kiểm thử chống FR-080 và FR-081: không có lớp cache mới cho sổ chiến dịch (không symbol cache nào tham chiếu `Campaign`), mỗi lượt cấp đọc sổ qua repository, và **không** có biến môi trường mới nào cho dữ liệu chiến dịch trong các tệp `.env*` — phụ thuộc T016
- [x] T077 [P] Ghi `specs/026-campaign-registry-db/deferred.md` cho mọi phát hiện ngoài phạm vi (ví dụ cache TTL tương lai), theo `.specify/templates/deferred-template.md`
- [x] T078 Chạy các kịch bản còn lại của `quickstart.md` trên CSDL kiểm thử: V5 (hợp đồng đọc không đổi — chỉ thêm `closedAt`/`budgetLocked`/`version`, đúng 12 mã `reason`), V6 (quét bù nhận chiến dịch mới, và `lastSweepAt IS NULL` trước lần quét đầu — FR-072), và nhánh `plan_unavailable` của FR-014/SC-009 — phụ thuộc T075
  - ⚠️ **Chạy một phần, không chạy trọn vẹn.** Lần chốt này bị cấm mọi lệnh CSDL (`make migrate-up`/`migrate-down`) và không có sẵn môi trường đăng ký tài khoản, nên **V5/V6/`plan_unavailable` chưa từng chạy trên CSDL thật**. Phần *đã làm* là bằng chứng thay thế bằng fake: `campaign_read_contract_shape_test.go` (khoá đúng 21 khoá JSON của `CampaignSummaryRes`, `targetId` luôn `null`, `lastSweepAt`/`watermarkAt`/`closedAt` null trước lần quét đầu) và `campaign_sweep_reads_registry_each_run_test.go` (quét bù nhận chiến dịch viết vào sổ *sau khi* use case đã dựng, trên cả hai cửa; sổ rỗng không ghi dòng sức khoẻ). Nhánh `plan_unavailable` đã có `[X] T024` phủ (`campaign_registry_cutover_regression_test.go`). Phần chưa thực hiện ghi ở `deferred.md` **D-01**, nêu rõ 10 task mà bằng chứng CSDL của chúng là *đã viết, chưa chạy*.
- [x] T079 Chạy `make check` đầy đủ một lần lúc chốt (7 bước: fmt-check, tidy-check, generate-check, nginx-check, vet, test-race, build-all)
  - **[X] bởi orchestrator.** Đã chạy `make check` sau khi `make generate` và **staging** tệp sinh ra (vì `generate-check` so cây làm việc với **index**, nên phải stage trước). Kết quả: cả 7 bước xanh — `fmt-check` sạch, `tidy-check` sạch, `generate-check` "Generated files are up to date", `nginx-check` xanh, `vet` sạch, `test-race` xanh trên cả 5 mô-đun, `build-all` thành công (3 binary).
- [x] T080 Xác nhận các ràng buộc ngoài phạm vi bằng bằng chứng máy đọc được: `git diff --stat -- services/ai services/commerce` **rỗng** (FR-101); `subscriptioneventtype.All` vẫn đúng **7** giá trị và không có `uow.Execute` lồng nhau trong `registry_uc.go`/`campaign_uc.go` (FR-044); `specs/025-` không bị sửa (FR-103); **không** thay đổi giá/quyền lợi/định nghĩa của bất kỳ gói cước nào (FR-100 — so `subscription_plans` trước/sau); và **không** có endpoint nào dưới `/api/v1` ngoài `/admin/campaigns` mới do 026 thêm, tức không có giao diện cho khách hàng cuối (FR-102)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: không phụ thuộc
- **Foundational (Phase 2)**: phụ thuộc Setup — **chặn mọi câu chuyện**
- **US1/US2/US3/US6 (P1)**: sau Foundational
- **US4 (P2)**: sau US3 (T052). **US5 (P2)**: sau Foundational; không phụ thuộc US4
- **Polish (Phase 9)**: sau khi các câu chuyện mong muốn xong

### User Story Dependencies

- **US1 (P1)** 🎯 MVP: sau Foundational; không phụ thuộc câu chuyện khác
- **US2 (P1)**: dùng chung `registry_uc.go` với US1/US3 → làm sau US1
- **US3 (P1)**: dùng chung `registry_uc.go` + `campaign_registry_handler.go` → làm sau US1/US2
- **US6 (P1)**: phần seed đã nằm ở T008 (Phase 2); US6 chỉ kiểm chứng → có thể làm song song với US1–US3
- **US4 (P2)**: sau US3 (T052)
- **US5 (P2)**: sau Foundational

### Trong mỗi câu chuyện

- DTO/validate trước use case; use case trước handler; handler trước route; route trước kiểm thử hợp đồng
- Kiểm thử tích hợp cần migration đã chạy (`make migrate-up svc=account`)

### Tệp dùng chung giữa nhiều câu chuyện

Các tệp sau bị nhiều phase sửa, và **US1 → US2 → US3 phải tuần tự** để không sửa đè:

| Tệp | Sửa ở |
|---|---|
| `dto/subscription/campaign_dto.go` | T029, T039, T042, T050, T051 |
| `application/usecase/campaign/registry_uc.go` | T033, T041, T052, T061 |
| `application/mapper/campaign_mapper.go` | T032, T050 |
| `presentation/handler/campaign_registry_handler.go` | T034, T043, T053 |
| `di/wire_gen.go` | T020, T035, T068 (chạy `make wire` mỗi lần) |

### Parallel Opportunities

- Setup: T001, T002, T003
- Foundational: T009, T010, T011, T014 (mỗi tệp một tệp). **T013 KHÔNG song song** — nó là bước cuối của
  đơn vị cut-over T007 → T008 → T013
- US1: T029, T030 · US2: T039 · US5: T064
- US6 chạy song song được với US1–US3 (tác vụ kiểm thử độc lập, không cùng sửa tệp với các phase đó)
- **Không** song song: T012 (sau T011+T013), T050/T051 (cùng một tệp DTO), T073 (sau 3 task), T076 (sau T016)

---

## Implementation Strategy

### MVP First (User Story 1 + Foundational)

1. Phase 1 Setup → Phase 2 Foundational (bắt buộc)
2. Phase 3 US1 — mở chiến dịch qua API, tài khoản mới nhận gói
3. **DỪNG và KIỂM CHỨNG**: quickstart V1 phần mở, không deploy
4. Có thể phát hành/demo — nhưng **cut-over production cần US6** (kiểm chứng seed) và **US2** (đóng) để vận hành an toàn

### Incremental Delivery

1. Setup + Foundational → nền sẵn sàng
2. US1 → kiểm chứng độc lập → giá trị đầu tiên
3. US2 → đóng sớm hoạt động
4. US3 → bất biến ngân sách được giữ
5. US6 → cut-over production không đổi nghĩa
6. US4 → sửa thời gian linh hoạt
7. US5 → nhật ký kiểm toán
8. Polish → guide, swagger, `make check`

### Ghi chú cut-over

- Migration **phải chạy trước** binary mới: T007 (bảng) rồi T008 (seed) là một đơn vị; T013 (gỡ YAML) chỉ
  được triển khai cùng đợt với hai migration đó, theo thứ tự nêu trong mô tả T008/T013.
- Khối YAML `campaigns` có thể tồn tại tạm trong một cửa sổ triển khai làm bản sao rollback, nhưng binary
  mới **không đọc** nó.
- Không bật email khi kiểm chứng nhiều tài khoản (cổng an toàn #2 của quickstart).

---

## Notes

- `[P]` = khác tệp và không phụ thuộc tác vụ chưa xong; phụ thuộc còn lại được nêu trong mô tả
- Tệp dùng chung giữa các phase: xem bảng ở mục trên; **US1 → US2 → US3 tuần tự**
- Không đụng `specs/025-`; mọi thay thế đã ghi trong `spec.md` mục 4
- Không sửa `services/ai`/`services/commerce`; không thêm loại sự kiện; không lồng `uow.Execute`
- Hợp đồng đặt ở `services/account/contract/admin/` theo tiền lệ 024 (`AdminGrantLookup`) và 025
  (`CampaignAccountLister`); xem `research.md` R8
- Cam kết sau mỗi phase khi cổng xanh; không commit khi chưa được cho phép

---

## Phase 10: Convergence

**Purpose**: Phần đặc tả mà `tasks.md` đã tick nhưng mã hoặc bằng chứng **chưa thật sự có mặt** trong kho.
Vòng lặp hội tụ, không phải vòng lặp phân tích: tài liệu thiết kế đã đóng băng, nên ở đây chỉ bổ sung
tác vụ cho mã và kiểm thử — **không** sửa `spec.md`, `plan.md`, `research.md`, `data-model.md`,
`contracts/` hay `quickstart.md`.

**Vì sao cần phase này.** Mười task đã tick (T022, T027, T038, T047, T049, T057, T059, T063, T072,
T078) dựa vào kiểm thử tích hợp CSDL mà `deferred.md` **D-01** đã thừa nhận là **chưa từng chạy**.
Giờ soát lại tĩnh, các kiểm thử đó còn một vấn đề lớn hơn nhiều: chúng **không cô lập sổ chiến dịch
khỏi hàng seed `launch-2026-10`**, và `baseInstant()` trùng đúng `starts_at` của hàng seed. Bốn
task dưới đây sửa đúng chỗ đó, thay vì thêm việc mới ở nơi khác.

- [x] T081 Cô lập sổ chiến dịch trong các fixture kiểm thử tích hợp CSDL của gói `campaign` khỏi hàng seed `launch-2026-10`, và khẳng định điều đó bằng một tiền điều kiện trong mỗi tệp, theo FR-031/FR-045/FR-014/FR-023/FR-001 (partial)
  - **Phục vụ**: US1/US2/US4 Independent Test, SC-001, SC-002, SC-009 — và là điều kiện tiên quyết cho T038, T047, T049, T063 đã tick.
  - **Bằng chứng chưa có**: `20261003165500_seed_launch_campaign.sql:35` chèn `launch-2026-10` với `starts_at = '2026-10-02T00:00:00+07:00'`, `ends_at = NULL`; `campaign_harness_test.go:26 baseInstant()` trùng **đúng** mốc đó. Mọi fixture tạo chiến dịch ở `baseInstant().Add(-24 * time.Hour)` (`campaign_db_close_integration_test.go:93`, `campaign_db_close_granted_integration_test.go:65`, `campaign_db_time_integration_test.go:67`, `campaign_db_grant_integration_test.go:259` và `:362`). `openGrantTestDB`/`openCloseTestDB` chạy migration nhúng nên hàng seed có mặt, và `t.Cleanup` chỉ xoá theo prefix của tệp — không tệp nào trung hòa nó.
  - **Vì sao hỏng**: `admission_uc.go:270` + `startsLaterThan` (`:279`) chọn `starts_at` **lớn nhất** và `isAwake` coi hàng seed là thức (không có `ends_at`), nên hàng seed ăn trước mọi chiến dịch fixture cho mọi tài khoản mà các kiểm thử này tạo.
  - **Khẳng định đang đứng trên giả định sai**: `campaign_db_grant_integration_test.go:306` (`claim.CampaignCode != campaignCode`) và `:379`; `campaign_db_close_granted_integration_test.go:82` và `:90` (tiền điều kiện "đã cấp" và "đúng gói"); `campaign_db_close_integration_test.go:201` và `:299` (`reason != campaign_sleeping`); `campaign_db_time_integration_test.go:267` và `:310`.
  - **Hướng làm**: trung hòa hàng seed ngay trong `t.Cleanup`/`t.Setup` của mỗi tệp (đóng hoặc xoá rồi khôi phục đúng giá trị gốc, **không** đụng `campaign_claims`), và thêm một tiền điều kiện khẳng định `ListAll` trả **đúng** các mã của tệp đó — để một fixture bị bẩn báo tên file thay vì trượt âm thầm vào một lời giải thích sai.
  - **[X] Đã làm.** Tệp mới `campaign_db_registry_isolation_test.go`: `isolateSeededCampaign` **xoá** hàng seed rồi **khôi phục nguyên vẹn cả 10 cột** trong `t.Cleanup` (kèm đối chiếu sau khôi phục; `campaign_claims` không bị đụng); `requireRegistryIsolatedTo` là tiền điều kiện — `ListAll` phải trả **đúng** các mã của tệp, và thông báo lỗi đọc **tên tệp** từ stack nên báo đúng fixture. Cả ba opener (`openGrantTestDB`, `openCloseTestDB`, `openAuditTestDB`) gọi cô lập **trước** khi đăng ký cleanup theo prefix, nên hàng seed về lại **sau cùng**. Tiền điều kiện được gọi ở cả **5** tệp CSDL của gói: grant (2), close (3, qua `openRunningCampaign`), close-granted (2), time (3), audit (4).
  - ⚠️ **Chưa chạy trên CSDL thật.** `TEST_DATABASE_URL` không được đặt, nên 15 kiểm thử `TestIntegration*` của gói đều `t.Skip` và **không khẳng định nào** trong T081 được thực thi. Phần bằng chứng thay thế là **đối chiếu tĩnh**: hàng seed có `starts_at = baseInstant()` và mọi fixture tạo ở `baseInstant()-24h`, nên FR-040 chọn hàng seed — đúng như mô tả "Vì sao hỏng" ở trên. Xem `deferred.md` **D-01**.
  - ⚠️ **Một khẳng định cũ hoá, đã sửa và phải nói rõ.** `campaign_db_close_integration_test.go` trước đây đòi `campaign_sleeping` cho tài khoản đến sau khi đóng (`reason != SkipCampaignSleeping`). Sau khi cô lập, chiến dịch của tệp là hàng **duy nhất** trong sổ, `ListOpen` trả về rỗng, và câu trả lời của đường thông điệp là `campaign_not_configured` (`admission_uc.go:81`) — hành vi này là **đã chốt**, không sửa mã sản xuất. Khẳng định được đổi sang `SkipCampaignNotConfigured` kèm một khẳng định bổ sung (`ListOpen` thực sự rỗng) và ghi chú trỏ tới hai nơi `campaign_sleeping` **vẫn** được chứng minh cho chiến dịch đã đóng: `TestEvaluatorExcludesForceClosedCampaigns` (`admission_uc_test.go`) và tra cứu điều kiện (`campaign_query_eligibility_test.go`).
  - ⚠️ **Lỗi tài liệu phát hiện được, CẦN NGƯỜI QUYẾT — không tự sửa `quickstart.md`.** V2 bước 2 của `quickstart.md:192` yêu cầu "API tra cứu trả `eligibility = not_eligible`, `reason = campaign_sleeping`" cho một chiến dịch `closed_at IS NOT NULL`. Với `ListOpen` lọc `closed_at` (đã đóng băng từ T016), sổ **chỉ có** chiến dịch đã đóng thì **cả hai** đường — thông điệp lẫn tra cứu — đều trả `campaign_not_configured`, vì `campaign_query_claims_uc.go:193` cũng đọc `ListOpen`. `campaign_sleeping` chỉ đạt được khi sổ còn **một** chiến dịch khác *không đóng* và *không thức* (đã qua `ends_at`), đúng cái thiết lập mà V2 không mô tả. Hệ quả: khẳng định `campaign_sleeping` cũ ở `campaign_db_close_integration_test.go` **không bao giờ quan sát được** mã đó trên CSDL thật — với hàng seed còn mở, FR-040 chọn hàng seed và tài khoản bị **cấp** từ nó (nếu có plan `premium-monthly`) hoặc trả `plan_unavailable` (nếu không); còn sau khi cô lập thì đáp án là `campaign_not_configured`. Cả `spec.md` lẫn `quickstart.md` đều **không** sửa ở phase này.
  - ⚠️ **Cửa sổ chạy song song giữa hai gói (chưa sửa, ngoài phạm vi).** `make test-integration` chạy `go test ./internal/modules/subscription/...`, tức các **binary gói khác nhau chạy song song** trên cùng một `TEST_DATABASE_URL`. Trong lúc hàng seed bị xoá, `campaign_seed_integration_test.go:TestIntegrationSeededCampaignCarriesTheDocumentedValues` ở gói `persistence` sẽ không thấy hàng đó. Cửa sổ bằng thời lượng một fixture và cần hai kiểm thử trùng nhau, nên mô tả trung thực là **flake hiếm**, không phải "bộ kiểm thử hỏng". Đóng cửa sổ này cần một khoá mà **cả hai** gói cùng nắm (session-level `pg_advisory_lock` trên một `*sql.Conn` đã ghim), tức là sửa thêm các opener của gói `persistence` — nằm ngoài phạm vi T081 nên **không** tự sửa.

- [x] T082 Thêm kiểm thử đơn vị cho cổng chặn của FR-045 trên đường cấp gói: chiến dịch bị đóng **sau** khi điều kiện 1–7 đã chạy thì lượt đó bị bỏ qua, không ghi lượt cấp và không đổi gói cước của tài khoản, theo FR-045/FR-031 (partial)
  - **Phục vụ**: FR-045, FR-031, SC-002, US2 kịch bản 1–2, edge case "Đóng chiến dịch đúng lúc một thông điệp đang xử lý", quickstart V2.
  - **Bằng chứng chưa có**: cổng chặn nằm ở `quota_uc.go:123`–`:145` (`GetByCodeForShare` → `SkipCampaignSleeping`), nhưng khẳng định duy nhất về nó là `campaign_db_close_integration_test.go:228` — tệp `t.Skip` khi thiếu `TEST_DATABASE_URL` (`deferred.md` D-01) **và** vô dụng theo T081. Không có kiểm thử nào chạy được.
  - **Diễn đạt được bằng fake**: `fakeCampaignRepo.GetByCodeForShare` (`campaign_registry_fakes_test.go:162`) trả hàng sống, và `fakeClaimRepo.LockClaimWindow` — hàm mà `quota_uc.go:99` gọi **trước** lần đọc lại — là chỗ đóng campaign giữa hai thời điểm. `shareCalls` đã được instrument sẵn để chứng minh đường sản xuất thật sự gọi lệnh đọc có khoá.
  - **[X] Đã làm.** Tệp mới `campaign_fr045_gate_test.go`, kiểm thử `TestACampaignClosedBetweenEvaluationAndTheClaimWriteGrantsNothing`. Fake được mở rộng bằng hai trường `closeOnShareCode` / `budgetOnShare`, áp dụng **đúng tại thời điểm `GetByCodeForShare` phục vụ lệnh đọc có khoá** — tức sau điều kiện 1–7 và trước các phép đếm; hàng được **thay thế** chứ không sửa tại chỗ, nên bản copy mà evaluator đã chọn vẫn mang giá trị cũ và "quyết định từ hàng đã đọc lại" trở thành khẳng định máy đọc được. Khẳng định: `shareCalls == 1`, khoá tư vấn đã lấy **trước**, sổ lượt cấp rỗng, không publish đổi gói, không có dòng lịch sử, gói cước tài khoản không đổi, và reason `campaign_sleeping`.
  - **Bằng chứng phản chứng (đã chạy).** Tạm vô hiệu `if locked.ClosedAt != nil` trong `quota_uc.go` → kiểm thử **FAIL** với 6 lỗi; đã khôi phục mã sản xuất và xác nhận `go vet` + toàn gói xanh.

- [x] T083 Thêm kiểm thử cho vế thứ hai của FR-045: ngân sách đổi giữa lúc đánh giá và lúc ghi lượt cấp thì phải quyết định lại trên hàng đã đọc lại chứ không dùng `quota`/`reserve` cũ, theo FR-045 (missing)
  - **Phục vụ**: FR-045 (mệnh đề "hoặc ngân sách đã thay đổi khiến lượt không còn hợp lệ"), tương tác với FR-020/FR-021, edge case "Sửa ngân sách đúng lúc một lượt cấp đang xử lý ⇒ hai thao tác nối tiếp nhau … không có kết quả nửa vời".
  - **Bằng chứng chưa có**: mã **đã** cài (`quota_uc.go:145 in.Campaign = locked`, rồi `chooseBucket` đọc `quota`/`reserve` của hàng đã nạp lại tại `:215`–`:216`) nhưng **không kiểm thử nào** — kể cả kiểm thử CSDL — phủ vế này. Nó là vế khó hơn của FR-045 và cũng là vế duy nhất chỉ có mã, không có khẳng định.
  - **[X] Đã làm.** Cùng tệp `campaign_fr045_gate_test.go`, ba kiểm thử dùng chung một fixture `newShrunkenBudgetFixture(t, shrink)` để hai lần chạy **chỉ khác đúng một thứ** là bản sửa ngân sách: (1) `TestACampaignWhoseBudgetShrankBeforeTheClaimWriteGrantsNobody` — hàng đã đọc là `quota 3` với sổ đã có 2 lượt chính, lần đọc lại trả `quota 1` ⇒ **bỏ qua**, không ghi lượt cấp, không publish, watermark vẫn đúng 1, reason `quota_compensation`; (2) `TestTheSameAccountIsGrantedWhenNothingChangesTheBudget` — **đối chứng** của (1), được cấp từ ngân sách chính; (3) `TestTheWatermarkFollowsTheBudgetThatWasReread` — sổ rỗng, `quota 3 → 1`, lượt cấp phải mang `fills_main_quota` vì nó **là** lượt lấp đầy ngân sách chính; tính từ `quota 3` cũ thì không mang, tức là watermark FR-030 bị gán sai.
  - ⚠️ **Trạng thái fixture này bị tầng dữ liệu cấm trong vận hành** (`trg_campaigns_immutable_budget` giữ ngân sách khi đã có lượt cấp). Điều đó được nêu ngay trong doc của kiểm thử: vế FR-045 này vẫn đáng chứng minh vì hợp đồng của bộ quyết định là "quyết định từ hàng vừa đọc lại", chứ không phải "giả định không ai sửa". Riêng kiểm thử (3) thì **hợp lệ** trong vận hành vì sổ vẫn rỗng.
  - **Bằng chứng phản chứng (đã chạy).** Tạm bỏ `in.Campaign = locked` trong `quota_uc.go` → cả (1) và (3) **FAIL**, còn (2) vẫn xanh (đúng như thiết kế đối chứng); đã khôi phục mã sản xuất.

- [x] T084 Thêm kiểm thử đơn vị chứng minh worker quét bù thôi quét chiến dịch đã đóng và hàng sức khoẻ lẫn sổ lượt cấp của chiến dịch đó không bị xoá, theo FR-070/FR-072 và spec.md Câu 9 (missing)
  - **Phục vụ**: FR-070, FR-072, SC-007, Câu 9 ("Chiến dịch đã đóng thôi được quét; hàng sức khoẻ và sổ lượt cấp của nó không bị xoá"), edge case "Chiến dịch bị đóng giữa hai lần quét".
  - **Bằng chứng chưa có**: `Select-String "closed"` trên toàn bộ `campaign_sweep*_test.go` không khớp dòng nào — hướng "chiến dịch tạo giữa chừng" đã có (`campaign_sweep_reads_registry_each_run_test.go:38`), hướng "chiến dịch đã đóng" thì không có gì cả. Lưu ý sự im lặng này là do thiết kế: `fakeCampaignRepo.ListOpen` (`campaign_registry_fakes_test.go:130`) **tự** lọc `closed_at`, nên một fixture đặt `closedCampaign(...)` sẽ biến mất khỏi danh sách quét mà không có khẳng định nào buộc phải giải thích vì sao.
  - **Cần khẳng định**: (a) lần quét sau khi đóng không quét chiến dịch đó; (b) `campaign_sweep_runs` của nó vẫn còn và **không** bị `RecordRun` ghi đè/xoá; (c) sổ lượt cấp của nó vẫn nguyên; (d) `lastSweepAt` của một chiến dịch **chưa từng** quét là `null` và `degraded = false` — vế (d) hiện mới chỉ có ở phía đọc (`campaign_read_contract_shape_test.go:295`), không có ở phía sweep.
  - **[X] Đã làm.** Tệp mới `campaign_sweep_closed_campaign_test.go`, hai kiểm thử:
    - `TestASweepStopsScanningACampaignThatWasClosedBetweenRuns` — **(a)(b)(c)**. Fixture có **hai** chiến dịch: cái đóng giữa chu kỳ có `starts_at` muộn hơn nên thắng FR-040 ở lần quét một; cái còn mở là "người sống sót" chứng minh lần quét hai **thực sự quét và cấp** (không phải "không làm gì"), và chiến dịch đóng không được chọn. `sweep.now` dời tiếp 15 phút giữa hai lần quét — nếu không, hàng sức khoẻ ghi đè sẽ mang **cùng** mốc thời gian và khẳng định (b) sẽ rỗng. Khẳng định: `Scanned` đủ, lượt cấp mới đến từ chiến dịch còn mở, `ListOpen` không chứa chiến dịch đã đóng, hàng `campaign_sweep_runs` **vẫn còn** với `last_sweep_at` **bằng mốc lần quét một** và `degraded = false`, `writeCount` đúng 2, và toàn bộ lượt cấp của chiến dịch đó — **kể cả dòng watermark FR-030** — được so từng trường.
    - `TestACampaignClosedBeforeItsFirstSweepIsNeverRecorded` — **(d)** ở cả hai đầu: phía sweep thì **không** có hàng sức khoẻ nào (`runs.row(...) == nil`) sau hai lần quét, phía đọc thì trang quản trị trả `lastSweepAt = null`, `degraded = false`, `degradedReason = ""`. Chiến dịch đóng trước lần quét đầu có **cửa sổ vẫn mở**, nên nếu câu trả lời là "ngủ vì cửa sổ đóng" thì kiểm thử vẫn xanh vì lý do sai.
  - **Bằng chứng phản chứng (đã chạy).** Tạm bỏ điều kiện lọc `ClosedAt == nil` trong `fakeCampaignRepo.ListOpen` → **cả hai** kiểm thử FAIL, đúng vào các khẳng định FR-070 (hàng sức khoẻ bị ghi đè; hàng của chiến dịch chưa từng quét bị tạo ra). Đã khôi phục fake và xác nhận gói xanh.

---

## Phase 11: Convergence (lần 2)

**Purpose**: Phần đặc tả mà `tasks.md` đã tick nhưng **bằng chứng duy nhất nằm trong kiểm thử `t.Skip` khi thiếu CSDL**, hoặc nằm sau một cổng chặn khiến nó không bao giờ chạy. Đây là cùng một lớp phát hiện như Phase 10, ở tầng sâu hơn: Phase 10 sửa *fixture sai*, Phase 11 bổ sung *bằng chứng thay thế chạy được* cho những yêu cầu mà bằng chứng gốc chưa từng thực thi.

**Vì sao cần phase này.** Phase 10 đã dựng tiền lệ: mỗi khi một khẳng định chỉ có thể chứng minh trên CSDL thật, hãy viết một dạng **máy đọc được, chạy trong `go test` thường**. Bốn task dưới đây làm đúng điều đó cho bốn yêu cầu còn nằm trong `deferred.md` **D-01** mà chưa có dạng thay thế nào. Lưu ý: đây **không** phải ngoại lệ hợp đồng cho D-01 — kiểm thử CSDL gốc vẫn phải chạy một lần; các task này chỉ bảo đảm rằng trong lúc chờ, một lần sửa sai **không trượt âm thầm**.

**Điều kiện chung cho cả bốn task.** Tiền lệ kỹ thuật đã có sẵn trong chính kho này, không cần hạ tầng mới: `campaign_seed_integration_test.go` đọc section `Up`/`Down` của migration **nhúng** (`accountmigrations.EmbedFS`) qua `migrationSection` và `stripSQLComments`, và hai kiểm thử sinh ra từ cách đó **chạy được không cần CSDL** (đã kiểm chạy: `TestIntegrationRegistryDownSectionsDoNotTouchTheClaimLedger` và `TestIntegrationSeedDownIsScopedToTheSeededRow` → PASS). Vì sao đọc **file nhúng** chứ không đọc thư mục: chính tệp đó đã viết lý do — đây là tập mà binary mang theo và `RunMigrations` áp dụng.

- [x] T085 Khẳng định tĩnh rằng hai migration của 026 **không** khai báo khoá ngoại nào và **không** `ALTER` hai bảng của 025, theo FR-091 (missing)
  - **Phục vụ**: FR-091 (`MUST NOT` thêm khoá ngoại, `MUST NOT` đổi lược đồ hai bảng đó), Câu 5 ("không thêm khoá ngoại từ `campaign_claims`/`campaign_sweep_runs` sang `campaigns`"), research.md R5.
  - **Bằng chứng chưa có — và đây là toàn bộ bằng chứng**: năm khẳng định duy nhất về FR-091 nằm trong `campaign_schema_guard_integration_test.go`, và **cả năm** đều gọi `openRegistryTestDB(t)` ở dòng đầu tiên (`:34`, `:61`, `:84`, `:109`, `:138`). `openRegistryTestDB` là `if dsn == "" { t.Skip(...) }` (`campaign_repo_integration_test.go:58`–`:61`). Không `TEST_DATABASE_URL` ⇒ **0/5** khẳng định này được thực thi. Đây là phần của D-01 chưa có dạng thay thế: T081 sửa *fixture* của các kiểm thử CSDL, không sửa *tính chất không-thể-chạy* của các kiểm thử lược đồ.
  - **Vì sao hỏng, và vì sao nó là điều kiện tiên quyết cho US6**: một `FOREIGN KEY campaign_claims.campaign_code → campaigns.code` biên dịch và chạy trôi trên môi trường trống, chỉ lộ ra trên dữ liệu thật — nơi sổ lượt cấp có thể chứa mã không nằm trong sổ (giả định §10 của `spec.md`). Đó cũng chính là lý do Câu 5 từ chối khoá ngoại: nó có thể **chặn migration trên dữ liệu thật**, biến cut-over "không đổi nghĩa" thành bước rủi ro nhất.
  - **Hướng làm**: đọc `migrationSection(t, registrySchemaMigration, "Up")` và `migrationSection(t, registrySeedMigration, "Up")`, `stripSQLComments` rồi khẳng định **vắng** `FOREIGN KEY`, `REFERENCES`, và `ALTER TABLE`/`ALTER COLUMN`/`DROP` nhắm `campaign_claims` hoặc `campaign_sweep_runs`. Phải strip comment **bắt buộc**: chính phần giải thích cũng nhắc `campaign_claims` ("no foreign key from the ledger to this table"), và đó là lý do `stripSQLComments` đã tồn tại — đừng viết lại một cái grep thô sẽ FAIL đúng vào câu tài liệu hoá sự an toàn đó.
  - **Phạm vi**: chỉ phần `Up`. Phần `Down` đã có `TestIntegrationRegistryDownSectionsDoNotTouchTheClaimLedger` (FR-091/FR-092) và test đó **chạy được**.
  - **[X] Đã làm.** Tệp mới `campaign_registry_migration_text_test.go` (cùng package `persistence`), kiểm thử `TestTheRegistryUpSectionsDeclareNoForeignKeyAndLeaveThe025TablesAlone`. Đọc **phần `Up`** của cả hai migration từ `accountmigrations.EmbedFS`, `stripSQLComments` rồi `collapseWhitespace`, và khẳng định vắng `FOREIGN KEY`/`REFERENCES` cùng **8 dạng `reach` neo trên tên bảng** — `ALTER TABLE`, `DROP TABLE`, `TRUNCATE`, `RENAME TO`, `SET SCHEMA`, `CREATE [UNIQUE] INDEX … ON`, `CREATE [OR REPLACE] TRIGGER … ON`, `INSERT INTO`/`UPDATE`/`DELETE FROM`/`MERGE INTO` — nhắm `campaign_claims` hoặc `campaign_sweep_runs`.
  - **Vì sao neo trên tên bảng chứ không neo trên động từ** (điểm dễ làm sai nhất của task này): phần `Up` **được phép** đọc `subscription.campaign_claims` — đó chính là `EXISTS` làm cho trigger ngân sách bất biến có tác dụng (FR-020/FR-021). *Đọc* là tính năng chạy, *sửa* mới là lỗi; một cái grep `campaign_claims` đơn thuần sẽ báo động giả ngay tại dòng đầu tiên.
  - ⚠️ **Hai điều phát hiện được khi làm, CẦN NGƯỜI QUYẾT — không tự sửa.**
    1. `migrationSection(t, name, "Up")` **không** chỉ trả phần `Up`. Nó gọi `betweenMarkers(body, "-- +goose Up", "")` với marker đóng **rỗng**, nên chạy tới **cuối tệp** — tức kèm luôn phần `Down`. Với câu hỏi "đoạn `Down` có xoá gì không" thì vô hại; với câu hỏi của T085 thì không, vì phần `Down` của chính migration lược đồ có `DROP TABLE … subscription.campaigns`. Vì thế tệp mới có `registryUpSection` riêng (bọc `betweenMarkers` với marker đóng `-- +goose Down`) và **không** sửa helper dùng chung.
    2. Hệ quả với T059: `TestIntegrationSeedInsertIsIdempotent` (`campaign_seed_integration_test.go:150`) lấy `Up` bằng helper đó rồi `dbConn.Exec` — theo đọc mã thì nó sẽ chạy **cả** `INSERT … ON CONFLICT DO NOTHING` **và** `DELETE FROM subscription.campaigns WHERE code = 'launch-2026-10'` của phần `Down`, rồi đếm và đòi đúng 1 hàng ⇒ kiểm thử đó **chắc chắn FAIL** khi có CSDL, chứ không chỉ "chưa từng chạy". **Chưa kiểm chứng được** vì lần này cấm mọi lệnh CSDL; cần một lần chạy có `TEST_DATABASE_URL` để xác nhận trước khi sửa.
  - **Bằng chứng phản chứng (đã chạy).** Chèn `ALTER TABLE subscription.campaign_claims ADD CONSTRAINT … FOREIGN KEY (campaign_code) REFERENCES subscription.campaigns (code)` vào `Up` của migration lược đồ ⇒ FAIL với **2** lỗi (khớp `FOREIGN KEY` + khớp `reach` "an ALTER of it"). Chèn `ALTER TABLE subscription.campaign_sweep_runs ADD COLUMN …` vào `Up` của migration seed ⇒ FAIL, và thông báo chỉ đúng tên tệp + tên bảng. Cả hai đã hoàn nguyên; in ra phần `Up` trong thông báo lỗi xác nhận phạm vi **không** còn lẫn phần `Down`.

- [ ] T086 Khẳng định tĩnh hình dạng mà migration lược đồ khai báo — đúng **10** cột FR-010, **5** ràng buộc `chk_campaigns_*`, `ix_campaigns_starts_at`, và `trg_campaigns_immutable_budget` — theo FR-010/FR-011/FR-012/FR-013/FR-021/FR-031 (missing)
  - **Phục vụ**: FR-010 (danh sách cột bắt buộc), và qua tên ràng buộc: FR-011 (`chk_campaigns_code_format`), FR-012 (`chk_campaigns_quota`, `chk_campaigns_reserve`), FR-013 (`chk_campaigns_window`), FR-014 (`chk_campaigns_plan_slug_not_blank`), FR-021 (`trg_campaigns_immutable_budget`), FR-031/FR-032 (cột `closed_at`); quickstart §1.1 liệt kê **chính xác** tên cột, tên ràng buộc, tên chỉ mục và tên trigger này.
  - **Bằng chứng chưa có**: `TestIntegrationRegistryKeepsItsShape` (`campaign_schema_guard_integration_test.go:137`, danh sách 10 cột + 2 index tại `:140`–`:154`) và `TestIntegrationTriggerExistsOnTheCampaignsTable` (`campaign_guard_integration_test.go`, truy vấn `pg_trigger`) đều `t.Skip` không có CSDL. **Không** có khẳng định nào khác về tên trigger hay tên ràng buộc — grep toàn kho cho `chk_campaigns`/`trg_campaigns` chỉ ra các dòng nằm trong `*_integration_test.go` và trong chính phần migration.
  - **Vì sao đáng bổ sung dù không cấp hạ tầng**: `campaign_repo_impl.go:176`–`:177` ghi lý do của riêng nó — danh sách cột `Update` là điểm mấu chốt để `code` không bao giờ bị viết lại và `closed_at` không bao giờ bị xoá (FR-024/FR-034). Nếu một cột bị đổi tên trong migration mà entity Go chưa kịp đổi, GORM **không** báo lỗi biên dịch; nó im lặng bỏ cột khỏi mọi câu `SELECT`/`INSERT`. Đó là loại hỏng chỉ CSDL mới thấy, và nó đang không được ai thấy.
  - **Hướng làm**: phân tích khối `CREATE TABLE` trong section `Up` của migration nhúng và so **tập** tên cột/tên ràng buộc (bằng `assertSameNames` sẵn có ở `campaign_schema_guard_integration_test.go:212`), cộng với khẳng định hiện diện của `CREATE INDEX … ix_campaigns_starts_at` và `CREATE TRIGGER trg_campaigns_immutable_budget BEFORE UPDATE`. **Không** grep từng dòng: `20261003165440_add_campaigns.sql:53`–`:57` căn cột bằng khoảng trắng, nên một so khớp theo dòng sẽ hụt khi ai đó căn lại — hãy tách danh sách cột ra rồi so, đúng như `seedValues` đã làm cho tuple của seed.
  - **Phạm vi**: không thêm cột, không thêm ràng buộc mới. Đây là khẳng định về cái **đang** được khai báo.
  - **[X] Đã làm.** Cùng tệp `campaign_registry_migration_text_test.go`, hai kiểm thử:
    - `TestTheRegistryUpSectionDeclaresExactlyTheRequiredColumnsAndConstraints` — so **tập** đúng **10** cột FR-010 và **5** tên `chk_campaigns_*`. Cột và ràng buộc được **tách ra** qua chuỗi `createTableBody` (cân bằng ngoặc, nên `CHECK (code ~ '…')` và `btrim(plan_slug) <> ''` không cắt nhầm) → `splitTopLevel` (cắt theo dấu phẩy ở **tầng ngoài cùng**, bỏ qua dấu phẩy trong ngoặc và trong literal `'…'`, có xử lý quote kép `''`) → `splitColumnAndConstraintDefinitions` (phân biệt `CONSTRAINT <tên>` với định nghĩa cột), rồi so bằng `assertSameNames`. **Cột thừa cũng FAIL**, không phải pass: so bằng **bằng** chứ không phải so chứa — và đó là cùng lập luận mà `assertColumns` của T027 đã viết.
    - `TestTheRegistryUpSectionDeclaresTheSortIndexAndTheBudgetTrigger` — `CREATE INDEX IF NOT EXISTS ix_campaigns_starts_at ON subscription.campaigns (starts_at DESC)`, `CREATE TRIGGER trg_campaigns_immutable_budget BEFORE UPDATE ON subscription.campaigns`, và `CREATE OR REPLACE FUNCTION subscription.campaigns_guard_immutable_budget()`, khớp trên bản **đã nén khoảng trắng** nên căn lại tệp không làm hụt.
  - **Tái dùng `assertSameNames` đổi một chữ ký**: thêm tham số `requirement` (`"FR-091"` / `"FR-010"` / `"FR-011/FR-012/FR-013/FR-014"`) để thông báo lỗi nói rõ đang kiểm tra yêu cầu nào, vì hai nơi gọi nó hỏi hai câu hỏi khác nhau. Hai lệch gọi trong `campaign_schema_guard_integration_test.go` được cập nhật theo; T027 vẫn xanh.
  - **Không** thêm cột, **không** thêm ràng buộc, **không** sửa hai tệp migration.
  - **Bằng chứng phản chứng (đã chạy).** Đổi `version INTEGER` → `version_no INTEGER` ⇒ FAIL, in ra `got: [… version_no]` / `want: [… version]` — đúng loại hỏng mà `campaign_repo_impl.go:176`–`:177` cảnh báo (GORM âm thầm bỏ cột lạ khỏi mọi `SELECT`/`INSERT`, không lỗi biên dịch). Đổi `chk_campaigns_window` → `chk_campaigns_period` ⇒ FAIL ở khẳng định ràng buộc (kiểm thử sort/trigger vẫn xanh, đúng vì nó không phải chủ đề của nó — kiểm thử tách bạch). Xoá hẳn `CREATE INDEX ix_campaigns_starts_at` ⇒ FAIL ở kiểm thử sort/trigger. Cả ba đã hoàn nguyên.

- [x] T087 Tách khẳng định thứ tự migration (schema **trước** seed) ra khỏi kiểm thử rollback phá huỷ, để nó chạy được trong `go test` thường, theo FR-090/US6.4 (partial)
  - **Phục vụ**: FR-090 ("Migration tạo bảng MUST chạy trước"), US6 kịch bản 4, quickstart V4 bước 5 (rollback **hai lần**), và tiền đề "hai lần" đó.
  - **Bằng chứng chưa chạy được, dù đã viết**: khẳng định thứ tự nằm ở `campaign_seed_integration_test.go:209`–`:214` (`if seedVersion <= schemaVersion`), nhưng nó nằm **sau** hai cổng chặn trong cùng hàm: `requireRollbackConsent(t)` ở `:190` (`:67`–`:74`, skip trừ khi `TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK=true`) và `openRegistryTestDB(t)` ở `:191`. Nghĩa là phép so sánh thứ tự bị chặn bởi **hai** điều kiện mà nó không cần.
  - **Vì sao đây là `partial` chứ không phải `missing`**: phép so sánh **đã tồn tại** và `migrationVersions` (`:377`) đọc `fs.ReadDir(accountmigrations.EmbedFS, ".")` — tức **số học thuần trên tên tệp**, không đụng CSDL, không đụng `DROP TABLE`. Nó chỉ bị chặn nhầm vì nằm chung nhà với phần phá huỷ. Hiện trạng thứ tự **đúng** (`20261003165440_add_campaigns.sql` < `20261003165500_seed_launch_campaign.sql`); điều thiếu là không có gì bắt được một lần đổi tên làm đảo nó — mà đảo thứ tự thì `goose down` sẽ `DROP` bảng **trước** khi `DELETE` hàng seed, tức là rollback âm thầm để lại một hàng không còn bảng nào mô tả.
  - **Hướng làm**: chuyển phép so sánh phiên bản ra một kiểm thử riêng, chỉ đọc embed FS, chạy trong `go test` thường. **Giữ nguyên** `TestIntegrationRegistryRollbackNeedsTwoStepsAndKeepsClaims` như nó là, kèm `requireRollbackConsent` — phần `DROP` là phá huỷ và phải giữ cổng.
  - **Phạm vi**: chỉ di dời một khẳng định. Không sửa hai tệp migration.
  - **[X] Đã làm.** Kiểm thử mới `TestTheSeedMigrationRunsAfterTheSchemaMigration` trong `campaign_seed_integration_test.go`: chỉ gọi `migrationVersions(t)` — thuần số học trên tên tệp trong embed FS — **không** `requireRollbackConsent`, **không** `openRegistryTestDB`, **không** đụng `DROP`. Phép so sánh đã **di dời khỏi** `TestIntegrationRegistryRollbackNeedsTwoStepsAndKeepsClaims`; kiểm thử phá huỷ giữ nguyên `requireRollbackConsent(t)` ở dòng đầu rồi `openRegistryTestDB(t)` ngay sau, không sửa gì thêm. `migrationVersions` vẫn được dùng (nay bởi kiểm thử mới) nên không rơi vào dead code.
  - **Thêm một tiền điều kiện bản cũ không có**: cả hai phiên bản phải khác `0`. Bản cũ dùng `seedVersion <= schemaVersion`, nên khi **chỉ** mất tệp schema (`schema = 0`, `seed > 0`) thì phép so sánh **lọt** và báo "đúng thứ tự" cho một migration không tồn tại. Nay `0` ở bất kỳ bên nào cũng là `t.Fatalf` nói rõ hai lần rollback mà quickstart V4 dựa vào cần **hai** migration.
  - Cố tình **không** đặt tên `TestIntegration…` dù nằm trong một tệp kiểm thử tích hợp: nó không cần CSDL, và tên là tín hiệu nhanh nhất cho người đọc.
  - **Bằng chứng phản chứng (đã chạy).** Đổi tên tệp seed thành `20261002120000_seed_launch_campaign.sql` (tức **thấp hơn** phiên bản lược đồ) ⇒ FAIL, in ra cả hai số phiên bản; đã đổi tên lại về `20261003165500_seed_launch_campaign.sql` và xác nhận PASS.

- [x] T088 Khẳng định tĩnh **SQL thật** của `CampaignRepository.ListOpen` và `ListAll` — vế lọc `closed_at IS NULL` và `ORDER BY starts_at DESC, code DESC` — theo FR-040/FR-041/FR-070 (missing)
  - **Phục vụ**: FR-070 (chỉ xét chiến dịch `closed_at IS NULL`), FR-040 (`tie-break` bằng `code DESC`), FR-041 (kết quả tất định); Câu 6 (thứ tự trả về ổn định).
  - **Bằng chứng chưa có**: hai khẳng định duy nhất về truy vấn thật là `TestIntegrationListOpenExcludesClosedCampaigns` và `TestIntegrationListAllOrdersNewestFirstWithCodeTieBreak` (`campaign_repo_integration_test.go`), cả hai `openRegistryTestDB` ⇒ skip. T084 khoá vế lọc của **fake** (`fakeCampaignRepo.ListOpen`), **không** phải của repository thật.
  - **Mã nguồn tự thừa nhận khoảng trống này** — `campaign_repo_impl.go:63`–`:67`: *"a copy that dropped `code DESC` would keep every behavioural test green while making the result non-deterministic"*. Tức là thứ tự sắp xếp của truy vấn thật **không có** bảo vệ nào ngoài CSDL. Vế lọc còn nặng hơn: mất `closed_at IS NULL` trong `ListOpen` (`:57`–`:60`) là đường cấp gói và quét bù bắt đầu cấp lại cho một chiến dịch đã đóng — đúng thứ FR-031 sinh ra để chặn — trong khi mọi kiểm thử hành vi vẫn xanh vì chúng đi qua fake.
  - **Hướng làm**: bắt SQL thật mà không cần CSDL, theo đúng tiền lệ đã có trong kho: `admin_audit_log_campaign_test.go:124` (`campaignAuditStatements`) chạy repository trên một dialector **không tới được** với `gormlogger.Interface` tự bắt câu lệnh. Ở đây thuận hơn: `listOrdered` chỉ phát **một** câu `Find` (không đếm trước), nên câu bắt được mang trọn cả `WHERE` lẫn `ORDER BY`. Khẳng định `closed_at IS NULL` cho `ListOpen`, `code DESC` cho `ListAll`, và thứ tự không có `starts_at`/`ends_at` cho cả hai.
  - **Phạm vi**: chỉ đọc. Không sửa `campaign_repo_impl.go`, không sửa `ListOpen` — vế `closed_at IS NULL` một mình là quyết định đã đóng băng (A-1/C-9), và điều này chỉ ghi lại nó.
  - **[X] Đã làm.** Tệp mới `campaign_repo_list_query_sql_test.go`, hai kiểm thử: `TestTheListOpenQueryFiltersOnClosedAtInSQL` và `TestTheListAllQueryOrdersNewestFirstWithCodeTieBreak`. Chạy `NewCampaignRepository` trên dialector **không tới được** (`127.0.0.1:1`) với `DisableAutomaticPing` và một `registrySQLCapture` (nhúng `gormlogger.Interface`, `ParamsFilter` giữ placeholder) bắt câu lệnh — đúng tiền lệ `admin_audit_log_campaign_test.go:124`. `registryListStatement` còn khẳng định **đúng một** câu được bắt: nếu sau này có thêm một lần đếm trước `Find` thì các khẳng định này sẽ lặng lẽ bắt đầu đọc **sai** câu, và đó chính xác là lý do `campaignAuditStatements` chỉ bắt được `WHERE` chứ không bắt được `ORDER BY`.
  - **Câu SQL thật mà kiểm thử bắt được** (chạy thật, không phải suy ra): `SELECT * FROM "campaigns" WHERE closed_at IS NULL ORDER BY starts_at DESC, code DESC` cho `ListOpen`, và `SELECT * FROM "campaigns" ORDER BY starts_at DESC, code DESC` cho `ListAll`. **Không có** định ngữ schema trong câu lệnh: `TableName()` trả tên trần và schema đến từ `search_path` của kết nối, nên các khẳng định đòi `FROM "campaigns"` chứ không phải `FROM "subscription"."campaigns"` — đòi định ngữ mà repository không bao giờ viết ra là đòi sai trên một câu lệnh đúng.
  - **Bốn khẳng định phụ đáng giữ, mỗi cái bắt một hỏng khác nhau**: (1) `ORDER BY` phải **đúng** `starts_at DESC, code DESC` — so từng phần sẽ chấp nhận `ORDER BY code DESC, starts_at DESC`, tức thứ tự ngược; (2) `= NULL` bị cấm, vì trong SQL nó là **UNKNOWN** chứ không phải so sánh, nên sẽ lặng lẽ khớp **không** hàng nào và biến mọi chiến dịch thành `campaign_not_configured` thay vì lọc; (3) `ListAll` **không** được mang `closed_at` (FR-032 — danh sách vận hành viên phải vẫn thấy chiến dịch đã đóng); (4) **cửa sổ thời gian không được nằm trong SQL** (FR-042, `spec.md` Câu 6) — vế kiểm thử là `ends_at`, `now()`, `starts_at >`/`<`/`IS`/`BETWEEN`, **không** phải tên cột trần, vì `starts_at DESC` trong `ORDER BY` không phải vế cửa sổ.
  - **Chỉ đọc**: `campaign_repo_impl.go` **không** bị sửa (đã sửa rồi hoàn nguyên trong lúc làm phản chứng; đã đối chiếu lại `closed_at IS NULL` và `starts_at DESC, code DESC` ở đúng dòng cũ). Vế `closed_at IS NULL` được **ghi lại**, không đổi; **không** có kiểm thử nào khẳng định điều ngược lại (xem `deferred.md` D-04).
  - **Bằng chứng phản chứng (đã chạy).** Gỡ `, code DESC` khỏi `Order(...)` ⇒ **cả hai** kiểm thử FAIL ở khẳng định `ORDER BY`. Bỏ hẳn vế `Where("closed_at IS NULL")` ⇒ kiểm thử `ListOpen` FAIL ở FR-070, còn `ListAll` vẫn xanh (đúng: nó không mang vế lọc nào). Thêm `.Where("ends_at IS NULL OR ends_at > now()")` vào `ListOpen` ⇒ FAIL ở cả hai vế `ends_at` và `now()`, đúng tinh thần FR-042. Cả ba đã hoàn nguyên và xác nhận gói xanh.
