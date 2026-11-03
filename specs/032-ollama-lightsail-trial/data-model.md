# Data Model: Thử nghiệm Ollama trên EC2 và tính tiền Gemini-only

**Feature**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26

Tài liệu này mô tả dữ liệu của thử nghiệm. Điểm quan trọng: **không thêm bảng, không migration và
không thay đổi schema hiện hữu**. Trạng thái cuối của một request vẫn nằm trong
`ai_usage_events`; chi tiết từng attempt chỉ tồn tại ở structured log/metric.

## 1. Phạm vi lưu trữ

| Dữ liệu | Nơi lưu | Thay đổi |
|---------|---------|----------|
| Cấu hình endpoint/model/timeout/concurrency | env Lightsail + Terraform variables | Thêm biến, không thêm bảng |
| Bearer token proxy | AWS Secrets Manager (EC2); `.env.aws` (Lightsail) | Mới, không commit |
| Kết quả request cuối | `ai_usage_events` hiện hữu | Không đổi schema |
| Attempt/fallback/thinking/latency | CloudWatch Logs + metric | Mới, không có bảng |
| Quyết định go/no-go | `docs/operations/ollama-trial-report.md` | Mới, tài liệu |

Không có entity nghiệp vụ mới trong PostgreSQL. `AIProviderAttempt`, `TrialObservation` và
`TrialDecision` là mô hình khái niệm cho telemetry/báo cáo, không phải bảng.

## 2. Entities

### 2.1 OllamaEndpoint (cấu hình, không persist trong DB)

| Trường | Kiểu | Nguồn | Quy tắc |
|--------|------|-------|---------|
| `hostname` | string | Terraform output / A record | Subdomain riêng; không phải domain API |
| `elastic_ip` | string | Terraform output `elastic_ip` | Tĩnh trong tuần trial |
| `base_url` | string | env `*_TEXT_PRIMARY_ENDPOINT` | Phải là URL đầy đủ kết thúc `/v1/chat/completions` |
| `model` | string | env `*_TEXT_PRIMARY_MODEL` | Đúng `qwen2.5:3b-instruct-q4_K_M` |
| `ollama_version` | string | cloud-init variable | Pin version; không auto-update |
| `model_digest` | string | `ollama list` / output `ollama pull` | Pin `sha256:...`; so khớp trước khi bật trial |
| `bearer_secret_ref` | string | Secrets Manager ARN / env | Không xuất hiện trong log, repo hoặc workflow output |
| `allowed_cidr` | string | Terraform variable | `/32` của static IP box Lightsail; không dùng `0.0.0.0/0` cho 443 |
| `expected_hosts` | CSV | env `AI_OPENAI_TRIAL_EXPECTED_HOSTS` | Host được coi là Ollama trial; trống = tắt cảnh báo |
| `concurrency` | int | env AI | Mặc định 2; `> 0`; dùng chung cho mọi attempt text tới Ollama |
| `ollama_num_parallel` | int | EC2 env | Mặc định 2; khớp `concurrency` |
| `ollama_max_queue` | int | EC2 env | Hữu hạn, đủ cho tải 2 đồng thời |
| `primary_timeout_seconds` | int | env AI | 90 |
| `fallback_timeout_seconds` | int | env AI | 30 |
| `reasoning_effort` | string | env `AI_OPENAI_REASONING_EFFORT` | Trống trong tuần trial (model non-thinking); trống = không gửi |
| `trial_started_at` | timestamp | runbook | Mốc bắt đầu tính ngày 1 |
| `expires_on` | date | tag `ExpiresOn` | Ngày 7; chỉ nhắc báo cáo, không teardown |

**Validation**

- `base_url` phải parse được, scheme `https` và path `/v1/chat/completions`; ngoại lệ local drill:
  cho phép `http://127.0.0.1:<port>`/`http://localhost:<port>`.
- `model` + `ollama_version` + `model_digest` phải khớp giữa local và EC2 trước khi đo.
- `allowed_cidr` phải là `/32` và khớp static IP hiện tại của Lightsail.
- `primary_timeout_seconds + fallback_timeout_seconds <= 120` (write timeout hiện hữu).
- `primary_timeout_seconds <= ai_chat_chunk_deadline_seconds` để stream chat không bị cắt trước
  attempt Ollama; default ngoài trial là 30/30/30, trial đặt 90/30/90.
- Deadline áp theo provider: `openai` dùng `primary_timeout_seconds`; `google` primary giữ timeout
  hiện hữu của từng file config (production 30/45/30; local 30/30/30); vision/embedding không đổi.
- `concurrency >= 1`; giá trị mặc định 2 khi env trống.

### 2.2 AIProviderAttempt (runtime, telemetry only)

| Trường | Kiểu | Ghi chú |
|--------|------|---------|
| `request_id` | UUID | Khóa liên kết với `ai_usage_events.request_id` |
| `attempt_index` | int | 1 = primary Ollama, 2 = Gemini fallback; không có 3 |
| `provider` | enum | `openai` hoặc `google` |
| `model` | string | `qwen2.5:3b-instruct-q4_K_M` hoặc model Gemini thực tế |
| `operation` | enum nghiệp vụ | `chat`, `outfit`, `summary`, `rewriter`, `vision`, `embedding` |
| `capability` | enum cấu hình | `chat_text`, `recommendation_text`, `free_text`, `vision`, `embedding`; `summary`/`rewriter` là `free_text` khi route free, `chat_text` khi route paid |
| `route` | enum | `paid` hoặc `free` (route nghiệp vụ hiện hữu) |
| `billing_class` | enum | `unbilled` khi `provider=openai`; ngược lại theo capability như hiện hữu |
| `reasoning_effort` | string | Trống trong trial; lấy từ `AI_OPENAI_REASONING_EFFORT`, trống = không gửi |
| `started_at` / `first_token_at` / `finished_at` | timestamp | `first_token_at` rỗng với non-stream |
| `ttft_ms` | int | Chỉ có khi streaming |
| `latency_ms` | int | Từ started đến finished |
| `outcome` | enum | `success`, `timeout`, `transport`, `http_4xx`, `http_429`, `http_5xx`, `malformed`, `schema_invalid`, `max_tokens`, `canceled` |
| `http_status` | int | Khi có response |
| `fallback_eligible` | bool | Tính theo ma trận §4 |
| `usage_reported` | bool | False khi Ollama không trả `usage` |
| `prompt_tokens` / `output_tokens` / `thinking_tokens` | int64 | 0 + `usage_reported=false` khi không biết |

**Quy tắc**

- `thinking_tokens` phải bằng 0 trên đường `openai` trong trial vì model là non-thinking; nếu khác 0,
  trial chưa hợp lệ.
- `attempt_index` không vượt quá 2. Mọi retry thêm là vi phạm FR-023.
- Attempt thất bại không tạo chi phí tiền tệ; chỉ attempt thắng mới đi vào confirm.
- State machine 2 attempt và timeout 90/30 chỉ áp dụng cho **text**. Vision/embedding vẫn là
  attempt 1 = Gemini với timeout 20 giây hiện hữu; không đi qua nhánh Ollama.

### 2.3 OperationalUsageRecord (ánh xạ `ai_usage_events`)

Không thêm cột. Các trường hiện hữu được dùng như sau:

| Cột hiện hữu | Giá trị trong trial |
|--------------|---------------------|
| `request_id` | UUID của request |
| `provider` / `model` | Provider/model **thắng** (Ollama hoặc Gemini) |
| `operation` | AI operation nghiệp vụ (`chat`/`outfit`/`summary`/`rewriter`/`vision`/`embedding`) |
| `logical_route` | Giữ route nghiệp vụ (`plan.Route`); `provider=openai` trên route paid vẫn tiêu thụ daily attempt cap hiện hữu, route free/local không đổi; không thêm quota mới |
| `prompt_tokens` / `output_tokens` / `thinking_tokens` | Token biết được; 0 nếu provider không trả |
| `reserved_cost_micro_vnd` | 0 với `openai`; reserve Gemini trước khi gọi Gemini có tính phí |
| `actual_cost_micro_vnd` | 0 với `openai`; actual Gemini theo pricing hiện hữu |
| `status` | Reserved → Confirmed/Released/Unknown như hiện hữu |
| `error_code` | `pricing_config_missing` / `usage_unreported` hiện hữu nếu áp dụng |

**Quy tắc tiền tệ**

1. `provider=openai` (bất kể capability/route/usage): `reserved_cost = actual_cost = 0`; **không**
   gọi pricing repository, `ReserveBillableAttempt` hoặc `ConfirmBillableAttempt`; event giữ
   `logical_route = plan.Route` (trừ nhánh hết cap: ghi `logical_route=free` theo degrade hiện hữu)
   và route paid vẫn tiêu thụ daily attempt cap hiện hữu, không thêm quota mới.
2. `provider=google` và capability **không** nằm trong `AI_UNBILLED_CAPABILITIES`: reserve trước
   khi gọi, confirm actual sau khi có usage theo quy tắc hiện hữu.
3. `provider=google` và capability nằm trong `AI_UNBILLED_CAPABILITIES` (`free_text`, `embedding`
   trong trial): actual 0, giữ nguyên hành vi hiện hữu.
4. Fallback: event cuối ghi provider/model **thắng**; attempt Ollama thất bại không tạo cost.
   `PrepareBillableAttempt` khóa event hiện có theo `request_id` (`FOR UPDATE`), nâng cấp cùng event
   sang Gemini + reserved cost và cộng ledger; không tạo event thứ hai vì `request_id` UNIQUE.

### 2.4 TrialObservation (telemetry)

| Trường | Kiểu | Ghi chú |
|--------|------|---------|
| `timestamp` | RFC3339 | UTC |
| `metric` | string | `ai_ttft_ms`, `ai_latency_ms`, `ai_fallback_total`, `ai_schema_failure_total`, `ai_thinking_token_total`, `ai_cost_micro_vnd`, `ec2_cpu_percent`, `ec2_memory_bytes`, `ec2_disk_free_bytes`, `proxy_5xx_total`, `proxy_401_total` (nguồn chuẩn: `contracts/trial-telemetry.md`) |
| `labels` | map | `provider`, `model`, `capability`, `attempt`, `fallback_reason`, `outcome` |
| `value` | number | Giá trị đo |
| `incident_note` | string | Tùy chọn; không chứa prompt/secret |

**Cấm log**: prompt, system prompt, message content, Bearer token, secret ARN đầy đủ, thân request
Ollama. Chỉ log `input_characters`, `max_output_tokens`, `request_id`, provider/model, outcome.

### 2.5 TrialDecision (báo cáo ngày 7)

| Trường | Kiểu | Quy tắc |
|--------|------|---------|
| `decided_at` | timestamp | Sau ngày thứ 7 |
| `decided_by` | string | Danh tính người quyết định |
| `decision` | enum | `go`, `adjust`, `no-go` |
| `rationale` | text | Tham chiếu số liệu, không cảm tính |
| `evidence` | list | Link metric/log/report |
| `follow_ups` | list | Hạng mục còn thiếu trước production |

Không có `decision` thì cấu hình Ollama vẫn active; không có hành động teardown tự động.

## 3. Quan hệ

```text
OllamaEndpoint (config)
   │ 1
   │
   *  AIProviderAttempt ── request_id ── 1 OperationalUsageRecord (ai_usage_events)
   │                                          │
   │                                          └─ cost theo provider thắng
   │
   *  TrialObservation ── metric/labels ── được tổng hợp vào TrialDecision
```

- Một request có 1–2 attempt; chỉ attempt thắng ánh xạ tới cost của usage record.
- Một endpoint có nhiều attempt/observation; không lưu endpoint trong DB.
- TrialDecision tổng hợp observation theo ngày/capability/outcome.

## 4. State machine và ma trận fallback

### 4.1 Attempt

```text
admitted
  └─> started ──> success
        │
        ├─> timeout            (90s hết hạn, chưa emit chunk)  ──> eligible
        ├─> transport          (connect/DNS/TLS)               ──> eligible
        ├─> http_429                                        ──> eligible
        ├─> http_5xx                                        ──> eligible
        ├─> malformed/empty                                 ──> eligible
        ├─> schema_invalid     (trước khi trả content)       ──> eligible
        ├─> max_tokens         (finish_reason=length)        ──> eligible
        ├─> http_4xx (≠429)                                 ──> not eligible
        └─> canceled           (client hủy)                 ──> not eligible

eligible + chưa có attempt 2 ──> Gemini fallback (reserve trước khi gọi)
eligible + đã có attempt 2   ──> kết thúc lỗi, không retry thêm
```

Với streaming: nếu đã emit chunk đầu, lỗi được trả trên stream; **không** fallback/ghép nội dung.

### 4.2 Request

```text
request
  └─> admission
        ├─ provider=openai ──> AdmitUnbilledAttempt (kiểm tra daily cap theo logical_route=paid;
        │                       nếu hết cap: degrade free route; reserved=0, upsert theo request_id)
        └─ provider=google ──> Prepare (pricing + reserve) nếu capability billed;
                               PrepareBillableAttempt chỉ dùng để nâng cấp event openai → Gemini fallback

attempt 1 (Ollama) ── success ──> Confirm(unbilled) ──> response
                   └─ eligible failure ──> PrepareBillableAttempt(Gemini) ──> attempt 2
                   └─ eligible failure trên nhánh degraded free ──> Gemini free route,
                      không PrepareBillableAttempt, không billed
attempt 2 (Gemini) ── success ──> Confirm(actual) ──> response
                   └─ failure ──> Release/MarkUnknown theo quy tắc hiện hữu
```

## 5. Quy tắc kiểm chứng (từ FR/SC)

| Quy tắc | Nguồn |
|---------|-------|
| `openai` không chạm pricing/reserve, kể cả khi thiếu cấu hình giá | FR-013, FR-014, SC-002 |
| `openai` vẫn ghi token/provider/model để quan sát | FR-015, SC-002 |
| Gemini billed reserve trước, confirm sau | FR-016 |
| Fallback tối đa 1 lần, chỉ cho lỗi đủ điều kiện | FR-023, SC-009 |
| Client cancel/4xx không fallback | FR-023, SC-009 |
| Model non-thinking: không reasoning content/token, JSON hợp lệ | FR-026, SC-011 |
| Model/version/digest khớp local ↔ EC2 | FR-007, FR-026 |
| Không log prompt/secret | FR-020 |
| Concurrency 2, timeout 90/30 | FR-019, SC-005 |
| Không auto teardown ngày 7 | FR-024, SC-010 |

## 6. Không thay đổi

- Không migration Goose, không bảng mới, không cột mới.
- Không đổi `ai_usage_events` schema. `logical_route` vẫn giữ route nghiệp vụ; `provider=openai`
  chỉ miễn phần cost, vẫn tiêu thụ daily attempt cap hiện hữu, còn Gemini giữ route nghiệp vụ.
- Không thêm event/outbox mới; attempt detail không đi qua RabbitMQ.
- Không lưu secret, prompt hoặc response body vào telemetry.
