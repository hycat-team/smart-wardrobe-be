# 10 — Chuyển dữ liệu VPS → Supabase (dump/restore + backfill + checksum)

Dùng cho lần dump đầu (test `api-v2`) và **re-dump tại cutover**.

## 1. Preflight

- [ ] Supabase đã có schema + `search_path` (`02-supabase-postgres.md`).
- [ ] VPS còn phục vụ monolith; chấp nhận cửa sổ ghi nhỏ (ghi rõ mốc thời gian).
- [ ] Đủ dung lượng đĩa cho file dump trên VPS (`df -h`).

## 2. Dump VPS

```bash
TS=$(date +%F_%H%M)
mkdir -p /opt/closy/backup
cd /opt/closy
docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
  exec -T postgres pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc > "backup/closy_${TS}.dump"
sha256sum "backup/closy_${TS}.dump" | tee "backup/closy_${TS}.sha256"
```

## 3. Restore vào Supabase (direct 5432)

```bash
pg_restore --clean --if-exists --no-owner --no-acl \
  -d "host=$DB_LISTENER_HOST port=5432 user=$DB_USER dbname=$DB_NAME sslmode=require" \
  "backup/closy_${TS}.dump"
```
Sau restore, đảm bảo `search_path` server-side vẫn đúng (`02` §1.4).

## 4. Backfill (nếu có migration mới giữa chừng)

```bash
make migrate-up svc=account
make migrate-up svc=commerce
make migrate-up svc=ai
```

## 5. Verify checksum (SC-008)

```bash
# Row-count bảng chính — chạy trên cả VPS và Supabase rồi so sánh
psql "<DSN nguồn>" -c "
  select relname, n_live_tup
  from pg_stat_user_tables
  where relname in ('users','subscriptions','wardrobe_items','outbox')
  order by relname;"
```
- [ ] Checksum khớp 100% tại mốc dump.
- [ ] Supabase query đúng 8 schema module, 0 lỗi prepared statement qua pooler.
- [ ] Chênh lệch chỉ đến từ ghi mới sau mốc dump (ghi rõ cửa sổ).

## 6. Re-dump tại cutover

Vì VPS vẫn nhận ghi của người dùng thật, lặp lại §2–§5 tại thời điểm merge/cutover và
xác nhận không mất ghi trong cửa sổ chuyển.

## 7. Dùng khi song song 2 trục (đồng bộ định kỳ)

Cùng luồng §2–§5 nhưng chạy định kỳ trước mỗi đợt verify `api-v2`: dump Postgres local
VPS → restore vào Supabase → verify checksum. VPS local luôn là live; Supabase chỉ verify.

## 8. Lưu ý

- Lần dump đầu chỉ phục vụ **test nội bộ** `api-v2`; VPS 3-service là nguồn ghi chính
  ở GĐ1. Khi song song 2 trục, mỗi bên ghi DB riêng (divergence có chủ đích, xem `09` §4).
- Mọi thao tác production đều backup + verify checksum trước.
- Drill (FR-027) dùng schema tạm, không áp dụng quy trình này lên schema chính.
