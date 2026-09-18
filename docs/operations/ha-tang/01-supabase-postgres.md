# 01 — Supabase Postgres (1 project, 1 database dùng chung)

## Tài nguyên

- 1 project Supabase **Pro**, **1 database dùng chung** cho môi trường Cloud
  (không tách schema staging/prod; drill dùng schema tạm rồi xoá — FR-027).
- Extension `pgvector` bật. 8 schema module + `public`/`shared` trong `search_path`.

## Hai đầu nối (FR-046 — bắt buộc)

| Mục đích | DSN | Cổng | Yêu cầu |
|----------|-----|------|---------|
| Runtime app | Pooler (Supavisor **transaction mode**) | **6543** | `default_query_exec_mode=simple_protocol` (tắt prepared statement); `search_path` set **server-side** (`ALTER ROLE/DATABASE`), KHÔNG truyền qua DSN |
| Migration / `LISTEN` / advisory lock | Direct connection | **5432** | session-scoped; dùng `database.listener_*` / session-DSN |

> **Implementation (Option A)**: service tự mở **connection direct riêng** cho Goose
> migration (từ `DB_LISTENER_HOST`/`DB_LISTENER_PORT`/`DB_LISTENER_SSLMODE`) — không
> dùng connection runtime qua pooler. `LISTEN/NOTIFY` (outbox listener) và job
> advisory lock cũng dùng chung direct DSN này.

## Migration ownership (FR-035)

- Toàn bộ migration lịch sử do **Account** giữ (`services/account/database/migrations`).
- Migration mới tạo trong chính service phát sinh.
- Mỗi service tự chạy khi boot (Goose) với **bảng version riêng**
  (`goose_db_version` / `goose_db_version_commerce` / `goose_db_version_ai`) +
  `WithAllowOutofOrder` + advisory lock dùng chung.
- Env: `DATABASE_MIGRATE_ON_START=true` (managed), `DATABASE_MIGRATION_TABLE`
  (per-service), `DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER=true`.

## Verify

- [ ] App boot qua pooler, 0 lỗi prepared statement (SC-020).
- [ ] Query resolve đúng 8 schema module.
- [ ] `LISTEN`/advisory lock chạy trên direct 5432.
- [ ] Account không chạy lại migration đã apply.
