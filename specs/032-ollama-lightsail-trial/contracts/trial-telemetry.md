# Contract: Telemetry và báo cáo thử nghiệm

**Feature**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26

Hợp đồng này khóa dữ liệu vận hành cần thu trong tuần trial và cách ánh xạ sang báo cáo go/no-go.
Không có bảng mới; attempt detail nằm ở structured log/metric.

## 1. Structured log (Go AI Service)

Mỗi attempt ghi một dòng JSON với các khóa sau (tên cố định):

| Khóa | Kiểu | Bắt buộc | Ghi chú |
|------|------|----------|---------|
| `timestamp` | RFC3339 | Có | UTC |
| `level` | string | Có | `info`/`warn`/`error` |
| `service` | string | Có | `ai`/`collector` |
| `version` | string | Có | Image tag |
| `environment` | string | Có | `aws-course` |
| `request_id` | UUID | Có | Khóa liên kết usage event |
| `operation` | string | Có | AI operation: `chat`/`outfit`/`summary`/`rewriter`/`vision`/`embedding` |
| `capability` | string | Có | `chat_text`/`recommendation_text`/`free_text`/`vision`/`embedding` |
| `attempt_index` | int | Có | 1 hoặc 2 |
| `provider` | string | Có | `openai`/`google` |
| `model` | string | Có | qwen3:4b hoặc model Gemini |
| `billing_class` | string | Có | `unbilled`/`billed` |
| `outcome` | string | Có | Theo contract Ollama §4 |
| `http_status` | int | Không | Khi có response |
| `fallback_reason` | string | Không | Chỉ attempt 1 thất bại |
| `ttft_ms` | int | Không | Streaming |
| `latency_ms` | int | Có | Attempt |
| `prompt_tokens` | int64 | Không | 0 nếu không biết |
| `output_tokens` | int64 | Không | 0 nếu không biết |
| `thinking_tokens` | int64 | Không | Phải bằng 0 trong trial |
| `usage_reported` | bool | Có | Provider có trả usage không |
| `cost_micro_vnd` | int64 | Có | 0 với `openai`; actual với Gemini billed |

**Cấm tuyệt đối**: `prompt`, `system_prompt`, `messages`, `content`, `Authorization`, `token`,
`secret`. Chỉ log `input_characters` và `max_output_tokens`.

## 2. Metric (CloudWatch)

| Metric | Đơn vị | Labels | Ngưỡng cảnh báo gợi ý |
|--------|--------|--------|----------------------|
| `ai_ttft_ms` | ms | provider, model, capability | p95 > SC-005 tương ứng |
| `ai_latency_ms` | ms | provider, model, capability, outcome | p95 > 90s (Ollama) |
| `ai_fallback_total` | count | capability, fallback_reason | Tỷ lệ fallback > 20% trong 1 giờ |
| `ai_schema_failure_total` | count | provider, model, capability | > 0 trong 1 giờ |
| `ai_thinking_token_total` | count | provider, model | > 0 → trial không hợp lệ |
| `ai_cost_micro_vnd` | micro VND | provider, model | Theo dõi ngân sách |
| `ec2_cpu_percent` | % | instance | > 85% kéo dài 10 phút |
| `ec2_memory_bytes` | bytes | instance | > 90% RAM |
| `ec2_disk_free_bytes` | bytes | instance | < 10 GiB |
| `proxy_5xx_total` | count | host | > 0 kéo dài 5 phút |
| `proxy_401_total` | count | host | > 0 → cảnh báo token/allowlist |

Metric do CloudWatch agent trên EC2 thu cho log Nginx/Ollama; metric AI do Go AI Service phát qua
log metric filter hoặc exporter hiện hữu. Retention log 14 ngày.

## 3. Log nguồn trên EC2

| Nguồn | Đường dẫn | Nội dung |
|-------|-----------|----------|
| Nginx access | `/var/log/nginx/ollama-access.log` | IP, status, bytes, request time; **không** body |
| Nginx error | `/var/log/nginx/ollama-error.log` | Lỗi TLS/upstream |
| Ollama | journal `ollama` | Startup, model load, OOM; **không** prompt |
| Cloud-init | `/var/log/cloud-init-output.log` | Bootstrap, certbot, pin digest |

## 4. Ánh xạ sang báo cáo SC-007

| Mục báo cáo | Nguồn |
|-------------|-------|
| Smoke test 6 container | `docker compose ps` trên Lightsail + smoke script |
| Kiểm soát truy cập/xác thực | `proxy_401_total`, `proxy_5xx_total`, test từ IP ngoài |
| Lỗi | `outcome` breakdown trong log |
| Độ trễ AI | `ai_ttft_ms`, `ai_latency_ms` theo capability |
| Tỷ lệ fallback | `ai_fallback_total` / tổng request, theo `fallback_reason` |
| Hàng đợi | `OLLAMA_MAX_QUEUE`, log queue full, `429`/`503` |
| CPU/RAM/disk EC2 | metric CloudWatch |
| Chi phí AWS | AWS Cost Explorer + Budget 80 USD |
| Kết luận go/no-go | `TrialDecision` trong báo cáo |

## 5. Quy tắc dữ liệu

- Mọi dòng log phải có `request_id` để nối attempt ↔ usage event.
- Không log prompt/secret, kể cả ở mức debug.
- `thinking_tokens > 0` trên provider `openai` là lỗi cấu hình; phải dừng đo và sửa.
- Báo cáo chỉ dùng dữ liệu đã thu; không chỉnh ngưỡng sau khi bắt đầu đo mà không ghi lý do.
- Sau ngày 7, log vẫn tiếp tục thu cho đến khi có quyết định thủ công.
