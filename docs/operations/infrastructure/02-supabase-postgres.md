# 02 — Supabase Postgres (1 project, 1 database dùng chung)

Áp dụng cho môi trường **managed** (Cloud Run). Điều kiện: đã có project Supabase Pro (`00` §0.3).

## 1. Tạo project & extension

1. Mở https://supabase.com/dashboard → **New project** → chọn region Singapore.
2. Lưu **Database password** (dùng cho `DB_PASSWORD`).
3. Vào **SQL Editor**, chạy:
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
4. Đặt `search_path` **server-side** cho role thật (Supabase thường là `postgres`):
   ```sql
   alter role postgres set search_path =
     public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared;
   ```

## 2. Lấy 2 đầu nối (FR-046 — bắt buộc)

Vào **Project Settings → Database → Connection string**:

| Mục đích | DSN | Cổng | Ghi chú |
|---|---|---|---|
| Runtime app | Pooler (Supavisor **transaction mode**) | **6543** | app tự dùng `simple_protocol` (không set env) |
| Migration / `LISTEN` / advisory lock | Direct connection | **5432** | session-scoped |

Map vào secret GitHub (`00` §5):

```bash
gh secret set DB_HOST            --body "<pooler-host>"            # ...pooler.supabase.com
gh secret set DB_PORT            --body "6543"
gh secret set DB_USER            --body "postgres.<project-ref>"   # theo connection string hiển thị
gh secret set DB_PASSWORD        --body "<database-password>"
gh secret set DB_NAME            --body "postgres"
gh secret set DB_LISTENER_HOST   --body "db.<project-ref>.supabase.co"
gh secret set DB_LISTENER_SSLMODE --body "require"
```
> Không cần set `DB_LISTENER_PORT` (mặc định `5432`).

## 3. Kiểm tra kết nối

```bash
# Runtime pooler (giả lập app): phải trả về search_path gồm 8 schema
psql "host=$DB_HOST port=$DB_PORT user=$DB_USER dbname=$DB_NAME sslmode=require" \
  -c "select current_setting('search_path');"

# Direct (migration/LISTEN): phải kết nối được
psql "host=$DB_LISTENER_HOST port=5432 user=$DB_USER dbname=$DB_NAME sslmode=require" -c "select 1;"
```

## 4. Migration ownership (FR-035)

- Migration lịch sử do **Account** giữ (`services/account/database/migrations`).
- Migration mới tạo trong chính service phát sinh.
- Mỗi service chạy Goose khi boot với **bảng version riêng**:
  `goose_db_version` / `goose_db_version_commerce` / `goose_db_version_ai`
  + `WithAllowOutofOrder` + advisory lock dùng chung.
- Env do workflow managed tự set: `DATABASE_MIGRATE_ON_START=true`,
  `DATABASE_MIGRATION_TABLE` (per-service), `DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER=true`.

## 5. Verify

- [ ] App boot qua pooler, **0 lỗi prepared statement** (SC-020).
- [ ] Query resolve đúng 8 schema module.
- [ ] `LISTEN`/advisory lock chạy trên direct 5432.
- [ ] Account không chạy lại migration đã apply.

## 6. Xử lý sự cố

| Hiện tượng | Nguyên nhân | Xử lý |
|---|---|---|
| `prepared statement already exists` | query qua pooler transaction | Kiểm tra `DATABASE_QUERY_EXEC_MODE` không bị set sai; pooler 6543 phải dùng `simple_protocol` |
| `relation ... does not exist` | thiếu schema trong `search_path` | Chạy lại `alter role ... set search_path` ở bước 1.4 |
| Migration treo | LISTEN/advisory lock qua pooler | Xác nhận `DB_LISTENER_HOST` trỏ direct 5432 |
