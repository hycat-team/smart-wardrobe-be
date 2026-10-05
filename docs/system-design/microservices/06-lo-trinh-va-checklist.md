# 06 — Lộ trình tách service và checklist

## Lộ trình đã thực hiện (US5)

1. **Config dùng chung** (Phase 2): `shared/config` + env override + validation
   (`APP_MODE`, ES, pooler/listener, TLS, JWT, ingress, SSE).
2. **Proto + stubs** (T026–T030): 4 domain (`identity`, `subscription`, `admin`, `ai`)
   → `contracts/<owner>/v1/` + `buf-gen`; generated code commit.
3. **Tách module** (T031–T033): 9 module về `services/<svc>/internal/modules/`;
   `shared/` tách khỏi `internal/`; 24→toàn bộ migration lịch sử về Account.
4. **Adapter gRPC** (T035–T039): 4 client (`contracts/.../client`) + 4 server
   (`presentation/grpc`, phụ thuộc contract interface) + `contracts/grpcx`
   (lỗi 2 chiều, dial, ID-token provider).
5. **Xoá monolith** (T034, sau T097→T097c): xoá `cmd/`, `internal/`, `api/`,
   module root; Makefile/CI dọn legacy; `token-helper` tách module.
6. **Ranh giới** (T121, gắn với T034): xoá `services/*/providers/` (transitional);
   composition per-service tự chứa; verify 0 import xuyên service.
7. **Runtime** (T034a/T034b): bootstrap 3 service + `APP_MODE` + worker singleton
   + advisory lock job không idempotent.
8. **Auth** (T044–T049a): bỏ origin-auth, ES256 + legacy, revocation Redis,
   rate-limit Redis-backed.

## Checklist vận hành 3 service

- [ ] `go work sync` pass; `make build-all` xanh 3 binary; `cmd/server` không tồn tại.
- [ ] Import implementation xuyên service → build fail (thử import entity/repo/infra/usecase).
- [ ] Mỗi service `wire` regenerate sạch (`make wire-all`), `wire_gen.go` commit.
- [ ] Contract test (in-process + gRPC) pass; idempotency quota/audit pass.
- [ ] Swagger per-service generate được; hợp 3 service đủ 9 section (CI parity).
- [ ] Mỗi service khởi động với `migrate_on_start` + bảng version riêng, không chạy lại lịch sử.
- [ ] Deploy độc lập từng service (không rebuild service khác); pin SHA/digest.
- [x] Rollback monolith-legacy: **bỏ** theo ADR 012 (tag vẫn giữ làm tham chiếu;
  profile `rollback` chỉ dùng thủ công, không còn workflow tự động).
