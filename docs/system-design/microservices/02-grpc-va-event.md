# 02 — gRPC, event và xác thực hai lớp

## gRPC liên service

- Transport: HTTP/2 (`--use-http2` trên Cloud Run), client dial `https://<svc>.run.app`
  với TLS + keepalive; Nginx **không** tham gia đường gRPC (nội bộ service↔service).
- Health: `grpc.health.v1` cho probe; heartbeat SSE 15s + `ai_chat_chunk_deadline_seconds=30s`
  (thống nhất mọi môi trường).
- Lỗi: map `apperror` ⇄ `google.rpc.Status` qua `contracts/grpcx`
  (`ToAppError` cho client, `ToGRPCError` cho server) — hai đầu thấy cùng error shape.

## Xác thực hai lớp (FR-034)

Mỗi gRPC call mang **hai metadata key tách biệt**:

| Lớp | Key | Nội dung | Kiểm tra ở đâu |
|-----|-----|----------|----------------|
| Hạ tầng | `authorization` | Google ID token (`run.invoker`) | Cloud Run IAM (từ chối nếu thiếu/sai) |
| Ứng dụng | `x-closy-jwt` | Closy JWT ES256 của user | Service đích verify bằng public key cục bộ |

Client gắn ID token qua interceptor (`contracts/grpcx`: provider theo `IsManagedEnv`,
cache theo audience, fail-fast khi mint lỗi); local/VPS là no-op (không IAM).
Xem chi tiết JWT ở `04-auth-asymmetric-jwt.md`.

## Event bất đồng bộ (giữ nguyên từ monolith)

- RabbitMQ cho tác vụ nặng/chậm (AI, thanh toán, đồng bộ) qua **transactional outbox**
  (`outbox_records` + worker phát lên broker, at-least-once).
- Worker nền là presentation adapter (`presentation/worker/`) điều phối theo lifecycle app.
- `LISTEN/NOTIFY` + migration + advisory lock dùng **direct 5432**, runtime dùng pooler 6543.

## Thứ tự đọc tiếp

- `03-adapter-va-composition.md` — inbound/outbound adapter, Wire
- `04-auth-asymmetric-jwt.md` — ES256 + revocation + ID token
