# Contract: Env AI cho tuần thử nghiệm

**Feature**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26

Hợp đồng này khóa các biến môi trường và output hạ tầng. Không có secret nào được commit; mọi giá
trị nhạy cảm chỉ nằm trong `.env.aws` trên Lightsail và AWS Secrets Manager trên EC2.

## 1. Lightsail `.env.aws` — AI service

### 1.1 Ba capability text

Áp dụng cho `CHAT_TEXT`, `RECOMMENDATION_TEXT`, `FREE_TEXT`. Giá trị dưới đây là hợp đồng, không
phải mẫu gợi ý.

| Biến | Giá trị trial | Ghi chú |
|------|---------------|---------|
| `<CAP>_TEXT_PRIMARY_PROVIDER` | `openai` | Bắt buộc; adapter OpenAI-compatible |
| `<CAP>_TEXT_PRIMARY_API_KEY` | `<bearer-token>` | Trùng token proxy; không log |
| `<CAP>_TEXT_PRIMARY_ENDPOINT` | `https://<ollama-host>/v1/chat/completions` | URL đầy đủ, adapter không nối path |
| `<CAP>_TEXT_PRIMARY_MODEL` | `qwen3:4b` | Phải khớp digest đã pin |
| `<CAP>_TEXT_FALLBACK_PROVIDER` | `google` | Gemini fallback |
| `<CAP>_TEXT_FALLBACK_API_KEY` | `<gemini-key-hiện-hữu>` | Lấy từ cấu hình Gemini hiện tại |
| `<CAP>_TEXT_FALLBACK_ENDPOINT` | `<gemini-endpoint-hiện-hữu>` | Giữ nguyên giá trị đang chạy |
| `<CAP>_TEXT_FALLBACK_MODEL` | `<gemini-text-model-hiện-hữu>` | Giữ nguyên model Gemini đang dùng |

`<CAP>` nhận `CHAT_TEXT`, `RECOMMENDATION_TEXT`, `FREE_TEXT`.

### 1.2 Vision và embedding

Không đổi. Giữ nguyên provider/endpoint/model hiện hữu (Gemini). Trial không được trỏ vision hoặc
embedding sang endpoint Ollama.

### 1.3 Điều khiển trial

| Biến | Giá trị | Kiểu | Ghi chú |
|------|---------|------|---------|
| `AI_UNBILLED_CAPABILITIES` | `free_text,embedding` | CSV | Giữ hành vi hiện hữu; **không** dùng để miễn phí OpenAI |
| `AI_TEXT_MAX_CONCURRENCY` | `2` | int | Semaphore trong Go AI Service; mặc định 2 nếu trống |
| `AI_TEXT_PRIMARY_TIMEOUT_SECONDS` | `90` | int | Attempt `openai`/Ollama (trial); default code 30 |
| `AI_TEXT_FALLBACK_TIMEOUT_SECONDS` | `30` | int | Attempt Gemini fallback; tổng hai timeout ≤ 120 |
| `AI_OPENAI_REASONING_EFFORT` | `none` | string | Gửi `reasoning_effort` cho provider `openai`; trống = không gửi field |
| `AI_CHAT_CHUNK_DEADLINE_SECONDS` | `90` | int | Override field hiện hữu `server.sse.ai_chat_chunk_deadline_seconds`; default 30 ở nơi khác |
| `AI_OPENAI_TRIAL_EXPECTED_HOSTS` | `<ollama-host>` | CSV | Host được coi là Ollama trial; trống = tắt cảnh báo cấu hình sai |

### 1.4 Validation

- `PROVIDER` chỉ nhận `openai` hoặc `google`.
- `ENDPOINT` phải là URL `https` và kết thúc bằng `/v1/chat/completions` khi provider là `openai`;
  ngoại lệ local drill: chỉ nới **scheme** cho `http://127.0.0.1:<port>` hoặc
  `http://localhost:<port>`, path `/v1/chat/completions` vẫn bắt buộc.
- `AI_TEXT_PRIMARY_TIMEOUT_SECONDS + AI_TEXT_FALLBACK_TIMEOUT_SECONDS <= 120`.
- `AI_TEXT_PRIMARY_TIMEOUT_SECONDS <= AI_CHAT_CHUNK_DEADLINE_SECONDS`.
- `AI_TEXT_MAX_CONCURRENCY >= 1`.
- `AI_OPENAI_REASONING_EFFORT` chỉ nhận `none`, `low`, `medium`, `high` hoặc trống.
- `AI_CHAT_CHUNK_DEADLINE_SECONDS` > 0.
- Default code ngoài trial: primary 30, fallback 30, chunk 30; trial đặt 90/30/90.
- `AI_OPENAI_TRIAL_EXPECTED_HOSTS` là CSV host (không gồm scheme/path); trống = không cảnh báo.
- `AI_OPENAI_REASONING_EFFORT` map vào `reasoning_effort` của **cả ba** block `*_text_primary` qua
  YAML alias; block text fallback để trống.
- `provider=openai` trên route paid vẫn tiêu thụ daily attempt cap hiện hữu; route free/local
  không đổi; contract này không thêm quota mới.
- Thiếu biến mới thì dùng default trong code; không được để empty string làm config parse lỗi.

### 1.5 Rollback về Gemini-only

Đổi ba `*_TEXT_PRIMARY_*` về cấu hình Gemini hiện hữu (provider `google`, cùng key/endpoint/model
với fallback), giữ nguyên `AI_UNBILLED_CAPABILITIES`, không cần đổi Compose/Nginx/Dockerfile.

## 2. EC2 bootstrap env

Đặt qua Terraform variables/cloud-init, không commit secret.

| Biến | Giá trị mặc định | Ghi chú |
|------|------------------|---------|
| `OLLAMA_HOST` | `127.0.0.1:11434` | Chỉ loopback |
| `OLLAMA_MODEL` | `qwen3:4b` | Pin tag |
| `OLLAMA_MODEL_DIGEST` | `sha256:<digest>` | So khớp sau khi pull |
| `OLLAMA_VERSION` | `<version-pin>` | Cài đúng version, không auto-update |
| `OLLAMA_NUM_PARALLEL` | `2` | Khớp concurrency phía Go |
| `OLLAMA_MAX_QUEUE` | `8` | Hữu hạn; tránh queue vô hạn |
| `OLLAMA_KEEP_ALIVE` | `30m` | Giữ model nóng trong tuần trial |
| `PROXY_HTTPS_PORT` | `443` | TLS từ Lightsail |
| `PROXY_ACME_PORT` | `80` | Chỉ ACME challenge/redirect |
| `PROXY_ALLOWED_CIDR` | `<lightsail-static-ip>/32` | Không dùng `0.0.0.0/0` cho 443 |
| `PROXY_BEARER_SECRET_ARN` | `<secret-arn>` | Secrets Manager; instance profile chỉ đọc secret này |

## 3. Terraform outputs (hợp đồng bàn giao)

| Output | Kiểu | Dùng cho |
|--------|------|----------|
| `instance_id` | string | SSM Session Manager |
| `elastic_ip` | string | Tạo A record tay |
| `proxy_hostname` | string | env endpoint Lightsail |
| `secret_arn` | string | Kiểm tra quyền đọc secret |
| `ollama_version` | string | Đối chiếu version pin |
| `model_digest` | string | Đối chiếu digest local/EC2 |
| `security_group_id` | string | Kiểm tra allowlist |

## 4. Ràng buộc bất biến

- Không sửa `.github/workflows/deploy-aws.yml`, `deployments/docker-compose.aws.yml`, Dockerfile
  hoặc `deployments/nginx/templates/aws/**`.
- Không đưa secret vào repo, manifest, log workflow, Terraform state được commit hoặc output CI.
- Không đổi DNS của domain API; chỉ thêm A record cho subdomain Ollama.
- Không thêm biến nào nhắm tới VPS.
