# 03 — Adapter và composition (Wire)

## Vị trí adapter (Ports & Adapters)

| Adapter | Hướng | Vị trí | Ghi chú |
|---------|-------|--------|---------|
| gRPC server | inbound | `services/<service>/internal/modules/<owner>/presentation/grpc/server.go` | Phụ thuộc **contract interface**, Wire bind interface → impl; không inject usecase trực tiếp |
| gRPC client | outbound | `contracts/<owner>/v1/client/` | Implements contract interface của owner; gọi qua mạng |
| In-process impl | local | `services/<service>/internal/modules/<owner>/presentation/contract/*_impl.go` | Implements contract interface; gọi usecase in-process |

Server không inject usecase trực tiếp vì `presentation/contract` là **port công khai**:
cả in-process impl lẫn gRPC server bind cùng interface → local và remote caller
nhận cùng API/hành vi.

## Cross-service transport mapping (A′)

Enum/const gắn entity **ở lại `domain/constants/`**. Mapper proto⇄DTO do **owner**
sở hữu tại `services/<service>/contract/<owner>/transport/` (ngoài `internal/`)
để cả gRPC server nội bộ và `contracts/.../client` dùng chung; client MUST NOT
tham chiếu enum domain. Xem `design-pattern.md` mục tương ứng.

## Composition root (Wire)

| Service | Injector | Nội dung |
|---------|----------|----------|
| Account | `services/account/internal/di.InitializeApp` | `AccountApp`: router/worker/contract 4 module + 5 gRPC server; host only |
| Commerce | `services/commerce/internal/di.InitializeApp` | App HTTP + `CommerceContracts` (8 client qua `NewCommerceContracts` + dial Account/AI) |
| AI | `services/ai/internal/di.InitializeApp` | App HTTP + gRPC AI + `AIContracts` (Subscription/Audit qua dial Account) |

- Connection gRPC dùng chung qua `contracts/grpcx.DialService` (TLS/plaintext theo
  scheme URL, keepalive, cleanup đóng conn).
- Regenerate: `make wire svc=<account|commerce|ai>` (hoặc `wire-all`).
- Monolith (`cmd/server`, `internal/`, `services/*/providers`) đã xoá — xem T034/T121.
