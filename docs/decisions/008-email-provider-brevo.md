# ADR 008 - Email Provider: Brevo thay Resend

*   **Status**: Accepted
*   **Decision Date**: 2026-09-15
*   **Decision Maker**: Duck
*   **Supersedes**: ADR 006 §2.3 (Email: chuyển Gmail SMTP sang Resend)

## 1. Context

ADR 006 §2.3 chốt chuyển email sản xuất sang **Resend** (SMTP relay `smtp.resend.com:587`), với
trigger reconsider khi vượt hạn mức free (3.000 email/tháng, 100 email/ngày) → nâng Pro hoặc AWS SES.

Sau khi cân nhắc lại hạn mức gửi của từng nhà cung cấp, owner quyết định dùng **Brevo** vì cho
hạn mức free cao hơn đáng kể so với Resend, phù hợp hơn cho giai đoạn hiện tại. Config đã được
đổi thủ công sang Brevo (`configs/config.yaml`, `configs/config.production.yaml`:
`host: smtp-relay.brevo.com`).

## 2. Decision

1.  **Email provider sản xuất là Brevo** qua SMTP relay `smtp-relay.brevo.com`.
2.  Giữ nguyên các thay đổi cấu trúc email mà ADR 006 §2.3 đã chốt: tách `smtp_user` khỏi
    `sender_email`, giữ `reply_to` optional, env key `EMAIL_SMTP_KEY` (thay cho
    `EMAIL_APP_PASSWORD`).
3.  `configs/config.yaml` và `configs/config.production.yaml` giữ `host: smtp-relay.brevo.com`
    là cấu hình chuẩn cho cả dev và production.
4.  Điều kiện để gửi branded đạt deliverability vẫn như ADR 006: owner verify domain
    `hycat.online` (SPF/DKIM/DMARC do Brevo cấp) trên DNS Vercel.

## 3. Consequences

- **Positive**: Hạn mức gửi free cao hơn Resend; không phát sinh thêm chi phí ở giai đoạn hiện tại;
  cấu trúc config email không đổi (chỉ đổi nhà cung cấp + host).
- **Negative**: Khác biệt so với ghi chép cũ tại ADR 006 §2.3 → đã đánh dấu supersede để tránh nhầm.
- **Trigger reconsider**: vượt hạn mức free của Brevo → cân nhắc nâng gói hoặc chuyển nhà cung cấp
  khác (AWS SES/Resend); đây là quyết định vận hành, không đụng code.

## 4. Không thay đổi

- Không đổi interface gửi email trong code (SMTP relay), không đổi tên env, không đổi luồng nghiệp vụ.
