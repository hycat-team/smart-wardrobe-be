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

- [ ] T050 [US3] Thêm trường `BudgetLocked bool` (`budgetLocked`) và `Version int` (`version`) vào `CampaignSummaryRes` trong `services/account/dto/subscription/campaign_dto.go`, và tính `budgetLocked = grantedMainCount+grantedCompensationCount > 0` trong `services/account/internal/modules/subscription/application/mapper/campaign_mapper.go` — cùng đợi T051 vì cùng sửa một tệp DTO
- [ ] T051 [US3] Thêm DTO `CampaignUpdateReq` (version, planSlug?, quota?, reserve?, startsAt?, endsAt?) vào `services/account/dto/subscription/campaign_dto.go` — cùng đợi T050 vì cùng sửa một tệp DTO
- [ ] T052 [US3] Cài `Update` trong `registry_uc.go` theo thứ tự: `GetByCode` trong transaction (không có ⇒ `404`) → so từng trường trong thân với giá trị hiện có; **không trường nào đổi** ⇒ trả `200` no-op, **không** `UPDATE`, **không** tăng `version`, **không** ghi audit (FR-055/US3.4) → kiểm bất biến tầng ứng dụng (có lượt cấp + đổi một trong `plan_slug`/`quota`/`reserve` ⇒ `422` với thông báo hướng dẫn đóng và mở mã mới) → `UPDATE ... WHERE version=?` (0 dòng ⇒ `412`); audit `campaign.update` với `payload.before`/`payload.after` — phụ thuộc T033, T050
- [ ] T053 [US3] Thêm handler `Update` vào `services/account/internal/modules/subscription/presentation/handler/campaign_registry_handler.go` và route `PATCH /:code` — phụ thuộc T052
- [ ] T054 [US3] Kiểm thử chống FR-024: `PATCH` với thân chứa `code` không đổi được giá trị nào (mã bất biến) trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_code_immutable_test.go` — phụ thuộc T052
- [ ] T055 [US3] Kiểm thử chống FR-005 và FR-034: `ICampaignRepository` không khai báo `Delete`/`Remove`, và trong `router.go` **không** có route xoá chiến dịch lẫn route mở lại chiến dịch đã đóng — kiểm bằng phản chiếu trên interface + liệt kê route, trong `services/account/internal/modules/subscription/presentation/handler/campaign_no_delete_route_test.go` — phụ thuộc T053
- [ ] T056 [US3] Kiểm thử đơn vị `Update`: chưa cấp sửa được cả ba cột; đã cấp chặn ⇒ `422`; version lệch ⇒ `412`; mã lạ ⇒ `404`; audit có `before`/`after` đúng trường đổi; và **`PATCH` không trường nào đổi ⇒ `200`, `version` giữ nguyên, không có dòng outbox audit thứ hai** (FR-055, US3.4) trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_update_test.go` — phụ thuộc T052
- [ ] T057 [US3] Kiểm thử tích hợp quickstart V3 trên CSDL thật: API trả `422`; `UPDATE subscription.campaigns SET quota = quota + 1 WHERE code = '...'` nâng `check_violation` và giá trị không đổi, trong `services/account/internal/modules/subscription/infrastructure/persistence/campaign_guard_integration_test.go` — phụ thuộc T052, T007
- [ ] T058 [US3] Kiểm thử hợp đồng `PATCH /campaigns/{code}` trả đủ mã `200`/`400`/`404`/`412`/`422`/`403`/`429` trong `services/account/internal/modules/subscription/presentation/handler/campaign_registry_router_test.go` — phụ thuộc T053

**Checkpoint**: Bất biến ngân sách được giữ ở cả tầng ứng dụng lẫn tầng dữ liệu.

---

## Phase 6: User Story 6 — Chuyển `launch-2026-10` sang sổ không đổi nghĩa (Priority: P1)

**Goal**: Seed (T008, đã làm ở Phase 2) được chứng minh trung thực: số liệu tra cứu giống trước; rollback không mất `campaign_claims`.

**Independent Test**: So output 4 API đọc trước/sau migration; chạy migration hai lần; rollback rồi kiểm tra `campaign_claims` còn nguyên.

- [ ] T059 [US6] Kiểm thử tích hợp seed: chạy `make migrate-up svc=account` hai lần không nhân đôi hàng; rollback bằng `make migrate-down svc=account` **hai lần** (`goose down` chỉ lùi **một** phiên bản mỗi lần: một lần cho migration seed, một lần cho migration lược đồ) ⇒ bảng `campaigns` biến mất nhưng `campaign_claims` giữ nguyên số dòng, trong `services/account/internal/modules/subscription/infrastructure/persistence/campaign_seed_integration_test.go` — phụ thuộc T008
- [ ] T060 [US6] Kiểm thử quickstart V4: output 4 API đọc giống hệt trước/sau migration (đếm đã cấp, `watermarkAt`, `status`) trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_migration_fidelity_test.go` — phụ thuộc T059

**Checkpoint**: Cut-over production không đổi nghĩa; rollback an toàn.

---

## Phase 7: User Story 4 — Sửa thời gian linh hoạt (Priority: P2)

**Goal**: `starts_at`/`ends_at` sửa được bất kể đã cấp hay chưa, miễn `ends_at > starts_at`; dời `ends_at` về quá khứ làm chiến dịch ngừng cấp.

**Independent Test**: Sửa `ends_at` về tương lai → cấp lại; về quá khứ → ngừng cấp; `ends_at <= starts_at` → `422`.

- [ ] T061 [US4] Hoàn thiện xử lý trường thời gian trong `registry_uc.go`: chỉ cập nhật trường có mặt trong thân; `endsAt: null` xoá ngày kết thúc; kiểm `ends_at > starts_at` **sau khi áp giá trị mới** (trường vắng mặt giữ giá trị cũ) — phụ thuộc T052
- [ ] T062 [US4] Kiểm thử đơn vị sửa thời gian: dời `ends_at`, xoá `ends_at`, `ends_at <= starts_at` ⇒ `422`, không gửi `starts_at` thì `starts_at` giữ nguyên trong `services/account/internal/modules/subscription/application/usecase/campaign/registry_uc_time_test.go` — phụ thuộc T061
- [ ] T063 [US4] Kiểm thử tích hợp: dời `ends_at` về quá khứ làm `deriveCampaignStatus` trả `expired` và điều kiện 2 trả `campaign_sleeping`; dời về tương lai thì cấp lại, trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_db_time_integration_test.go` — phụ thuộc T061

**Checkpoint**: Thời gian chiến dịch sửa linh hoạt, không phá lượt cấp đã ghi.

---

## Phase 8: User Story 5 — Nhật ký kiểm toán (Priority: P2)

**Goal**: Màn hình đọc nhật ký **ai / lúc nào / đổi gì** cho một chiến dịch.

**Independent Test**: Thực hiện tạo/sửa/đóng, đợi bộ phát hàng đỗi và consumer kiểm toán chạy xong (FR-060), rồi đọc `GET /api/v1/admin/campaigns/{code}/audit` và đối chiếu từng bản ghi.

- [ ] T064 [P] [US5] Khai báo hợp đồng hẹp `ICampaignAuditLookup` (`ListByCampaignCode(ctx, code, page, limit)`) trong `services/account/contract/admin/campaign_audit_contract.go` (không đụng `IAuditLogContract` gRPC)
- [ ] T065 [US5] Thêm `ListByCampaignCode` vào `services/account/internal/modules/admin/domain/repositories/admin_audit_log_repo.go` (lọc `target_type='campaign' AND payload->>'campaignCode' = ?`) và cài trong `services/account/internal/modules/admin/infrastructure/persistence/admin_audit_log_repo.go` — phụ thuộc T064
- [ ] T066 [US5] Cài bộ chuyển đổi `ICampaignAuditLookup` trong module `admin` tại `services/account/internal/modules/admin/provider.go`, trả DTO `dto/admin.AdminAuditLogDTO` — phụ thuộc T065
- [ ] T067 [US5] Thêm handler `GetCampaignAudit` vào `services/account/internal/modules/subscription/presentation/handler/campaign_admin_handler.go` và route `GET /:code/audit` (chỉ đọc, dùng `adminGrantRateLimit()`); handler tra `ICampaignRepository.GetByCode` **trước** và trả `404` bằng `ErrCampaignNotFound` nếu mã không tồn tại, **trước khi** gọi `ICampaignAuditLookup` (module `admin` không biết chiến dịch nào tồn tại và chỉ trả được trang rỗng); ánh xạ sang DTO `CampaignAuditLogRes` khai `targetId` **không** `omitempty` để phát `null` — phụ thuộc T066, T039
- [ ] T068 [US5] Wire hợp đồng hẹp vào `CampaignAdminHandler` (`.../presentation/router/router.go`, `services/account/internal/di/wire_gen.go`, `make wire`) — phụ thuộc T067
- [ ] T069 [US5] Thêm `CampaignAuditLogRes` vào `services/account/dto/subscription/campaign_dto.go` với `TargetID *uuid.UUID` khai `json:"targetId"` **không** `omitempty`, cùng mapper trong `services/account/internal/modules/subscription/application/mapper/campaign_mapper.go` — phụ thuộc T065
- [ ] T070 [US5] Kiểm thử đơn vị `ListByCampaignCode` lọc đúng theo `target_type='campaign' AND payload->>'campaignCode' = ?`, trả `AdminAuditLogDTO` có `TargetID = nil`, và **không** có tác vụ nào trong `admin` tra cứu bảng `subscription.campaigns` — đặt trong `services/account/internal/modules/admin/infrastructure/persistence/admin_audit_log_campaign_test.go` — phụ thuộc T065
- [ ] T071 [US5] Kiểm thử hợp đồng `GET /campaigns/{code}/audit` trả `200`/`403`/`404`/`429` trong `services/account/internal/modules/subscription/presentation/handler/campaign_audit_router_test.go` — phụ thuộc T068
- [ ] T072 [US5] Kiểm thử tích hợp quickstart V7 (kịch bản đã có sẵn ở `spec.md` §11.8 và `quickstart.md`): tạo → sửa → đóng rồi đọc `GET /campaigns/{code}/audit`; kiểm `action` đúng `campaign.create`/`campaign.update`/`campaign.close`, **`targetId` phát ra `null` chứ không bị bỏ**, `payload.campaignCode`, `payload.before`/`after`, `payload.reason`, và `404` cho mã lạ — trong `services/account/internal/modules/subscription/application/usecase/campaign/campaign_db_audit_read_integration_test.go` — phụ thuộc T071

**Checkpoint**: Vòng đời chiến dịch truy vết được ai/lúc nào/đổi gì.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Tài liệu, swagger, và kiểm chứng đầu-cuối

- [ ] T073 Cập nhật Swagger của dịch vụ `account`: thêm annotation cho 4 endpoint mới ở handler, rồi chạy `make swagger svc=account` + `make swagger-all` + `make swagger-combine` (hiến phán: union của các dịch vụ phải giữ mọi mục đã tài liệu) — phụ thuộc T043, T053, T067
- [ ] T074 [P] Viết `specs/026-campaign-registry-db/frontend-guide.md`: 3 endpoint ghi + 1 endpoint đọc nhật ký (method, path, đánh dấu **mới**), trường bổ sung `closedAt`/`budgetLocked`/`version`, `status=closed`, tập `reason` 12 mã không đổi, 6 màn hình admin, và bảng **trước → sau** cho 4 API đọc
- [ ] T075 Đồng bộ `frontend-guide.md` sau khi mã nguồn ổn định: đối chiếu **có đếm** mọi thẻ `json:"…"` mà DTO của 026 phát ra, mọi giá trị của mọi enum guide mô tả (so hai chiều với khối hằng Go), mọi route (dịch `:param` của gin sang `{param}`), mọi mã HTTP guide khẳng định; sửa mọi chỗ lệch — phụ thuộc T074
- [ ] T076 Thêm kiểm thử chống FR-080 và FR-081: không có lớp cache mới cho sổ chiến dịch (không symbol cache nào tham chiếu `Campaign`), mỗi lượt cấp đọc sổ qua repository, và **không** có biến môi trường mới nào cho dữ liệu chiến dịch trong các tệp `.env*` — phụ thuộc T016
- [ ] T077 [P] Ghi `specs/026-campaign-registry-db/deferred.md` cho mọi phát hiện ngoài phạm vi (ví dụ cache TTL tương lai), theo `.specify/templates/deferred-template.md`
- [ ] T078 Chạy các kịch bản còn lại của `quickstart.md` trên CSDL kiểm thử: V5 (hợp đồng đọc không đổi — chỉ thêm `closedAt`/`budgetLocked`/`version`, đúng 12 mã `reason`), V6 (quét bù nhận chiến dịch mới, và `lastSweepAt IS NULL` trước lần quét đầu — FR-072), và nhánh `plan_unavailable` của FR-014/SC-009 — phụ thuộc T075
- [ ] T079 Chạy `make check` đầy đủ một lần lúc chốt (7 bước: fmt-check, tidy-check, generate-check, nginx-check, vet, test-race, build-all)
- [ ] T080 Xác nhận các ràng buộc ngoài phạm vi bằng bằng chứng máy đọc được: `git diff --stat -- services/ai services/commerce` **rỗng** (FR-101); `subscriptioneventtype.All` vẫn đúng **7** giá trị và không có `uow.Execute` lồng nhau trong `registry_uc.go`/`campaign_uc.go` (FR-044); `specs/025-` không bị sửa (FR-103); **không** thay đổi giá/quyền lợi/định nghĩa của bất kỳ gói cước nào (FR-100 — so `subscription_plans` trước/sau); và **không** có endpoint nào dưới `/api/v1` ngoài `/admin/campaigns` mới do 026 thêm, tức không có giao diện cho khách hàng cuối (FR-102)

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
