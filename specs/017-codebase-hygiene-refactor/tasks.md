---
description: "Task list for feature implementation"
---

# Tasks: Chuẩn hóa Codebase — Scan & Refactor theo Checklist Hygiene

**Input**: Design documents from `/specs/017-codebase-hygiene-refactor/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/hygiene-scan-cli.md, quickstart.md

**Tests**: Có test task cho tool scan (`rules_test.go`, `report_test.go`) — đã quy định trong
plan.md. Refactor không thêm test mới, chỉ bảo vệ bằng bộ test hiện có.

**Organization**: Tasks grouped by user story. Thứ tự thực hiện: US1 (tool scan) → US5 (chốt
chuẩn) → US2 (refactor god-file) → US3 (ranh giới module) → US4 (dead code/dup/naming).

**Git compliance (bắt buộc theo AGENTS.md, cập nhật 2026-08-30)**: Agent KHÔNG tự chạy bất kỳ
git mutation nào. Chỉ có MỘT điểm commit duy nhất ở cuối feature (T043) — SAU KHI toàn bộ
tasks.md hoàn tất và `make check` + `make test-race` pass toàn repo — và chỉ thực hiện khi
developer chỉ định. Checkpoint cuối mỗi phase chỉ là điểm dừng để validate, không commit.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5 theo spec.md)
- Include exact file paths in descriptions

## Path Conventions

- Dev tool: `internal/tools/hygiene-scan/` (pattern `swagger-split`)
- Chuẩn chính thức: `docs/development/code-hygiene.md`
- Báo cáo: `specs/017-codebase-hygiene-refactor/reports/`
- Refactor: `internal/modules/**`, `internal/shared/**`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Chuẩn bị vị trí artifact và khung Makefile

- [x] T001 Tạo thư mục `specs/017-codebase-hygiene-refactor/reports/` với file
      `specs/017-codebase-hygiene-refactor/reports/.gitkeep` (báo cáo commit cùng nhánh)
- [x] T002 Thêm target `hygiene-scan` vào `Makefile` (chạy
      `go run ./internal/tools/hygiene-scan`) và mô tả trong mục help tương ứng
- [x] T003 Xác nhận danh sách thư mục loại trừ trong `internal/tools/hygiene-scan/main.go`
      placeholder: `api/swagger`, `bin`, `deployments`, `specs`, `docs`, `database` (đồng bộ
      research.md R8 — chỉ note, chưa implement)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Khung tool scan — mọi rule của US1 phụ thuộc phần này

**⚠️ CRITICAL**: Không bắt đầu US1 trước khi xong phase này

- [x] T004 Tạo `internal/tools/hygiene-scan/model.go`: kiểu `Rule`, `Violation`, `ScanRun`
      đúng schema trong data-model.md §1–§3 (enum severity fail/warning/needs-confirmation)
- [x] T005 Tạo `internal/tools/hygiene-scan/scanner.go`: file walker đọc toàn bộ `.go` trong
      repo, phân loại file production/test/generated (marker `// Code generated`)/excluded
      (R8), trả về danh sách file đã phân loại
- [x] T006 Tạo `internal/tools/hygiene-scan/main.go`: CLI entry với flags `-root`, `-out`,
      `-baseline`, `-format` đúng contract contracts/hygiene-scan-cli.md; exit code 0/2;
      chưa có rule thật (stub rỗng) nhưng phải chạy được end-to-end ghi output
- [x] T007 Chạy `go run ./internal/tools/hygiene-scan` thủ công để xác nhận walker đếm đúng
      tổng số file Go (~645) và loại trừ đúng `api/swagger/**`

**Checkpoint**: Skeleton tool chạy được, in số liệu tổng quan ra stdout

---

## Phase 3: User Story 1 — Scan toàn codebase và xuất báo cáo vi phạm (Priority: P1) 🎯 MVP

**Goal**: `make hygiene-scan` sinh `scan-report.md` + `snapshot-*.json` đầy đủ 8 rule H1–H8,
so sánh baseline được.

**Independent Test**: Chạy `make hygiene-scan` trên codebase hiện tại, đối chiếu ngẫu nhiên 5
vi phạm với code thực (US1 acceptance scenario 3).

### Tests for User Story 1

- [x] T008 [P] [US1] Viết unit test rule H1 file-length với fixture giả lập trong
      `internal/tools/hygiene-scan/rules_test.go` (file >500 → warning, >800 → fail, test file
      nới 750/1200, generated loại trừ)
- [x] T009 [P] [US1] Viết unit test renderer trong `internal/tools/hygiene-scan/report_test.go`
      (JSON snapshot khớp schema data-model.md §3; markdown có đủ 6 section theo contract;
      deterministic — chạy 2 lần cùng input ra cùng nội dung violations)

### Implementation for User Story 1

- [x] T010 [US1] Implement rule H1 (file-length + func-length >80) và H2 (struct-per-file,
      quy tắc "cùng mối quan tâm" = struct cùng phục vụ 1 use case) trong
      `internal/tools/hygiene-scan/rules.go`, dùng `go/parser`/`go/ast` (depends T004, T005)
- [x] T011 [US1] Implement rule H4 boundary/layering: import chéo module bị cấm
      (entities/repositories/infrastructure của module khác), bỏ qua tầng
      (presentation→infrastructure, domain→bất kỳ tầng nào), miễn `provider.go`,
      `internal/di/**`, `internal/bootstrap/**` cho rule bỏ qua tầng (depends T005)
- [x] T012 [P] [US1] Implement rule advisory H5 dead-code (tích hợp output
      `go run golang.org/x/tools/cmd/deadcode` nếu tool có mặt, gắn needs-confirmation),
      H6 duplication gợi ý cụm (file/hàm cùng tên ≥2 module), H7 naming + H8 error-handling
      heuristic (regex/AST, mức warning) — trong `internal/tools/hygiene-scan/rules_advisory.go`
- [x] T013 [US1] Implement renderer trong `internal/tools/hygiene-scan/report.go`: ghi
      `snapshot-<run_id>.json` + `scan-report.md` (6 section theo contract) vào
      `specs/017-codebase-hygiene-refactor/reports/` (depends T004)
- [x] T014 [US1] Implement so sánh baseline trong `internal/tools/hygiene-scan/report.go`:
      flag `-baseline` (mặc định snapshot mới nhất), hiển thị ↑/↓/" theo rule trong section
      Summary (depends T013)
- [x] T015 [US1] Tích hợp tất cả rule vào `main.go` orchestration + Makefile target đã tạo ở
      T002 (depends T010, T011, T012, T013, T014)
- [x] T016 [US1] Chạy lần scan đầu tiên thật: `make hygiene-scan`, rà `scan-report.md` có đủ
      8 rule, H1 fail bắt được `brand_loyalty_handler.go` (~958 dòng), không có vi phạm nào
      trỏ vào `api/swagger/` (quickstart Scenario 1)

**Checkpoint**: 🎯 MVP — US1 độc lập hoàn chỉnh. Điểm dừng validate (không commit): chạy
quickstart Scenario 1 và xác minh báo cáo đầu tiên trước khi chuyển sang US5.

---

## Phase 4: User Story 5 — Chốt checklist và ngưỡng thành chuẩn chính thức (Priority: P1)

**Goal**: `docs/development/code-hygiene.md` là nguồn sự thật duy nhất cho hygiene; có tham
chiếu từ design-pattern.md và AGENTS.md.

**Independent Test**: Đọc doc — mỗi mục checklist có ngưỡng định lượng; mọi RuleID trong báo cáo
scan quy chiếu được về điều luật (US5 acceptance scenario 1–2).

### Implementation for User Story 5

- [x] T017 [US5] Viết `docs/development/code-hygiene.md` theo cấu trúc bắt buộc
      data-model.md §4: mục đích/phạm vi, 8 mục H1–H8 với bảng ngưỡng đã chốt
      (500/800, func 80, test 750/1200), quy tắc severity, hướng dẫn `make hygiene-scan`,
      ví dụ trước/sau cho god-file và import vi phạm, tham chiếu constitution §II/§IV và
      design-pattern.md
- [x] T018 [P] [US5] Thêm tham chiếu tới `docs/development/code-hygiene.md` trong
      `docs/system-design/design-pattern.md` (mục coding conventions) và trong `AGENTS.md`
      (mục Code Compliance) — KHÔNG định nghĩa lại quy ước đã có ở design-pattern.md
- [x] T019 [US5] Kiểm tra chéo: mở `scan-report.md`, đối chiếu từng RuleID xuất hiện có điều
      luật tương ứng trong code-hygiene.md (SC-005)

**Checkpoint**: Chuẩn đã chốt. Điểm dừng validate (không commit).

---

## Phase 5: User Story 2 — Refactor vi phạm nghiêm trọng về kích thước file (Priority: P2)

**Goal**: 0 file production >800 dòng; các file >500 dòng được tách theo responsibility, giữ
nguyên hành vi.

**Independent Test**: `metrics.files_over_800 == 0` trong snapshot sau refactor; `make check &&
make test-race` pass (US2 independent test).

**Quy tắc refactor (research.md R7)**: chỉ di chuyển/tổ chức lại code; giữ cùng package nếu có
thể; mapper giữ trong `application/mapper`, helper đuôi `_helper.go` (constitution Code Style);
không sửa logic; mapper god-file tách theo use case nhóm (VD loyalty split theo handler method).

- [x] T020 [US2] Refactor `internal/modules/brand/presentation/handler/brand_loyalty_handler.go`
      (958 dòng, fail): tách theo nhóm use case loyalty thành các file handler cùng package
      (VD `brand_loyalty_handler.go` + `brand_loyalty_rewards_handler.go`...), chạy lại scan
      xác nhận hết fail (depends T016)
- [x] T021 [US2] Refactor `internal/modules/brand/application/usecase/loyalty/usecase.go`
      (705 dòng): tách theo nhóm hành vi (earn/redeem/tier...) thành các file usecase cùng
      package, giữ interface public không đổi (depends T016)
- [x] T022 [P] [US2] Refactor `internal/modules/fashion/application/usecase/worker/worker_uc.go`
      (621 dòng): tách theo nhóm hành vi worker (depends T016)
- [x] T023 [P] [US2] Refactor `internal/modules/brand/presentation/handler/brand_portal_handler.go`
      (618 dòng): tách theo nhóm use case portal (depends T016)
- [x] T024 [P] [US2] Refactor `internal/modules/brand/application/usecase/brand/usecase.go`
      (608 dòng) và `internal/modules/brand/application/usecase/item/usecase.go` (596 dòng):
      tách theo nhóm hành vi (depends T016)
- [x] T025 [P] [US2] Refactor `internal/shared/infrastructure/ai/google/google.go` (601 dòng)
      và `internal/shared/infrastructure/ai/core.go` (512 dòng): tách theo nhóm hành vi
      (client/core/fallback), không đổi contract công khai của package (depends T016)
- [x] T026 [P] [US2] Refactor `internal/modules/identity/application/usecase/user_uc.go`
      (595 dòng) và `internal/modules/dashboard/application/usecase/dynamic_query_uc_impl.go`
      (553 dòng): tách theo nhóm hành vi (depends T016)
- [x] T027 [US2] Xử lý các file warning 500–800 dòng còn lại trong báo cáo scan theo cùng quy
      tắc (danh sách lấy từ snapshot T016, có thể chấp nhận-with-reason nếu tách gây hại rõ ràng
      — ghi lý do vào báo cáo) (depends T020–T026)
- [x] T028 [US2] Chạy lại `make hygiene-scan` với baseline: xác nhận
      `metrics.files_over_800 == 0` (SC-001), số warning giảm đúng; `make check &&
      make test-race` pass (FR-005)

**Checkpoint**: US2 xong. Điểm dừng validate (không commit): SC-001 đạt, test pass.

---

## Phase 6: User Story 3 — Kiểm tra và sửa vi phạm ranh giới module & phân lớp (Priority: P2)

**Goal**: `cross_module_prohibited_imports == 0` và `layer_skips == 0` (SC-002), sửa đúng theo
constitution §II/§IV, không exception ad-hoc.

**Independent Test**: Danh sách import trong snapshot sau refactor: 0 import thuộc nhóm cấm
(US3 independent test).

- [x] T029 [US3] Rút danh sách vi phạm H4 từ snapshot mới nhất (`scan-report.md` section
      Violations, rule H4); nếu danh sách rỗng thì ghi nhận và bỏ qua T030–T032 (depends T016)
- [x] T030 [US3] Sửa từng cụm vi phạm import chéo module (A import entities/repositories/
      infrastructure của B) theo Pure Contract Usecase Principle (constitution §II): chuyển qua
      `presentation/contract/` + DTO, caller tự enrich bằng `sliceutils.IndexBy`/`GroupBy` theo
      `docs/system-design/contract-purity-pattern.md` — mỗi module một cụm thay đổi
      (depends T029)
- [x] T031 [US3] Sửa từng cụm vi phạm bỏ qua tầng (presentation gọi thẳng infrastructure):
      chuyển qua use case interface của cùng module; giữ handler mỏng theo §IV (depends T029)
- [x] T032 [US3] Chạy lại `make hygiene-scan`: xác nhận `layer_skips == 0` &&
      `cross_module_prohibited_imports == 0` (SC-002); `make check && make test-race` pass
      (depends T030, T031)

**Checkpoint**: US3 xong. Điểm dừng validate (không commit): SC-002 đạt, test pass.

---

## Phase 7: User Story 4 — Dọn dead code, trùng lặp và thống nhất naming (Priority: P3)

**Goal**: Dead code xóa có bằng chứng; trùng lặp gộp/giữ có lý do; naming + error handling
thống nhất theo design-pattern.md.

**Independent Test**: Mỗi lần xóa dead code kèm bằng chứng không tham chiếu; test vẫn pass
(US4 independent test).

- [x] T033 [US4] Rà section `## Needs Confirmation` của báo cáo: xác nhận tay từng dead code
      (grep tham chiếu toàn repo, kiểm tra interface/reflect); xóa phần chắc chắn, phần không
      chắc giữ nguyên với `accepted-with-reason` trong báo cáo (FR-008) (depends T016)
- [x] T034 [P] [US4] Audit cụm trùng lặp H6 (file/hàm cùng tên ≥2 module): gộp về vị trí dùng
      chung nếu KHÔNG vi phạm ranh giới module; nếu gộp vi phạm ranh giới thì ghi nhận
      `accepted-with-reason` (CHK013) (depends T016)
- [x] T035 [P] [US4] Sửa vi phạm naming H7: đổi tên file khớp struct chính, thống nhất suffix
      (`_helper.go`...), method query `Find...` → `Get...` theo design-pattern.md — chú ý không
      đổi hành vi, chỉ đổi tên (depends T016)
- [x] T036 [P] [US4] Sửa vi phạm error handling H8: chuyển error trả về thô sang `apperror`
      factory, wrap `%w` đúng chỗ, theo docs quy ước hiện hành — mức warning, ưu tiên các file
      đã đụng đến trong T020–T031 (depends T016)
- [x] T037 [US4] Chạy lại `make hygiene-scan` + `make check && make test-race`: các mục
      H5/H6/H7/H8 xử lý xong hoặc có lý do chấp nhận; không phát sinh vi phạm fail mới
      (depends T033, T034, T035, T036)

**Checkpoint**: US4 xong. Điểm dừng validate (không commit).

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Xác minh end-to-end theo quickstart và chốt báo cáo cuối

- [x] T038 [P] Rà soát `docs/development/code-hygiene.md` lần cuối: khớp 100% với rule engine
      đã implement (rule nào trong tool phải có điều luật trong doc và ngược lại)
- [x] T039 Chạy full quickstart.md Scenario 1–5 từ đầu đến cuối, ghi kết quả từng scenario
- [x] T040 Thực hiện audit SC-003: chọn 20 vi phạm ngẫu nhiên từ báo cáo cuối, đối chiếu code,
      ghi tỉ lệ đúng vào cuối `scan-report.md` (≥90% = pass)
- [x] T041 Kiểm chứng SC-006: giải thích vị trí ≥5 chức năng đại diện (brand loyalty, fashion
      worker, AI pipeline, identity user, dashboard query) chỉ qua tên folder/file — ghi kết
      quả vào báo cáo
- [x] T042 Chạy lần scan cuối cùng: xác nhận Definition of Done trong quickstart.md
      (SC-001, SC-002, SC-004, SC-005) và snapshot/report cuối cùng commit vào `reports/`
- [x] T043 Điểm commit duy nhất của feature (SAU KHI TOÀN BỘ tasks.md hoàn tất): chạy
      `make wire && make lint && make test-race` toàn repo, tổng hợp kết quả + số liệu
      SC-001…SC-005 từ báo cáo cuối và TRÌNH developer quyết định commit/PR (agent không tự
      thực hiện git mutation — theo AGENTS.md)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (P1) → Foundational (P2)**: tuần tự, blocks tất cả user story
- **US1 (P3) → US5 (P4)**: US5 cần báo cáo thật (T016) để đối chiếu RuleID (T019)
- **US2 (P5) → US3 (P6)**: refactor tách file có thể làm xuất hiện/di chuyển import —
  sửa ranh giới sau khi tách xong để tránh làm hai lần
- **US4 (P7)**: sau US2/US3 vì dead code/dup/naming cần nhìn codebase đã ổn định
- **Polish (P8)**: sau tất cả

### User Story Dependencies

- US1 độc lập tuyệt đối (MVP) — chỉ phụ thuộc Setup + Foundational
- US5 phụ thuộc đầu ra scan của US1 (mức thấp: chỉ cần danh sách RuleID)
- US2/US3/US4 đều phụ thuộc báo cáo US1 làm input; không phụ thuộc lẫn nhau về code nhưng
  khuyến nghị tuần tự theo lý do ở trên

### Parallel Opportunities

- Trong US1: T008+T009 (tests) song song; T012 song song với T010/T011
- Trong US2: T022–T026 chạm module khác nhau → song song được (T020/T021 cùng package brand
  nên tuần tự)
- Trong US4: T034/T035/T036 song song

---

## Implementation Strategy

### MVP First

1. Phase 1–2: Setup + Foundational
2. Phase 3 (US1): tool scan chạy thật, xuất báo cáo đầu tiên → **STOP, validate theo
   quickstart Scenario 1** — đây là giá trị tức thời: bức tranh vi phạm toàn codebase
3. Phase 4 (US5): chốt chuẩn — từ đây mọi review/scan đối chiếu một nguồn duy nhất

### Incremental Delivery

4. Phase 5 (US2): god-file về ngưỡng — validate, không commit
5. Phase 6 (US3): ranh giới sạch — validate, không commit
6. Phase 7 (US4): dead code/dup/naming — validate, không commit
7. Phase 8 (Polish): audit SC-003, SC-006, DoD — sau đó T043 là điểm commit duy nhất của
   feature, chỉ thực hiện khi developer chỉ định (yêu cầu cập nhật 2026-08-30)

---

## Notes

- Refactor = behavior-preserving: chỉ di chuyển/tổ chức code; không sửa logic, không đổi API
- Mỗi task refactor kết thúc bằng `make check && make test-race` trước khi coi là xong
- Commit: CHỈ MỘT LẦN sau khi toàn bộ tasks.md hoàn tất (T043) — checkpoint giữa phase là điểm
  validate, không commit (FR-006 bản cập nhật 2026-08-30)
- [P] tasks = different files, no dependencies
- Stop at any checkpoint để validate độc lập từng story

---

## Phase 9: Convergence — xử lý dứt điểm H5 dead-code (32 candidate) & H8 error handling (41 cảnh báo + leak)

> Nguồn: `/speckit-converge` chạy 2026-08-30 sau khi T001–T043 hoàn tất và commit 27c69ab.
> Truy vết: `reports/h5-verification.json` (32 mục kind `referenced`), `reports/snapshot-20260830-180210.json`
> (41 violation H8), đánh giá leak client tại `apperror/error.go:105-130` + `errorutils/errorutils.go:208`.
> Quy tắc: task H5 mỗi symbol phải kết thúc bằng MỘT trong hai kết luận — **xóa** (kèm grep chứng minh
> không caller thật) hoặc **keep-reason** (ghi 1 dòng lý do vào `docs/development/code-hygiene.md`
> mục "Advisory backlog"). Task H8 KHÔNG đổi message/status trả về client (FR-005), chỉ sửa chuỗi
> error nội bộ. Sau mỗi task: `make wire && make lint && make test-race`.

### Chặn rò rỉ error ra client (ưu tiên cao nhất — làm TRƯỚC các task %w)

- [x] T044 [H8/leak] Chặn rò rỉ message nội bộ ở biên mapper: sửa `apperror.From` và
  `apperror.Wrap` (internal/shared/application/constants/apperror/error.go) để error không nhận
  diện được trả Message generic từ catalog cố định (không dùng `err.Error()` làm Message), giữ
  `Cause` nguyên vẹn cho log; sửa `errorutils.MapErrorToProblem` (errorutils.go:208) tương tự.
  Bổ sung chế độ lộ chi tiết chỉ bật khi `appEnv == "development"` (mô phỏng cách StackTrace đang
  làm ở middleware/error_handler.go:86) per FR-005 (partial)
- [x] T045 [H8/leak] Cập nhật test theo hành vi mới của T044: `middleware/error_handler_test.go`,
  `errorutils/errorutils_test.go`, `apperror/error_test.go` — client nhận message generic, log vẫn
  chứa chi tiết `rawErr`/cause; thêm test case error GORM/RabbitMQ thô không lộ text nội bộ per
  FR-005 (partial)

### H8: chuyển `fmt.Errorf` thiếu `%w` sang wrap chuẩn (MEDIUM — làm sau T044)

- [x] T046 [P] [H8] `internal/modules/aiexecutor/application/usecase/orchestrator/plan.go:21` và
  `internal/modules/aiexecutor/application/usecase/pricing/usecase.go:43`: chuyển `fmt.Errorf`
  mất chain sang `%w` (hoặc `apperror.Wrap` nếu error chảy lên handler) per US4/T036 (partial)
- [x] T047 [P] [H8] `internal/modules/fashion/application/usecase/ai/recommendation/retrieval/llm_rewriter.go`
  — 6 vị trí: dòng 83, 173, 217, 220, 231, 246: chuyển sang `%w` per US4/T036 (partial)
- [x] T048 [P] [H8] `internal/modules/fashion/application/usecase/ai/recommendation/synthesis/` —
  7 vị trí: `generation.go` dòng 69, 90; `json.go` dòng 38, 52, 57, 61; `prompt.go` dòng 54:
  chuyển sang `%w` per US4/T036 (partial)
- [x] T049 [P] [H8] `internal/modules/fashion/application/usecase/worker/analyze_uc.go:187` và
  `worker/shared/fashion_helper.go:32`: chuyển sang `%w` per US4/T036 (partial)
- [x] T050 [P] [H8] `internal/modules/notification/application/usecase/email/outbox_email_uc.go`
  — 3 vị trí: dòng 53, 58, 94: chuyển sang `%w` per US4/T036 (partial)
- [x] T051 [P] [H8] `internal/modules/subscription/application/usecase/subscription/state_machine.go`
  — 6 vị trí: dòng 67, 73, 78, 85, 90, 180: chuyển sang `%w` per US4/T036 (partial)
- [x] T052 [P] [H8] `internal/modules/subscription/application/usecase/subscription/subscription_renewal_workflow.go`
  — 6 vị trí: dòng 134, 267, 435, 440, 443, 449: chuyển sang `%w` per US4/T036 (partial)
- [x] T053 [P] [H8] `internal/modules/wardrobe/application/usecase/wardrobe/search_sync/search_sync_uc.go:57`:
  chuyển sang `%w` per US4/T036 (partial)

### H8: audit sentinel `errors.New` bare (LOW)

- [x] T054 [H8] Audit 8 vị trí `errors.New` bare: `aiexecutor/orchestrator/orchestrator.go` dòng
  93, 151, 221; `fashion/.../ai/chat/summary.go` dòng 30, 64; `fashion/.../llm_rewriter.go` dòng
  71, 147, 154 — từng cái: nếu là sentinel error dùng lặp lại thì chuyển về package errors chuẩn
  của module; nếu là lỗi one-off thì giữ và ghi keep-reason. Cập nhật
  `docs/development/code-hygiene.md`: sentinel `errors.New` là acceptable + thêm loại trừ tương
  ứng vào heuristic H8 trong `internal/tools/hygiene-scan/rules_advisory.go` per US4/T036 (partial)

### H5: verify 32 dead-code candidate `referenced` (xóa hoặc ghi keep-reason từng symbol)

- [x] T055 [P] [H5] `internal/modules/admin/application/errors/errors.go`: verify 4 sentinel —
  `ErrUserNotFound` (fileCount=12), `ErrBrandMemberNotFound`, `ErrCannotRemoveLastOwner`,
  `ErrCannotDemoteLastOwner` (fileCount=3): xác định caller thật qua interface/alias; xóa nếu
  chết, ngược lại ghi keep-reason per US4/T033 (partial)
- [x] T056 [P] [H5] `internal/modules/brand/application/mapper/brand.go`: verify
  `MapBrandMembers` (fileCount=3), `MapBrandCustomers` (fileCount=2) per US4/T033 (partial)
- [x] T057 [P] [H5] `internal/modules/brand/application/usecase/benefit/helper.go`: verify
  `parseValidDurationDays` (fileCount=5) per US4/T033 (partial)
- [x] T058 [P] [H5] `internal/modules/fashion/application/errors/chat_errors.go`: verify
  `ErrInvalidChatIDFormat` (fileCount=2) — đối chiếu với bản trùng tên ở
  `internal/modules/wardrobe/application/errors/errors.go` per US4/T033 (partial)
- [x] T059 [P] [H5] `internal/modules/fashion/application/usecase/ai/chat/helper.go`: verify
  `buildChatSystemPrompt`, `isWardrobeRelatedQuery` (fileCount=2) per US4/T033 (partial)
- [x] T060 [P] [H5] `internal/modules/fashion/application/usecase/ai/recommendation/synthesis/prompt.go`:
  verify `BuildRecommendationPrompt` (fileCount=3) per US4/T033 (partial)
- [x] T061 [P] [H5] `internal/modules/fashion/application/usecase/ai/recommendation/types/structs.go`:
  verify method `Total` (fileCount=98 — nghi là trùng tên phổ biến, cần phân tích chính xác) per
  US4/T033 (partial)
- [x] T062 [P] [H5] `internal/modules/fashion/infrastructure/persistence/fashion_item_repo_hybrid_helpers.go`:
  verify `buildRecommendationSeasonalityCondition` (fileCount=3),
  `recommendationSearchDocumentSQL` (fileCount=2) per US4/T033 (partial)
- [x] T063 [P] [H5] `internal/modules/subscription/application/usecase/subscription/state_machine.go`:
  verify 4 symbol — `RestoreLifetimeFallbackIfExpired`, `expiryFor`, `AddProductDays`,
  `clearFallback` (fileCount=2) per US4/T033 (partial)
- [x] T064 [P] [H5] `internal/modules/wardrobe/application/errors/errors.go`: verify 3 sentinel —
  `ErrInvalidChatIDFormat`, `ErrMinimumWardrobeItemsRequired`, `ErrChatNotFound` per US4/T033 (partial)
- [x] T065 [P] [H5] `internal/modules/wardrobe/application/usecase/wardrobe/shared/`: verify
  `lifecycle_decay.go: CalculateLifecycleDecayFactor` (fileCount=2) và
  `string_helper.go: RemoveVietnameseSigns` (fileCount=4) per US4/T033 (partial)
- [x] T066 [P] [H5] `internal/shared/domain/entities/`: verify method `Scan` trong `jsonb.go` và
  `vector.go` (fileCount=26 — dự kiến keep vì thỏa interface `sql.Scanner` của GORM) per US4/T033 (partial)
- [x] T067 [P] [H5] `internal/shared/infrastructure/`: verify 6 symbol —
  `ai/helper.go: parseSSEStream` (fileCount=4), `httptransport/transport.go: DefaultClient`
  (fileCount=2), `repositories/pagination.go: Offset` (fileCount=10) + `BuildPaginationMetadata`
  (fileCount=19), `middleware/ratelimit.go: Stop` (fileCount=33 — nghi thỏa interface),
  `worker/base_worker.go: Name` (fileCount=156 — nghi thỏa interface worker) per US4/T033 (partial)
- [x] T068 [H5] `internal/pkg/utils/errorutils/errorutils.go: MapErrorToProblem` (fileCount=2):
  quyết định chung với T044 — nếu T044 giữ hàm (đã fix leak) thì kiểm tra caller thật và xác nhận
  keep-reason; nếu không còn caller nào sau T044 thì xóa per US4/T033 (partial)

### Kiểm chứng chốt Phase 9

- [x] T069 [H5/H8] Kiểm chứng chốt: chạy `make hygiene-scan` — mọi H5/H8 còn lại trong báo cáo
  đều có keep-reason tương ứng trong `docs/development/code-hygiene.md` (mục Advisory backlog);
  chạy `make wire && make lint && make test-race` pass toàn bộ; cập nhật snapshot mới +
  `reports/scan-report.md` (section Final Audit bổ sung Phase 9) per SC-004, SC-005 (partial)

### Mở rộng Phase 9: map lỗi bên thứ 3 tại biên integration (kết quả audit 2026-08-30)

> Nguồn: audit toàn bộ 7 integration (OpenAI, Gemini, Cloudinary, SMTP, Elasticsearch, Redis,
> RabbitMQ) theo yêu cầu developer — mapper central (T044) chỉ là lưới an toàn cuối; các kênh
> bypass mapper phải sửa tại nguồn. Các task này thuộc cụm commit (1) leak fix + test theo
> chính sách 3 commit của Phase 9.

- [x] T070 [P] [leak-HIGH] AI clients map tường minh tại biên: `internal/shared/infrastructure/ai/openai/openai.go`
  (dòng 77, 183, 268, 328 — error đang nhúng toàn bộ response body của provider) và
  `internal/shared/infrastructure/ai/google/google_text.go` (dòng 77, 192) +
  `google_vision_embed.go` (dòng 68, 147): chuyển HTTP-status error thành `apperror` static
  (502/503, message catalog "Dịch vụ AI đang bận, vui lòng thử lại sau"), response body chỉ log
  qua zap, không nằm trong text error; giữ sentinel safe-path hiện có per FR-005 (partial)
- [x] T071 [leak-HIGH] SSE chat stream không gửi `err.Error()`: `internal/modules/fashion/presentation/handler/fashion_ai_handler.go:319-322`
  — event `error` trong SSE phải chứa message generic an toàn (chi tiết lỗi chỉ vào log);
  kiểm tra cả nhánh partial-content rồi fail per FR-005 (partial)
- [x] T072 [leak-HIGH] Đứt kênh payload.Error từ worker ra SSE: `internal/modules/fashion/application/usecase/worker/analyze_uc.go:113`
  đưa `err.Error()` (raw "Google Vision API error...") vào event payload →
  `internal/modules/wardrobe/application/usecase/wardrobe/fashion_event_uc.go:118,174` phát qua
  SSE `/api/v1/wardrobe-items/tasks/{taskId}/sse` — thay bằng reason code / message an toàn từ
  catalog, chi tiết chỉ log; cập nhật SSEEventPayload nếu cần per FR-005 (partial)
- [x] T073 [leak-MED] Elasticsearch client không nhúng body vào Message:
  `internal/shared/infrastructure/search/elasticsearch.go:96` đang `apperror.NewError(status,
  "Hệ thống gặp sự cố", string(respBody))` — Message đổi thành static, body chỉ log; đối chiếu
  nhánh circuit breaker (dòng 204) và 404 per FR-005 (partial)
- [x] T074 [leak-MED] Redis OTP không lộ topology mạng: `internal/modules/identity/infrastructure/caching/redis_otp_service.go`
  (GenerateOtp/VerifyOtp/IsInResendCooldown trả raw redis error) — map tại biên register_uc /
  password_recovery_uc thành `apperror` 503 "tạm thời không khả dụng", log cause đầy đủ per
  FR-005 (partial)
- [x] T075 [leak-LOW] SMTP mapping điểm client-facing log cause trước khi map static:
  `internal/modules/identity/application/usecase/register_uc.go:119-125,198-204` và
  `password_recovery_uc.go:92-97` — thêm log chi tiết SMTP error (hiện bị mất) trước khi trả
  static apperror per FR-005 (partial)
- [x] T076 [leak] Kiểm chứng leak-fix: test `GlobalErrorHandler` + SSE handler + task SSE với
  error giả lập từ GORM/Redis/AI provider — client không bao giờ thấy text chứa SQL, constraint,
  dial tcp, response body provider; dev môi trường thấy qua `debugMessage` per FR-005 (missing)

---

## Phase 10: Convergence (lần 2) — đồng bộ tài liệu contract với hành vi FR-003/FR-011

> Nguồn: `/speckit-converge` chạy 2026-08-30 sau khi Phase 9 (T044–T076) hoàn tất. Tool đã
> implement keep-list + heuristic mới nhưng 2 tài liệu design chưa phản ánh — cập nhật doc thuần,
> không đụng code.

- [x] T077 [FR-011/FR-003] Cập nhật `specs/017-codebase-hygiene-refactor/contracts/hygiene-scan-cli.md`:
  (1) mục Output contract bổ sung keep-list `internal/tools/hygiene-scan/accepted.json` và ý nghĩa
  status `accepted-with-reason` (loại khỏi tổng đếm, vẫn in trong bảng Violations kèm reason);
  (2) Behavioral guarantees bổ sung dòng console `accepted-keep: N`; (3) ghi exemption H8:
  sentinel `var ErrXxx = errors.New(...)` package-level và `fmt.Errorf` message-only không bị
  flag per FR-003 (partial)
- [x] T078 [FR-011/FR-003] Cập nhật `specs/017-codebase-hygiene-refactor/data-model.md`: thêm
  field `acceptedKeeps` vào schema ScanRun §3; cập nhật mô tả rule H8 ở bảng rule để phản ánh
  sentinel/message-only exemption per FR-003 (partial)
