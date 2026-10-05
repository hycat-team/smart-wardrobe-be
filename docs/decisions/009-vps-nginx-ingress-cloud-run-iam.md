# ADR 009 - Ingress bằng VPS Nginx, Cloud Run IAM, bỏ Cloudflare khỏi đường API

*   **Status**: Accepted
*   **Decision Date**: 2026-09-15
*   **Decision Maker**: Duck
*   **Supersedes**: ADR 007 §2.1 (Cloudflare zone + Worker), §2.2 (cutover SaaS — hoãn), §2.6 (origin-auth `X-Gateway-Secret`), §2.7 (SSE qua Worker), §2.8 (route table single-source với Worker).
*   **Superseded by**: ADR 010 (Cloud Run compute)

## 1. Context

Plan cũ (ADR 007) giả định `hycat.online` được NS-delegate sang Cloudflare và Cloudflare Worker đứng
trước API (`api-v2`). Thực tế:

- DNS `hycat.online` **đang** do **Vercel** quản lý và **phải giữ nguyên**; không chuyển nameserver.
- **VPS hiện tại đã được đầu tư kinh phí** và đang chạy Nginx reverse proxy với đầy đủ TLS (Let's
  Encrypt), HSTS, security headers, WebSocket, `X-Forwarded-*`, `/nginx-health`, timeout — đang proxy
  `api.closy.hycat.online` → `http://backend:8080` (monolith Go).
- Mục tiêu là tách backend thành **3 Cloud Run service độc lập** và phục vụ chúng dưới
  `api-v2.closy.hycat.online`, **song song** với API cũ; chưa cần cutover API cũ.

Bối cảnh này khiến việc đưa Cloudflare vào đường API là **không cần thiết** và làm phức tạp DNS/path.
Cloudflare Pro vẫn là hạng mục funded (dùng cho mục đích khác nếu muốn), nhưng không nằm trong API path.

## 2. Decision

1.  **Ingress = VPS Nginx hiện hữu** (không dựng reverse proxy mới):
    - `api.closy.hycat.online` → `http://backend:8080` (monolith) — **giữ nguyên**, không đổi routing.
    - `api-v2.closy.hycat.online` → `https://<service>.run.app` (3 Cloud Run), route longest-prefix
      (xử lý chồng lấn `/me/*`, `/admin/*`). Route map đặt trong Nginx (single source).
2.  **DNS = Vercel**: thêm bản ghi A/CNAME cho `api-v2` trỏ VPS. **Không** NS-delegate, **không**
    chuyển nameserver, **không** Cloudflare DNS/Worker, **không** GCP Load Balancer / API Gateway.
3.  **Cloud Run IAM**: không bật `allUsers`; grant `roles/run.invoker` cho service account của token
    helper. Nginx gửi Google ID token ở **`X-Serverless-Authorization: Bearer <token>`** (Cloud Run
    strip header này trước khi tới container) → giữ nguyên `Authorization: Bearer <Closy JWT>` cho app.
    Audience = URL `run.app` của service (custom audience nếu cần; custom domain không dùng làm `aud`).
4.  **Token helper trên VPS**: sinh/cache/refresh ID token cho service account; chọn **service account
    key** (least-privilege, file 0600, rotation) vì đơn giản và chạy được ngoài GCP; **WIF** ghi nhận
    làm hướng nâng cấp (cần IdP/OIDC issuer, phức tạp hơn kiến trúc hiện tại).
5.  **TLS**: Nginx terminate cho cả 2 hostname; **mở rộng cert Let's Encrypt** (`certbot --expand`
    /`--cert-name`) để gồm `api-v2`; giữ cơ chế renew hiện có.
6.  **Song song; cutover `api.closy` là thao tác THỦ CÔNG có runbook**: `api-v2` là deployment mới,
    chạy song song; `api.closy` vẫn monolith. Việc đưa `api.closy` sang Cloud Run do **owner thực hiện
    thủ công** theo runbook (đổi Nginx/DNS + đồng bộ dữ liệu), timing do owner quyết định dựa trên độ
    ổn định của `api-v2`; MUST có runbook cutover + **rollback về monolith** và drill (FR-041). Không
    tự động hoá cutover.
7.  **SSE** đi qua Nginx → Cloud Run: tắt response buffering cho path SSE, timeout đọc đủ lớn,
    heartbeat ~15s, `ai_chat_chunk_deadline_seconds` = 30s thống nhất mọi môi trường.

## 3. Consequences

- **Positive**: dùng lại hạ tầng đã đầu tư (VPS/Nginx/TLS/hardening) → ít thành phần mới; `api.closy`
  không bị ảnh hưởng; DNS giữ ở Vercel, không phụ thuộc Cloudflare cho API; auth mạnh nhờ Cloud Run IAM.
- **Negative**: **VPS là single ingress** → VPS/Nginx down ảnh hưởng cả 2 hostname (đã chấp nhận; cần
  health/restart/runbook). Token helper dùng SA key = secret dài hạn → phải least-privilege + rotation.
  Nginx phải cấu hình `proxy_ssl_server_name`, resolve động `run.app`, và xử lý SSE buffering/timeout.
- **Trigger reconsider**: nếu cần WAF/rate-limit toàn cục hoặc tách VPS khỏi đường tới hạn → cân nhắc
  Cloudflare hoặc GCP LB (ngoài phạm vi hiện tại).

## 4. Verify list (phải xác nhận khi implement)

- `run.app` resolve động + SNI đúng (`proxy_ssl_server_name on`), không pin IP.
- Token helper: audience đúng, token cache/refresh trước `exp`, retry/backoff khi lỗi; IAM reject khi
  thiếu token (smoke test).
- `Authorization` của app tới container nguyên vẹn (kiểm bằng log/echo header).
- Cert Let's Encrypt đã gồm `api-v2` + renew thành công.
- SSE qua Nginx không bị buffer/timeout; stream dài thật pass.
- `api.closy` không đổi hành vi cho tới khi owner cutover.
- Runbook cutover thủ công tồn tại + đã drill; rollback về monolith hoạt động.

## 5. Không thay đổi

- Không đổi đường dẫn API `/api/v1/...`; không đổi luồng nghiệp vụ; không đổi compute/data/queue
  (giữ theo ADR 007 §2.3–2.5) và repatriation/funding clock.
