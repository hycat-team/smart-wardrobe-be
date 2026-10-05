# Funding clock checklist — T-30/T-14/T-7/T-1 (T107/US7)

Kích hoạt khi còn 30 ngày tới hạn funded (3 tháng). Mỗi checkpoint cần người
xác nhận + bằng chứng thực thi. Repatriation phải hoàn tất trước ngày renew
(SC-011).

## T-30 — Bắt đầu funding clock

- [ ] Xác nhận ngày hết hạn từng hạng mục (Supabase/CloudAMQP/Redis/Cloud Run trial).
- [ ] Backup verify lần 1 (Supabase dump + checksum).
- [ ] Drill restore vào Postgres local (SC-008).
- [ ] Drill repatriation đầy đủ trên môi trường Cloud, ghi thời gian (mục tiêu ≤ 4h, T108).
- [ ] Người xác nhận: ______________ Ngày: ______________

## T-14 — Rehearsal rollback DNS

- [ ] Rehearsal đổi DNS về VPS + rollback lại (ghi thời gian).
- [x] ~~Image rollback `monolith-legacy` đã push GHCR + verify profile `rollback` (T097a).~~
  Bỏ — ADR 012: không còn rollback monolith tự động.
- [ ] Deploy 3-service VPS qua workflow thử 1 lần (T097b).
- [ ] Người xác nhận: ______________ Ngày: ______________

## T-7 — Migrate ngược + tắt funded

- [ ] Migrate ngược dữ liệu Supabase → Postgres VPS + checksum.
- [ ] DNS về VPS; verify host × route + SSE + auth + upload.
- [ ] Tắt funded items **trước ngày renew** (ghi bằng chứng không bị charge).
- [ ] Người xác nhận: ______________ Ngày: ______________

## T-1 — Đóng cửa sổ funded

- [ ] Xác nhận 0 hạng mục bị charge thêm chu kỳ (SC-011).
- [ ] Teardown trial GCP (một bước trong repatriation) + bill về 0 (SC-010, T105/T106).
- [ ] `api-v2` redirect 301 → `api.closy` (grace 14–30 ngày) hoặc đã xoá DNS (SC-029).
- [ ] Người xác nhận: ______________ Ngày: ______________
