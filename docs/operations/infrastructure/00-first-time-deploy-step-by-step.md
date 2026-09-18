# 00 — First-time deploy step-by-step (managed + VPS song song)

> Runbook triển khai lần đầu: Cloud Run (3 service) + VPS Nginx ingress chạy song song
> với monolith. **Thực hiện tuần tự từ trên xuống, không bỏ bước.**
>
> Mỗi secret đều ghi rõ: **nguồn lấy → giá trị/định dạng → lệnh set → cách verify.**
> Cuối §5 có block lệnh copy-paste và lệnh kiểm tra toàn bộ secret/variable.
>
> **Điền placeholder trước khi chạy** (thay trong toàn bộ tài liệu):
> `<OWNER/REPO>` = `hycat-team/smart-wardrobe-be` · `<PROJECT>` = GCP project id
> (phải khớp `AR_PROJECT` trong `deploy-platform.yml`, mặc định `smart-wardrobe`) ·
> `<REGION>` = `asia-southeast1` · `<VPS_HOST>` · `<VPS_USER>` · `<VPS_SSH_PORT>`.
>
> Biến môi trường app đầy đủ: xem `08-environment-variables.md`.

## 0. Đăng ký nền tảng & lấy giá trị

Bạn chỉ cần **VPS + domain Vercel** có sẵn; các nền tảng còn lại đăng ký mới.
Mỗi mục dưới ghi rõ **bấm vào đâu để lấy giá trị** — điền luôn vào §5.

### 0.1 GitHub (repo đã có: `<OWNER/REPO>`)

**a) Tạo 2 Environments** (để bật approval gate + chỗ chứa secret theo môi trường):
1. Mở `https://github.com/<OWNER/REPO>/settings/environments`.
2. **New environment** → nhập `gcp-production` → **Configure environment**.
   - (Tuỳ chọn) tick **Required reviewers** → thêm người duyệt (mỗi lần deploy Cloud Run phải approve).
3. Lặp lại tạo `vps-production` (nên bật Required reviewers cho môi trường này).
4. Verify:
   ```bash
   gh api repos/<OWNER/REPO>/environments --jq '.environments[].name'
   # gcp-production
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

### 0.2 Google Cloud (Cloud Run + Artifact Registry)
1. Tạo Google account → https://console.cloud.google.com → **Start free trial** (cần thẻ; nhận credit trial).
2. **Tạo project**: Console → thanh chọn project → **New project** → ID phải **khớp `AR_PROJECT`**
   (mặc định `smart-wardrobe`; nếu đặt id khác thì sửa `AR_PROJECT` trong `deploy-platform.yml`).
3. **Bật billing**: Console → Billing → **Link a billing account**.
4. Cài `gcloud`, rồi xác thực:
   ```bash
   gcloud auth login
   gcloud config set project <PROJECT>
   gcloud config list                       # xác nhận account + project đúng
   gcloud projects describe <PROJECT> --format="value(projectNumber)"   # ghi lại PROJECT_NUMBER
   ```
5. Phần API/Artifact Registry/WIF/SA/billing cap làm ở **§2** (không làm ở đây).

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
- [ ] GCP project trial đã bật billing (id khớp `AR_PROJECT`), đã có `PROJECT_NUMBER`.
- [ ] GitHub repo có quyền Actions + Packages (GHCR); đã tạo PAT `read:packages`.
- [ ] Supabase / Redis Cloud / CloudAMQP đã tạo, đã có host/port/password (mục 0.3–0.5).
- [ ] VPS hiện hữu đang chạy monolith (`api.closy`), có Docker + Nginx/`/opt/closy`.
- [ ] Domain `hycat.online` quản lý DNS ở **Vercel** (không NS-delegate).
- [ ] Đã cài: `gcloud`, `gh`, `docker`, `git`.

## 1. Git: push tag rollback (BẮT BUỘC)

`deploy-production.yml` verify `refs/tags/monolith-legacy`; nếu tag chưa lên remote,
workflow rollback fail.

```bash
git tag -a monolith-legacy <BASELINE_SHA> -m "frozen monolith-green before 020"
git push origin monolith-legacy
git ls-remote --tags origin | grep monolith-legacy   # phải thấy
```

## 2. GCP one-time setup

```bash
gcloud auth login
gcloud config set project <PROJECT>
gcloud services enable run.googleapis.com artifactregistry.googleapis.com \
  iam.googleapis.com iamcredentials.googleapis.com cloudresourcemanager.googleapis.com \
  monitoring.googleapis.com billingbudgets.googleapis.com
```

### 2.1 Artifact Registry (khớp `AR_REPO`)

```bash
gcloud artifacts repositories create smart-wardrobe \
  --repository-format=docker --location=asia-southeast1 \
  --description="Closy service images"
```
Verify: `gcloud artifacts repositories list --location=asia-southeast1`.

### 2.2 Deployer SA + Workload Identity Federation (cho GitHub Actions)

```bash
gcloud iam service-accounts create github-deployer --display-name="GitHub Actions deployer"

gcloud projects add-iam-policy-binding <PROJECT> \
  --member="serviceAccount:github-deployer@<PROJECT>.iam.gserviceaccount.com" --role="roles/run.admin"
gcloud projects add-iam-policy-binding <PROJECT> \
  --member="serviceAccount:github-deployer@<PROJECT>.iam.gserviceaccount.com" --role="roles/artifactregistry.writer"
gcloud projects add-iam-policy-binding <PROJECT> \
  --member="serviceAccount:github-deployer@<PROJECT>.iam.gserviceaccount.com" --role="roles/iam.serviceAccountUser"

gcloud iam workload-identity-pools create github --location=global
gcloud iam workload-identity-pools providers create-oidc github \
  --location=global --workload-identity-pool=github \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository" \
  --attribute-condition="assertion.repository=='<OWNER/REPO>'"

PROJECT_NUMBER=$(gcloud projects describe <PROJECT> --format="value(projectNumber)")
gcloud iam service-accounts add-iam-policy-binding github-deployer@<PROJECT>.iam.gserviceaccount.com \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/attribute.repository/<OWNER/REPO>"
```

**Ghi lại 2 giá trị này để set secret ở §5.3:**
- `GCP_WORKLOAD_IDENTITY_PROVIDER` = `projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/providers/github`
- `GCP_DEPLOYER_SERVICE_ACCOUNT` = `github-deployer@<PROJECT>.iam.gserviceaccount.com`

### 2.3 Token-helper SA (Nginx → Cloud Run IAM)

```bash
gcloud iam service-accounts create token-helper --display-name="Nginx token helper"
gcloud iam service-accounts keys create token-helper-sa.json \
  --iam-account=token-helper@<PROJECT>.iam.gserviceaccount.com
```
Sau khi deploy (bước 6) grant `run.invoker` cho **đúng 3 service**:
```bash
for svc in account commerce ai; do
  gcloud run services add-iam-policy-binding smart-wardrobe-$svc \
    --region=asia-southeast1 --member="serviceAccount:token-helper@<PROJECT>.iam.gserviceaccount.com" \
    --role="roles/run.invoker"
done
```
Chép `token-helper-sa.json` lên VPS `/opt/closy/token-helper-sa.json`, `chmod 600`.

### 2.4 Billing cap + alert

Console GCP → Billing → Budgets & alerts: tạo budget cho `<PROJECT>`, cap = ngân sách
trial, alert 50/90/100%. (Hoặc `gcloud billing budgets create --billing-account=<ID>
--budget-amount=<USD> --threshold-rule=percent=0.9`.)

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
  - `gcp-production` ← toàn bộ nhóm GCP + app (DB/Redis/RabbitMQ/JWT/...).
  - `vps-production` ← `VPS_*`, `GHCR_*`.

> Vì workflow hiện tại chỉ cần secret có mặt ở 1 trong 2 scope, đặt repo-level là
> chạy được hết. Dưới đây liệt kê lệnh cho **repo-level**; nếu muốn environment-level
> thì thêm `--env <tên-env>`.

### 5.2 Set nhanh qua `gh` (copy-paste, sửa giá trị)

```bash
OWNER_REPO="<OWNER/REPO>"

# --- GCP (từ §2.2) ---
gh secret set GCP_WORKLOAD_IDENTITY_PROVIDER -R "$OWNER_REPO" \
  --body "projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/github/providers/github"
gh secret set GCP_DEPLOYER_SERVICE_ACCOUNT -R "$OWNER_REPO" \
  --body "github-deployer@<PROJECT>.iam.gserviceaccount.com"

# --- Database (Supabase, §3) ---
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
```

> **INGRESS_ACCOUNT_URL / INGRESS_COMMERCE_URL / INGRESS_AI_URL / INGRESS_TOKEN_AUDIENCE**
> **điền SAU bước 6** (lấy URL `run.app`).

### 5.3 Từng secret: lấy ở đâu + cách set

| Secret | Lấy ở đâu | Set bằng |
|---|---|---|
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | output §2.2 | `--body "projects/<NUMBER>/.../providers/github"` |
| `GCP_DEPLOYER_SERVICE_ACCOUNT` | output §2.2 | `--body "github-deployer@<PROJECT>.iam.gserviceaccount.com"` |
| `DB_USER` | Supabase pooler user (§3) | `--body` |
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
| `INGRESS_ACCOUNT/COMMERCE/AI_URL` | **§6 sau khi deploy** | `--body "https://smart-wardrobe-<svc>-xxx.run.app"` |
| `INGRESS_TOKEN_AUDIENCE` | **§6 sau khi deploy** | `--body "<URL Account run.app>"` |

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
| `VPS_SMOKE_URL` | `deploy-vps-services.yml` smoke test | base URL `api-v2` (**điền sau §7**) | `gh variable set VPS_SMOKE_URL -R "<OWNER/REPO>" --body "https://api-v2.closy.hycat.online"` |

> Để trống `VPS_SMOKE_URL` cũng chạy được — step smoke test tự skip khi variable rỗng.

### 5.5 Verify đã đủ secret/variable

```bash
gh secret list   -R "<OWNER/REPO>"
gh variable list -R "<OWNER/REPO>"
# Nếu đặt ở environment:
gh secret list --env gcp-production -R "<OWNER/REPO>"
gh secret list --env vps-production -R "<OWNER/REPO>"
```
Checklist tối thiểu để `deploy-platform.yml` không chết vì secret rỗng:
- [ ] `GCP_WORKLOAD_IDENTITY_PROVIDER`, `GCP_DEPLOYER_SERVICE_ACCOUNT`
- [ ] `DB_*` (7 biến), `REDIS_*` (3), `RABBITMQ_*` (4)
- [ ] `JWT_SECRET`, `JWT_ALGORITHM`
- [ ] `SWAGGER_ACCESS_CODE`, `CLOUDINARY_*`, `EMAIL_SMTP_*`, `PAYOS_*`
- [ ] `INGRESS_*` (điền ở §6)
- [ ] `VPS_*` (5) + `GHCR_*` (2) cho VPS workflows

> Workflow tự set (không tạo secret): `SERVER_ENV=managed`, `APP_MODE=http|worker`,
> `DB_SSLMODE=require`, `DB_LISTENER_SSLMODE=require`, `DATABASE_MIGRATE_ON_START=true`,
> `DATABASE_MIGRATION_TABLE` (per-service), `REDIS_TLS_ENABLED=true`,
> `RABBITMQ_TLS_ENABLED=true`, `ELASTICSEARCH_ENABLED=false`.

## 6. Deploy Cloud Run

```bash
gh workflow run deploy-platform.yml -R "<OWNER/REPO>" --ref main -f services=all
RUN_ID=$(gh run list --workflow deploy-platform.yml -R "<OWNER/REPO>" --limit 1 \
  --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" -R "<OWNER/REPO>"
```
> Nếu environment `gcp-production` có Required reviewers, run sẽ **chờ approve** — vào
> `Actions → run → Review deployments → Approve`.

Sau khi xong, lấy URL từng service:
```bash
for svc in account commerce ai; do
  echo "$svc = $(gcloud run services describe smart-wardrobe-$svc --region asia-southeast1 --format='value(status.url)')"
done
```
→ set 3 secret ingress (URL `run.app` ở trên):
```bash
gh secret set INGRESS_ACCOUNT_URL  -R "<OWNER/REPO>" --body "https://smart-wardrobe-account-xxx.run.app"
gh secret set INGRESS_COMMERCE_URL -R "<OWNER/REPO>" --body "https://smart-wardrobe-commerce-xxx.run.app"
gh secret set INGRESS_AI_URL       -R "<OWNER/REPO>" --body "https://smart-wardrobe-ai-xxx.run.app"
```
→ grant `run.invoker` cho token-helper SA (§2.3), set `INGRESS_TOKEN_AUDIENCE` = URL Account:
```bash
gh secret set INGRESS_TOKEN_AUDIENCE -R "<OWNER/REPO>" --body "https://smart-wardrobe-account-xxx.run.app"
```
→ **chạy lại** để app nhận ingress URL:
```bash
gh workflow run deploy-platform.yml -R "<OWNER/REPO>" --ref main -f services=all
```

Verify: mỗi service 200 `GET /api/v1/health` (qua token helper hoặc
`gcloud run services proxy smart-wardrobe-account --region asia-southeast1`).

### 6.1 Dispatch các workflow VPS (chạy thủ công)

| Workflow | Lệnh | Input |
|---|---|---|
| `deploy-platform.yml` (Cloud Run) | `gh workflow run deploy-platform.yml -R "<OWNER/REPO>" --ref main -f services=all` | `services` (`all` hoặc `account,commerce,ai`), `sha` (rỗng = HEAD) |
| `deploy-vps-services.yml` (3 service trên VPS) | `gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all` | `services` |
| `deploy-production.yml` (rollback monolith khẩn cấp) | `gh workflow run deploy-production.yml -R "<OWNER/REPO>" --ref main -f confirm=rollback` | `confirm` phải = `rollback` |

- Xem tiến độ: `gh run list -R "<OWNER/REPO>" --limit 5` rồi `gh run watch <id> -R "<OWNER/REPO>"`.
- Nếu environment có Required reviewers → run dừng ở trạng thái chờ; vào
  `Actions → run → Review deployments → Approve`.
- `deploy-vps-services` cần VPS đã có `/opt/closy/.env.production` (§7) và secret `VPS_*` (§5).

## 7. VPS: ingress song song (`api-v2`)

1. Chuẩn bị `/opt/closy` — **copy từ `.env.production.example`** rồi điền các biến VPS:
   ```bash
   cd /opt/closy
   cp /duong/dan/repo/.env.production.example .env.production
   cp token-helper-sa.json /opt/closy/token-helper-sa.json && chmod 600 token-helper-sa.json
   ```
   Điền các biến **chỉ có trên VPS** (giá trị app còn lại lấy từ secrets ở §5):

   | Biến trong `.env.production` | Giá trị |
   |---|---|
   | `API_DOMAIN` | `api.closy.hycat.online` |
   | `API_V2_DOMAIN` | `api-v2.closy.hycat.online` |
   | `NGINX_MODE` | `parallel` |
   | `COMPOSE_PROFILES` | `parallel` |
   | `ACCOUNT_RUN_URL` / `COMMERCE_RUN_URL` / `AI_RUN_URL` | URL `run.app` từ §6 |
   | `TOKEN_HELPER_URL` | `http://token-helper:8082` |
   | `TOKEN_HELPER_SA_KEY_FILE` | `./token-helper-sa.json` |
   | `DB_HOST`/`DB_PORT`/`DB_LISTENER_HOST`/`DB_*` | nếu VPS dùng Postgres local (`postgres`) hoặc Supabase — tuỳ mode |

2. **Cert Let's Encrypt** cho cả 2 hostname. Do Nginx đang chạy (monolith), cấp cert
   bằng standalone (giải phóng port 80) rồi mở lại:
   ```bash
   docker stop nginx || true
   docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
     --profile maintenance run --rm certbot certonly --standalone \
     -d api.closy.hycat.online -d api-v2.closy.hycat.online --expand --agree-tos -m <email>
   ```
   (renewal sau này qua webroot; `certbot_www` đã mount sẵn.)
3. Bật ingress (chỉ nginx + token-helper ở parallel):
   ```bash
   docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
     up -d nginx token-helper
   docker compose --env-file .env.production -f deployments/docker-compose.prod.yml ps
   ```
4. DNS Vercel: tạo bản ghi `api-v2.closy.hycat.online` (A/CNAME) → VPS **sau khi** cert
   + server block sẵn sàng.
5. Smoke:
   ```bash
   curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health   # monolith giữ nguyên
   curl -sS -o /dev/null -w '%{http_code}\n' https://api-v2.closy.hycat.online/api/v1/system-catalog/wardrobe-items
   curl -sSI https://api-v2.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"
   ```

## 8. Dump dữ liệu VPS → Supabase (song song)

Theo `10-data-migration-vps-supabase.md` (backup + checksum + re-dump tại cutover).

## 9. Gate song song (T082)

- [ ] `api-v2` ổn: route 3 service + SSE + auth + upload (SC-014).
- [ ] `api.closy` **0 thay đổi** routing (SC-015); thiếu ID token → Cloud Run 403 (SC-013).
- [ ] 0 request `api-v2` qua Cloudflare (SC-016); SSE stream dài không đứt.
- [ ] Ghi mốc thời gian + checksum.

## 10. Vận hành / chuyển mode / rollback

- Đổi `NGINX_MODE` trong `.env.production` rồi recreate nginx:
  ```bash
  docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
    up -d --force-recreate nginx
  ```
  `parallel` (api.closy→monolith, api-v2→Cloud Run) · `repatriation` (api.closy→3 service,
  api-v2→301) · `rollback` (api.closy→monolith, api-v2→301).
- Repatriation: `../repatriate-to-vps.md` + `12-repatriate-vps.md`.
- Cutover `api.closy`: `../manual-cutover-api-closy-to-cloudrun.md` (owner-triggered).
- Rollback khẩn cấp: `deploy-production.yml` (đổi `NGINX_MODE=rollback` **trước** khi chạy,
  vì workflow chỉ start `backend`).

## 11. Known limitations (đọc trước khi run)

- **Migration đã tách sang direct connection** (Option A): Goose + session advisory
  lock mở connection riêng từ `DB_LISTENER_HOST`/`DB_LISTENER_PORT`/`DB_LISTENER_SSLMODE`
  (5432), runtime query vẫn qua pooler 6543. Vì vậy `DB_LISTENER_HOST` là **bắt buộc**
  ở managed (workflow enforce).
- **Token-helper SA key** là bí mật file trên VPS; rotation thủ công (tạo key mới,
  redeploy helper, revoke key cũ) — xem `13-stop-vps-and-revoke-secrets.md`.
- **`:latest` chỉ ở CI**; production pin theo `<git-sha>`/digest.
- Ảnh private: **không** public; GHCR cần token đọc trên VPS.
