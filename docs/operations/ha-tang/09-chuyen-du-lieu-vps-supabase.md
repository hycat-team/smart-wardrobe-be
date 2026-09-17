# 09 — Chuyển dữ liệu VPS → Supabase (dump/restore + backfill + checksum)

## Quy trình (US3 AC2)

1. **Backup VPS**: `pg_dump` Postgres VPS (+ ghi mốc thời gian + checksum SHA256).
2. **Restore vào Supabase**: `psql`/restore qua direct 5432; bật `search_path` server-side.
3. **Backfill** (nếu có migration mới giữa chừng): chạy migration per-service trên
   Supabase trước khi mở traffic.
4. **Verify checksum**: so sánh row-count + checksum bảng chính (users, subscriptions,
   wardrobe items, outbox) giữa VPS và Supabase; chênh lệch phải = 0 (ngoại trừ
   ghi mới sau mốc dump — ghi rõ cửa sổ).
5. **Re-dump tại cutover**: vì VPS vẫn nhận ghi của người dùng thật, dump lại toàn
   bộ tại thời điểm merge/cutover và verify lại.

## Lưu ý song song

- Lần dump đầu chỉ phục vụ **test nội bộ** `api-v2`; monolith VPS vẫn là nguồn ghi chính.
- Mọi thao tác production đều backup + verify checksum trước (quy tắc tasks.md).
- Drill (FR-027) dùng schema tạm, không áp dụng quy trình này lên schema chính.

## Verify

- [ ] Checksum khớp 100% tại mốc dump; re-dump tại cutover khớp (không mất ghi).
- [ ] Supabase query đúng 8 schema module, 0 lỗi prepared statement qua pooler.
