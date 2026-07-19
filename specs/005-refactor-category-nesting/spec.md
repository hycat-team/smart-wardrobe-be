# Feature Specification: Refactor Category Nesting

**Feature Branch**: `005-refactor-category-nesting`

**Created**: 2026-07-18

**Status**: Draft

**Input**: User description: "Bàn luận về việc chuyển đổi 3 trường category phẳng (CategoryID, CategoryName, CategorySlug) trong FashionItemRes thành 1 đối tượng lồng Category và loại bỏ phần category tự định nghĩa ở wardrobe_items/brand_items."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Đồng bộ hóa cấu trúc dữ liệu Wardrobe Item (Priority: P1)

Là một lập trình viên hoặc client tiêu thụ API tủ đồ, tôi muốn thông tin danh mục (`Category`) được lồng trực tiếp bên trong đối tượng trang phục (`FashionItem`) thay vì nằm phẳng ở root của `WardrobeItem`, để cấu trúc dữ liệu phản ánh chính xác nghiệp vụ (Domain Model) và loại bỏ việc tự định nghĩa các cấu trúc Category trùng lặp ở module Wardrobe.

**Why this priority**: Đây là phần cốt lõi của việc tái cấu trúc, giúp đồng bộ hóa mô hình dữ liệu của tủ đồ theo đúng quan hệ thực tế (Wardrobe Item -> Fashion Item -> Category).

**Independent Test**: API lấy danh mục tủ đồ (`GET /api/v1/system-catalog/wardrobe-items`) và API tủ đồ cá nhân trả về trường `fashionItem` chứa đối tượng `category` đầy đủ (`id`, `name`, `slug`), đồng thời loại bỏ trường `category` ở tầng ngoài cùng của tủ đồ.

**Acceptance Scenarios**:

1. **Given** một wardrobe item có liên kết với fashion item có danh mục là "Áo Thun", **When** gọi API lấy chi tiết hoặc danh sách tủ đồ, **Then** cấu trúc JSON trả về không có trường `category` ở ngoài cùng, mà có `fashionItem.category` chứa `id`, `name`, `slug` của danh mục "Áo Thun".

---

### User Story 2 - Đồng bộ hóa cấu trúc dữ liệu Brand Item (Priority: P2)

Là một lập trình viên hoặc client tiêu thụ API của thương hiệu, tôi muốn struct `FashionItemRes` dùng chung cho Brand Item cũng sử dụng cấu trúc danh mục lồng tương tự, để đồng bộ hóa DTO và tránh việc định nghĩa các trường phẳng `CategoryName`, `CategorySlug` độc lập.

**Why this priority**: Đảm bảo tất cả các module trong hệ thống (bao gồm cả module Brand) đều sử dụng một cấu trúc trang phục (`FashionItem`) đồng nhất.

**Independent Test**: API danh sách sản phẩm thương hiệu (`GET /api/v1/brand-items`) trả về trường `fashionItem` với đối tượng `category` lồng bên trong.

**Acceptance Scenarios**:

1. **Given** một brand item liên kết với fashion item, **When** gọi API lấy thông tin brand item, **Then** thông tin danh mục của fashion item được trả về dưới dạng đối tượng lồng `category` thay vì các trường phẳng `categoryName`/`categorySlug` trong DTO của Brand.

---

### Edge Cases

- Làm thế nào hệ thống xử lý khi trang phục (`FashionItem`) chưa được phân loại danh mục (Category) hoặc Category bị xóa?
  - *Giải pháp:* Trường `category` trong `FashionItemRes` sẽ là `nil` (hoặc đối tượng rỗng). Hệ thống phải kiểm tra nil an toàn để tránh panic.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST định nghĩa kiểu dữ liệu `CategoryBriefRes` (chỉ bao gồm `ID`, `Name`, `Slug`, không chứa `SortOrder`) lồng bên trong `FashionItemRes` thay cho các trường phẳng độc lập.
- **FR-002**: Module `fashion` MUST cập nhật contract `IFashionContract` để trả về cấu trúc trang phục mới sử dụng `CategoryBriefRes`.
- **FR-003**: Module `wardrobe` MUST loại bỏ trường `category` tự định nghĩa ở ngoài cùng của `WardrobeItem` response DTO.
- **FR-004**: Module `brand` MUST cập nhật DTO `BrandItemRes` cục bộ để sử dụng chung cấu trúc `FashionItemRes` mới từ module `fashion`.
- **FR-005**: Mapper trong tất cả các module MUST cập nhật logic chuyển đổi sang cấu trúc lồng, đảm bảo kiểm tra nil an toàn.

### Key Entities *(include if feature involves data)*

- **FashionItemRes**: Thực thể đại diện cho thông tin trang phục gửi qua contract, chứa trường `category` kiểu `CategoryBriefRes`.
- **CategoryBriefRes**: Thực thể danh mục thu gọn gồm các trường `id` (uuid.UUID), `name` (string), `slug` (string), loại bỏ hoàn toàn thuộc tính `sortOrder`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Loại bỏ 100% các trường phẳng `categoryName`, `categorySlug` ở root của các cấu trúc liên quan đến trang phục.
- **SC-002**: Giảm số lượng struct `Category` tự định nghĩa trùng lặp ở module `wardrobe` và `brand` xuống còn 0 (sử dụng chung struct của module `fashion`).
- **SC-003**: 100% các API liên quan (Wardrobe, Brand, Fashion) vượt qua kiểm thử biên dịch và unit tests sau khi refactor.

## Assumptions

- Phía Frontend/Client chấp nhận breaking change này và sẽ cập nhật code hiển thị tương ứng với cấu trúc JSON mới.
- Các module giao tiếp với nhau bằng contract nội bộ (Google Wire), không thông qua dịch vụ mạng khác nên có thể đồng bộ code Go cùng một lúc.


