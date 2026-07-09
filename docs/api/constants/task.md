# Hằng số Trạng thái Phân tích & Nhiệm vụ (Task Constants)

Các hằng số dùng trong luồng Server-Sent Events (SSE) để theo dõi tiến trình phân tích trang phục (cả tủ đồ cá nhân và sản phẩm nhãn hàng):

## 1. Trạng thái Nhiệm vụ Phân tích (AnalyzeTaskStatus)
*   **Đường dẫn package:** `internal/shared/application/constants/analyzetaskstatus`
*   **Các giá trị hợp lệ:**
    *   `completed`: Phân tích hoàn tất thành công.
    *   `failed`: Phân tích thất bại (không thể đọc ảnh, lỗi AI vĩnh viễn...).
    *   `needs_review`: AI phân tích xong nhưng độ tin cậy thấp hoặc không phân loại được thể loại chính xác, cần chờ duyệt/kiểm tra lại từ người dùng hoặc hệ thống.

## 2. Loại Vật phẩm Phân tích (FashionItemType)
*   **Đường dẫn package:** `internal/modules/fashion/domain/constants/fashionitemtype`
*   **Các giá trị hợp lệ:**
    *   `wardrobe`: Vật phẩm thuộc tủ đồ cá nhân của người dùng.
    *   `brand`: Vật phẩm mẫu (sample) hoặc sản phẩm thương mại của nhãn hàng.
