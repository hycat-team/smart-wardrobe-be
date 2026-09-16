# Bảng biến môi trường (020-platform-migration)

> Nguồn đầy đủ: `.env.example`, `.env.production.example`. File này giải thích các
> biến 020 theo nhóm. T090 (Phase 6) hoàn thiện tài liệu runbook chi tiết từng biến.

## Migrate qua session DSN (T025 — FR-012/FR-035)

Migration, `LISTEN/NOTIFY` và advisory lock là **session-scoped**: chúng MUST chạy
trên **direct connection 5432**, KHÔNG bao giờ qua transaction pooler 6543 (FR-046).

```bash
# Trạng thái migrate của một service (direct connection):
goose -dir services/<svc>/database/migrations \
  postgres "host=$DB_LISTENER_HOST user=$DB_USER password=$DB_PASSWORD \
    dbname=$DB_NAME port=$DB_LISTENER_PORT sslmode=$DB_LISTENER_SSLMODE" status

# Chạy migrate pending (ngoài luồng khởi động service):
goose -dir services/<svc>/database/migrations \
  postgres "host=$DB_LISTENER_HOST user=$DB_USER password=$DB_PASSWORD \
    dbname=$DB_NAME port=$DB_LISTENER_PORT sslmode=$DB_LISTENER_SSLMODE" up

# Hoặc qua Makefile (đọc cùng biến từ môi trường):
make migrate-status svc=<account|commerce|ai>
make migrate-up svc=<account|commerce|ai>
```

Biến liên quan: `DB_LISTENER_HOST`, `DB_LISTENER_PORT` (mặc định 5432),
`DB_LISTENER_SSLMODE`, `DATABASE_MIGRATE_ON_START`, `DATABASE_QUERY_EXEC_MODE`,
`DATABASE_DRILL_SCHEMA`.

## Các nhóm biến 020 khác (T090 chi tiết hoá)

- `APP_MODE`, `SERVER_ENV`, `PORT`/`SERVER_PORT` — topology runtime (FR-044).
- `ELASTICSEARCH_ENABLED`, `ELASTICSEARCH_ADDRESSES` — search engine (FR-001).
- `REDIS_TLS_ENABLED`, `RABBITMQ_TLS_ENABLED`, `RABBITMQ_MANAGEMENT_TLS_ENABLED`,
  `OUTBOX_LISTENER_ENABLED` — TLS + listener (FR-004/FR-009).
- `JWT_ALGORITHM`, `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY` — ký bất đối xứng (FR-022).
- `INGRESS_*` — routing managed, bắt buộc ở profile managed (FR-005).
