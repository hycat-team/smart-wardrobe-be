# Phase 06 Report - Brand Items & AI Vision Worker

Báo cáo kết quả tái cấu trúc module AI sang module `fashion`, thiết lập cơ chế gửi sự kiện bất đồng bộ qua RabbitMQ `fashion.event.analyze_item`, triển khai APIs & Use Cases quản lý sản phẩm thực/mẫu thử của Brand (Brand Items) và đánh giá mẫu thử kỹ thuật số (Digital Sample Feedback).

## Files Changed

- **Database Mapping & Configuration**:
  - `internal/shared/domain/entities/brand_entities.go` (Thêm các thực thể `BrandItem` và `DigitalSampleResponse`)
  - `internal/shared/domain/entities/table_name.go` (Đăng ký TableName mapping)
  - `internal/shared/infrastructure/messaging/rabbitmq_constants.go` (Thêm hằng số hàng đợi `QueueFashionAnalyzeItem` và routing key `RoutingKeyFashionAnalyzeItem`)
  - `internal/shared/infrastructure/messaging/rabbitmq_connection.go` (Khai báo và liên kết topology hàng đợi mới)
- **AI Fashion Module Integration**:
  - `internal/modules/fashion/contract/service.go` (Cập nhật signature `CreateFashionItem` chấp nhận userID, itemID và itemType)
  - `internal/modules/fashion/contract/service_impl.go` (Triển khai tự động publish sự kiện phân tích bất đồng bộ qua RabbitMQ)
  - `internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go` (Cập nhật hàm tải lên hàng loạt để gọi qua Fashion contract)
  - `internal/modules/fashion/presentation/worker/fashion_analyze_worker.go` (Tạo worker lắng nghe hàng đợi mới, phân tích hình ảnh qua AI Vision bất đồng bộ)
  - `internal/modules/fashion/application/usecase/worker/worker_uc.go` (Tái cấu trúc bộ xử lý nghiệp vụ AI worker hỗ trợ cả Wardrobe và Brand items)
- **Brand Domain & Application**:
  - `internal/modules/brand/domain/repositories/interfaces.go` (Khai báo repo interface cho Brand Item & Digital Sample Response)
  - `internal/modules/brand/infrastructure/persistence/brand_item_repo.go` (Triển khai repositories cụ thể bằng GORM)
  - `internal/modules/brand/application/dto/brand.go` (Thêm các DTO CreateBrandItemReq, UpdateBrandItemReq, BrandItemRes, SubmitSampleFeedbackReq, DigitalSampleResponseRes)
  - `internal/modules/brand/application/usecase/brand_core_uc.go` (Triển khai nghiệp vụ CRUD Brand Item của staff, xem feedback và cho phép user gửi feedback mẫu thử)
- **Presentation & Routing**:
  - `internal/modules/brand/presentation/handler/brand_handler.go` (Triển khai các gin handlers)
  - `internal/api/routes/brand/router.go` (Cấu hình endpoints cho Brand Items và Feedbacks)

## RabbitMQ Topology

- **Queue**: `fashion_analyze_item_queue`
- **Routing Key / Event**: `fashion.event.analyze_item`
- Tự động kích hoạt khi Wardrobe hoặc Brand tạo mới một `FashionItem`. Worker sử dụng GPT Vision phân tích bất đồng bộ và tự động cập nhật trạng thái của item về `ACTIVE` / `InWardrobe` sau khi xử lý thành công.

## APIs Added

- **Staff portal (`/brand-portal`)**:
  - `POST /api/v1/brand-portal/brands/:brandId/items` (Tạo sản phẩm hoặc mẫu thử của Brand, tự động trigger phân tích AI)
  - `GET /api/v1/brand-portal/brands/:brandId/items` (Lấy danh sách sản phẩm hoặc mẫu thử của Brand)
  - `PUT /api/v1/brand-portal/brands/:brandId/items/:itemId` (Cập nhật thông tin/trạng thái sản phẩm hoặc mẫu thử)
  - `GET /api/v1/brand-portal/brands/:brandId/items/:itemId/feedbacks` (Lấy phản hồi/đóng góp ý kiến mẫu thử kỹ thuật số từ người dùng)
- **User-facing (`/brands`)**:
  - `GET /api/v1/brands/:brandId/items` (Lấy danh sách sản phẩm hoặc mẫu thử hoạt động của Brand)
  - `POST /api/v1/brands/:brandId/items/:itemId/feedbacks` (Gửi phản hồi, đánh giá mẫu thử kỹ thuật số)

## Tests Added

- `internal/modules/brand/application/usecase/brand_core_uc_item_test.go`:
  - `TestBrandItemAndFeedbackFlow`: Xác minh luồng staff tạo BrandItem, kích hoạt AI vision liên kết qua contract, cập nhật trạng thái lên `ACTIVE`, user xem danh sách và gửi feedback phản hồi đánh giá mẫu thử thành công.

## Manual Verification Steps

1. Chạy `make generate` để cập nhật Wire DI và tài liệu Swagger.
2. Kiểm tra Swagger tại `/swagger/index.html` cho các endpoint Brand Items và Feedbacks.
