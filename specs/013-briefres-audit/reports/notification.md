# Báo cáo audit BriefRes — Module Notification

## Tổng quan

Module `notification` là module **service-only** (chỉ gửi email nền), **không có HTTP handler**. Quy trình xử lý:

1. Các usecase khác (identity, subscription) gọi `INotificationContract.Send(...)` để ghi bản ghi outbox.
2. `OutboxListener` đọc outbox và publish message lên topic `notification.email` (queue `notification_email_queue`).
3. `EmailWorker` consume message, unmarshal payload, và gọi `EmailUseCase.SendEmail(...)` để gửi qua SMTP (`SmtpSender`).

Module **không có thư mục `application/dto/`** và **không có thư mục `presentation/handler/`** — không tồn tại response DTO hay endpoint API nào thuộc module này.

## Danh sách struct

Chỉ có **1 struct payload nội bộ** trong `presentation/worker/email_worker.go` (không phải response DTO):

| Struct | Vị trí | Loại |
|---|---|---|
| `payload` (anonymous struct) | `internal/modules/notification/presentation/worker/email_worker.go:44-49` | Payload consume message từ queue, 4 fields: `to`, `subject`, `body`, `channel` |

Ngoài ra, các type trong `presentation/contract/notification_contract.go` (`ChannelEnum`, `TemplateEnum`, interface `INotificationContract`) là **input contract nội bộ**, không phải response DTO.

## Overdelivery

**Không có.** Không tồn tại response struct nào từ HTTP API. Struct payload nội bộ chỉ có 4 fields, tất cả đều được dùng trực tiếp khi gửi email.

## Underdelivery

**Không có.** Module không trả response chứa UUID FK. Payload nội bộ chứa `to` (email người nhận) đầy đủ, không có FK cần related info.

## Đề xuất BriefRes

Không có đề xuất. Module service-only không cần `<Entity>BriefRes`.

## Ưu tiên

- **Không áp dụng** (không có vấn đề): Module không có response DTO, không có HTTP handler nên không nằm trong phạm vi audit BriefRes. Không cần thay đổi gì.

## API bị ảnh hưởng

**Không có API nào.** Module `notification` không đăng ký route HTTP nào (`internal/di/wire.go` chỉ đưa `EmailWorker` vào bootstrap, không có handler).