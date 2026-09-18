# 14 — First-time deploy step-by-step (managed + VPS song song)

> Runbook **cầm tay chỉ việc** cho lần host đầu tiên: Cloud Run (3 service) + VPS
> Nginx ingress song song với monolith. Thứ tự bắt buộc từ trên xuống.
>
> **Điền các placeholder trước khi chạy**: `<PROJECT>` (GCP project id — phải khớp
> `AR_PROJECT` trong `deploy-platform.yml`, mặc định `smart-wardrobe`),
> `<REGION>` = `asia-southeast1`, `<OWNER/REPO>` (GitHub), `<VPS_HOST>`, `<VPS_USER>`.
>
> Biến môi trường đầy đủ: xem `07-bang-bien-moi-truong.md`.

## 0. Đăng ký nền tảng & điều kiện tiên quyết

Bạn chỉ cần **VPS + domain Vercel**; các nền tảng dưới đây đăng ký mới. Làm **theo
thứ tự** — mỗi mục ghi rõ lấy giá trị nào để điền vào secrets/`.env` (xem `07`).

### 0.1 GitHub (đã có repo)
- Tạo 2 **Environments** (Settings → Environments): `gcp-production`, `vps-production`.
- Tạo **GHCR read token** cho VPS: Personal access token (classic) scope `read:packages`
  → secret `GHCR_READ_TOKEN` + `GHCR_USERNAME`.

### 0.2 Google Cloud (Cloud Run + Artifact Registry)
1. Tạo Google account → https://console.cloud.google.com → **Start free trial** (cần thẻ;
   nhận credit trial).
2. **Tạo project**: id phải **khớp `AR_PROJECT`** trong `deploy-platform.yml`
   (mặc định `smart-wardrobe`; nếu đặt id khác thì sửa workflow).
3. **Bật billing** cho project (Billing → link account).
4. Cài `gcloud`, rồi: `gcloud auth login` + `gcloud config set project <PROJECT>`.
5. (Sau, ở §2) bật API, tạo Artifact Registry/WIF/SA, đặt **billing cap**.

### 0.3 Supabase Pro (Postgres + pgvector)
1. Đăng ký https://supabase.com → **New project** (plan Pro hoặc dùng trial).
2. Chọn region **gần `asia-southeast1`** (Singapore) để giảm latency.
3. Lưu lại: **database password**, **project ref**, host pooler (`...pooler.supabase.com`)
   và host direct (`db.<ref>.supabase.co`).

### 0.4 Redis Cloud Essentials 250MB
1. Đăng ký https://redis.io/cloud → **New subscription** → Essentials 250MB.
2. Lấy **endpoint (host), port riêng, password**; bật **TLS**.

### 0.5 CloudAMQP Tough Tiger
1. Đăng ký https://www.cloudamqp.com → tạo instance (Tough Tiger; có thể dùng gói nhỏ/free
   để thử trước).
2. Lấy **amqps host/port (thường 5671), user, password**; tạo **vhost riêng**.

### 0.6 Cloudflare Pro (funded, KHÔNG nằm trong đường API)
- **Không bắt buộc** cho việc host API; chỉ đăng ký khi có mục đích khác. DNS vẫn ở
  **Vercel**, không NS-delegate, không Cloudflare trong đường API (FR-025/029).

### 0.7 VPS + domain (đã có)
- VPS đang chạy monolith (`api.closy`); Docker + `/opt/closy`.
- `hycat.online` giữ DNS ở **Vercel**; sẽ thêm **subdomain `api-v2`** ở §7.

### 0.8 Điều kiện tiên quyết (checklist)
- [ ] GCP project trial đã bật billing (id khớp `AR_PROJECT`).
- [ ] GitHub repo có quyền Actions + Packages (GHCR).
- [ ] Supabase / Redis Cloud / CloudAMQP đã tạo, đã có host/port/password.
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

Ghi lại để làm GitHub secret:
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

1. Tạo **1 project Pro**. Trong SQL editor:
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
2. `search_path` **server-side** (không truyền qua DSN runtime pooler):
   ```sql
   alter role <DB_USER> set search_path =
     public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared;
   ```
3. Lấy 2 DSN (Project Settings → Database):
   - **Pooler (transaction, 6543)** → `DB_HOST`, `DB_PORT=6543`.
   - **Direct (5432)** → `DB_LISTENER_HOST` (host `db.<ref>.supabase.co`), `DB_LISTENER_PORT=5432`.
   Cả hai dùng `DB_USER/DB_PASSWORD/DB_NAME`.

> Migration chạy lúc boot. Lưu ý đã biết: hiện service migrate trên **connection chính**
> (runtime pooler); nếu boot lỗi prepared/session, tạm đặt `DB_HOST/DB_PORT` = direct
> cho lần migrate đầu rồi đổi lại pooler (xem §11 known limitations).

## 4. Redis Cloud + CloudAMQP

- **Redis Cloud Essentials 250MB**: lấy host/port/password (endpoint có **port riêng**,
  vd 12000+), TLS bật → secrets `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`.
- **CloudAMQP Tough Tiger**: lấy `amqps` host/port (thường **5671**)/user/password,
  vhost riêng → `RABBITMQ_HOST`, `RABBITMQ_PORT`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD`.

## 5. GitHub: Environments + Secrets

Tạo 2 Environments: `gcp-production` (deploy-platform), `vps-production` (VPS workflows).

**Secrets** (Settings → Secrets and variables → Actions):

| Nhóm | Secret |
|---|---|
| GCP | `GCP_WORKLOAD_IDENTITY_PROVIDER`, `GCP_DEPLOYER_SERVICE_ACCOUNT` |
| Database | `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_HOST`, `DB_PORT`, `DB_LISTENER_HOST`, `DB_LISTENER_SSLMODE` |
| Redis | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` |
| RabbitMQ | `RABBITMQ_HOST`, `RABBITMQ_PORT`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD` |
| JWT | `JWT_SECRET`, `JWT_ALGORITHM`, `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY` |
| Swagger | `SWAGGER_ACCESS_CODE` |
| Media/Email/PayOS | `CLOUDINARY_*`, `EMAIL_SMTP_USER`, `EMAIL_SMTP_KEY`, `PAYOS_*` |
| Ingress | `INGRESS_ACCOUNT_URL`, `INGRESS_COMMERCE_URL`, `INGRESS_AI_URL`, `INGRESS_TOKEN_AUDIENCE` |
| VPS | `VPS_HOST`, `VPS_USER`, `VPS_SSH_PORT`, `VPS_SSH_KEY`, `VPS_KNOWN_HOSTS`, `GHCR_USERNAME`, `GHCR_READ_TOKEN` |

**Variables**: `VPS_SMOKE_URL` (deploy-vps-services), `PRODUCTION_HEALTH_URL` (deploy-production).

> `INGRESS_*_URL` cho managed = URL `run.app` của từng service (điền **sau** bước 6).

## 6. Deploy Cloud Run

```bash
gh workflow run deploy-platform.yml -f services=all
gh run watch
```
Sau khi xong, lấy URL:
```bash
for svc in account commerce ai; do
  echo "$svc = $(gcloud run services describe smart-wardrobe-$svc --region asia-southeast1 --format='value(status.url)')"
done
```
→ điền secret `INGRESS_ACCOUNT_URL/COMMERCE/AI_URL` = các URL trên, rồi **chạy lại**
`gh workflow run deploy-platform.yml -f services=all` để app có ingress URL.
→ grant `run.invoker` cho token-helper SA (§2.3) và điền `INGRESS_TOKEN_AUDIENCE`
= URL Account (hoặc union audience theo spec).

Verify: mỗi service 200 `GET /api/v1/health` (gọi qua token helper/`gcloud run services proxy`).

## 7. VPS: ingress song song (`api-v2`)

1. Chuẩn bị `/opt/closy` (compose + nginx + `.env.production` + SA key):
   ```bash
   cd /opt/closy
   cp token-helper-sa.json /opt/closy/token-helper-sa.json && chmod 600 token-helper-sa.json
   # .env.production: API_DOMAIN, API_V2_DOMAIN, NGINX_MODE=parallel,
   # COMPOSE_PROFILES=parallel, *_RUN_URL, TOKEN_HELPER_URL, TOKEN_HELPER_SA_KEY_FILE
   ```
2. **Cert Let's Encrypt** cho cả 2 hostname. Do Nginx đang chạy (monolith), cấp cert
   bằng standalone (giải phóng port 80) rồi mở lại:
   ```bash
   docker stop nginx || true
   docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
     --profile maintenance run --rm certbot certonly --standalone \
     -d api.closy.hycat.online -d api-v2.closy.hycat.online --expand --agree-tos -m <email>
   ```
   (renewal sau này qua webroot; certbot_www đã mount sẵn.)
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

Theo `09-chuyen-du-lieu-vps-supabase.md` (backup + checksum + re-dump tại cutover).

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
- Repatriation: `repatriate-to-vps.md` + `ha-tang/11-ve-lai-vps.md`.
- Cutover `api.closy`: `manual-cutover-api-closy-to-cloudrun.md` (owner-triggered).
- Rollback khẩn cấp: `deploy-production.yml` (đổi `NGINX_MODE=rollback` **trước** khi chạy,
  vì workflow chỉ start `backend`).

## 11. Known limitations (đọc trước khi run)

- **Migration đã tách sang direct connection** (Option A): Goose + session advisory
  lock mở connection riêng từ `DB_LISTENER_HOST`/`DB_LISTENER_PORT`/`DB_LISTENER_SSLMODE`
  (5432), runtime query vẫn qua pooler 6543. Vì vậy `DB_LISTENER_HOST` là **bắt buộc**
  ở managed (workflow enforce).
- **Token-helper SA key** là bí mật file trên VPS; rotation thủ công (tạo key mới,
  redeploy helper, revoke key cũ) — xem `12-ngung-vps-va-thu-hoi-secret.md`.
- **`:latest` chỉ ở CI**; production pin theo `<git-sha>`/digest.
- Ảnh private: **không** public; GHCR cần token đọc trên VPS.
