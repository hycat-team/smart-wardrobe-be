# Hằng số Nghiệp vụ Wardrobe & Outfit (Wardrobe Constants)

Các hằng số dùng trong APIs liên quan đến Tủ đồ, Trang phục và Gợi ý phối đồ AI:

## 1. Trạng thái Vật phẩm trong tủ đồ (WardrobeStatus)
*   **Đường dẫn package:** `internal/shared/domain/constants/wardrobe/wardrobestatus`
*   **Các giá trị hợp lệ:**
    *   `active`: Quần áo đang sẵn sàng để phối đồ và hiển thị trên tủ đồ số.
    *   `archived`: Quần áo đã bị lưu trữ (không tham gia gợi ý phối đồ).

## 2. Bối cảnh nguồn gốc của Outfit Item (OutfitItemContext)
*   **Đường dẫn package:** `internal/shared/domain/constants/wardrobe/outfititemcontext`
*   **Các giá trị hợp lệ:**
    *   `user_wardrobe`: Quần áo thuộc tủ đồ cá nhân của người dùng.
    *   `brand_item`: Mẫu quần áo ảo từ Digital Sample Lab của đối tác nhãn hàng được thử nghiệm.

## 3. Phân loại Loại trang phục chính (ItemType)
*   **Đường dẫn package:** `internal/shared/domain/constants/wardrobe/itemtype`
*   **Các giá trị hợp lệ:**
    *   `top`: Áo (Áo phông, Áo sơ mi...).
    *   `bottom`: Quần / Váy (Chân váy, Quần dài...).
    *   `outerwear`: Áo khoác.
    *   `one_piece`: Đầm liền / Jumpsuit.
    *   `footwear`: Giày dép.
    *   `accessory`: Phụ kiện (Kính, Mũ, Túi...).

## 4. Tình trạng vật lý của quần áo (ItemCondition)
*   **Đường dẫn package:** `internal/shared/domain/constants/wardrobe/itemcondition`
*   **Các giá trị hợp lệ:**
    *   `excellent`: Quần áo còn rất mới, ít sử dụng.
    *   `good`: Đang sử dụng tốt.
    *   `fair`: Hơi cũ hoặc có vết mòn nhẹ.
    *   `poor`: Đã cũ hoặc hỏng nhẹ cần sửa.
