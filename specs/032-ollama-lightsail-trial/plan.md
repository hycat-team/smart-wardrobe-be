# Implementation Plan: Thử nghiệm Ollama trên EC2 cho backend Lightsail

**Branch**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26 | **Spec**: [spec.md](./spec.md)

**Input**: Đặc tả tính năng tại [spec.md](./spec.md)

## Summary

Giữ nguyên backend ba dịch vụ (sáu container) trên box AWS Lightsail hiện hữu và chỉ thay đổi cấu
hình môi trường AI. Thêm một EC2 CPU-only `c6i.2xlarge` chạy Ollama `qwen3:4b` (pin version +
digest), đặt sau Nginx proxy có TLS và Bearer, chỉ cho static IP của box Lightsail truy cập qua
security group; EC2, Elastic IP, EBS và secret được quản lý bằng một module Terraform tối giản
(local state) và quản trị qua Systems Manager.

Trước khi lên EC2, cùng model/digest được chạy local bằng Ollama native (GPU để test chức năng,
sau đó ép CPU để đo latency tham chiếu). Go AI Service tiếp tục dùng adapter OpenAI-compatible
nhưng phải bổ sung contract test, phân loại lỗi, vòng đời attempt và tắt thinking của qwen3.
Ollama là primary cho `chat_text`, `recommendation_text`, `free_text`; Gemini chỉ fallback tối đa
một lần cho lỗi đủ điều kiện và vẫn là primary cho vision/embedding. Attempt `provider=openai` đi
qua nhánh usage vận hành không tính tiền, không gọi pricing hay reserve/confirm chi phí. Attempt
Gemini có tính phí vẫn được pricing và reserve trước khi gọi. Timeout mỗi attempt: Ollama 90 giây,
Gemini 30 giây; concurrency 2 được giữ ở cả Go AI Service và Ollama. Không thay đổi Dockerfile,
Compose, Nginx, pipeline deploy hiện hữu; không tạo tài nguyên EKS/Kubernetes; không thay đổi DNS
của API; không thao tác hoặc phụ thuộc VPS.

**Scope authority**: AI Engine (`services/ai/**`) nằm trong phạm vi feature này vì FR-013–FR-018
yêu cầu thay đổi admission/usage/cost và FR-020 yêu cầu telemetry attempt. Đây không phải vi phạm
Constitution §I. Worker contract hiện có rule cấm sửa `services/ai`; các phase chạm `services/ai`
sẽ được giao cho subagent `general` (vẫn áp dụng nguyên tắc: không git, không gate, không đổi
scope), không sửa worker contract trong feature này.

## Technical Context

**Language/Version**: Go 1.26.0; Terraform HCL; Docker Compose (Lightsail hiện hữu); Nginx; shell

**Primary Dependencies**: Gin 1.12, GORM 1.31, gRPC 1.83, RabbitMQ client 1.15, Redis client 9.22,
Goose 3.28, Ollama `qwen3:4b` (pin runtime version + digest) qua OpenAI-compatible API, AWS
EC2/Elastic IP/EBS/SSM/Secrets Manager

**Storage**: Giữ nguyên Supabase PostgreSQL, Redis Cloud, CloudAMQP và Cloudinary; EBS gp3 trên
EC2 giữ model Ollama; AWS Secrets Manager chỉ giữ Bearer token của proxy. Không thêm bảng nghiệp
vụ hoặc migration database.

**Testing**: `go test`, `go test -race`, `httptest`, contract/integration test adapter AI (bao gồm
thinking off và structured output), Terraform `validate`/`plan`, kiểm tra cấu hình Nginx proxy,
smoke test local trước, smoke test trên Lightsail và drill bật/tắt Ollama có kiểm soát

**Target Platform**: Box Lightsail hiện hữu; một EC2 Linux CPU-only `c6i.2xlarge` (8 vCPU/16 GiB)
tại `ap-southeast-1`; public HTTPS qua proxy riêng; frontend tiếp tục trên Vercel; test trước trên
máy dev i5-13500HX/RTX 4050 6 GiB bằng Ollama native

**Project Type**: Backend ba dịch vụ độc lập + worker, hạ tầng EC2 tối giản và cấu hình môi trường

**Performance Goals**: Gate theo SC-005 của spec. Load profile: 50 request cho mỗi capability text
(`chat_text`, `free_text`, `recommendation_text`), concurrency 2, cửa sổ 30 phút; p95 tính trên mẫu
này. ≥95% `chat_text`/`free_text` có TTFT ≤15 giây; ≥95% `recommendation_text` có TTFT ≤60 giây và
hoàn tất ≤120 giây; ≥95% attempt Ollama hoàn tất trong 90 giây. Attempt vượt 90 giây bị hủy và
fallback Gemini được thử trong ngân sách còn lại; request bị write timeout 120 giây cắt được ghi
nhận là failure. Uptime không phải gate.

**Constraints**: Một tuần thử nghiệm, không có request/người dùng thật; đúng một attempt Ollama và
tối đa một attempt Gemini; timeout 90 giây/30 giây; concurrency 2; Ollama không public; không chạm
VPS; không đổi public endpoint/payload/event/client DTO; cấu hình Ollama tiếp tục active sau ngày
7 đến quyết định thủ công; không log prompt, token xác thực hoặc secret; ngân sách cảnh báo 80 USD.
Tổng 90 + 30 bằng `write_timeout_seconds = 120`, nên fallback là best-effort: request dài có thể bị
cắt ở biên và phải được ghi nhận là failure trong báo cáo, không tự tăng timeout. SSE chat hiện có
`server.sse.ai_chat_chunk_deadline_seconds = 30`, nhỏ hơn attempt Ollama 90 giây; field này **đã
tồn tại** và phải được override qua env `AI_CHAT_CHUNK_DEADLINE_SECONDS=90` trong trial (giữ default
30 cho môi trường khác) để không cắt stream trước khi fallback kịp chạy. Text client hiện có
`http.Client.Timeout` 30/45/30 giây; phải bỏ timeout toàn cục cho text và dùng deadline từng attempt.

**Scale/Scope**: Sáu container Lightsail giữ nguyên; một EC2 Ollama `c6i.2xlarge` 8 vCPU/16 GiB,
EBS gp3 50 GiB, model `qwen3:4b` pin version/digest; concurrency 2 ở cả Go AI Service và Ollama.

## Constitution Check

*GATE: phải đạt trước Phase 0 và được kiểm tra lại sau Phase 1.*

### Trước Phase 0

| Gate | Kết quả | Bằng chứng/điều kiện |
|------|---------|----------------------|
| Giữ đúng ba service độc lập | PASS | Lightsail vẫn chạy Account, Commerce, AI với HTTP/worker cùng image; thử nghiệm không thêm service nghiệp vụ. |
| Ranh giới module và giao tiếp liên service | PASS | Không đổi domain/application boundary; gRPC và RabbitMQ/outbox hiện hữu được giữ nguyên. |
| Image/config portable giữa môi trường | PASS | Không tạo nhánh business code theo môi trường; khác biệt chỉ ở env/endpoint. AWS SDK không đi vào domain/application. |
| External data services và database portability | PASS | Supabase/Redis Cloud/CloudAMQP/Cloudinary giữ nguyên; không dùng tính năng database độc quyền AWS. |
| AI metering/reservation-first | PASS CÓ ĐIỀU KIỆN | `openai` là unbilled và không đi qua logic tiền tệ; trước mỗi Gemini attempt có tính phí phải pricing + reserve thành công rồi mới gọi provider. |
| Independent deployment | PASS | Không thay đổi pipeline; từng service vẫn phát hành độc lập qua `deploy-aws.yml`. Thử nghiệm chỉ tác động env của AI. |
| Frontend contract documentation | PASS | Không thay đổi public contract; không thêm endpoint public nên không cần frontend guide mới. |
| Email verification safety | PASS | Drill dùng queue không gửi email hoặc tắt `email.enabled`; nếu kiểm tra email thật thì tối đa một email tới một tài khoản test. |
| VPS độc lập | PASS | Không sửa, deploy, probe, đọc secret, đổi DNS hay gọi workflow VPS; kiểm chứng bằng diff/repository evidence. |
| Hiến pháp có cần sửa | PASS | Đây là thay đổi cấu hình provider và một EC2 phụ, không định nghĩa lại boundary hoặc nguyên tắc kiến trúc. |

Không có vi phạm hiến pháp cần biện minh trước khi nghiên cứu.

Ghi chú cho gate Reservation-first: `provider=openai` được miễn tiền theo quy tắc sản phẩm đã ghi
trong spec (assumption: `openai` là phân loại không tính tiền, kể cả khi endpoint khác Ollama),
không phải dùng bảng giá 0 để lách reservation-first. Trong trial, endpoint `openai` chỉ trỏ tới
Ollama; cấu hình sai tới endpoint trả phí phải tạo cảnh báo vận hành và không được coi là hợp lệ.

### Sau Phase 1

| Gate | Kết quả sau thiết kế |
|------|----------------------|
| Ba service/independent deploy | PASS — sáu container Lightsail giữ nguyên; EC2 Ollama là dependency ngoài, không phải service thứ tư. |
| Portability | PASS — khác biệt nằm ở env và endpoint; Terraform/EC2 tách khỏi business code; không nhánh code theo môi trường. |
| Reservation-first | PASS — contract attempt yêu cầu `PrepareBillableAttempt` hoàn tất trước khi Gemini được gọi; OpenAI dùng đường `AdmitUnbilledAttempt` riêng, không gọi pricing/cost reservation. |
| Public contract/frontend | PASS — không thêm endpoint, không đổi HTTP/event/DTO; frontend không phải thay đổi. |
| Data ownership | PASS — không có migration hoặc store nghiệp vụ mới; usage event hiện hữu giữ kết quả cuối, attempt chi tiết nằm ở telemetry. |
| Vận hành an toàn | PASS — allowlist security group + TLS + Bearer; Ollama chỉ bind loopback; không có trạng thái hai provider được tính tiền. |

## Project Structure

### Documentation (this feature)

```text
specs/032-ollama-lightsail-trial/
├── plan.md
├── research.md
├── data-model.md
├── contracts/
│   ├── env-ai-trial.md
│   ├── ollama-openai-subset.md
│   ├── attempt-lifecycle.md
│   └── trial-telemetry.md
├── quickstart.md
├── checklists/requirements.md
└── tasks.md                         # sinh ở /speckit-tasks, không thuộc /speckit-plan
```

### Source Code (repository root)

```text
deployments/
├── terraform/ollama-ec2/            # mới: EC2 c6i.2xlarge, EIP, SG, EBS, IAM/SSM, secret
│   ├── versions.tf                  # provider + local state, không backend S3
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   └── templates/cloud-init.yaml.tftpl
├── ollama/                          # mới: bootstrap + proxy config
│   ├── bootstrap.sh
│   ├── nginx-ollama.conf.template
│   └── local-test.ps1               # chạy Ollama native + ép CPU để đo latency
├── docker-compose.aws.yml           # KHÔNG sửa
└── nginx/templates/aws/             # KHÔNG sửa

.env.aws.example                     # chỉ bổ sung biến AI, concurrency, timeout (không secret)
.env.ollama.example                  # mới: placeholder phía EC2
deployments/scripts/aws-env/aws-env-bootstrap.ps1   # bổ sung key AI mới vào allowlist copy
docs/operations/ollama-trial-runbook.md
docs/operations/ollama-trial-report.md   # mẫu báo cáo go/no-go ngày thứ 7

shared/config/
├── config.go                        # thêm concurrency + timeout primary/fallback
├── validation.go                    # validate concurrency > 0, timeout > 0
├── loader.go                        # overrideInt cho env mới; SSE deadline alias
├── ai_billing.go                    # giữ nguyên cơ chế unbilled capability
└── ai_trial_config_test.go          # mới: default/env/validation

shared/infrastructure/ai/
├── core.go                          # semaphore dùng chung cho text + timeout riêng mỗi attempt
├── telemetry.go                     # mới: field chuẩn + redaction
├── telemetry_test.go                # mới: redaction/required fields
├── openai/openai.go                 # typed error, temperature, thinking off, usage winner
├── openai/errors.go                 # mới: typed upstream error + max_tokens
├── openai/openai_test.go            # mới: contract test
├── google/google_text.go            # luôn stamp provider/model khi usage thiếu
├── text_service.go                  # bỏ freeFailover cho text, phân loại fallback
├── text_provider.go                 # routing text + timeout/limiter theo attempt
└── text_routing_test.go             # mới: routing regression

services/ai/internal/modules/aiexecutor/
├── application/interface/usecase/   # thêm AdmitUnbilledAttempt/PrepareBillableAttempt
├── application/usecase/cost/        # tách admission khỏi cost reservation
└── application/usecase/orchestrator/# gọi callback đúng attempt

configs/config.production.yaml       # default chung; override timeout/concurrency qua env Lightsail
configs/config.yaml                  # default local tương đương để quickstart chạy được
```

**Structure Decision**: Giữ nguyên multi-module workspace, sáu container Lightsail và pipeline
`deploy-aws.yml`. EC2 Ollama là hạ tầng phụ được quản lý bằng Terraform tối giản, không dùng
Kubernetes. Nginx proxy chỉ là lớp xác thực/TLS trước Ollama, không phải service nghiệp vụ và
không thay thế Nginx gateway của Lightsail. Cây trên liệt kê các file chính; `tasks.md` là nguồn
đầy đủ cho mọi file sẽ tạo/sửa (gồm `shared/application/ai/interfaces.go`,
`shared/infrastructure/ai/vision_embedding_service.go`, `.../domain/repositories/ai_cost_repo.go`,
`.../infrastructure/persistence/ai_cost_repo_impl.go`, các test `cost_admission_test.go`,
`orchestrator_fallback_test.go` và `deferred.md`).

## Key Implementation Changes

### 1. Adapter OpenAI-compatible

- Endpoint cấu hình phải là URL đầy đủ tới `/v1/chat/completions` vì adapter POST nguyên chuỗi và
  không tự nối path.
- Thêm typed upstream error giữ được HTTP status (429/5xx/4xx) thay vì ép mọi non-2xx thành 503;
  `finish_reason=length`/`max_tokens` là outcome riêng, đủ điều kiện fallback và có test.
- Truyền `temperature` kể cả giá trị 0; giữ `response_format`/`json_schema`, `stream_options`.
- Plumbing `reasoning_effort`: thêm `ReasoningEffort` vào `config.APIProviderConfig` và
  `app_ai.TextProviderConfig`, map qua `overrideProvider`/`toTextProviderConfig`, rồi truyền vào
  payload OpenAI. Giá trị rỗng = không gửi field; contract test cả hai trường hợp.
- Luôn phát provider/model winner ngay cả khi usage bằng 0 hoặc thiếu; áp dụng cho cả
  `openai.CallTextStream` và `google.CallTextStream` (hoặc stamp tại `text_service` sau attempt),
  để fallback Gemini không bị ghi nhầm provider Ollama.
- Validate structured response theo `ResponseSchema` trước khi trả content.
- Contract test: request/header, non-stream, SSE, `[DONE]`, usage-only, zero/missing usage,
  cancellation, malformed body, pinned Ollama version/model.
- qwen3: gửi `reasoning_effort: "none"` (env `AI_OPENAI_REASONING_EFFORT`, trống = không gửi) để
  tắt thinking; contract test phải chứng minh response không chứa thinking/reasoning content và
  JSON vẫn hợp lệ. Nếu Ollama/version được pin bỏ qua field này, dùng `/no_think` ở prompt và ghi
  lại bằng chứng; không bắt đầu đo khi chưa chứng minh.
- Timeout: mỗi attempt có deadline riêng (Ollama 90 giây, Gemini 30 giây), không dùng chung
  `client.Timeout` cho primary và fallback; context của attempt bị hủy khi hết hạn để fallback còn
  ngân sách trong `write_timeout_seconds = 120`.

### 2. Routing và fallback

- `chat_text`, `recommendation_text`, `free_text` dùng `openai` primary; vision/embedding giữ
  Gemini primary. `summary`/`rewriter` đi theo route nghiệp vụ hiện hữu: route free dùng nhóm
  `free_text`, route paid dùng nhóm `chat_text`; trial đặt cả hai nhóm primary = `openai`.
- Bỏ `freeFailover` khỏi đường text: đúng một primary attempt và tối đa một Gemini fallback.
- Fallback chỉ khi timeout/deadline upstream, transport/connection, HTTP 429, HTTP 5xx, response
  malformed/empty hoặc output/schema validation thất bại trước khi trả content.
- Không fallback khi `context.Canceled`, HTTP 4xx khác 429, auth/config/unsupported request.
- Streaming chỉ fallback nếu chưa emit chunk; sau chunk đầu thì trả lỗi trên stream, không ghép
  nội dung từ provider khác.
- Fallback phải qua limiter Gemini.
- Concurrency 2: một semaphore dùng chung cho mọi attempt text tới Ollama
  (chat/recommendation/free/summary/rewriter) đặt trong Go AI Service (mặc định 2, cấu hình qua
  env) và `OLLAMA_NUM_PARALLEL=2` + `OLLAMA_MAX_QUEUE` trên EC2; semaphore giữ chỗ trước limiter
  và nhả sau khi attempt kết thúc để không giữ slot khi chờ rate limit.
- Deadline theo provider: attempt tới `openai`/Ollama dùng
  `ai.text_primary_timeout_seconds = 90` (trial) cho mọi text, kể cả nhánh free; attempt Gemini
  fallback dùng `ai.text_fallback_timeout_seconds = 30`. Khi primary là `google` (ngoài trial),
  giữ timeout hiện hữu của từng file config (production 30/45/30; local 30/30/30); không áp 90 giây
  cho Gemini primary.

### 3. Tách admission/usage không tính tiền khỏi cost reservation

1. `provider=openai`: gọi `AdmitUnbilledAttempt` để mirror đúng admission hiện hữu —
   `observe_only` kiểm tra daily attempt cap theo `logical_route=paid`; `strict` không có attempt
   cap (hard-cost/unknown không áp vì cost 0) và luôn admit — rồi tạo/cập nhật usage event với
   `logical_route = plan.Route` (giữ route nghiệp vụ), provider `openai`,
   `reserved_cost=actual_cost=0`; tuyệt đối không gọi pricing repository, phép tính tiền,
   `ReserveBillableAttempt` hoặc `ConfirmBillableAttempt`. Attempt `openai` trên route paid vẫn
   tiêu thụ daily attempt cap hiện hữu (không thêm quota mới; cap chỉ đếm event
   `logical_route=paid`, route free/local không đổi); rate limiter và semaphore vẫn áp dụng. Nếu hết
   cap, giữ hành vi degrade hiện hữu: ghi event `logical_route=free`, không reserve, vẫn gọi Ollama
   (không trả 429 mới); nếu Ollama lỗi eligible trong nhánh degraded, fallback Gemini ở free route
   (không gọi `PrepareBillableAttempt`, không billed, không đụng paid cap).
2. Nếu Ollama lỗi đủ điều kiện, đóng attempt bằng telemetry và trước khi gọi Gemini phải chạy
   `PrepareBillableAttempt`: `SELECT ... FOR UPDATE` usage event theo `request_id`, lấy pricing,
   cập nhật cùng event sang provider/model Gemini + reserved cost, và cộng ledger. Không tạo event
   thứ hai vì `ai_usage_events.request_id` là UNIQUE.
3. Chỉ khi `PrepareBillableAttempt` thành công mới gọi Gemini; sau đó confirm/release theo
   provider/model thực tế trên cùng event.
4. Usage event database ghi provider thắng và kết quả cuối; từng attempt được audit bằng structured
   log/metric để không phải thêm bảng.

Repository path non-monetary phải có tên/API riêng, không tái sử dụng hàm cost `Reserve` với số 0.
`CostUseCase.Prepare` không ghi usage event cho provider `openai` (chỉ trả decision/limit);
`AdmitUnbilledAttempt` là writer duy nhất và phải **upsert** theo `request_id`
(`ON CONFLICT (request_id) DO UPDATE`), idempotent nếu `BeforeAttempt` bị gọi lại. Với Gemini
primary ngoài trial, `Prepare` giữ hành vi cũ.
`withUsageStamp`/`stampUsage` phải đặt `Unbilled=true` khi `provider=openai` **trước** khi xét danh
sách capability, để `Confirm` không tra pricing. Confirm có guard theo provider để `openai` luôn
cost 0 ngay cả khi caller truyền cờ sai. Không dùng `AI_UNBILLED_CAPABILITIES` để triển khai quy
tắc này vì nó sẽ làm Gemini fallback của capability text thành miễn phí.

Attempt callback nằm trong `shared/application/ai` (ví dụ `AttemptHooks{BeforeAttempt,
AfterAttempt}`), được `text_service` gọi ngay trước/sau mỗi provider call cho cả non-stream và
stream; orchestrator (services/ai) cài đặt callback để gọi `AdmitUnbilledAttempt` hoặc
`PrepareBillableAttempt`. Callback trả lỗi thì không gọi provider; cancellation không tạo fallback.


### 4. Hạ tầng EC2 và cấu hình

- Terraform local state (không commit, không thêm S3/DynamoDB), apply tay từ máy dev: EC2
  `c6i.2xlarge` x86 (8 vCPU/16 GiB), Elastic IP, security group (443 chỉ từ static IP Lightsail;
  80 chỉ phục vụ ACME/redirect; 11434 loopback), EBS gp3 50 GiB, IAM instance profile cho SSM và
  đọc secret, Secrets Manager chứa Bearer token. Tag `Project`, `Feature=032`,
  `Environment=aws-course`, `Owner`, `ExpiresOn`; AWS Budget cảnh báo 50/80/100% của 80 USD.
- `.gitignore`: chặn `.terraform/`, `*.tfstate*`, `*.tfvars`; cho phép `.env.ollama.example`
  (hiện `.env.*` đang bị ignore) nhưng không cho phép file env thật.
- `deployments/scripts/aws-env/aws-env-bootstrap.ps1`: thêm các key AI mới vào `$CopyExact`/
  `$EmptyOk`/`$Const` để env trial không bị rơi khỏi file `.env.aws` sinh tự động.
- DNS: apply hai bước — cấp Elastic IP trước, tạo tay A record của subdomain trỏ tới IP đó, rồi
  apply phần còn lại. Terraform chỉ xuất Elastic IP; không dùng Route53 provider, không đụng DNS
  của API.
- Cloud-init: cài Ollama pin version, pull `qwen3:4b` pin digest, chạy Nginx proxy TLS, bind Ollama
  `127.0.0.1:11434`, đặt `OLLAMA_NUM_PARALLEL=2`/`OLLAMA_MAX_QUEUE`, pre-warm model và chứng minh
  thinking đã tắt trước khi đánh dấu trial-ready.
- Env Lightsail chỉ đổi nhóm `*_TEXT_PRIMARY_*=openai`, endpoint/model/token tương ứng,
  `*_TEXT_FALLBACK_*=google`, concurrency và timeout; vision/embedding giữ Gemini. `free_text`
  vẫn nằm trong `AI_UNBILLED_CAPABILITIES` cùng `embedding`; rule miễn phí cho `openai` nằm ở code
  theo provider, không dựa vào danh sách này. Secret không đưa vào repo, manifest hay log workflow.
- Config mới: `ai.text_max_concurrency` (2), `ai.text_primary_timeout_seconds` (default 30, trial
  90), `ai.text_fallback_timeout_seconds` (30), `reasoning_effort` trong các block text primary
  (`AI_OPENAI_REASONING_EFFORT`, trống = không gửi), `ai.openai_trial_expected_hosts` (CSV, trống =
  tắt cảnh báo). Field
  `server.sse.ai_chat_chunk_deadline_seconds` **đã tồn tại** (default 30); thêm env alias
  `AI_CHAT_CHUNK_DEADLINE_SECONDS` để override 90 trong trial. Loader dùng `overrideInt` trong
  `applyEnvOverrides` cho 4 biến int (3 field mới + 1 alias) thay vì dựa vào `os.ExpandEnv`; validate
  `AI_TEXT_PRIMARY_TIMEOUT_SECONDS <= AI_CHAT_CHUNK_DEADLINE_SECONDS`. Cập nhật cả
  `configs/config.production.yaml` và `configs/config.yaml`. Timeout theo provider: primary
  `openai` dùng 90 giây trong trial; primary `google` và vision/embedding giữ timeout hiện hữu của
  từng file config (production: chat 30/recommendation 45/free 30; local: 30/30/30); text client bỏ
  `Timeout` toàn cục và dùng deadline từng attempt.
- Runbook: provision, cập nhật allowlist khi static IP đổi, bật/tắt Ollama, drill rollback về
  Gemini-only, ngày thứ 7 và thu thập số liệu báo cáo.

### 5. Kiểm chứng local trước khi lên EC2

- Cùng `qwen3:4b` pin version/digest chạy bằng Ollama native trên máy dev; GPU dùng để chạy
  contract test và drill fallback trước, sau đó `OLLAMA_NUM_GPU=0` để đo latency CPU tham chiếu.
- Contract test phải chứng minh: endpoint `/v1/chat/completions`, Bearer, non-stream, SSE,
  structured JSON, usage thiếu/0, thinking off, lỗi 4xx/429/5xx và cancellation.
- Trước khi bắt đầu tuần đo trên EC2, benchmark recommendation prompt thực tế; nếu p95 TTFT vượt
  60 giây, giảm `recommendation_prompt_candidate_limit` hoặc ghi nhận điều chỉnh trong báo cáo
  trước khi đo — không hạ SC-005 sau khi đã bắt đầu.
- Nếu local gate thất bại (contract, thinking, structured output hoặc latency không thể đạt), dừng
  trước Phase B, đổi model/prompt/digest và cập nhật spec/plan trước khi apply EC2.
- Cùng phép đo phải kiểm tra TTFT + thời gian sinh output có nằm trong 90 giây không. Nếu vượt,
  phải giảm prompt/`recommendation_max_output_tokens` hoặc chấp nhận recommendation fallback và
  ghi rõ rằng SC-001 (text không fallback do Ollama phục vụ) không đạt — không được im lặng hạ
  ngưỡng.
- Trên EC2, chạy đúng load profile SC-005: 50 request/capability, concurrency 2, cửa sổ 30 phút;
  lưu raw sample để tính p95 và phân biệt success/fallback/failure do write timeout.

## Complexity Tracking

Không có vi phạm Constitution Check cần theo dõi.
