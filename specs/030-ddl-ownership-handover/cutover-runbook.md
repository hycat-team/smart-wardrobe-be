# Runbook cắt over (cutover) production — bàn giao quyền sở hữu DDL (030)

**Spec**: [spec.md](spec.md) | **Kiểm chứng đã xong**: [verification-log.md](verification-log.md)
(T023 V2 + V5, T025 — đọc trước khi đặt giờ G)

Tài liệu này DÀNH CHO THAO TÁC TAY của chủ hệ thống trên production.
Agent không chạy các bước dưới đây (xem tasks.md T032).
Mọi lệnh mẫu trong tài liệu là tiếng Anh nguyên văn (tên lệnh, tên bảng, tên cột);
phần diễn giải là tiếng Việt.

> Nguyên tắc cao nhất: rollback khi hỏng = khôi phục từ backup.
> KHÔNG BAO GIỜ chạy `Down` trên database có dữ liệu
> (`Down` của 2 baseline là `DROP TABLE` toàn bộ 7+3 bảng — destructive,
> đã chứng minh ở T025 chỉ an toàn trên database dựng trắng).
> Không sửa chữa thủ công trên dữ liệu sống (no fix-forward on live data).

---

## 0. Điều kiện tiên quyết (preconditions) — thiếu 1 điều là chưa được đặt giờ G

1. **Rehearsal dev xanh 100% (cổng go/no-go SC-007)**: T023 V5 đã pass trên
   database dev (schema y chang production): versions `35 / 4 / 2`, chỉ còn
   constraint tên mới, đếm dòng nghiệp vụ trước/sau bằng nhau, 3 service boot
   xanh, smoke V4 xanh. Không có V5 xanh thì không có giờ G.
2. **Backup mới nhất + đã kiểm chứng khôi phục được**: xem §1.
3. **Cửa sổ full-stop 1–2h không có user truy cập**, thời điểm do chủ hệ thống
   tự quyết (spec Clarifications). Trong cửa sổ: dừng traffic vào cả 3 service
   (gateway + worker), không yêu cầu cơ chế chặn read-only tự động.
4. **Lệnh ngừng bắn migration FR-011 vẫn hiệu lực cho tới T028**: không tạo
   migration mới chạm `aiexecutor.*` / `dashboard.*` trước và trong cutover.
5. **Mọi lần boot binary kiểm chứng đều tắt đường mail thật**:
   `EMAIL_ENABLED=false` cho MỌI binary khởi động ngoài production containers
   (V1b từng lọt 1 mail tới địa chỉ seed vì thiếu bước này — trong ngưỡng hiến
   pháp nhưng không được lặp lại). Production containers giữ cấu hình mail
   hiện hành của production, không đổi.
6. **Ngưỡng abort đã chốt**: `lock_timeout = 10 giây` (đo rename thật 0.415ms,
   cả file baseline ~7–40ms — xem verification-log T023; 10s lớn hơn 20000 lần
   thời gian rename nhưng đủ nhanh để abort khi kẹt lock thật).

---

## 1. Backup + kiểm chứng restore (đường về duy nhất)

```powershell
# 1a. Backup đầy đủ production (custom format).
docker exec -e PGPASSWORD=<mat-khau-prod> <ten-container-postgres-prod> pg_dump -U <user-prod> -Fc -f /tmp/prod_before_030.dump <ten-db-prod>
# 1b. Xác minh file backup đọc được (bắt buộc trước khi đi tiếp).
docker exec <ten-container-postgres-prod> pg_restore --list /tmp/prod_before_030.dump
# Kỳ vọng: đọc được, ~399 TOC entries (mốc dev; prod theo số thực tế).
# 1c. Giữ 1 bản sao thứ hai ngoài container (host/VPS hoặc object storage).
docker cp <ten-container-postgres-prod>:/tmp/prod_before_030.dump /duong-dan-an-toan/prod_before_030.dump
```

Chỉ đi tiếp khi `pg_restore --list` đọc được và 2 bản backup đã nằm 2 nơi.
Ghi số bytes + số TOC entries vào sổ giờ G.

---

## 2. Chụp số liệu trước cutover (để đối chiếu §5)

```powershell
# 2a. Sổ version (mốc dev: 35 / 3 / chưa có bảng AI; prod đối chiếu điểm cắt T001).
docker exec -e PGPASSWORD=<mat-khau-prod> <ten-container-postgres-prod> psql -U <user-prod> -d <ten-db-prod> -t -c "SELECT count(*), max(version_id) FROM goose_db_version;"
docker exec -e PGPASSWORD=<mat-khau-prod> <ten-container-postgres-prod> psql -U <user-prod> -d <ten-db-prod> -t -c "SELECT count(*), max(version_id) FROM goose_db_version_commerce;"
docker exec -e PGPASSWORD=<mat-khau-prod> <ten-container-postgres-prod> psql -U <user-prod> -d <ten-db-prod> -t -c "SELECT to_regclass('public.goose_db_version_ai');"
# 2b. Tên constraint grant hiện tại (kỳ vọng: còn tên cũ user_ai_policy_grants_check).
docker exec -e PGPASSWORD=<mat-khau-prod> <ten-container-postgres-prod> psql -U <user-prod> -d <ten-db-prod> -t -c "SELECT conname FROM pg_constraint WHERE conname IN ('user_ai_policy_grants_check','ai_policy_grants_effective_window_check');"
# 2c. Đếm dòng toàn bộ bảng nghiệp vụ 8 schema module
# (identity, subscription, wardrobe, fashion, brand, community, dashboard, aiexecutor)
# — danh sách 55 bảng và kết quả mẫu xem verification-log T023.
```

---

## 3. Thứ tự deploy: Account → AI → Commerce

1. Dừng traffic (full-stop): dừng gateway và 3 service cũ.
2. Deploy image mới của **Account** trước, chờ boot xanh
   (`service listening` + `/api/v1/health` 200). Kỳ vọng migration: `Up ~0`
   (`goose: no migrations to run`, version Account giữ nguyên).
3. Deploy **AI**, chờ boot xanh. Kỳ vọng: baseline `20261016213000` applied
   1 lần (rename constraint cũ → tên mới diễn ra tại đây, metadata-only).
4. Deploy **Commerce**, chờ boot xanh. Kỳ vọng: baseline `20261016213001`
   applied 1 lần.
5. Boot lại lần nữa (hoặc quan sát log): cả 3 cây phải no-op
   (`no migrations to run`) — chứng minh hội tụ.

Không đảo thứ tự (Account giữ toàn bộ lịch sử cũ nên phải xong trước).

---

## 4. Giám sát lock trong lúc rename + ngưỡng abort

Rename giữ lock độc quyền (AccessExclusive) trên bảng grant trong chốc lát.
Mở 1 session monitor riêng, chạy trước khi deploy AI:

```sql
-- 4a. Session nào đang giữ/chờ lock trên bảng grant.
SELECT pid, usename, wait_event_type, wait_event, query
FROM pg_stat_activity WHERE query ILIKE '%ai_policy_grants%';
SELECT locktype, mode, granted, pid
FROM pg_locks WHERE relation = 'aiexecutor.ai_policy_grants'::regclass;
-- 4b. Abort an toàn nếu chờ lock quá ngưỡng: đặt trước mỗi câu DDL tay
-- (migration baseline đã idempotent nên boot lại sau abort là an toàn).
SET lock_timeout = '10s';
```

**Tiêu chí abort (1 điều đỏ là dừng + restore §6, không sửa tiếp):**

- Bất kỳ service nào boot đỏ (migration failed, không listening, health ≠ 200).
- Đếm dòng trước/sau lệch ở bất kỳ bảng nghiệp vụ nào.
- Schema sau hội tụ lệch phạm vi handover so với V1 (so 10 bảng / 156 cột /
  48 constraint / 27 index — xem §5).
- Smoke §5 đỏ.
- Lock chờ vượt `lock_timeout = 10s` hoặc xuất hiện deadlock.
- Bất kỳ lệnh nào trỏ nhầm database / xóa volume (dừng ngay lập tức).

---

## 5. Smoke sau cutover (V4) + checklist nghiệm thu

Smoke trên production sau khi 3 service đã xanh (dùng tài khoản kiểm chứng
đánh dấu rõ, KHÔNG dùng tài khoản người thật):

1. Đăng ký → confirm OTP → login được (identity viết/xác thực xanh).
2. Xem danh sách gói cước — đủ gói đang bán (subscription đọc xanh).
3. Tạo + liệt kê phiên chat AI (AI phục vụ xanh, grant ghi đúng — kiểm tra
   `aiexecutor.ai_policy_grants` có bản ghi cho tài khoản kiểm chứng).
4. Gọi dashboard admin bằng tài khoản kiểm chứng (không phải admin) → kỳ vọng
   **403** (RBAC còn nguyên); dashboard đọc bằng admin → 200.

**Checklist post-cutover (ghi từng dòng vào sổ giờ G):**

- [ ] `goose_db_version`: **35 dòng**, max giữ nguyên (không mất/không thêm).
- [ ] `goose_db_version_commerce`: **4 dòng** (3 cũ + 1 baseline `20261016213001`).
- [ ] `goose_db_version_ai`: **2 dòng** (dòng `0` + baseline `20261016213000`).
- [ ] Constraint grant chỉ còn `ai_policy_grants_effective_window_check`,
      tên cũ `user_ai_policy_grants_check` đã biến mất, không trùng.
- [ ] Đếm dòng toàn bộ bảng nghiệp vụ trước/sau **bằng nhau tuyệt đối**
      (ngoại trừ các dòng do chính smoke tạo ra — ghi riêng).
- [ ] 3 service boot xanh, health 200, boot lần 2 no-op toàn bộ.
- [ ] Smoke 4 bước trên xanh (bước 4 ghi nhận 403 đúng vai trò + 200 đúng admin).
- [ ] Không có email lạ nào gửi ra ngoài trong cửa sổ
      (kiểm tra log worker `Email delivery skipped` nếu có boot kiểm chứng).

---

## 6. Rollback = restore từ backup (KHÔNG dùng Down)

Nếu bất kỳ tiêu chí abort nào đỏ:

```powershell
# 6a. Dừng toàn bộ 3 service mới (giữ nguyên trạng, không sửa gì thêm).
# 6b. Khôi phục production từ backup §1 (lệnh mẫu; thay tên theo môi trường).
docker exec -e PGPASSWORD=<mat-khau-prod> <ten-container-postgres-prod> pg_restore -U <user-prod> -d <ten-db-prod> --clean -c /tmp/prod_before_030.dump
# 6c. Deploy lại image cũ (trước bàn giao), boot xanh, chạy lại checklist §5
# trên dữ liệu đã khôi phục (số liệu phải khớp §2).
```

Nhắc lại: **không chạy `Down` trên database có dữ liệu** — `Down` của 2
baseline là `DROP TABLE` 7+3 bảng, sẽ xóa dữ liệu chứ không "quay về tên cũ".
`Down` chỉ an toàn trên database dựng trắng (đã chứng minh ở T025).

---

## 7. Môi trường bị quên, sót (staging bỏ xó, laptop dev, backup cũ)

Không cần phẫu thuật tay: boot bản mới tự hội tụ nhờ migration idempotent
(`IF NOT EXISTS` + seed guard + rename có điều kiện `pg_constraint`) —
môi trường nào boot bản mới là tự chữa lành, kẹt mới báo lỗi rõ ràng để thử
lại. Không có runbook riêng cho từng môi trường sót.

---

## 8. Đóng cửa sổ

- Mở lại traffic, quan sát log 15 phút (không migration-failed, không deadlock,
  không lỗi FK/constraint mới).
- Giữ file backup §1 ít nhất tới khi T031 (final report) xong.
- Ghi sổ giờ G: thời gian full-stop thực tế, số liệu §2/§5, kết quả smoke,
  mọi abort (nếu có) và cách xử lý.
