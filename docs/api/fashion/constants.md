# Hằng số Nghiệp vụ Fashion Module (Fashion Constants)

Các hằng số dùng trong Fashion/AI Styling APIs:

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
