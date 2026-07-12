# Phase 05e & 05f Report - Brand Chat & Privacy Rules

Báo cáo kết quả triển khai kênh chat trực tiếp MVP giữa người dùng và brand (CS Chat) và cấu hình bảo mật thông tin (Privacy Rules).

## Files Changed

- **Database Migration**:
  - `migrations/20260628144413_create_brand_chat.sql`
- **Domain & Constants**:
  - `internal/shared/domain/entities/brand_entities.go` (Thêm entity `BrandConversation`, `BrandConversationMessage`)
  - `internal/shared/domain/entities/table_name.go` (Đăng ký TableName)
  - `internal/shared/domain/constants/brandchat/...` (Định nghĩa các hằng số trạng thái hội thoại và vai trò gửi)
- **Repositories**:
  - `internal/modules/brand/domain/repositories/interfaces.go`
  - `internal/modules/brand/infrastructure/persistence/chat_repo.go`
  - `internal/modules/brand/provider.go`
- **Application (Use Cases & DTO)**:
  - `internal/modules/brand/application/dto/brand.go` (SendChatMessageReq, BrandConversationRes, BrandConversationMessageRes)
  - `internal/modules/brand/application/usecase/brand_core_uc.go` (Triển khai luồng gửi/nhận tin nhắn, kiểm soát quyền hạn theo brand_id)
  - `internal/modules/brand/application/mapper/brand.go` (Ánh xạ DTO)
- **Presentation (API)**:
  - `internal/modules/brand/presentation/handler/brand_handler.go`
  - `internal/api/routes/brand/router.go`

## Migrations Added

- `20260628144413_create_brand_chat.sql`: Tạo các bảng `brand_conversations` ( unique index cặp `brand_id, user_id` ) và `brand_conversation_messages`.

## APIs Added/Changed

- `GET /api/v1/brands/:brandId/conversation` (Lấy cuộc hội thoại hiện tại - User)
- `POST /api/v1/brands/:brandId/conversation/messages` (Gửi tin nhắn lên brand - User, tự động mở hoặc tạo hội thoại)
- `GET /api/v1/brand-portal/brands/:brandId/conversations` (Lấy danh sách các cuộc hội thoại - Staff)
- `GET /api/v1/brand-portal/brands/:brandId/conversations/:conversationId/messages` (Lấy danh sách tin nhắn - Staff)
- `POST /api/v1/brand-portal/brands/:brandId/conversations/:conversationId/messages` (Gửi phản hồi - Staff)

## Tests Added/Updated

- `internal/modules/brand/application/usecase/brand_core_uc_chat_test.go`:
  - `TestSendUserMessage_AutoCreateAndReopen`: Kiểm tra gửi tin nhắn tự động khởi tạo hội thoại OPEN, và tự động reopen khi đang CLOSED.
  - `TestSendStaffMessage_Authorization`: Ràng buộc phân quyền, staff của brand khác không thể nhắn/xem tin nhắn thuộc brand ngoài phạm vi quản lý.

## Backward Compatibility Notes

- Độc lập hoàn toàn, không ảnh hưởng đến các luồng dữ liệu cũ.

## Manual Verification Steps

1. Chạy `make migration-up` để cập nhật database schema.
2. Kiểm tra Swagger tại `/swagger/index.html` tìm các API có Tag `Brand` và `Brand Portal` liên quan đến `conversation`.

## Known Limitations

- Chưa hỗ trợ real-time Websocket/SSE trong bản Chat CS MVP này (gửi nhận qua HTTP polling thông thường).
