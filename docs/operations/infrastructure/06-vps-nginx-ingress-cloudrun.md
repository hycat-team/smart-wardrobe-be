# 06 — VPS Nginx ingress tới Cloud Run

`api-v2.closy.hycat.online` phục vụ qua **VPS Nginx hiện hữu** (không Cloudflare/LB mới).
`api.closy...` **giữ nguyên** proxy tới monolith (`http://backend:8080`) ở giai đoạn song song.

Nginx render theo **`NGINX_MODE`** (`parallel`/`parallel-v2`/`repatriation`/`rollback`);
token-helper là service compose (profile `parallel` hoặc `parallel-v2`). Mục 11 mô tả
mode `parallel-v2` (api.closy→VPS 3-service, api-v2→Cloud Run, không monolith).

## 1. Điều kiện tiên quyết

- [ ] `00` §5 xong (secret) và `00` §6 xong (có `ACCOUNT_RUN_URL`/`COMMERCE_RUN_URL`/`AI_RUN_URL`).
- [ ] `/opt/closy` có `deployments/` + `.env.production` + `token-helper-sa.json` (`0600`).
- [ ] `deployer` SA `token-helper` đã được grant `run.invoker` trên 3 service (`00` §2.3).

## 2. Route map single-source

Mọi route `api-v2` → 3 Cloud Run nằm trong `deployments/nginx/`:
`templates/parallel/api-v2.conf.template` + `api-v2-routes/`. Sinh lại bằng:
```bash
make nginx-routes      # sinh route map từ swagger
make nginx-check       # verify route map phủ hết API paths
```

Các mode (`deployments/nginx/templates/<mode>/`):
`parallel` (api.closy→monolith, api-v2→Cloud Run) · `parallel-v2` (api.closy→3 service VPS,
api-v2→Cloud Run, không monolith — mục 11) · `repatriation` (api.closy→3 service,
api-v2→301) · `rollback` (api.closy→monolith, api-v2→301).

## 3. Cấu hình `.env.production`

Đặt tối thiểu (xem `08-environment-variables.md`):
```
API_DOMAIN=api.closy.hycat.online
API_V2_DOMAIN=api-v2.closy.hycat.online
NGINX_MODE=parallel
COMPOSE_PROFILES=parallel
ACCOUNT_RUN_URL=https://closy-account-xxx.run.app
COMMERCE_RUN_URL=https://closy-commerce-xxx.run.app
AI_RUN_URL=https://closy-ai-xxx.run.app
TOKEN_HELPER_URL=http://token-helper:8082
TOKEN_HELPER_SA_KEY_FILE=./token-helper-sa.json
```

## 4. Cấp cert Let's Encrypt

Do Nginx đang chạy (monolith), cấp cert bằng standalone (giải phóng port 80):
```bash
docker stop nginx || true
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  --profile maintenance run --rm certbot certonly --standalone \
  -d api.closy.hycat.online -d api-v2.closy.hycat.online --expand --agree-tos -m <email>
```

## 5. Bật ingress

```bash
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  up -d nginx token-helper
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml ps
```
Compose mount `./nginx/templates/${NGINX_MODE}` vào `/etc/nginx/templates`; entrypoint
nginx chạy `envsubst` để render conf. Đổi mode: sửa `NGINX_MODE` rồi:
```bash
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  up -d --force-recreate nginx
```
Nếu Nginx chạy native trên VPS (không container): `make nginx-reload NGINX_MODE=<mode>`.

## 6. DNS

Tạo bản ghi `api-v2.closy.hycat.online` (A/CNAME) → VPS **sau khi** cert + server block
sẵn sàng (chi tiết `11-dns-vercel-ci-cd.md`).

## 7. Xác thực Nginx → Cloud Run (FR-033/FR-039)

- Cloud Run IAM (`roles/run.invoker` cho SA token-helper; KHÔNG bật `allUsers`).
- Nginx dùng `auth_request` tới `http://token-helper:8082/token?audience=<run.app>`;
  helper trả header `X-Serverless-Authorization: Bearer <id token>`. Cloud Run strip
  header này trước khi tới container → `Authorization` (Closy JWT) giữ nguyên cho app.
- SA key file quyền `0600`; rotation = tạo key mới + redeploy helper + revoke key cũ.

Kiểm tra helper mint token:
```bash
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  exec nginx wget -qSO- "http://token-helper:8082/healthz"
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  exec nginx wget -qSO- "http://token-helper:8082/token?audience=$ACCOUNT_RUN_URL" 2>&1 \
  | grep -i x-serverless-authorization
```

## 8. Smoke test

```bash
# api.closy giữ nguyên (monolith)
curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health

# api-v2 qua Nginx
curl -sS -o /dev/null -w '%{http_code}\n' https://api-v2.closy.hycat.online/api/v1/system-catalog/wardrobe-items

# Cloud Run IAM đóng khi thiếu token
curl -sS -o /dev/null -w '%{http_code}\n' "$ACCOUNT_RUN_URL/api/v1/health"   # 403

# Không có Cloudflare trong đường API
curl -sSI https://api-v2.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"

# SSE stream dài không đứt (giữ kết nối)
curl -N -m 35 https://api-v2.closy.hycat.online/<sse-path>
```

## 9. Verify

- [ ] `api-v2` 200 qua Nginx↔Cloud Run; thiếu ID token bị Cloud Run từ chối (SC-013).
- [ ] `api.closy` 0 thay đổi routing (SC-015); 0 request `api-v2` qua Cloudflare (SC-016).
- [ ] SSE stream dài giữ kết nối, 0 lỗi buffering (SC-014).

## 11. Mode parallel-v2 (api.closy→VPS 3-service, api-v2→Cloud Run)

1. Điều kiện: đã vào được `parallel` ổn định; image 3 service đã có (`.env.deploy`).
2. Dừng backend cũ: `... stop backend || true`; verify `docker ps` trống backend.
3. Đặt `NGINX_MODE=parallel-v2`, `COMPOSE_PROFILES=parallel-v2`; template render từ
   `deployments/nginx/templates/parallel-v2/` (default + vps-3service + api-v2).
4. Bật ingress (kèm cả 2 env-file):
   ```bash
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
     up -d nginx token-helper
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml ps
   ```
5. Smoke:
   ```bash
   curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health          # 200 qua VPS
   curl -sS -o /dev/null -w '%{http_code}\n' https://api-v2.closy.hycat.online/api/v1/health        # 200 qua Cloud Run
   docker ps --filter name=backend --format '{{.Names}}'                                            # trống
   ```
6. DB: VPS local là live; dump VPS→Supabase (`10-data-migration-vps-supabase.md`) trước đợt
   verify `api-v2`; `api-v2` chỉ verify.

## 10. Residual risk (FR-054)

Nginx **không** rate-limit (chỉ body-limit/timeout/security-header); rate-limit nằm ở app
(Redis-backed). Không có WAF/OWASP đầy đủ sau khi bỏ Cloudflare — chấp nhận; có thể bổ
sung ModSecurity sau (ngoài phạm vi 020).
