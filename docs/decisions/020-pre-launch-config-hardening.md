# ADR 020 - Hardening Config Pre-launch: Pool Connection, Timeout, và Các Nợ Vận Hành Chấp Nhận

*   **Status**: Accepted
*   **Decision Date**: 2026-09-06
*   **Decision Maker**: Duck (review config pre-launch sau spec 019)

## 1. Context

Sau spec 019 (hardening HTTP server) và trước khi mở cửa đón người dùng thật, đã rà soát toàn bộ `configs/config.yaml` và `configs/config.production.yaml`. Audit phát hiện 5 điểm cần quyết định: pool connection Postgres sát trần, kết nối DB không TLS, email sản xuất chạy Gmail SMTP, tổng thời lượng request dev/prod lệch nhau cắt luồng AI chat, và Swagger vẫn được expose ở production. Đây là các quyết định vận hành, không đụng code.

## 2. Decision

1.  **`database.max_open_conns: 90 → 30` (cả dev và production)**. Postgres mặc định `max_connections = 100`; 90/instance chỉ chừa ~10 slot cho migration, psql, monitoring — và với 2 instance chạy rolling deploy là 180 kết nối vượt trần chắc chắn. 30/instance (x2 instance = 60) chừa đủ headroom. Revisit khi scale vượt 3 instance hoặc tăng `max_connections` của Postgres một cách chủ đích.

2.  **`database.ssl_mode: disable` — giữ nguyên, ghi nhận là rủi ro chấp nhận.** Hiện trạng toàn bộ hạ tầng (app, Postgres, Redis, Elasticsearch) chạy trên cùng 1 VPS trong docker network nội bộ — traffic DB không rời khỏi host, bật TLS đòi hỏi Postgres cấu hình cert (self-signed cùng host = phức tạp config, không tăng bảo mật thật). **Trigger reconsider (bắt buộc):** tách DB sang server/VPS riêng, hoặc mở port Postgres ra ngoài host → phải đổi sang `require` (lý tưởng `verify-full` với cert chuẩn).

3.  **Email: chuyển Gmail SMTP sang Resend (SMTP relay, thay vì Brevo).** Ban đầu ghi nhận Gmail SMTP là nợ chấp nhận (giới hạn ~500 email/ngày của app-password, rủi ro Google khóa key, không có domain reputation) và từng cân nhắc Brevo. Quyết định chính thức: chuyển sang **Resend** qua SMTP relay (`smtp.resend.com:587`) vì Resend cấp DKIM/SPF/DMARC riêng cho domain thương hiệu (`hycat.online`), deliverability cao và tích hợp đơn giản. Thay đổi: block `email:` trỏ `smtp.resend.com:587`, tách `smtp_user` (`resend` hoặc account email) khỏi `sender_email` (branded address), đổi env `EMAIL_APP_PASSWORD` → `EMAIL_SMTP_KEY` (Resend API key `re_...`), thêm `reply_to` optional (env trống = không set header). **Điều kiện để send production thực sự hoạt động:** owner verify domain `hycat.online` trên DNS Vercel (DKIM/SPF/DMARC records do Resend cung cấp). **Trigger reconsider:** vượt hạn mức gói free của Resend (3.000 email/tháng, 100 email/ngày) → nâng gói Pro hoặc cân nhắc AWS SES ($0.10/1.000).

4.  **`server.timeout_seconds`: dev `30 → 300`, production giữ `300`.**
    - Production 300s là con số hợp lý hiện tại: phải vượt tổng thời lượng stream chat AI hợp lệ (từng khoảng lặng giữa các chunk tối đa 180s theo `sse.ai_chat_chunk_deadline_seconds`, tổng stream thực tế 1–3 phút); đồng thời đủ ngắn để giải phóng connection kẹt. 300s ≈ 1.67× chunk deadline — biên an toàn mà không ôm connection vô hạn.
    - Dev 30s trước đây tự cắt luồng AI chat ở giây 30 (nhỏ hơn cả chunk deadline 180s) — dev không thể test được một stream hoàn chỉnh qua curl. Dev không có rủi ro internet nên 300s vô hại và đồng bộ với prod, giảm khác biệt môi trường.

5.  **Swagger ở production — nợ chấp nhận.** `/swagger` + `/api-docs` vẫn live ở production (chặn bằng `swagger_access_code` tĩnh) vì hệ thống còn khả năng lỗi và cần công cụ test trực tiếp. Access code là lớp obscurity, không phải biên bảo mật. **Trigger reconsider:** khi hệ thống ổn định post-launch → chỉ register swagger group khi `env != production` (1 dòng điều kiện trong `routes.NewEngine`), tách luôn khỏi config.

6.  **`rabbitmq.prefetch: 20 → 5` (cả dev và production).** `consumer_concurrency = 1` (1 consumer tuần tự) không tiêu thụ nổi buffer 20 message; prefetch 5 đủ tạo pipeline mượt mà giảm memory giữ message chưa ACK. Revisit khi tăng `consumer_concurrency`.

## 3. Nhật ký thay đổi config

| Key | Dev | Production | Ghi chú |
|---|---|---|---|
| `database.max_open_conns` | 90 → 30 | 90 → 30 | ADR §2.1 |
| `database.ssl_mode` | disable (thêm comment) | disable (thêm comment) | ADR §2.2 |
| `server.timeout_seconds` | 30 → 300 | giữ 300 | ADR §2.4 |
| `rabbitmq.prefetch` | 20 → 5 | 20 → 5 | ADR §2.6 |
| `email.*` | Gmail → Resend (`smtp.resend.com:587`), tách `smtp_user`, thêm `reply_to`, env `EMAIL_APP_PASSWORD` → `EMAIL_SMTP_KEY` | như dev | ADR §2.3 |
| BOM đầu `config.production.yaml` | — | đã xóa | dọn noise |

Giữ nguyên (đã cân nhắc và quyết định không đổi): `payos.return_url` trùng `cancel_url` (PayOS append query param để client phân biệt), `logger.log_to_file: false` (deploy thu stdout qua docker logs).

## 4. Consequences

- **Positive**: Pool connection an toàn trước trần Postgres và kịch bản multi-instance; dev/prod timeout đồng nhất — không còn "dev chạy được prod fail" (hoặc ngược lại) trên luồng AI chat; các nợ vận hành có trigger reconsider rõ ràng thay vì bị lãng quên.
- **Negative**: DB vẫn không TLS trong docker network (chấp nhận có chủ đích, xem §2.2); email chuyển Resend nhưng cần owner verify domain `hycat.online` trên DNS Vercel trước khi send branded thực sự đạt deliverability (§2.3); swagger vẫn lộ surface ở prod sau access code (§2.5).
- **Follow-up**: khi bất kỳ trigger nào ở §2.2/2.3/2.5 kích hoạt, mở ADR mới thay vì sửa trực tiếp ADR này.
