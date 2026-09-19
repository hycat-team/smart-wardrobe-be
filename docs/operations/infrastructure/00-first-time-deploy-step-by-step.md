# 00 — VPS lên 3-service step-by-step (GĐ1)

> Runbook đưa VPS từ monolith sang 3 service compose (`NGINX_MODE=services`).
> Cloud Run đã loại khỏi kiến trúc (chưa từng tồn tại) — không còn § GCP/Cloud Run/token-helper.
> Box ngoài AWS (`api-v2`) làm sau ở GĐ2 (runbook `15-aws-compute.md`); managed
> (Supabase/Redis Cloud/CloudAMQP) đăng ký khi tới GĐ2.
> **Thực hiện tuần tự từ trên xuống, không bỏ bước.**
>
> Mỗi secret đều ghi rõ: **nguồn lấy → giá trị/định dạng → lệnh set → cách verify.**
> Cuối §5 có block lệnh copy-paste và lệnh kiểm tra toàn bộ secret/variable.
>
> **Điền placeholder trước khi chạy** (thay trong toàn bộ tài liệu):
> `<OWNER/REPO>` = `hycat-team/smart-wardrobe-be` ·
> `<VPS_HOST>` · `<VPS_USER>` · `<VPS_SSH_PORT>`.
>
> Biến môi trường app đầy đủ: xem `08-environment-variables.md`.

## 0. Đăng ký nền tảng & lấy giá trị

Bạn chỉ cần **VPS + domain Vercel** có sẵn; các nền tảng còn lại đăng ký mới.
Mỗi mục dưới ghi rõ **bấm vào đâu để lấy giá trị** — điền luôn vào §5.

### 0.1 GitHub (repo đã có: `<OWNER/REPO>`)

**a) Tạo 2 Environments** (để bật approval gate + chỗ chứa secret theo môi trường):
1. Mở `https://github.com/<OWNER/REPO>/settings/environments`.
2. **New environment** → nhập `aws-production` → **Configure environment**.
   - (Tuỳ chọn) tick **Required reviewers** → thêm người duyệt (mỗi lần deploy box ngoài phải approve).
3. Lặp lại tạo `vps-production` (nên bật Required reviewers cho môi trường này).
4. Verify:
   ```bash
   gh api repos/<OWNER/REPO>/environments --jq '.environments[].name'
   # aws-production
   # vps-production
   ```

> GitHub sẽ **tự tạo** environment nếu workflow tham chiếu mà chưa có; tạo thủ công
> trước là để bật reviewer.

**b) Tạo GHCR read token cho VPS** (VPS dùng để pull ảnh private):
1. Mở `https://github.com/settings/tokens/new` (Personal access token **classic**).
2. Note: `closy-vps-ghcr-read`; Expiration: tuỳ bạn (nên 90 ngày, nhớ lịch rotate).
3. Tick scope **`read:packages`** (không cần scope nào khác) → **Generate token**.
4. Copy token (`ghp_...`) → secret `GHCR_READ_TOKEN`. `GHCR_USERNAME` = username GitHub
   sở hữu token (ví dụ `hycat-team-bot` hoặc user cá nhân của bạn).

### 0.2 Cloud compute ngoài (GĐ2 — chưa làm)

Cloud Run đã loại (chưa từng tồn tại: không project, không secret, không deploy).
Compute ngoài dùng **AWS** (box Lightsail/EC2 Singapore) ở GĐ2 — xem runbook
`15-aws-compute.md` + workflow `deploy-aws.yml`. GĐ1 bỏ qua mục này.

### 0.3 Supabase Pro (Postgres + pgvector)
1. Đăng ký https://supabase.com → **New project** (plan Pro hoặc trial).
2. Region chọn **Southeast Asia (Singapore)** để gần `<REGION>`.
3. Vào **Project Settings → Database → Connection string**; lưu lại:
   - **Connection pooling / Transaction** (port **6543**): host dạng
     `aws-0-ap-southeast-1.pooler.supabase.com`, user dạng `postgres.<project-ref>`.
   - **Direct connection** (port **5432**): host dạng `db.<project-ref>.supabase.co`,
     user `postgres`.
   - **Database password** (đặt khi tạo project; nếu quên → **Reset database password**).

### 0.4 Redis Cloud Essentials 250MB
1. Đăng ký https://redis.io/cloud → **New subscription** → plan **Essentials**, 250MB.
2. Sau khi tạo: mở subscription → tab **Configuration** (hoặc **Connect**), lưu:
   - **Public endpoint** (host) — thường `redis-XXXXX.c...cloud.redislabs.com`.
   - **Port** riêng (thường `12000+`, KHÔNG phải 6379) → `REDIS_PORT`.
   - **Default user password** → `REDIS_PASSWORD`.
3. Bật **TLS** (managed workflow tự set `REDIS_TLS_ENABLED=true`, không cần nhập lại).

### 0.5 CloudAMQP Tough Tiger
1. Đăng ký https://www.cloudamqp.com → **Create New Instance** (Tough Tiger; có thể dùng gói nhỏ/free để thử).
2. Mở instance → tab **Details** → copy:
   - **AMQP URL** dạng `amqps://<user>:<pass>@<host>/<vhost>` (amqps = TLS, port `5671`).
   - Tách ra: `RABBITMQ_HOST` (host), `RABBITMQ_PORT` (`5671`), `RABBITMQ_USER`, `RABBITMQ_PASSWORD`.
   - Nên tạo **vhost riêng** (tab **Vhosts** → Create) rồi dùng vhost đó.

### 0.6 Cloudflare Pro (funded, KHÔNG nằm trong đường API)
- **Không bắt buộc** cho việc host API; chỉ đăng ký khi có mục đích khác. DNS vẫn ở
  **Vercel**, không NS-delegate, không Cloudflare trong đường API (FR-025/029).

### 0.7 VPS + domain (đã có)
- VPS đang chạy monolith (`api.closy`); Docker + `/opt/closy`.
- `hycat.online` giữ DNS ở **Vercel**; sẽ thêm **subdomain `api-v2`** ở §7.

### 0.8 Điều kiện tiên quyết (checklist)
- [ ] GitHub repo có quyền Actions + Packages (GHCR); đã tạo PAT `read:packages`.
- [ ] VPS hiện hữu đang chạy monolith (`api.closy.hycat.online`), có Docker + `/opt/closy`.
- [ ] Domain `hycat.online` quản lý DNS ở **Vercel** (không NS-delegate).
- [ ] Đã cài: `gh`, `docker`, `git`.
- [ ] (GĐ2) Supabase / Redis Cloud / CloudAMQP đã tạo, đã có host/port/password (mục 0.3–0.5).

## 1. Git: push tag rollback (BẮT BUỘC)

`deploy-production.yml` verify `refs/tags/monolith-legacy`; nếu tag chưa lên remote,
workflow rollback fail.

```bash
git tag -a monolith-legacy <BASELINE_SHA> -m "frozen monolith-green before 020"
git push origin monolith-legacy
git ls-remote --tags origin | grep monolith-legacy   # phải thấy
```

## 2. Cloud one-time setup (GĐ2 — bỏ qua ở GĐ1)

Cloud Run/GCP đã loại. Setup compute + managed ngoài ở GĐ2 theo runbook
`15-aws-compute.md` (mục 0.3–0.5 giữ nguyên cách lấy host/port/password khi tới GĐ2).

## 3. Supabase (Postgres + pgvector)

1. Tạo **1 project Pro**. Trong **SQL editor**, chạy:
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
   ```
2. `search_path` **server-side** cho **role thật** (Supabase thường là `postgres`;
   user pooler `postgres.<project-ref>` chỉ là user đăng nhập, không phải role) —
   không truyền `search_path` qua DSN runtime pooler:
   ```sql
   alter role postgres set search_path =
     public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared;
   ```
3. Lấy 4 giá trị (Project Settings → Database → Connection string) và map vào secret:

   | Secret | Lấy ở đâu | Ví dụ |
   |---|---|---|
   | `DB_HOST` | **Pooler / Transaction** host | `aws-0-ap-southeast-1.pooler.supabase.com` |
   | `DB_PORT` | Pooler port | `6543` |
   | `DB_LISTENER_HOST` | **Direct connection** host | `db.abcdxyz.supabase.co` |
   | `DB_LISTENER_SSLMODE` | Managed = TLS | `require` |
   | `DB_USER` | Pooler user | `postgres.abcdxyz` |
   | `DB_PASSWORD` | Database password | *(đã lưu ở 0.3)* |
   | `DB_NAME` | Database name | `postgres` |

   > `DB_LISTENER_PORT` không cần set (mặc định `5432`).

> Migration chạy lúc boot. Lưu ý đã biết: migration dùng **connection direct** riêng
> (`DB_LISTENER_HOST:5432`); runtime query qua pooler 6543 (`DATABASE_QUERY_EXEC_MODE=auto`).
> Vì vậy `DB_LISTENER_HOST` là **bắt buộc** ở managed (workflow enforce).

## 4. Redis Cloud + CloudAMQP — map vào secret

Lấy giá trị ở mục 0.4 / 0.5 rồi map:

| Secret | Lấy ở đâu | Định dạng |
|---|---|---|
| `REDIS_HOST` | Redis Cloud → Configuration → Public endpoint | `redis-XXXXX.c...redislabs.com` |
| `REDIS_PORT` | Redis Cloud → Configuration → Port | số riêng, vd `12000` |
| `REDIS_PASSWORD` | Redis Cloud → Security → Default user password | chuỗi |
| `RABBITMQ_HOST` | CloudAMQP → Details → AMQP URL (phần host) | `...rmq.cloudamqp.com` |
| `RABBITMQ_PORT` | CloudAMQP (amqps) | `5671` |
| `RABBITMQ_USER` | CloudAMQP → Details → AMQP URL | chuỗi |
| `RABBITMQ_PASSWORD` | CloudAMQP → Details → AMQP URL | chuỗi |

## 5. GitHub: Environments + Secrets (từng bước)

### 5.1 Secret nên đặt ở đâu?
- **Repo-level** (`Settings → Secrets and variables → Actions`): mọi job đọc được
  (kể cả job có `environment:`). **Đơn giản nhất — khuyến nghị.**
- **Environment-level** (`Settings → Environments → <env> → Environment secrets`):
  chỉ job khai `environment: <env>` mới đọc được. Nếu muốn tách quyền:
  - `aws-production` ← `AWS_*` + toàn bộ nhóm app (DB/Redis/RabbitMQ/JWT/...) cho box ngoài.
  - `vps-production` ← `VPS_*`, `GHCR_*`.

> Vì workflow hiện tại chỉ cần secret có mặt ở 1 trong 2 scope, đặt repo-level là
> chạy được hết. Dưới đây liệt kê lệnh cho **repo-level**; nếu muốn environment-level
> thì thêm `--env <tên-env>`.

### 5.2 Set nhanh qua `gh` (copy-paste, sửa giá trị)

```bash
OWNER_REPO="<OWNER/REPO>"

# --- Database (Supabase, §3 — GĐ2; GĐ1 VPS dùng local, bỏ qua) ---
gh secret set DB_USER -R "$OWNER_REPO"     --body "postgres.<project-ref>"
gh secret set DB_PASSWORD -R "$OWNER_REPO" --body "<db-password>"
gh secret set DB_NAME -R "$OWNER_REPO"     --body "postgres"
gh secret set DB_HOST -R "$OWNER_REPO"     --body "aws-0-ap-southeast-1.pooler.supabase.com"
gh secret set DB_PORT -R "$OWNER_REPO"     --body "6543"
gh secret set DB_LISTENER_HOST -R "$OWNER_REPO"     --body "db.<project-ref>.supabase.co"
gh secret set DB_LISTENER_SSLMODE -R "$OWNER_REPO"  --body "require"

# --- Redis (Redis Cloud, §4) ---
gh secret set REDIS_HOST -R "$OWNER_REPO"     --body "redis-XXXXX.c...redislabs.com"
gh secret set REDIS_PORT -R "$OWNER_REPO"     --body "12000"
gh secret set REDIS_PASSWORD -R "$OWNER_REPO" --body "<redis-password>"

# --- RabbitMQ (CloudAMQP, §4) ---
gh secret set RABBITMQ_HOST -R "$OWNER_REPO"     --body "xxx.rmq.cloudamqp.com"
gh secret set RABBITMQ_PORT -R "$OWNER_REPO"     --body "5671"
gh secret set RABBITMQ_USER -R "$OWNER_REPO"     --body "<rabbit-user>"
gh secret set RABBITMQ_PASSWORD -R "$OWNER_REPO" --body "<rabbit-password>"

# --- JWT (tự sinh; xem 5.3) ---
gh secret set JWT_ALGORITHM -R "$OWNER_REPO" --body "HS256"
gh secret set JWT_SECRET -R "$OWNER_REPO"    --body "$(openssl rand -base64 48)"
# (Tuỳ chọn ES256 — xem 5.3; HS256 thì KHÔNG cần private/public key)

# --- App secrets ---
gh secret set SWAGGER_ACCESS_CODE -R "$OWNER_REPO" --body "$(openssl rand -hex 16)"
gh secret set CLOUDINARY_CLOUD_NAME -R "$OWNER_REPO" --body "<cloud-name>"
gh secret set CLOUDINARY_API_KEY -R "$OWNER_REPO"    --body "<api-key>"
gh secret set CLOUDINARY_API_SECRET -R "$OWNER_REPO" --body "<api-secret>"
gh secret set EMAIL_SMTP_USER -R "$OWNER_REPO" --body "<brevo-login-email>"
gh secret set EMAIL_SMTP_KEY -R "$OWNER_REPO"  --body "<brevo-smtp-key>"
gh secret set PAYOS_CLIENT_ID -R "$OWNER_REPO"   --body "<payos-client-id>"
gh secret set PAYOS_API_KEY -R "$OWNER_REPO"     --body "<payos-api-key>"
gh secret set PAYOS_CHECKSUM_KEY -R "$OWNER_REPO" --body "<payos-checksum-key>"

# --- VPS + GHCR (§0.1b) ---
gh secret set GHCR_USERNAME -R "$OWNER_REPO"     --body "<github-username>"
gh secret set GHCR_READ_TOKEN -R "$OWNER_REPO"   --body "ghp_xxx"
gh secret set VPS_HOST -R "$OWNER_REPO"          --body "<VPS_HOST>"
gh secret set VPS_USER -R "$OWNER_REPO"          --body "<VPS_USER>"
gh secret set VPS_SSH_PORT -R "$OWNER_REPO"      --body "22"
# SSH key + known_hosts là nội dung nhiều dòng → set từ file (xem 5.3)

# --- AWS box ngoài (GĐ2; runbook 15) ---
gh secret set AWS_HOST -R "$OWNER_REPO"          --body "<BOX_IP>"
gh secret set AWS_USER -R "$OWNER_REPO"          --body "ubuntu"
gh secret set AWS_SSH_PORT -R "$OWNER_REPO"      --body "22"
# SSH key + known_hosts là nội dung nhiều dòng → set từ file (xem 5.3)
```

> **INGRESS_ACCOUNT_URL / INGRESS_COMMERCE_URL / INGRESS_AI_URL** trên VPS là tên
> service compose (`http://account:8080`...) — điền trong `.env.production`, không phải secret.

### 5.3 Từng secret: lấy ở đâu + cách set

| Secret | Lấy ở đâu | Set bằng |
|---|---|---|
| `DB_USER` | Supabase pooler user (§3, GĐ2) / `closy` local (VPS) | `--body` |
| `DB_PASSWORD` | Supabase tạo project (§0.3) | `--body` |
| `DB_NAME` | Supabase database name | `--body "postgres"` |
| `DB_HOST` | Supabase pooler host (§3) | `--body` |
| `DB_PORT` | pooler | `--body "6543"` |
| `DB_LISTENER_HOST` | Supabase direct host (§3) | `--body` |
| `DB_LISTENER_SSLMODE` | managed | `--body "require"` |
| `REDIS_HOST/PORT/PASSWORD` | Redis Cloud (§4) | `--body` |
| `RABBITMQ_HOST/PORT/USER/PASSWORD` | CloudAMQP (§4) | `--body` |
| `JWT_ALGORITHM` | tự chọn | `--body "HS256"` (mặc định) |
| `JWT_SECRET` | tự sinh | `--body "$(openssl rand -base64 48)"` |
| `JWT_PRIVATE_KEY` | chỉ khi dùng **ES256** | `gh secret set JWT_PRIVATE_KEY < ec_private.pem` |
| `JWT_PUBLIC_KEY` | chỉ khi dùng **ES256** | `gh secret set JWT_PUBLIC_KEY < ec_public.pem` |
| `SWAGGER_ACCESS_CODE` | tự sinh | `--body "$(openssl rand -hex 16)"` |
| `CLOUDINARY_CLOUD_NAME/API_KEY/API_SECRET` | Cloudinary Dashboard → Product Environment Credentials | `--body` |
| `EMAIL_SMTP_USER` | Brevo → SMTP & API → SMTP (login) | `--body "<email>"` |
| `EMAIL_SMTP_KEY` | Brevo → SMTP key | `--body` |
| `PAYOS_CLIENT_ID/API_KEY/CHECKSUM_KEY` | PayOS merchant dashboard → Kênh thanh toán | `--body` |
| `VPS_HOST` | VPS của bạn | `--body "<ip-or-host>"` |
| `VPS_USER` | SSH user (vd `root`/`ubuntu`) | `--body` |
| `VPS_SSH_PORT` | SSH port | `--body "22"` |
| `VPS_SSH_KEY` | private key | `gh secret set VPS_SSH_KEY < ~/.ssh/id_ed25519` |
| `VPS_KNOWN_HOSTS` | `ssh-keyscan` | `ssh-keyscan -p 22 <VPS_HOST> > known_hosts` rồi `gh secret set VPS_KNOWN_HOSTS < known_hosts` |
| `GHCR_USERNAME` | GitHub username (§0.1b) | `--body` |
| `GHCR_READ_TOKEN` | PAT `read:packages` (§0.1b) | `--body "ghp_..."` |
| `AWS_HOST` | IP box ngoài (static IP, runbook `15` §2) | `--body "<ip>"` |
| `AWS_USER` | SSH user box ngoài (vd `ubuntu`) | `--body` |
| `AWS_SSH_PORT` | SSH port box ngoài | `--body "22"` |
| `AWS_SSH_KEY` | private key box ngoài | `gh secret set AWS_SSH_KEY < ~/.ssh/aws_box` |
| `AWS_KNOWN_HOSTS` | `ssh-keyscan` box ngoài | `ssh-keyscan -p 22 <BOX_IP> > aws_known_hosts` rồi `gh secret set AWS_KNOWN_HOSTS < aws_known_hosts` |

ES256 (tuỳ chọn, FR-022): sinh cặp EC P-256 rồi set từ file:
```bash
openssl ecparam -genkey -name prime256v1 -noout -out ec_private.pem
openssl ec -in ec_private.pem -pubout -out ec_public.pem
gh secret set JWT_PRIVATE_KEY -R "<OWNER/REPO>" < ec_private.pem
gh secret set JWT_PUBLIC_KEY  -R "<OWNER/REPO>" < ec_public.pem
```

### 5.4 Variables (Settings → Secrets and variables → Actions → Variables)

| Variable | Dùng cho | Lấy giá trị | Lệnh |
|---|---|---|---|
| `PRODUCTION_HEALTH_URL` | `deploy-production.yml` smoke test | health URL của monolith hiện tại | `gh variable set PRODUCTION_HEALTH_URL -R "<OWNER/REPO>" --body "https://api.closy.hycat.online/api/v1/health"` |

> Smoke của `deploy-vps-services.yml` / `deploy-aws.yml` bắn thẳng vào container vừa
> deploy (poll health qua SSH) nên không cần variable nào.

### 5.5 Verify đã đủ secret/variable

```bash
gh secret list   -R "<OWNER/REPO>"
gh variable list -R "<OWNER/REPO>"
# Nếu đặt ở environment:
gh secret list --env aws-production -R "<OWNER/REPO>"
gh secret list --env vps-production -R "<OWNER/REPO>"
```
Checklist tối thiểu để `deploy-vps-services.yml` chạy (GĐ1):
- [ ] `VPS_*` (5) + `GHCR_*` (2)

Checklist bổ sung khi tới GĐ2 (`deploy-aws.yml`):
- [ ] `AWS_HOST/USER/SSH_PORT/SSH_KEY/KNOWN_HOSTS`
- [ ] `DB_*` (7 biến), `REDIS_*` (3), `RABBITMQ_*` (4)
- [ ] `JWT_SECRET`, `JWT_ALGORITHM`
- [ ] `SWAGGER_ACCESS_CODE`, `CLOUDINARY_*`, `EMAIL_SMTP_*`, `PAYOS_*`

> Workflow box ngoài (GĐ2) tự set (không tạo secret): `SERVER_ENV=vps`,
> `APP_MODE=http|worker`, `DATABASE_MIGRATE_ON_START=true`,
> `DATABASE_MIGRATION_TABLE` (per-service), `ELASTICSEARCH_ENABLED=false`.
> Riêng managed TLS (`DB_SSLMODE`/`DB_LISTENER_SSLMODE=require`,
> `REDIS_TLS_ENABLED=true`, `RABBITMQ_TLS_ENABLED=true`) lấy từ secret/plaintext
> đã set ở trên theo dashboard từng nền tảng.

## 6. Deploy 3 service lên VPS (GĐ1)

```bash
gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all
RUN_ID=$(gh run list --workflow deploy-vps-services.yml -R "<OWNER/REPO>" --limit 1 \
  --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" -R "<OWNER/REPO>"
```
> Nếu environment `vps-production` có Required reviewers, run sẽ **chờ approve** — vào
> `Actions → run → Review deployments → Approve`.

Workflow build matrix per service → push GHCR (pin digest) → `compose up -d <svc>` trên
VPS; 1 service fail không chặn 2 còn lại. Gate smoke poll health đúng container vừa
deploy qua SSH (miễn nhiễm trạng thái Nginx).

Deploy box ngoài AWS làm ở GĐ2 (`15-aws-compute.md` + `deploy-aws.yml`).

### 6.1 Dispatch các workflow (chạy thủ công) hoặc push tag

| Workflow | Dispatch tay | Push tag (all services) | Input |
|---|---|---|---|
| `deploy-vps-services.yml` (3 service trên VPS) | `gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all` | `git tag -a vps-v1.0.0 -m "..." && git push origin vps-v1.0.0` | `services` (tag push = all) |
| `deploy-aws.yml` (GĐ2, box ngoài) | `gh workflow run deploy-aws.yml -R "<OWNER/REPO>" --ref main -f services=all` | `git tag -a aws-v1.0.0 -m "..." && git push origin aws-v1.0.0` | `services` (tag push = all) |
| `deploy-production.yml` (rollback monolith khẩn cấp) | `gh workflow run deploy-production.yml -R "<OWNER/REPO>" --ref main -f confirm=rollback` | (không có — bắt buộc dispatch + xác nhận) | `confirm` phải = `rollback` |

> Quy ước tag: `vps-*` chỉ chạy trục VPS, `aws-*` chỉ chạy trục box ngoài — 2 trục
> độc lập, không tag nào deploy cả 2. Tag push vẫn qua approval gate của environment
> (run chờ Approve như dispatch). Deploy lẻ 1 service chỉ làm được bằng dispatch
> (`-f services=commerce`). Image GHCR được alias thêm tên tag để truy vết.

- Xem tiến độ: `gh run list -R "<OWNER/REPO>" --limit 5` rồi `gh run watch <id> -R "<OWNER/REPO>"`.
- Nếu environment có Required reviewers → run dừng ở trạng thái chờ; vào
  `Actions → run → Review deployments → Approve`.
- `deploy-vps-services` cần VPS đã có `/opt/closy/.env.production` (§7) và secret `VPS_*` (§5).

## 7. VPS: vào mode `services` (cắt monolith → 3 service)

0. Deploy 3 service lên VPS **trước** để có `.env.deploy` chứa `ACCOUNT/COMMERCE/AI_IMAGE`
   (pin digest) — nginx `depends_on` 3 service nên thiếu 3 biến này là compose báo lỗi ngay:
   ```bash
   gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all
   ```
   Mọi lệnh `docker compose` dưới đây phải kèm **cả 2 env-file**:
   `--env-file .env.production --env-file .env.deploy`.

1. Chuẩn bị `/opt/closy` — **copy từ `.env.production.example`** rồi điền các biến VPS:
   ```bash
   cd /opt/closy
   cp /duong/dan/repo/.env.production.example .env.production
   ```
   Điền tối thiểu (xem `06-vps-nginx-ingress-cloudrun.md` §3):

   | Biến trong `.env.production` | Giá trị |
   |---|---|
   | `API_DOMAIN` | `api.closy.hycat.online` |
   | `API_V2_DOMAIN` | `api-v2.closy.hycat.online` |
   | `NGINX_MODE` | `services` |
   | `DB_HOST`/`DB_PORT`/`DB_LISTENER_HOST`/`DB_*` | **luôn Postgres local** (`postgres`) — cấm trỏ ra Supabase (luật isolation, xem `01` §4) |

   Không còn `COMPOSE_PROFILES`, `*_RUN_URL`, `TOKEN_HELPER_*`.

2. **Cert Let's Encrypt** cho `api.closy.hycat.online`. Do Nginx đang chạy (monolith),
   cấp cert bằng standalone (giải phóng port 80) rồi mở lại:
   ```bash
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml stop nginx
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
     --profile maintenance run --rm certbot certonly --standalone \
     -d api.closy.hycat.online --expand --agree-tos -m <email>
   ```
   (`docker stop nginx` không trúng container thật `closy-prod-nginx-1`; phải stop qua compose.)
   (renewal sau này qua webroot; `certbot_www` đã mount sẵn.)
   Cert `api-v2` chỉ cấp khi cần redirect grace (DNS api-v2 đã về VPS).
3. Guard isolation (chạy trên VPS, chỉ in tên biến khi FAIL — không in giá trị):
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
4. Dừng backend cũ rồi bật ingress 3-service:
   ```bash
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
     stop backend || true
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
     up -d --force-recreate nginx
   docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml ps
   ```
   > Không bao giờ `down -v` trên prod (xóa volumes `postgres_data`... = mất dữ liệu).
   > Backup DB trước mọi lần recreate postgres.
   > Rollback: `NGINX_MODE=rollback` + chạy `deploy-production.yml` (§10).
5. Smoke:
   ```bash
   curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health   # 200 qua 3 service VPS
   curl -sSI https://api.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"
   docker ps --filter name=backend --format '{{.Names}}'                                    # trống
   ```

## 8. Dump dữ liệu VPS → Supabase (GĐ2)

Theo `10-data-migration-vps-supabase.md` (backup + checksum + re-dump tại cutover).
Chỉ làm sau khi đã đăng ký Supabase ở GĐ2.

## 9. Gate VPS `services`

- [ ] `api.closy` 200 qua 3 service VPS (route + SSE + auth + upload).
- [ ] Backend đã dừng; VPS infra local-only (guard pass).
- [ ] 0 request qua Cloudflare; SSE stream dài không đứt.
- [ ] Ghi mốc thời gian + checksum.

## 10. Vận hành / chuyển mode / rollback

- Đổi `NGINX_MODE` trong `.env.production` rồi recreate nginx (luôn kèm cả 2 env-file):
  ```bash
  docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml \
    up -d --force-recreate nginx
  ```
  `services` (api.closy→3 service VPS, monolith dừng) ·
  `services-redirect` (api.closy→3 service + api-v2→301, grace sau khi tắt box ngoài) ·
  `rollback` (api.closy→monolith, api-v2→301, khẩn cấp).

### 10.1 Vào mode `services` (chi tiết cắt monolith)

1. Deploy 3 service mới nhất (§6) + smoke nội bộ qua `docker exec` từng container.
2. Dừng backend cũ: `... stop backend || true`; verify `docker ps` trống backend.
3. Đặt `NGINX_MODE=services` trong `/opt/closy/.env.production` (xóa `COMPOSE_PROFILES`,
   `*_RUN_URL`, `TOKEN_HELPER_*` nếu còn từ trước).
4. Recreate nginx (§7 bước 4) + smoke `api.closy` 200 qua 3 service VPS.
5. DB: VPS local là live duy nhất ở GĐ1 (Supabase chưa có).
- Gom về VPS khi cắt box ngoài: `../repatriate-to-vps.md` + `12-repatriate-vps.md`.
- Rollback khẩn cấp: `deploy-production.yml` (đổi `NGINX_MODE=rollback` **trước** khi chạy;
  workflow tự recreate nginx).

## 11. Known limitations (đọc trước khi run)

- **Migration đã tách sang direct connection** (Option A): Goose + session advisory
  lock mở connection riêng từ `DB_LISTENER_HOST`/`DB_LISTENER_PORT`/`DB_LISTENER_SSLMODE`
  (5432), runtime query vẫn qua pooler 6543. Vì vậy `DB_LISTENER_HOST` là **bắt buộc**
  ở managed (workflow enforce).
- **`:latest` chỉ ở CI**; production pin theo `<git-sha>`/digest.
- Ảnh private: **không** public; GHCR cần token đọc trên VPS.
