# Tasks: Bàn giao quyền sở hữu DDL (030)

**Input**: Design documents from `/specs/030-ddl-ownership-handover/`

**Prerequisites**: plan.md, spec.md, research.md (R1–R6), data-model.md, contracts/ownership-contract.md, quickstart.md

**Tests**: Không có test TDD riêng; mọi kiểm chứng là verification tasks chạy trên database tạm `verify_d06_*` theo quickstart.md (V1–V4). **Cấm `down -v`, cấm xóa volume, cấm target production.**

**Organization**: Tasks grouped by user story (US1 P1 → US2 P1 → US3 P2 → US4 P2).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Chốt điểm cắt, lệnh ngừng bắn, baseline đo đạc trước khi đụng bất cứ file migration nào

- [X] T001 Record cutover point: query `max(version_id)` + count của `goose_db_version` / `goose_db_version_commerce` trên mọi môi trường (dev, staging, prod read-only) và ghi vào `specs/030-ddl-ownership-handover/verification-log.md`
- [X] T002 [P] Read `docs/system-design/design-pattern.md` bản mới nhất (hiến pháp §VII bắt buộc trước khi viết/sửa code)
- [X] T003 [P] Snapshot schema-only của DB dev (`pg_dump --schema-only`) vào `specs/030-ddl-ownership-handover/baseline-old.sql` làm mốc diff hội tụ (chỉ đọc, không đổi DB)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Xác minh thực nghiệm R1 + wiring AI — MỌI user story đều chặn ở đây

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Verify R1 trên database tạm: sửa 1 dòng comment trong `services/account/database/migrations/20261011090001_add_user_activity_events.sql`, boot Account vào `verify_d06_exp`, assert migrate xanh (chứng minh edit-in-place vô hình với DB đã applied), rồi revert file và ghi kết quả vào `specs/030-ddl-ownership-handover/verification-log.md`
- [X] T005 Wire migration tree cho AI: tạo `services/ai/database/migrations/migrations.go` (embed `*.sql`, copy mẫu `services/account/database/migrations/migrations.go`), sửa `provideMigrationsFS` trong `services/ai/internal/di/providers.go` sang return `EmbedFS`, regen `services/ai/internal/di/wire_gen.go` via `make wire`, verify `services/ai/Dockerfile` đã copy `/migrations` (dòng 32, không sửa)
- [X] T006 [P] Verify `make build-all` xanh cho cả 3 service sau wiring (chưa có migration mới nên boot phải no-op) — build AI từng đỏ transient vì embed thiếu `.sql`, đã xanh sau T014

**Checkpoint**: Foundation ready — R1 đã chứng minh bằng thực nghiệm, AI đã có cây migration rỗng wired sẵn

---

## Phase 3: User Story 1 - Single writer cho `aiexecutor`/`dashboard` (Priority: P1) 🎯 MVP

**Goal**: Cây Account sạch DDL ngoài chủ; baseline AI/Commerce tự đứng được; boot xanh trên DB trắng và DB cũ

**Independent Test**: Replay trắng `verify_d06_fresh` (account→ai→commerce) ra đủ 7+3 bảng + tên constraint mới; boot bản mới trên DB cũ hội tụ no-op sau lần đầu; `goose_db_version` vẫn 35 dòng

- [X] T007 [P] [US1] Rewrite `services/account/database/migrations/20260622131631_add_ai_cost_policy.sql`: gỡ tạo 5 bảng `aiexecutor` + CHECK nội tuyến, giữ tên file + version + Down tương ứng
- [X] T008 [P] [US1] Rewrite `services/account/database/migrations/20260704062159_init_ai_executor_module.sql`: gỡ tạo `ai_model_configs` + rename bảng, giữ tên file + version + Down
- [X] T009 [P] [US1] Rewrite `services/account/database/migrations/20260723000001_create_dashboard_snapshots_tables.sql`: gỡ tạo 3 bảng `dashboard`, giữ tên file + version + Down
- [X] T010 [P] [US1] Rewrite `services/account/database/migrations/20260801000000_separate_module_schemas.sql` (T037 follow-up xong: gỡ nốt dòng FK ai/dashboard Up+Down, giữ còn lại)
- [X] T011 [P] [US1] Rewrite `services/account/database/migrations/20260930120000_admin_premium_grant_schema.sql`: gỡ 4 DDL `aiexecutor` (index grants, ledgers nullable + partial unique, operations nullable, tạo audit log), giữ tên file + version + Down
- [X] T012 [P] [US1] Rewrite `services/account/database/migrations/20260930150000_free_default_outfit_attempt_cap.sql`: gỡ `UPDATE` data `aiexecutor`, giữ tên file + version + Down
- [X] T013 [P] [US1] Rewrite `services/account/database/migrations/20261011090002_add_payos_status_counts.sql`: gỡ `ALTER dashboard.*`, giữ tên file + version + Down
- [X] T014 [US1] Create AI baseline chain trong `services/ai/database/migrations/20261016213000_adopt_aiexecutor_ownership.sql` (timestamp thật, m=1): định nghĩa đầy đủ 7 bảng `aiexecutor` ở trạng thái cuối (tên constraint tường minh mới, index, nullable đúng), hội tụ no-op trên DB cũ nhờ `IF NOT EXISTS` + seed guard (không precondition fail-fast vì baseline CREATE từ trắng), `Down` hoàn nguyên (depends on T005, T007–T013)
- [X] T015 [US1] Create Commerce dashboard baseline trong `services/commerce/database/migrations/20261016213001_adopt_dashboard_ownership.sql` (k=1): tạo 3 bảng đã gồm 3 cột payos, idempotent, `Down` đầy đủ (depends on T009, T010, T013)
- [X] T016 [US1] Replay trắng quickstart V1 vào `verify_d06_fresh`, assert: đủ 7+3 bảng, `\d` chỉ còn `ai_policy_grants_effective_window_check`, version counts = 35 / 3+k / m, rồi `DROP DATABASE verify_d06_fresh`; sau đó quickstart V1b: boot đồng thời 3 binary mới vào `verify_d06_race` trống, assert lock serialize (không lỗi bảng đã tồn tại, không deadlock) + schema đồng nhất V1, rồi dọn DB tạm (depends on T014, T015, T033–T039)
- [X] T017 [US1] Boot bản mới trên bản sao DB cũ ở mức goose CLI với đúng bảng version của app (restore vào `verify_d06_old`): assert Account `Up` ~0, AI/Commerce hội tụ 1 lần rồi no-op, lần boot 2 no-op toàn bộ, đếm dòng bằng nhau, rồi dọn DB tạm (depends on T016) — ĐẠT ở mức goose CLI (ghi nhận trong verification-log.md); boot binary thật gộp vào T040

### US1 follow-up — mở rộng rewrite + sửa hạ tầng verify (phát hiện ở T016, quyết định điều phối)

- [X] T033 [P] [US1] Rewrite `services/account/database/migrations/20260627143730_add_token_estimation_audit.sql`: gỡ toàn bộ ALTER ai_usage_events (ai-only) thành no-op `SELECT 1`, giữ tên file + version, Down đảo ngược (file mới rỗng nên Down cũng no-op)
- [X] T034 [P] [US1] Rewrite `services/account/database/migrations/20260627225700_update_ai_cost_quota_limits.sql`: gỡ các UPDATE `ai_cost_policy_operations`, GIỮ NGUYÊN các UPDATE `subscription_plans`, giữ tên file + version, Down đảo ngược (ai → no-op, subscription giữ)
- [X] T035 [P] [US1] Rewrite `services/account/database/migrations/20260629072235_lowercase_constants_and_constraints.sql`: gỡ đúng các dòng chạm `ai_cost_policies`/`user_ai_policy_grants`/`ai_usage_events` (drop/constraint/UPDATE/index, cả Up lẫn Down), GIỮ NGUYÊN các dòng `brand_*`, giữ tên file + version
- [X] T036 [P] [US1] Rewrite `services/account/database/migrations/20260724173038_fix_total_tokens_and_check_constraint.sql`: gỡ toàn bộ DDL/data `ai_*` (ai-only) thành no-op `SELECT 1`, giữ tên file + version, Down đảo ngược (depends on T035 vì cùng họ lowercase — kiểm tra không xung đột thứ tự chạy)
- [X] T037 [US1] Follow-up `services/account/database/migrations/20260801000000_separate_module_schemas.sql`: gỡ các dòng chạm ai/dashboard còn sót — Up: 3 dòng DROP CONSTRAINT trên `public.ai_policy_grants` (21–23) + 1 dòng trên dashboard (28); Down: orphan-checks ai (135–146), re-add FK ai (164–171), orphan-check + re-add dashboard (159–161, 180–181); giữ toàn bộ dòng còn lại + comment cập nhật (depends on T033–T036)
- [X] T038 [US1] Init fix: bổ sung `CREATE TABLE public.user_style_profiles` (+ HNSW index) đã mất do drift vào `database/init/01-schema.sql`, khớp shape trong `20261011090000-Down` (public schema, vì init chạy trước split); verify: `grep user_style_profiles database/init/01-schema.sql` có kết quả
- [X] T039 Fix `Makefile`: `migrate-status/up/down/version` truyền `-table` đúng từng service (`account→goose_db_version`, `commerce→goose_db_version_commerce`, `ai→goose_db_version_ai`); verified trên DB tạm (row vào đúng bảng)
- [X] T040 [US1] Re-run V1+V1b quickstart sau T033–T039: replay trắng từ init hiện tại phải xanh toàn chain Account → AI → Commerce baseline; boot đồng thời 3 binary vào DB trắng quan sát lock serialize; assert version counts + tên constraint + đồng nhất schema V1 vs baseline-old; dọn DB tạm (depends on T033–T039; BLOCKS T023 rehearsal restore)

**Checkpoint**: US1 xong = single-writer đã thiết lập, kiểm chứng độc lập được trên DB cách ly

---

## Phase 4: User Story 2 - Ownership nhìn được và chặn được (Priority: P1)

**Goal**: Gate máy chặn 3 mẫu vi phạm; mọi tham chiếu tên cũ đã cập nhật; ADR + deferred D-06 resolved

**Independent Test**: 3 mẫu migration vi phạm (mỗi luật 1 mẫu) đều bị gate đỏ kèm tên luật + file + dòng; tìm kiếm tên constraint cũ toàn repo = 0 (trừ dòng lịch sử)

- [ ] T018 [P] [US2] Create gate script đọc `specs/030-ddl-ownership-handover/contracts/ownership-contract.md` làm single source of truth (vị trí theo quy ước Makefile, ví dụ `shared/tools/migration-ownership-check`), implement 3 luật G-ACCT/G-COMM/G-AI kèm ngoại lệ `REFERENCES identity.users(`
- [ ] T019 [US2] Makefile: thêm target `migration-check` và đấu vào `make check` sau `vet` (depends on T018)
- [ ] T020 [P] [US2] Chạy quickstart V3: 3 mẫu vi phạm đỏ + xóa mẫu xanh, ghi kết quả vào `specs/030-ddl-ownership-handover/verification-log.md` (depends on T019)
- [ ] T021 [US2] Update mọi tham chiếu tên constraint cũ sang `ai_policy_grants_effective_window_check`: file test T016 (`*_integration_test.go` có ALTER tay), `grant_error_class_integration_test.go:32`, `specs/027-grant-repair-unblock/runbook.md` (grep), docs kiến trúc/data-model có nhắc tên cũ
- [ ] T022 [US2] Docs: ADR bàn giao mới trong `docs/decisions/` + đánh dấu D-06 resolved trong `specs/027-grant-repair-unblock/deferred.md` kèm dấu vết sang spec 030

**Checkpoint**: US1 + US2 đều đứng độc lập — code xong và bằng chứng gate xong

---

## Phase 5: User Story 3 - Rehearsal + runbook + rollback sẵn sàng (Priority: P2)

**Goal**: Rehearsal đồng nhất 2 đường, runbook thao tác tay, rollback đã chứng minh — mọi thứ sẵn sàng chờ giờ G

**Independent Test**: Schema V1 (trắng mới) vs V2 (restore hội tụ) đồng nhất 100%; đếm dòng nghiệp vụ trước/sau bằng nhau; `Down` chạy về ổn định

- [ ] T023 [US3] Rehearsal quickstart V2: restore backup → `verify_d06_rehearsal`, chạy handover, diff schema V1 vs V2 (bảng/cột/index/constraint), đếm dòng toàn bộ bảng nghiệp vụ thuộc 8 schema module (`identity`, `subscription`, `wardrobe`, `fashion`, `brand`, `community`, `dashboard`, `aiexecutor`) trước/sau và assert bằng nhau tuyệt đối, đo thời gian giữ lock rename để chốt `lock_timeout`, ghi hết vào `specs/030-ddl-ownership-handover/verification-log.md` (depends on Phase 3); sau đó rehearsal quickstart V5 trên DB dev hiện tại (backup dev trước, schema dev y chang prod): pass 100% là cổng go/no-go giờ G, đỏ thì restore dev và dừng
- [ ] T024 [US3] Write `specs/030-ddl-ownership-handover/cutover-runbook.md` DÀNH CHO THAO TÁC TAY của chủ hệ thống: backup + verify restore, dừng traffic (full-stop 1–2h, không cần cơ chế chặn tự động), thứ tự deploy (Account→AI→Commerce), query monitor lock, ngưỡng abort, bước smoke, bước rollback = restore backup (depends on T023). BẮT BUỘC trong runbook: mọi lần boot binary kiểm chứng (V1b/V4/V5) phải tắt đường mail thật (`EMAIL_ENABLED=false`) — V1b đã lọt 1 mail tới địa chỉ seed vì thiếu bước này (trong ngưỡng hiến pháp nhưng không được lặp lại)
- [ ] T025 [US3] Test rollback trên rehearsal: chạy `Down` handover (tên cũ trở lại) + verify boot ổn định, chứng minh đường về hoạt động trước giờ G (depends on T024)

**Checkpoint**: US3 xong = rehearsal + runbook tay + rollback đã sẵn sàng chờ giờ G (chưa chạm prod)

---

## Phase 6: User Story 4 - Verify an toàn và quét cuối (Priority: P2)

**Goal**: Mọi SC còn lại được chứng minh bằng quét repo + chạy quickstart đầy đủ, không chạm data thật

**Independent Test**: 3 câu quét (tên cũ / lệnh xóa volume / target production) đều 0 kết quả ngoài cho phép; quickstart V1–V5 xanh hết (gồm V1b boot đồng thời và V5 DB dev)

- [ ] T026 [US4] Repo-wide scans: tên constraint cũ = 0 (trừ dòng lịch sử tài liệu bàn giao), tài liệu/kịch bản 0 lệnh xóa volume + 0 target production (SC-003, SC-006)
- [ ] T027 [US4] Run quickstart V1–V5 end-to-end trên database tạm (riêng V5 chạy trên DB dev đã backup trước theo quickstart), ghi kết quả vào `specs/030-ddl-ownership-handover/verification-log.md`, dọn đúng DB/stack tạm (depends on Phase 3–5)
- [ ] T028 [US4] Run `make check` đầy đủ (gồm `migration-check` mới) xanh và dỡ lệnh ngừng bắn FR-011 bằng thông báo nhóm (depends on T027)

**Checkpoint**: Mọi user story đã verify độc lập xong

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Đóng gói spec theo governance trước khi review/merge

- [ ] T029 Record frontend-guide exemption trong `tasks.md` này: 0 thay đổi endpoint/event/DTO client-facing (giữ nhánh này, xoá nhánh guide bên dưới)
- [ ] T030 Ghi mọi phát hiện ngoài phạm vi phát sinh trong lúc làm vào `specs/030-ddl-ownership-handover/deferred.md` (là gì / vì sao ngoài phạm vi / điều gì gỡ được + nêu số task bị ảnh hưởng nếu có), KHÔNG để `- [ ]` treo ở file này
- [ ] T031 Final report: đối chiếu SC-001..SC-007 với bằng chứng trong `verification-log.md`, liệt kê supported/ outstanding (nếu có) và đề xuất follow-up

### Frontend Integration Guide *(required — hiến phán §Governance)*

> Quyết định: **Không đổi bề mặt client** (xem T029 ở trên) — không thêm / xoá / sửa HTTP endpoint, event payload hay DTO client-facing → không viết `frontend-guide.md`. Nhánh "có đổi" không áp dụng vì spec đã chốt 0 thay đổi client-facing ở Assumptions.

### Phát hiện ngoài phạm vi *(hiến phán §Governance)*

> Thực hiện ở T030: ghi vào `specs/030-ddl-ownership-handover/deferred.md`, không để `- [ ]` treo ở file này.

---

## Phase 8: Go-live production — BẠN TỰ CHẠY TAY (không phải việc của agent)

**Purpose**: Chủ hệ thống thực hiện cutover trên prod theo runbook, vào thời điểm tự quyết, sau khi T001–T031 đã xanh hết dưới local

- [ ] T032 [GOLIVE] Bạn chạy tay `specs/030-ddl-ownership-handover/cutover-runbook.md` trên production trong cửa sổ full-stop 1–2h: backup + verify restore trước, deploy theo thứ tự runbook, smoke đăng ký→mua gói→AI→dashboard + đếm dòng nghiệp vụ trước/sau, abort và restore backup nếu bất kỳ check nào đỏ (agent không chạy task này)

**Checkpoint**: T032 xong = spec đóng — đối chiếu SC-005 lần cuối trên prod

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories (R1 chưa chứng minh thì không được đụng file cũ)
- **US1 (Phase 3, P1)**: After Foundational - tạo ra baseline để các phase sau dùng
- **US2 (Phase 4, P1)**: After Foundational; cần T014–T015 xong mới quét hết nghĩa (ownership đầy đủ) — chạy sau US1
- **US3 (Phase 5, P2)**: Depends on US1 (cần baseline + rewrite để rehearsal); kết thúc ở runbook + rollback test, chưa chạm prod
- **US4 (Phase 6, P2)**: Depends on US1–US3
- **Polish (Phase 7)**: Depends on all stories
- **Go-live (Phase 8, manual)**: Bạn tự chạy T032 theo runbook sau khi T001–T031 xanh hết; agent không chạy phase này

### User Story Dependencies

- **US1**: No dependencies on other stories (chỉ cần Foundational)
- **US2**: Nên sau US1 (gate quét toàn diện khi baseline đã tồn tại)
- **US3**: Bắt buộc sau US1 (rehearsal cần history mới)
- **US4**: Sau US1–US3 (quét cuối + V1–V4 đầy đủ)

### Within Each User Story

- Rewrite trước, baseline sau (T007–T013 → T014–T015), follow-up mở rộng (T033–T040: 4 file sót + T010 + init + Makefile), verify cuối (T016–T017, T040)
- Gate script trước, Makefile sau, mẫu đỏ sau cùng (T018 → T019 → T020)
- Rehearsal trước, runbook tay sau, rollback test cuối (T023 → T024 → T025); prod (T032) bạn tự chạy tay theo runbook khi T001–T031 đã xanh hết

### Parallel Opportunities

- [P] Setup: T002 + T003 song song
- [P] Foundational: T006 song song với T004/T005 (khác file, không phụ thuộc)
- [P] US1: T007–T013 toàn bộ song song (7 file khác nhau); T014–T015 song song sau khi rewrite xong
- [P] US2: T018 + T020-chuẩn-bị song song; T021 quét nhiều file song song nội bộ
- US3/US4: tuần tự (rehearsal → runbook → rollback test → quét cuối); prod (T032) tách riêng cuối cùng, bạn tự chạy

---

## Parallel Example: User Story 1 (rewrite wave)

```bash
# 7 file Account khác nhau — chạy song song:
Task: "T007 rewrite 20260622131631 (aiexecutor DDL)"
Task: "T008 rewrite 20260704062159 (model_configs + rename)"
Task: "T009 rewrite 20260723000001 (dashboard create)"
Task: "T010 rewrite 20260801000000 (SET SCHEMA)"
Task: "T011 rewrite 20260930120000 (4 DDL aiexecutor)"
Task: "T012 rewrite 20260930150000 (UPDATE data)"
Task: "T013 rewrite 20261011090002 (ALTER dashboard)"
# Sau đó T014 + T015 song song (AI baseline + Commerce baseline)
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup (điểm cắt + moratorium + snapshot)
2. Complete Phase 2: Foundational (R1 thực nghiệm + wiring AI)
3. Complete Phase 3: US1 (rewrite + baseline + replay trắng + boot hội tụ)
4. **STOP and VALIDATE**: single-writer đã thiết lập trên DB cách ly, `goose_db_version` vẫn 35 dòng
5. Đây là điểm dừng an toàn đầu tiên — US2–US4 có thể theo sau mà không ảnh hưởng

### Incremental Delivery

1. Setup + Foundational → R1 chứng minh + AI wired
2. + US1 → lịch sử sạch, baseline đứng được (giá trị cốt lõi)
3. + US2 → gate + docs (khóa lại, chống tái phạm)
4. + US3 → rehearsal + runbook tay + rollback test (sẵn sàng giờ G, chưa chạm prod)
5. + US4 + Polish → quét cuối + đóng spec theo governance (agent xong việc ở đây)

### Parallel Team Strategy

1. Cùng hoàn thành Setup + Foundational (R1 cần thống nhất trước)
2. Sau đó: Dev A rewrite wave US1 (T007–T013) + Dev B wiring/baseline US1 (T005/T014–T015) — khác file nên song song được
3. US2–US4 tuần tự theo dependency

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to spec user stories US1–US4 for traceability
- Mỗi task verify trên database tạm `verify_d06_*`; vi phạm FR-006 (đụng prod/xóa volume) = dừng ngay
- T032 (go-live prod) bạn tự chạy tay theo runbook sau khi T001–T031 xanh hết; agent kết thúc ở T031
- Timestamp baseline đã chốt số thật lúc implement: AI `20261016213000`, Commerce `20261016213001` (duy nhất, allowOutOfOrder=true nên hợp lệ)
