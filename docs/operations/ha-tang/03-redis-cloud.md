# 03 — Redis Cloud Essentials 250MB

## Tài nguyên

- Redis Cloud **Essentials 250MB**, **256 connections**, **TLS bắt buộc** (`rediss`).
- Dùng chung cho: cache, session blacklist (thu hồi JWT), quota lock, **rate-limit
  backend** (`IRateLimiter`), outbox coordination.

## Kết nối (FR-004)

- `redis.tls_enabled=true` trên managed (client dùng `TLSConfig` TLS1.2);
  mặc định `false` cho VPS tương thích.
- Env: `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_TLS_ENABLED=true`.

## Ngân sách connection (FR-047)

- Tổng client conn (3 service × HTTP `max-instances=2` + worker singleton) **≤ 256**
  với headroom ≥ 20%; `database.max_open_conns`/`concurrency=40` đã tính trong bảng budget (plan).
- Verify bằng test tải nhẹ trước khi mở traffic (SC-021).

## Verify

- [ ] Kết nối TLS thành công từ Cloud Run (`rediss://`).
- [ ] Blacklist/quota/rate-limit đọc-ghi đúng qua TLS.
- [ ] Không vượt 256 conn trong test tải.
