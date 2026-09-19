# 06 — VPS Nginx ingress (3 service)

`api.closy.hycat.online` phục vụ qua **VPS Nginx hiện hữu** (không Cloudflare/LB mới),
proxy tới 3 service compose (account/commerce/ai). Monolith (backend) đã dừng ở mode này.

Nginx render theo **`NGINX_MODE`** (`services`/`services-redirect`/`rollback`);
không còn token-helper, không còn Cloud Run (đã xóa toàn bộ ở GĐ1).

## 1. Điều kiện tiên quyết

- [ ] `00` §5 xong (secret VPS/GHCR) và image 3 service đã có (`.env.deploy` do
  `deploy-vps-services.yml` ghi — nginx `depends_on` 3 service nên thiếu là compose báo lỗi).
- [ ] `/opt/closy` có `deployments/` + `.env.production` (copy từ `.env.production.example`).

## 2. Route map single-source

Mọi route `api.closy` → 3 service nằm trong `deployments/nginx/`: vùng
`# BEGIN/END GENERATED ROUTES` của `templates/services/vps-3service.conf.template`
(+ bản sao ở `templates/services-redirect/`) và `local.conf`. Sinh lại bằng:
```bash
make nginx-routes      # sinh route map từ swagger
make nginx-check       # verify route map phủ hết API paths
```

Các mode (`deployments/nginx/templates/<mode>/`):
`services` (api.closy→3 service VPS, monolith dừng) ·
`services-redirect` (services + api-v2→301 về api.closy, chỉ dùng trong grace period
sau khi tắt box ngoài, xong quay về `services`) ·
`rollback` (api.closy→monolith, api-v2→301, khẩn cấp).

## 3. Cấu hình `.env.production`

Đặt tối thiểu (xem `08-environment-variables.md`):
```
API_DOMAIN=api.closy.hycat.online
API_V2_DOMAIN=api-v2.closy.hycat.online
NGINX_MODE=services
```
Không còn `COMPOSE_PROFILES`, `*_RUN_URL`, `TOKEN_HELPER_*`.

> Luật isolation: DB/Redis/RabbitMQ trên VPS luôn là tên service docker local
> (`postgres`/`redis`/`rabbitmq`) — cấm trỏ ra hostname managed (xem `01` §4).
> Chạy guard ở `00` §7 bước 3 trước mọi lệnh `up`.

## 4. Cấp cert Let's Encrypt

Do Nginx đang chạy (monolith), cấp cert bằng standalone (giải phóng port 80):
```bash
docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml stop nginx
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  --profile maintenance run --rm certbot certonly --standalone \
  -d api.closy.hycat.online --expand --agree-tos -m <email>
```
Cert `api-v2` chỉ cấp khi cần (mode `services-redirect`, DNS api-v2 đã về VPS) — thêm
`-d api-v2.closy.hycat.online` vào lệnh trên khi đó.

## 5. Vào mode `services` (cắt monolith → 3 service)

1. Deploy 3 service mới nhất: `gh workflow run deploy-vps-services.yml --ref main -f services=all`;
   smoke nội bộ từng service qua `docker exec` (xem `00` §7).
2. Dừng backend cũ: `docker compose ... stop backend || true`; verify `docker ps` trống backend.
3. Đặt `NGINX_MODE=services` trong `/opt/closy/.env.production`.
4. Recreate nginx (luôn kèm cả 2 env-file):
   ```bash
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
     up -d --force-recreate nginx
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml ps
   ```
   Compose mount `./nginx/templates/${NGINX_MODE}` vào `/etc/nginx/templates`; entrypoint
   nginx chạy `envsubst` để render conf.
   Nếu Nginx chạy native trên VPS (không container): `make nginx-reload NGINX_MODE=services`.
5. Smoke:
   ```bash
   curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health
   docker ps --filter name=backend --format '{{.Names}}'   # trống
   ```
   Rollback bất cứ lúc nào: `NGINX_MODE=rollback` + chạy `deploy-production.yml`
   (xem `00` §10).

## 6. DNS

`api.closy.hycat.online` (A/CNAME) → VPS: giữ nguyên, không đổi.
Bản ghi `api-v2.closy.hycat.online` chỉ tạo khi bật box ngoài (GĐ2, runbook `15`)
hoặc khi cần redirect grace (`services-redirect`) — chi tiết `11-dns-vercel-ci-cd.md`.

## 7. Smoke test

```bash
# api.closy qua 3 service VPS
curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health
curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/system-catalog/wardrobe-items

# Không có Cloudflare trong đường API
curl -sSI https://api.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"

# SSE stream dài không đứt (giữ kết nối)
curl -N -m 35 https://api.closy.hycat.online/<sse-path>
```

## 8. Verify

- [ ] `api.closy` 200 qua 3 service VPS; backend đã dừng.
- [ ] 0 request qua Cloudflare; SSE stream dài giữ kết nối, 0 lỗi buffering.
- [ ] VPS infra local-only (guard `00` §7 bước 3 pass).

## 9. Residual risk (FR-054)

Nginx **không** rate-limit (chỉ body-limit/timeout/security-header); rate-limit nằm ở app
(Redis-backed). Không có WAF/OWASP đầy đủ sau khi bỏ Cloudflare — chấp nhận; có thể bổ
sung ModSecurity sau (ngoài phạm vi 020).
