# 04 — Xác thực JWT bất đối xứng (ES256) + ID token

## Hai lớp độc lập (FR-022/FR-034)

| Lớp | Cơ chế | Key giữ ở đâu |
|-----|--------|---------------|
| Ứng dụng | JWT **ES256**: identity ký bằng private key; service khác verify bằng public key cục bộ (không gọi mạng) | Private key: chỉ identity; public key: mọi service |
| Hạ tầng | Google ID token (`run.invoker`) ở metadata gRPC / header `X-Serverless-Authorization` (HTTP qua Nginx) | Token helper (VPS) hoặc metadata server (Cloud Run) |

Giai đoạn chuyển tiếp hỗ trợ legacy **HS256** (fallback khi ES256 không áp dụng được);
thu hồi phiên tra cứu qua **Redis blacklist dùng chung** (không gọi lại identity
mỗi request) — FR-023.

## Triển khai

- Ký/verify: `shared/pkg/utils/jwtutils` (`GenerateTokenES256`, `LoadECDSA*Key`,
  `ValidateTokenFlexible` với peek `alg` → dispatch ES256/HS256-legacy; fail-closed
  khi thiếu public key).
- Middleware: `shared/presentation/middleware/auth.go` (`Handle` bắt buộc,
  `OptionalHandle` tuỳ chọn) — dùng chung mọi service; blacklist qua
  `ITokenBlacklistService` (`shared/application/interface/security`) với impl
  Redis duy nhất (`shared/infrastructure/security`).
- Config: `jwt.algorithm` (`HS256`→`ES256`), `jwt.private_key`/`jwt.public_key`
  (PEM path), `ingress.*` (base URL/audience/header).

## Quy tắc

- Private key KHÔNG rời identity; public key phân phối qua config (KHÔNG log/commit).
- Mọi service verify cục bộ; revocation tra Redis chung.
- gRPC client gắn ID token ở metadata hạ tầng; JWT app ở `x-closy-jwt` (xem `02-grpc-va-event.md`).
