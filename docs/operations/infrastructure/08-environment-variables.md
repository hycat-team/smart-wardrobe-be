# 08 — Bảng biến môi trường & cách điền (020-platform-migration)

> Nguồn giá trị mẫu: `.env.example` (local), `.env.production.example` (VPS).
> **Managed (Cloud Run)** KHÔNG dùng `.env` — biến được tiêm từ **GitHub Actions
> secrets** trong `.github/workflows/deploy-platform.yml`.
> Default không-secret nằm ở `configs/config.yaml` / `configs/config.production.yaml`;
> `.env` chỉ ghi đè thứ khác theo môi trường. Nhiều biến để trống là **có chủ đích**.

## 1. Ba profile

| Profile | File env | `CONFIG_FILE` | `SERVER_ENV` | Ghi chú |
|---|---|---|---|---|
| Local dev | `.env` | `configs/config.yaml` | `local` | `make docker-dev-up` / `make run` |
| VPS (repatriation/self-host/song song) | `.env.production` | `configs/config.production.yaml` | `vps` | 3 service compose + Nginx |
| Managed (Cloud Run) | (không) | `configs/config.production.yaml` | `managed` | env qua GitHub secrets |

Thứ tự ưu tiên: `PORT` (Cloud Run) > `SERVER_PORT` > `server.port` (YAML).

## 2. Cách điền theo từng profile

### 2.1 Local
```bash
cp .env.example .env
# điền host/port/password local; host service-name (postgres/redis/rabbitmq) do compose tự set
make docker-dev-up
```

### 2.2 VPS
```bash
cd /opt/closy
cp .env.production.example .env.production
# điền các biến bắt buộc — xem §6
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml config >/dev/null  # verify syntax
```

### 2.3 Managed
Không tạo file; set secret theo `00` §5, rồi dispatch `deploy-platform.yml` (`00` §6).

## 3. Nhóm cấu hình lõi

| Biến | Default (configs) | Ghi chú |
|---|---|---|
| `APP_MODE` | `all` | `all`/`http`/`worker` (FR-044); Cloud Run set `http`/`worker` |
| `SERVER_ENV` | `development`/`production` | chọn profile; `managed` bật validation managed |
| `PORT` / `SERVER_PORT` | `8080` | Cloud Run inject `PORT` |
| `FRONT_END_ORIGIN` | — | CORS; danh sách origin, phẩy phân tách |
| `SWAGGER_ACCESS_CODE` | — | Chỉ enforce khi `SERVER_ENV=production`; local có thể để trống |
| `TZ` | `Asia/Ho_Chi_Minh` | Container timezone |

## 4. Database / Redis / RabbitMQ / Search

| Biến | Default (configs) | Ghi chú |
|---|---|---|
| `DB_HOST`, `DB_PORT` | port `5432` | Managed: pooler host + `6543` |
| `DB_USER`, `DB_PASSWORD`, `DB_NAME` | — | secret |
| `DB_SSLMODE` | `disable` (loader) | Managed (Supabase) = `require` |
| `DB_LISTENER_HOST` | — | host **direct** cho migrate/`LISTEN`/advisory lock |
| `DB_LISTENER_PORT` | `5432` | direct port (chỉ override khi khác) |
| `DB_LISTENER_SSLMODE` | `disable` | Managed = `require` |
| `DATABASE_MIGRATE_ON_START` | `false`/`true` | theo `config*.yaml` (VPS/managed = true) |
| `DATABASE_MIGRATION_TABLE` | rỗng = `goose_db_version` | per-service: `_commerce`/`_ai`; deploy set (FR-035) |
| `DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER` | `true` | trong configs |
| `DATABASE_QUERY_EXEC_MODE` | auto | **không cần set**: pooler 6543 → `simple_protocol`, direct → prepared (FR-046) |
| `DB_DRILL_SCHEMA` | rỗng | chỉ dùng khi drill runbook (FR-027) |
| `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` | port `6379` | secret |
| `REDIS_TLS_ENABLED` | `false` | Managed (Redis Cloud) = `true` |
| `RABBITMQ_HOST`, `RABBITMQ_PORT`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD` | port `5672` | secret |
| `RABBITMQ_TLS_ENABLED`, `RABBITMQ_MANAGEMENT_TLS_ENABLED` | `false` | Managed (CloudAMQP) = `true` |
| `OUTBOX_LISTENER_ENABLED` | `true` | tắt → fallback poll (FR-009/010) |
| `ELASTICSEARCH_ENABLED` | `false` | FR-001; fallback DB |
| `ELASTICSEARCH_ADDRESSES`, `ELASTICSEARCH_USER`, `ELASTICSEARCH_PASSWORD` | — | chỉ dùng khi bật ES |

> **Option A**: migration Goose mở **connection direct riêng** từ
> `DB_LISTENER_HOST`/`DB_LISTENER_PORT`/`DB_LISTENER_SSLMODE` (không qua pooler);
> `LISTEN/NOTIFY` + job advisory lock dùng cùng direct DSN. Vì vậy `DB_LISTENER_HOST`
> là **bắt buộc ở managed** (workflow enforce).

## 5. Auth, ingress, email, thanh toán, media, AI

| Biến | Default (configs) | Ghi chú |
|---|---|---|
| `JWT_SECRET` | — | secret; HS256 |
| `JWT_ALGORITHM` | `HS256` | Managed nên `ES256` (FR-022) |
| `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY` | rỗng | PEM; **chỉ identity giữ private key**. Rỗng = HS256 tạm thời |
| `INGRESS_ACCOUNT_URL` | — | VPS compose: `http://account:8080`; Managed: URL `run.app` (bắt buộc) |
| `INGRESS_COMMERCE_URL` | — | VPS: `http://commerce:8080`; Managed: `run.app` |
| `INGRESS_AI_URL` | — | VPS: `http://ai:8080`; Managed: `run.app` |
| `INGRESS_TOKEN_AUDIENCE` | — | Managed **bắt buộc** (validation); audience ID token |
| `INGRESS_TOKEN_HELPER_URL`, `INGRESS_AUTH_HEADER` | — | **chưa dùng ở app** (helper do Nginx gọi); có thể để trống |
| `EMAIL_SENDER_EMAIL`, `EMAIL_SMTP_USER`, `EMAIL_SMTP_KEY`, `EMAIL_REPLY_TO` | — | Brevo SMTP (ADR 022); `REPLY_TO` tùy chọn |
| `PAYOS_CLIENT_ID`, `PAYOS_API_KEY`, `PAYOS_CHECKSUM_KEY`, `PAYOS_ENDPOINT` | — | secret |
| `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET`, `CLOUDINARY_API_URL` | — | dependency có sẵn (FR-053) |
| `VISION_*`, `EMBEDDING_*`, `CHAT_TEXT_*`, `RECOMMENDATION_TEXT_*`, `FREE_TEXT_*` | — | provider/model/endpoint/key; fallback tùy chọn |

## 6. Nginx / compose / Cloud Run (không thuộc app)

| Biến | Default | Ghi chú |
|---|---|---|
| `COMPOSE_PROJECT_NAME`, `BACKEND_IMAGE`, `BACKEND_HEALTH_PATH` | — | compose VPS |
| `API_DOMAIN` | — | hostname `api.closy` |
| `API_V2_DOMAIN` | — | hostname `api-v2` |
| `NGINX_MODE` | `parallel` | `parallel`/`parallel-v2`/`repatriation`/`rollback` (`parallel-v2`: api.closy → 3 service VPS, api-v2 → Cloud Run, không monolith) |
| `COMPOSE_PROFILES` | `parallel` | `parallel` cho mode parallel, `parallel-v2` cho mode parallel-v2 (bật profile `token-helper`); để trống ở repatriation/rollback |
| `NGINX_CLIENT_MAX_BODY_SIZE` | `20m` | |
| `ACCOUNT_RUN_URL`, `COMMERCE_RUN_URL`, `AI_RUN_URL` | — | target proxy `api-v2` (parallel) |
| `TOKEN_HELPER_URL` | `http://token-helper:8082` | Nginx gọi trong compose network |
| `TOKEN_HELPER_SA_KEY_FILE` | `./token-helper-sa.json` | SA key `0600` |
| `NGROK_AUTHTOKEN`, `NGROK_DOMAIN`, `NGROK_TARGET` | — | chỉ local tunnel |

## 7. Biến để trống — có cần điền không?

### 7.1 `.env` (local)
Các biến trống dưới đây **không bắt buộc** cho `make docker-dev-up` (compose tự set
host/service-name), chỉ cần khi chạy host (`make run`) nhiều service:

| Biến trống | Cần điền khi | Giá trị |
|---|---|---|
| `PORT` | không (dùng `SERVER_PORT`) | — |
| `DB_DRILL_SCHEMA` | chỉ khi chạy drill | `drill_<ngày>_<n>` |
| `DATABASE_MIGRATION_TABLE` | compose set per-service | để trống (Account default) |
| `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY` | khi chuyển ES256 | PEM |
| `INGRESS_ACCOUNT/COMMERCE/AI_URL` | khi `make run` nhiều service | `http://localhost:8080/8102/8103` |
| `INGRESS_TOKEN_HELPER_URL`, `INGRESS_TOKEN_AUDIENCE` | không (app chưa dùng/không managed) | — |
| `SWAGGER_ACCESS_CODE`, `EMAIL_REPLY_TO`, `VISION_FALLBACK_*` | tùy chọn | — |

### 7.2 `.env.production` (VPS) — **PHẢI điền trước khi deploy**

| Biến trống | Khi nào cần | Cách lấy |
|---|---|---|
| `API_V2_DOMAIN` | mode `parallel` | `api-v2.closy.hycat.online` (`11-dns-vercel-ci-cd.md`) |
| `ACCOUNT_RUN_URL`, `COMMERCE_RUN_URL`, `AI_RUN_URL` | mode `parallel` | `gcloud run services describe <svc> --region asia-southeast1 --format='value(status.url)'` (`05-compute-cloudrun.md`) |
| `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY` | khi bật ES256 | sinh cặp EC P-256, dán PEM (`docs/system-design/microservices/04-auth-asymmetric-jwt.md`) |
| `INGRESS_TOKEN_AUDIENCE` | chỉ managed | URL `run.app` (secret GitHub) |
| `EMAIL_REPLY_TO`, `PORT`, `DB_DRILL_SCHEMA`, `DATABASE_MIGRATION_TABLE` | không (default/drill/per-service) | — |

## 8. GitHub Actions secrets cho managed

Xem bảng đầy đủ ở `00` §5. Danh sách tên: `DB_USER`, `DB_PASSWORD`, `DB_NAME`,
`DB_HOST`, `DB_PORT`, `DB_LISTENER_HOST`, `DB_LISTENER_SSLMODE`, `REDIS_HOST`,
`REDIS_PASSWORD`, `RABBITMQ_HOST`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD`,
`JWT_SECRET`, `JWT_ALGORITHM`, `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`,
`SWAGGER_ACCESS_CODE`, `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`,
`CLOUDINARY_API_SECRET`, `EMAIL_SMTP_USER`, `EMAIL_SMTP_KEY`, `PAYOS_CLIENT_ID`,
`PAYOS_API_KEY`, `PAYOS_CHECKSUM_KEY`, `INGRESS_ACCOUNT_URL`, `INGRESS_COMMERCE_URL`,
`INGRESS_AI_URL`, `INGRESS_TOKEN_AUDIENCE`.

Workflow tự đặt: `SERVER_ENV=managed`, `APP_MODE=http|worker`, `DB_SSLMODE=require`,
`DB_LISTENER_SSLMODE=require`, `DATABASE_MIGRATE_ON_START=true`,
`DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER=true`, `ELASTICSEARCH_ENABLED=false`,
`REDIS_TLS_ENABLED=true`, `RABBITMQ_TLS_ENABLED=true`,
`RABBITMQ_MANAGEMENT_TLS_ENABLED=true`, `DATABASE_MIGRATION_TABLE` (per-service).

## 9. Migration qua session DSN (FR-012/FR-035)

Migration, `LISTEN/NOTIFY` và advisory lock là **session-scoped**: chạy trên **direct
connection 5432**, KHÔNG qua transaction pooler 6543 (FR-046).

```bash
goose -dir services/<svc>/database/migrations \
  postgres "host=$DB_LISTENER_HOST user=$DB_USER password=$DB_PASSWORD \
    dbname=$DB_NAME port=${DB_LISTENER_PORT:-5432} sslmode=${DB_LISTENER_SSLMODE:-disable}" status

make migrate-up svc=<account|commerce|ai>
```

## 10. Biến đã bỏ / không nên set

- `DB_LISTENER_SSLMODE`, `DB_LISTENER_PORT` — cần khi direct khác mặc định (managed);
  local/VPS để trống là được.
- `DATABASE_QUERY_EXEC_MODE` — **auto** (pooler-only); chỉ set để override direct.
- `INGRESS_AUTH_HEADER` — hằng số trong `configs` (đã có default).
- `ELASTICSEARCH_ENABLED`, `OUTBOX_LISTENER_ENABLED`, `DATABASE_MIGRATE_ON_START`,
  `DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER`, `REDIS_TLS_ENABLED`, `RABBITMQ_TLS_ENABLED`,
  `RABBITMQ_MANAGEMENT_TLS_ENABLED`, `JWT_ALGORITHM` — default trong `configs`;
  chỉ set khi cần khác.
