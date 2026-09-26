# Realtime Transport Contract: Redis Pub/Sub cho SSE

**Feature**: `023-analyze-status-handling`
**Date**: 2026-09-26

Mục tiêu: đảm bảo sự kiện SSE từ tiến trình worker tới được client ở tiến trình http trong cả hai topology (VPS `APP_MODE=all`, AWS tách `http`/`worker`). Đặt sau interface để có thể thay bằng kho chia sẻ/kênh khác.

---

## 1. Interface (shared/application/sse)

```go
// Publisher phía producer (worker consumer) — chỉ đẩy sự kiện ra kênh chia sẻ.
type IRealtimePublisher interface {
    Publish(ctx context.Context, taskId string, data []byte) error
}

// Relay phía consumer (http process) — nhận từ kênh chia sẻ, forward vào SSEManager cục bộ.
type IRealtimeRelay interface {
    Start(ctx context.Context)
    Stop()
}
```

- `ISSEManager` (giữ nguyên chữ ký `Subscribe/Unsubscribe/Publish`) chịu trách nhiệm **fan-out cục bộ + buffer phát lại**.
- Consumer dùng `IRealtimePublisher` thay cho `ISSEManager.Publish`.
- Handler SSE dùng `ISSEManager.Subscribe/Unsubscribe`.

## 2. Kênh và payload

| Thuộc tính | Giá trị |
|-----------|---------|
| Channel | `sse:task:{taskId}` |
| Subscribe pattern (relay) | `sse:task:*` |
| Payload | bytes JSON của `SSEEventPayload` |

## 3. Luồng

```text
worker (consumer)                      http (relay + subscribers)
──────────────────                     ─────────────────────────────
build SSEEventPayload
  │
  ▼
IRealtimePublisher.Publish(taskId,bytes)
  │  Redis PUBLISH sse:task:{taskId}
  └──────────────► IRealtimeRelay subscriber (PSubscribe sse:task:*)
                        │
                        ▼
                   ISSEManager.Publish(taskId, bytes)  (buffer + fan-out)
                        │
                        ▼
                   client SSE channel
```

- `APP_MODE=all`: publisher và relay cùng tiến trình; publish → Redis → relay → local manager → subscriber (một lần, không trùng).
- `APP_MODE=http`: publisher không chạy (không có consumer); relay + subscriber chạy.
- `APP_MODE=worker`: publisher chạy; relay bỏ qua (không có subscriber HTTP).

## 4. Buffer phát lại

- `ISSEManager` giữ ring buffer tối đa `server.sse.replay_buffer_size` (default 100) event/`taskId`.
- `Subscribe(taskId)`:
  1. Tạo channel mới.
  2. Phát lại toàn bộ event trong buffer hiện có (FIFO) cho channel mới.
  3. Đăng ký channel để nhận event tiếp theo.
- Vượt N: loại event cũ nhất.
- Buffer bị loại theo size (vượt N) hoặc TTL (`server.sse.replay_buffer_ttl_seconds`, default 300). KHÔNG xóa ngay khi subscriber cuối cùng `Unsubscribe` — nhờ đó reconnect muộn (kể cả sau khi phiên kết thúc) vẫn phát lại được.

## 5. Lifecycle

- Relay là component riêng, **không** thuộc `bootstrap.Workers`.
- Khởi động trong `App.Run()` khi `mode ∈ {http, all}`; dừng khi shutdown.
- Ở `worker` mode: không start (tránh goroutine thừa).

## 6. Tương thích & vận hành

- Không đổi hợp đồng phía client (SSE payload giữ nguyên).
- Redis là dependency đã có; nếu Redis Pub/Sub lỗi, realtime có thể suy giảm (event mất) nhưng luồng nghiệp vụ (DB) không bị ảnh hưởng — client có thể refetch.
- Log lỗi publish/subscribe nhưng không chặn luồng nghiệp vụ.
