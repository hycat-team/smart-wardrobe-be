# 11 — Về lại VPS (migrate ngược, DNS, tắt funded)

Chi tiết kỹ thuật cho bước repatriation; runbook tổng ở
`../repatriate-to-vps.md`.

## Migrate ngược dữ liệu

- **Supabase → Postgres VPS**: dump (direct 5432) → restore vào docker Postgres;
  verify checksum bảng chính (SC-008).
- **Drain AMQP**: ngừng producer managed, đợi queue CloudAMQP rỗng, rồi mới tắt;
  event chưa xử lý phải được migrate/poll hết.
- **Redis**: snapshot RDB (nếu cần warmup cache) hoặc bỏ (cache rebuild tự nhiên).

## DNS về VPS

- Trỏ `api.closy` (+ `api-v2` trong grace period) về VPS; TTL thấp trước khi đổi.
- `api-v2` redirect 301 → `api.closy` 14–30 ngày rồi xoá DNS (FR-055/SC-029).

## Tắt funded trước renew

- Supabase Pro, CloudAMQP Tiger, Redis Cloud Essentials: tắt/huỷ **trước ngày
  renew** theo funding clock T-30/T-14/T-7/T-1 (T107); giữ bằng chứng không bị
  charge thêm chu kỳ (SC-011).

## Env profile `vps`

`sslmode=disable`, TLS off, `migrate_on_start=true`; xem `07-bang-bien-moi-truong.md`.
Kiểm thử local profile `vps` trước (T104): 3 service chạy đủ, không sửa code.
