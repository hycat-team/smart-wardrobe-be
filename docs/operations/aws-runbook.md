# Runbook box ngoài AWS — GĐ2 start-to-finish

Box chạy 6 container (`APP_MODE=http`×3 + `worker`×3) + Nginx phục vụ
`api-v2.closy.hycat.online`, chỉ nói chuyện với managed (Supabase/Redis Cloud/
CloudAMQP). Không IAM, không token-helper. Độc lập 100% với VPS — bật/tắt box
không đụng VPS. Đọc xuôi từ trên xuống; bảng biến/secrets/DNS/monitor tra ở
`shared-reference.md`.

Placeholder (thay trong mọi lệnh): `hycat-team/smart-wardrobe-be` (điền sẵn)
· `<BOX_IP>` (static IP box).

## Bước 0 — Verify quyết định (đọc trước khi làm)

- Compute: **Lightsail Small-2GB \$12/tháng, region Singapore** (x86_64 — image amd64
  hiện tại dùng nguyên, không build lại). Ước tính ~\$36/3 tháng + overage SG \$0.12/GB.
  Lightsail chạy trên chính hạ tầng EC2/AWS (cùng Region/AZ) — không phải VPS bên thứ
  ba; chọn Lightsail vì giá gộp (SSD + transfer allowance) rẻ hơn EC2 cùng RAM.
- Credit \$100 hết hạn 3/2027; Support đã Basic (\$0).
- Fallback nếu Lightsail hết capacity: EC2 `t3.small` cùng region (2 GiB, ngang plan này).
  KHÔNG dùng `t3.micro` (1 GiB) — `mem_limit` 6 container cộng lại ≈ 3.1 GiB.
- Không dùng: Fargate+ALB+NAT (~\$100+/tháng), App Runner (đóng khách mới 30/4/2026).

## Bước 1 — Điều kiện tiên quyết (điểm bắt đầu GĐ2, tick từng ô)

- [ ] Tài khoản AWS + Lightsail khả dụng ở Singapore; đã bật billing alarm \$20/40/60.
- [ ] Supabase / Redis Cloud / CloudAMQP đã tạo, đã có host/port/password (Bước 2).
- [ ] payOS: đã tạo kênh thanh toán riêng cho box AWS (Bước 2.4).
- [ ] GitHub secrets `AWS_*` (5 secret — Bước 5); environment `aws-production`
      (nên bật Required reviewers).
- [ ] Image 3 service đã có trên GHCR (Bước 7 tự build+push).

## Bước 2 — Đăng ký managed (Phase 0, làm trên browser)

### 2.1 Supabase Pro — Postgres + pgvector (Singapore)

1. https://supabase.com → **New project** (Singapore) → lưu **Database password**.
2. **SQL Editor**, chạy:
```sql
create extension if not exists vector;
create schema if not exists shared;
create schema if not exists identity;
create schema if not exists wardrobe;
create schema if not exists fashion;
create schema if not exists brand;
create schema if not exists subscription;
create schema if not exists aiexecutor;
create schema if not exists dashboard;
alter role postgres set search_path =
  public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared;
```
(`search_path` set **server-side** cho role thật, không truyền qua DSN pooler.)
3. **Project Settings → Database → Connection string**, lấy:
   - Pooler/Transaction (port **6543**): host `aws-0-ap-southeast-1.pooler.supabase.com`,
     user `postgres.<project-ref>` → runtime app.
   - Direct (port **5432**): host `db.<project-ref>.supabase.co` → migration/`LISTEN`/lock.

Verify: `psql` pooler phải trả `search_path` đủ 8 schema; direct `select 1` OK.
Sự cố: `prepared statement already exists` = query lọt qua pooler sai mode;
`relation does not exist` = thiếu schema trong `search_path`.

### 2.2 Redis Cloud Essentials 250MB

1. https://app.redislabs.com → **New subscription** → **Essentials** 250MB.
2. Tab **Configuration**/**Security**, lấy: Public endpoint (host, KHÔNG phải 6379 —
   thường port 12000+) → `REDIS_HOST`/`REDIS_PORT`; Default user password →
   `REDIS_PASSWORD`. User Redis luôn là `default` — app chỉ dùng password, không có biến username.
   TLS: gói **free 30MB không có TLS** → `REDIS_TLS_ENABLED=false`; gói trả phí (Essentials+) → `true`.
3. Verify: `redis-cli -h ... -p ... -a ... -c PING` → `PONG` (thêm `--tls` chỉ khi gói có TLS).
   Ngân sách: tổng client conn ≤ 256, headroom ≥ 20%.

### 2.3 CloudAMQP Tough Tiger

1. https://customer.cloudamqp.com → **Create New Instance** → Tough Tiger →
   tab **Vhosts** → Create vhost riêng.
2. Tab **Details** → copy **AMQP URL** `amqps://<user>:<pass>@<host>/<vhost>` → tách
   `RABBITMQ_HOST`/`PORT=5671`/`USER`/`PASSWORD`/`VHOST` (giữ lại phần path sau
   host làm `VHOST` — app luôn dial đúng vhost này; bỏ sót là broker trả
   `403 no access to this vhost` và cả 3 service crash-loop). Workflow tự set TLS flags.
3. Verify: `openssl s_client -connect "$RABBITMQ_HOST:5671"` → `Verify return code: 0`;
   management API qua HTTPS OK. Topology (quorum/DLX) code tự khai báo.
   Ngân sách: tổng conn ≤ 100.

### 2.4 payOS — kênh thanh toán RIÊNG cho box AWS

Mỗi kênh payOS chỉ có **một** webhook URL → 2 trục song song bắt buộc dùng 2 kênh
riêng (không share credentials với VPS; doanh thu tách theo kênh).

1. https://my.payos.vn → tạo **Kênh thanh toán** mới cho box AWS → lấy
   `PAYOS_CLIENT_ID` / `PAYOS_API_KEY` / `PAYOS_CHECKSUM_KEY`.
2. Điền 3 key này vào `.env.aws` (Bước 4) — **KHÔNG copy từ VPS**.
3. Webhook URL (`https://api-v2.closy.hycat.online/api/v1/subscriptions/payos-webhook`)
   không khai ở bước này mà confirm bằng API sau khi box đã chạy (Bước 7.5).

## Bước 3 — Provision box (làm 1 lần)

1. Lightsail Console → **Create instance** → Region **Singapore** → Linux →
   **Ubuntu 22.04** → Plan **\$12 Small-2GB** → tên `closy-api-v2` → Create.
2. **Static IP**: Networking → **Create static IP** → attach vào instance (ghi IP →
   `BOX_IP`, dùng cho Bước 4–5). Không dùng public IP động. **Giữ IP này xuyên suốt
   các chu kỳ.**
3. **Firewall**: mở `22` (giới hạn IP bạn), `80`, `443`. Đóng hết còn lại.
4. SSH vào box (dùng key `.pem` của Lightsail — mọi lệnh ssh/scp sau này đều `-i` key này),
   rồi cài Docker:
```powershell
ssh -i "$HOME\Downloads\LightsailDefaultKey-ap-southeast-1.pem" ubuntu@<BOX_IP>
```
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # đăng nhập lại
docker --version && docker compose version
```

## Bước 4 — Tạo `.env.aws` bằng script (không chép tay)

Hai script PowerShell local (gitignored ở `docs/temp/`) lo phần dễ sai. Chạy từ
gốc repo.

1. Scaffold từ VPS — copy sẵn JWT/Brevo/Cloudinary/AI từ `.env.production`
   (payOS KHÔNG copy — kênh riêng, xem Bước 2.4), điền sẵn hằng số box, để trống
   các giá trị managed:
```powershell
powershell -ExecutionPolicy Bypass -File docs/temp/aws-env-bootstrap.ps1
# -> docs/temp/.env.aws, còn ~15 dòng '# TODO'
```
2. Điền ~15 dòng `# TODO` (Supabase/Redis Cloud/CloudAMQP ở Bước 2, payOS ở Bước 2.4,
   `BOX_IP` ở Bước 3).
   **Không sinh `JWT_SECRET` mới** — script copy nguyên từ VPS để token 2 trục verify chéo.
3. Sinh khối lệnh copy-paste cho Bước 5:
```powershell
powershell -ExecutionPolicy Bypass -File docs/temp/aws-env-to-copyblock.ps1
# -> docs/temp/aws-secrets-to-set.txt (gitignored, chứa secret — xóa sau khi dùng)
```

**Lưu `.env.aws` vào password manager ngay** — mục 9.2 tạo lại fresh cần nó. Không
bao giờ commit (đã gitignore).

## Bước 5 — GitHub secrets + đưa `.env.aws` lên box

Box đọc app config từ `/opt/closy-aws/.env.aws`, **không** qua GitHub secrets.
`deploy-aws.yml` chỉ đọc **5 secret** (tái dùng `GHCR_READ_TOKEN`/`GHCR_USERNAME`
đã set cho VPS). Không đặt `DB_*`/`REDIS_*`/`RABBITMQ_*`/`JWT_*`/`CLOUDINARY_*`/
`EMAIL_*`/`PAYOS_*` làm secret — chúng nằm trong `.env.aws`.

Mở `docs/temp/aws-secrets-to-set.txt` (sinh ở Bước 4) rồi chạy 2 khối:
- **[1/2]** — 5 lệnh `gh secret set` cho `AWS_HOST`, `AWS_USER`, `AWS_SSH_PORT`,
  `AWS_SSH_KEY`, `AWS_KNOWN_HOSTS`.
- **[2/2]** — tạo `/opt/closy-aws` trên box + `scp` `.env.aws` lên + `chmod 600`.

Verify: `gh secret list -R "hycat-team/smart-wardrobe-be" | grep AWS_` đủ 5 dòng.

## Bước 6 — Seed data VPS → Supabase (trước deploy đầu / mỗi lần tạo lại)

```bash
TS=$(date +%F_%H%M)
mkdir -p /opt/closy/backup
cd /opt/closy
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  exec -T postgres pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc > "backup/closy_${TS}.dump"
sha256sum "backup/closy_${TS}.dump" | tee "backup/closy_${TS}.sha256"
pg_restore --clean --if-exists --no-owner --no-acl \
  -d "host=<direct-host> port=5432 user=<user> dbname=<dbname> sslmode=require" \
  "backup/closy_${TS}.dump"
```
Verify checksum: row-count bảng chính (`users`, `subscriptions`, `wardrobe_items`,
`outbox`) trên cả 2 phía phải khớp tại mốc dump; Supabase query đúng 8 schema,
0 lỗi prepared statement. Chênh lệch chỉ đến từ ghi mới sau mốc dump.

## Bước 7 — TLS + deploy + smoke IP

1. Hạ TTL `api-v2.closy.hycat.online` xuống 60 → trỏ về IP box (Vercel DNS) →
   cấp cert lần đầu trên box (box fresh chỉ có `.env.aws`, chưa có
   `deployments/` hay `.env.deploy` — 2 thứ này chỉ xuất hiện sau deploy đầu —
   nên KHÔNG dùng `docker compose ... run certbot` ở đây mà chạy certbot trực tiếp):
```bash
docker run --rm -p 80:80 \
  -v /etc/letsencrypt:/etc/letsencrypt \
  certbot/certbot:v5.3.1 certonly --standalone \
  -d api-v2.closy.hycat.online --agree-tos -m <email> --no-eff-email --non-interactive
sudo ls -l /etc/letsencrypt/live/api-v2.closy.hycat.online/
# kỳ vọng: fullchain.pem + privkey.pem
```
Lưu ý: port 80 lúc này phải trống (nginx còn chưa deploy nên không xung đột).
Verify DNS trước nếu lỗi: `dig +short api-v2.closy.hycat.online` phải ra `<BOX_IP>`.
Xong mục 1 → qua mục 2 (deploy). Phần chuyển renew sang webroot + auto-renew nằm ở
mục 4 — chỉ làm sau khi deploy + smoke (mục 2–3) xong.
2. Deploy (dispatch hoặc push tag — tag luôn = all services):
```bash
gh workflow run deploy-aws.yml -R "hycat-team/smart-wardrobe-be" --ref main -f services=all
# hoặc: git tag -a aws-v1.0.0 -m "..." && git push origin aws-v1.0.0
```
Pipeline 3 chặng (Approve nếu có reviewer): build+push GHCR → scp compose+templates
→ `.env.deploy` pin digest → `pull`/`up -d` → recreate nginx 1 lần → `smoke-all`.
3. Smoke tay từng service (SNI override, chưa cần DNS ổn định; body đúng tên service):
```bash
for svc in account commerce ai; do
  curl --fail -sS --resolve "api-v2.closy.hycat.online:443:<BOX_IP>" \
    "https://api-v2.closy.hycat.online/api/v1/$svc/health" | grep -o '"service":"[a-z]*"'
done
```
4. Chuyển renew sang webroot + auto-renew (chỉ làm SAU khi mục 2–3 xong — lúc này
pipeline đã bung `deployments/` lên box, `.env.deploy` đã có, nginx đang chạy;
đừng để tới lúc cert sắp hết hạn mới làm):

a) Kiểm tra ACME challenge đi qua nginx đang chạy (làm trên box):
```bash
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  --profile maintenance run --rm --entrypoint sh certbot \
  -c 'mkdir -p /var/www/certbot/.well-known/acme-challenge && echo closy-certbot-ok > /var/www/certbot/.well-known/acme-challenge/test'
```
Từ máy local kiểm tra: `curl http://api-v2.closy.hycat.online/.well-known/acme-challenge/test`
phải trả `closy-certbot-ok` (không được thì kiểm tra firewall port 80 + block
`/.well-known/acme-challenge/` trong template nginx).
b) Chuyển renewal method sang webroot + dry-run (làm trên box):
```bash
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  --profile maintenance run --rm certbot reconfigure --cert-name api-v2.closy.hycat.online \
  --authenticator webroot --webroot-path /var/www/certbot
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml \
  --profile maintenance run --rm certbot renew --dry-run
# kỳ vọng: Congratulations, all simulated renewals succeeded
```
c) Script renew thông minh — chỉ reload nginx khi cert thực sự đổi (làm trên box):
```bash
sudo nano /usr/local/sbin/closy-renew-cert-aws.sh
```
```bash
#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="/opt/closy-aws"
CERT_FILE="/etc/letsencrypt/live/api-v2.closy.hycat.online/fullchain.pem"
DOCKER_COMPOSE_FILE="deployments/docker-compose.aws.yml"
ENV_AWS_FILE=".env.aws"
ENV_DEPLOY_FILE=".env.deploy"

cd "$DEPLOY_DIR"

BEFORE="$(stat -L -c %Y "$CERT_FILE" 2>/dev/null || echo 0)"

docker compose --env-file "$ENV_AWS_FILE" --env-file "$ENV_DEPLOY_FILE" -f "$DOCKER_COMPOSE_FILE" \
  --profile maintenance run --rm certbot renew --quiet

AFTER="$(stat -L -c %Y "$CERT_FILE" 2>/dev/null || echo 0)"

if [ "$BEFORE" != "$AFTER" ]; then
    docker compose --env-file "$ENV_AWS_FILE" --env-file "$ENV_DEPLOY_FILE" -f "$DOCKER_COMPOSE_FILE" \
      exec -T nginx nginx -t
    docker compose --env-file "$ENV_AWS_FILE" --env-file "$ENV_DEPLOY_FILE" -f "$DOCKER_COMPOSE_FILE" \
      exec -T nginx nginx -s reload
    echo "$(date --iso-8601=seconds) Certificate renewed and Nginx reloaded successfully."
else
    echo "$(date --iso-8601=seconds) Certificate is not due for renewal. No action taken."
fi
```
```bash
sudo chmod 750 /usr/local/sbin/closy-renew-cert-aws.sh
sudo /usr/local/sbin/closy-renew-cert-aws.sh
# kỳ vọng khi cert còn hạn: Certificate is not due for renewal. No action taken.
```
d) Cron root 2 lần/ngày (làm trên box):
```bash
sudo crontab -e
# thêm dòng:
# 17 3,15 * * * /usr/local/sbin/closy-renew-cert-aws.sh >> /var/log/closy-certbot.log 2>&1
sudo tail -n 50 /var/log/closy-certbot.log
```
Lưu ý vòng đời phù du (Bước 9): mỗi lần tạo lại box fresh là cert mới → làm lại
mục 1 (docker run standalone) + mục 4 này; khi xóa box thì cron mất theo máy, không cần cleanup.
5. **Confirm webhook payOS cho kênh AWS** (làm 1 lần, sau khi `api-v2` sống + cert OK):
```bash
curl -sS -X POST "https://api-merchant.payos.vn/confirm-webhook" \
  -H "x-client-id: <PAYOS_CLIENT_ID kênh AWS>" -H "x-api-key: <PAYOS_API_KEY kênh AWS>" \
  -H "Content-Type: application/json" \
  -d '{"webhookUrl":"https://api-v2.closy.hycat.online/api/v1/subscriptions/payos-webhook"}'
```
payOS gửi một payload mẫu (`orderCode 123`) tới webhook, endpoint phải trả **2XX**.
Kỳ vọng `{"code":"00",...}`. Nếu `400 Webhook url invalid` / `5XX`: xem log account —
`ignored_unknown_order` (đã ack 200, đúng thiết kế), sai `PAYOS_CHECKSUM_KEY` của kênh,
hoặc nginx/DNS chưa thông. Confirm chỉ cần **một lần**; URL không đổi giữa các chu kỳ.

## Bước 8 — Cắt DNS + quan sát

1. DNS `api-v2` đã trỏ box từ Bước 7 — smoke công khai + quan sát 24–48h
   (`docker compose logs -f`, error-rate, p95).
2. Rollback: trỏ DNS về IP VPS (grace `services-redirect` trên VPS nếu đã bật).
3. Vận hành thường ngày:
```bash
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml ps
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml logs -f --tail=100 <svc>
gh workflow run deploy-aws.yml -R "hycat-team/smart-wardrobe-be" --ref main -f services=commerce  # redeploy lẻ
```
> Không bao giờ `down -v` (tránh nhầm sang VPS). Migration chạy lúc boot per
> container (http+worker cùng service share 1 bảng qua advisory lock).

## Bước 9 — Chu kỳ xóa / tạo lại box (luồng phù du)

Test vài ngày → xóa → gần demo tạo lại fresh → chạy ~5 ngày → xóa. Managed sống
xuyên suốt — chỉ box là phù du. VPS là canonical data.

> Giữ lại giữa các chu kỳ: **giữ managed, giữ static IP** (~\$0.005/h khi unattached —
> đổi lại tạo lại không đụng DNS, secret `AWS_HOST` không bao giờ sửa). Chỉ instance
> là xóa/tạo mới.

### 9.1 Xóa tạm (sau test vài ngày / sau demo)

1. **Drain payment kênh AWS trước khi xóa**: để reconciliation (cron 10 phút) chốt
   hết giao dịch payOS đang `pending` — sau khi reunify sang VPS, credentials là kênh
   VPS nên **không** query được payment link của kênh AWS (sẽ rơi vào `investigation`).
   Kiểm tra: không còn `deposit_transactions` trạng thái pending của kênh AWS.
2. Reunify Supabase → VPS (mục 9.3 dưới đây) — VPS gom đủ ghi cả 2 phía.
3. Trỏ DNS `api-v2` về VPS + `NGINX_MODE=services-redirect` trên VPS (hết grace quay
   về `services`).
4. Delete instance trên Lightsail Console (snapshot **tùy chọn** — mục 9.4).
5. KHÔNG hủy gì khác: managed giữ, static IP giữ (unattached), secrets GitHub giữ,
   kênh payOS AWS giữ (webhook URL đã confirm, không cần làm lại).

### 9.2 Tạo lại fresh (trước demo vài ngày)

1. Create instance mới (cùng region/blueprint/plan) → attach **static IP cũ** →
   firewall → Docker (như Bước 3, khỏi cấp IP mới).
2. `.env.aws` từ password manager (không có backup thì dừng lại).
3. DNS `api-v2` → IP box → certbot (Bước 7.1) → tag `aws-v*` → Approve → smoke IP.
4. Seed VPS → Supabase (Bước 6) → quan sát. DNS đã đúng nên không đổi lần 2.
5. Kiểm tra webhook payOS: nếu dashboard báo URL invalid (do box down lâu) → chạy lại
   confirm ở Bước 7.5; nếu vẫn `active` thì bỏ qua.

### 9.3 Reunify về VPS (chạy trước mỗi lần xóa)

```bash
# a) dump Supabase qua direct 5432
pg_dump "host=<direct-host> port=5432 user=<user> dbname=<dbname> sslmode=require" \
  -Fc > /opt/closy/backup/supabase_$(date +%F_%H%M).dump
sha256sum /opt/closy/backup/supabase_*.dump
# b) restore vào Postgres docker VPS
docker compose --env-file /opt/closy/.env.production -f /opt/closy/deployments/docker-compose.prod.yml \
  exec -T postgres pg_restore --clean --if-exists --no-owner -U "$DB_USER" -d "$DB_NAME" \
  < /opt/closy/backup/supabase_<TS>.dump
```
- [ ] Checksum bảng chính khớp.
- [ ] Không còn payment kênh AWS ở trạng thái `pending` (Bước 9.1 bước 1).
- Drain AMQP: ngừng worker box → đợi queue CloudAMQP rỗng
  (`curl -u user:pass https://<host>/api/queues/<vhost> | grep -o '"messages":[0-9]*'`)
  → event xử lý hết trước khi tắt managed (nếu teardown cuối).
- Redis: snapshot RDB nếu cần warmup, hoặc bỏ qua (cache rebuild).

### 9.4 Snapshot (tùy chọn, "nếu cần")

Giữ khi muốn tạo lại y nguyên trong vài phút hoặc cần bằng chứng đĩa:
Lightsail → instance → **Snapshots** → Create (~\$0.05/GB-tháng). Tạo lại từ snapshot
thay Bước 9.2.1–9.2.2 (vẫn attach IP cũ + smoke lại). Hết cần thì xóa — quên là charge.

## Bước 10 — Teardown toàn bộ sau demo (giữ VPS)

Xóa hết AWS + managed để không bị tính phí tiếp. VPS, `api.closy`, GHCR images,
repo, AWS account (\$0 + alarm) và kênh payOS AWS (không tốn phí, giữ cho vòng sau)
giữ nguyên.

0. Điều kiện: demo xong, reunify cuối về VPS (mục 9.3) xong, grace redirect xong,
   record DNS `api-v2` đã xóa.
1. Backup cuối: dump Supabase 1 bản (mục 9.3.a) + lưu `.env.aws` — xong mới được xóa.
2. Thứ tự xóa (không đảo):
   1. Delete Lightsail instance.
   2. Delete snapshots (quên là charge \$/tháng).
   3. **Release static IP** (quên là charge unattached IP).
   4. Delete Supabase project → Redis subscription → CloudAMQP instance (trước kỳ renew).
   5. Revoke secrets GitHub:
      ```bash
      for s in AWS_HOST AWS_USER AWS_SSH_PORT AWS_SSH_KEY AWS_KNOWN_HOSTS; do
        gh secret remove "$s" -R "hycat-team/smart-wardrobe-be"
      done
      ```
      (pattern rotate/revoke chi tiết: `shared-reference.md` §6).
3. Verify \$0:
   - [ ] AWS Billing/Cost Explorer = 0, alarm vẫn bật canh.
   - [ ] Dashboard 3 managed xác nhận deleted, hết chu kỳ renew.
   - [ ] `gh secret list` hết `AWS_*`; `api.closy` phục vụ bình thường từ VPS.

## Bước 11 — Sự cố + gate

| Triệu chứng | Kiểm tra | Xử lý |
|---|---|---|
| `api-v2` 502/timeout | `ps`, log app, managed conn | restart service; redeploy nếu do image mới |
| Cert hết hạn | `certbot certificates` | renew + recreate nginx |
| Box idle-stop/không SSH | Lightsail console, static IP còn attach | start lại; kiểm tra firewall |
| OOM (2GB) | `docker stats`, log oom-killer | giảm `mem_limit` / nâng plan |
| Bill vượt | billing alarm, transfer metrics | overage \$0.12/GB; consider tắt box |

Gate: 6 container healthy; `api-v2` 200 công khai + cert auto-renew; webhook payOS đã
confirm (`code:"00"`); VPS không đổi gì (trừ chủ động cắt); billing trong hạn mức;
mỗi lần xóa: DNS đúng pha + reunify xong + hết payment pending kênh AWS.
