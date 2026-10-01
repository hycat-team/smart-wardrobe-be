# Runbook VPS — monolith → 3 service (GĐ1)

Đưa VPS từ monolith sang 3 service compose (`NGINX_MODE=services`), đọc xuôi từ
trên xuống, không bỏ bước. Mọi bảng biến/secret/DNS/monitor tra ở
`shared-reference.md` (file này chỉ trỏ sang khi cần).

**Khi nào dùng**: VPS đang chạy monolith (`api.closy.hycat.online`), có Docker +
`/opt/closy`; domain `hycat.online` DNS ở **Vercel**.
**Không dùng khi**: dựng box ngoài AWS (đọc `aws-runbook.md`).

Placeholder (thay trong mọi lệnh): `hycat-team/smart-wardrobe-be` (điền sẵn) ·
`<VPS_HOST>` · `<VPS_USER>` · `<VPS_SSH_PORT>`.

## Bước 0 — GitHub: environment + secrets VPS (làm 1 lần)

1. Tạo 2 environment `vps-production` (+ `aws-production` để dành GĐ2) ở
   `github.com/hycat-team/smart-wardrobe-be/settings/environments` (nên bật Required reviewers).
2. Tạo GHCR read token cho VPS: GitHub Settings → Personal access token
   (classic) → note `closy-vps-ghcr-read`, scope **`read:packages`** → copy token.
3. Set secrets (copy-paste, sửa giá trị) — chạy bằng **PowerShell**:
```powershell
$OWNER_REPO = "hycat-team/smart-wardrobe-be"
gh secret set GHCR_USERNAME -R $OWNER_REPO --body "<github-username>"
gh secret set GHCR_READ_TOKEN -R $OWNER_REPO --body "ghp_xxx"
gh secret set VPS_HOST -R $OWNER_REPO --body "<VPS_HOST>"
gh secret set VPS_USER -R $OWNER_REPO --body "<VPS_USER>"
gh secret set VPS_SSH_PORT -R $OWNER_REPO --body "22"
Get-Content -Raw "$HOME/.ssh/id_ed25519" | gh secret set VPS_SSH_KEY -R $OWNER_REPO
# Windows built-in ssh-keyscan loi KEX (sntrup761...); dung ban cua Git for Windows.
$keyscan = "$env:ProgramFiles\Git\usr\bin\ssh-keyscan.exe"
if (-not (Test-Path $keyscan)) { $keyscan = "$env:LOCALAPPDATA\Programs\Git\usr\bin\ssh-keyscan.exe" }
if (-not (Test-Path $keyscan)) { $keyscan = "ssh-keyscan" }
& $keyscan -p 22 "<VPS_HOST>" | Out-File -Encoding ascii docs/temp/vps_known_hosts
Get-Content -Raw docs/temp/vps_known_hosts | gh secret set VPS_KNOWN_HOSTS -R $OWNER_REPO
gh variable set PRODUCTION_HEALTH_URL -R $OWNER_REPO --body "https://api.closy.hycat.online/api/v1/health"
```
4. Verify: `gh secret list -R "$OWNER_REPO"` đủ 7 secrets trên; environment
   `vps-production` đã có (tự tạo ở lần dispatch đầu nếu chưa tạo tay).

## Bước 1 — Push tag rollback (BẮT BUỘC, làm 1 lần)

`deploy-production.yml` verify `refs/tags/monolith-legacy`; thiếu là rollback fail.
```bash
git tag -a monolith-legacy <BASELINE_SHA> -m "frozen monolith-green before 020"
git push origin monolith-legacy
git ls-remote --tags origin | grep monolith-legacy   # phải thấy
```
Tag này đóng băng, cấm di chuyển.

## Bước 2 — Chuẩn bị `/opt/closy` trên VPS

```bash
cd /opt/closy
cp /duong/dan/repo/.env.production.example .env.production
```
Điền tối thiểu (bảng biến đầy đủ: `shared-reference.md` §2):

| Biến trong `.env.production` | Giá trị |
|---|---|
| `API_DOMAIN` | `api.closy.hycat.online` |
| `API_V2_DOMAIN` | `api-v2.closy.hycat.online` |
| `NGINX_MODE` | `services` |
| `DB_HOST`/`DB_PORT`/`DB_LISTENER_HOST`/`DB_*` | **luôn Postgres local** (`postgres`) — cấm trỏ Supabase (luật isolation) |
| `GOOGLE_WEB_CLIENT_ID`/`GOOGLE_WEB_CLIENT_SECRET` | client Google **Web application** (không phải Android/iOS) |
| `GOOGLE_CLIENT_IDS` | client iOS/Android (để trống nếu chỉ có web); không khai client web ở đây |
| `GOOGLE_REDIRECT_URI` | `https://api.closy.hycat.online/api/v1/auth/google/callback` — phải khớp Authorized redirect URI trong Google Cloud Console |

Thiếu `GOOGLE_WEB_CLIENT_ID`/`GOOGLE_WEB_CLIENT_SECRET` khi có `GOOGLE_REDIRECT_URI` thì service
không boot (validation lúc startup), và web login trả `redirect_uri_mismatch` nếu trỏ nhầm client
Android/iOS.

Không còn `COMPOSE_PROFILES`, `*_RUN_URL`, `TOKEN_HELPER_*` — thấy là xóa.

Guard isolation (chạy trên VPS, chỉ in tên biến khi FAIL — không in giá trị):
```bash
for v in DB_HOST DB_LISTENER_HOST REDIS_HOST RABBITMQ_HOST; do
  val=$(grep "^$v=" .env.production | cut -d= -f2)
  case "$val" in
    *supabase.co*|*redislabs.com*|*cloudamqp.com*)
      echo "FAIL: $v points outside VPS" && exit 1 ;;
  esac
done
echo "OK: VPS infra is local-only"
```

## Bước 3 — Cert Let's Encrypt cho `api.closy` (làm 1 lần)

Do Nginx đang chạy (monolith), cấp bằng standalone (giải phóng port 80):
```bash
docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml stop nginx
docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
  --profile maintenance run --rm certbot certonly --standalone \
  -d api.closy.hycat.online --expand --agree-tos -m <email>
```
(`docker stop nginx` không trúng container thật `closy-prod-nginx-1`; phải stop qua
compose.) Renewal sau này qua webroot (`certbot_www` đã mount sẵn). Cert `api-v2`
chỉ cấp khi cần redirect grace (DNS api-v2 đã về VPS) — thêm
`-d api-v2.closy.hycat.online` khi đó.

## Bước 4 — Deploy 3 service (dispatch hoặc push tag)

```bash
gh workflow run deploy-vps-services.yml -R "hycat-team/smart-wardrobe-be" --ref main -f services=all
# hoặc: git tag -a vps-v1.0.0 -m "..." && git push origin vps-v1.0.0  (luôn = all services)
RUN_ID=$(gh run list --workflow deploy-vps-services.yml -R "hycat-team/smart-wardrobe-be" --limit 1 \
  --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" -R "hycat-team/smart-wardrobe-be"
```
Nếu environment có Required reviewers → vào `Actions → run → Review deployments →
Approve`. Pipeline 3 chặng: build matrix → GHCR (pin digest) → `compose up -d`
→ recreate nginx 1 lần → `smoke-all` per-service qua nginx. Deploy lẻ 1 service:
`-f services=commerce` (smoke-all vẫn check cả 3, không lỗi).
> Không bao giờ `down -v` trên prod (xóa volumes `postgres_data`... = mất dữ liệu).
> Backup DB trước mọi lần recreate postgres.

### Queue type drift (nếu deploy đỏ ở `Check RabbitMQ queue topology`)

Workflow **tự dừng trước khi đụng container** khi broker lệch type queue so với
`topology.go`. RabbitMQ không đổi được `x-queue-type` trên queue đã tồn tại, nên nếu
bỏ qua bước này thì service sẽ crash-loop với `406 PRECONDITION_FAILED`.

```bash
cd /opt/closy
export TOOL_IMAGE=ghcr.io/hycat/rabbitmq-topology@sha256:<digest>   # digest ở log workflow
bash deployments/scripts/rabbitmq-topology/01-check-drift.sh vps     # read-only
bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh         # xem trước
bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh --yes   # dừng worker → migrate → verify
```

Migrate dừng `account commerce ai` (mỗi container trên VPS chạy cả http lẫn worker) và
**không** start lại — start tay khi nào bạn sẵn sàng. Chi tiết:
[`deployments/scripts/rabbitmq-topology/README.md`](../../deployments/scripts/rabbitmq-topology/README.md).

## Bước 5 — Cắt traffic monolith → 3 service (ngoài giờ thấp điểm)

Mọi lệnh `docker compose` dưới đây kèm **cả 2 env-file**:
`--env-file .env.production --env-file .env.deploy`.

1. Deploy 3 service mới nhất (Bước 4) để có `.env.deploy` chứa
   `ACCOUNT/COMMERCE/AI_IMAGE` (nginx `depends_on` 3 service nên thiếu là lỗi ngay).
2. Dừng backend cũ: `... stop backend || true`; verify `docker ps` trống backend.
3. Đặt `NGINX_MODE=services` trong `/opt/closy/.env.production`.
4. Recreate nginx + kiểm tra:
```bash
docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
  up -d --force-recreate nginx
docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml ps
```
Compose mount `./nginx/templates/${NGINX_MODE}` vào `/etc/nginx/templates`;
entrypoint nginx chạy `envsubst` render conf. Route map `api.closy` → 3 service nằm
trong vùng `# BEGIN/END GENERATED ROUTES` của
`deployments/nginx/templates/services/vps-3service.conf.template` (sinh bằng
`make nginx-routes`, verify `make nginx-check` — single source chung với local).
5. Smoke từng service (body chứa đúng tên service):
```bash
for svc in account commerce ai; do
  curl -sS "https://api.closy.hycat.online/api/v1/$svc/health" | grep -o '"service":"[a-z]*"'
done
curl -sSI https://api.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"
docker ps --filter name=backend --format '{{.Names}}'   # trống
```
Rollback bất cứ lúc nào: `NGINX_MODE=rollback` + chạy `deploy-production.yml`
(`-f confirm=rollback`, đổi mode **trước** khi chạy; workflow tự recreate nginx).

## Bước 6 — Gate + vận hành

- [ ] `api.closy` 200 qua đủ 3 service (`/api/v1/<svc>/health` đúng tên; route + SSE + auth + upload).
- [ ] Backend dừng; guard isolation pass; 0 request qua Cloudflare; ghi mốc + checksum.
- Đổi `NGINX_MODE` trong `.env.production` rồi recreate nginx (luôn kèm cả 2 env-file).
  `services` (ổn định) · `services-redirect` (grace sau khi tắt box ngoài, xong quay về
  `services`) · `rollback` (khẩn cấp).
- DNS `api.closy` giữ nguyên, không đổi. Log/health/monitor: `shared-reference.md` §4.

## Phụ lục — giới hạn đã biết

- Migration Goose mở connection direct riêng (`DB_LISTENER_HOST`/`PORT`/`SSLMODE`,
  5432); runtime query qua pooler/direct tùy môi trường.
- `:latest` chỉ ở CI; production pin `<git-sha>`/digest; image private không public.
- Nginx **không** rate-limit (chỉ body-limit/timeout/security-header); rate-limit nằm
  ở app (Redis-backed). Không WAF đầy đủ — chấp nhận, ngoài phạm vi.
