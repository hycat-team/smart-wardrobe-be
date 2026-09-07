# HTTP Behavior Contract: Tối Ưu Hóa Pre-Launch (019)

**Date**: 2026-09-06

Hợp đồng hành vi HTTP mà feature này cam kết với client và hệ thống vận hành. **Không có endpoint mới và không đổi schema JSON nào** — mọi hợp đồng API hiện hành (swagger) giữ nguyên 100%.

## 1. Response Status & Error Envelope

| Tình huống | Status | Body |
|---|---|---|
| Request body vượt ngưỡng tầng áp dụng (2 MB / 10 MB) | **413** | Error envelope chuẩn của hệ thống (`status`, `title`, `message` tiếng Việt, cấu trúc thống nhất qua `apperror`) — factory mới `NewRequestEntityTooLarge` |
| Request hợp lệ | không đổi so với hiện trạng | không đổi |

Ràng buộc: server KHÔNG xử lý use case khi đã từ chối body; lỗi 413 được ghi log có cấu trúc (method, path, content-length, tier).

## 2. Timeout Behavior

| Hành vi | Cam kết |
|---|---|
| Kết nối mở nhưng không gửi dữ liệu | bị đóng sau `read_timeout_seconds` (mặc định 30s) |
| Response thường | không được viết quá `write_timeout_seconds` (mặc định 60s) từ lúc nhận xong header |
| Kết nối idle giữa 2 request keep-alive | đóng sau `idle_timeout_seconds` (mặc định 120s) |
| SSE stream (`POST /api/v1/ai/chat/sessions/:contextID/messages/stream`) | KHÔNG bị cắt bởi write timeout toàn cục — handler gia hạn deadline theo từng chunk; stream chết tự kết thúc theo timeout nghiệp vụ hiện có |
| Request chậm vẫn xử lý dở khi timeout bắn | server được phép hủy; client nhận lỗi phù hợp (không treo vô hạn) |

## 3. Body Limit Tiers

| Tầng | Ngưỡng | Áp dụng |
|---|---|---|
| free | bỏ qua | registry config `server.body_limit.free_routes` — hiện TRỐNG; thêm qua config, không sửa code |
| elevated | 10 MB | `POST /api/v1/ai/*`; chat user↔brand (`/brands/:id/conversation*`, `/brand-portal/brands/:id/conversations*`); mọi `POST|PUT|PATCH /api/v1/brand-portal/**` và `/api/v1/admin/brands/**`; user-write brand (`/brands/claim`, `/:brandId/join-loyalty`, `/brand-items/:id/feedbacks`, `/brand-benefits/:id/redeem`) |
| default | 2 MB | mọi route còn lại |

Thứ tự đánh giá: free → elevated → default. Pattern đầy đủ và biên cụ thể được liệt kê chi tiết trong tasks khi implement (nguồn: research.md D2).

## 4. Compression Contract

| Tình huống | Cam kết |
|---|---|
| Client gửi `Accept-Encoding: gzip` + response JSON đủ lớn | response nén gzip, header `Content-Encoding: gzip` |
| Client không gửi Accept-Encoding | response như hiện trạng 100% |
| SSE stream | KHÔNG nén (nằm trong excluded paths) |
| Schema JSON | bất biến trước/sau nén (chỉ đổi transport) |

## 5. Subscription Overview Cache Semantics (qua `IUserSubscriptionContract`)

| Hành vi | Cam kết |
|---|---|
| Đọc overview (AI hot path) | trả từ cache nếu entry còn TTL; miss/lỗi cache → đọc trực tiếp như hiện trạng |
| Staleness | tối đa = TTL cấu hình (mặc định ≤ 60s); mutation trong module xóa cache ngay |
| Số đếm hạn mức ngày | KHÔNG bao giờ đi qua cache này — vẫn ghi/đọc trực tiếp từng request |
| Isolation | key theo userID — không rò rỉ giữa tài khoản |
| Failure | cache backend lỗi không làm fail request AI |

## 6. Test Suite Contract

| Lệnh | Phạm vi |
|---|---|
| `make test` / `make test-race` | toàn bộ unit test mới + cũ pass (không cần env ngoài) |
| `make test-integration` | chạy thêm integration test subscription (webhook, renewal) — tự skip khi thiếu `TEST_DATABASE_URL` |
| `make lint` / `make check` | không regression; generated files cập nhật |

## 7. Operational Signals (log có cấu trúc — clarify Q2)

- Từ chối body-limit: method, path, tier, content-length.
- Migration: bắt đầu/kết thúc run, từng migration áp dụng, kết quả lock.
- Cache overview: invalidation (userID, lý do mutation); miss/fallback chỉ ở mức debug.
- Không thêm hệ thống metrics mới.
