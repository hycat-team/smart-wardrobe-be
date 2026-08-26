# Nghiên cứu & Quyết định Kỹ thuật (Research & Decisions): 014-decouple-fashion-worker-sse

**Feature**: [spec.md](./spec.md) | **Date**: 2026-08-26

---

## 1. Quyết định Về Kiến trúc Sự kiện (Event-Driven Architecture)

### Quyết định 1: Định danh Topic và Cơ chế Xuất bản Sự kiện (Event Publishing)

* **Quyết định**: Sử dụng Topic `fashion.event.analyzed` và `fashion.event.failed` thông qua **Transactional Outbox Pattern** (`IEventPublisher`).
* **Lý do chọn**: 
  * Dự án SmartWardrobe Backend đã chuẩn hóa toàn bộ event publishing qua `internal/shared/application/constants/eventconstants/` và bảng `outbox_records` (Hiến pháp §III).
  * Đảm bảo tính tin cậy "At-Least-Once Delivery", tránh việc AI phân tích xong nhưng mất kết nối RabbitMQ dẫn đến mất event.
* **Các phương án đã cân nhắc**:
  * *Gọi hàm trực tiếp (RPC/Direct Interface)*: Bị loại bỏ vì gây ra Runtime Call Cycle và khớp nối cứng (Coupling).
  * *Publish trực tiếp vào RabbitMQ không qua Outbox*: Bị loại bỏ vì không đảm bảo tính nhất quán dữ liệu nếu RabbitMQ gặp sự cố tức thời.

---

### Quyết định 2: Cấu trúc Payload Sự kiện Phân tích (`FashionAnalyzedEventPayload`)

* **Quyết định**: Định nghĩa struct sự kiện chuẩn đặt trong `internal/modules/fashion/application/dto/contract_dtos.go`:
  ```go
  type FashionAnalyzedEventPayload struct {
      TaskID        string                              `json:"taskId"`
      ItemID        uuid.UUID                           `json:"itemId"`
      ItemType      fashionitemtype.FashionItemType     `json:"itemType"` // "wardrobe" | "brand"
      FashionItemID *uuid.UUID                          `json:"fashionItemId,omitempty"`
      Status        analyzetaskstatus.AnalyzeTaskStatus `json:"status"` // "completed" | "needs_review" | "failed"
      Total         int                                 `json:"total"`
      Index         int                                 `json:"index"`
      FashionBrief  *FashionItemBriefRes                `json:"fashionBrief,omitempty"`
      Error         string                              `json:"error,omitempty"`
  }
  ```
* **Lý do chọn**:
  * Đóng gói sẵn `FashionBrief` (ảnh, màu hex, style, category) ngay tại Fashion Worker $\rightarrow$ Consumer của Wardrobe / Brand không cần phải truy vấn lại bảng `fashion_items` (triệt tiêu vòng lặp Read-After-Write).
  * `ItemType` cho phép các consumer lọc chính xác thông điệp thuộc phạm vi quản lý của mình.

---

### Quyết định 3: Thiết kế Consumer cho Module Wardrobe và Brand

* **Quyết định**: 
  1. Tạo `WardrobeAnalyzeConsumer` đặt tại `internal/modules/wardrobe/presentation/worker/` kết nối với queue `wardrobe.fashion.analyzed.queue`.
  2. Tạo `BrandAnalyzeConsumer` đặt tại `internal/modules/brand/presentation/worker/` kết nối với queue `brand.fashion.analyzed.queue`.
* **Lý do chọn**:
  * Tuân thủ Clean Architecture (§IV): Worker/Consumer thuộc tầng `presentation/worker`, gọi vào UseCase của module đó.
  * Mỗi module có queue độc lập để đảm bảo nếu một bên xử lý chậm không làm nghẽn bên còn lại.

---

### Quyết định 4: Cơ chế Phân phối SSE Realtime (SSE Distribution)

* **Quyết định**: 
  * Sử dụng `ISSEManager` dùng chung của `internal/shared/application/sse/` được khởi tạo dưới dạng Singleton qua Wire DI.
  * `WardrobeItemHandler` (mở stream SSE) và `WardrobeAnalyzeConsumer` (nhận event và bắn SSE) chia sẻ cùng một instance `ISSEManager`.
  * Tương tự cho `BrandPortalHandler` và `BrandAnalyzeConsumer`.
* **Lý do chọn**:
  * Tận dụng hạ tầng SSE thread-safe in-memory sẵn có của dự án.
  * Không làm thay đổi giao thức, định dạng JSON hay URL endpoint phía Client.

---

### Quyết định 5: Khắc phục lỗi thiếu trường `FashionItem` trong DTO gửi về Client

* **Quyết định**:
  * Trong `WardrobeAnalyzeConsumer`: Khởi tạo `WardrobeItemBriefRes` và gán trường `FashionItem` trực tiếp từ `event.FashionBrief`.
  * Trong `BrandAnalyzeConsumer`: Khởi tạo `BrandItemBriefRes` và gán trường `FashionItem` trực tiếp từ `event.FashionBrief`.
* **Lý do chọn**:
  * Xử lý dứt điểm tình trạng `fashionItem` bị `nil` khi trả về cho Client.
  * Đảm bảo tính toàn vẹn dữ liệu cho giao diện người dùng.
