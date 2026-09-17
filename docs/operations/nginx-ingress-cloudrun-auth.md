# Nginx ingress + Cloud Run auth (route map, token helper, IAM)

## Route map single-source (FR-028)

Mọi route `api-v2` → 3 Cloud Run nằm trong **một chỗ**: `deployments/nginx/`
(`api-v2.conf.template` + `api-v2-routes/`). `api.closy` giữ nguyên route tới
monolith (`http://backend:8080`). Route longest-prefix, xử lý chồng lấn
`/me/*`, `/admin/*`. Nguồn suy ra từ route đã tách theo service (T040–T042).

Thay đổi route map → render (`make nginx-reload`: `envsubst` + `nginx -t` +
reload) → smoke test đủ 3 service trước khi merge.

## Token helper (FR-039)

Tiến trình trên VPS (`deployments/token-helper/`, T071) sinh/cache/refresh
Google ID token (audience = URL `run.app` từng service) cho service account
least-privilege (`roles/run.invoker` trên đúng 3 service). Nginx dùng
`auth_request` + `auth_request_set` để inject `X-Serverless-Authorization`.

- SA key file quyền `0600`; rotation = tạo key mới + redeploy helper + revoke key cũ.
- Helper lỗi/hết hạn → `api-v2` 401/403 → health check + retry/backoff + alert.

## Smoke test IAM (FR-033/SC-013)

- Request **không** ID token hợp lệ → Cloud Run từ chối (403).
- Request có token → 200; `Authorization` (Closy JWT) tới app nguyên vẹn.
- `proxy_ssl_server_name on` + resolve động hostname `run.app` (không pin IP).

## Residual risk (FR-054)

Nginx KHÔNG rate-limit (chỉ body-limit/timeout/security-header); rate-limit nằm ở
app (Redis-backed). Không có WAF/OWASP đầy đủ sau khi bỏ Cloudflare — chấp nhận,
có thể bổ sung ModSecurity sau (ngoài phạm vi 020).
