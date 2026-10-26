# Tasks: Thá»­ nghiá»‡m Ollama trÃªn EC2 vÃ  tÃ­nh tiá»n Gemini-only

**Input**: Design documents from `/specs/032-ollama-lightsail-trial/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Báº¯t buá»™c cho feature nÃ y vÃ¬ spec yÃªu cáº§u báº±ng chá»©ng contract/fallback/pricing (`FR-012`,
`FR-018`, `SC-002`, `SC-009`, `SC-011`). Viáº¿t test trÆ°á»›c, xÃ¡c nháº­n fail rá»“i má»›i implement.

**Organization**: Task theo user story Ä‘á»ƒ má»—i story cÃ³ thá»ƒ implement vÃ  kiá»ƒm chá»©ng Ä‘á»™c láº­p.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: cháº¡y song song Ä‘Æ°á»£c (file khÃ¡c nhau, khÃ´ng phá»¥ thuá»™c task chÆ°a xong)
- **[Story]**: user story tÆ°Æ¡ng á»©ng (`US1`â€“`US4`)
- Má»i task Ä‘á»u cÃ³ Ä‘Æ°á»ng dáº«n file cá»¥ thá»ƒ

## Path Conventions

- Go workspace Ä‘a module: `shared/`, `services/<service>/`, `database/`
- Háº¡ táº§ng thá»­ nghiá»‡m: `deployments/terraform/ollama-ec2/`, `deployments/ollama/`
- TÃ i liá»‡u váº­n hÃ nh: `docs/operations/`
- KhÃ´ng sá»­a: `deployments/docker-compose.aws.yml`, Dockerfile, `deployments/nginx/templates/aws/`,
  `.github/workflows/deploy-aws.yml`, báº¥t ká»³ artefact VPS nÃ o.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Táº¡o khung tÃ i liá»‡u, cáº¥u hÃ¬nh mÃ´i trÆ°á»ng vÃ  háº¡ táº§ng thá»­ nghiá»‡m.

- [x] T001 Táº¡o khung `docs/operations/ollama-trial-runbook.md` vÃ  `docs/operations/ollama-trial-report.md` theo cÃ¡c má»¥c trong `quickstart.md`
- [x] T002 Táº¡o máº«u baseline trong `docs/operations/ollama-trial-runbook.md` (commit, tráº¡ng thÃ¡i 6 container, static IP Lightsail, Gemini text model/key reference, xÃ¡c nháº­n khÃ´ng truy cáº­p VPS); operator Ä‘iá»n giÃ¡ trá»‹ tháº­t khi cháº¡y
- [x] T003 [P] Cáº­p nháº­t `.gitignore`: cháº·n `.terraform/`, `*.tfstate*`, `*.tfvars`; cho phÃ©p `!.env.ollama.example`
- [x] T004 [P] Táº¡o `.env.ollama.example` chá»‰ chá»©a placeholder phÃ­a EC2 (comment tiáº¿ng Anh theo quy Ä‘á»‹nh env)
- [x] T005 [P] Bá»• sung 6 biáº¿n má»›i vÃ o `.env.aws.example`: `AI_TEXT_MAX_CONCURRENCY`, `AI_TEXT_PRIMARY_TIMEOUT_SECONDS`, `AI_TEXT_FALLBACK_TIMEOUT_SECONDS`, `AI_OPENAI_REASONING_EFFORT`, `AI_CHAT_CHUNK_DEADLINE_SECONDS`, `AI_OPENAI_TRIAL_EXPECTED_HOSTS`; giá»¯ nguyÃªn 24 key cá»§a ba nhÃ³m `*_TEXT_*` vÃ  16 key `VISION_*`/`EMBEDDING_*`; máº«u chá»‰ minh há»a â€” giÃ¡ trá»‹ trial tháº­t (cáº£ 3 primary = `openai`) theo contract Â§1.1 vÃ  T039 (comment tiáº¿ng Anh)
- [x] T006 [P] Táº¡o khung `deployments/terraform/ollama-ec2/versions.tf`, `variables.tf`, `outputs.tf`, `templates/cloud-init.yaml.tftpl`; `terraform init` lÃ  bÆ°á»›c operator khi Terraform cÃ³ trong PATH
- [x] T007 [P] Táº¡o khung `deployments/ollama/bootstrap.sh` vÃ  `deployments/ollama/nginx-ollama.conf.template` vá»›i placeholder TLS/Bearer/loopback
- [x] T008 Táº¡o `deployments/ollama/local-test.ps1` Ä‘á»ƒ Ä‘o GPU/CPU vÃ  ghi hÆ°á»›ng dáº«n vÃ o runbook; cÃ i Ollama native + pull `qwen3:4b` + ghi digest lÃ  bÆ°á»›c operator náº¿u binary chÆ°a cÃ³

**Checkpoint**: CÃ³ Ä‘á»§ khung Ä‘á»ƒ báº¯t Ä‘áº§u foundational vÃ  cÃ³ báº±ng chá»©ng baseline.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Cáº¥u hÃ¬nh, attempt lifecycle, lá»—i upstream vÃ  telemetry dÃ¹ng chung cho má»i story.

**âš ï¸ CRITICAL**: KhÃ´ng báº¯t Ä‘áº§u user story nÃ o trÆ°á»›c khi phase nÃ y xong.

- [x] T009 ThÃªm field/default/validation cho `ai.text_max_concurrency` (default 2), `ai.text_primary_timeout_seconds` (default 30, trial 90), `ai.text_fallback_timeout_seconds` (default 30), `ai.openai_trial_expected_hosts` (CSV, trá»‘ng = táº¯t cáº£nh bÃ¡o), `ReasoningEffort` trong `shared/config/config.go`, `shared/config/validation.go`, `shared/config/loader.go`, `configs/config.production.yaml`, `configs/config.yaml`; dÃ¹ng `overrideInt` trong `applyEnvOverrides` cho 4 biáº¿n int (3 field má»›i + `AI_CHAT_CHUNK_DEADLINE_SECONDS`); thÃªm env alias `AI_CHAT_CHUNK_DEADLINE_SECONDS` cho field hiá»‡n há»¯u `server.sse.ai_chat_chunk_deadline_seconds` (default 30, trial 90); map `AI_OPENAI_REASONING_EFFORT` vÃ o `reasoning_effort` cá»§a cáº£ 3 block `*_text_primary`; validate endpoint `https` + `/v1/chat/completions` (trá»« loopback local), `AI_OPENAI_REASONING_EFFORT`, vÃ  `primary timeout <= chunk deadline`
- [x] T010 [P] Viáº¿t test config má»›i trong `shared/config/ai_trial_config_test.go` (default, env override, validation timeout tá»•ng â‰¤120, primary timeout â‰¤ chunk deadline, concurrency â‰¥1, endpoint sai path, reasoning effort khÃ´ng há»£p lá»‡, SSE deadline override)
- [x] T011 ThÃªm semaphore text dÃ¹ng chung vÃ  helper deadline theo provider trong `shared/infrastructure/ai/core.go`; bá» `http.Client.Timeout` toÃ n cá»¥c cá»§a text client, giá»¯ timeout hiá»‡n há»¯u cho Google primary vÃ  vision/embedding
- [x] T012 Äá»‹nh nghÄ©a attempt lifecycle vÃ  interface `AdmitUnbilledAttempt`/`PrepareBillableAttempt` trong `services/ai/internal/modules/aiexecutor/application/interface/usecase/cost_uc.go` vÃ  `cost_types.go`; thÃªm `Hooks *AttemptHooks` vÃ o `TextGenerationOptions` vÃ  `AttemptHooks`/`AttemptMeta`/`AttemptResult` vÃ o `shared/application/ai/interfaces.go` (Route/BillingClass do orchestrator Ä‘iá»n)
- [x] T013 ThÃªm helper phÃ¢n loáº¡i unbilled theo provider vÃ  Ã¡p dá»¥ng vÃ o `withUsageStamp`/`stampUsage` trong `shared/infrastructure/ai/text_provider.go` (provider `openai` tháº¯ng trÆ°á»›c danh sÃ¡ch capability)
- [x] T014 ThÃªm typed upstream error giá»¯ HTTP status vÃ  outcome `max_tokens` trong `shared/infrastructure/ai/openai/errors.go` (hoáº·c tÃ¡ch khá»i `openai.go` náº¿u file hiá»‡n táº¡i quÃ¡ lá»›n)
- [x] T015 ThÃªm telemetry helper vá»›i field báº¯t buá»™c + redaction trong `shared/infrastructure/ai/telemetry.go` (khÃ´ng nháº­n prompt/secret)
- [x] T016 [P] Viáº¿t test foundational cho stamping provider vÃ  mapping lá»—i trong `shared/infrastructure/ai/text_provider_test.go`, `shared/infrastructure/ai/openai/errors_test.go`

**Checkpoint**: Config, semaphore, attempt contract, lá»—i upstream vÃ  telemetry Ä‘Ã£ sáºµn sÃ ng.

---

## Phase 3: User Story 1 - Ollama phá»¥c vá»¥ text AI qua EC2 (Priority: P1) ðŸŽ¯ MVP

**Goal**: EC2 CPU-only cháº¡y Ollama sau proxy TLS/Bearer, chá»‰ Lightsail truy cáº­p Ä‘Æ°á»£c; adapter vÃ 
routing phá»¥c vá»¥ 3 capability text báº±ng Ollama, vision/embedding váº«n Gemini.

**Independent Test**: Apply Terraform, táº¡o A record, xÃ¡c minh proxy tá»« Lightsail Ä‘Æ°á»£c cháº¥p nháº­n,
tá»« ngoÃ i bá»‹ cháº·n, `11434` khÃ´ng public; gá»i text tháº¥y provider `openai`, gá»i vision/embedding tháº¥y
`google`.

### Tests for User Story 1 (REQUIRED) âš ï¸

- [x] T017 [P] [US1] Viáº¿t contract test adapter trong `shared/infrastructure/ai/openai/openai_test.go`: request/header, non-stream, SSE, `[DONE]`, usage-only, zero/missing usage, malformed, 4xx/429/5xx, cancellation, thinking off, reasoning effort rá»—ng = khÃ´ng gá»­i field, `finish_reason=length`, `json_schema`
- [x] T018 [P] [US1] Viáº¿t routing regression test trong `shared/infrastructure/ai/text_routing_test.go`: text chá»n `openai`, vision/embedding chá»n `google`, `summary`/`rewriter` route free â†’ `free_text`/`FREE_TEXT` vÃ  route paid â†’ `chat_text`/`CHAT_TEXT` (cáº£ hai Ä‘á»u `openai` trong trial), tá»‘i Ä‘a 2 attempt, khÃ´ng cÃ³ `freeFailover`

### Implementation for User Story 1

- [x] T019 [US1] Implement EC2/EIP/SG/EBS/IAM/SSM/Secrets Manager/Budget/tags trong `deployments/terraform/ollama-ec2/main.tf` vá»›i `c6i.2xlarge`, 443 chá»‰ tá»« `/32` Lightsail, 80 chá»‰ ACME, 11434 loopback; Budget 80 USD cáº£nh bÃ¡o 50/80/100%; tag `Project`, `Feature=032`, `Environment=aws-course`, `Owner`, `ExpiresOn`
- [x] T020 [US1] HoÃ n thiá»‡n `deployments/terraform/ollama-ec2/variables.tf`, `outputs.tf`, `versions.tf`, `templates/cloud-init.yaml.tftpl`: pin Ollama version + model digest, `OLLAMA_NUM_PARALLEL=2`, `OLLAMA_MAX_QUEUE`, pre-warm, Nginx TLS, certbot, kiá»ƒm tra thinking off
- [x] T021 [US1] Implement proxy TLS/Bearer/allowlist/ACME trong `deployments/ollama/nginx-ollama.conf.template`
- [x] T022 [US1] Implement bootstrap idempotent trong `deployments/ollama/bootstrap.sh` (cÃ i Ä‘Ãºng version, pull digest, cáº¥u hÃ¬nh systemd, cháº¡y Nginx)
- [x] T023 [US1] HoÃ n thiá»‡n adapter trong `shared/infrastructure/ai/openai/openai.go`: truyá»n `temperature` ká»ƒ cáº£ `0`, plumb `ReasoningEffort` tá»« `config.APIProviderConfig`/`TextProviderConfig` vÃ o payload, map lá»—i cÃ³ status + `max_tokens`, luÃ´n stamp provider/model winner; sá»­a `shared/infrastructure/ai/google/google_text.go` Ä‘á»ƒ stamp provider/model khi usage thiáº¿u; validate structured response
- [x] T024 [US1] Sá»­a routing/fallback trong `shared/infrastructure/ai/text_service.go` vÃ  `text_provider.go`: bá» `freeFailover`, Ä‘Ãºng 1 Ollama + 1 Gemini, ma tráº­n lá»—i (gá»“m `max_tokens`), streaming chá»‰ fallback trÆ°á»›c chunk Ä‘áº§u vÃ  khÃ´ng fallback giá»¯a stream, fallback qua limiter Gemini, dÃ¹ng semaphore + deadline tá»«ng attempt, dÃ¹ng `AI_CHAT_CHUNK_DEADLINE_SECONDS` cho stream
- [ ] T025 [US1] Cháº¡y `deployments/ollama/local-test.ps1` trÃªn mÃ¡y dev: GPU cho contract/fallback, `OLLAMA_NUM_GPU=0` cho latency CPU; ghi káº¿t quáº£ vÃ o `docs/operations/ollama-trial-runbook.md`
- [ ] T026 [US1] Thá»±c hiá»‡n apply 2 bÆ°á»›c trong `docs/operations/ollama-trial-runbook.md`: `terraform apply -target=aws_eip.ollama` â†’ táº¡o A record â†’ apply pháº§n cÃ²n láº¡i; xÃ¡c minh SSM, proxy, `11434`, digest vÃ  ghi báº±ng chá»©ng
- [ ] T027 [US1] Cháº¡y `terraform -chdir=deployments/terraform/ollama-ec2 validate` + `plan` vÃ  `nginx -t` vá»›i `deployments/ollama/nginx-ollama.conf.template`; ghi káº¿t quáº£ vÃ o `docs/operations/ollama-trial-runbook.md`

**Checkpoint**: US1 cháº¡y Ä‘á»™c láº­p: text do Ollama phá»¥c vá»¥, vision/embedding váº«n Gemini, truy cáº­p ngoÃ i bá»‹ cháº·n.

---

## Phase 4: User Story 2 - DÃ¹ng Ollama nhÆ°ng chá»‰ Gemini Ä‘Æ°á»£c tÃ­nh tiá»n (Priority: P1)

**Goal**: Má»i attempt `provider=openai` cÃ³ cost 0 vÃ  khÃ´ng cháº¡m pricing/reserve; Gemini billed váº«n
reserve trÆ°á»›c, confirm sau; fallback chá»‰ tÃ­nh Ä‘Ãºng provider tháº¯ng.

**Independent Test**: Cháº¡y bá»™ test vá»›i dependency spy: `openai` thÃ nh cÃ´ng khi khÃ´ng cÃ³ cáº¥u hÃ¬nh
giÃ¡, 0 lá»i gá»i pricing/reserve, `actual_cost=0`; Gemini billed váº«n gá»i pricing/reserve/confirm;
fallback khÃ´ng tÃ­nh trÃ¹ng.

### Tests for User Story 2 (REQUIRED) âš ï¸

- [x] T028 [P] [US2] Viáº¿t test admission trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/cost_admission_test.go`: `openai` khÃ´ng gá»i pricing/reserve, event giá»¯ `logical_route=plan.Route`, `observe_only` route paid váº«n tiÃªu thá»¥ daily cap, `strict` khÃ´ng cÃ³ attempt cap, route free/local khÃ´ng Ä‘á»•i, upsert idempotent, háº¿t cap â†’ degrade free route (khÃ´ng 429 má»›i), degraded fallback Gemini khÃ´ng billed
- [x] T029 [P] [US2] Viáº¿t test confirm trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/cost_confirm_test.go`: `openai` cost 0; Gemini billed giá»¯ reserve/confirm; `free_text`/`embedding` unbilled giá»¯ hÃ nh vi
- [x] T030 [P] [US2] Viáº¿t test orchestrator trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator_fallback_test.go`: `PrepareBillableAttempt` nÃ¢ng cáº¥p cÃ¹ng event vÃ  reserve Gemini trÆ°á»›c fallback, provider/model tháº¯ng, cancel/4xx khÃ´ng fallback, khÃ´ng táº¡o event thá»© hai, khÃ´ng tÃ­nh trÃ¹ng, nhÃ¡nh degraded free fallback Gemini khÃ´ng billed/khÃ´ng Ä‘á»¥ng cap

### Implementation for User Story 2

- [x] T031 [US2] Implement `AdmitUnbilledAttempt` vÃ  `PrepareBillableAttempt` trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/usecase.go` (tÃ¡ch admission khá»i pricing/reservation; `Prepare` khÃ´ng ghi event cho `openai`)
- [x] T032 [US2] ThÃªm Ä‘Æ°á»ng non-monetary + nÃ¢ng cáº¥p event trong `services/ai/internal/modules/aiexecutor/domain/repositories/ai_cost_repo.go` vÃ  `services/ai/internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go`: `AdmitUnbilledAttempt` **upsert** theo `request_id` (idempotent, 2 láº§n gá»i = 1 event, cap khÃ´ng tÄƒng), giá»¯ `logical_route=plan.Route`, `reserved=actual=0`; `PrepareBillableAttempt` khÃ³a event theo `request_id` (`FOR UPDATE`), nÃ¢ng cáº¥p cÃ¹ng event sang Gemini + reserved cost + ledger
- [x] T033 [US2] Refactor `CostUseCase.Prepare`/`Confirm` trong `services/ai/internal/modules/aiexecutor/application/usecase/cost/usecase.go`: branch theo provider, guard `openai` cost 0 ká»ƒ cáº£ caller truyá»n cá» sai, giá»¯ pricing cho Gemini billed
- [x] T034 [US2] Ná»‘i callback attempt-aware trong `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go` vÃ  `plan.go`: cÃ i `AttemptHooks` Ä‘á»ƒ gá»i `AdmitUnbilledAttempt` trÆ°á»›c Ollama, `PrepareBillableAttempt` trÆ°á»›c Gemini fallback, Ä‘Ã³ng attempt tháº¥t báº¡i báº±ng telemetry; xá»­ lÃ½ callback lá»—i vÃ  cancellation
- [x] T035 [US2] Ãp dá»¥ng provider-based stamping cho má»i call site trong `shared/infrastructure/ai/vision_embedding_service.go` vÃ  `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go` Ä‘á»ƒ vision/embedding khÃ´ng bá»‹ miá»…n phÃ­ sai
- [x] T036 [US2] Cáº­p nháº­t `docs/operations/ollama-trial-runbook.md` vÃ  `.env.aws.example` Ä‘á»ƒ kháº³ng Ä‘á»‹nh `AI_UNBILLED_CAPABILITIES=free_text,embedding` chá»‰ dÃ¹ng cho capability, cÃ²n rule `openai` náº±m á»Ÿ code

**Checkpoint**: US2 cháº¡y Ä‘á»™c láº­p: khÃ´ng cÃ³ lá»i gá»i pricing/reserve cho `openai`; Gemini billed khÃ´ng Ä‘á»•i; fallback khÃ´ng tÃ­nh trÃ¹ng.

---

## Phase 5: User Story 3 - Váº­n hÃ nh thá»­ nghiá»‡m trÃªn Ä‘Æ°á»ng deploy Lightsail hiá»‡n há»¯u (Priority: P2)

**Goal**: Báº­t/táº¯t thá»­ nghiá»‡m chá»‰ báº±ng env trÃªn Lightsail, qua pipeline hiá»‡n há»¯u; rollback
Gemini-only â‰¤15 phÃºt; khÃ´ng Ä‘á»•i Dockerfile/Compose/Nginx/pipeline vÃ  khÃ´ng Ä‘á»¥ng VPS.

**Independent Test**: Chá»‰ sá»­a `.env.aws`, cháº¡y deploy hiá»‡n há»¯u, xÃ¡c minh 6 container khá»e + text
qua Ollama; Ä‘á»•i láº¡i Gemini vÃ  xÃ¡c minh phá»¥c há»“i, khÃ´ng thay artefact deployment.

- [ ] T037 [US3] Viáº¿t quy trÃ¬nh báº­t/táº¯t, rollback vÃ  cáº­p nháº­t allowlist khi static IP Lightsail Ä‘á»•i trong `docs/operations/ollama-trial-runbook.md` theo `contracts/env-ai-trial.md` Â§1.5
- [ ] T038 [US3] Viáº¿t má»¥c kiá»ƒm tra báº¥t biáº¿n trong `docs/operations/ollama-trial-runbook.md`: `git diff` khÃ´ng cháº¡m Dockerfile/`docker-compose.aws.yml`/Nginx AWS/`deploy-aws.yml`, khÃ´ng cÃ³ tÃ i nguyÃªn EKS, khÃ´ng cÃ³ lá»‡nh VPS
- [ ] T039 [US3] TrÃªn Lightsail: cáº­p nháº­t `/opt/closy-aws/.env.aws` theo env contract (ká»ƒ cáº£ `deployments/scripts/aws-env/aws-env-bootstrap.ps1` cho key má»›i), cháº¡y pipeline deploy hiá»‡n há»¯u, xÃ¡c minh 6 container + 3 capability text do Ollama phá»¥c vá»¥, cháº¡y load profile SC-005 (50 request/capability, concurrency 2, 30 phÃºt), xÃ¡c nháº­n Supabase/Redis Cloud/CloudAMQP/Cloudinary khÃ´ng Ä‘á»•i vÃ  uptime khÃ´ng pháº£i tiÃªu chÃ­; ghi báº±ng chá»©ng
- [ ] T040 [US3] Cháº¡y kiá»ƒm tra cÃ´ láº­p trong `docs/operations/ollama-trial-runbook.md`: IP ngoÃ i allowlist, thiáº¿u/sai Bearer, truy cáº­p `11434`, request há»£p lá»‡ tá»« Lightsail; ghi SC-003
- [ ] T041 [US3] Cháº¡y drill rollback vá» Gemini-only trong `docs/operations/ollama-trial-runbook.md`, Ä‘o thá»i gian â‰¤15 phÃºt, xÃ¡c minh khÃ´ng máº¥t dá»¯ liá»‡u Ä‘Ã£ xÃ¡c nháº­n; ghi SC-006
- [ ] T042 [P] [US3] Cháº¡y bá»™ regression API hiá»‡n há»¯u + xÃ¡c nháº­n vision/embedding chá»n Gemini vÃ  khÃ´ng cÃ³ thay Ä‘á»•i client DTO; ghi SC-008

**Checkpoint**: US3 cháº¡y Ä‘á»™c láº­p: báº­t/táº¯t chá»‰ báº±ng env, rollback trong 15 phÃºt, khÃ´ng Ä‘á»•i artefact deployment.

---

## Phase 6: User Story 4 - ÄÃ¡nh giÃ¡ káº¿t quáº£ thá»­ nghiá»‡m sau 1 tuáº§n (Priority: P2)

**Goal**: Äá»§ log/metric Ä‘á»ƒ so sÃ¡nh 7 ngÃ y vÃ  táº¡o bÃ¡o cÃ¡o go/no-go; khÃ´ng tá»± rollback/táº¯t/teardown.

**Independent Test**: Cháº¡y request AI, kiá»ƒm tra log/metric cÃ³ provider/model/attempt/fallback/latency/
usage/CPU; táº¡o bÃ¡o cÃ¡o tá»« dá»¯ liá»‡u Ä‘Ã£ thu vÃ  Ä‘á»‘i chiáº¿u Ä‘á»§ má»¥c SC-007.

- [ ] T043 [US4] ThÃªm CloudWatch agent + log group + metric filter/dashboard cho Nginx/Ollama/CPU/RAM/disk trong `deployments/terraform/ollama-ec2/templates/cloud-init.yaml.tftpl` vÃ  `deployments/ollama/`, retention 14 ngÃ y
- [ ] T044 [US4] Ná»‘i metric AI (TTFT, latency, fallback reason, schema failure, thinking tokens, cost) vÃ  cáº£nh bÃ¡o váº­n hÃ nh khi `provider=openai` nhÆ°ng host khÃ´ng khá»›p endpoint Ollama trial trong `shared/infrastructure/ai/telemetry.go`, `shared/infrastructure/ai/text_service.go`, `services/ai/internal/modules/aiexecutor/application/usecase/orchestrator/orchestrator.go`
- [ ] T045 [P] [US4] Viáº¿t test telemetry/redaction trong `shared/infrastructure/ai/telemetry_test.go`: Ä‘á»§ field báº¯t buá»™c, khÃ´ng cÃ³ prompt/token/secret, `thinking_tokens` pháº£i 0 trÃªn `openai`, cáº£nh bÃ¡o khi endpoint `openai` khÃ´ng khá»›p Ollama trial
- [ ] T046 [US4] HoÃ n thiá»‡n template bÃ¡o cÃ¡o SC-007 trong `docs/operations/ollama-trial-report.md`: smoke, access/auth, lá»—i, latency, fallback, queue, CPU/RAM/disk, chi phÃ­ AWS
- [ ] T047 [US4] Viáº¿t quy trÃ¬nh ngÃ y thá»© 7 trong `docs/operations/ollama-trial-runbook.md`: thu sá»‘ liá»‡u, sá»± cá»‘, chi phÃ­, quyáº¿t Ä‘á»‹nh `go`/`adjust`/`no-go`, ngÆ°á»i quyáº¿t Ä‘á»‹nh, thá»i Ä‘iá»ƒm; giá»¯ cáº¥u hÃ¬nh active; ghi rÃµ uptime khÃ´ng pháº£i tiÃªu chÃ­ vÃ  cÃ¡c dá»‹ch vá»¥ dá»¯ liá»‡u khÃ´ng Ä‘á»•i
- [ ] T048 [US4] Thá»±c hiá»‡n bÃ¡o cÃ¡o go/no-go trong `docs/operations/ollama-trial-report.md` vÃ  audit quyáº¿t Ä‘á»‹nh; ghi SC-007/SC-010
- [ ] T049 [US4] Kiá»ƒm tra khÃ´ng cÃ³ schedule/destroy/auto-teardown trong `deployments/terraform/ollama-ec2/` vÃ  runbook; ghi báº±ng chá»©ng SC-010

**Checkpoint**: US4 cháº¡y Ä‘á»™c láº­p: cÃ³ Ä‘á»§ dá»¯ liá»‡u 7 ngÃ y vÃ  bÃ¡o cÃ¡o go/no-go, khÃ´ng cÃ³ hÃ nh Ä‘á»™ng tá»± Ä‘á»™ng sau ngÃ y 7.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Gate cháº¥t lÆ°á»£ng, Ä‘á»“ng bá»™ tÃ i liá»‡u, xÃ¡c nháº­n pháº¡m vi vÃ  bÃ n giao.

- [ ] T050 [P] Cháº¡y `make lint` vÃ  test scoped `go test -race -count=3 ./shared/infrastructure/ai/... ./services/ai/internal/modules/aiexecutor/...`; sá»­a má»i lá»—i
- [ ] T051 Cháº¡y `make check` Ä‘áº§y Ä‘á»§ **chá»‰ khi dev yÃªu cáº§u push/tag**; khÃ´ng commit/push náº¿u chÆ°a Ä‘Æ°á»£c phÃ©p
- [ ] T052 Cháº¡y láº¡i toÃ n bá»™ `quickstart.md`, cáº­p nháº­t lá»‡nh/Ä‘Æ°á»ng dáº«n náº¿u thá»±c táº¿ khÃ¡c; ghi káº¿t quáº£ vÃ o runbook
- [ ] T053 [P] RÃ  soÃ¡t comment trong `.env.aws.example` vÃ  `.env.ollama.example` lÃ  tiáº¿ng Anh, khÃ´ng cÃ³ secret tháº­t
- [ ] T054 [P] XÃ¡c nháº­n khÃ´ng cÃ³ `aws-sdk` trong domain/application, khÃ´ng cÃ³ nhÃ¡nh code theo mÃ´i trÆ°á»ng, khÃ´ng Ä‘á»•i Dockerfile/Compose/Nginx/pipeline/VPS, khÃ´ng Ä‘á»•i Supabase/Redis Cloud/CloudAMQP/Cloudinary; ghi diff evidence

### Frontend Integration Guide *(required â€” hiáº¿n phÃ¡p Â§Governance)*

- [ ] T055 [P] **KhÃ´ng Ä‘á»•i bá» máº·t client**: khÃ´ng thÃªm/xoÃ¡/sá»­a HTTP endpoint, event payload hoáº·c DTO client-facing; khÃ´ng cáº§n `frontend-guide.md` má»›i

### PhÃ¡t hiá»‡n ngoÃ i pháº¡m vi *(required â€” hiáº¿n phÃ¡p Â§Governance)*

- [ ] T056 [P] Táº¡o/cáº­p nháº­t `specs/032-ollama-lightsail-trial/deferred.md` cho phÃ¡t hiá»‡n ngoÃ i pháº¡m vi (VPS, EKS, readiness, migration dá»¯ liá»‡u); náº¿u khÃ´ng cÃ³, ghi rÃµ "khÃ´ng cÃ³ phÃ¡t hiá»‡n"

### BÃ n giao

- [ ] T057 Cáº­p nháº­t chá»‰ má»¥c báº±ng chá»©ng cuá»‘i trong `docs/operations/ollama-trial-runbook.md` vÃ  Ä‘á»‘i chiáº¿u Ä‘á»§ SC-001â€¦SC-011

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: khÃ´ng phá»¥ thuá»™c, báº¯t Ä‘áº§u ngay.
- **Foundational (Phase 2)**: phá»¥ thuá»™c Setup; **cháº·n toÃ n bá»™ user story**.
- **US1 (Phase 3)**: sau Foundational.
- **US2 (Phase 4)**: sau Foundational; tÃ­ch há»£p vá»›i adapter/routing cá»§a US1 (T023/T024) nhÆ°ng test Ä‘á»™c láº­p báº±ng spy.
- **US3 (Phase 5)**: sau US1 (T026) vÃ  US2 (T031â€“T034).
- **US4 (Phase 6)**: sau US1 (T023/T024) vÃ  US2 (T034).
- **Polish (Phase 7)**: sau táº¥t cáº£ story mong muá»‘n.

### User Story Dependencies

- **US1 (P1)**: Ä‘á»™c láº­p sau Foundational; khÃ´ng phá»¥ thuá»™c US2.
- **US2 (P1)**: Ä‘á»™c láº­p vá» test; cáº§n adapter/routing US1 Ä‘á»ƒ cháº¡y end-to-end trÃªn Ollama.
- **US3 (P2)**: cáº§n US1 + US2 vÃ¬ báº­t env tháº­t sáº½ Ä‘i qua cáº£ routing vÃ  billing.
- **US4 (P2)**: cáº§n US1 + US2 Ä‘á»ƒ cÃ³ dá»¯ liá»‡u provider/fallback/cost tháº­t.

### Within Each User Story

- Test trÆ°á»›c, xÃ¡c nháº­n fail, rá»“i implement.
- Model/contract trÆ°á»›c service; service trÆ°á»›c endpoint/tÃ­ch há»£p.
- HoÃ n táº¥t checkpoint cá»§a story trÆ°á»›c khi chuyá»ƒn story káº¿ tiáº¿p.

### Parallel Opportunities

- Setup: T003/T004/T005/T006/T007 cháº¡y song song.
- Foundational: T010/T011/T014/T015 cháº¡y song song; T013 Ä‘á»™c láº­p vá»›i T011.
- US1: T017/T018 test song song; T019/T021/T022/T023 cÃ³ thá»ƒ song song sau khi T020 xong variables.
- US2: T028/T029/T030 test song song; T031/T032 khÃ¡c file cÃ³ thá»ƒ song song; T033 sau T031/T032; T034 sau T033.
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

1. HoÃ n táº¥t Phase 1 Setup.
2. HoÃ n táº¥t Phase 2 Foundational (báº¯t buá»™c).
3. HoÃ n táº¥t US1 (háº¡ táº§ng + adapter + routing).
4. HoÃ n táº¥t US2 (billing split + fallback cost).
5. **STOP vÃ  VALIDATE**: cháº¡y local gate + EC2 access/isolation + billing spy tests.
6. Chá»‰ khi hai story P1 Ä‘áº¡t má»›i coi lÃ  MVP cá»§a thá»­ nghiá»‡m.

US2 khÃ´ng thá»ƒ bá» khá»i MVP: náº¿u chá»‰ lÃ m US1, request `openai` cÃ³ thá»ƒ cháº¡m pricing vÃ  fail hoáº·c ghi
cost sai, trÃ¡i FR-013/FR-014.

### Incremental Delivery

1. Setup + Foundational â†’ ná»n sáºµn sÃ ng.
2. US1 â†’ test háº¡ táº§ng/adapter/routing Ä‘á»™c láº­p.
3. US2 â†’ test billing/fallback Ä‘á»™c láº­p.
4. US3 â†’ báº­t/táº¯t trÃªn Lightsail + rollback drill.
5. US4 â†’ thu sá»‘ liá»‡u 7 ngÃ y + bÃ¡o cÃ¡o go/no-go.
6. Polish â†’ gate `make lint`/`make check`, Ä‘á»“ng bá»™ tÃ i liá»‡u, xÃ¡c nháº­n pháº¡m vi.

### Parallel Team Strategy

- Sau Foundational: má»™t ngÆ°á»i lÃ m US1 (infra/adapter), má»™t ngÆ°á»i lÃ m US2 (cost/orchestrator) â€” hai
  nhÃ¡nh chá»‰ gáº·p nhau á»Ÿ T023/T024 vÃ  T034.
- US3/US4 chá»‰ báº¯t Ä‘áº§u sau khi US1+US2 há»™i tá»¥.

---

## Notes

- `[P]` = file khÃ¡c nhau, khÃ´ng phá»¥ thuá»™c task chÆ°a xong.
- `[Story]` dÃ¹ng Ä‘á»ƒ trace vá» `spec.md`.
- KhÃ´ng commit/push/merge náº¿u dev chÆ°a cho phÃ©p.
- KhÃ´ng cháº¡y `make check` trong vÃ²ng láº·p sá»­a nhá»; chá»‰ trÆ°á»›c push/tag.
- KhÃ´ng cháº¡y probe hay lá»‡nh nÃ o lÃªn VPS.
- Náº¿u phÃ¡t hiá»‡n viá»‡c ngoÃ i pháº¡m vi, ghi vÃ o `specs/032-ollama-lightsail-trial/deferred.md`, khÃ´ng
  Ä‘á»ƒ láº¡i checkbox á»Ÿ file nÃ y.
