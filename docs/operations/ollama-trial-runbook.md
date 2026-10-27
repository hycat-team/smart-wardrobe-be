# Runbook — Thử nghiệm Ollama trên EC2 (feature 032)

**Feature**: `specs/032-ollama-lightsail-trial/` | **Ngày tạo**: 2026-10-26 | **Cập nhật**: 2026-10-27
(Phase 6 / US4: CloudWatch agent + metric filter/dashboard + quy trình ngày thứ 7 ở mục 6; T043–T049
đã có mã/tài liệu, các bước live vẫn là operator step **NOT RUN**) | **Trạng thái**: Phase 3 (US1),
Phase 4 (US2) và Phase 5 (US3) đã có mã + test/tài liệu; Phase 6 mới có mã telemetry/hạ tầng và quy
trình — **chưa chạy trial 7 ngày**, chưa có quyết định go/no-go. Xem mục 6, mục 8 và mục 9.

Runbook này là quy trình vận hành duy nhất cho tuần thử nghiệm Ollama: đo baseline, chạy local,
dựng EC2, bật/tắt bằng env trên Lightsail, rollback và chốt quyết định ngày thứ 7. Hợp đồng chi
tiết nằm ở [contracts/env-ai-trial.md](../../specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md),
[contracts/trial-telemetry.md](../../specs/032-ollama-lightsail-trial/contracts/trial-telemetry.md)
và [quickstart.md](../../specs/032-ollama-lightsail-trial/quickstart.md).

**Bất biến của thử nghiệm**

- Backend vẫn là sáu container trên box Lightsail hiện hữu; không tạo EKS/Kubernetes/ALB/ACM và
  không đổi DNS của API.
- Không sửa `Dockerfile`, `deployments/docker-compose.aws.yml`,
  `deployments/nginx/templates/aws/**`, `.github/workflows/deploy-aws.yml`.
- Không có lệnh, probe hay thao tác nào nhắm vào VPS; mọi kiểm chứng VPS chỉ bằng diff repository.
- Ollama chỉ bind `127.0.0.1:11434`; proxy TLS + Bearer + allowlist là đường vào duy nhất.
- Uptime **không** phải tiêu chí quyết định. Sau ngày thứ 7 không có rollback/tắt/teardown tự động.

> **Ghi chú môi trường tạo tài liệu (Phase 1, 2026-10-26)**: tại thời điểm tạo khung, Terraform
> và Ollama CLI **chưa được cài** trên máy tạo tài liệu. Mọi lệnh `terraform`, `ollama`, `aws ssm`
> và lệnh trên Lightsail dưới đây là việc của người vận hành, chạy trên máy dev hoặc box tương ứng.
> Không lệnh nào trong số đó được chạy tự động bởi worker. Nếu `ollama` không có trong `PATH`, cài
> Ollama native và pull `qwen2.5:3b-instruct-q4_K_M` là **bước thủ công bắt buộc** trước A1; không có đường tắt tự
> động.

## 0. Điều kiện tiên quyết

- [ ] Go 1.26+, Docker Desktop, Ollama native trên Windows (cài tay; xem A1).
- [ ] AWS credentials có quyền EC2/EIP/EBS/IAM/SSM/Secrets Manager/Budget tại `ap-southeast-1`.
- [ ] Terraform ≥ 1.6 và AWS Session Manager plugin.
- [ ] Quyền tạo một A record cho subdomain Ollama (không phải domain API).
- [ ] Static IP hiện tại của box Lightsail (mục 1).
- [ ] Quyền truy cập Lightsail host và biết quy trình deploy AWS hiện hữu
      (`docs/operations/aws-runbook.md`, `docs/operations/deployment-pipeline.md`).

## 1. Baseline (chốt trước khi thay đổi bất cứ thứ gì)

Điền hết bảng dưới đây **trước** Phase A. Giá trị `TODO(operator)` là bắt buộc và phải do người
vận hành lấy từ nguồn thật; không suy đoán, không bịa số, không dán secret vào file này.

| Hạng mục | Giá trị | Nguồn / cách lấy | Trạng thái |
|---|---|---|---|
| Commit đang chạy trên Lightsail | `TODO(operator)` | Chạy `git rev-parse --short HEAD` trên máy dev (read-only) tại thời điểm chốt baseline; ghi cả nhánh `032-ollama-lightsail-trial` nếu khác `main` | Chưa điền |
| Ngày bắt đầu tuần thử nghiệm (`trial_started_at`) | `TODO(operator)` | Ngày bật env trial trên Lightsail (UTC) | Chưa điền |
| Ngày hết hạn tag (`ExpiresOn`) | `TODO(operator)` | `trial_started_at + 7 ngày`; dùng cho tag Terraform. Dự kiến `2026-11-02` nếu bắt đầu 2026-10-26 | Chưa điền |
| Trạng thái 6 container Lightsail | `TODO(operator)` | `docker compose ps` trên box: 3 container `http` + 3 container `worker`, tất cả healthy/up; dán output (không kèm env) vào evidence index | Chưa điền |
| Static IP của box Lightsail | `TODO(operator)` | Lightsail Console → Instances → Networking → Static IP. Đây là giá trị cho `PROXY_ALLOWED_CIDR` và security group (`<ip>/32`) | Chưa điền |
| Gemini text model đang dùng | `TODO(operator)` | Đọc `/opt/closy-aws/.env.aws`: ghi **tên model** và **endpoint** của `*_TEXT_FALLBACK_*` hiện hữu | Chưa điền |
| Tham chiếu key Gemini text (không ghi giá trị) | `TODO(operator)` | Chỉ ghi: "`<CAP>_TEXT_FALLBACK_API_KEY` trong `/opt/closy-aws/.env.aws` (managed secret, không copy vào repo)". **Không** dán key | Chưa điền |
| AWS account / region | `TODO(operator)` | Account dùng cho EC2 trial; region cố định `ap-southeast-1`. Không đưa account ID vào Terraform hay tài liệu nếu không cần | Chưa điền |
| Ollama version local (A1) | `0.40.11` | `ollama --version` ngày 2026-10-27 | Đã điền |
| Digest `qwen2.5:3b-instruct-q4_K_M` local (A1) | `sha256:5ee4f07cdb9beadbbb293e85803c569b01bd37ed059d2715faa7bb405f31caa6` (ID `357c53fb659c`) | `ollama list` / `ollama show --modelfile` ngày 2026-10-27 | Đã điền |
| Xác nhận không truy cập VPS | `TODO(operator)` | Tick sau khi rà lại phiên làm việc: không SSH, không deploy, không đọc secret, không đổi DNS VPS | Chưa xác nhận |

**Nguyên tắc baseline**

- Không chạy lệnh nào lên VPS để "kiểm tra baseline". Baseline của VPS là diff repository (mục 5).
- Không đưa key/token vào bảng này; chỉ ghi tham chiếu tới nơi lưu.
- Sáu container phải healthy **trước** khi bật trial; nếu baseline đã lỗi, dừng và xử lý ngoài phạm
  vi feature này.

## 2. Phase A — Kiểm chứng local (bắt buộc trước EC2)

### A1. Cài và pin model (thủ công)

```powershell
ollama --version
ollama pull qwen2.5:3b-instruct-q4_K_M
ollama list
ollama show qwen2.5:3b-instruct-q4_K_M --modelfile
```

- Nếu `ollama` không có trong `PATH`: cài Ollama native cho Windows trước, rồi chạy lại. Đây là
  **bước thủ công**; runbook không tự cài.
- Ghi `ollama --version` và digest `sha256:...` vào mục 1.
- Version + digest phải khớp `OLLAMA_VERSION`/`OLLAMA_MODEL_DIGEST` của Terraform trước khi lên EC2.
- **Đạt khi**: version và digest đã ghi, model load được.

### A2. Contract test adapter

```powershell
go test ./shared/infrastructure/ai/... -count=1
go test ./shared/infrastructure/ai/openai/... -count=1
```

**Đạt khi**: toàn bộ contract test của
[contracts/ollama-openai-subset.md](../../specs/032-ollama-lightsail-trial/contracts/ollama-openai-subset.md)
§5 pass (không reasoning, structured output, SSE, usage thiếu/0, lỗi 4xx/429/5xx, cancellation).

### A3. Probe thủ công

Non-stream:

```powershell
$body = @{
  model = "qwen2.5:3b-instruct-q4_K_M"
  messages = @(
    @{ role = "system"; content = "Trả lời ngắn gọn." },
    @{ role = "user"; content = "Xin chào" }
  )
  stream = $false
  max_tokens = 50
  temperature = 0
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:11434/v1/chat/completions" `
  -ContentType "application/json" -Body $body
```

Streaming và structured output: cùng body với `stream = $true`, và thêm
`response_format = @{ type = "json_object" }`.

**Đạt khi**:

- Không có `reasoning_content`/thinking trong response.
- JSON structured parse được và đúng schema.
- SSE có `data: [DONE]`; usage (nếu có) được đọc đúng.
- `thinking_tokens = 0` theo telemetry.

### A4. Drill fallback local

Dùng compose local hiện hữu cho PostgreSQL/Redis/RabbitMQ; **không** sửa compose AWS.

1. Chạy AI service với primary `openai` trỏ `http://127.0.0.1:11434/v1/chat/completions`, fallback
   Gemini; đặt `AI_TEXT_PRIMARY_TIMEOUT_SECONDS=90`, `AI_TEXT_FALLBACK_TIMEOUT_SECONDS=30`,
   `AI_CHAT_CHUNK_DEADLINE_SECONDS=90`.
2. Gọi một request text → provider thắng phải là `openai`.
3. Dừng Ollama hoặc trỏ primary endpoint sang một port chết (ví dụ `http://127.0.0.1:11435/v1/chat/completions`);
   nếu Ollama desktop tự khởi động lại, dùng port chết làm transport failure có kiểm soát.
4. Gọi lại → fallback Gemini đúng một lần, không treo.
5. Bật Ollama lại → request trở về Ollama, không cần đổi cấu hình.

**Đạt khi**: đúng 1 attempt Ollama + tối đa 1 attempt Gemini; usage/cost gắn provider thắng; client
cancel không fallback; lỗi sau chunk đầu không fallback mà trả lỗi trên stream.

### A5. Đo latency CPU tham chiếu

```powershell
$env:OLLAMA_NUM_GPU = "0"
# Khởi động lại Ollama, sau đó chạy kiểm tra chức năng + lấy mẫu:
.\deployments\ollama\local-test.ps1 -Capability chat_text -Samples 50 -Concurrency 2
.\deployments\ollama\local-test.ps1 -Capability free_text -Samples 50 -Concurrency 2 -SkipFunctional
.\deployments\ollama\local-test.ps1 -Capability recommendation_text -Samples 50 -Concurrency 2 -SkipFunctional `
  -Prompt "<prompt recommendation thực tế>"
```

`local-test.ps1` chạy kiểm tra chức năng (non-stream, streaming, structured output, không reasoning)
rồi lấy mẫu latency ghi ra `docs/temp/ollama-trial/local-latency-<capability>.csv` và
`local-test-summary-<capability>.json` (thư mục này đã bị gitignore). Xem tham số bằng
`Get-Help .\deployments\ollama\local-test.ps1 -Full`. Kịch bản đo tham chiếu theo SC-005: 50 request
cho mỗi capability `chat_text`, `free_text`, `recommendation_text`, concurrency 2, cửa sổ 30 phút;
thêm 5 request `summary` và 5 request `rewriter` để xác nhận route (`free_text` khi route free,
`chat_text` khi route paid). Ghi TTFT, latency, output tokens, outcome và fallback reason; lưu raw
sample để tính p95. Script chỉ lấy mẫu request trực tiếp tới Ollama (không qua Go service), nên
fallback reason và provider thắng phải lấy thêm từ log AI service ở A4/B6.

**Đạt khi**: có số liệu tham chiếu và raw sample. **KHÔNG** dùng số GPU local thay SC-005; nếu CPU
local không đạt, phải điều chỉnh prompt/model/digest **trước** khi lên EC2.

### A5.1 Kết quả local ngày 2026-10-27 (bằng chứng thật trên máy dev)

| Hạng mục | Kết quả | Ghi chú |
|---|---|---|
| Ollama | `0.40.11` | Cài bằng winget, server native Windows |
| Model | `qwen2.5:3b-instruct-q4_K_M` | Pin digest `sha256:5ee4f07...31caa6`, ID `357c53fb659c` |
| A3 functional GPU | **PASS** | non-stream 5.7s (cold) → `Hello!`; SSE TTFT 81ms/total 177ms; JSON pass; không reasoning |
| A4 CPU-only (Docker) | **PASS** | Container `ollama-cpu` port 11435, `/api/ps size_vram=0`; p50/p95: chat 2547/2937ms, free 2320/2560ms, recommendation 5917/6321ms (50 mẫu/capability, concurrency 2) |
| A5 text qua AI service | **PASS** | `chat`, `outfit`, `summary`, `rewriter` → `provider=openai`, `billing_class=unbilled`, `actual_cost=0` |
| A6 vision/embedding | **PASS** | `provider=google`; vision billed (`actual_cost=8531875` microVND), embedding unbilled |
| A6b gRPC streaming | **PASS** | Success + `[DONE]`; fail-before → fallback Gemini; fail-after → chunk rồi lỗi, không fallback. Bug stream thiếu `[DONE]` đã sửa (`fixup! b9530ef4`) |
| A7 fallback | **PASS** | Primary trỏ port chết/stub → attempt 1 lỗi, attempt 2 Gemini `gemini-3.1-flash-lite` success; user paid route ghi `actual_cost=144375` microVND |
| A8 restore | **PASS** | Trỏ lại `11435` (container CPU) → `provider=openai`, cost 0 |
| L2 json_schema | **PASS** | Ollama trả JSON đúng schema `outfit`; chất lượng tiếng Việt của 3B còn yếu, cần prompt thật khi đánh giá |
| L3 failure matrix | **PASS** | 429/500/malformed/schema_invalid/timeout → Gemini fallback đúng 1 lần; 400 → 0 fallback; cancel → 0 fallback |
| L5 concurrency | **PASS** | 5 request đồng thời: stub log cho thấy 2 → 2 → 1; elapsed 15.8s (semaphore 2 hoạt động) |
| L5 soak | **PASS** | 100 request summary, 2 luồng × 50, concurrency 2: ok=100, fail=0, 145s |

Ghi chú local:
- CPU-only đo bằng container `ollama/ollama:0.40.11` ở port `11435`; host Ollama desktop vẫn giữ GPU nên
  không dùng app desktop để đo CPU.
- Fallback model mặc định `gemma-4-31b-it`/`gemma-4-26b-a4b-it` quá chậm hoặc sinh thinking, vượt timeout
  30s. Local drill dùng `*_FALLBACK_MODEL=gemini-3.1-flash-lite`; khi lên Lightsail phải chọn fallback
  Gemini nhanh tương tự.
- Streaming bug: OpenAI adapter từng coi EOF không có `[DONE]` là success; đã sửa thành `malformed` và
  thêm test (stream fail-before vẫn fallback, fail-after trả lỗi).
- `version` trong attempt log rỗng vì chưa có `APP_VERSION`; đây là deferred D-04, không chặn local.
- `L2` chất lượng câu trả lời 3B chưa tốt với prompt chung; trước khi chốt go/no-go cần chạy prompt thật
  của outfit/rewriter và ghi nhận.

### A5.2 Kiểm chứng API cũ bằng Swagger/HTTP với Ollama local (2026-10-27)

Dựng full stack local (`account`, `commerce`, `ai`, Postgres/Redis/RabbitMQ) và cho container Ollama
`ollama-in-ai` chạy trong network namespace của `ai-closy-dev`; AI service gọi
`http://127.0.0.1:11434/v1/chat/completions`. Nginx gốc của local và AWS đã tăng
`/api/v1/ai/outfit-recommendations` lên 300s (`proxy_send_timeout`/`proxy_read_timeout`); VPS không đổi.

| API cũ | Kết quả | Ghi chú |
|---|---|---|
| `POST /api/v1/ai/outfit-recommendations` | **PASS** | 200, `isFallback=false`, attempt `provider=openai`, `model=qwen2.5:3b-instruct-q4_K_M`, `actual_cost=0`; request ấm ~19.7s, request chậm 83.7s vẫn 200 |
| Outfit AI semantic validation | **PASS** | Trước fix: provider success nhưng commerce fallback `invalid_outfit_structure`. Sau fix dynamic `CORE_STRATEGY`/`ROLE_ALIASES` + UUID-first alias: AI path thật, `isFallback=false`; `includeBrandItems=false` ~75.9s, `true` ~52.1s |
| Nginx outfit timeout | **PASS** | Local/AWS route có `proxy_send_timeout 300s` + `proxy_read_timeout 300s`; request AI 82.3s không bị 504. VPS giữ 60s |
| `POST /api/v1/ai/chat/sessions/{id}/messages/stream` (câu hỏi thường) | **PASS** | SSE trả `1+1 bằng 2.` |
| Chat fashion/greeting trước khi sửa prompt | **FAIL** | Qwen2.5 chỉ trả `ACTION:REDIRECT_OUTFIT`; commerce trả `"Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này."` |
| Chat fashion/greeting sau khi sửa prompt | **PASS** | Model trả lời tiếng Việt trước; greeting/fashion đều có nội dung, không còn lỗi fullResponse rỗng |
| Fallback qua API cũ | **PASS** | Stop `ollama-in-ai` → attempt 1 `transport`, attempt 2 `provider=google/gemini-3.1-flash-lite` success, 200 sau ~2.9s, `isFallback=false` |
| Uppercase JSON schema | **FAIL trước fix** | Ollama trả 400; `normalizeOpenAISchemaTypes` hạ type về lowercase, test pass; fixup `fixup! b9530ef4` |
| Vision/embedding qua API cũ | **Chưa retest** | Đã xác nhận qua gRPC ở A6: vẫn `provider=google` |
| Admin AI usage | **Chưa test** | Không có credential admin local (`admin`/`123456` không đúng); không chặn luồng user |

Ghi chú:
- User test `user` có 20 món trong tủ nên prompt chat yêu cầu trả lời trực tiếp, không chèn action token.
- Sau vài lần gọi outfit, quota ngày của user hết → API trả 400 `"Bạn đã dùng hết lượt tạo trang phục bằng AI trong hôm nay."`; đây là quota, không phải lỗi AI.
- Prompt chat được chỉnh để **luôn trả lời trước**, chỉ chèn `[ACTION:REDIRECT_OUTFIT]` khi cần và không bao giờ chỉ trả token.
- Container `nginx-local-test` chỉ là workaround tạm; sau khi tăng timeout trong config gốc, đã quay lại `nginx-closy-dev`.
- Fallback outfit trước đây không phải lỗi Ollama: provider trả JSON thành công nhưng prompt example luôn dạy `top + bottom` trong khi pool chỉ có `fullbody`, khiến commerce từ chối cấu trúc và chạy HSL. Prompt nay phát `CORE_STRATEGY` + `ROLE_ALIASES` và example theo đúng pool.
- Alias resolver cũng được sửa để ưu tiên UUID trước alias, tránh UUID bắt đầu bằng `a` + chữ số (ví dụ brand item `a5e506ed-...`) bị hiểu nhầm thành alias `A5`.

### A6. Cổng ra Phase A

Chỉ apply Terraform khi tất cả đạt:

- [x] Contract test pass.
- [x] Không có reasoning content/token được chứng minh.
- [x] Structured output hợp lệ.
- [x] Drill fallback pass.
- [x] SSE chat không bị cắt: `AI_CHAT_CHUNK_DEADLINE_SECONDS=90` và primary timeout 90 đã đặt.
- [x] Recommendation: TTFT + thời gian sinh output nằm trong 90 giây, hoặc đã ghi nhận điều chỉnh/
      fallback trước khi đo.
- [x] Digest/version đã ghi.

Nếu bất kỳ mục nào không đạt: dừng, không apply EC2; đổi model/prompt/digest và cập nhật
`spec.md`/`plan.md` trước khi thử lại.

## 3. Phase B — EC2 và Lightsail

> Trước khi apply: xác nhận lại static IP Lightsail (mục 1) vẫn đúng. `terraform state` là **local**
> và nằm ngoài repo (`.gitignore` đã chặn `.terraform/`, `*.tfstate*`, `*.tfvars`). Không commit
> state, không dùng backend S3/DynamoDB.

### B1. Cấp Elastic IP và tạo DNS (apply hai bước)

```powershell
terraform -chdir=deployments/terraform/ollama-ec2 init
terraform -chdir=deployments/terraform/ollama-ec2 apply -target=aws_eip.ollama
terraform -chdir=deployments/terraform/ollama-ec2 output elastic_ip
```

Tạo **tay** A record `ollama.<domain>` trỏ tới IP vừa xuất (không phải domain API). Sau đó:

```powershell
terraform -chdir=deployments/terraform/ollama-ec2 apply
terraform -chdir=deployments/terraform/ollama-ec2 output
```

**Đạt khi**: apply thành công; output có `instance_id`, `elastic_ip`, `proxy_hostname`, `secret_arn`,
`ollama_version`, `model_digest`, `security_group_id`; không đụng DNS của API.

### B2. Kiểm tra EC2 qua SSM (không mở SSH)

```powershell
aws ssm start-session --target <instance_id>
```

Trong phiên SSM kiểm tra:

- `systemctl status ollama nginx`
- `curl -s localhost:11434/api/version`
- `ollama list` khớp digest đã pin
- `ss -lntp` cho thấy Ollama chỉ `127.0.0.1:11434`
- `certbot certificates` còn hạn
- security group chỉ mở 443 cho `<lightsail-ip>/32`; 80 chỉ ACME/redirect; không mở 11434

**Đạt khi**: proxy TLS sẵn sàng, model đúng digest, thinking off, Ollama không public.

### B3. Bật trial trên Lightsail

Cập nhật `/opt/closy-aws/.env.aws` theo
[contracts/env-ai-trial.md](../../specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md) §1,
thêm các key mới vào allowlist copy của `deployments/scripts/aws-env/aws-env-bootstrap.ps1` nếu
script này sinh file, rồi chạy quy trình deploy AWS hiện hữu. **Không** sửa file deployment.

**Đạt khi**: chỉ container AI được tạo lại; sáu container vẫn healthy; không có diff Dockerfile/
Compose/Nginx/pipeline.

### B4. Smoke test

- Gọi `chat_text`, `recommendation_text`, `free_text` qua API thật → provider thắng là `openai`.
- Gọi `summary` và `rewriter` → provider thắng cũng là `openai` (route free dùng `free_text`, route
  paid dùng `chat_text`; trial đặt cả hai nhóm primary = `openai`).
- Gọi vision và embedding → provider thắng vẫn là `google`.
- Log có `provider`, `model`, `attempt_index`, `billing_class`, `cost_micro_vnd`.
- `ai_usage_events`: ba đường text có `actual_cost_micro_vnd = 0`.

> Lệnh + output kỳ vọng chi tiết ở §4.1.3/§4.1.4. Lưu ý: `version` trong dòng attempt lấy từ
> `APP_VERSION` (rỗng vẫn hợp lệ); `cost_micro_vnd` vẫn là `0` **theo thiết kế** vì attempt result
> không mang cost — cost chỉ đọc từ `ai_usage_events`; vision/embedding không có dòng attempt nào.

### B5. Kiểm tra cô lập truy cập

Từ máy ngoài allowlist:

```powershell
curl.exe -v https://<ollama-host>/v1/chat/completions
curl.exe -v http://<elastic-ip>:11434/api/version
```

**Đạt khi**: request ngoài allowlist bị chặn/timeout; `11434` không truy cập được; thiếu/sai Bearer
trả 401/403; request hợp lệ từ Lightsail trả 200. Ghi bằng chứng cho SC-003.

### B6. Drill fallback và tải

- Stop Ollama qua SSM → gọi text → Gemini phục vụ đúng một lần → restart Ollama.
- Chạy đúng load profile SC-005: 50 request/capability, concurrency 2, cửa sổ 30 phút; lưu raw
  sample, tính p95 và phân loại success/fallback/failure do write timeout.
- Xác nhận không có attempt thứ ba và không có Gemini cost cho lượt Ollama thắng.

**Đạt khi**: SC-005 và SC-009 có số liệu; semaphore/Ollama queue hoạt động.

### B7. Drill rollback Gemini-only

Đổi ba `*_TEXT_PRIMARY_*` về Gemini như §1.5 của env contract rồi chạy lại deploy hiện hữu. Quy
trình chi tiết (từng lệnh + output kỳ vọng) ở §4.2; kịch bản drill có bấm giờ và bảng bằng chứng
SC-006 ở §9.3. Regression API phục vụ SC-008 ở §9.4.

**Đạt khi**: trong ≤15 phút, text phục vụ bằng Gemini; sáu container healthy; không mất dữ liệu đã
xác nhận; không thao tác VPS. Ghi bằng chứng cho SC-006.

## 4. Chuyển cấu hình (bật/tắt), rollback và cập nhật allowlist

> Bật/tắt thử nghiệm **chỉ** bằng env. Không sửa Dockerfile, `deployments/docker-compose.aws.yml`,
> `deployments/nginx/templates/aws/**`, `.github/workflows/deploy-aws.yml`, không đổi DNS của API và
> không có lệnh nào nhắm vào VPS. Nếu một bước nào đó đòi sửa các file trên thì **dừng lại** — đó
> không phải bước của thử nghiệm này.
>
> Giá trị trial do **người vận hành** đặt vào `/opt/closy-aws/.env.aws`. Script sinh env
> (`deployments/scripts/aws-env/aws-env-bootstrap.ps1`) chỉ bảo đảm các key AI mới **không bị rơi
> khỏi** file `.env.aws` sinh tự động; nó **không** tự bật trial và không chứa secret.
>
> Toàn bộ mục 4 **chưa được chạy** trong môi trường implement (không có quyền truy cập Lightsail).
> Mọi output dưới đây là **kỳ vọng**, không phải kết quả đã đo.

### 4.1 Bật trial (Gemini-only → Ollama primary)

#### 4.1.1 Sáu nhóm biến môi trường

Nguồn: [contracts/env-ai-trial.md](../../specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md)
§1.1–§1.4 và `plan.md` §4 ("Env Lightsail"). Tổng cộng 47 key AI liên quan: 24 key trong ba nhóm
text (8 key/nhóm), 16 key `VISION_*`/`EMBEDDING_*`, và 7 key điều khiển trial.

| # | Nhóm | Key (đủ) | Giá trị trial |
|---|---|---|---|
| 1 | `CHAT_TEXT` primary | `CHAT_TEXT_PRIMARY_PROVIDER`, `..._API_KEY`, `..._ENDPOINT`, `..._MODEL` | `openai` / `<bearer-token>` / `https://<ollama-host>/v1/chat/completions` / `qwen2.5:3b-instruct-q4_K_M` |
| 2 | `RECOMMENDATION_TEXT` primary | `RECOMMENDATION_TEXT_PRIMARY_PROVIDER`, `..._API_KEY`, `..._ENDPOINT`, `..._MODEL` | như nhóm 1 (cùng endpoint/token/model) |
| 3 | `FREE_TEXT` primary | `FREE_TEXT_PRIMARY_PROVIDER`, `..._API_KEY`, `..._ENDPOINT`, `..._MODEL` | như nhóm 1 (cùng endpoint/token/model) |
| 4 | Ba nhóm fallback | `{CHAT,RECOMMENDATION,FREE}_TEXT_FALLBACK_PROVIDER`, `..._API_KEY`, `..._ENDPOINT`, `..._MODEL` | `google` + **đúng** key/endpoint/model Gemini đang chạy (lấy từ block `*_TEXT_FALLBACK_*` hiện hữu, không đổi giá trị) |
| 5 | Vision + embedding | `VISION_{PRIMARY,FALLBACK}_{PROVIDER,API_KEY,ENDPOINT,MODEL}`, `EMBEDDING_{PRIMARY,FALLBACK}_{PROVIDER,API_KEY,ENDPOINT,MODEL}` | **Không đổi** — giữ nguyên provider/endpoint/model hiện hữu (Gemini). Trial **không** được trỏ vision/embedding sang Ollama |
| 6 | Điều khiển trial | `AI_UNBILLED_CAPABILITIES`, `AI_TEXT_MAX_CONCURRENCY`, `AI_TEXT_PRIMARY_TIMEOUT_SECONDS`, `AI_TEXT_FALLBACK_TIMEOUT_SECONDS`, `AI_OPENAI_REASONING_EFFORT`, `AI_CHAT_CHUNK_DEADLINE_SECONDS`, `AI_OPENAI_TRIAL_EXPECTED_HOSTS` | xem bảng dưới |

Chi tiết nhóm 6 (giá trị là hợp đồng, không phải gợi ý):

| Key | Giá trị trial | Ràng buộc kiểm tra lúc khởi động |
|---|---|---|
| `AI_UNBILLED_CAPABILITIES` | `free_text,embedding` | Giữ nguyên; danh sách này **chỉ** phân loại capability. Miễn phí `openai` là rule **theo provider** trong code, **không** dùng danh sách này |
| `AI_TEXT_MAX_CONCURRENCY` | `2` | `>= 1` (default code 2) |
| `AI_TEXT_PRIMARY_TIMEOUT_SECONDS` | `90` | `> 0`; `+ AI_TEXT_FALLBACK_TIMEOUT_SECONDS <= 120`; `<= AI_CHAT_CHUNK_DEADLINE_SECONDS` |
| `AI_TEXT_FALLBACK_TIMEOUT_SECONDS` | `30` | `> 0` |
| `AI_OPENAI_REASONING_EFFORT` | `(trống)` | chỉ `none`/`low`/`medium`/`high` hoặc trống (trống = không gửi field) |
| `AI_CHAT_CHUNK_DEADLINE_SECONDS` | `90` | `> 0`; override field có sẵn `server.sse.ai_chat_chunk_deadline_seconds` (default 30 ngoài trial) |
| `AI_OPENAI_TRIAL_EXPECTED_HOSTS` | `<ollama-host>` | CSV **host** (không scheme/path); trống = tắt cảnh báo "openai trỏ sai host" |

Ghi chú hợp đồng:

- `ENDPOINT` của `openai` phải là URL **đầy đủ** tới `/v1/chat/completions` (adapter POST nguyên
  chuỗi, không tự nối path) và phải là `https`. Ngoại lệ duy nhất: drill local
  `http://127.0.0.1:<port>/v1/chat/completions` — path vẫn bắt buộc.
- `MODEL` phải đúng `qwen2.5:3b-instruct-q4_K_M` và khớp digest đã pin trên EC2 (mục 1, B2).
- `API_KEY` của ba nhóm primary phải **trùng** Bearer token của proxy (Secrets Manager trên EC2).
  Không log, không dán vào tài liệu.
- `summary`/`rewriter` không có nhóm env riêng: route free dùng nhóm `free_text`, route paid dùng
  nhóm `chat_text`; trial đặt cả hai nhóm `openai` nên cả hai đi Ollama.
- Vision/embedding **không** được đổi. Nếu vô tình trỏ sang Ollama, đó là lỗi cấu hình: dừng và
  hoàn nguyên ngay.

#### 4.1.2 Các bước bật (người vận hành chạy trên máy dev + box Lightsail)

1. **Chốt baseline** (mục 1) và lưu bản sao env chỉ để so sánh, **không** đưa vào repo:

   ```bash
   # trên box Lightsail
   cd /opt/closy-aws
   sudo cp -a .env.aws .env.aws.bak-$(date -u +%Y%m%dT%H%M%SZ)
   ```

2. **Sửa tại chỗ** `/opt/closy-aws/.env.aws` theo bảng §4.1.1 (dùng `vi`/`nano`; `chmod 600`).
   Không thêm block trùng key ở cuối file — giá trị trùng key làm kết quả phụ thuộc thứ tự và
   không kiểm chứng được. Chỉ sửa **giá trị**, không đổi tên key.

3. **Kiểm tra env trước khi deploy** (che key):

   ```bash
   cd /opt/closy-aws
   grep -E '^(CHAT_TEXT|RECOMMENDATION_TEXT|FREE_TEXT|VISION|EMBEDDING)_[A-Z_]+=' .env.aws \
     | sed -E 's/(_API_KEY=).*/\1<redacted>/'
   grep -E '^(AI_UNBILLED_CAPABILITIES|AI_TEXT_|AI_OPENAI_|AI_CHAT_CHUNK_)' .env.aws
   ```

   **Kỳ vọng**: 40 dòng nhóm text/vision/embedding (24 + 16) với đúng provider/endpoint/model như
   bảng; `VISION_*`/`EMBEDDING_*` giống bản backup; 7 dòng nhóm 6 đúng giá trị trial.

4. **Chạy quy trình deploy AWS hiện hữu** (`docs/operations/deployment-pipeline.md`):

   ```powershell
   gh workflow run deploy-aws.yml -R "hycat-team/smart-wardrobe-be" --ref main -f services=ai
   gh run list --workflow=deploy-aws.yml --limit 1
   gh run watch <run-id> -R "hycat-team/smart-wardrobe-be"
   ```

   **Kỳ vọng**: job `plan` → `build-push` → `deploy` → `smoke-all` xanh; chỉ `ai-http`/`ai-worker`
   được tạo lại; `nginx` được recreate một lần sau khi mọi service deploy xong; smoke-all xanh cho
   cả ba service.

   Đường thay thế **chỉ khi** không có thay đổi image và chỉ đổi env (đã có sẵn
   `deployments/` + `.env.deploy` trên box, xem block 3/3 của
   `deployments/scripts/aws-env/aws-env-to-copyblock.ps1`):

   ```bash
   ssh -i "<key>" ubuntu@<box-ip> "cd /opt/closy-aws && docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml up -d --force-recreate ai-http ai-worker"
   ```

   Ghi lại đường đã dùng (pipeline hay recreate) và thời lượng vào evidence T039/§9.1.

5. **Xác minh** theo §4.1.3 và §4.1.4. Nếu bất kỳ mục nào lệch kỳ vọng: dừng, rollback theo §4.2,
   không "chỉnh tiếp cho vừa".

#### 4.1.3 Xác minh container + env đã nạp

```bash
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml ps
# Container AI là distroless (không có shell), nên đọc env từ docker inspect, không `exec sh`.
docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' \
  "$(docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml ps -q ai-http)" \
  | grep -E '^(AI_TEXT_|AI_OPENAI_|AI_CHAT_CHUNK_|AI_UNBILLED_)' | sort
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  logs --since 5m ai-http | grep -iE 'must not exceed|must be at least 1|is invalid' || echo "no config validation error"
```

**Kỳ vọng**:

- `ps`: 7 dòng — `account-http`, `account-worker`, `commerce-http`, `commerce-worker`, `ai-http`,
  `ai-worker`, `nginx` — tất cả `Up`/`running` (container có healthcheck của image phải `healthy`).
  Không có container nào `Restarting`.
- Lệnh `docker inspect`: đúng 7 dòng, không thừa không thiếu:

  ```text
  AI_CHAT_CHUNK_DEADLINE_SECONDS=90
  AI_OPENAI_REASONING_EFFORT=
  AI_OPENAI_TRIAL_EXPECTED_HOSTS=<ollama-host>
  AI_TEXT_FALLBACK_TIMEOUT_SECONDS=30
  AI_TEXT_MAX_CONCURRENCY=2
  AI_TEXT_PRIMARY_TIMEOUT_SECONDS=90
  AI_UNBILLED_CAPABILITIES=free_text,embedding
  ```

- Lệnh `logs | grep ...`: chỉ in `no config validation error`. Nếu thấy
  `ai.text_primary_timeout_seconds must not exceed ...` thì env sai (thường do
  `AI_CHAT_CHUNK_DEADLINE_SECONDS` chưa phải 90) → sửa env rồi lặp lại bước 3–5.

#### 4.1.4 Xác minh provider thắng (B4) + cost

Gọi thật ba capability text, `summary`, `rewriter`, vision, embedding qua API hiện hữu (dùng tài
khoản test; không đổi endpoint/payload/DTO). Sau đó:

```bash
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  logs --since 15m ai-http | grep '"msg":"ai provider attempt"' \
  | jq -r '[.operation,.capability,.attempt_index,.provider,.model,.billing_class,.outcome]|@tsv' \
  | sort | uniq -c
```

> Các lệnh trong mục này dùng `jq`. Box Ubuntu mặc định **chưa** có `jq`; cài bằng
> `sudo apt-get install -y jq` (hoặc lọc JSON bằng `grep`/`python3`). Ghi lại nếu phải cài thêm.

**Kỳ vọng** (mỗi dòng là một tổ hợp operation/capability/attempt):

| operation | capability | attempt_index | provider | model | billing_class | outcome |
|---|---|---|---|---|---|---|
| `chat` | `chat_text` | 1 | `openai` | `qwen2.5:3b-instruct-q4_K_M` | `unbilled` | `success` |
| `outfit` | `recommendation_text` | 1 | `openai` | `qwen2.5:3b-instruct-q4_K_M` | `unbilled` | `success` |
| `summary` | `free_text` (route free) hoặc `chat_text` (route paid) | 1 | `openai` | `qwen2.5:3b-instruct-q4_K_M` | `unbilled` | `success` |
| `rewriter` | `free_text` hoặc `chat_text` | 1 | `openai` | `qwen2.5:3b-instruct-q4_K_M` | `unbilled` | `success` |

**Vision và embedding KHÔNG có dòng `ai provider attempt`**: hook attempt chỉ được gắn cho request
text trial (`attachTrialHooks` chỉ chạy khi `plan.Provider == openai`). Bằng chứng cho
vision/embedding là `ai_usage_events` ở query dưới (provider/model do code stamp) cộng log provider
hiện hữu của service — đừng kết luận "thiếu log = fail".

Bất kỳ dòng `provider=openai` + `billing_class=billed`, hoặc `provider=google` cho capability text
(khi Ollama không lỗi), hoặc `thinking_tokens != 0` trên `openai` đều là **fail**: dừng, ghi vào
evidence, không tiếp tục load profile. `version` lấy từ `APP_VERSION` (rỗng vẫn hợp lệ);
`cost_micro_vnd` vẫn là `0` theo thiết kế vì attempt result không mang cost — **không** dùng nó làm
tiêu chí và không đọc cost từ log, chỉ đọc cost từ `ai_usage_events`.

Đối chiếu usage event (Supabase; chỉ đọc, không sửa dữ liệu):

```sql
SELECT provider, operation, count(*) AS events,
       max(actual_cost_micro_vnd) AS max_actual_cost,
       max(reserved_cost_micro_vnd) AS max_reserved,
       max(thinking_tokens) AS max_thinking
FROM ai_usage_events
WHERE created_at > now() - interval '30 minutes'
GROUP BY provider, operation
ORDER BY provider, operation;
```

**Kỳ vọng**: mọi dòng `provider = openai` có `max_actual_cost = 0`, `max_reserved = 0`,
`max_thinking = 0`; `provider = google` chỉ xuất hiện cho `vision`/`embedding` (và cho attempt
fallback nếu có), với cost theo pricing hiện hữu. Không có hai event cho cùng một `request_id`
(`request_id` là UNIQUE).

### 4.2 Rollback về Gemini-only (mục tiêu ≤15 phút, SC-006)

Hợp đồng: [contracts/env-ai-trial.md](../../specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md)
§1.5. Không cần đổi Compose/Nginx/Dockerfile; chỉ env + recreate container AI.

1. **T0**: ghi giờ bắt đầu (UTC, `date -u +%Y-%m-%dT%H:%M:%SZ`).
2. Đổi ba `*_TEXT_PRIMARY_*` về cấu hình Gemini hiện hữu: provider `google`, và key/endpoint/model
   **bằng đúng** block `*_TEXT_FALLBACK_*` tương ứng (copy từ chính file env, không gõ lại tay).
3. **Giữ nguyên** `AI_UNBILLED_CAPABILITIES`, `VISION_*`, `EMBEDDING_*`, `AI_TEXT_*`,
   `AI_OPENAI_*`, `AI_CHAT_CHUNK_DEADLINE_SECONDS`. Không cần hạ timeout: khi primary là `google`,
   code dùng timeout hiện hữu của từng capability (production chat 30 / recommendation 45 / free 30),
   không áp 90 giây của nhánh `openai`. Điều kiện validate `90 <= 90` vẫn đúng.
4. Chạy lại deploy hiện hữu (`gh workflow run deploy-aws.yml ... -f services=ai`), hoặc đường
   recreate env-only ở §4.1.2 bước 4 nếu không có thay đổi image.
5. **Xác minh** (cùng lệnh §4.1.3 + §4.1.4):
   - `ps`: 7 container `Up`, không `Restarting`.
   - log: 100% `operation` text (`chat`/`outfit`/`summary`/`rewriter`) có `provider=google`,
     `billing_class=billed`; **không** còn dòng nào `provider=openai` sau T1.
   - usage event: không còn event `openai` mới; Gemini billed vẫn có reserved + actual.
   - `.env.aws`: `grep -E '^[A-Z]+_TEXT_PRIMARY_PROVIDER=' .env.aws` → cả ba dòng `google`.
6. **T1**: ghi giờ kết thúc; `T1 - T0 <= 15 phút` là điều kiện SC-006. Ghi cả đường đã dùng
   (pipeline/recreate) vì thời gian build ảnh không thuộc phần rollback env.
7. Nếu trial tiếp tục sau drill: bật lại theo §4.1 và ghi thời gian bật lại **riêng** (không tính
   vào SC-006).

### 4.3 Cập nhật allowlist khi static IP Lightsail đổi

Dấu hiệu: request từ Lightsail bị chặn/timeout trong khi EC2 và proxy vẫn sống; kiểm tra log Nginx
trên EC2 thấy IP nguồn lạ, hoặc Lightsail Console báo static IP mới.

Allowlist tồn tại ở **hai** lớp và phải khớp nhau: security group (lớp ngoài, chặn gói tin) và
`allow`/`deny all` trong Nginx (lớp trong, trả 403). Cập nhật một lớp mà quên lớp kia sẽ ra trạng
thái "chặn nhưng không rõ vì sao", hoặc tệ hơn là mở rộng quá mức.

1. Lấy static IP mới từ Lightsail Console → Instances → Networking → Static IP.
2. **Lớp security group**: cập nhật biến Terraform `proxy_allowed_cidr` = `<ip-mới>/32` (qua
   `$env:TF_VAR_proxy_allowed_cidr` hoặc file tfvars **không commit**), rồi:

   ```powershell
   terraform -chdir=deployments/terraform/ollama-ec2 plan
   terraform -chdir=deployments/terraform/ollama-ec2 apply
   ```

   **Kỳ vọng**: plan chỉ đổi `aws_security_group.ollama` (1 to change, 0 to add/destroy); apply mở
   `443` cho đúng `<ip-mới>/32` và **không** mở `11434`. Nếu hotfix tay bằng AWS Console/CLI thì phải
   reconcile lại Terraform ngay sau đó (nếu không, lần apply sau sẽ ghi đè ngược).

3. **Lớp Nginx**: qua SSM sửa `/etc/ollama-trial/trial.env` (file runtime do cloud-init render,
   người vận hành được phép sửa) — chỉ đổi dòng `PROXY_ALLOWED_CIDR=` sang `<ip-mới>/32`, rồi render
   lại cấu hình bằng script idempotent:

   ```bash
   sudo sed -i 's|^PROXY_ALLOWED_CIDR=.*|PROXY_ALLOWED_CIDR=<ip-mới>/32|' /etc/ollama-trial/trial.env
   sudo grep -E '^PROXY_ALLOWED_CIDR=' /etc/ollama-trial/trial.env
   sudo /opt/ollama-trial/bootstrap.sh     # idempotent: render lại nginx-ollama.conf + nginx -t + reload
   sudo nginx -T 2>/dev/null | grep -nE 'allow [0-9]|deny all|listen .*443'
   ```

   **Kỳ vọng**: `grep` in đúng một dòng `<ip-mới>/32`; `bootstrap.sh` exit `0` (nếu chỉ đổi allowlist
   thì model/cert đã có sẵn nên không cần pull lại); `nginx -T` cho thấy `allow <ip-mới>/32;` và
   `deny all;` trong block `listen ... 443`. Nếu `nginx -t` fail, bootstrap dừng trước khi reload —
   đó là hành vi đúng, không được bỏ qua.

4. **Xác minh sau khi đổi** (lặp lại §9.2 ở mức tối thiểu):
   - Từ box Lightsail: `POST /v1/chat/completions` với Bearer đúng → `200`.
   - Từ máy ngoài allowlist: timeout/bị chặn (hoặc `403` nếu tới được Nginx).
   - `http://<elastic-ip>:11434/api/version` → timeout/refused.
5. Nếu Bearer token bị rotate cùng lúc: cập nhật Secrets Manager và `.env.aws` trên Lightsail, rồi
   tạo lại container AI; không log token.

> Static IP chỉ đổi khi box bị tạo lại hoặc gỡ/gán lại static IP. Ghi mọi lần đổi IP (IP cũ → IP
> mới, thời điểm, ai làm) vào mục 7. Nếu IP đổi mà chưa cập nhật allowlist, triệu chứng là **mọi**
> request text bị timeout rồi rơi vào fallback Gemini (provider `google`, `fallback_reason` là
> lỗi transport) — đừng nhầm với lỗi Ollama.

## 5. Kiểm tra bất biến (T038 — chạy trước khi báo cáo)

Mục này chỉ dùng **lệnh đọc** (`git diff`, `git status`, `git grep`, `git log`) do **người vận
hành** chạy trên máy dev. Worker implement **không** chạy git trong feature này, nên không có dòng
nào dưới đây là kết quả đã đo; mọi ô "Kết quả" để trống cho tới khi người vận hành chạy và điền.

Biến `<base>` = nhánh gốc để so sánh (`main`, hoặc `origin/main` sau khi `git fetch`). Chạy từ thư
mục gốc repo.

### 5.1 Bộ lệnh read-only

```powershell
# 5.1.1 Không thay đổi artefact deployment
git diff --stat "<base>...HEAD" -- ':(glob)**/Dockerfile' deployments/docker-compose.aws.yml deployments/nginx/templates/aws .github/workflows/deploy-aws.yml
git diff --stat -- ':(glob)**/Dockerfile' deployments/docker-compose.aws.yml deployments/nginx/templates/aws .github/workflows/deploy-aws.yml

# 5.1.2 Danh sách file feature này chạm vào (đọc bằng mắt)
git diff --name-only "<base>...HEAD"
git status --porcelain

# 5.1.3 Không có tài nguyên EKS/Kubernetes/Helm
git grep -nEi '\b(eks|kubernetes|k8s|helm|kustomize|aws_eks)\b' -- deployments/terraform/ollama-ec2 deployments/ollama

# 5.1.4 Không có lệnh/probe/tham chiếu VPS trong artefact của feature
git grep -nEi '(ssh[^#]*vps|vps-[a-z0-9]|deploy-vps-services|/opt/closy/\.env\.production|vps_runbook)' -- deployments/terraform/ollama-ec2 deployments/ollama

# 5.1.5 Không có auto-teardown/schedule
git grep -nEi '(aws_autoscaling_schedule|scheduled_action|auto_?teardown|terraform destroy|aws s3 rm)' -- deployments/terraform/ollama-ec2 deployments/ollama

# 5.1.6 Dịch vụ dữ liệu ngoài không đổi (.env.aws.example)
git diff "<base>...HEAD" -- .env.aws.example | Select-String -Pattern '^[+-](DB_|REDIS_|RABBITMQ_|CLOUDINARY_|PAYOS_)'

# 5.1.7 Không có secret/private key trong diff
git grep -nE 'BEGIN (RSA|OPENSSH|EC) PRIVATE KEY|AIza[0-9A-Za-z_-]{35}|ghp_[A-Za-z0-9]{20,}' -- . ':!*.md'
```

Ghi chú: `git grep` chỉ quét file đang được track; với `.terraform/`, `*.tfstate*`, `*.tfvars` thì
`.gitignore` (đã cập nhật ở T003) là bằng chứng chúng không vào repo — kiểm thêm bằng
`git status --porcelain --ignored` nếu cần.

### 5.2 Bảng pass/fail

| # | Hạng mục bất biến | Lệnh | Kỳ vọng | Pass khi | Kết quả |
|---|---|---|---|---|---|
| 1 | `Dockerfile**` không đổi | 5.1.1 (dòng 1 và 2) | không có output | cả hai lệnh in rỗng | `TODO(operator)` |
| 2 | `deployments/docker-compose.aws.yml` không đổi | 5.1.1 | không có output | rỗng | `TODO(operator)` |
| 3 | `deployments/nginx/templates/aws/**` không đổi | 5.1.1 | không có output | rỗng | `TODO(operator)` |
| 4 | `.github/workflows/deploy-aws.yml` không đổi | 5.1.1 | không có output | rỗng | `TODO(operator)` |
| 5 | File feature chạm vào đúng phạm vi | 5.1.2 | chỉ thấy: `docs/operations/ollama-trial-*`, `specs/032-*`, `deployments/terraform/ollama-ec2/**`, `deployments/ollama/**`, `deployments/scripts/aws-env/aws-env-bootstrap.ps1`, `.env.aws.example`, `.env.ollama.example`, `.gitignore`, `shared/**`, `services/ai/**`, `configs/**` | không có file lạ ngoài danh sách; **không** có artefact VPS | `TODO(operator)` |
| 6 | 0 tài nguyên EKS/Kubernetes/Helm | 5.1.3 | 0 dòng, hoặc chỉ dòng **comment** nói "không dùng Kubernetes" | không có `resource`/`data`/`locals` nào; không có manifest `.yaml` kiểu k8s | `TODO(operator)` |
| 7 | 0 lệnh/probe VPS | 5.1.4 | 0 dòng | rỗng | `TODO(operator)` |
| 8 | 0 auto rollback/teardown/schedule | 5.1.5 | 0 dòng | rỗng | `TODO(operator)` |
| 9 | Supabase/Redis Cloud/CloudAMQP/Cloudinary/ payOS không đổi | 5.1.6 | 0 dòng | rỗng (diff `.env.aws.example` chỉ có dòng AI) | `TODO(operator)` |
| 10 | Không có secret trong repo | 5.1.7 | 0 dòng | rỗng | `TODO(operator)` |
| 11 | Không đổi public contract/client DTO | 5.1.2 + `git diff "<base>...HEAD" -- shared/application/dto shared/presentation services/ai/dto api docs/swagger` | không có file nào trong các path đó | rỗng | `TODO(operator)` |

Tất cả 11 mục phải `PASS` trước khi ghi SC-001/SC-004 vào báo cáo. Bất kỳ mục nào `FAIL` → dừng,
hoàn nguyên thay đổi ngoài phạm vi, **không** "ghi chú rồi bỏ qua".

## 6. Ngày thứ 7 — thu số liệu và chốt quyết định (operator, **NOT RUN**)

> **Trạng thái tại thời điểm implement (2026-10-27)**: mục này là **quy trình** cho người vận
> hành. Không bước nào dưới đây được chạy trong môi trường implement: không có quyền truy cập
> Lightsail/AWS/CloudWatch và không được deploy. Mọi ô kết quả là `TODO(operator)` / **NOT RUN**;
> không có số liệu hay quyết định nào được tạo trước.
>
> **Bất biến**: sau ngày thứ 7 cấu hình Ollama **tiếp tục active**. Không rollback, không tắt, không
> xoá, không teardown, không hạ cấu hình và không chạy `terraform destroy` chỉ vì hết 7 ngày.
> Uptime **không** phải tiêu chí quyết định. Các dịch vụ dữ liệu (Supabase/Postgres, Redis,
> RabbitMQ/CloudAMQP, Cloudinary) **không thay đổi** trong tuần trial; xác nhận bằng diff env và
> log kết nối ở §6.6.

### 6.1 Chốt mốc thời gian và trạng thái trước khi thu số liệu

```bash
cd /opt/closy-aws
date -u +%Y-%m-%dT%H:%M:%SZ
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml ps
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  logs --since 7d ai-http ai-worker | grep '"msg":"ai provider attempt"' | wc -l
```

Ghi `trial_started_at` (mục 1), thời điểm bắt đầu thu số liệu, `trial_finished_at` (UTC) và số dòng
attempt thô vào `docs/operations/ollama-trial-report.md` mục 1. Nếu `ps` không còn đủ 7 container
`Up` (3 http + 3 worker + nginx), ghi sự cố vào báo cáo mục 4; **không** tự rollback.

### 6.2 Thu log attempt AI (lỗi, latency, fallback, schema, thinking)

```bash
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  logs --since 7d ai-http ai-worker | grep '"msg":"ai provider attempt"' > /tmp/ai-attempts-7d.jsonl
```

Từ file đó, tổng hợp (dùng `jq`; cài `sudo apt-get install -y jq` nếu thiếu):

```bash
# outcome breakdown (mục 3 báo cáo)
jq -r '[.provider,.capability,.outcome]|@tsv' /tmp/ai-attempts-7d.jsonl | sort | uniq -c
# fallback theo lý do (mục 5 báo cáo)
jq -r 'select(.fallback_reason!="") | [.capability,.fallback_reason]|@tsv' /tmp/ai-attempts-7d.jsonl | sort | uniq -c
# schema failure và thinking > 0 trên openai (phải điều tra ngay)
jq -r 'select(.outcome=="schema_invalid") | [.capability,.model]|@tsv' /tmp/ai-attempts-7d.jsonl | sort | uniq -c
jq -r 'select(.provider=="openai" and (.thinking_tokens // 0) > 0)' /tmp/ai-attempts-7d.jsonl
# p95 latency attempt 1 (đổi capability khi cần)
jq -r 'select(.capability=="chat_text" and .attempt_index==1) | .latency_ms' /tmp/ai-attempts-7d.jsonl \
  | sort -n > /tmp/latency-chat.txt
awk '{a[NR]=$1} END {i=int((NR*95+99)/100); if(i<1)i=1; print "n="NR, "p95="a[i]}' /tmp/latency-chat.txt
```

Kỳ vọng: `thinking_tokens = 0` trên mọi dòng `provider=openai`; bất kỳ dòng nào `> 0` là lỗi cấu
hình (FR-026) → ghi sự cố, không coi trial hợp lệ. Điền mục 3 (lỗi), mục 4 (latency) và mục 5
(fallback) của báo cáo từ dữ liệu thô này; lưu đường dẫn file log đã lọc (không kèm prompt/secret)
vào mục 7.

### 6.3 Thu metric CPU/RAM/disk và proxy

Mở CloudWatch console → Dashboards → `ollama-trial-032` (Terraform T043). Đọc các metric theo
namespace `OllamaTrial`:

- `ec2_cpu_percent`, `ec2_memory_bytes`, `ec2_disk_free_bytes` (CloudWatch agent trên EC2).
- `proxy_5xx_total`, `proxy_401_total` (metric filter từ `/var/log/nginx/ollama-access.log`).

CloudWatch agent aggregate các metric agent theo `InstanceId` (và `InstanceId,path` cho disk), nên
dashboard/CLI dưới đây chỉ cần dimension `InstanceId`; muốn xem từng mount thì mở **Metrics** và chọn
dimension `path` trong namespace `OllamaTrial`.

Hoặc bằng CLI **chỉ đọc** (thay `<instance-id>` bằng `terraform output instance_id`):

```bash
aws cloudwatch get-metric-statistics --region ap-southeast-1 --namespace OllamaTrial \
  --metric-name ec2_cpu_percent --dimensions Name=InstanceId,Value=<instance-id> \
  --start-time <trial_started_at> --end-time <trial_finished_at> --period 3600 --statistics Average Maximum
aws cloudwatch get-metric-statistics --region ap-southeast-1 --namespace OllamaTrial \
  --metric-name ec2_memory_bytes --dimensions Name=InstanceId,Value=<instance-id> \
  --start-time <trial_started_at> --end-time <trial_finished_at> --period 3600 --statistics Average Maximum
aws cloudwatch get-metric-statistics --region ap-southeast-1 --namespace OllamaTrial \
  --metric-name ec2_disk_free_bytes --dimensions Name=InstanceId,Value=<instance-id> \
  --start-time <trial_started_at> --end-time <trial_finished_at> --period 3600 --statistics Minimum
aws cloudwatch get-metric-statistics --region ap-southeast-1 --namespace OllamaTrial \
  --metric-name proxy_5xx_total --start-time <trial_started_at> --end-time <trial_finished_at> \
  --period 3600 --statistics Sum
aws cloudwatch get-metric-statistics --region ap-southeast-1 --namespace OllamaTrial \
  --metric-name proxy_401_total --start-time <trial_started_at> --end-time <trial_finished_at> \
  --period 3600 --statistics Sum
```

Ghi giá trị lớn nhất/nhỏ nhất và thời điểm vào mục 7 của báo cáo. Ngưỡng điều tra: CPU >85% kéo dài
10 phút; RAM >90%; disk free <10 GiB; `proxy_5xx_total` >0 kéo dài 5 phút; `proxy_401_total` >0.
Không tự đổi cấu hình khi vượt ngưỡng trong lúc thu số liệu; ghi sự cố trước.

Kiểm tra queue và log Ollama (không cần CloudWatch):

```bash
sudo journalctl -u ollama --since "<7 days ago>" | grep -iE 'queue|out of memory|oom|error' | tail -n 50
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  logs --since 7d ai-http | grep -ciE 'queue full|too many requests|429|503'
```

Ghi queue depth đã đặt (`OLLAMA_MAX_QUEUE=8`), số lần queue full/429/503 và xác nhận không có attempt
thứ ba vào mục 6 của báo cáo.

**Log nguồn EC2 (CloudWatch)** — CloudWatch agent trên EC2 gửi mọi log nguồn vào **cùng một log
group** `/ollama-trial/ec2` (Terraform T043), retention **14 ngày**; bảng nguồn chuẩn ở
[contracts/trial-telemetry.md](../../specs/032-ollama-lightsail-trial/contracts/trial-telemetry.md) §3:

| Nguồn | Đường dẫn trên EC2 | Log stream | Nội dung |
|-------|--------------------|------------|----------|
| Nginx access | `/var/log/nginx/ollama-access.log` | `<instance-id>/nginx-access` | IP, status, bytes, request time; **không** body |
| Nginx error | `/var/log/nginx/ollama-error.log` | `<instance-id>/nginx-error` | Lỗi TLS/upstream |
| Ollama | journal `ollama.service` | `<instance-id>/ollama-journal` | Startup, model load, OOM; **không** prompt |
| Bootstrap | `/var/log/ollama-trial-bootstrap.log` | `<instance-id>/bootstrap` | Output của `bootstrap.sh` |
| Cloud-init | `/var/log/cloud-init-output.log` | `<instance-id>/cloud-init` | Output cloud-init: cài package, chạy `bootstrap.sh`, certbot, kiểm tra version/digest model |

`/var/log/cloud-init-output.log` được thêm vào `collect_list` của agent từ T059 để giữ lại log của
**toàn bộ lần boot đầu** (không chỉ phần `bootstrap.sh`): cài package, gọi bootstrap, certbot và đối
chiếu digest model. Dùng nó làm bằng chứng cho §6.1 và §6.4, và để đối chiếu version/digest khi
`bootstrap.log` thiếu dòng. Đọc bằng CLI **chỉ đọc** (thay `<instance-id>` bằng
`terraform output instance_id`):

```bash
aws logs tail /ollama-trial/ec2 --since 7d --log-stream-name-prefix <instance-id>/cloud-init \
  --region ap-southeast-1 --format short
```

Không cấu hình lại agent, không xoá log stream và không đổi retention trong tuần trial.

### 6.4 Thu sự cố trong tuần

Tập hợp mọi sự cố đã ghi trong tuần (từ §9.2, §9.3, log và nhật ký phiên) vào mục 4 của báo cáo:
thời điểm UTC, sự cố, ảnh hưởng, xử lý, còn tồn hay không. Không suy đoán sự cố không có bằng chứng;
nếu không có sự cố, ghi rõ "không ghi nhận sự cố" thay vì để trống.

### 6.5 Thu chi phí AWS

```bash
aws ce get-cost-and-usage --region us-east-1 \
  --time-period Start=<trial_started_at>,End=<trial_finished_at+1d> \
  --granularity DAILY --metrics UnblendedCost \
  --filter '{"Tags":{"Key":"Feature","Values":["032"]}}'
aws budgets describe-budgets --account-id <account-id> --region us-east-1 \
  --query 'Budgets[?BudgetName==`ollama-trial-80usd`]'
```

Ghi chi phí EC2, EBS, Elastic IP, Secrets Manager/SSM/CloudWatch vào mục 5 của báo cáo và đối chiếu
Budget 80 USD (cảnh báo 50/80/100%). Nếu vượt 80 USD: ghi nhận, **không** tự hạ cấu hình/teardown.

### 6.6 Xác nhận dịch vụ dữ liệu không đổi và uptime không phải tiêu chí

```bash
cd /opt/closy-aws
BACKUP="$(ls -t .env.aws.bak-* | head -1)"
diff <(grep -E '^(DB_|REDIS_|RABBITMQ_|CLOUDINARY_|PAYOS_)' "$BACKUP") \
     <(grep -E '^(DB_|REDIS_|RABBITMQ_|CLOUDINARY_|PAYOS_)' .env.aws) \
  && echo "data-service env unchanged"
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  logs --since 7d ai-http ai-worker commerce-http commerce-worker account-http account-worker \
  | grep -icE 'connection refused|no such host|authentication failed|certificate'
```

Ghi kết quả vào báo cáo mục 2. Kỳ vọng: `data-service env unchanged`; số dòng lỗi kết nối không tăng
so với baseline. **Uptime không phải tiêu chí** cho `go`/`adjust`/`no-go`; không dùng uptime để thay
thế các chỉ số SC-007.

### 6.7 Chốt quyết định `go` / `adjust` / `no-go` (có người và thời điểm)

Điền mục 6 của `docs/operations/ollama-trial-report.md`:

- `decided_at`: UTC, sau ngày thứ 7.
- `decided_by`: danh tính người quyết định (không phải tên nhóm chung).
- `decision`: một trong `go` / `adjust` / `no-go`.
- `rationale`: tham chiếu bảng SC-007, sự cố và chi phí; không cảm tính.
- `evidence`: link/đường dẫn metric/log/báo cáo đã thu.
- `follow_ups`: hạng mục còn thiếu trước production.

Nếu chưa có quyết định: ghi "chưa có quyết định", giữ nguyên cấu hình Ollama active, tiếp tục quan
sát. Quyết định `no-go`/`adjust` **không** kích hoạt rollback tự động; rollback vẫn là hành động thủ
công theo §4.2 nếu người vận hành chọn, và phải ghi thời gian thực hiện.

### 6.8 Kiểm tra read-only SC-010 — không có schedule/destroy/auto-teardown

T049 chỉ là kiểm tra **đọc**. Đã chạy trong môi trường implement ngày 2026-10-27 bằng PowerShell
`Select-String` (không chạy git, không chạy AWS), quét 8 file trong
`deployments/terraform/ollama-ec2/**` và `deployments/ollama/**`:

```powershell
$patterns = 'aws_autoscaling_schedule|scheduled_action|auto_teardown|auto-teardown|terraform destroy|aws s3 rm'
$files = @(Get-ChildItem deployments/terraform/ollama-ec2 -Recurse -File | ForEach-Object { $_.FullName })
$files += @(Get-ChildItem deployments/ollama -Recurse -File | ForEach-Object { $_.FullName })
Select-String -Path $files -Pattern $patterns -AllMatches
```

**Kết quả: PASS (read-only)** — `NO MATCHES` trên 8 file. Quét mở rộng thêm
`aws_autoscaling|aws_scheduler|schedule|prevent_destroy|force_destroy` chỉ thấy đúng một dòng
**comment** trong `variables.tf` ("Nothing here may create ... an auto-destroy schedule."), không có
`resource`/`data`/`locals` nào. Trong runbook, các từ `teardown`/`schedule` chỉ xuất hiện ở câu phủ
định ("không có ... tự động", §5.1.5 và §6); không có lệnh thực thi nào.

Lặp lại bằng git khi người vận hành chạy §5.1.5 (read-only). Không chạy AWS/Lightsail/EC2 để kiểm
tra mục này. Ghi kết quả PASS và bằng chứng vào báo cáo mục 8 (SC-010).

### 6.9 Bảng bằng chứng ngày thứ 7

| Hạng mục | Nguồn | Kỳ vọng | Kết quả | Bằng chứng |
|---|---|---|---|---|
| Log attempt 7 ngày | `ai-http`/`ai-worker` logs | đủ `request_id`, provider/model, outcome | `TODO(operator)` | đường dẫn `/tmp/ai-attempts-7d.jsonl` |
| Outcome breakdown | `jq` | đủ nhóm lỗi contract | `TODO(operator)` | output đã lọc |
| Latency p95 | `jq`/`awk` | theo SC-005 | `TODO(operator)` | file raw + output p95 |
| Fallback | `jq` | theo `fallback_reason` | `TODO(operator)` | output đã lọc |
| `thinking_tokens` trên `openai` | `jq` | `0` trên mọi dòng | `TODO(operator)` | output đã lọc |
| CPU/RAM/disk | CloudWatch dashboard/CLI | theo ngưỡng §6.3 | `TODO(operator)` | ảnh/output metric |
| `proxy_5xx_total`/`proxy_401_total` | CloudWatch | ghi nhận, điều tra nếu >0 | `TODO(operator)` | ảnh/output metric |
| Queue/Ollama | journal + log AI | queue hữu hạn, không attempt 3 | `TODO(operator)` | output |
| Sự cố | báo cáo mục 4 | đầy đủ hoặc "không ghi nhận" | `TODO(operator)` | nhật ký |
| Chi phí | Cost Explorer + Budget | đối chiếu 80 USD | `TODO(operator)` | output/ảnh |
| Dịch vụ dữ liệu | diff env + log | không đổi | `TODO(operator)` | output |
| Quyết định | báo cáo mục 6 | `go`/`adjust`/`no-go` + người + giờ | `TODO(operator)` | báo cáo |
| SC-010 | §6.8 | PASS read-only, không auto-teardown | **PASS (read-only)** | output `Select-String` trên |

## 7. Chỉ mục bằng chứng

Cập nhật đường dẫn/ghi chú thật khi có bằng chứng. Không dán secret vào đây.

| Tiêu chí | Bằng chứng cần có | Vị trí / ghi chú | Trạng thái |
|---|---|---|---|
| SC-001 | Smoke 6 container + diff artefact deployment + 0 tài nguyên EKS | mục 1, mục 5 (§5.2 #1–#6), log Lightsail, §9.1 | TODO |
| SC-002 | Test `openai` không chạm pricing/reserve, cost 0; Gemini billed giữ nguyên | output `go test` scoped, §9.4 | TODO |
| SC-003 | Chặn IP ngoài, thiếu/sai Bearer, chặn `11434`; Lightsail 200 | B5 + §9.2 (bảng bằng chứng) | TODO |
| SC-004 | 0 lệnh/probe VPS; 0 thay đổi artefact VPS | mục 5 (§5.2 #5, #7) | TODO |
| SC-005 | Load profile 50 request/capability, concurrency 2, 30 phút; p95 TTFT/latency | B6 + §9.1 bước 6 + raw sample | TODO |
| SC-006 | Drill rollback ≤15 phút, không mất dữ liệu đã xác nhận | B7 + §4.2 + §9.3 (bảng bằng chứng) | TODO |
| SC-007 | Báo cáo go/no-go đủ 8 nhóm chỉ số | `docs/operations/ollama-trial-report.md` | TODO |
| SC-008 | Regression API; text→Ollama, vision/embedding→Gemini; không đổi DTO | §9.4 (đã chạy phần local) + log provider ở §4.1.4 + §5.2 #11 | Một phần (local) |
| SC-009 | Timeout/429/5xx/malformed→Gemini đúng 1 lần; lỗi giữa stream không fallback; cancel/4xx 0 lần | test + drill B6 | TODO |
| SC-010 | 0 auto rollback/teardown sau ngày 7; quyết định có người/thời điểm | mục 6 (§6.7, §6.8) + báo cáo | **Một phần**: read-only PASS cho schedule/teardown (T049, §6.8); quyết định live vẫn `TODO` |
| SC-011 | Version + digest local/EC2 khớp; contract test pass trước EC2 | A1, A2, B2 | TODO |

Bằng chứng tối thiểu khác (theo quickstart §4): output contract test local, Terraform plan/apply
output không secret, log/metric provider thắng cho 3 text + vision/embedding, drill fallback và
rollback, báo cáo go/no-go có người quyết định.

## 8. Phase 3 (US1) — mã đã xong, các bước dưới đây là việc của người vận hành

> **Trạng thái tại thời điểm implement (2026-10-26)**: US1 đã được implement bằng mã và
> bằng unit/contract test chạy local. **T025, T026, T027 chưa được chạy**: môi trường
> implement không có `ollama`, `terraform`, `nginx`, không có AWS credentials và không có
> quyền truy cập Lightsail/EC2. Vì vậy **không có bằng chứng nào** cho SC-003, SC-005 hay
> SC-011 được tạo ra ở bước này; mọi ô "Trạng thái" ở mục 7 vẫn là `TODO` cho tới khi người
> vận hành chạy và điền.

**Trạng thái code phase (2026-10-27)**: Phase 1–4 và Phase 6 đã có commit; Phase 5 đã có tài liệu và
script env. Các mục **T025/T026/T027/T039/T040/T041/T048** vẫn là operator step — chưa có bằng chứng
live. T042 đã chạy phần regression local (mục 9.4). T050–T057 là gate/kiểm tra cuối, không tạo bằng
chứng thay cho live run. Các phát hiện ngoài phạm vi nằm ở
[deferred.md](../../specs/032-ollama-lightsail-trial/deferred.md).

### 8.1 T025 — kiểm tra chức năng + latency local (thủ công, chưa chạy)

Điều kiện: Ollama native đã cài và `qwen2.5:3b-instruct-q4_K_M` đã pull (A1). Nếu thiếu, đó là bước thủ công;
`deployments/ollama/local-test.ps1` **không** cài Ollama và **không** pull model — nó dừng với
exit code 2 và thông báo rõ.

```powershell
# Bắt buộc ép CPU trước khi đo tham chiếu, rồi khởi động lại Ollama.
$env:OLLAMA_NUM_GPU = "0"

# 1) Kiểm tra chức năng (non-stream, SSE, structured JSON, không reasoning) + 50 mẫu chat_text
.\deployments\ollama\local-test.ps1 -Capability chat_text -Samples 50 -Concurrency 2

# 2) free_text và recommendation_text (bỏ kiểm tra chức năng để không lặp lại)
.\deployments\ollama\local-test.ps1 -Capability free_text -Samples 50 -Concurrency 2 -SkipFunctional
.\deployments\ollama\local-test.ps1 -Capability recommendation_text -Samples 50 -Concurrency 2 -SkipFunctional `
  -Prompt "<prompt recommendation thực tế>"
```

**Kết quả kỳ vọng**: exit code `0`; log có `thinking off OK` (assert không reasoning), `structured output OK`, `stream OK`;
mỗi capability sinh `docs/temp/ollama-trial/local-latency-<capability>.csv` + `local-test-summary-<capability>.json`
(thư mục đã gitignore) và in `ttft p50/p95`, `total p50/p95`.

**Ghi bằng chứng**: dán p50/p95 + `ollama --version` + digest vào mục 1 và dòng SC-005 của mục 7,
kèm đường dẫn file raw. Số local CPU **chỉ là tham chiếu** — không thay SC-005 trên EC2.

**Nếu chưa cài Ollama/model**: ghi vào mục 7 là "chưa chạy", **không** đánh dấu đạt. Script có
`-AllowGpu` cho phép đo tham chiếu GPU nhưng kết quả đó không được dùng thay SC-005.

### 8.2 T026 — apply Terraform hai bước + DNS + SSM + kiểm tra cô lập (thủ công, chưa chạy)

Chuẩn bị input (dùng biến môi trường tạm, **không** tạo file `*.tfvars` được commit):

```powershell
$env:TF_VAR_proxy_allowed_cidr = "<lightsail-static-ip>/32"
$env:TF_VAR_proxy_hostname     = "ollama.<domain>"
$env:TF_VAR_proxy_acme_email   = "<email>"
$env:TF_VAR_owner              = "<owner>"
$env:TF_VAR_ollama_version     = "<version đo ở A1>"
$env:TF_VAR_model_digest       = "sha256:<digest đo ở A1>"
$env:TF_VAR_budget_alert_emails = '["<email>"]'   # biến list: truyền dạng chuỗi JSON
$env:TF_VAR_proxy_bearer_token = "<bearer-token dài, ngẫu nhiên>"
```

> `budget_alert_emails` là biến mới của module: alert 50/80/100% cần ít nhất một người nhận,
> nếu để trống apply sẽ dừng ở precondition của AWS Budget. `TF_VAR_proxy_bearer_token` là secret:
> chỉ đặt trong phiên shell, không dán vào file, không ghi vào log/evidence.

Apply hai bước (đúng thứ tự, không đảo):

```powershell
terraform -chdir=deployments/terraform/ollama-ec2 init
terraform -chdir=deployments/terraform/ollama-ec2 apply -target=aws_eip.ollama
terraform -chdir=deployments/terraform/ollama-ec2 output elastic_ip
# -> Tạo tay A record cho $env:TF_VAR_proxy_hostname trỏ tới IP vừa in (không phải domain API)
terraform -chdir=deployments/terraform/ollama-ec2 apply
terraform -chdir=deployments/terraform/ollama-ec2 output
```

**Kết quả kỳ vọng**: bước 1 chỉ tạo Elastic IP; bước 2 tạo host + proxy + secret + budget và in đủ
7 output của contract §3 (`instance_id`, `elastic_ip`, `proxy_hostname`, `secret_arn`,
`ollama_version`, `model_digest`, `security_group_id`). State là local, nằm ngoài repo.

Xác minh sau apply (qua SSM, **không** mở SSH):

```powershell
aws ssm start-session --target <instance_id>
# trong phiên SSM:
cat /etc/ollama-trial/TRIAL_READY          # phải tồn tại khi đã trial-ready
cat /etc/ollama-trial/TLS_PENDING          # nếu chưa có A record/cert: đọc "next_step" rồi chạy lại
sudo /opt/ollama-trial/bootstrap.sh        # idempotent; chạy lại sau khi DNS đã trỏ đúng
sudo ss -lntp | grep 11434                 # chỉ 127.0.0.1:11434
sudo ollama list                           # digest khớp TF_VAR_model_digest
sudo certbot certificates                  # còn hạn
sudo systemctl status ollama nginx
```

Kiểm tra cô lập (B5, SC-003) — chạy từ máy **ngoài** allowlist:

```powershell
curl.exe -v https://<ollama-host>/v1/chat/completions     # kỳ vọng: timeout/bị chặn
curl.exe -v http://<elastic-ip>:11434/api/version         # kỳ vọng: timeout/connection refused
curl.exe -v -H "Authorization: Bearer sai" https://<ollama-host>/v1/chat/completions
                                                          # kỳ vọng: 401 từ proxy
```

**Ứng dụng dùng** (không phải bước kiểm tra cô lập): từ Lightsail, request đúng endpoint + Bearer
phải trả `200`. Ghi tất cả output (đã lọc token) vào SC-003 của mục 7.

**Lưu ý hạ tầng**:
- Module dùng VPC/subnet mặc định của account (không tạo VPC riêng). Account chưa có default VPC thì
  bước 2 sẽ lỗi; xử lý ngoài phạm vi feature này và ghi vào `specs/032-ollama-lightsail-trial/deferred.md`.
- Nếu DNS chưa trỏ đúng lúc host boot, `bootstrap.sh` **không** fail cloud-init: nó giữ config
  ACME/HTTP, ghi `/etc/ollama-trial/TLS_PENDING` và thoát 0. Chạy lại script qua SSM sau khi DNS
  resolve; **không** đánh dấu trial-ready trước khi `TRIAL_READY` xuất hiện.

### 8.3 T027 — validate / plan / `nginx -t` (thủ công, chưa chạy)

```powershell
terraform -chdir=deployments/terraform/ollama-ec2 validate
terraform -chdir=deployments/terraform/ollama-ec2 plan
```

**Kết quả kỳ vọng**:

- `validate` → `Success! The configuration is valid.` (không cần biến nào).
- `plan` → `15 to add, 0 to change, 0 to destroy` (security group, Elastic IP, EIP association, IAM
  role + 2 policy attachment + 1 inline policy, instance profile, secret + version, instance, EBS
  volume + attachment, log group, budget) và 1 data source đọc AMI Ubuntu 24.04 từ SSM.
- `plan` **cần đủ input** ở §8.2 (kể cả `TF_VAR_proxy_bearer_token`) vì precondition fail-fast;
  thiếu input thì plan dừng với thông báo rõ — đó là hành vi đúng, không phải lỗi cấu hình.
- Không có `data`/`resource` nào nhắm Route53, EKS, ALB, ACM hoặc VPS.

`nginx -t` với `deployments/ollama/nginx-ollama.conf.template` **không chạy được nguyên bản** vì
file còn placeholder (`__PROXY_*__`) và đường dẫn chứng chỉ chưa tồn tại; máy dev Windows cũng không
có `nginx`. Cách kiểm tra đúng:

1. **Trên EC2 sau bootstrap** (khuyến nghị, đây là nơi template được render thật):
   `sudo nginx -t` → kỳ vọng `syntax is ok` + `test is successful` (bootstrap.sh đã gọi lệnh này
   trước mỗi lần reload; nếu template sai, bootstrap fail trước khi ghi `TRIAL_READY`).
2. **Kiểm tra cú pháp local có kiểm soát** (chỉ khi máy có nginx + openssl): render template với giá
   trị giả, tạo chứng chỉ self-signed tạm, include file đã render vào một `nginx.conf` tối thiểu rồi
   `nginx -t -c <file> -p <prefix>`. Kết quả mong đợi cũng là `syntax is ok` / `test is successful`;
   đây chỉ là kiểm tra cú pháp, **không** phải bằng chứng TLS/allowlist của SC-003.

Không được đánh dấu T025/T026/T027 là đã chạy cho tới khi có output thật (không kèm secret) được
dán vào mục 7.

## 9. Phase 5 (US3) — checklist người vận hành (T039–T042)

> **Trạng thái tại thời điểm implement (2026-10-27)**: T037, T038, T040, T041 mới là **tài liệu**;
> phần script của T039 (`deployments/scripts/aws-env/aws-env-bootstrap.ps1`) đã sửa, còn **mọi bước
> chạy trên Lightsail/EC2 là operator step và CHƯA được chạy**. T042 mới chạy phần **regression
> local** (§9.4); phần bằng chứng provider trên môi trường thật vẫn là operator step.
>
> Môi trường implement **không** có quyền truy cập Lightsail, AWS, EC2, không có `gh`, không có
> `docker` trên box, và **không** được deploy. Vì vậy không ô "Kết quả" nào dưới đây được điền sẵn,
> và **không** được coi là đã đạt.

### 9.1 T039 — thực thi trên Lightsail (operator chạy, chưa chạy)

Điều kiện vào: baseline mục 1 đã chốt, EC2 trial-ready (`/etc/ollama-trial/TRIAL_READY` tồn tại —
A6/B2), sáu container đang healthy.

- [ ] **1. Chốt T0 và trạng thái trước khi đổi env**

  ```bash
  cd /opt/closy-aws
  date -u +%Y-%m-%dT%H:%M:%SZ
  docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml ps
  ```

  Kỳ vọng: 7 dòng `Up` (3 http + 3 worker + nginx). Nếu baseline đã lỗi → dừng, xử lý ngoài phạm vi.

- [ ] **2. Xác nhận 6 key AI mới đã có trong `.env.aws`** (do `aws-env-bootstrap.ps1` bảo đảm
      không bị rơi; giá trị trial vẫn do người vận hành đặt):

  ```bash
  grep -cE '^(AI_TEXT_MAX_CONCURRENCY|AI_TEXT_PRIMARY_TIMEOUT_SECONDS|AI_TEXT_FALLBACK_TIMEOUT_SECONDS|AI_OPENAI_REASONING_EFFORT|AI_CHAT_CHUNK_DEADLINE_SECONDS|AI_OPENAI_TRIAL_EXPECTED_HOSTS)=' .env.aws
  ```

  Kỳ vọng: `6`. Thiếu key nào → bổ sung theo `.env.aws.example` **trước** khi bật trial.

- [ ] **3. Bật trial** đúng §4.1.2 (sửa `.env.aws` → chạy pipeline hiện hữu
      `gh workflow run deploy-aws.yml -R "hycat-team/smart-wardrobe-be" --ref main -f services=ai`).
      Không sửa Dockerfile/Compose/Nginx/pipeline.

- [ ] **4. Xác minh 6 container + env** theo §4.1.3 (7 dòng `Up`, 7 dòng env đúng, không có lỗi
      validation cấu hình).

- [ ] **5. Xác minh 3 capability text + `summary`/`rewriter` do Ollama phục vụ** theo §4.1.4
      (provider `openai`, `billing_class=unbilled`, `thinking_tokens=0`, `actual_cost=0`), và
      vision/embedding vẫn `google`/`billed`.

- [ ] **6. Chạy load profile SC-005** (operator; repo **không** có load generator, xem ghi chú dưới):

  - 50 request cho **mỗi** capability `chat_text`, `free_text`, `recommendation_text`,
    concurrency 2, toàn bộ mẫu nằm trong cửa sổ **30 phút**; ghi `trial_load_started_at` /
    `trial_load_finished_at`.
  - Gửi qua **API thật** (không gọi trực tiếp Ollama): mỗi request phải đi qua AI service để có một
    dòng `ai provider attempt` với `request_id`.
  - Lưu raw sample phía client (`request_id`, capability, `ttft_ms`, `latency_ms`, HTTP status,
    lỗi/timeout nếu có) và **join** với log service theo `request_id`.
  - Phân loại từng mẫu: `success` (attempt 1 `openai`), `fallback` (`provider=google` +
    `fallback_reason` khác rỗng), `failure` (request bị write timeout 120 giây cắt — client thấy
    504/EOF và **không** có nội dung).

  Tính p95 từ log (đổi `chat_text` cho từng capability; chỉ lấy attempt 1 để không trộn fallback):

  ```bash
  cd /opt/closy-aws
  docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
    logs --since 45m ai-http | grep '"msg":"ai provider attempt"' \
    | jq -r 'select(.capability=="chat_text" and .attempt_index==1) | .ttft_ms' | sort -n > /tmp/ttft_chat.txt
  awk '{a[NR]=$1} END {i=int((NR*95+99)/100); if(i<1)i=1; print "n="NR, "p95="a[i]}' /tmp/ttft_chat.txt
  ```

  Kỳ vọng (SC-005): `chat_text`/`free_text` p95 `ttft_ms` ≤ 15000; `recommendation_text` p95
  `ttft_ms` ≤ 60000 và hoàn tất ≤ 120000 ms; ≥95% attempt `openai` có `latency_ms` ≤ 90000.
  Ghi cả số mẫu `n` — nếu `n` nhỏ hơn 50 thì mẫu **không** hợp lệ, không được báo p95.

- [ ] **7. Xác nhận dịch vụ dữ liệu ngoài không đổi và uptime không phải tiêu chí**

  ```bash
  cd /opt/closy-aws
  BACKUP="$(ls -t .env.aws.bak-* | head -1)"
  diff <(grep -E '^(DB_|REDIS_|RABBITMQ_|CLOUDINARY_|PAYOS_)' "$BACKUP") \
       <(grep -E '^(DB_|REDIS_|RABBITMQ_|CLOUDINARY_|PAYOS_)' .env.aws) && echo "data-service env unchanged"
  docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
    logs --since 60m ai-http ai-worker commerce-http commerce-worker account-http account-worker \
    | grep -icE 'connection refused|no such host|authentication failed|certificate'
  ```

  Kỳ vọng: in `data-service env unchanged`; số dòng lỗi kết nối không tăng so với baseline (ghi cả
  con số, không chỉ "không đổi"). **Uptime không phải tiêu chí** — không dùng nó làm pass/fail cho
  SC-005; không có schedule/auto-teardown.

- [ ] **8. Điền bảng bằng chứng T039**

  | Hạng mục | Nguồn | Kỳ vọng | Kết quả | Bằng chứng (không kèm secret) |
  |---|---|---|---|---|
  | 6 container + nginx | `docker compose ... ps` | 7/7 `Up`, không `Restarting` | `TODO(operator)` | `TODO(operator)` |
  | 7 key điều khiển trial trong `ai-http` | `docker inspect` (§4.1.3) | đúng 7 dòng §4.1.3 | `TODO(operator)` | `TODO(operator)` |
  | `chat_text` | log `ai provider attempt` | `openai`/`unbilled`/success | `TODO(operator)` | `TODO(operator)` |
  | `free_text` | log | `openai`/`unbilled`/success | `TODO(operator)` | `TODO(operator)` |
  | `recommendation_text` | log | `openai`/`unbilled`/success | `TODO(operator)` | `TODO(operator)` |
  | `summary` (route free/paid) | log | `openai`, capability `free_text` hoặc `chat_text` | `TODO(operator)` | `TODO(operator)` |
  | `rewriter` (route free/paid) | log | `openai`, capability `free_text` hoặc `chat_text` | `TODO(operator)` | `TODO(operator)` |
  | vision, embedding | `ai_usage_events` (provider/model) + log service hiện hữu | `google`/`billed` | `TODO(operator)` | `TODO(operator)` |
  | cost 3 đường text | `ai_usage_events` SQL (§4.1.4) | `actual_cost_micro_vnd = 0`, `reserved = 0` | `TODO(operator)` | `TODO(operator)` |
  | SC-005 load | raw sample + p95 | theo ngưỡng SC-005, `n >= 50`/capability | `TODO(operator)` | đường dẫn raw sample |
  | Dịch vụ dữ liệu ngoài | `diff` env + log | không đổi, không lỗi kết nối mới | `TODO(operator)` | `TODO(operator)` |
  | Đường deploy đã dùng | `gh run` hoặc recreate | pipeline hiện hữu, không diff artefact | `TODO(operator)` | run id / thời lượng |

### 9.2 T040 — checklist kiểm tra cô lập truy cập (operator chạy, chưa chạy)

Mục tiêu: SC-003 — 100% request ngoài allowlist bị chặn, 100% thiếu/sai xác thực bị từ chối, 100%
request hợp lệ từ Lightsail được chấp nhận, 100% phép thử trực tiếp `11434` từ Internet bị chặn.

Quy ước: `<ollama-host>` là subdomain Ollama (không phải domain API), `<elastic-ip>` là output
`elastic_ip`, `<bearer-token>` đọc từ Secrets Manager (không dán vào evidence).

| ID | Mục đích | Nơi chạy | Lệnh | Kỳ vọng |
|---|---|---|---|---|
| A1 | IP ngoài allowlist, không Bearer | máy **không** thuộc Lightsail | `curl.exe -sv --max-time 15 https://<ollama-host>/v1/chat/completions` | `curl: (28)` timeout (security group) hoặc connection reset; **không** có HTTP 200; nếu tới được Nginx thì phải `403` |
| A2 | IP ngoài allowlist, Bearer **đúng** | máy ngoài allowlist | `curl.exe -sv --max-time 15 -H "Authorization: Bearer <bearer-token>" https://<ollama-host>/v1/chat/completions` | vẫn bị chặn như A1 — chứng minh chặn theo IP, không phải theo token |
| B1 | Thiếu Bearer | **trên box Lightsail** | `curl -s -o /dev/null -w "%{http_code}\n" -X POST https://<ollama-host>/v1/chat/completions -H "Content-Type: application/json" -d '{}'` | `401` (Nginx trả trước khi tới Ollama) |
| B2 | Sai Bearer | trên box Lightsail | `curl -s -o /dev/null -w "%{http_code}\n" -X POST https://<ollama-host>/v1/chat/completions -H "Authorization: Bearer wrong-token" -H "Content-Type: application/json" -d '{}'` | `401` |
| B3 | Request hợp lệ từ Lightsail | trên box Lightsail | `curl -s -o /dev/null -w "%{http_code}\n" -X POST https://<ollama-host>/v1/chat/completions -H "Authorization: Bearer <bearer-token>" -H "Content-Type: application/json" -d '{"model":"qwen2.5:3b-instruct-q4_K_M","messages":[{"role":"user","content":"ping"}],"stream":false,"max_tokens":8}'` | `200` |
| C1 | Truy cập trực tiếp `11434` | máy ngoài (và lặp lại từ box Lightsail) | `curl.exe -sv --max-time 10 http://<elastic-ip>:11434/api/version` | `curl: (28)` timeout hoặc `(7)` refused; **không** có HTTP status, không có JSON version |
| C2 | Ollama chỉ loopback | SSM trên EC2 | `sudo ss -lntH \| grep 11434` | chỉ `127.0.0.1:11434` |
| D1 | Path không hỗ trợ | trên box Lightsail | `curl -s -o /dev/null -w "%{http_code}\n" https://<ollama-host>/api/version` | `404` (proxy đóng mọi path khác) |
| D2 | Health probe không cần token | trên box Lightsail | `curl -s -o /dev/null -w "%{http_code}\n" https://<ollama-host>/healthz` | `200` |

Đọc log proxy để đối chiếu (metric `proxy_401_total`/`proxy_5xx_total` do T043 tạo từ chính access log
này; field thứ 3 là status):

```bash
# trên EC2 qua SSM
sudo awk '$3 == 401 {count++} END {print count+0}' /var/log/nginx/ollama-access.log
sudo tail -n 20 /var/log/nginx/ollama-access.log
sudo tail -n 20 /var/log/nginx/ollama-error.log
```

**Bảng bằng chứng SC-003**

| Test | Kỳ vọng | Kết quả quan sát | Bằng chứng (log/ảnh đã lọc token) | Pass? |
|---|---|---|---|---|
| A1 | timeout / `403`, không 200 | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| A2 | chặn như A1 | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| B1 | `401` | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| B2 | `401` | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| B3 | `200` | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| C1 | timeout / refused | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| C2 | chỉ `127.0.0.1:11434` | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| D1 | `404` | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |
| D2 | `200` | `TODO(operator)` | `TODO(operator)` | `TODO(operator)` |

SC-003 chỉ đạt khi **toàn bộ** 9 dòng `Pass`. Ghi rõ giờ chạy và máy chạy từng test (A1/A2 phải
chạy từ máy **không** phải box Lightsail, nếu không bằng chứng vô hiệu).

### 9.3 T041 — drill rollback ≤15 phút (operator chạy, chưa chạy)

Điều kiện: §9.1 đã xong và trial đang phục vụ text bằng `openai`; có đồng hồ bấm giờ (điện thoại
hoặc `Measure-Command`/`date`).

- [ ] **1. Dữ liệu đã xác nhận trước drill (chạy trước T0)** — Supabase, chỉ đọc:

  ```sql
  SELECT count(*) AS confirmed_rows, max(created_at) AS last_confirmed
  FROM ai_usage_events
  WHERE status = 'confirmed' AND created_at >= now() - interval '24 hours';
  ```

  Lưu kết quả vào evidence (SC-006).

- [ ] **2. T0**: ghi giờ UTC bắt đầu drill.
- [ ] **3. Thực hiện rollback** đúng §4.2 bước 1–5 (chỉ đổi ba `*_TEXT_PRIMARY_*`, giữ
      `AI_UNBILLED_CAPABILITIES`/vision/embedding/timeout).
- [ ] **4. Xác minh sau rollback**:
  - 7 container `Up`, không `Restarting`;
  - 100% request text có `provider=google`, không còn dòng `openai` mới;
  - `ai_usage_events` không còn event `openai` mới; Gemini billed có `reserved_cost_micro_vnd > 0`
    trước khi gọi và `actual_cost_micro_vnd` được confirm.
- [ ] **5. Không mất dữ liệu đã xác nhận** — lặp lại query ở bước 1 **sau** T1:

  ```sql
  SELECT count(*) AS confirmed_rows, max(created_at) AS last_confirmed
  FROM ai_usage_events
  WHERE status = 'confirmed' AND created_at >= now() - interval '24 hours';

  SELECT status, count(*) FROM ai_usage_events
  WHERE status IN ('reserved','in_flight') AND created_at < now() - interval '5 minutes'
  GROUP BY status;
  ```

  Kỳ vọng: `confirmed_rows` **không giảm**; query thứ hai trả **0 dòng** (không còn reservation treo
  quá write timeout 120 giây). Nếu có dòng treo: ghi vào báo cáo sự cố, không tự sửa dữ liệu.
- [ ] **6. T1**: ghi giờ UTC kết thúc; `Δ = T1 - T0` phải `<= 15 phút`.
- [ ] **7. Không thao tác VPS**: xác nhận bằng §5.2 #7 và bằng nhật ký phiên (không SSH/deploy/đọc
      secret/đổi DNS VPS).
- [ ] **8. Nếu trial tiếp tục**: bật lại theo §4.1 và ghi thời gian bật lại **riêng** (không tính
      vào SC-006). Không đánh tráo số đo hai chiều.

**Bảng bằng chứng SC-006**

| Hạng mục | Kỳ vọng | Kết quả | Bằng chứng |
|---|---|---|---|
| T0 (UTC) | — | `TODO(operator)` | `TODO(operator)` |
| T1 (UTC) | — | `TODO(operator)` | `TODO(operator)` |
| Δ | ≤ 15 phút | `TODO(operator)` | `TODO(operator)` |
| Đường rollback | pipeline hiện hữu hoặc recreate env-only | `TODO(operator)` | run id / lệnh |
| 7 container | `Up`, không `Restarting` | `TODO(operator)` | `docker compose ps` |
| Text provider sau rollback | 100% `google` | `TODO(operator)` | log `ai provider attempt` |
| `confirmed_rows` trước/sau | không giảm | `TODO(operator)` | 2 kết quả SQL |
| Reservation treo | 0 dòng | `TODO(operator)` | kết quả SQL |
| Thao tác VPS | 0 | `TODO(operator)` | §5.2 #7 |
| Artefact deployment | 0 diff | `TODO(operator)` | §5.2 #1–#4 |

### 9.4 T042 — regression API local (**đã chạy** ngày 2026-10-27)

Môi trường: Windows, `go1.26.3 windows/amd64`, `CGO_ENABLED=0`, workspace `go.work`. Đây là chạy
**local**, không phải live. Không chạy `-race` (xem lý do bên dưới).

| # | Lệnh | Kết quả |
|---|---|---|
| 1 | `go test ./shared/... -count=1` | **PASS** — mọi package có test đều `ok`, không package nào `FAIL`: `shared/application/constants/apperror`, `shared/application/event`, `shared/application/sse`, `shared/config`, `shared/domain/constants/subscriptioneventtype`, `shared/domain/entities`, `shared/domain/repositories`, `shared/infrastructure/ai`, `shared/infrastructure/ai/google`, `shared/infrastructure/ai/openai`, `shared/infrastructure/caching`, `shared/infrastructure/db`, `shared/infrastructure/messaging`, `shared/infrastructure/search`, `shared/pkg/logger`, `shared/pkg/utils/*` (colorutils, errorutils, jwtutils, numberutils, timeutils, validation), `shared/presentation`, `shared/presentation/middleware`, `shared/presentation/worker`, `shared/tools/*` (hygiene-scan, migration-ownership-check, nginx-check, nginx-routes, rabbitmq-topology, swagger-combine). Các package còn lại `[no test files]` |
| 2 | `go test ./services/ai/internal/modules/aiexecutor/application/usecase/... -count=1` | **PASS** — `usecase/admin` ok, `usecase/cost` ok, `usecase/grant` ok, `usecase/orchestrator` ok; `usecase/pricing` `[no test files]` |
| 3 | `go test -race ...` | **KHÔNG chạy** — `CGO_ENABLED=0` trên host này nên race detector không khả dụng (`-race requires cgo`). **Không** đánh dấu pass cho race; race là gate của Phase 7 (T050) trên môi trường có CGO |
| 4 | Kiểm tra vision/embedding vẫn chọn Gemini | **PASS (bằng chứng tĩnh + test)** — `configs/config.yaml` và `configs/config.production.yaml` giữ `${VISION_PRIMARY_PROVIDER}`/`${EMBEDDING_PRIMARY_PROVIDER}` (không hardcode `openai`); test `TestVisionAndEmbeddingStayOnGoogle` trong `shared/infrastructure/ai/text_routing_test.go` khẳng định vision/embedding chọn `google` và đã pass ở lệnh 1 |
| 5 | Không đổi client DTO/HTTP contract | **Một phần** — phase này chỉ sửa `docs/operations/ollama-trial-runbook.md` và `deployments/scripts/aws-env/aws-env-bootstrap.ps1`; không sửa file DTO/handler nào. Bằng chứng **đầy đủ** phải là diff git do **người vận hành** chạy (§5.2 #11) vì worker bị cấm chạy git trong feature này |
| 6 | Lỗi do môi trường (sqlite/CGO) | **Không gặp** — không package nào fail vì sqlite/CGO ở hai lệnh trên; không có mục nào bị bỏ qua |

Ghi chú cho SC-008: phần **local** đã có bằng chứng (routing/config/test). Phần **live** (text →
Ollama, vision/embedding → Gemini trên box, client DTO không đổi) vẫn phải lấy từ §4.1.4 + §5.2 #11
khi người vận hành chạy T039 — không được coi là đã đạt chỉ vì hai lệnh `go test` ở trên xanh.
