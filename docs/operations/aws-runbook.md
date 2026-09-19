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
- Credit \$100 hết hạn 3/2027; Support đã Basic (\$0).
- Fallback nếu Lightsail hết capacity: EC2 `t3.micro` cùng region.
- Không dùng: Fargate+ALB+NAT (~\$100+/tháng), App Runner (đóng khách mới 30/4/2026).

## Bước 1 — Điều kiện tiên quyết (điểm bắt đầu GĐ2, tick từng ô)

- [ ] Tài khoản AWS + Lightsail khả dụng ở Singapore; đã bật billing alarm \$20/40/60.
- [ ] Supabase / Redis Cloud / CloudAMQP đã tạo, đã có host/port/password (Bước 2).
- [ ] GitHub secrets `AWS_*` + nhóm app/managed (Bước 3); environment `aws-production`
      (nên bật Required reviewers).
- [ ] Image 3 service đã có trên GHCR (Bước 5 tự build+push).

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
   `REDIS_PASSWORD`. Bật **TLS** (workflow tự set `REDIS_TLS_ENABLED=true`).
3. Verify: `redis-cli -h ... -p ... -a ... --tls -c PING` → `PONG`.
   Ngân sách: tổng client conn ≤ 256, headroom ≥ 20%.

### 2.3 CloudAMQP Tough Tiger

1. https://customer.cloudamqp.com → **Create New Instance** → Tough Tiger →
   tab **Vhosts** → Create vhost riêng.
2. Tab **Details** → copy **AMQP URL** `amqps://<user>:<pass>@<host>/<vhost>` → tách
   `RABBITMQ_HOST`/`PORT=5671`/`USER`/`PASSWORD`. Workflow tự set TLS flags.
3. Verify: `openssl s_client -connect "$RABBITMQ_HOST:5671"` → `Verify return code: 0`;
   management API qua HTTPS OK. Topology (quorum/DLX) code tự khai báo.
   Ngân sách: tổng conn ≤ 100.

## Bước 3 — GitHub secrets (copy-paste, sửa giá trị)

```bash
OWNER_REPO="hycat-team/smart-wardrobe-be"
# --- Box (điền sau Bước 4: static IP) ---
gh secret set AWS_HOST -R "$OWNER_REPO"          --body "<BOX_IP>"
gh secret set AWS_USER -R "$OWNER_REPO"          --body "ubuntu"
gh secret set AWS_SSH_PORT -R "$OWNER_REPO"      --body "22"
gh secret set AWS_SSH_KEY -R "$OWNER_REPO" < ~/.ssh/aws_box
ssh-keyscan -p 22 <BOX_IP> > aws_known_hosts
gh secret set AWS_KNOWN_HOSTS -R "$OWNER_REPO" < aws_known_hosts
# --- Database (Supabase, Bước 2.1) ---
gh secret set DB_USER -R "$OWNER_REPO"     --body "postgres.<project-ref>"
gh secret set DB_PASSWORD -R "$OWNER_REPO" --body "<db-password>"
gh secret set DB_NAME -R "$OWNER_REPO"     --body "postgres"
gh secret set DB_HOST -R "$OWNER_REPO"     --body "aws-0-ap-southeast-1.pooler.supabase.com"
gh secret set DB_PORT -R "$OWNER_REPO"     --body "6543"
gh secret set DB_LISTENER_HOST -R "$OWNER_REPO"     --body "db.<project-ref>.supabase.co"
gh secret set DB_LISTENER_SSLMODE -R "$OWNER_REPO"  --body "require"
# --- Redis / RabbitMQ (Bước 2.2–2.3) ---
gh secret set REDIS_HOST -R "$OWNER_REPO"     --body "<redis-host>"
gh secret set REDIS_PORT -R "$OWNER_REPO"     --body "<redis-port>"
gh secret set REDIS_PASSWORD -R "$OWNER_REPO" --body "<redis-password>"
gh secret set RABBITMQ_HOST -R "$OWNER_REPO"     --body "<xxx.rmq.cloudamqp.com>"
gh secret set RABBITMQ_PORT -R "$OWNER_REPO"     --body "5671"
gh secret set RABBITMQ_USER -R "$OWNER_REPO"     --body "<rabbit-user>"
gh secret set RABBITMQ_PASSWORD -R "$OWNER_REPO" --body "<rabbit-password>"
# --- App (JWT tự sinh; còn lại COPY NGUYÊN từ VPS .env.production — share quota) ---
gh secret set JWT_ALGORITHM -R "$OWNER_REPO" --body "HS256"
gh secret set JWT_SECRET -R "$OWNER_REPO"    --body "$(openssl rand -base64 48)"
gh secret set SWAGGER_ACCESS_CODE -R "$OWNER_REPO" --body "$(openssl rand -hex 16)"
gh secret set CLOUDINARY_CLOUD_NAME -R "$OWNER_REPO" --body "<copy-tu-VPS>"
gh secret set CLOUDINARY_API_KEY -R "$OWNER_REPO"    --body "<copy-tu-VPS>"
gh secret set CLOUDINARY_API_SECRET -R "$OWNER_REPO" --body "<copy-tu-VPS>"
gh secret set EMAIL_SMTP_USER -R "$OWNER_REPO" --body "<copy-tu-VPS>"
gh secret set EMAIL_SMTP_KEY -R "$OWNER_REPO"  --body "<copy-tu-VPS>"
gh secret set PAYOS_CLIENT_ID -R "$OWNER_REPO"   --body "<copy-tu-VPS>"
gh secret set PAYOS_API_KEY -R "$OWNER_REPO"     --body "<copy-tu-VPS>"
gh secret set PAYOS_CHECKSUM_KEY -R "$OWNER_REPO" --body "<copy-tu-VPS>"
# (AI provider keys *_API_KEY: copy y nguyên từ VPS)
```
Verify: `gh secret list -R "$OWNER_REPO"` đủ các nhóm trên. Không còn variable
smoke nào (pipeline `smoke-all` không đọc variable).

## Bước 4 — Provision box (làm 1 lần)

1. Lightsail Console → **Create instance** → Region **Singapore** → Linux →
   **Ubuntu 22.04** → Plan **\$12 Small-2GB** → tên `closy-api-v2` → Create.
2. **Static IP**: Networking → **Create static IP** → attach vào instance (ghi IP →
   secret `AWS_HOST`). Không dùng public IP động. **Giữ IP này xuyên suốt các chu kỳ.**
3. **Firewall**: mở `22` (giới hạn IP bạn), `80`, `443`. Đóng hết còn lại.
4. SSH vào box, cài Docker:
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # đăng nhập lại
docker --version && docker compose version
```

## Bước 5 — Chuẩn bị `/opt/closy-aws` (làm 1 lần, backup để tạo lại fresh)

```bash
sudo mkdir -p /opt/closy-aws && sudo chown $USER:$USER /opt/closy-aws
cd /opt/closy-aws
# copy .env.aws.example từ repo rồi điền:
# - DB_*/REDIS_*/RABBITMQ_*: từ dashboard managed (Bước 2)
# - JWT/Cloudinary/Brevo/PayOS/AI keys: COPY NGUYÊN từ VPS .env.production
# - ELASTICSEARCH_ENABLED=false (box không chạy ES)
# - INGRESS_*_URL: KHÔNG set (compose hardcode *-http)
chmod 600 .env.aws
```
**Lưu `.env.aws` vào password manager ngay** — tạo lại fresh cần nó, không commit bao giờ.

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
   cấp cert trên box:
```bash
cd /opt/closy-aws
docker compose --env-file .env.aws -f deployments/docker-compose.aws.yml \
  --profile maintenance run --rm certbot certonly --standalone \
  -d api-v2.closy.hycat.online --agree-tos -m <email>
```
Renew sau này qua webroot (`certbot_www` đã mount; cron gọi `certbot renew` + reload nginx).
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

1. Reunify Supabase → VPS (mục 9.3 dưới đây) — VPS gom đủ ghi cả 2 phía.
2. Trỏ DNS `api-v2` về VPS + `NGINX_MODE=services-redirect` trên VPS (hết grace quay
   về `services`).
3. Delete instance trên Lightsail Console (snapshot **tùy chọn** — mục 9.3... xem dưới).
4. KHÔNG hủy gì khác: managed giữ, static IP giữ (unattached), secrets GitHub giữ.

### 9.2 Tạo lại fresh (trước demo vài ngày)

1. Create instance mới (cùng region/blueprint/plan) → attach **static IP cũ** →
   firewall → Docker (như Bước 4, khỏi cấp IP mới).
2. `.env.aws` từ password manager (không có backup thì dừng lại).
3. DNS `api-v2` → IP box → certbot (Bước 7.1) → tag `aws-v*` → Approve → smoke IP.
4. Seed VPS → Supabase (Bước 6) → quan sát. DNS đã đúng nên không đổi lần 2.

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
repo và AWS account (\$0 + alarm) giữ nguyên.

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

Gate: 6 container healthy; `api-v2` 200 công khai + cert auto-renew; VPS không đổi gì
(trừ chủ động cắt); billing trong hạn mức; mỗi lần xóa: DNS đúng pha + reunify xong.
