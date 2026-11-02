# Transactional Email — Danh mục template

Nguồn sự thật cho câu hỏi "hệ thống gửi những email nào, khi nào".
Mọi email đi qua **một** đường: use case ghi `shared.outbox_records` (topic
`notification.email`) trong cùng transaction nghiệp vụ → outbox worker publish
→ RabbitMQ → `email_worker` → SMTP. Nội dung được render **tại thời điểm
enqueue**, nên sửa template không làm đổi các email đã nằm trong outbox.

Code: `services/account/internal/modules/notification/`.

## Danh mục

| Template | Subject | Kích hoạt | Người nhận |
|---|---|---|---|
| `otp` | `[Closy] Mã xác thực OTP` | Đăng ký tài khoản; gửi lại OTP đăng ký (có cooldown) | Email đang đăng ký |
| `forgot_password_otp` | `[Closy] Đặt lại mật khẩu` | Yêu cầu quên mật khẩu; gửi lại OTP | Email yêu cầu |
| `topup_success` | `[Closy] Nạp tiền vào ví thành công` | PayOS webhook, `wallet_topup`, thanh toán thành công | Chủ ví |
| `payment_success` | `[Closy] Xác nhận thanh toán thành công` | PayOS webhook, `direct_purchase` | Người mua |
| `wallet_purchase_success` | `[Closy] Mua gói thành công qua ví nội bộ` | `POST /subscriptions/me/purchase-with-wallet` | Người mua |
| `renewal_success` | `[Closy] Gia hạn gói thành công` | Worker `subscription_renewal`: hết hạn + auto-renew ON + ví đủ | Chủ gói |
| `subscription_expired` | `[Closy] Gói dịch vụ của bạn đã hết hạn` | Worker: gói hết hạn → hạ cấp | Chủ gói |
| `subscription_expiring` | `[Closy] Gói dịch vụ của bạn sắp hết hạn` | Worker cron 00:00: còn ≤ N ngày + auto-renew OFF | Chủ gói |
| `wallet_warning` | `[Closy] Cảnh báo số dư ví không đủ` | Worker cron 00:00: còn ≤ N ngày + auto-renew ON + ví < giá | Chủ gói |
| `admin_grant` | `[Closy] Gói của bạn đã được cấp thành công` | `POST /admin/subscription-grants` (mỗi item một email) | Từng user được cấp |
| `campaign_grant` | `[Closy] Chúc Mừng Bạn Nhận Được Quà Tặng!` | Campaign claim khi user-created, hoặc catch-up sweep | User đủ điều kiện |

`N` = `subscription.wallet_balance_warning_days` (mặc định 3).

## Quy tắc

- **Subject là bắt buộc.** `subjectByTemplate` là map, không có giá trị mặc định;
  template thiếu subject làm `Send` trả lỗi ngay lúc enqueue.
  `TestEveryTemplateHasASubject` đi qua `template.All()` nên thiếu subject là đỏ CI.
- **Placeholder phải khớp params của call site.**
  `TestEveryTemplateRendersItsCallSiteParameters` render đúng bộ params mỗi
  use case gửi; lệch key sẽ ra `<no value>` và fail.
- **Gói khôi phục phải lấy từ DB.** Tài khoản có thể giữ gói trọn đời bên dưới
  gói hạn hạn; khi hết hạn, nó quay về gói trọn đời chứ không phải Free. Email
  dùng `RestoredPlanName` (`HasLifetimeFallback` với campaign), không hardcode.
- **Tiền và ngày** đi qua `money.FormatVND` và `timeutils.FormatDateVN`. Không
  truyền `float64` vào template: Go render `1000000` thành `1e+06`.
- **Không in mã nội bộ** (`downgraded_auto_renew_disabled`) ra nội dung email.

## Chống trùng

- Mỗi lần gửi mang `outboxId` của row `shared.outbox_records`. Worker kiểm tra
  Redis `email:delivered:<outboxId>` (TTL 24h) trước khi gửi và ghi key **sau khi**
  SMTP nhận. Redis lỗi thì **fail-open**: vẫn gửi, chỉ log warn.
- `renewal_success` idempotent theo `subscription_renewal_attempts`.
- `subscription_expired` chỉ một lần: sau hạ cấp, row không còn khớp query.
- Webhook idempotent theo inbox + trạng thái `deposit_transactions`.
- `subscription_expiring` / `wallet_warning` lặp **mỗi ngày** trong cửa sổ N ngày,
  và chỉ chạy ở cron 00:00 **dưới advisory lock** — nhiều instance không nhân đôi.

## Retry khi SMTP lỗi

- `smtp_sender` phân loại lỗi: 4xx / timeout / dial / TLS → **transient**;
  5xx / địa chỉ không hợp lệ → **permanent** (`notification/domain/errors`).
- Permanent → `event.PermanentError` → DLQ ngay.
- Transient → publish bản copy sang routing key `notification.retry.email` với
  `retryCount+1`, rồi **ACK** message đang giữ (publish trước, ack sau).
- `notification_email_retry_queue` (classic) giữ message **5 phút** rồi dead-letter
  về `notification.email`; tối đa **4 lần** (~20 phút) thì vào DLQ.
- Queue này **không có consumer** và **không dùng `x-queue-type=quorum`**: retry
  của quorum là tức thì, không có delay. Nó được thêm dưới dạng queue mới nên
  không phải migrate queue nào đang chạy.
- Routing key cố tình là `notification.retry.email`, không phải
  `notification.email.retry`: queue chính bind `notification.email.#` và `#` khớp
  nhiều word, nên key sau sẽ vào cả hai queue.

## Những trường hợp KHÔNG gửi email (tính đến hiện tại)

1. Admin huỷ gói (`POST /admin/subscriptions/:userId/cancel`).
2. Gia hạn lỗi kỹ thuật (plan inactive/missing, ví không tồn tại, lỗi DB) — chỉ log.
3. Thanh toán / nạp ví thất bại.
4. Hoàn tiền.
5. Đổi mật khẩu / đổi email thành công.

Đây là danh sách cần review khi thay đổi kỳ vọng sản phẩm; không phải bug đã biết.
