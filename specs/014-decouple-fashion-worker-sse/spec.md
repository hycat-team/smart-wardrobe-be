# Đặc tả Tính năng (Feature Specification): Refactor Phân tách Luồng Sự kiện Phân tích Trang phục và Quản lý SSE (Decouple Fashion Worker & SSE Events)

**Feature Branch**: `014-decouple-fashion-worker-sse`

**Created**: 2026-08-26

**Status**: Draft

**Input**: User description: "Lên specs để thực hiện việc refactor phân tách Fashion AI Worker và luồng SSE theo mô hình Event-Driven, đánh giá nhiều phương diện và vị trí bị ảnh hưởng, kể cả phía client"

---

## Clarifications

### Session 2026-08-26
- Q: Khi Consumer (Wardrobe/Brand) nhận lại sự kiện trùng lặp từ RabbitMQ (do redelivery) cho món đồ đã hoàn tất, Consumer nên xử lý thế nào? → A: Bỏ qua cập nhật CSDL nếu status đã xong, chỉ phát lại SSE nếu channel còn mở để đảm bảo Idempotency cho DB nhưng vẫn hỗ trợ client vừa reconnect.
- Q: Trường `data` trong sự kiện SSE có bắt buộc phải nhúng `FashionItemBriefRes` đầy đủ không? → A: Có, bắt buộc nhúng đầy đủ (ảnh, màu hex, style, category) để client không phải gọi API refetch lại toàn bộ tủ đồ.

---

## Bối cảnh & Mục tiêu Refactor

Hiện tại, `FashionWorkerUseCase` (trong module `fashion`) đang hoạt động như một bộ điều phối tập trung (Orchestrator): sau khi chạy AI phân tích ảnh, worker trực tiếp gọi ngược sang `WardrobeContract` / `BrandContract` để cập nhật trạng thái (`UpdateItemStatus`), truy vấn lại dữ liệu hoàn chỉnh (`GetItemsByIDs` - dẫn tới việc Wardrobe lại gọi ngược về Fashion để ghép metadata tạo thành **Runtime Call Cycle**), tự ánh xạ DTO tóm tắt và trực tiếp quản lý phát tín hiệu SSE cho Client của tủ đồ / thương hiệu.

Mục tiêu của đợt refactor này:
1. **Chuyển dịch sang mô hình Event Choreography**: Fashion AI Worker sau khi phân tích ảnh xong chỉ ghi nhận vào CSDL của module mình và phát ra Domain Event qua RabbitMQ.
2. **Triệt tiêu Runtime Call Cycle & Coupling**: Cắt đứt hoàn toàn việc Fashion Worker phụ thuộc vào `WardrobeContract`, `BrandContract` và `sseManager`.
3. **Phân định ranh giới sở hữu nghiệp vụ**: Module `wardrobe` và `brand` tự quản lý vòng đời trạng thái của trang phục/sản phẩm và tự chịu trách nhiệm kết nối SSE thông báo tiến trình cho Client của chính mình.
4. **Đảm bảo tính tương thích và bảo vệ trải nghiệm Client**: Giữ nguyên hoặc nâng cấp trải nghiệm realtime SSE ở phía Frontend/Client, không làm gián đoạn luồng xử lý tải lên hàng loạt của người dùng cá nhân lẫn đối tác nhãn hàng.

---

## User Scenarios & Testing *(Bắt buộc)*

### User Story 1 - Người dùng tải lên hàng loạt ảnh trang phục cá nhân và theo dõi tiến trình realtime (Priority: P1)

Là một người dùng cá nhân (B2C), tôi muốn tải lên nhiều ảnh trang phục vào tủ đồ và nhận được phản hồi cập nhật trạng thái tức thời từng món đồ qua kết nối SSE mà không bị gián đoạn hay sai lệch dữ liệu.

**Tại sao ưu tiên P1**: Đây là luồng nghiệp vụ cốt lõi của tính năng thêm trang phục thông minh cho người dùng B2C, sử dụng trực tiếp AI vision và SSE.

**Độc lập kiểm thử**: Có thể kiểm thử độc lập bằng cách gửi yêu cầu batch upload vào tủ đồ cá nhân, mở kết nối lắng nghe SSE `/api/v1/wardrobe-items/tasks/{taskId}/sse` và kiểm tra xem các sự kiện cập nhật tiến trình (kèm dữ liệu tóm tắt của trang phục) có được đẩy về liên tục và chính xác hay không.

**Acceptance Scenarios**:
1. **Given** Người dùng tải lên 3 ảnh trang phục vào tủ đồ cá nhân với một `taskId`, **When** Hệ thống AI phân tích xong từng ảnh thành công, **Then** Client nhận được event SSE `message` chứa trạng thái `completed`, tiến trình `index/total` tương ứng, và dữ liệu `data` chứa thông tin tóm tắt trang phục (`WardrobeItemBriefRes`) với đầy đủ metadata nhận diện (`FashionItemBriefRes`).
2. **Given** Người dùng tải lên ảnh trang phục nhưng AI không nhận diện được (độ tin cậy thấp), **When** Xử lý hoàn tất với trạng thái cần người dùng xác nhận, **Then** Client nhận được event SSE với trạng thái `needs_review` để hiển thị màn hình điều chỉnh.
3. **Given** Có lỗi xảy ra trong quá trình phân tích ảnh (ví dụ ảnh lỗi/mất kết nối AI), **When** Worker ghi nhận thất bại, **Then** Client nhận được event SSE với trạng thái `failed` kèm thông báo lỗi dễ hiểu.

---

### User Story 2 - Nhân viên nhãn hàng (Brand Staff) nhập danh sách sản phẩm mẫu/mẫu thử (Priority: P2)

Là một nhân viên thương hiệu B2B, tôi muốn tải lên danh mục sản phẩm của nhãn hàng và theo dõi tiến trình phân tích tự động qua SSE trên Brand Portal để chuẩn bị đưa sản phẩm vào hệ thống.

**Tại sao ưu tiên P2**: Đảm bảo luồng vận hành B2B trên Brand Portal diễn ra thông suốt và độc lập hoàn toàn với tủ đồ cá nhân.

**Độc lập kiểm thử**: Có thể kiểm thử độc lập bằng cách gọi API tạo sản phẩm thương hiệu, mở kết nối SSE `/api/v1/brand-portal/brands/{brandId}/items/tasks/{taskId}/sse` và kiểm tra sự kiện trả về.

**Acceptance Scenarios**:
1. **Given** Brand Staff tạo sản phẩm thương hiệu cần xử lý AI, **When** AI hoàn tất phân tích, **Then** Brand Consumer nhận sự kiện, chuyển sản phẩm sang trạng thái `Active`, và phát SSE `message` chứa thông tin tóm tắt `BrandItemBriefRes` tới Brand Portal.
2. **Given** Xử lý thất bại, **When** Hệ thống ghi nhận lỗi, **Then** Brand Consumer chuyển sản phẩm sang `Draft` hoặc `Failed` và phát SSE thông báo lỗi cho Brand Portal.

---

### User Story 3 - Phục hồi và xử lý lỗi hàng đợi bất đồng bộ (Resilience & Event Redelivery) (Priority: P3)

Là một kỹ sư vận hành hệ thống, tôi muốn khi có sự cố tạm thời ở một Consumer (ví dụ: DB của Wardrobe tạm khóa), các sự kiện phân tích AI không bị mất và được thử lại mà không ảnh hưởng tới tiến trình chung của Fashion Worker.

**Tại sao ưu tiên P3**: Tăng cường tính ổn định và khả năng tự phục hồi của hệ thống theo chuẩn Event-Driven.

**Độc lập kiểm thử**: Mô phỏng lỗi ở Consumer của Wardrobe, đảm bảo Fashion Worker vẫn hoàn thành phân tích, ghi nhận Outbox event thành công và RabbitMQ giữ lại message cho đến khi Consumer phục hồi.

**Acceptance Scenarios**:
1. **Given** Worker phân tích ảnh xong và publish event `fashion.event.analyzed`, **When** Consumer của Wardrobe gặp lỗi tạm thời khi cập nhật CSDL, **Then** Message được nack/requeue hoặc đưa vào Dead Letter Queue theo chính sách retry mà không gây crash Worker.

---

### Các trường hợp biên (Edge Cases)

- **Mất kết nối Client trước khi AI xử lý xong**: Client ngắt kết nối SSE giữa chừng $\rightarrow$ Consumer của Wardrobe/Brand vẫn phải hoàn tất việc cập nhật trạng thái trong CSDL, chỉ bỏ qua việc ghi vào SSE stream khi channel đã đóng.
- **Thứ tự hoàn tất không theo thứ tự gửi**: AI có thể hoàn tất item 2 trước item 1 $\rightarrow$ Consumer xử lý độc lập từng item dựa trên `itemId` và `taskId`, Client cập nhật UI dựa trên `itemId` và số lượng `index/total`.
- **Sự kiện trùng lặp (Idempotency)**: Consumer nhận lại cùng 1 event phân tích cho một `itemId` $\rightarrow$ Nếu item trong CSDL đã ở trạng thái hoàn thành (`in_wardrobe`/`active`), Consumer bỏ qua việc UPDATE CSDL để tránh ghi đè thừa thãi, nhưng vẫn phát lại sự kiện SSE nếu Client đang mở kết nối channel để hỗ trợ các trường hợp Client vừa kết nối lại.

---

## Yêu cầu Nghiệp vụ & Kỹ thuật (Requirements)

### Yêu cầu Chức năng (Functional Requirements)

- **FR-001**: `FashionWorkerUseCase` PHẢI là một worker thuần túy: chỉ thực hiện tác vụ AI/Vision, cập nhật bảng `fashion_items`, và phát sinh sự kiện Domain Event (`fashion.event.analyzed` hoặc `fashion.event.failed`).
- **FR-002**: `FashionWorkerUseCase` KHÔNG ĐƯỢC trực tiếp import hoặc gọi `WardrobeContract`, `BrandContract` cũng như `sseManager`.
- **FR-003**: Hệ thống PHẢI định nghĩa cấu trúc dữ liệu sự kiện phân tích chuẩn (`FashionAnalyzedEventPayload`) chứa đầy đủ: `TaskID`, `ItemID`, `ItemType`, `FashionItemID`, `Status`, `Index`, `Total`, `FashionBrief` (`FashionItemBriefRes`), và `Error`.
- **FR-004**: Module `wardrobe` PHẢI có một Event Consumer (`WardrobeAnalyzeResultConsumer`) lắng nghe sự kiện `fashion.event.analyzed` thuộc loại `Wardrobe` để tự cập nhật trạng thái của `wardrobe_items` và tự phát tín hiệu SSE cho Client của tủ đồ cá nhân.
- **FR-005**: Module `brand` PHẢI có một Event Consumer (`BrandAnalyzeResultConsumer`) lắng nghe sự kiện `fashion.event.analyzed` thuộc loại `Brand` để tự cập nhật trạng thái của `brand_items` và tự phát tín hiệu SSE cho Brand Portal.
- **FR-006**: Sự kiện SSE phát tới Client PHẢI giữ nguyên cấu trúc JSON payload (`itemId`, `status`, `total`, `index`, `data`, `error`) để đảm bảo tính tương thích ngược hoàn toàn (Zero Breaking Change) cho các ứng dụng Client (Frontend Web / Mobile).
- **FR-007**: Trường `data` trong sự kiện SSE gửi cho Client PHẢI chứa thông tin tóm tắt có nhúng `fashionItem` hợp lệ (không bị `nil`) thay vì bị bỏ sót như trước đây.
- **FR-008**: Consumer PHẢI đảm bảo tính Idempotency: khi nhận trùng lặp sự kiện của item đã được xử lý xong, Consumer chỉ phát SSE nếu stream đang mở và không thực hiện câu lệnh UPDATE CSDL dư thừa.

---

### Đánh giá các phương diện bị ảnh hưởng

#### 1. Phương diện Phía Client (Frontend Web / Mobile App)
- **Hợp đồng API SSE (SSE Contract)**:
  - Endpoint không đổi: `GET /api/v1/wardrobe-items/tasks/{taskId}/sse` và `GET /api/v1/brand-portal/brands/{brandId}/items/tasks/{taskId}/sse`.
  - Format event giữ nguyên: `event: message`, dữ liệu JSON bọc trong `SSEEventPayload`.
- **Tác động tích cực tới Client**:
  - Dữ liệu `data` trong event SSE giờ đây sẽ có đầy đủ trường `fashionItem` (ảnh, màu sắc, phong cách, danh mục), giúp Client có thể render trực tiếp thẻ trang phục mới vừa hoàn thành mà không cần phải gọi thêm một HTTP request GET toàn bộ tủ đồ để lấy lại dữ liệu.
- **Tác động tiêu cực / Breaking Changes**: **Không có (Zero Breaking Changes)**. Client không cần sửa đổi mã nguồn lắng nghe SSE.

#### 2. Phương diện Backend & Ranh giới Module
- **Module `fashion`**:
  - Tinh gọn tối đa: Xóa bỏ các dependency của `wardrobeContract`, `brandContract`, `sseManager` khỏi `FashionWorkerUseCase` và Wire DI.
  - Loại bỏ các hàm map DTO mượn tạm `mapWardrobeItemToBrief` và `mapBrandItemToBrief` khỏi worker của fashion.
- **Module `wardrobe` & `brand`**:
  - Nhận quyền sở hữu hoàn toàn luồng xử lý kết quả AI và thông báo realtime cho người dùng của mình.
  - Bổ sung presentation consumer adapter để consume message từ RabbitMQ.

#### 3. Phương diện Hiệu năng & Hàng đợi (RabbitMQ & Database)
- **Loại bỏ Round-trip Query**: Triệt tiêu hoàn toàn vòng lặp truy vấn `Fashion Worker -> Wardrobe -> Fashion DB`. Dữ liệu AI metadata đã được đóng gói ngay trong message RabbitMQ nên Consumer chỉ cần 1 câu lệnh UPDATE trạng thái trong DB cục bộ.
- **Khả năng mở rộng**: Trong tương lai nếu có thêm module mới cần phân tích AI (ví dụ: module mạng xã hội chia sẻ outfit, module sàn thương mại điện tử), chỉ cần bind thêm Queue vào Topic mà không cần chạm vào mã nguồn của Module Fashion.

---

### Thực thể & Cấu trúc Dữ liệu Chính (Key Entities)

- **`FashionAnalyzedEventPayload`**: Payload sự kiện phát ra từ Fashion Worker lên Message Broker.
  - `TaskId` (string): Mã tiến trình tải lên hàng loạt.
  - `ItemId` (UUID): ID của Wardrobe Item hoặc Brand Item.
  - `ItemType` (enum): Phân loại đối tượng (`wardrobe` | `brand`).
  - `FashionItemId` (UUID): ID của bản ghi metadata trong bảng `fashion_items`.
  - `Status` (enum): `completed` | `needs_review` | `failed`.
  - `Total` (int): Tổng số lượng item trong task.
  - `Index` (int): Thứ tự của item hiện tại.
  - `FashionBrief` (object): Thông tin tóm tắt của fashion item (`id`, `imageUrl`, `color`, `colorHex`, `style`, `category`).
  - `Error` (string): Chi tiết lỗi nếu có.

- **`SSEEventPayload`**: Payload JSON chuẩn phát qua kết nối SSE cho Client.
  - `itemId` (UUID): ID trang phục/sản phẩm.
  - `status` (string): Trạng thái xử lý.
  - `total` (int), `index` (int): Chỉ số tiến độ.
  - `data` (object): `WardrobeItemBriefRes` hoặc `BrandItemBriefRes`.
  - `error` (string, optional): Thông báo lỗi thân thiện.

---

## Tiêu chí Thành công (Success Criteria *(Bắt buộc)*)

### Kết quả Đo lường được (Measurable Outcomes)

- **SC-001**: 100% các phụ thuộc trực tiếp từ `FashionWorkerUseCase` sang `WardrobeContract`, `BrandContract` và `sseManager` được dọn dẹp hoàn toàn (kiểm tra bằng dependency graph / Wire DI).
- **SC-002**: Triệt tiêu hoàn toàn hiện tượng Runtime Call Cycle giữa `fashion` và `wardrobe/brand` khi xử lý tác vụ nền.
- **SC-003**: 100% các sự kiện SSE phát về Client khi upload thành công đều chứa đầy đủ thông tin `fashionItem` hợp lệ (không bị `nil` hay thiếu trường màu/ảnh).
- **SC-004**: Thời gian phản hồi và hoàn tất cập nhật trạng thái item sau khi AI xử lý xong giảm tối thiểu 30% nhờ cắt giảm các truy vấn SQL đọc-sau-ghi thừa thãi giữa các module.
- **SC-005**: Toàn bộ unit test và integration test của luồng batch upload tủ đồ và sản phẩm brand chạy thành công 100%.

---

## Giả định (Assumptions)

- Hệ thống RabbitMQ và Outbox Pattern trong dự án đã hoạt động ổn định và sẵn sàng cho việc đăng ký routing key/queue mới.
- SSE Manager của hệ thống hỗ trợ thread-safe và có thể được inject độc lập vào các Consumer/Handler của từng module.
- Các Client hiện tại đã tuân thủ đúng định dạng JSON chuẩn của SSE event `message`.
