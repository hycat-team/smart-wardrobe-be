# 12 — Repatriation về VPS (migrate ngược, DNS, tắt funded)

Chi tiết kỹ thuật cho bước repatriation; runbook tổng ở `../repatriate-to-vps.md`.

## 1. Điều kiện tiên quyết

- [ ] Gate song song đạt (`09` §4) và owner quyết định repatriation.
- [ ] VPS còn dư đĩa; image 3 service đã push GHCR (`deploy-vps-services.yml`).
- [ ] `.env.production` profile `vps` đã kiểm thử local trước đó.

## 2. Deploy 3 service lên VPS

```bash
gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all
RUN_ID=$(gh run list --workflow deploy-vps-services.yml -R "<OWNER/REPO>" --limit 1 \
  --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" -R "<OWNER/REPO>"
# smoke
curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health
```

> Dừng backend cũ (monolith) nếu còn chạy — tránh 2 writer cùng DB (idempotent):
> ```bash
> docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml stop backend || true
> docker ps --filter name=backend --format '{{.Names}} {{.Status}}'   # phải trống
> ```
> Verify không còn writer cũ: consumer count RabbitMQ phía monolith về 0, outbox chỉ poll 1 phía.

## 3. Migrate ngược dữ liệu (Supabase → Postgres VPS)

```bash
# a) dump Supabase qua direct 5432
pg_dump "host=$DB_LISTENER_HOST port=5432 user=$DB_USER dbname=$DB_NAME sslmode=require" \
  -Fc > /opt/closy/backup/supabase_$(date +%F_%H%M).dump
sha256sum /opt/closy/backup/supabase_*.dump

# b) restore vào Postgres docker
docker compose --env-file /opt/closy/.env.production -f /opt/closy/deployments/docker-compose.prod.yml \
  exec -T postgres pg_restore --clean --if-exists --no-owner -U "$DB_USER" -d "$DB_NAME" \
  < /opt/closy/backup/supabase_<TS>.dump
```
- [ ] Checksum bảng chính khớp (SC-008).

## 4. Drain AMQP

1. Ngừng producer managed (dừng worker Cloud Run hoặc chặn publish).
2. Đợi queue CloudAMQP rỗng:
   ```bash
   curl -sf -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" \
     "https://$RABBITMQ_HOST/api/queues/<vhost>" | grep -o '"messages":[0-9]*'
   ```
3. Event chưa xử lý phải được migrate/poll hết trước khi tắt managed.

## 5. Redis

- Snapshot RDB nếu cần warmup: `docker compose ... exec redis redis-cli -a "$REDIS_PASSWORD" BGSAVE`.
- Hoặc bỏ qua (cache tự rebuild) — không bắt buộc.

## 6. DNS về VPS

1. Hạ TTL `api.closy` (và `api-v2`) xuống thấp trước khi đổi.
2. Trỏ `api.closy` về VPS; `api-v2` **redirect 301** → `api.closy` 14–30 ngày rồi xoá DNS
   (FR-055/SC-029).
3. Verify host × route + SSE + auth + upload.

## 7. Tắt funded trước renew

Supabase Pro, CloudAMQP Tiger, Redis Cloud Essentials: tắt/huỷ **trước ngày renew** theo
funding clock T-30/T-14/T-7/T-1; giữ bằng chứng không bị charge thêm chu kỳ (SC-011).
Teardown GCP trial: xem `../gcp-trial-teardown.md`.

## 8. Env profile `vps`

`sslmode=disable`, TLS off, `migrate_on_start=true`; xem `08-environment-variables.md`.
Kiểm thử local profile `vps` trước: 3 service chạy đủ, **không sửa code**.
