# Nginx ingress + Cloud Run auth (route map, token helper, IAM)

## Route map single-source (FR-028)

Mọi route `api-v2` → 3 Cloud Run nằm trong **một chỗ**: `deployments/nginx/`
(`templates/parallel/api-v2.conf.template` + `api-v2-routes/`). `api.closy` giữ
nguyên route tới monolith (`http://backend:8080`). Route longest-prefix, xử lý
chồng lấn `/me/*`, `/admin/*`. Nguồn suy ra từ route đã tách theo service
(T040–T042).

Nginx render theo **`NGINX_MODE`** (`deployments/nginx/templates/<mode>/`):
`parallel` (api.closy→monolith, api-v2→Cloud Run), `repatriation`
(api.closy→3 service, api-v2→301), `rollback` (api.closy→monolith, api-v2→301).
Đặt trong `.env.production` (kèm `COMPOSE_PROFILES=parallel` khi cần token-helper).

Thay đổi route map → render (`make nginx-reload NGINX_MODE=<mode>`: `envsubst` +
`nginx -t` + reload) → smoke test đủ 3 service trước khi merge.

## Token helper (FR-039)

Tiến trình trên VPS (`deployments/token-helper/`) sinh/cache/refresh Google ID
token (audience = URL `run.app` từng service) cho service account least-privilege
(`roles/run.invoker` trên đúng 3 service), dùng `google.golang.org/api/idtoken`
với key JSON. Nginx dùng `auth_request` + `auth_request_set` để inject
`X-Serverless-Authorization` (giữ nguyên `Authorization` = Closy JWT).

- Chạy trong compose network (`profile parallel`), bind `:8082` (mọi interface,
  chỉ `expose` nội bộ, không publish); Nginx gọi `http://token-helper:8082`.
- `TOKEN_HELPER_SA_KEY_FILE` trỏ file key (mặc định `/run/secrets/token-helper-sa.json`),
  quyền `0600`; rotation = tạo key mới + redeploy helper + revoke key cũ.
- Helper lỗi/hết hạn → `api-v2` 401/403 → health check + retry/backoff + alert.

## Smoke test IAM (FR-033/SC-013)

- Request **không** ID token hợp lệ → Cloud Run từ chối (403).
- Request có token → 200; `Authorization` (Closy JWT) tới app nguyên vẹn.
- `proxy_ssl_server_name on` + resolve động hostname `run.app` (không pin IP).

## Residual risk (FR-054)

Nginx KHÔNG rate-limit (chỉ body-limit/timeout/security-header); rate-limit nằm ở
app (Redis-backed). Không có WAF/OWASP đầy đủ sau khi bỏ Cloudflare — chấp nhận,
có thể bổ sung ModSecurity sau (ngoài phạm vi 020).
