# Contract: Attempt lifecycle và callback

**Feature**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26

Hợp đồng này khóa cách `shared/infrastructure/ai` báo cho `services/ai` biết một attempt sắp chạy
và đã kết thúc, để admission/usage/cost chạy đúng provider thực tế. Không có bảng mới.

## 1. Vị trí type

- Type callback nằm trong `shared/application/ai` (ví dụ `AttemptHooks`), không nằm trong
  `services/ai`, vì `text_service.go` là nơi biết attempt nào sắp gọi.
- `TextGenerationOptions` mang `Hooks *AttemptHooks` (nil = không dùng, giữ hành vi cũ).
- `AttemptMeta.Route` và `AttemptMeta.BillingClass` do orchestrator điền từ `plan.Route` và
  provider đã capture; `text_service` điền `Provider`, `Model`, `AttemptIndex`, `Operation`,
  `Capability`, `Timeout`, `RequestID`.
- `AttemptHooks` tối thiểu:

```go
type AttemptHooks struct {
    BeforeAttempt func(ctx context.Context, meta AttemptMeta) error
    AfterAttempt  func(ctx context.Context, meta AttemptMeta, result AttemptResult)
}
```

- `AttemptMeta`: `RequestID`, `AttemptIndex` (1/2), `Provider`, `Model`, `Operation`,
  `Capability`, `Route`, `BillingClass`, `Timeout`.
- `AttemptResult`: `Outcome`, `HTTPStatus`, `Usage`, `UsageReported`, `FallbackEligible`,
  `LatencyMs`, `TTFTMs`, `Error`.

## 2. Điểm gọi

| Đường | BeforeAttempt | AfterAttempt |
|-------|---------------|--------------|
| Non-stream text | Ngay trước `client.Do` của attempt | Sau khi parse xong hoặc lỗi |
| Stream text | Ngay trước `client.Do`, trước khi emit chunk đầu | Khi stream kết thúc/lỗi/cancel |
| Vision/embedding | Không dùng hooks (Gemini primary, timeout riêng) | Không dùng hooks |

- `BeforeAttempt` lỗi → **không gọi provider**, trả lỗi cho caller.
- `AfterAttempt` phải chạy cả khi lỗi/cancel; dùng `context.WithoutCancel` khi ghi usage.
- Với streaming: nếu đã emit chunk đầu, `AfterAttempt` ghi `outcome` lỗi và `fallback_eligible=false`.

## 3. Nghĩa vụ của orchestrator (services/ai)

1. `provider=openai`, attempt 1: `BeforeAttempt` gọi `AdmitUnbilledAttempt` (mirror admission hiện
   hữu: `observe_only` → daily attempt cap theo `logical_route=paid`; `strict` → không có attempt
   cap, luôn admit; **upsert** event theo `request_id`, `logical_route=plan.Route`, cost 0).
   `CostUseCase.Prepare` không ghi event cho `openai`; `AdmitUnbilledAttempt` là writer duy nhất.
   Gọi lại `BeforeAttempt` không được tạo event thứ hai và không được tăng cap lần nữa.
   Nếu `AdmitUnbilledAttempt` từ chối vì hết daily cap: giữ hành vi degrade hiện hữu — ghi event
   `logical_route=free`, không reserve, vẫn cho phép gọi Ollama; không trả lỗi 429 mới. Nếu Ollama
   lỗi eligible trong nhánh degraded, fallback Gemini ở free route, **không** gọi
   `PrepareBillableAttempt`, không billed và không đụng paid cap.
2. Ollama thất bại eligible: `AfterAttempt` đóng attempt; trước attempt 2 gọi
   `PrepareBillableAttempt` để khóa event theo `request_id`, lấy pricing, nâng cấp cùng event sang
   Gemini + reserved cost và cộng ledger.
3. `PrepareBillableAttempt` lỗi → không gọi Gemini; release/unknown theo quy tắc hiện hữu.
4. Kết thúc request: `Confirm` theo provider/model thắng; `openai` luôn cost 0.

## 4. Bất biến

- Tối đa 2 attempt text; không retry thêm.
- Không tạo event thứ hai cho cùng `request_id` (`ai_usage_events.request_id` UNIQUE).
- Callback không được log prompt/secret; chỉ field trong `contracts/trial-telemetry.md`.
- Cancellation/4xx (khác 429) không kích hoạt attempt 2.
- Vision/embedding không đi qua callback này và không gọi Ollama.
