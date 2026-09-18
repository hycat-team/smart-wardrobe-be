# Nginx ingress + Cloud Run auth (route map, token helper, IAM)

> **Khi nào dùng**: trong giai đoạn song song, khi dựng/đổi ingress `api-v2` hoặc
> token-helper. Là tài liệu chi tiết cho `infrastructure/00` §6–§7 và `infrastructure/06`.
> **Điều kiện**: đã deploy Cloud Run (có URL `run.app`) và VPS có `/opt/closy` +
> `.env.production`.
> **Không dùng khi**: chưa có URL Cloud Run, hoặc đang ở mode `repatriation`/`rollback`
> (không chạy token-helper).

## 1. Route map single-source (FR-028)

Mọi route `api-v2` → 3 Cloud Run nằm trong **một chỗ**: `deployments/nginx/`
(`templates/parallel/api-v2.conf.template` + `api-v2-routes/`). `api.closy` giữ nguyên
route tới monolith (`http://backend:8080`). Route longest-prefix, xử lý chồng lấn
`/me/*`, `/admin/*`.

**Quy trình đổi route**:
```bash
make nginx-routes     # sinh lại route map từ swagger
make nginx-check      # verify phủ hết API paths
# trên VPS:
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  up -d --force-recreate nginx
# smoke đủ 3 service trước khi merge
```

Nginx render theo `NGINX_MODE` (`deployments/nginx/templates/<mode>/`): `parallel`
(api.closy→monolith, api-v2→Cloud Run), `parallel-v2` (api.closy→3 service VPS,
api-v2→Cloud Run, không monolith), `repatriation` (api.closy→3 service, api-v2→301),
`rollback` (api.closy→monolith, api-v2→301). Đặt trong `.env.production` (kèm
`COMPOSE_PROFILES=parallel` hoặc `parallel-v2` khi cần token-helper).

Mode `parallel-v2` dùng thư mục `deployments/nginx/templates/parallel-v2/`
(default + vps-3service cho api.closy + api-v2 copy từ parallel); block VPS của nó đã
đăng ký vùng generated trong `make nginx-routes`/`nginx-check` (chi tiết `06` §11).

## 2. Token helper (FR-039)

Tiến trình trên VPS (`deployments/token-helper/`) sinh/cache/refresh Google ID token
(audience = URL `run.app` từng service) cho service account least-privilege
(`roles/run.invoker` trên đúng 3 service), key JSON `0600`.

- Chạy trong compose network (profile `parallel`), `expose` `:8082` nội bộ (không publish);
  Nginx gọi `http://token-helper:8082/token?audience=<run.app>`.
- Nginx dùng `auth_request` + `auth_request_set` để inject
  `X-Serverless-Authorization` (giữ nguyên `Authorization` = Closy JWT).
- `TOKEN_HELPER_SA_KEY_FILE` trỏ file key (mặc định `/run/secrets/token-helper-sa.json`),
  quyền `0600`.

**Rotation SA key**:
```bash
gcloud iam service-accounts keys create new.json \
  --iam-account=token-helper@<PROJECT>.iam.gserviceaccount.com
# thay file + chmod 600 trên VPS, recreate token-helper
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  up -d --force-recreate token-helper
gcloud iam service-accounts keys delete <OLD_KEY_ID> \
  --iam-account=token-helper@<PROJECT>.iam.gserviceaccount.com
```
Helper lỗi/hết hạn → `api-v2` 401/403 → health check + retry/backoff + alert.

## 3. Smoke test IAM (FR-033/SC-013)

```bash
# Không ID token hợp lệ -> Cloud Run từ chối 403
curl -sS -o /dev/null -w '%{http_code}\n' "$ACCOUNT_RUN_URL/api/v1/health"   # 403

# Token helper mint được
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  exec nginx wget -qSO- "http://token-helper:8082/token?audience=$ACCOUNT_RUN_URL" 2>&1 \
  | grep -i x-serverless-authorization

# Có token -> 200; Authorization (Closy JWT) tới app nguyên vẹn
TOKEN=$(docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  exec -T nginx wget -qSO- "http://token-helper:8082/token?audience=$ACCOUNT_RUN_URL" 2>&1 \
  | grep -i x-serverless-authorization | awk '{print $2" "$3}' | tr -d '\r')
curl -sS -o /dev/null -w '%{http_code}\n' -H "X-Serverless-Authorization: $TOKEN" \
  "$ACCOUNT_RUN_URL/api/v1/health"   # 200
```
`proxy_ssl_server_name on` + resolve động hostname `run.app` (không pin IP).

## 4. Residual risk (FR-054)

Nginx KHÔNG rate-limit (chỉ body-limit/timeout/security-header); rate-limit nằm ở app
(Redis-backed). Không có WAF/OWASP đầy đủ sau khi bỏ Cloudflare — chấp nhận; có thể bổ
sung ModSecurity sau (ngoài phạm vi 020).
