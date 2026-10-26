# Quickstart: Kiểm chứng thử nghiệm Ollama

**Feature**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26

Hướng dẫn này là kịch bản kiểm chứng end-to-end. Không chứa code triển khai; chi tiết hợp đồng
nằm ở [contracts/](./contracts/) và [data-model.md](./data-model.md).

## 0. Điều kiện tiên quyết

- Go 1.26+, Docker Desktop (đã có), Ollama native trên Windows.
- AWS credentials có quyền EC2/EIP/EBS/IAM/SSM/Secrets Manager/Budget trong `ap-southeast-1`.
- AWS Session Manager plugin cho `aws ssm start-session`.
- Terraform 1.6+.
- Quyền tạo một A record cho subdomain Ollama (không phải domain API).
- Static IP hiện tại của box Lightsail.
- Quyền truy cập Lightsail host và biết quy trình deploy AWS hiện hữu.

Không cần và không được: SSH vào EC2, sửa `docker-compose.aws.yml`, Dockerfile, Nginx template,
`deploy-aws.yml`, hoặc thao tác trên VPS.

## 1. Phase A — Kiểm chứng local (bắt buộc trước EC2)

### A1. Cài và pin model

```powershell
ollama --version
ollama pull qwen3:4b
ollama list
ollama show qwen3:4b --modelfile
```

Ghi lại `ollama --version` và digest `sha256:...` lấy từ `ollama list` (hoặc output của `ollama
pull`). Hai giá trị này phải khớp với Terraform variables khi lên EC2.

**Đạt khi**: version và digest được ghi vào runbook; model load được.

### A2. Chạy contract test adapter

```powershell
go test ./shared/infrastructure/ai/... -count=1
go test ./shared/infrastructure/ai/openai/... -count=1
```

**Đạt khi**: toàn bộ contract test trong [contracts/ollama-openai-subset.md](./contracts/ollama-openai-subset.md)
§5 pass, gồm thinking off, structured output, SSE, usage thiếu/0, lỗi 4xx/429/5xx, cancellation.

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

Streaming và structured output: dùng cùng body với `stream = $true` và
`response_format = @{ type = "json_object" }`.

**Đạt khi**:

- Không có `reasoning_content`/thinking trong response.
- JSON structured parse được và đúng schema.
- SSE có `data: [DONE]`; usage (nếu có) được đọc đúng.
- Token `thinking_tokens = 0` theo telemetry.

### A4. Drill fallback local

Dùng local compose hiện hữu cho PostgreSQL/Redis/RabbitMQ; không sửa file compose AWS.

1. Chạy AI service trỏ primary `openai` về `http://127.0.0.1:11434/v1/chat/completions`, fallback
   Gemini.
2. Gọi một request text → phải do Ollama phục vụ.
3. Dừng Ollama (`Stop-Process -Name ollama` hoặc kill process đang phục vụ).
4. Gọi lại → phải fallback Gemini đúng một lần, không treo.
5. Bật Ollama lại → request trở lại Ollama không cần đổi cấu hình.

**Đạt khi**: đúng 1 attempt Ollama + tối đa 1 attempt Gemini; usage/cost gắn provider thắng; client
cancel không fallback; lỗi sau chunk đầu không fallback và trả lỗi trên stream.

### A5. Đo latency CPU tham chiếu

```powershell
$env:OLLAMA_NUM_GPU = "0"
# khởi động lại Ollama, sau đó chạy:
.\deployments\ollama\local-test.ps1
```

Kịch bản đo theo SC-005: 50 request cho mỗi capability `chat_text`, `free_text`,
`recommendation_text`, concurrency 2, cửa sổ 30 phút. Ghi TTFT, latency, output tokens, outcome và
fallback reason; lưu raw sample để tính p95. Thêm 5 request `summary` và 5 request `rewriter` để
xác nhận chúng đi theo route nghiệp vụ (`free_text` khi route free, `chat_text` khi route paid).

**Đạt khi**: có số liệu tham chiếu và raw sample. **Không** dùng số GPU local để thay SC-005; nếu
CPU local đã không đạt, phải điều chỉnh prompt/model trước khi lên EC2.

### A6. Cổng ra Phase A

Chỉ apply Terraform khi:

- [ ] Contract test pass.
- [ ] Thinking off được chứng minh.
- [ ] Structured output hợp lệ.
- [ ] Drill fallback pass.
- [ ] SSE chat không bị cắt: `AI_CHAT_CHUNK_DEADLINE_SECONDS=90` và primary timeout 90 đã được đặt
      trước khi chạy A4/A6.
- [ ] Recommendation: TTFT + thời gian sinh output nằm trong 90 giây, hoặc đã ghi nhận điều chỉnh/
      fallback trước khi đo.
- [ ] Digest/version đã ghi lại.

Nếu bất kỳ mục nào không đạt: dừng, không apply EC2; đổi model/prompt/digest và cập nhật
`spec.md`/`plan.md` trước khi thử lại.

## 2. Phase B — EC2 và Lightsail

### B1. Cấp Elastic IP và tạo DNS

```powershell
terraform -chdir=deployments/terraform/ollama-ec2 init
terraform -chdir=deployments/terraform/ollama-ec2 apply -target=aws_eip.ollama
terraform -chdir=deployments/terraform/ollama-ec2 output elastic_ip
```

Tạo tay A record `ollama.<domain>` trỏ tới IP vừa xuất. Sau đó:

```powershell
terraform -chdir=deployments/terraform/ollama-ec2 apply
```

**Đạt khi**: apply thành công, không đụng DNS của API, state nằm ngoài repo.

### B2. Kiểm tra EC2 qua SSM

```powershell
aws ssm start-session --target <instance_id>
```

Trong phiên SSM, kiểm tra:

- `systemctl status ollama nginx`
- `curl -s localhost:11434/api/version`
- `ollama list` khớp digest đã pin
- `ss -lntp` cho thấy Ollama chỉ `127.0.0.1:11434`
- `certbot certificates` còn hạn
- security group chỉ mở 443 cho `<lightsail-ip>/32`

**Đạt khi**: proxy TLS sẵn sàng, model đúng digest, thinking off, Ollama không public.

### B3. Bật trial trên Lightsail

Cập nhật `/opt/closy-aws/.env.aws` theo [contracts/env-ai-trial.md](./contracts/env-ai-trial.md)
§1, rồi chạy quy trình deploy AWS hiện hữu. Không sửa file deployment.

**Đạt khi**: đúng container AI được tạo lại; sáu container vẫn healthy; không có diff Dockerfile/
Compose/Nginx/pipeline.

### B4. Smoke test

- Gọi `chat_text`, `recommendation_text`, `free_text` qua API thật → provider thắng là `openai`.
- Gọi `summary` và `rewriter` → provider thắng cũng là `openai` (route free dùng `free_text`, route
  paid dùng `chat_text`; cả hai nhóm đặt `openai` trong trial).
- Gọi vision và embedding → provider thắng vẫn là `google`.
- Kiểm tra log có `provider`, `model`, `attempt_index`, `billing_class`, `cost_micro_vnd`.
- Kiểm tra `ai_usage_events`: ba đường text có `actual_cost_micro_vnd = 0`.

### B5. Kiểm tra cô lập truy cập

Từ máy ngoài allowlist:

```powershell
curl.exe -v https://<ollama-host>/v1/chat/completions
curl.exe -v http://<elastic-ip>:11434/api/version
```

**Đạt khi**: request từ ngoài allowlist bị chặn/timeout; `11434` không truy cập được; thiếu/sai
Bearer trả 401/403; request hợp lệ từ Lightsail trả 200.

### B6. Drill fallback và tải

- Stop Ollama qua SSM → gọi text → Gemini phục vụ đúng một lần → restart Ollama.
- Chạy đúng load profile SC-005: 50 request/capability, concurrency 2, cửa sổ 30 phút; lưu raw
  sample, tính p95 và phân loại success/fallback/failure do write timeout.
- Xác nhận không có attempt thứ ba và không có Gemini cost cho lượt Ollama thắng.

**Đạt khi**: SC-005 và SC-009 có số liệu; semaphore/Ollama queue hoạt động.

### B7. Drill rollback Gemini-only

Đổi ba `*_TEXT_PRIMARY_*` về Gemini như §1.5 của env contract, chạy lại deploy hiện hữu.

**Đạt khi**: trong ≤15 phút, text phục vụ bằng Gemini; sáu container healthy; không mất dữ liệu đã
xác nhận; không thao tác VPS.

## 3. Ngày thứ 7

- Không rollback/tắt/teardown tự động.
- Thu đủ số liệu theo [contracts/trial-telemetry.md](./contracts/trial-telemetry.md) §4.
- Điền `docs/operations/ollama-trial-report.md`: kết quả smoke, access/auth, lỗi, latency, fallback,
  queue, CPU/RAM/disk, chi phí AWS, quyết định `go`/`adjust`/`no-go`, người quyết định, thời điểm.
- Nếu chưa có quyết định: giữ nguyên cấu hình và tiếp tục quan sát.

## 4. Bằng chứng tối thiểu phải lưu

- [ ] Output contract test local.
- [ ] Version + digest model local và EC2.
- [ ] Terraform plan/apply output (không có secret).
- [ ] Ảnh/log `docker compose ps` sáu container.
- [ ] Log/metric provider thắng cho 3 text + vision/embedding.
- [ ] Bằng chứng chặn IP ngoài và `11434`.
- [ ] Drill fallback + rollback.
- [ ] Báo cáo go/no-go có chữ ký người quyết định.
