# 05 — VPS Nginx ingress tới Cloud Run

`api-v2.closy.hycat.online` phục vụ qua **VPS Nginx hiện hữu** (không Cloudflare/LB
mới). `api.closy...` **giữ nguyên** proxy tới monolith (`http://backend:8080`)
trong giai đoạn song song.

## Server block `api-v2` (FR-028)

- Route longest-prefix tới 3 Cloud Run `run.app` (xử lý chồng lấn `/me/*`, `/admin/*`).
- Route map single-source trong Nginx; smoke test đủ 3 service.
- TLS terminate tại Nginx; cert Let's Encrypt mở rộng gồm `api-v2` + renew tự động (FR-040).
- **SSE**: `proxy_buffering off` cho path SSE, timeout đọc lớn, heartbeat 15s,
  `ai_chat_chunk_deadline_seconds=30s`; `proxy_ssl_server_name on` + resolve động
  hostname `run.app` (không pin IP) (FR-036/FR-037).

## Xác thực Nginx → Cloud Run (FR-033/FR-039)

- **Cloud Run IAM** (`roles/run.invoker` cho service account của token helper;
  KHÔNG bật `allUsers`).
- Nginx gửi Google ID token ở `X-Serverless-Authorization` (token helper trên VPS
  sinh/cache/refresh, audience = URL `run.app` từng service); Cloud Run strip header
  này trước khi tới container → `Authorization` (Closy JWT) giữ nguyên cho app.
- SA key least-privilege + rotation; file quyền `0600`.

## DNS (FR-029)

- Bản ghi A/CNAME trên **Vercel** trỏ về VPS; **không** NS-delegate, **không** Cloudflare DNS.
- Cutover `api.closy` sang Cloud Run là **thao tác thủ công có runbook**
  (`manual-cutover-api-closy-to-cloudrun.md`, FR-041).

## Verify

- [ ] `api-v2` 200 qua Nginx↔Cloud Run; thiếu ID token bị Cloud Run từ chối (SC-013).
- [ ] `api.closy` 0 thay đổi routing (SC-015); 0 request `api-v2` qua Cloudflare (SC-016).
- [ ] SSE stream dài giữ kết nối, 0 lỗi buffering (SC-014).
