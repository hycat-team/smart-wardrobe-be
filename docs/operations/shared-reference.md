# Tài liệu tham chiếu chung (dùng cho cả 2 runbook)

File này KHÔNG phải runbook thực thi — không có thứ tự bước. Hai runbook
(`vps-runbook.md`, `aws-runbook.md`) trỏ vào đây khi cần tra cứu. Đọc 1 lần cho biết,
không cần đọc xuôi.

Placeholder dùng chung toàn bộ tài liệu vận hành (thay trước khi chạy lệnh):
`hycat-team/smart-wardrobe-be` (điền sẵn) · `<VPS_HOST>` · `<VPS_USER>` ·
`<VPS_SSH_PORT>` · `<BOX_IP>` (IP box AWS).

## 1. Ma trận 2 trục + luật isolation

VPS và box ngoài là **2 trục độc lập** — bật/tắt một bên không đụng bên còn lại.
DNS là điểm phối hợp duy nhất.

| Hạng mục | VPS (`services/`) | Box ngoài AWS Lightsail (GĐ2, EC2-backed) |
|---|---|---|
| Compute | Docker Compose 3 service (`APP_MODE=all`) | 6 container (`http`×3 + `worker`×3) |
| Postgres | container `postgres` local (`pgvector/pg17`) | Supabase pooler 6543 / direct 5432 |
| Redis | container `redis` local (no TLS) | Redis Cloud (TLS chỉ ở gói trả phí; free 30MB không TLS) |
| RabbitMQ | container `rabbitmq` local | CloudAMQP (amqps 5671) |
| Elasticsearch | container ngoài compose, tắt tay được (app fallback DB) | không chạy (`ENABLED=false`) |
| Email/Media/AI | Brevo / Cloudinary / keys hiện tại | copy y nguyên, share quota |
| Nginx upstream | tên service compose | tên service compose (`*-http` trên box) |
| `SERVER_ENV` | `vps` | `vps` (không còn profile `managed`) |

**Luật isolation (cứng):** VPS chỉ dùng infra local trong docker
(`postgres`/`redis`/`rabbitmq`); box ngoài chỉ dùng managed. Cấm VPS trỏ
DB/Redis/RabbitMQ ra hostname managed ở runtime — hai môi trường không bao giờ
dùng chung infra (ngoại lệ duy nhất: đồng bộ dump/restore offline).

**Trục VPS (`NGINX_MODE` trên VPS, chỉ quyết chuyện VPS):**

| Mode | `api.closy.hycat.online` | Monolith | Dùng khi |
|---|---|---|---|
| `services` | 3 service compose local | **dừng** | ổn định (bất kể box ngoài) |
| `services-redirect` | 3 service + `api-v2`→301 về api.closy | dừng | grace 14–30 ngày sau khi tắt box, xong quay về `services` + xóa DNS |
| `rollback` | monolith (`backend`) | thủ công | ADR 012 (workflow đã lưu trữ) |

**Trục box ngoài:** tắt (chưa có DNS) · bật (`api-v2` → box AWS) · cắt (về VPS
`services-redirect` → xóa DNS). Song song 2 trục thì mỗi bên ghi DB riêng
(divergence có chủ đích) — reunify duy nhất bằng dump/restore tay trong cửa sổ
bảo trì; song song càng lâu, divergence càng lớn.

## 2. Biến môi trường

Nguồn giá trị mẫu: `.env.example` (local), `.env.production.example` (VPS),
`.env.aws.example` (box ngoài). Default không-secret nằm ở `configs/config.yaml` /
`configs/config.production.yaml`; `.env` chỉ ghi đè thứ khác theo môi trường.
Nhiều biến để trống là **có chủ đích**. Box ngoài KHÔNG dùng file env local —
biến tiêm từ GitHub secrets trong `deploy-aws.yml`.

| Profile | File env | `CONFIG_FILE` | `SERVER_ENV` | Ghi chú |
|---|---|---|---|---|
| Local dev | `.env` | `configs/config.yaml` | `local` | `make docker-dev-up` / `make run` |
| VPS (services) | `.env.production` | `configs/config.production.yaml` | `vps` | 3 service compose + Nginx |
| Box ngoài | (không) | `configs/config.production.yaml` | `vps` | env qua GitHub secrets |

Thứ tự ưu tiên: `PORT` (cloud) > `SERVER_PORT` > `server.port` (YAML).

> `SERVER_ENV` là **profile triển khai**, không phải cờ "production". Mọi giá trị
> ngoài `local`/`development`/`dev` (kể cả `vps`) được coi là production → bật gin
> release mode, enforce `SWAGGER_ACCESS_CODE`, chặn `FRONT_END_ORIGIN` wildcard
> (`config.IsProductionEnv`, fail-closed cho env trống/lạ).

### 2.1 Cấu hình lõi

| Biến | Ghi chú |
|---|---|
| `APP_MODE` | `all` (VPS) / `http`+`worker` per container (box ngoài) |
| `FRONT_END_ORIGIN` | CORS; danh sách origin, phẩy phân tách |
| `COOKIE_DOMAIN` | Domain attribute cho cookie phiên; rỗng = host-only (local); production đặt `closy.hycat.online` để FE (`closy.hycat.online`) và API (`api.closy.hycat.online`) dùng chung cookie |
| `SWAGGER_ACCESS_CODE` | Enforce khi `SERVER_ENV` không phải local/dev (tức `vps`/`production`/managed) — cùng điều kiện bật gin release mode + chặn CORS wildcard |
| `TZ` | `Asia/Ho_Chi_Minh` |
| `SERVER_PORT` | `8080` (compose hardcode; entrypoint bắt buộc có `PORT` hoặc `SERVER_PORT`) |

### 2.2 Database / Redis / RabbitMQ / Search

| Biến | Ghi chú |
|---|---|
| `DB_HOST`, `DB_PORT` | VPS: `postgres`/`5432`; box: pooler host + `6543` |
| `DB_USER`, `DB_PASSWORD`, `DB_NAME` | secret |
| `DB_SSLMODE` | VPS `disable`; box (Supabase) `require` |
| `DB_LISTENER_HOST` / `DB_LISTENER_PORT` / `DB_LISTENER_SSLMODE` | host **direct** cho migrate/`LISTEN`/advisory lock (5432; box `require`) |
| `DATABASE_MIGRATE_ON_START` | `true` (theo `config*.yaml`) |
| `DATABASE_MIGRATION_TABLE` | rỗng = `goose_db_version`; per-service: `_commerce`/`_ai` (compose/workflow set) |
| `DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER` | `true` (trong configs) |
| `DB_DRILL_SCHEMA` | rỗng; chỉ dùng khi drill |
| `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` | secret (box: endpoint/port riêng, KHÔNG phải 6379) |
| `REDIS_TLS_ENABLED` | VPS `false`; box: `false` cho free 30MB, `true` cho gói có TLS |
| `RABBITMQ_HOST`, `RABBITMQ_PORT`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD`, `RABBITMQ_VHOST` | secret (box: `5671` amqps; `VHOST` = vhost riêng trên CloudAMQP, VPS/local để trống = `/`) |
| `RABBITMQ_TLS_ENABLED`, `RABBITMQ_MANAGEMENT_TLS_ENABLED` | VPS `false`; box `true` |
| `OUTBOX_LISTENER_ENABLED` | `true`; tắt → fallback poll |
| `ELASTICSEARCH_ENABLED` | `false` mọi nơi; fallback DB (chi tiết §5) |

Migration/`LISTEN`/advisory lock là **session-scoped**: chạy trên **direct 5432**,
KHÔNG qua transaction pooler 6543. Kiểm tra thủ công:
```bash
goose -dir services/<svc>/database/migrations \
  postgres "host=$DB_LISTENER_HOST user=$DB_USER password=$DB_PASSWORD \
    dbname=$DB_NAME port=${DB_LISTENER_PORT:-5432} sslmode=${DB_LISTENER_SSLMODE:-disable}" status
```

### 2.3 Auth, ingress, email, thanh toán, media, AI

| Biến | Ghi chú |
|---|---|
| `JWT_SECRET` / `JWT_ALGORITHM` | secret; `HS256` |
| `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY` | PEM; rỗng = HS256 tạm thời |
| `GOOGLE_WEB_CLIENT_ID`, `GOOGLE_WEB_CLIENT_SECRET`, `GOOGLE_CLIENT_IDS`, `GOOGLE_REDIRECT_URI`, `GOOGLE_STATE_TTL_MINUTES` | Google OAuth. `WEB_CLIENT_ID`/`WEB_CLIENT_SECRET` là client **Web application**, dùng cho *duy nhất* luồng redirect web — dùng nhầm client Android/iOS sẽ khiến Google trả `redirect_uri_mismatch`. `CLIENT_IDS` chỉ là danh sách phẩy phân tách client iOS/Android được chấp nhận làm audience của ID token; client web đã được thêm tự động nên không khai báo lại, và thứ tự trong danh sách không mang ý nghĩa. `REDIRECT_URI` là callback backend; box ngoài copy y nguyên từ VPS |
| `INGRESS_ACCOUNT/COMMERCE/AI_URL` | VPS: `http://account:8080`...; box: hardcode `*-http` trong compose |
| `INGRESS_TOKEN_AUDIENCE` | để trống (không còn Cloud Run) |
| `EMAIL_SENDER_EMAIL`, `EMAIL_SMTP_USER`, `EMAIL_SMTP_KEY`, `EMAIL_REPLY_TO` | Brevo SMTP; box copy y nguyên từ VPS |
| `PAYOS_CLIENT_ID`, `PAYOS_API_KEY`, `PAYOS_CHECKSUM_KEY` | **kênh riêng mỗi trục** (VPS và box KHÔNG share — mỗi kênh có webhook URL riêng) |
| `PAYOS_ENDPOINT` | `https://api-merchant.payos.vn` |
| `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET`, `CLOUDINARY_API_URL` | box copy y nguyên từ VPS |
| `VISION_*`, `EMBEDDING_*`, `CHAT_TEXT_*`, `RECOMMENDATION_TEXT_*`, `FREE_TEXT_*` | provider/model/endpoint/key; box copy y nguyên từ VPS |

### 2.4 Nginx / compose (không thuộc app)

| Biến | Ghi chú |
|---|---|
| `COMPOSE_PROJECT_NAME`, `BACKEND_IMAGE`, `BACKEND_HEALTH_PATH` | compose VPS (`BACKEND_IMAGE` legacy, compose hiện tại không đọc) |
| `API_DOMAIN` | `api.closy.hycat.online` |
| `API_V2_DOMAIN` | `api-v2.closy.hycat.online` (box ngoài / redirect grace) |
| `NGINX_MODE` | `services` / `services-redirect` / `rollback` (thủ công, ADR 012) |
| `NGINX_CLIENT_MAX_BODY_SIZE` | `20m` |
| `NGROK_AUTHTOKEN`, `NGROK_DOMAIN`, `NGROK_TARGET` | chỉ local tunnel |

### 2.5 GitHub secrets (chỉ host-access; app config nằm trong file env)

Workflow CHỈ đọc secret truy cập host + GHCR:
`AWS_HOST`, `AWS_USER`, `AWS_SSH_PORT`, `AWS_SSH_KEY`, `AWS_KNOWN_HOSTS` (box AWS);
`VPS_HOST`, `VPS_USER`, `VPS_SSH_PORT`, `VPS_SSH_KEY`, `VPS_KNOWN_HOSTS` (VPS);
`GHCR_USERNAME`, `GHCR_READ_TOKEN` (PAT `read:packages`, dùng chung).

Toàn bộ `DB_*`, `REDIS_*`, `RABBITMQ_*`, `JWT_*`, `CLOUDINARY_*`, `EMAIL_*`,
`PAYOS_*`, `SWAGGER_ACCESS_CODE`, AI provider keys **KHÔNG phải** GitHub secret —
chúng nằm trong `.env.production` (VPS) và `.env.aws` (box), đưa lên host bằng
`scp`/editor, không qua GitHub. (Giai đoạn Cloud Run cũ từng set chúng làm secret;
nay không workflow nào đọc nữa.)

Không workflow nào tiêm env app cho box. `docker-compose.aws.yml` hardcode mỗi
container: `CONFIG_FILE`, `APP_MODE=http|worker`, `DATABASE_MIGRATION_TABLE`,
`INGRESS_*_URL`, `SERVER_PORT=8080`, `TZ`. Còn lại lấy từ `.env.aws`: `SERVER_ENV=vps`,
`API_V2_DOMAIN`, `DB_*`/`REDIS_*`/`RABBITMQ_*` (kèm cờ TLS theo gói), `JWT_*`,
`EMAIL_*`, `PAYOS_*`, `CLOUDINARY_*`, AI keys, `ELASTICSEARCH_ENABLED=false`,
`FRONT_END_ORIGIN`, `SWAGGER_ACCESS_CODE`, `GOOGLE_*`.

Verify đã đủ:
```bash
gh secret list   -R "hycat-team/smart-wardrobe-be"
gh variable list -R "hycat-team/smart-wardrobe-be"
gh api repos/hycat-team/smart-wardrobe-be/environments --jq '.environments[].name'
# kỳ vọng: vps-production, aws-production
```

### 2.6 Biến đã bỏ / không nên set

- `DB_LISTENER_PORT`, `DATABASE_QUERY_EXEC_MODE` (`auto`), `INGRESS_AUTH_HEADER`,
  `ELASTICSEARCH_ENABLED`, `OUTBOX_LISTENER_ENABLED`, `DATABASE_MIGRATE_ON_START`,
  `REDIS_TLS_ENABLED`, `RABBITMQ_TLS_ENABLED`, `JWT_ALGORITHM` — default trong
  `configs`; chỉ set khi cần khác.
- Đã xóa khỏi kiến trúc: `COMPOSE_PROFILES`, `*_RUN_URL`, `TOKEN_HELPER_*`,
  `VPS_SMOKE_URL`, `AWS_SMOKE_URL` — thấy ở đâu thì xóa.

## 3. DNS + CI/CD

`hycat.online` giữ DNS ở **Vercel**; **không** NS-delegate, **không** đổi nameserver,
**không** Cloudflare DNS.
- `api.closy.hycat.online` (A/CNAME) → VPS: giữ nguyên, không đổi.
- `api-v2.closy.hycat.online`: chỉ tạo khi bật box ngoài — Type `A` (hoặc `CNAME`),
  Name `api-v2`, Value IP box, TTL thấp (`60`) trong giai đoạn chuyển; chỉ thêm
  **sau khi** cert + server block `api-v2` sẵn sàng. Grace khi cắt: về VPS +
  redirect 301 rồi xóa sau 14–30 ngày.

Verify DNS:
```bash
dig +short api-v2.closy.hycat.online
dig +short api.closy.hycat.online       # không đổi
curl -sSI https://api-v2.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"
```

| Workflow | Trigger | Nội dung |
|---|---|---|
| `ci.yml` | PR/push `main` | fmt/tidy/generate-check, vet, test(-race), build matrix, buf lint + proto drift, parity swagger, image-build matrix |
| `deploy-vps-services.yml` | dispatch hoặc push tag `vps-*` + approval | 3 chặng: build matrix → GHCR (pin digest, alias tên tag) → `compose up -d <svc>` → recreate nginx 1 lần → `smoke-all` curl `/api/v1/<svc>/health` đủ 3 service + grep tên service |
| `deploy-aws.yml` | dispatch hoặc push tag `aws-*` + approval | 3 chặng tương tự (6 container http×3 + worker×3); `smoke-all` qua IP box, chưa cần DNS |
| `deploy-production.yml` | đã lưu trữ (ADR 012) | Rollback monolith khẩn cấp — chuyển vào `docs/archive/`; chỉ còn đường thủ công |

Quy ước image: tag immutable `<svc>:<git-sha>` (+ alias tên release tag);
`:latest` chỉ trong CI; image private KHÔNG public. Tag `monolith-legacy` đóng băng,
cấm di chuyển. Deploy lẻ 1 service chỉ bằng dispatch (`-f services=<svc>`);
push tag luôn = all services.

## 4. Monitoring

Không có Cloud Monitoring trong kiến trúc. Monitoring dựa trên: log JSON ra stdout
(`docker compose logs`), health endpoint per-service, billing alarm.

```bash
# VPS
cd /opt/closy
docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml logs -f --tail=100 account
# Box ngoài
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml logs -f --tail=100 commerce-worker
```
Trace `request_id` (Nginx `main_tid` ↔ app log) để nối request qua 2 lớp.

| Signal | Cách đo | Ngưỡng gợi ý |
|---|---|---|
| Health | `GET /api/v1/<svc>/health` từng service qua cron/Uptime Kuma (cả `api.closy` + `api-v2`) | alert khi down 2 lần liên tiếp |
| Latency | log `duration_ms` / access log Nginx | p95 > 500ms trong 5m |
| Error-rate | đếm 5xx trong log / access log | > 1% trong 5m |
| Worker liveness | `docker compose ps` + outbox backlog trong log | worker down / backlog > 10000 |
| Bill AWS | billing alarm \$20/40/60 + Lightsail transfer metrics | vượt ngưỡng (overage SG \$0.12/GB) |

Khi alert: health down → `ps` + log → redeploy service đó (`-f services=<svc>`);
error-rate cao → xem log → rollback về digest trước nếu do deploy mới; worker
down/backlog tăng → kiểm tra singleton + advisory lock rồi restart; p95 vượt →
kiểm tra pooler/AMQP/Redis conn; bill vượt → kiểm tra transfer, consider tắt box.

Verify: log JSON đọc được qua compose; health check ngoài hoạt động cả 2 hostname;
billing alarm AWS hoạt động.

## 5. Elasticsearch (tắt mặc định, cả 2 trục)

Search không được funded nên tắt (`ELASTICSEARCH_ENABLED=false`) mọi môi trường.
Không set `ELASTICSEARCH_*` trong secrets hay `.env.production`. Hành vi khi tắt:
tìm kiếm fallback **database** (200 bình thường); thêm/sửa/xoá thành công, sync job
no-op (không đầy DLQ); health phơi `elasticsearch: {enabled, healthy}`. Trên VPS có
thể tắt container ES bất cứ lúc nào (app đã fallback DB). Verify: search 200 từ DB;
`elasticsearch_sync_queue` và `.dlq` không tăng; muốn bật lại thì set
`ELASTICSEARCH_ENABLED=true` + addresses rồi redeploy.

## 6. Xoay vòng credential (tóm tắt)

Thứ tự: **rotate managed trước, VPS sau**; mỗi bước redeploy service tương ứng rồi
mới thu hồi (tránh downtime). JWT: sinh secret mới → set secret → dispatch cả 2
workflow (+ cập nhật `.env.production` nếu còn VPS). SSH box: xoá keypair trên
console + `gh secret remove AWS_SSH_KEY/AWS_KNOWN_HOSTS`. Managed (Supabase/AMQP/Redis):
rotate trong dashboard → cập nhật secrets → redeploy. Cloudinary/Brevo: rotate key →
cập nhật → redeploy. `SWAGGER_ACCESS_CODE`: rotate nếu từng lộ.
