# Runbook — Thử nghiệm Ollama trên EC2 (feature 032)

**Feature**: `specs/032-ollama-lightsail-trial/` | **Ngày tạo**: 2026-10-26 | **Trạng thái**: khung
chờ vận hành (Phase 1 Setup)

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
> Ollama native và pull `qwen3:4b` là **bước thủ công bắt buộc** trước A1; không có đường tắt tự
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
| Ollama version local (A1) | `TODO(operator)` | `ollama --version` sau khi cài tay | Chưa điền |
| Digest `qwen3:4b` local (A1) | `TODO(operator)` | `ollama list` / output `ollama pull` → `sha256:...` | Chưa điền |
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
ollama pull qwen3:4b
ollama list
ollama show qwen3:4b --modelfile
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
§5 pass (thinking off, structured output, SSE, usage thiếu/0, lỗi 4xx/429/5xx, cancellation).

### A3. Probe thủ công

Non-stream:

```powershell
$body = @{
  model = "qwen3:4b"
  messages = @(
    @{ role = "system"; content = "Trả lời ngắn gọn." },
    @{ role = "user"; content = "Xin chào" }
  )
  stream = $false
  max_tokens = 50
  temperature = 0
  reasoning_effort = "none"
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
3. Dừng Ollama (`Stop-Process -Name ollama` hoặc kill tiến trình phục vụ).
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

`local-test.ps1` chạy kiểm tra chức năng (non-stream, streaming, structured output, thinking off)
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

### A6. Cổng ra Phase A

Chỉ apply Terraform khi tất cả đạt:

- [ ] Contract test pass.
- [ ] Thinking off được chứng minh.
- [ ] Structured output hợp lệ.
- [ ] Drill fallback pass.
- [ ] SSE chat không bị cắt: `AI_CHAT_CHUNK_DEADLINE_SECONDS=90` và primary timeout 90 đã đặt.
- [ ] Recommendation: TTFT + thời gian sinh output nằm trong 90 giây, hoặc đã ghi nhận điều chỉnh/
      fallback trước khi đo.
- [ ] Digest/version đã ghi.

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

Đổi ba `*_TEXT_PRIMARY_*` về Gemini như §1.5 của env contract rồi chạy lại deploy hiện hữu.

**Đạt khi**: trong ≤15 phút, text phục vụ bằng Gemini; sáu container healthy; không mất dữ liệu đã
xác nhận; không thao tác VPS. Ghi bằng chứng cho SC-006.

## 4. Chuyển cấu hình (bật/tắt) và cập nhật allowlist

### 4.1 Bật trial (Gemini-only → Ollama primary)

Chỉ sửa `/opt/closy-aws/.env.aws`, không sửa artefact deployment. Giá trị theo
[contracts/env-ai-trial.md](../../specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md) §1:

| Nhóm biến | Giá trị khi bật |
|---|---|
| `CHAT_TEXT_PRIMARY_*`, `RECOMMENDATION_TEXT_PRIMARY_*`, `FREE_TEXT_PRIMARY_*` | `PROVIDER=openai`; `ENDPOINT=https://<ollama-host>/v1/chat/completions`; `MODEL=qwen3:4b`; `API_KEY=<bearer-token>` (trùng token proxy, không log) |
| `*_TEXT_FALLBACK_*` | Giữ `google` với key/endpoint/model Gemini hiện hữu |
| `VISION_*`, `EMBEDDING_*` | **Không đổi** (Gemini) |
| `AI_UNBILLED_CAPABILITIES` | Giữ `free_text,embedding`; **không** dùng để miễn phí OpenAI |
| `AI_TEXT_MAX_CONCURRENCY` | `2` |
| `AI_TEXT_PRIMARY_TIMEOUT_SECONDS` | `90` |
| `AI_TEXT_FALLBACK_TIMEOUT_SECONDS` | `30` |
| `AI_OPENAI_REASONING_EFFORT` | `none` |
| `AI_CHAT_CHUNK_DEADLINE_SECONDS` | `90` |
| `AI_OPENAI_TRIAL_EXPECTED_HOSTS` | `<ollama-host>` (host, không scheme/path) |

Sau khi sửa: chạy quy trình deploy AWS hiện hữu (`docs/operations/deployment-pipeline.md`), chờ
container AI healthy, rồi lặp lại B4 + B5.

### 4.2 Rollback về Gemini-only (mục tiêu ≤15 phút, SC-006)

1. Đổi ba `*_TEXT_PRIMARY_*` về cấu hình Gemini hiện hữu: provider `google`, cùng key/endpoint/model
   với block `*_TEXT_FALLBACK_*`.
2. Giữ nguyên `AI_UNBILLED_CAPABILITIES`, vision, embedding; không sửa Compose/Nginx/Dockerfile.
3. Chạy deploy hiện hữu, chờ container AI healthy.
4. Xác minh: text trả `provider=google`; sáu container healthy; không mất dữ liệu đã xác nhận;
   không có request nào vẫn trỏ endpoint Ollama.
5. Ghi mốc thời gian bắt đầu/kết thúc vào evidence index (SC-006).

### 4.3 Cập nhật allowlist khi static IP Lightsail đổi

Dấu hiệu: request từ Lightsail bị chặn/timeout trong khi EC2 và proxy vẫn sống; kiểm tra log Nginx
trên EC2 thấy IP nguồn lạ, hoặc Lightsail Console báo static IP mới.

1. Lấy static IP mới từ Lightsail Console → Instances → Networking → Static IP.
2. Cập nhật biến Terraform `proxy_allowed_cidr` = `<ip-mới>/32` (truyền qua `-var` hoặc file tfvars
   **không commit**), rồi `terraform -chdir=deployments/terraform/ollama-ec2 apply`. Đây là nguồn
   chuẩn của security group; nếu hotfix tay bằng AWS Console/CLI thì phải reconcile lại Terraform
   ngay sau đó.
3. Cập nhật `PROXY_ALLOWED_CIDR` trong `/etc/ollama-trial/trial.env` trên EC2 và render lại
   `nginx-ollama.conf.template`, rồi `nginx -t && systemctl reload nginx` qua SSM.
4. Xác minh lại: request từ IP mới trả 200; IP ngoài allowlist vẫn bị chặn; `11434` vẫn không public.
5. Nếu Bearer token bị rotate cùng lúc: cập nhật Secrets Manager và `.env.aws` trên Lightsail, rồi
   tạo lại container AI; không log token.

> Static IP chỉ đổi khi box bị tạo lại hoặc gỡ/gán lại static IP. Ghi mọi lần đổi IP vào mục 7.

## 5. Kiểm tra bất biến (chạy trước khi báo cáo)

Chạy trên máy dev (read-only, người vận hành; worker không tự chạy git):

```powershell
git diff --stat -- ':(glob)**/Dockerfile' deployments/docker-compose.aws.yml deployments/nginx/templates/aws .github/workflows/deploy-aws.yml
git status --porcelain
```

- [ ] Không có thay đổi ở `Dockerfile`, `deployments/docker-compose.aws.yml`,
      `deployments/nginx/templates/aws/**`, `.github/workflows/deploy-aws.yml`.
- [ ] Không có tài nguyên EKS/Kubernetes/ALB/ACM trong Terraform hay AWS.
- [ ] Không có lệnh, probe hay thay đổi DNS nào nhắm vào VPS.
- [ ] Supabase, Redis Cloud, CloudAMQP, Cloudinary không đổi.
- [ ] Không có secret trong repo, state, log workflow hay output CI.

## 6. Ngày thứ 7

1. **Không** rollback/tắt/teardown tự động. Cấu hình Ollama tiếp tục active.
2. Thu đủ số liệu theo
   [contracts/trial-telemetry.md](../../specs/032-ollama-lightsail-trial/contracts/trial-telemetry.md)
   §4 và lưu bằng chứng (metric/log raw, không kèm prompt hay secret).
3. Điền đầy đủ `docs/operations/ollama-trial-report.md`, gồm bảng SC-007 và bảng quyết định.
4. Chốt `go` / `adjust` / `no-go` với danh tính người quyết định và thời điểm. Nếu chưa quyết định:
   giữ nguyên cấu hình, tiếp tục quan sát và ghi rõ "chưa có quyết định" trong báo cáo.
5. Không hạ ngưỡng SC-005 sau khi đã bắt đầu đo; mọi điều chỉnh phải ghi lý do và thời điểm.

## 7. Chỉ mục bằng chứng

Cập nhật đường dẫn/ghi chú thật khi có bằng chứng. Không dán secret vào đây.

| Tiêu chí | Bằng chứng cần có | Vị trí / ghi chú | Trạng thái |
|---|---|---|---|
| SC-001 | Smoke 6 container + diff artefact deployment + 0 tài nguyên EKS | mục 1, mục 5, log Lightsail | TODO |
| SC-002 | Test `openai` không chạm pricing/reserve, cost 0; Gemini billed giữ nguyên | output `go test` scoped | TODO |
| SC-003 | Chặn IP ngoài, thiếu/sai Bearer, chặn `11434`; Lightsail 200 | B5 + log proxy | TODO |
| SC-004 | 0 lệnh/probe VPS; 0 thay đổi artefact VPS | mục 5 | TODO |
| SC-005 | Load profile 50 request/capability, concurrency 2, 30 phút; p95 TTFT/latency | B6 + raw sample | TODO |
| SC-006 | Drill rollback ≤15 phút, không mất dữ liệu đã xác nhận | B7 + mốc thời gian | TODO |
| SC-007 | Báo cáo go/no-go đủ 8 nhóm chỉ số | `docs/operations/ollama-trial-report.md` | TODO |
| SC-008 | Regression API; text→Ollama, vision/embedding→Gemini; không đổi DTO | output test + log provider | TODO |
| SC-009 | Timeout/429/5xx/malformed→Gemini đúng 1 lần; lỗi giữa stream không fallback; cancel/4xx 0 lần | test + drill B6 | TODO |
| SC-010 | 0 auto rollback/teardown sau ngày 7; quyết định có người/thời điểm | mục 6 + báo cáo | TODO |
| SC-011 | Version + digest local/EC2 khớp; contract test pass trước EC2 | A1, A2, B2 | TODO |

Bằng chứng tối thiểu khác (theo quickstart §4): output contract test local, Terraform plan/apply
output không secret, log/metric provider thắng cho 3 text + vision/embedding, drill fallback và
rollback, báo cáo go/no-go có người quyết định.
