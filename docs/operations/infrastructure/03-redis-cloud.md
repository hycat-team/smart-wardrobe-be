# 03 — Redis Cloud Essentials 250MB

Điều kiện: đã có subscription Redis Cloud (`00` §0.4). Dùng cho cache, session blacklist
(thu hồi JWT), quota lock, rate-limit backend (`IRateLimiter`), outbox coordination.

## 1. Tạo subscription

1. Mở https://app.redislabs.com → **New subscription** → **Essentials**, 250MB.
2. Chờ provision xong → mở subscription.

## 2. Lấy credential & bật TLS

Vào tab **Configuration** / **Security**:

| Giá trị | Lấy ở đâu | Secret |
|---|---|---|
| Public endpoint (host) | Configuration → Endpoint | `REDIS_HOST` |
| Port riêng (thường 12000+, KHÔNG phải 6379) | Configuration → Endpoint | `REDIS_PORT` |
| Default user password | Security → Access control | `REDIS_PASSWORD` |

Bật **TLS** (managed workflow tự set `REDIS_TLS_ENABLED=true`).

```bash
gh secret set REDIS_HOST     --body "<host>.cloud.redislabs.com"
gh secret set REDIS_PORT     --body "12000"
gh secret set REDIS_PASSWORD --body "<password>"
```

## 3. Kiểm tra kết nối TLS

```bash
redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" --tls -c PING
# PONG
```

## 4. Ngân sách connection (FR-047)

- Tổng client conn (3 HTTP × `max-instances=2` + 3 worker singleton) **≤ 256**, headroom ≥ 20%.
- `database.max_open_conns` / `concurrency=40` đã tính trong budget ở plan.

## 5. Verify

- [ ] Kết nối TLS thành công từ Cloud Run (`rediss://`).
- [ ] Blacklist/quota/rate-limit đọc-ghi đúng qua TLS.
- [ ] Không vượt 256 conn khi test tải (SC-021).

## 6. Xử lý sự cố

| Hiện tượng | Xử lý |
|---|---|
| `tls: handshake failure` | Xác nhận TLS bật trên Redis Cloud và `REDIS_TLS_ENABLED=true` |
| `connection refused` | Sai port (dùng port riêng của endpoint, không phải 6379) |
| Vượt 256 conn | Giảm `max-instances` hoặc `max_open_conns`; kiểm tra worker không nhân bản |
