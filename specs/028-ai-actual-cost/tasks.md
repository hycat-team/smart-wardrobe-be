# Tasks: Chi phí AI thật phục vụ báo cáo lợi nhuận

**Input**: Design documents từ `specs/028-ai-actual-cost/` (spec.md, plan.md, research.md R1–R7, data-model.md, contracts/admin-api-delta.md, quickstart.md, manual-seed.md)

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: FR-009 yêu cầu kiểm thử tự động → mỗi user story có test task đi kèm, viết test TRƯỚC, đảm bảo FAIL trước khi implement.

**Organization**: Tasks nhóm theo user story để implement/test độc lập từng story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Chạy song song được (khác file, không phụ thuộc task chưa xong)
- **[Story]**: Story nào (US1, US2, US3)
- Mọi task ghi đường dẫn file chính xác

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Spike kiểm chứng + dữ liệu giá — chặn mọi task implement phía sau

- [x] T001 Spike `countTokens` cho embedding model bằng key production trong `shared/infrastructure/ai/google/google_tokens.go` (gọi tay `gemini-embedding-2` và `gemini-embedding-001` với 2–3 chunk mẫu, ghi nhận pass/fail vào `specs/028-ai-actual-cost/research.md` R3; nếu cả hai 400 thì embedding ở lại nhánh `usage_unreported`)
- [x] T002 Chạy `specs/028-ai-actual-cost/manual-seed.md` trên DB môi trường verify, xả Redis pricing cache (restart service verify hoặc chờ hết TTL 30 phút) rồi kiểm tra đủ 7 model bằng câu SELECT trong file (chặn Kịch bản 2: thiếu giá `gemini-3.5-flash-lite` hoặc đọc cache cũ thì vision rơi nhánh `pricing_config_missing` oan)

**Checkpoint**: T001 pass (hoặc chốt fallback) + T002 đủ 7 model + T025 có baseline — mới được sang Phase 2

---

## Phase 1b: Baseline quota (chặn so sánh SC-002)

**Purpose**: Chụp hiện trạng quota/policy trước mọi thay đổi để T017 so sánh "giống hệt"

- [x] T025 Chạy và lưu kết quả suite `services/ai/internal/modules/aiexecutor/application/usecase/grant/` + `.../usecase/cost/` (`go test -count=1`, lưu output) và ma trận case Kịch bản 5 `quickstart.md` hiện tại vào `specs/028-ai-actual-cost/baseline-quota.md` (chỉ số liệu, không code)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Xương sống ghi actual + mã đối soát — mọi user story đều phụ thuộc

**⚠️ CRITICAL**: Không task user story nào được bắt đầu cho tới khi phase này xong

- [ ] T003 [P] Mở rộng interface `Confirm` thêm `errorCode` trong `services/ai/internal/modules/aiexecutor/domain/repositories/ai_cost_repo.go`
- [ ] T004 [P] Thêm `ErrorCode` vào `AIEventDrillDownDTO` trong `services/ai/internal/modules/aiexecutor/domain/repositories/readmodels.go`
- [ ] T005 Sửa `Confirm` trong `services/ai/internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go` (bỏ ép `cost=0` khi route free; ledger luôn cộng `actual_cost_micro_vnd`; ghi `error_code`; `GetAIEventsDrillDown` đọc thêm cột `error_code`) — phụ thuộc T003, T004
- [ ] T006 Sửa `Confirm` trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/usecase.go` (5 nhánh tường minh: (1) paid giữ nguyên; (2) free đủ pricing+token → actual; (3) free thiếu pricing non-empty → `pricing_config_missing`; (4) provider/model rỗng → `pricing_config_missing`; (5) token 0 → `usage_unreported`) — phụ thuộc T005
- [ ] T007 Thêm field cộng thêm `error_code = 7` vào message `AIEventDrillDown` trong `contracts/ai/v1/ai.proto`, regen stub, map 2 chiều trong `services/ai/contract/aiexecutor/transport/to_proto.go` và `services/ai/contract/aiexecutor/transport/mapper.go`, rồi rà soát consumer drill-down phía commerce bằng `grep -rn "GetAIEventsDrillDown\|AIEventDrillDown" services/commerce/` (lưu output) — có dùng thì map tiếp field 7, không có thì ghi rõ "không có consumer" vào `specs/028-ai-actual-cost/contracts/admin-api-delta.md` để đóng yêu cầu contract
- [ ] T008 Chạy `make lint` + `go test -race -count=3` scoped `services/ai/...` sau Phase 2

**Checkpoint**: Foundation sẵn sàng — actual ghi đúng 5 nhánh (paid / free-đủ / thiếu-pricing / rỗng-provider-model / token-0), drill-down thấy `error_code`, proto regen xanh

---

## Phase 3: User Story 1 — Admin xem lợi nhuận đúng chi phí thật (Priority: P1) 🎯 MVP

**Goal**: Báo cáo AI theo tác vụ và lợi nhuận ròng phản ánh chi phí thật (vision + embedding có số, embedding đếm thất bại thì có mã)

**Independent Test**: Kịch bản 1–3 trong `specs/028-ai-actual-cost/quickstart.md` (đối chiếu tay `tokens × đơn giá × tỉ giá` trong 5 phút)

### Tests cho User Story 1 ⚠️

> Viết test TRƯỚC, đảm bảo FAIL trước khi implement

- [ ] T009 [P] [US1] Unit test 5 nhánh `costUC.Confirm` trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/` ((1) paid giữ nguyên; (2) free có token → actual>0; (3) thiếu pricing non-empty → mã; (4) provider/model rỗng → mã; (5) token 0 → mã) + case `Confirm` luôn dùng đúng provider/model được truyền vào (không tự resolve lại — khóa ngoại lệ fallback R7)
- [ ] T010 [P] [US1] Repo test cộng dồn ledger trong `services/ai/internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl_test.go` (actual cộng cả paid lẫn free; reserved/token bucket không đổi) + case lượt failed (Released) không cộng `actual_cost_micro_vnd` của ledger
- [ ] T011 [P] [US1] Test parse `usageMetadata` của `CallVisionWithUsage` trong `shared/infrastructure/ai/google/` (mock response có/không có usage)
- [ ] T026 [P] [US1] Test orchestrator `GenerateEmbeddings` trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/` (`Confirm` nhận token 0 → fail-open `usage_unreported`, user vẫn thành công) — viết trước T015

### Implementation cho User Story 1

- [ ] T012 [P] [US1] Parse `usageMetadata` + thêm `CallVisionWithUsage` trong `shared/infrastructure/ai/google/google_vision_embed.go` (giữ nguyên `CallVision` cũ cho caller commerce)
- [ ] T013 [P] [US1] Thêm method `...WithUsage` cộng thêm trong `shared/application/ai/interfaces.go` (không đổi signature cũ)
- [ ] T014 [US1] Orchestrator vision dùng `WithUsage` và `Confirm` bằng token thật trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go` (hàm `AnalyzeImage`) — phụ thuộc T012, T006
- [ ] T015 [US1] Orchestrator embedding fail-open `usage_unreported` trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go` (hàm `GenerateEmbeddings`): `Confirm` với token 0 (không gọi `countTokens` — spike T001 đã chứng minh 400 với cả hai model; nếu Google hỗ trợ sau này thì bật lại tại đúng điểm này) — phụ thuộc T001, T006, T026
- [ ] T016 [US1] Verify Kịch bản 1–3 + 6 `quickstart.md` trên môi trường đã seed T002 (commerce KHÔNG sửa code — snapshot/margin tự đúng qua `SUM(actual)`; Kb6 kiểm `overview`/`kpi-cards` đối chiếu `SUM(actual_cost_micro_vnd)`)

**Checkpoint**: US1 chạy độc lập được — vision chi phí > 0, embedding có chi phí hoặc có mã, báo cáo đối chiếu khớp giá

---

## Phase 4: User Story 2 — Người dùng cuối không bị ảnh hưởng (Priority: P1)

**Goal**: Mọi quyết định quota/cap/chuyển tuyến/từ chối giống hệt trước tính năng — không khoản thu mới

**Independent Test**: Kịch bản 5 trong `quickstart.md` (cùng bộ case hạn mức trước/sau deploy, kết quả + số dư giống hệt)

### Tests cho User Story 2 ⚠️

- [ ] T017 [P] [US2] Chạy lại toàn bộ suite quota/policy của T025 sau Phase 3 (`go test -race -count=3`), diff với `specs/028-ai-actual-cost/baseline-quota.md` — mọi case phải giống hệt (pass/fail + số dư), kết hợp Kịch bản 5 `quickstart.md`

### Implementation cho User Story 2

- [ ] T018 [US2] Rà soát diff xác nhận không đụng reserve/hard-cap/chuyển tuyến (`cost/usecase.go::reserveAgainstPolicy`, `plan.go`, `ai_cost_repo_impl.go::Reserve`) + chạy Kịch bản 5 `quickstart.md` — không code mới nếu T017 xanh

**Checkpoint**: US2 đạt — quota user nguyên vẹn, SC-002 thỏa

---

## Phase 5: User Story 3 — Vận hành tự rà soát giá toàn bộ model (Priority: P2)

**Goal**: Giá 7 model trên production đúng bảng giá Google mới nhất, upsert an toàn chạy lại được

**Independent Test**: SC-004 — chạy seed 2 lần liên tiếp (lần 2 đổi giá), cả hai thành công, giá cuối bằng lần 2

- [ ] T019 [US3] Chạy `manual-seed.md` trên production (sau deploy) 2 lần liên tiếp, lần 2 đổi 1 giá khác rồi assert giá cuối bằng lần 2 (SC-004); verify đủ 7 model; spot-check lượt MỚI sau seed không còn sinh `pricing_config_missing` cho model đã seed (event cũ giữ mã — đúng ngữ nghĩa snapshot lịch sử)

**Checkpoint**: US3 đạt — không còn lượt thiếu giá do thiếu row (trừ model mới chưa seed)

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Hoàn thiện báo cáo, guide client, kiểm tra chéo

- [ ] T020 Viết `specs/028-ai-actual-cost/frontend-guide.md` (nhánh **có đổi bề mặt client**: field `error_code` cộng thêm trong drill-down; các API còn lại unchanged về shape) rồi **đồng bộ guide** sau khi code ổn định theo quy tắc đếm số trong template
- [ ] T021 Kiểm tra ops `gemini-embedding-001` còn sống không (hiện là fallback trong `.env.production`); nếu đã sunset thì ghi `specs/028-ai-actual-cost/deferred.md` (là gì / vì sao ngoài phạm vi / điều gì gỡ được), KHÔNG sửa config trong feature này
- [ ] T022 Chạy full Kịch bản 1–6 `quickstart.md` end-to-end lần cuối + `make lint` + `go test -race -count=3` scoped các package đã chạm
- [ ] T023 Re-validate `specs/028-ai-actual-cost/checklists/requirements.md` sau mọi chỉnh sửa spec trong quá trình implement

### Phát hiện ngoài phạm vi *(hiến pháp §Governance)*

- [ ] T024 **Ghi vào `specs/028-ai-actual-cost/deferred.md`**, KHÔNG để nguyên `- [ ]` ở file này (áp dụng cho T021 và mọi phát hiện tương tự)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Không phụ thuộc — làm ngay; T001 quyết định hình dạng T015, T002 chặn verify
- **Foundational (Phase 2)**: Phụ thuộc Phase 1 — BLOCK toàn bộ user story
- **User Stories (Phase 3+)**: Phụ thuộc Phase 2; US1 → US2 → US3 theo ưu tiên (US2/US3 có thể song song sau US1 nếu đủ người)
- **Polish (Phase 6)**: Sau mọi story mong muốn; T020 sau khi code ổn định

### User Story Dependencies

- **US1 (P1)**: Sau Foundational — không phụ thuộc story khác (MVP)
- **US2 (P1)**: Sau US1 (cần code US1 để chạy so sánh quota)
- **US3 (P2)**: Độc lập sau Foundational, nhưng nên chạy sau deploy US1 để verify hết mã thiếu giá

### Within Each User Story

- Test FAIL trước, implement sau
- Interface/repo trước, usecase/orchestrator sau
- Story xong mới sang story ưu tiên tiếp theo

### Parallel Opportunities

- T003, T004, T007 (khác file) song song được; T005, T006 tuần tự theo sau
- T009, T010, T011 song song; T012, T013 song song; T014, T015 song song sau khi xong test + foundation
- T017 chạy song song với Phase 5 nếu khác người làm

---

## Parallel Example: User Story 1

```bash
# Test trước (song song):
Task: "Unit test 5 nhánh costUC.Confirm (T009)"
Task: "Repo test cộng dồn ledger + case failed không cộng actual (T010)"
Task: "Test parse usageMetadata CallVisionWithUsage (T011)"
Task: "Test orchestrator GenerateEmbeddings 2 nhánh count (T026)"

# Implement (song song sau khi test FAIL + foundation xong):
Task: "CallVisionWithUsage trong google_vision_embed.go (T012)"
Task: "Method ...WithUsage trong shared interfaces.go (T013)"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Spike + seed (T001, T002)
2. Phase 2: Foundational (T003–T008)
3. Phase 3: US1 (T009–T016)
4. **STOP và VALIDATE**: Kịch bản 1–3 + 6 quickstart + đối chiếu tay giá
5. Deploy/demo nếu sẵn sàng

### Incremental Delivery

1. Setup + Foundational → ghi actual đúng 5 nhánh
2. + US1 → báo cáo đúng (MVP!)
3. + US2 → khóa quota nguyên vẹn
4. + US3 → giá production đủ
5. + Polish → guide + deferred + gate cuối

---

## Notes

- [P] = khác file, không phụ thuộc task chưa xong
- [Story] trace về user story trong spec.md
- Commerce KHÔNG có task code — tự đúng qua `SUM(actual)`; nếu verify phát hiện lệch thì mở task mới, không nhét vào story hiện tại
- Commit sau mỗi task hoặc nhóm logic; không commit file `.env.production` hay secret
- `make check` full chỉ chạy trước push/tag, không chạy trong vòng lặp từng task

---

## Format Validation

- Tổng 26 task (T001–T026), 100% theo format `- [ ] [ID] [P?] [Story?] mô tả + đường dẫn file`
- US1: 9 task (T009–T016, T026) • US2: 2 task (T017–T018) • US3: 1 task (T019)
- Setup: 2 + Baseline quota: 1 (T025) • Foundational: 6 (gồm gate lint/test) • Polish: 5 (gồm frontend-guide + deferred + re-validate checklist)
