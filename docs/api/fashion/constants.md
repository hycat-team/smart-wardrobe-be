# Hằng số Nghiệp vụ Fashion Module (Fashion Constants)

Các hằng số dùng trong Fashion/AI Styling APIs:

## 0. Lý do phân tích (feature 023)

Fashion item (contract DTO) mang 2 trường lý do phục vụ hiển thị Brand Portal:

- `reviewReason` — lý do AI không đủ tự tin (món cần duyệt thủ công).
- `processingErrorReason` — lý do lỗi hệ thống khi phân tích.

Cả hai được xóa (`nil`) khi phân tích lại thành công.

## 1. Bối cảnh nguồn gốc của Outfit Item (OutfitItemContext)

- **Đường dẫn package:** `internal/shared/domain/constants/wardrobe/outfititemcontext`
- **Các giá trị hợp lệ:**
    - `user_wardrobe`: Quần áo thuộc tủ đồ cá nhân của người dùng.
    - `brand_item`: Mẫu quần áo ảo từ Digital Sample Lab của đối tác nhãn hàng được thử nghiệm.

## 2. Loại yêu cầu gợi ý phối đồ (AIOutfitRequestType)

- **Các giá trị hợp lệ:**
    - `user_request`: Yêu cầu gợi ý phối đồ từ người dùng.
    - `system_request`: Yêu cầu gợi ý phối đồ từ hệ thống (tự động).

## 3. Trạng thái phân tích AI (AnalyzeTaskStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/task/analyzetaskstatus`
- **Các giá trị hợp lệ:**
    - `completed`: Phân tích thành công.
    - `failed`: Phân tích thất bại.
    - `needs_review`: Cần xem xét lại thủ công.

## 4. Danh mục Category Slugs cố định (Read-Only) & Vai trò ghép đồ (Wearing Role)

Danh mục 9 Category Slugs cố định hệ thống và bảng ánh xạ sang vai trò trong phối đồ (`WearingRole`):

- **Ánh xạ Category Slug sang Wearing Role:**
    - `ao` $\rightarrow$ `top` (Áo rời)
    - `quan`, `chan-vay` $\rightarrow$ `bottom` (Quần / Chân váy rời)
    - `dam` $\rightarrow$ `fullbody` (Đầm / Váy liền thân)
    - `ao-khoac` $\rightarrow$ `outerwear` (Áo khoác)
    - `giay` $\rightarrow$ `footwear` (Giày dép)
    - `mu` $\rightarrow$ `headwear` (Mũ nón)
    - `phu-kien` $\rightarrow$ `accessory` (Phụ kiện)
    - `other` $\rightarrow$ `other` (Mục khác / chưa phân loại)

