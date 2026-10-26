# Contract: Ollama qua tập con OpenAI-compatible

**Feature**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26

Hợp đồng này là nguồn sự thật cho adapter `shared/infrastructure/ai/openai` khi trỏ tới proxy
Ollama. Adapter hiện POST nguyên `endpoint` nên endpoint phải là URL đầy đủ.

## 1. Kênh gọi

| Thành phần | Hợp đồng |
|------------|----------|
| Public endpoint | `https://<ollama-host>/v1/chat/completions` |
| Auth | `Authorization: Bearer <token>` |
| TLS | Bắt buộc; chứng chỉ công khai (Let's Encrypt) |
| Cổng 443 | Chỉ chấp nhận nguồn `<lightsail-static-ip>/32` |
| Cổng 80 | Chỉ ACME challenge và redirect sang 443; không proxy suy luận |
| Cổng 11434 | Chỉ `127.0.0.1`; không truy cập được từ Internet |
| Token sai/thiếu | `401`/`403` tại proxy, không chuyển tiếp tới Ollama |

## 2. Request

```json
{
  "model": "qwen3:4b",
  "messages": [
    {"role": "system", "content": "<system prompt>"},
    {"role": "user", "content": "<user prompt>"}
  ],
  "stream": false,
  "max_tokens": 500,
  "temperature": 0,
  "response_format": {"type": "json_object"},
  "reasoning_effort": "none"
}
```

| Trường | Bắt buộc | Quy tắc |
|--------|----------|---------|
| `model` | Có | `qwen3:4b` |
| `messages` | Có | `system` + `user`; không gửi history ngoài phạm vi hiện hữu |
| `stream` | Có | `false` cho non-stream, `true` cho SSE |
| `stream_options.include_usage` | Khi stream | Phải gửi để nhận usage nếu có |
| `max_tokens` | Khi > 0 | Giữ nguyên `MaxOutputTokens` hiện hữu |
| `temperature` | Không | Phải truyền cả `0` (không được bỏ qua giá trị 0) |
| `response_format` | Khi cần JSON | `json_schema` nếu có `ResponseSchema`, ngược lại `json_object` |
| `reasoning_effort` | Không | Chỉ gửi khi `AI_OPENAI_REASONING_EFFORT` khác rỗng; trial đặt `"none"` |
| `format` | Không | Không dùng; chỉ dùng `response_format` qua `/v1` |

## 3. Response

### 3.1 Non-stream

```json
{
  "choices": [{"message": {"content": "..."}, "finish_reason": "stop"}],
  "usage": {"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150}
}
```

- `choices` rỗng hoặc `content` rỗng → lỗi `malformed` (eligible fallback).
- `usage` có thể vắng hoặc bằng 0 → `usage_reported=false`, không suy ra token từ prompt.
- `finish_reason=length` → outcome `max_tokens`, là lỗi output và **eligible fallback** (một lần);
  không coi là thành công, không trả content đã bị cắt.
- Nếu `reasoning_content`/thinking xuất hiện → trial **không hợp lệ**; adapter không trả nó cho
  client và contract test fail.

### 3.2 Streaming (SSE)

```text
data: {"choices":[{"delta":{"content":"Xin"}}]}
data: {"choices":[{"delta":{"content":" chào"}}]}
data: {"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}
data: [DONE]
```

- Chỉ parse dòng `data: `.
- `[DONE]` kết thúc stream, không phải lỗi.
- Usage có thể chỉ xuất hiện ở chunk cuối; không có usage thì vẫn success.
- Sau khi đã emit chunk đầu, lỗi không fallback; trả lỗi trên stream.
- Không có `choices[0].delta.content` và không có usage → coi là stream rỗng (`malformed`).

## 4. Ánh xạ lỗi

| Tình huống | Outcome | Fallback? |
|------------|---------|-----------|
| `http.StatusOK` + body hợp lệ | `success` | Không |
| `400`, `401`, `403`, `404`, `422` | `http_4xx` | Không |
| `429` | `http_429` | Có |
| `500`, `502`, `503`, `504` | `http_5xx` | Có |
| Timeout deadline 90 giây | `timeout` | Có |
| DNS/TCP/TLS/connection reset | `transport` | Có |
| JSON sai/empty/thiếu `choices` | `malformed` | Có |
| JSON đúng nhưng sai schema | `schema_invalid` | Có |
| `finish_reason=length` | `max_tokens` | Có |
| Client hủy (`context.Canceled`) | `canceled` | Không |
| Proxy trả 401/403 do token | `http_4xx` | Không |

Lưu ý: 401/403 từ proxy nghĩa là lỗi cấu hình token, không phải lỗi client; vẫn không fallback vì
Gemini không sửa được cấu hình. Phải cảnh báo vận hành.

## 5. Ma trận contract test (FR-012)

| Kịch bản | Kỳ vọng |
|----------|---------|
| Non-stream JSON hợp lệ | Trả content; usage stamp provider/model |
| Structured `json_object` | Parse JSON, validate theo schema |
| Structured `json_schema` | Gửi đúng schema; validate trước khi trả |
| Thinking off | Không có `reasoning_content`/thinking token; TTFT không bị đội |
| SSE nhiều chunk + `[DONE]` | Ghép đúng thứ tự, không lặp |
| SSE usage-only ở cuối | Success, usage ghi nhận |
| Zero usage / missing usage | Success, `usage_reported=false` |
| Body malformed | `malformed`, eligible fallback |
| Schema sai | `schema_invalid`, eligible fallback |
| `finish_reason=length` | `max_tokens`, eligible fallback |
| `429` / `5xx` | Eligible fallback |
| `4xx` khác 429 | Không fallback |
| Client cancel giữa stream | Không fallback, không ghi cost |
| Endpoint thiếu/sai Bearer | 401/403, không fallback, cảnh báo |
| Model/version/digest khác pin | Fail preflight, không bắt đầu đo |

## 6. Không thuộc hợp đồng

- Vision và embedding: vẫn Gemini; không gọi Ollama.
- OpenAI Responses API, embeddings, tools/function calling: không dùng trong trial.
- Native Ollama `/api/chat`, `/api/generate`: không dùng trong business code.
