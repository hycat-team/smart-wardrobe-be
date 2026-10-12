# Pipeline Docker / CI / CD (phạm vi VPS)

Tài liệu dành cho dev mới: vừa hiểu kiến trúc Docker/CI/CD, vừa vận hành được
môi trường production trên VPS. Đọc xuôi từ trên xuống.

**Phạm vi:** chỉ VPS (`/opt/closy`, `deployments/docker-compose.prod.yml`,
`deploy-vps-services.yml`). Không bao gồm AWS/box ngoài — xem `aws-runbook.md`
khi cần.

**Tài liệu liên quan:**

| File | Vai trò |
|---|---|
| `vps-runbook.md` | Runbook cắt monolith → 3 service, thao tác từng bước trên VPS |
| `shared-reference.md` | Bảng biến môi trường, DNS, monitoring, secrets |
| `deployments/scripts/rabbitmq-topology/README.md` | Chi tiết migrate queue RabbitMQ |

Placeholder dùng chung: `<VPS_HOST>`, `<VPS_USER>`, `<VPS_SSH_PORT>`,
`hycat-team/smart-wardrobe-be` (điền sẵn trong ví dụ lệnh).

## 0. Điều kiện tiên quyết

1. VPS đã có Docker + thư mục `/opt/closy` chứa `.env.production` (xem
   `vps-runbook.md` Bước 2). File này gitignored, đưa lên VPS bằng `scp`/editor,
   không qua GitHub.
2. GitHub environment `vps-production` đã tồn tại (nên bật Required reviewers).
3. GitHub secrets host-access đã đủ (xem §8). Thiếu secret thì workflow CD đỏ
   ngay ở bước SSH, chưa đụng tới container.
4. Hiểu 3 services độc lập (constitution §I): `account`, `commerce`, `ai`.
   Deploy một service không yêu cầu redeploy service khác.

## 1. Tổng quan pipeline

Luồng chuẩn trên VPS:

```text
PR/push main → ci.yml (check, test, build image không push)
  → dispatch workflow hoặc push tag vps-* → deploy-vps-services.yml
  → build + push GHCR → SSH vào VPS → compose pull + up -d từng service
  → recreate nginx 1 lần → smoke-all qua nginx
```

| Chặng | File/workflow | Kết quả |
|---|---|---|
| CI | `.github/workflows/ci.yml` | Pass mới được deploy; image build thử nhưng không push |
| Build + push | `.github/workflows/deploy-vps-services.yml` jobs `plan`, `build-push`, `build-tool` | Image `ghcr.io/<owner>/<svc>:<git-sha>` + alias tag release, image tool `rabbitmq-topology` pin digest |
| Deploy | job `deploy` (SSH) | Cập nhật `/opt/closy/.env.deploy`, `compose pull` + `up -d <svc>` |
| Ingress | job `reload-nginx` | `up -d --force-recreate nginx` một lần sau khi mọi service xong |
| Kiểm thử | job `smoke-all` | Curl `https://api.closy.hycat.online/api/v1/<svc>/health` đủ 3 service, grep đúng tên service |

## 2. Docker images

Mỗi service có Dockerfile riêng: `services/account/Dockerfile`,
`services/commerce/Dockerfile`, `services/ai/Dockerfile`. Nội dung 3 file
tương đương nhau, chỉ khác tên binary và thư mục migration/swagger.

| Lớp | Nội dung | Vì sao |
|---|---|---|
| Builder `golang:1.26` | Copy `go.work`, `go.mod/go.sum` mọi module → `go work sync` → `go mod download` → copy source → `CGO_ENABLED=0 go build -trimpath` | Tách layer dependency khỏi source: sửa code không tải lại module |
| Runtime `distroless/static-debian12:nonroot` | Copy binary, `database/migrations`, `api/swagger`, `configs/` | Image nhỏ, không shell, chạy user nonroot |
| `ENV PORT=8080` + `EXPOSE 8080` | Mọi container đều nghe 8080 | Compose và nginx upstream hardcode `http://<svc>:8080` |
| `HEALTHCHECK CMD ["/<svc>", "-healthcheck"]` | Probe do binary tự cung cấp | Compose `depends_on service_healthy` và nginx gate dựa vào đó |
| `ENTRYPOINT ["/<svc>"]` | Yêu cầu `PORT` hoặc `SERVER_PORT` | Compose set `SERVER_PORT=8080` (xem §3) |

Build context bắt buộc là repo root (để thấy `go.work`, `shared/`, `contracts/`):

```bash
docker build -f services/account/Dockerfile -t closy-account:local .
make docker-build svc=account
```

Image tool đối chiếu queue: `shared/tools/rabbitmq-topology/Dockerfile`,
build bằng `make image-build-tool`, push bằng CD và pin theo digest
(xem §9).

## 3. Compose production

File duy nhất cho VPS: `deployments/docker-compose.prod.yml`
(`name: closy-prod`). Mọi lệnh compose trên VPS luôn kèm cả 2 env-file:

```bash
cd /opt/closy
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml ps
```

### 3.1 Bảng services

| Service | Image | Ghi chú vận hành |
|---|---|---|
| `postgres` | `pgvector/pgvector:pg17` | Volume `postgres_data`, port publish `127.0.0.1:5432:5432`, healthcheck `pg_isready` |
| `redis` | `redis:7-alpine` | Volume `redis_data`, requirepass từ `REDIS_PASSWORD`, healthcheck `redis-cli ping` |
| `rabbitmq` | `rabbitmq:3-management-alpine` | Volume `rabbitmq_data`, mount `deployments/rabbitmq/rabbitmq.conf:ro`, healthcheck `rabbitmq-diagnostics ping` |
| `account` | `${ACCOUNT_IMAGE}` từ `.env.deploy` | `DATABASE_MIGRATION_TABLE=goose_db_version`, `SERVER_PORT=8080`, `APP_MODE=all` |
| `commerce` | `${COMMERCE_IMAGE}` | `DATABASE_MIGRATION_TABLE=goose_db_version_commerce` |
| `ai` | `${AI_IMAGE}` | `DATABASE_MIGRATION_TABLE=goose_db_version_ai` |
| `backend` (profile `rollback`) | `${MONOLITH_IMAGE}` | Chỉ dùng khi rollback thủ công về monolith frozen; ngày thường không chạy |
| `nginx` | `nginx:alpine` | Mount `./nginx/templates/${NGINX_MODE}` + `/etc/letsencrypt:ro` + `certbot_www:ro`, publish `80/443`, `depends_on` cả 3 services healthy |
| `certbot` (profile `maintenance`) | `certbot/certbot:v5.3.1` | Chỉ chạy khi cấp/renew cert, không chạy thường trực |

### 3.2 Networks, volumes, giới hạn

- Networks: `edge` (nginx ↔ ngoài), `app` (nginx ↔ services),
  `data` (`internal: true`, services ↔ postgres/redis/rabbitmq),
  `admin` (postgres maintenance).
- Volumes: `postgres_data`, `redis_data`, `rabbitmq_data`, `certbot_www`.
  **Không bao giờ `down -v` trên production** — xóa volume là mất dữ liệu.
  Backup DB trước mọi lần recreate `postgres`.
- `mem_limit`: postgres 1280m, redis 384m, rabbitmq 512m, mỗi service 512m,
  nginx 128m. `shm_size: 256m` cho postgres.
- `restart: always`, `stop_grace_period: 30s` cho services,
  `no-new-privileges:true`, log `json-file` `max-size 10m / max-file 5`.
- Migration Goose chạy lúc service start (`DATABASE_MIGRATE_ON_START=true`
  trong `configs/config.production.yaml`), mỗi service bảng version riêng
  nên không lệch thứ tự trên DB dùng chung.

Makefile bọc sẵn:

```bash
make docker-prod-up    # up -d toàn stack (đọc .env.production)
make docker-prod-stop  # stop không xóa volumes
make docker-prod-down  # stop + xóa volumes — cấm dùng trên prod
```

## 4. Registry GHCR

| Quy ước | Giá trị |
|---|---|
| Registry | `ghcr.io/<owner>/<svc>` (owner viết thường, ví dụ `ghcr.io/hycat-team/account`) |
| Tag immutable | `<svc>:<git-sha>` (ví dụ `:a1b2c3d`), kèm alias tên release tag khi push tag `vps-*` (ví dụ `:vps-v1.0.0`) |
| Tag `:latest` | Chỉ dùng trong CI cache, không dùng cho production |
| Pin production | Digest `name@sha256:<hex>` ghi trong `/opt/closy/.env.deploy` |
| Visibility | Private, VPS login bằng `GHCR_USERNAME` + `GHCR_READ_TOKEN` (scope `read:packages`) |

CI (`ci.yml` job `image-build`) build thử 3 images nhưng `push: false`.
CD (`deploy-vps-services.yml` job `build-push`) mới `push: true` và resolve
digest bằng `docker buildx imagetools inspect`.

## 5. CI (`ci.yml`)

Trigger: `pull_request` và `push` nhánh `main`, bỏ qua thay đổi chỉ docs
(`docs/**`, `specs/**`, `.specify/**`, `**/*.md`).

| Job | Lệnh tương ứng | Ý nghĩa với dev mới |
|---|---|---|
| `static` | `make fmt-check && make tidy-check && make generate-check` | Format, `go.mod` gọn, Wire/Swagger/nginx-route-map đồng bộ với code |
| `vet` | `make vet` | Static analysis mọi module workspace |
| `test` | `make test TEST_FLAGS=-short` | Unit + integration ngắn |
| `build` (matrix 3 svc) | `make build svc=<account\|commerce\|ai>` | Mọi service phải biên dịch được |
| `buf` | `make buf-lint`, `make buf-gen` + `git diff --exit-code -- contracts/` | Proto không drift khỏi stub đã commit |
| `swagger-parity` | `make swagger-all`, `make nginx-check`, `make nginx-routes` + diff, script check 8 modules HTTP + ≥50 paths | Swagger và route nginx phủ đủ API; `notification` worker-only được miễn |
| `image-build` (matrix 3 svc) | `docker/build-push-action` `push: false`, `platforms: linux/amd64` | Dockerfile phải build được amd64 (arch của VPS) |
| `test-race` | `make test-race TEST_FLAGS=-short` | Phát hiện data race |

Cache Go tách 2 lớp: `~/go/pkg/mod` theo hash `go.sum` (1 writer duy nhất
là job `vet` trên main), `~/.cache/go-build` theo job (`static/vet/test/...`).

## 6. CD (`deploy-vps-services.yml`)

Trigger:

- `workflow_dispatch` với input `services` (`all` hoặc `account,commerce`...).
- Push tag `vps-*` (luôn deploy all services).

Yêu cầu environment `vps-production` (approval gate).
`concurrency: closy-vps-services`, `cancel-in-progress: false`.

### 6.1 Job `plan`

Resolve input thành matrix JSON. `all` hoặc rỗng →
`["account","commerce","ai"]`; lẻ → parse comma-separated.

Khởi chạy tay 1 service:

```bash
gh workflow run deploy-vps-services.yml -R "hycat-team/smart-wardrobe-be" \
  --ref main -f services=commerce
```

Khởi chạy full:

```bash
git tag -a vps-v1.0.0 -m "..." && git push origin vps-v1.0.0
```

### 6.2 Job `build-push` (matrix theo plan)

1. Checkout, setup Buildx, login GHCR bằng `GITHUB_TOKEN`.
2. Resolve tên image viết thường `ghcr.io/<owner-lower>/<svc>`.
3. `docker/build-push-action` với `context: .`,
   `file: services/<svc>/Dockerfile`, `platforms: linux/amd64`,
   tags `<name>:<git-sha>` + `<name>:latest`, cache registry `buildcache`.
4. Nếu là tag push (`refs/tags/`), alias digest đã push sang tên tag
   bằng `docker buildx imagetools create` (không rebuild).

### 6.3 Job `build-tool`

Build image `rabbitmq-topology` từ
`shared/tools/rabbitmq-topology/Dockerfile`, push 2 tags như trên, sau đó
resolve digest và xuất `outputs.ref = <name>@<digest>`. Bắt buộc trỏ digest
chứ không trỏ tên, để bước check trên VPS chạy đúng binary đã test.

### 6.4 Job `deploy` (SSH vào VPS)

1. Checkout deployment files, setup Buildx, login GHCR.
2. Chuẩn bị SSH: ghi `VPS_SSH_KEY` → `~/.ssh/id_ed25519` (600),
   `VPS_KNOWN_HOSTS` → `~/.ssh/known_hosts` (600).
3. Đóng gói và upload:
   `tar -czf /tmp/closy-svc-deploy.tar.gz deployments/docker-compose.prod.yml deployments/nginx deployments/rabbitmq database/init`
   rồi `scp -P <port>` lên `/tmp/closy-svc-deploy.tar.gz` trên VPS.
4. Login GHCR trên VPS bằng `GHCR_READ_TOKEN` pipe qua
   `docker login ghcr.io`.
5. Check RabbitMQ topology (read-only, xem §9): pull image tool theo digest,
   chạy `docker run --rm --network closy-prod_data --env-file .env.production
   <tool-ref> --mode check` trong `/opt/closy`. Đỏ ở đây thì dừng, chưa đụng
   container nào.
6. Unpack: `mkdir -p /opt/closy`, `tar -xzf` vào đó, xóa tarball.
7. Loop deploy từng service (tuần tự, gom lỗi cuối cùng):
   resolve digest của `<name>:<git-sha>` → ghi `UPPER_IMAGE=<name>@<digest>`
   vào `.env.deploy` (read-modify-write, `chmod 600`) → backfill 2 service
   còn lại bằng `:latest` placeholder nếu là lần deploy đầu → `compose config`
   validate → `compose pull <svc>` → `compose up -d <svc>` → `compose ps <svc>`.

### 6.5 Job `reload-nginx`

Sau khi mọi service xong, recreate nginx một lần để nhận route map mới upload:

```bash
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml up -d --force-recreate nginx
```

`depends_on healthy` cả 3 services nên service nào unhealthy là job này đỏ.

### 6.6 Job `smoke-all` (matrix đủ 3 svc, không theo input)

Curl tối đa 12 lần, mỗi lần cách 5s:

```bash
curl -4 --fail --silent \
  "https://api.closy.hycat.online/api/v1/<svc>/health"
```

Pass khi body chứa `"service":"<svc>"` (chứng minh routing đúng, không chỉ
HTTP 200). Deploy lẻ 1 service vẫn smoke cả 3.

Theo dõi run:

```bash
RUN_ID=$(gh run list --workflow deploy-vps-services.yml \
  -R "hycat-team/smart-wardrobe-be" --limit 1 --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" -R "hycat-team/smart-wardrobe-be"
```

Nếu environment bật Required reviewers: vào `Actions → run → Review
deployments → Approve`.

## 7. Nginx + TLS

Template nguồn: `deployments/nginx/templates/services/`:

| File | Vai trò |
|---|---|
| `vps-3service.conf.template` | Server block `api.closy` → 3 services; vùng `# BEGIN/END GENERATED ROUTES` do `make nginx-routes` sinh từ Swagger |
| `default.conf.template` | Health ` /nginx-health`, redirect HTTP→HTTPS, default server |

Compose mount `./nginx/templates/${NGINX_MODE}:/etc/nginx/templates:ro`,
entrypoint nginx chạy `envsubst` với 3 biến
`API_DOMAIN`, `API_V2_DOMAIN`, `NGINX_CLIENT_MAX_BODY_SIZE`.

Single source route map: sửa API → `make swagger-all` →
`make nginx-routes` → `make nginx-check`. CI verify cả 2 chiều
(`git diff --exit-code -- deployments/nginx`).

Cert Let's Encrypt (`/etc/letsencrypt:ro` + volume `certbot_www`):

- Lần đầu (nginx đang chạy monolith): stop nginx qua compose rồi cấp
  standalone, vì `docker stop nginx` không trúng tên container thật
  (`closy-prod-nginx-1`):

```bash
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml stop nginx
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml --profile maintenance run --rm certbot \
  certonly --standalone -d api.closy.hycat.online --expand --agree-tos -m <email>
```

- Renew sau này qua webroot (`certbot_www` đã mount sẵn).
- `NGINX_MODE`: `services` (ổn định) · `services-redirect` (grace sau khi tắt
  box ngoài) · `rollback` (thủ công về monolith, xem §10). Đổi mode trong
  `.env.production` rồi recreate nginx (luôn kèm cả 2 env-file).

## 8. Env và secrets

### 8.1 Ba nguồn cấu hình trên VPS

| Nguồn | Vị trí | Ai ghi | Nội dung |
|---|---|---|---|
| `.env.production` | `/opt/closy/.env.production` (`chmod 600`) | Dev, 1 lần + sửa tay | Toàn bộ app config + infra local + domains (mẫu ở `.env.production.example`) |
| `.env.deploy` | `/opt/closy/.env.deploy` (`chmod 600`) | Workflow CD (§6.4) | 3 dòng pin image `*_IMAGE=<name>@<digest>` |
| GitHub secrets | Repo settings | Dev, 1 lần | Chỉ host-access + GHCR, không chứa app config |

Ví dụ thật đã che secret (cấu trúc thật, giá trị nhạy cảm thay bằng
`<redacted>`):

```text
# /opt/closy/.env.deploy — do workflow ghi
ACCOUNT_IMAGE=ghcr.io/<owner>/account@sha256:<64-hex-redacted>
COMMERCE_IMAGE=ghcr.io/<owner>/commerce@sha256:<64-hex-redacted>
AI_IMAGE=ghcr.io/<owner>/ai@sha256:<64-hex-redacted>
```

```text
# /opt/closy/.env.production — trích key thật, value nhạy cảm đã che
API_DOMAIN=api.closy.hycat.online
API_V2_DOMAIN=api-v2.closy.hycat.online
NGINX_MODE=services
NGINX_CLIENT_MAX_BODY_SIZE=20m
DB_HOST=postgres
DB_USER=<redacted>
DB_PASSWORD=<redacted>
DB_NAME=<redacted>
REDIS_HOST=redis
REDIS_PASSWORD=<redacted>
RABBITMQ_HOST=rabbitmq
RABBITMQ_USER=<redacted>
RABBITMQ_PASSWORD=<redacted>
JWT_SECRET=<redacted>
FRONT_END_ORIGIN=<redacted>
GOOGLE_WEB_CLIENT_ID=<redacted>
GOOGLE_WEB_CLIENT_SECRET=<redacted>
GOOGLE_REDIRECT_URI=https://api.closy.hycat.online/api/v1/auth/google/callback
```

Không set `ACCOUNT/COMMERCE/AI_IMAGE` trong `.env.production` — chúng sống
trong `.env.deploy`. Không commit file env thật (gitignored).

### 8.2 GitHub secrets (chỉ 7 secrets host-access)

| Secret | Dùng để |
|---|---|
| `VPS_HOST`, `VPS_USER`, `VPS_SSH_PORT` | `ssh/scp -P` vào VPS |
| `VPS_SSH_KEY` | Private key `id_ed25519` (set bằng `Get-Content ~/.ssh/id_ed25519 \| gh secret set`) |
| `VPS_KNOWN_HOSTS` | Output `ssh-keyscan -p <port> <host>` (dùng bản Git for Windows trên Windows) |
| `GHCR_USERNAME`, `GHCR_READ_TOKEN` | `docker login ghcr.io` trên VPS (PAT scope `read:packages`) |

Verify:

```bash
gh secret list -R "hycat-team/smart-wardrobe-be"
gh api repos/hycat-team/smart-wardrobe-be/environments --jq '.environments[].name'
```

Toàn bộ `DB_*`, `REDIS_*`, `RABBITMQ_*`, `JWT_*`, `EMAIL_*`, `PAYOS_*`,
`CLOUDINARY_*`, AI keys nằm trong `.env.production`, không phải GitHub secret.

### 8.3 Luật isolation VPS

VPS chỉ dùng infra local (`postgres`/`redis`/`rabbitmq` trong compose).
Cấm trỏ ra managed hostname. Guard chạy trên VPS (chỉ in tên biến khi FAIL):

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

## 9. RabbitMQ topology guard

RabbitMQ không đổi được `x-queue-type` trên queue đã tồn tại. Nếu registry
(`topology.go`) khai báo type khác broker đang giữ, service crash-loop với
`406 PRECONDITION_FAILED`.

CD chặn từ sớm: job `deploy` chạy tool ở `--mode check` trước khi đụng
container nào (xem §6.4). Đỏ ở đây thì xử lý thủ công trên VPS:

```bash
cd /opt/closy
export TOOL_IMAGE=ghcr.io/<owner>/rabbitmq-topology@sha256:<digest-redacted>
bash deployments/scripts/rabbitmq-topology/01-check-drift.sh vps
bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh
bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh --yes
```

Migrate dừng `account commerce ai` và không start lại — start tay khi sẵn
sàng. Chi tiết xem `deployments/scripts/rabbitmq-topology/README.md`.
Local check tương đương: `make rabbitmq-topology-check env=.env.production`.

## 10. Vận hành hàng ngày

```bash
cd /opt/closy
# Xem trạng thái + log
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml ps
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml logs -f --tail=100 account
# Validate interpolation
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml config >/dev/null
# Smoke từng service
for svc in account commerce ai; do
  curl -sS "https://api.closy.hycat.online/api/v1/$svc/health" | grep -o '"service":"[a-z]*"'
done
```

| Việc | Cách làm |
|---|---|
| Deploy 1 service | Dispatch với `-f services=<svc>` (§6.1); smoke-all vẫn check cả 3 |
| Rollback 1 service | Pin lại digest cũ trong `.env.deploy` rồi `up -d <svc>` |
| Rollback monolith (khẩn cấp, thủ công) | `make image-build-monolith-rollback` → push GHCR → đặt `NGINX_MODE=rollback` → `compose --profile rollback up -d backend` (tag `monolith-legacy` cấm di chuyển) |
| Đổi `NGINX_MODE` | Sửa `.env.production` → recreate nginx (luôn kèm cả 2 env-file) |
| Dừng/mở stack | `make docker-prod-stop` / `make docker-prod-up`; cấm `make docker-prod-down` trên prod |

## 11. Troubleshooting

| Dấu hiệu | Nguyên nhân thường gặp | Xử lý |
|---|---|---|
| CD đỏ ở `Check RabbitMQ queue topology` | Queue type drift | Làm theo §9, không skip |
| CD đỏ ở `Deploy pinned service images` | Digest chưa có (build-push fail) hoặc `.env.deploy` thiếu biến | Xem log `imagetools inspect`, chạy lại dispatch service đó |
| CD đỏ ở `reload-nginx` | Một service unhealthy (`depends_on` gate) | `compose ps` + `logs` service đó, fix rồi redeploy |
| Smoke fail 1 service | Routing nginx sai hoặc service chưa healthy | `curl /api/v1/<svc>/health` trực tiếp, check `deployments/nginx/templates/services/vps-3service.conf.template` đã upload bản mới chưa |
| `docker login ghcr.io` fail trên VPS | `GHCR_READ_TOKEN` hết hạn hoặc sai `GHCR_USERNAME` | Rotate PAT `read:packages`, update secret, login tay kiểm tra |
| SSH/scp fail | Sai port/key/known_hosts | `ssh -p <port> <user>@<host>` tay, `ssh-keyscan` lại nếu VPS rebuild |
| `config` báo missing `*_IMAGE` | Lần deploy đầu thiếu `.env.deploy` | Deploy all 1 lần để backfill, không set tay `*_IMAGE` vào `.env.production` |
| Disk đầy | Log json-file hoặc image cũ | `docker system df`, `docker image prune -a` (không đụng volumes), check `max-size` logging |
| Healthcheck fail sau deploy | Thiếu `GOOGLE_WEB_CLIENT_ID/SECRET` khi có `GOOGLE_REDIRECT_URI`, hoặc sai `FRONT_END_ORIGIN` | Service không boot do validation startup — điền env theo `shared-reference.md` §2 rồi `up -d` lại |

## 12. Giới hạn đã biết

- Migration Goose mở connection direct riêng (`DB_LISTENER_HOST`, port 5432);
  runtime query qua pooler/direct tùy môi trường.
- `:latest` chỉ tồn tại cho CI cache; production luôn pin sha/digest.
- Nginx không rate-limit (chỉ body-limit/timeout/security-header);
  rate-limit nằm ở app (Redis-backed). Không WAF đầy đủ — ngoài phạm vi.
- Search mặc định tắt (`ELASTICSEARCH_ENABLED=false`), fallback DB.
- Monitoring: log JSON qua compose, health endpoint per-service, trace
  `request_id` (Nginx `main_tid` ↔ app log). Chi tiết ngưỡng xem
  `shared-reference.md` §4.
