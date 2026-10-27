# Tasks: Thử nghiệm Ollama trên EC2 và tính tiền Gemini-only

**Input**: Design documents from `/specs/032-ollama-lightsail-trial/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Bắt buộc cho feature này vì spec yêu cầu bằng chứng contract/fallback/pricing (`FR-012`,
`FR-018`, `SC-002`, `SC-009`, `SC-011`). Viết test trước, xác nhận fail rồi mới implement.

**Organization**: Task theo user story để mỗi story có thể implement và kiểm chứng độc lập.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: chạy song song được (file khác nhau, không phụ thuộc task chưa xong)
- **[Story]**: user story tương ứng (`US1`–`US4`)
- Mọi task đều có đường dẫn file cụ thể

## Path Conventions

- Go workspace đa module: `shared/`, `services/<service>/`, `database/`
- Hạ tầng thử nghiệm: `deployments/terraform/ollama-ec2/`, `deployments/ollama/`
- Tài liệu vận hành: `docs/operations/`
- Không sửa: `deployments/docker-compose.aws.yml`, Dockerfile, `deployments/nginx/templates/aws/`,
  `.github/workflows/deploy-aws.yml`, bất kỳ artefact VPS nào.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Tạo khung tài liệu, cấu hình môi trường và hạ tầng thử nghiệm.

- [x] T001 Tạo khung `docs/operations/ollama-trial-runbook.md` và `docs/operations/ollama-trial-report.md` theo các mục trong `quickstart.md`
- [x] T002 Tạo mẫu baseline trong `docs/operations/ollama-trial-runbook.md` (commit, trạng thái 6 container, static IP Lightsail, Gemini text model/key reference, xác nhận không truy cập VPS); operator điền giá trị thật khi chạy
- [x] T003 [P] Cập nhật `.gitignore`: chặn `.terraform/`, `*.tfstate*`, `*.tfvars`; cho phép `!.env.ollama.example`
- [x] T004 [P] Tạo `.env.ollama.example` chỉ chứa placeholder phía EC2 (comment tiếng Anh theo quy định env)
- [x] T005 [P] Bổ sung 6 biến mới vào `.env.aws.example`: `AI_TEXT_MAX_CONCURRENCY`, `AI_TEXT_PRIMARY_TIMEOUT_SECONDS`, `AI_TEXT_FALLBACK_TIMEOUT_SECONDS`, `AI_OPENAI_REASONING_EFFORT`, `AI_CHAT_CHUNK_DEADLINE_SECONDS`, `AI_OPENAI_TRIAL_EXPECTED_HOSTS`; giữ nguyên 24 key của ba nhóm `*_TEXT_*` và 16 key `VISION_*`/`EMBEDDING_*`; mẫu chỉ minh họa — giá trị trial thật (cả 3 primary = `openai`) theo contract §1.1 và T039 (comment tiếng Anh)
- [x] T006 [P] Tạo khung `deployments/terraform/ollama-ec2/versions.tf`, `variables.tf`, `outputs.tf`, `templates/cloud-init.yaml.tftpl`; `terraform init` là bước operator khi Terraform có trong PATH
- [x] T007 [P] Tạo khung `deployments/ollama/bootstrap.sh` và `deployments/ollama/nginx-ollama.conf.template` với placeholder TLS/Bearer/loopback
- [x] T008 Tạo `deployments/ollama/local-test.ps1` để đo GPU/CPU và ghi hướng dẫn vào runbook; cài Ollama native + pull `qwen2.5:3b-instruct-q4_K_M` + ghi digest là bước operator nếu binary chưa có

**Checkpoint**: Có đủ khung để bắt đầu foundational và có bằng chứng baseline.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Cấu hình, attempt lifecycle, lỗi upstream và telemetry dùng chung cho mọi story.

**⚠️ CRITICAL**: Không bắt đầu user story nào trước khi phase này xong.

- [x] T009 Thêm field/default/validation cho `ai.text_max_concurrency` (default 2), `ai.text_primary_timeout_seconds` (default 30, trial 90), `ai.text_fallback_timeout_seconds` (default 30), `ai.openai_trial_expected_hosts` (CSV, trống = tắt cảnh báo), `ReasoningEffort` trong `shared/config/config.go`, `shared/config/validation.go`, `shared/config/loader.go`, `configs/config.production.yaml`, `configs/config.yaml`; dùng `overrideInt` trong `applyEnvOverrides` cho 4 biến int (3 field mới + `AI_CHAT_CHUNK_DEADLINE_SECONDS`); thêm env alias `AI_CHAT_CHUNK_DEADLINE_SECONDS` cho field hiện hữu `server.sse.ai_chat_chunk_deadline_seconds` (default 30, trial 90); map `AI_OPENAI_REASONING_EFFORT` vào `reasoning_effort` của cả 3 block `*_text_primary`; validate endpoint `https` + `/v1/chat/completions` (trừ loopback local), `AI_OPENAI_REASONING_EFFORT`, và `primary timeout <= chunk deadline`
- [x] T010 [P] Viết test config mới trong `shared/config/ai_trial_config_test.go` (default, env override, validation timeout tổng ≤120, primary timeout ≤ chunk deadline, concurrency ≥1, endpoint sai path, reasoning effort không hợp lệ, SSE deadline override)
- [x] T011 Thêm semaphore text dùng chung và helper deadline theo provider trong `shared/infrastructure/ai/core.go`; bỏ `http.Client.Timeout` toàn cục của text client, giữ timeout hiện hữu cho Google primary và vision/embedding
- [x] T012 Định nghĩa attempt lifecycle và interface `AdmitUnbilledAttempt`/`PrepareBillableAttempt` trong `services/ai/internal/modules/aiexecutor/application/interface/usecase/cost_uc.go` và `cost_types.go`; thêm `Hooks *AttemptHooks` vào `TextGenerationOptions` và `AttemptHooks`/`AttemptMeta`/`AttemptResult` vào `shared/application/ai/interfaces.go` (Route/BillingClass do orchestrator điền)
- [x] T013 Thêm helper phân loại unbilled theo provider và áp dụng vào `withUsageStamp`/`stampUsage` trong `shared/infrastructure/ai/text_provider.go` (provider `openai` thắng trước danh sách capability)
- [x] T014 Thêm typed upstream error giữ HTTP status và outcome `max_tokens` trong `shared/infrastructure/ai/openai/errors.go` (hoặc tách khỏi `openai.go` nếu file hiện tại quá lớn)
- [x] T015 Thêm telemetry helper với field bắt buộc + redaction trong `shared/infrastructure/ai/telemetry.go` (không nhận prompt/secret)
- [x] T016 [P] Viết test foundational cho stamping provider và mapping lỗi trong `shared/infrastructure/ai/text_provider_test.go`, `shared/infrastructure/ai/openai/errors_test.go`

**Checkpoint**: Config, semaphore, attempt contract, lỗi upstream và telemetry đã sẵn sàng.

---

## Phase 3: User Story 1 - Ollama phục vụ text AI qua EC2 (Priority: P1) 🎯 MVP

**Goal**: EC2 CPU-only chạy Ollama sau proxy TLS/Bearer, chỉ Lightsail truy cập được; adapter và
routing phục vụ 3 capability text bằng Ollama, vision/embedding vẫn Gemini.

**Independent Test**: Apply Terraform, tạo A record, xác minh proxy từ Lightsail được chấp nhận,
từ ngoài bị chặn, `11434` không public; gọi text thấy provider `openai`, gọi vision/embedding thấy
`google`.

### Tests for User Story 1 (REQUIRED) ⚠️

- [x] T017 [P] [US1] Viết contract test adapter trong `shared/infrastructure/ai/openai/openai_test.go`: request/header, non-stream, SSE, `[DONE]`, usage-only, zero/missing usage, malformed, 4xx/429/5xx, cancellation, không reasoning, reasoning effort rỗng = không gửi field, `finish_reason=length`, `json_schema`
- [x] T018 [P] [US1] Viết routing regression test trong `shared/infrastructure/ai/text_routing_test.go`: text chọn `openai`, vision/embedding chọn `google`, `summary`/`rewriter` route free → `free_text`/`FREE_TEXT` và route paid → `chat_text`/`CHAT_TEXT` (cả hai đều `openai` trong trial), tối đa 2 attempt, không có `freeFailover`

### Implementation for User Story 1

- [x] T019 [US1] Implement EC2/EIP/SG/EBS/IAM/SSM/Secrets Manager/Budget/tags trong `deployments/terraform/ollama-ec2/main.tf` với `c6i.2xlarge`, 443 chỉ từ `/32` Lightsail, 80 chỉ ACME, 11434 loopback; Budget 80 USD cảnh báo 50/80/100%; tag `Project`, `Feature=032`, `Environment=aws-course`, `Owner`, `ExpiresOn`
- [x] T020 [US1] Hoàn thiện `deployments/terraform/ollama-ec2/variables.tf`, `outputs.tf`, `versions.tf`, `templates/cloud-init.yaml.tftpl`: pin Ollama version + model digest, `OLLAMA_NUM_PARALLEL=2`, `OLLAMA_MAX_QUEUE`, pre-warm, Nginx TLS, certbot, kiểm tra không reasoning
- [x] T021 [US1] Implement proxy TLS/Bearer/allowlist/ACME trong `deployments/ollama/nginx-ollama.conf.template`
- [x] T022 [US1] Implement bootstrap idempotent trong `deployments/ollama/bootstrap.sh` (cài đúng version, pull digest, cấu hình systemd, chạy Nginx)
- [x] T023 [US1] Hoàn thiện adapter trong `shared/infrastructure/ai/openai/openai.go`: truyền `temperature` kể cả `0`, plumb `ReasoningEffort` từ `config.APIProviderConfig`/`TextProviderConfig` vào payload, map lỗi có status + `max_tokens`, luôn stamp provider/model winner; sửa `shared/infrastructure/ai/google/google_text.go` để stamp provider/model khi usage thiếu; validate structured response
- [x] T024 [US1] Sửa routing/fallback trong `shared/infrastructure/ai/text_service.go` và `text_provider.go`: bỏ `freeFailover`, đúng 1 Ollama + 1 Gemini, ma trận lỗi (gồm `max_tokens`), streaming chỉ fallback trước chunk đầu và không fallback giữa stream, fallback qua limiter Gemini, dùng semaphore + deadline từng attempt, dùng `AI_CHAT_CHUNK_DEADLINE_SECONDS` cho stream
- [x] T025 [US1] Chạy `deployments/ollama/local-test.ps1` trên máy dev: functional GPU + 50 mẫu latency reference; thử `OLLAMA_NUM_GPU=0` (bị desktop app giữ GPU, ghi blocked); chạy AI service end-to-end A5–A8 và ghi evidence vào `docs/operations/ollama-trial-runbook.md`
- [x] T026 [US1] Thực hiện apply 2 bước trong `docs/operations/ollama-trial-runbook.md`: `terraform apply -target=aws_eip.ollama` → tạo A record → apply phần còn lại; xác minh SSM, proxy, `11434`, digest và ghi bằng chứng
- [x] T027 [US1] Chạy `terraform -chdir=deployments/terraform/ollama-ec2 validate` + `plan` và `nginx -t` với `deployments/ollama/nginx-ollama.conf.template`; ghi kết quả vào `docs/operations/ollama-trial-runbook.md`

**Checkpoint**: US1 chạy độc lập: text do Ollama phục vụ, vision/embedding vẫn Gemini, truy cập ngoài bị chặn.

---

## Phase 4: User Story 2 - Dùng Ollama nhưng chỉ Gemini được tính tiền (Priority: P1)

**Goal**: Mọi attempt `provider=openai` có cost 0 và không chạm pricing/reserve; Gemini billed vẫn
reserve trước, confirm sau; fallback chỉ tính đúng provider thắng.

**Independent Test**: Chạy bộ test với dependency spy: `openai` thành công khi không có cấu hình
giá, 0 lời gọi pricing/reserve, `actual_cost=0`; Gemini billed vẫn gọi pricing/reserve/confirm;
fallback không tính trùng.

### Tests for User Story 2 (REQUIRED) ⚠️

- [x] T028 [P] [US2] Viết test admission trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/cost_admission_test.go`: `openai` không gọi pricing/reserve, event giữ `logical_route=plan.Route`, `observe_only` route paid vẫn tiêu thụ daily cap, `strict` không có attempt cap, route free/local không đổi, upsert idempotent, hết cap → degrade free route (không 429 mới), degraded fallback Gemini không billed
- [x] T029 [P] [US2] Viết test confirm trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/cost_confirm_test.go`: `openai` cost 0; Gemini billed giữ reserve/confirm; `free_text`/`embedding` unbilled giữ hành vi
- [x] T030 [P] [US2] Viết test orchestrator trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator_fallback_test.go`: `PrepareBillableAttempt` nâng cấp cùng event và reserve Gemini trước fallback, provider/model thắng, cancel/4xx không fallback, không tạo event thứ hai, không tính trùng, nhánh degraded free fallback Gemini không billed/không đụng cap

### Implementation for User Story 2

- [x] T031 [US2] Implement `AdmitUnbilledAttempt` và `PrepareBillableAttempt` trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/usecase.go` (tách admission khỏi pricing/reservation; `Prepare` không ghi event cho `openai`)
- [x] T032 [US2] Thêm đường non-monetary + nâng cấp event trong `services/ai/internal/modules/aiexecutor/domain/repositories/ai_cost_repo.go` và `services/ai/internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go`: `AdmitUnbilledAttempt` **upsert** theo `request_id` (idempotent, 2 lần gọi = 1 event, cap không tăng), giữ `logical_route=plan.Route`, `reserved=actual=0`; `PrepareBillableAttempt` khóa event theo `request_id` (`FOR UPDATE`), nâng cấp cùng event sang Gemini + reserved cost + ledger
- [x] T033 [US2] Refactor `CostUseCase.Prepare`/`Confirm` trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/usecase.go`: branch theo provider, guard `openai` cost 0 kể cả caller truyền cờ sai, giữ pricing cho Gemini billed
- [x] T034 [US2] Nối callback attempt-aware trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go` và `plan.go`: cài `AttemptHooks` để gọi `AdmitUnbilledAttempt` trước Ollama, `PrepareBillableAttempt` trước Gemini fallback, đóng attempt thất bại bằng telemetry; xử lý callback lỗi và cancellation
- [x] T035 [US2] Áp dụng provider-based stamping cho mọi call site trong `shared/infrastructure/ai/vision_embedding_service.go` và `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go` để vision/embedding không bị miễn phí sai
- [x] T036 [US2] Cập nhật `docs/operations/ollama-trial-runbook.md` và `.env.aws.example` để khẳng định `AI_UNBILLED_CAPABILITIES=free_text,embedding` chỉ dùng cho capability, còn rule `openai` nằm ở code

**Checkpoint**: US2 chạy độc lập: không có lời gọi pricing/reserve cho `openai`; Gemini billed không đổi; fallback không tính trùng.

---

## Phase 5: User Story 3 - Vận hành thử nghiệm trên đường deploy Lightsail hiện hữu (Priority: P2)

**Goal**: Bật/tắt thử nghiệm chỉ bằng env trên Lightsail, qua pipeline hiện hữu; rollback
Gemini-only ≤15 phút; không đổi Dockerfile/Compose/Nginx/pipeline và không đụng VPS.

**Independent Test**: Chỉ sửa `.env.aws`, chạy deploy hiện hữu, xác minh 6 container khỏe + text
qua Ollama; đổi lại Gemini và xác minh phục hồi, không thay artefact deployment.

- [x] T037 [US3] Viết quy trình bật/tắt, rollback và cập nhật allowlist khi static IP Lightsail đổi trong `docs/operations/ollama-trial-runbook.md` theo `contracts/env-ai-trial.md` §1.5
- [x] T038 [US3] Viết mục kiểm tra bất biến trong `docs/operations/ollama-trial-runbook.md`: `git diff` không chạm Dockerfile/`docker-compose.aws.yml`/Nginx AWS/`deploy-aws.yml`, không có tài nguyên EKS, không có lệnh VPS
- [ ] T039 [US3] Trên Lightsail: cập nhật `/opt/closy-aws/.env.aws` theo env contract (kể cả `deployments/scripts/aws-env/aws-env-bootstrap.ps1` cho key mới), chạy pipeline deploy hiện hữu, xác minh 6 container + 3 capability text do Ollama phục vụ, chạy load profile SC-005 (50 request/capability, concurrency 2, 30 phút), xác nhận Supabase/Redis Cloud/CloudAMQP/Cloudinary không đổi và uptime không phải tiêu chí; ghi bằng chứng
- [x] T040 [US3] Chạy kiểm tra cô lập trong `docs/operations/ollama-trial-runbook.md`: IP ngoài allowlist, thiếu/sai Bearer, truy cập `11434`, request hợp lệ từ Lightsail; ghi SC-003
- [x] T041 [US3] Chạy drill rollback về Gemini-only trong `docs/operations/ollama-trial-runbook.md`, đo thời gian ≤15 phút, xác minh không mất dữ liệu đã xác nhận; ghi SC-006
- [x] T042 [P] [US3] Chạy bộ regression API hiện hữu + xác nhận vision/embedding chọn Gemini và không có thay đổi client DTO; ghi SC-008

**Checkpoint**: US3 chạy độc lập: bật/tắt chỉ bằng env, rollback trong 15 phút, không đổi artefact deployment.

---

## Phase 6: User Story 4 - Đánh giá kết quả thử nghiệm sau 1 tuần (Priority: P2)

**Goal**: Đủ log/metric để so sánh 7 ngày và tạo báo cáo go/no-go; không tự rollback/tắt/teardown.

**Independent Test**: Chạy request AI, kiểm tra log/metric có provider/model/attempt/fallback/latency/
usage/CPU; tạo báo cáo từ dữ liệu đã thu và đối chiếu đủ mục SC-007.

- [x] T043 [US4] Thêm CloudWatch agent + log group + metric filter/dashboard cho Nginx/Ollama/CPU/RAM/disk trong `deployments/terraform/ollama-ec2/templates/cloud-init.yaml.tftpl` và `deployments/ollama/`, retention 14 ngày
- [x] T044 [US4] Nối metric AI (TTFT, latency, fallback reason, schema failure, thinking tokens, cost) và cảnh báo vận hành khi `provider=openai` nhưng host không khớp endpoint Ollama trial trong `shared/infrastructure/ai/telemetry.go`, `shared/infrastructure/ai/text_service.go`, `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go`
- [x] T045 [P] [US4] Viết test telemetry/redaction trong `shared/infrastructure/ai/telemetry_test.go`: đủ field bắt buộc, không có prompt/token/secret, `thinking_tokens` phải 0 trên `openai`, cảnh báo khi endpoint `openai` không khớp Ollama trial
- [x] T046 [US4] Hoàn thiện template báo cáo SC-007 trong `docs/operations/ollama-trial-report.md`: smoke, access/auth, lỗi, latency, fallback, queue, CPU/RAM/disk, chi phí AWS
- [x] T047 [US4] Viết quy trình ngày thứ 7 trong `docs/operations/ollama-trial-runbook.md`: thu số liệu, sự cố, chi phí, quyết định `go`/`adjust`/`no-go`, người quyết định, thời điểm; giữ cấu hình active; ghi rõ uptime không phải tiêu chí và các dịch vụ dữ liệu không đổi
- [ ] T048 [US4] Thực hiện báo cáo go/no-go trong `docs/operations/ollama-trial-report.md` và audit quyết định; ghi SC-007/SC-010
- [x] T049 [US4] Kiểm tra không có schedule/destroy/auto-teardown trong `deployments/terraform/ollama-ec2/` và runbook; ghi bằng chứng SC-010

**Checkpoint**: US4 chạy độc lập: có đủ dữ liệu 7 ngày và báo cáo go/no-go, không có hành động tự động sau ngày 7.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Gate chất lượng, đồng bộ tài liệu, xác nhận phạm vi và bàn giao.

- [x] T050 [P] Chạy `make lint` và test scoped `go test -race -count=3 ./shared/infrastructure/ai/... ./services/ai/internal/modules/aiexecutor/...`; sửa mọi lỗi
- [ ] T051 Chạy `make check` đầy đủ **chỉ khi dev yêu cầu push/tag**; không commit/push nếu chưa được phép
- [x] T052 Rà soát toàn bộ `quickstart.md`, chạy phần local (contract/routing tests, PowerShell parse) và ghi rõ phần live (Ollama/Terraform/Lightsail) là operator step trong runbook; cập nhật lệnh/đường dẫn nếu thực tế khác
- [x] T053 [P] Rà soát comment trong `.env.aws.example` và `.env.ollama.example` là tiếng Anh, không có secret thật
- [x] T054 [P] Xác nhận không có `aws-sdk` trong domain/application, không có nhánh code theo môi trường, không đổi Dockerfile/Compose/Nginx/pipeline/VPS, không đổi Supabase/Redis Cloud/CloudAMQP/Cloudinary; ghi diff evidence

### Frontend Integration Guide *(required — hiến pháp §Governance)*

- [x] T055 [P] **Không đổi bề mặt client**: không thêm/xoá/sửa HTTP endpoint, event payload hoặc DTO client-facing; không cần `frontend-guide.md` mới

### Phát hiện ngoài phạm vi *(required — hiến pháp §Governance)*

- [x] T056 [P] Tạo/cập nhật `specs/032-ollama-lightsail-trial/deferred.md` cho phát hiện ngoài phạm vi (VPS, EKS, readiness, migration dữ liệu); nếu không có, ghi rõ "không có phát hiện"

### Bàn giao

- [x] T057 Cập nhật chỉ mục bằng chứng cuối trong `docs/operations/ollama-trial-runbook.md` và đối chiếu đủ SC-001…SC-011

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: không phụ thuộc, bắt đầu ngay.
- **Foundational (Phase 2)**: phụ thuộc Setup; **chặn toàn bộ user story**.
- **US1 (Phase 3)**: sau Foundational.
- **US2 (Phase 4)**: sau Foundational; tích hợp với adapter/routing của US1 (T023/T024) nhưng test độc lập bằng spy.
- **US3 (Phase 5)**: sau US1 (T026) và US2 (T031–T034).
- **US4 (Phase 6)**: sau US1 (T023/T024) và US2 (T034).
- **Polish (Phase 7)**: sau tất cả story mong muốn.

### User Story Dependencies

- **US1 (P1)**: độc lập sau Foundational; không phụ thuộc US2.
- **US2 (P1)**: độc lập về test; cần adapter/routing US1 để chạy end-to-end trên Ollama.
- **US3 (P2)**: cần US1 + US2 vì bật env thật sẽ đi qua cả routing và billing.
- **US4 (P2)**: cần US1 + US2 để có dữ liệu provider/fallback/cost thật.

### Within Each User Story

- Test trước, xác nhận fail, rồi implement.
- Model/contract trước service; service trước endpoint/tích hợp.
- Hoàn tất checkpoint của story trước khi chuyển story kế tiếp.

### Parallel Opportunities

- Setup: T003/T004/T005/T006/T007 chạy song song.
- Foundational: T010/T011/T014/T015 chạy song song; T013 độc lập với T011.
- US1: T017/T018 test song song; T019/T021/T022/T023 có thể song song sau khi T020 xong variables.
- US2: T028/T029/T030 test song song; T031/T032 khác file có thể song song; T033 sau T031/T032; T034 sau T033.
- US4: T043/T044 song song; T045/T046/T047 song song.
- Polish: T053/T054/T055/T056 song song.

---

## Parallel Example: User Story 1

```text
# Tests first
Task: T017 contract test adapter in shared/infrastructure/ai/openai/openai_test.go
Task: T018 routing regression test in shared/infrastructure/ai/text_routing_test.go

# Infra song song sau khi variables xong
Task: T019 Terraform resources in deployments/terraform/ollama-ec2/main.tf
Task: T021 Nginx proxy template in deployments/ollama/nginx-ollama.conf.template
Task: T022 bootstrap script in deployments/ollama/bootstrap.sh
Task: T023 adapter implementation in shared/infrastructure/ai/openai/openai.go
```

## Parallel Example: User Story 2

```text
Task: T028 admission tests in .../cost/cost_admission_test.go
Task: T029 confirm tests in .../cost/cost_confirm_test.go
Task: T030 orchestrator fallback tests in .../orchestrator/orchestrator_fallback_test.go
```

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Hoàn tất Phase 1 Setup.
2. Hoàn tất Phase 2 Foundational (bắt buộc).
3. Hoàn tất US1 (hạ tầng + adapter + routing).
4. Hoàn tất US2 (billing split + fallback cost).
5. **STOP và VALIDATE**: chạy local gate + EC2 access/isolation + billing spy tests.
6. Chỉ khi hai story P1 đạt mới coi là MVP của thử nghiệm.

US2 không thể bỏ khỏi MVP: nếu chỉ làm US1, request `openai` có thể chạm pricing và fail hoặc ghi
cost sai, trái FR-013/FR-014.

### Incremental Delivery

1. Setup + Foundational → nền sẵn sàng.
2. US1 → test hạ tầng/adapter/routing độc lập.
3. US2 → test billing/fallback độc lập.
4. US3 → bật/tắt trên Lightsail + rollback drill.
5. US4 → thu số liệu 7 ngày + báo cáo go/no-go.
6. Polish → gate `make lint`/`make check`, đồng bộ tài liệu, xác nhận phạm vi.

### Parallel Team Strategy

- Sau Foundational: một người làm US1 (infra/adapter), một người làm US2 (cost/orchestrator) — hai
  nhánh chỉ gặp nhau ở T023/T024 và T034.
- US3/US4 chỉ bắt đầu sau khi US1+US2 hội tụ.

---

## Notes

- `[P]` = file khác nhau, không phụ thuộc task chưa xong.
- `[Story]` dùng để trace về `spec.md`.
- Không commit/push/merge nếu dev chưa cho phép.
- Không chạy `make check` trong vòng lặp sửa nhỏ; chỉ trước push/tag.
- Không chạy probe hay lệnh nào lên VPS.
- Nếu phát hiện việc ngoài phạm vi, ghi vào `specs/032-ollama-lightsail-trial/deferred.md`, không
  để lại checkbox ở file này.

---

## Phase 8: Convergence

- [x] T058 Bổ sung validation khởi động cho `provider` của mọi block AI (6 block text + vision/embedding) trong `shared/config/validation.go`: giá trị khác rỗng chỉ nhận `openai`/`google`, giá trị trống vẫn được phép để `.env.aws.example` và block chưa cấu hình tiếp tục khởi động; đồng thời từ chối entry của `AI_OPENAI_TRIAL_EXPECTED_HOSTS` chứa scheme hoặc path (chỉ host), kèm test trong `shared/config/ai_trial_config_test.go` per contracts/env-ai-trial.md §1.4 (missing)
- [x] T059 Bổ sung `/var/log/cloud-init-output.log` vào `collect_list` của CloudWatch agent trong `deployments/terraform/ollama-ec2/templates/cloud-init.yaml.tftpl` (cùng log group và retention 14 ngày; hiện chỉ thu `/var/log/ollama-trial-bootstrap.log`) và ghi nguồn log này vào mục 6.3 của `docs/operations/ollama-trial-runbook.md` per contracts/trial-telemetry.md §3 (partial)

## Phase 9: Convergence

- [x] T060 Phát đúng `environment=aws-course` trong attempt telemetry theo `contracts/trial-telemetry.md` §1, thay vì lấy nguyên `cfg.Server.Env` (`vps` trên Lightsail); giữ `SERVER_ENV`/ADR 011 nguyên vẹn, chỉ thêm nhãn trial cho log (không tạo nhánh business code theo môi trường) và thêm test khẳng định field `environment` của trial attempt log là `aws-course` trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/` hoặc `shared/infrastructure/ai/telemetry_test.go` per FR-020 + US4/AC1 + contracts/trial-telemetry.md §1 (partial)
