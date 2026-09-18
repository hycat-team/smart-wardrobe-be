# Kiến trúc 3 service (mô hình C — multi-module workspace)

> Nguồn chân lý giai đoạn 020: `specs/020-platform-migration/` + constitution v3.0.0.
> File `architecture.md` ở cấp trên vẫn mô tả modular monolith cũ và sẽ được cập nhật ở T111 —
> **không** dùng làm baseline cho 020.

- Account (`identity,subscription,admin,notification`) — `services/account/`
- Commerce Core (`wardrobe,fashion,brand,dashboard`) — `services/commerce/`
- AI Engine (`aiexecutor`) — `services/ai/`

Ranh giới service do compiler enforce qua `go.work` (mỗi service một Go module).
Chi tiết xem các tài liệu US5 (T057–T063):

- `00-tong-quan-3-service.md` (mô hình C, bỏ monolith)
- `01-ranh-gioi-va-contract.md` (bảng proto vs in-process)
- `02-grpc-va-event.md` (ID token + JWT hai lớp)
- `03-adapter-va-composition.md`
- `04-auth-asymmetric-jwt.md`
- `05-repo-va-ci.md`
- `06-lo-trinh-va-checklist.md`
