# 00 — Tổng quan 3 service (mô hình C, bỏ monolith)

SmartWardrobe Backend chạy dưới dạng **3 service triển khai độc lập** trên
multi-module Go workspace (`go.work`), thay thế modular monolith
(`cmd/server` đã xoá khỏi `main`, chỉ còn tag frozen `monolith-legacy`).

## Topology

| Service | Module Go | Module nghiệp vụ | Host gRPC | Gọi ra ngoài |
|---------|-----------|------------------|-----------|--------------|
| Account | `services/account` | identity, subscription, admin, notification | Identity, Subscription, Admin | **không** |
| Commerce Core | `services/commerce` | wardrobe, fashion, brand, dashboard | **không** | Identity, Subscription, Admin, AIOrchestrator (+ 3 reader) |
| AI Engine | `services/ai` | aiexecutor | AIOrchestrator, AIDashboard | Identity, Subscription, Admin |

Đồ thị gọi (DAG, không cycle): Commerce → Account, Commerce → AI, AI → Account.

## Module dùng chung

| Module | Vai trò |
|--------|---------|
| `shared` | Config, infrastructure (db/cache/messaging/search/media/ai), presentation dùng chung, DTO/event, pkg |
| `contracts` | Proto + generated stubs + gRPC client (`<owner>/v1/`), helper `grpcx` (lỗi, dial, auth) |

## Ranh giới

Ranh giới service do **compiler enforce**: module service không thấy package
implementation (`domain/entities`, `domain/repositories`, `infrastructure/**`,
`application/usecase/**`) của module thuộc service khác. Chi tiết xem
`01-ranh-gioi-va-contract.md`. Ngoại lệ duy nhất: `contract/*`, `dto/*`,
`domain/constants/*` được phép (Constitution §II).

## Tài liệu liên quan

- `01-ranh-gioi-va-contract.md` — bảng proto vs in-process
- `02-grpc-va-event.md` — gRPC, event, auth 2 lớp
- `03-adapter-va-composition.md` — adapter layout, Wire
- `04-auth-asymmetric-jwt.md` — JWT ES256 + ID token
- `05-repo-va-ci.md` — repo, CI, độc lập deploy
- `06-lo-trinh-va-checklist.md` — lộ trình, checklist
