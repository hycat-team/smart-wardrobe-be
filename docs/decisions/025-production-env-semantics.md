# ADR 025 - Ngữ nghĩa `SERVER_ENV`: `vps` cũng là production

*   **Status**: Accepted
*   **Decision Date**: 2026-09-20
*   **Decision Maker**: Duck

## 1. Context

`SERVER_ENV` là **profile triển khai** (`local` / `vps` / `managed`), nhưng code lại
dùng nó như **cờ production**: router chỉ gọi `gin.SetMode(gin.ReleaseMode)` khi
`cfg.Server.Env == "production"`, `SwaggerLockMiddleware` no-op khi env khác
`production`, và `validateConfig` chỉ chặn CORS wildcard khi env là `production`.

Thực tế không môi trường triển khai nào đặt `SERVER_ENV=production`: VPS
(`.env.production`) và box ngoài AWS (`.env.aws`) đều dùng `SERVER_ENV=vps`. Hệ quả:

- Gin chạy `debug` mode ở mọi deployment (log `[GIN-debug] [WARNING] Running in
  "debug" mode` khi deploy box AWS là lúc phát hiện).
- `/swagger` + `/api-docs` **mở công khai** trên VPS và box ngoài, vì điều kiện
  enforce `SWAGGER_ACCESS_CODE` chưa từng đúng — mâu thuẫn với ADR 020 §2.5, vốn
  giả định Swagger đã bị chặn bằng access code.
- Guard CORS wildcard không bao giờ chạy.

Giá trị `env: production` trong `config.production.yaml` bị biến môi trường
`SERVER_ENV=vps` ghi đè trong `applyEnvOverrides`.

## 2. Decision

1.  Tách bạch hai khái niệm bằng helper `config.IsProductionEnv(env)`:
    trả `false` **chỉ** cho `local` / `development` / `dev`; mọi giá trị khác —
    kể cả `vps`, `production`, `managed`, chuỗi rỗng hay giá trị lạ — được coi là
    production (fail-closed).
2.  Áp helper vào toàn bộ các gate hardening: gin release mode (3 service router),
    `SwaggerLockMiddleware`, `validateConfig` (CORS wildcard), và
    `GlobalErrorHandler` (hiện `stackTrace`/`debugMessage` theo `!IsProductionEnv`).
3.  **Không đổi giá trị `SERVER_ENV` ở bất kỳ env file nào** — hành vi production
    được bật thuần bằng code, nên cả VPS lẫn box ngoài nhận cùng lúc mà không phải
    sửa cấu hình.

## 3. Consequences

- **Positive**: gin release mode đúng ở mọi deployment; Swagger gate hoạt động
  đúng như ADR 020 §2.5 mô tả; guard CORS wildcard được bật; máy dev để
  `SERVER_ENV=local` cũng thấy stack trace khi debug (trước đây chỉ đúng khi
  `SERVER_ENV=development`).
- **Negative**: `/swagger` trên VPS và box ngoài giờ yêu cầu `SWAGGER_ACCESS_CODE`
  (dev nội bộ phải nhập code) — chấp nhận, đây là lớp obscurity có chủ đích. Env lạ
  (`staging`...) mặc định bị coi là production; nếu cần nới, thêm giá trị vào
  `nonProductionEnvs`.
- **Follow-up**: ADR 020 §2.5 được đính chính bởi ADR này (không sửa trực tiếp ADR
  020, theo đúng dặn dò §4 của nó). Khi hệ thống ổn định post-launch, cân nhắc
  trigger reconsider của ADR 020: chỉ đăng ký swagger group khi env không phải
  production.
