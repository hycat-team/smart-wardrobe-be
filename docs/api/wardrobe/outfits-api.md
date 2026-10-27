# Outfits API Specs

Tài liệu này đặc tả các API liên quan đến bộ phối đồ (outfit) trong hệ thống Closy Smart Wardrobe.

---

## Flow 3: Tự thiết kế bộ phối đồ (Outfit)

### 1. Lấy chữ ký tải lên ảnh bìa bộ phối đồ

- **Endpoint:** `GET /api/v1/outfits/upload-signature`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Không thay đổi dữ liệu nghiệp vụ.
- **Mô tả:** Lấy Cloudinary signature phục vụ client tải ảnh chụp/render cover của outfit lên Cloudinary.
- **Response:** `200 OK` — `UploadSignatureResult`
  (`signature`, `timestamp`, `apiKey`, `folder`, `resourceType`).
- **Lưu ý:** `publicId` **không có** trong response (endpoint này không ký `public_id`).
  Client **không được** gửi `public_id` lên Cloudinary; Cloudinary tự sinh `public_id` và client
  đọc từ response upload.
- **Form upload:** = các field đã ký + `file` + `api_key` + `signature`. Endpoint này ký
  2 field (`timestamp`, `folder`) nên gửi **đúng 5 field**: `file`, `api_key`, `timestamp`,
  `signature`, `folder`. Thêm field nào ngoài danh sách này (kể cả `public_id`, `eager`, `tags`)
  hoặc bỏ field nào đều làm lệch chữ ký → upload bị từ chối.

### 2. Lưu bộ phối đồ (outfit) tự thiết kế mới

- **Endpoint:** `POST /api/v1/outfits`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Tạo bản ghi outfit và danh sách chi tiết các item thuộc bộ phối đồ (`outfit_items`); có thể tạo bản sao `wardrobe_items` khi lưu món từ tủ hệ thống.
- **Mô tả:** Lưu trữ thông tin bộ trang phục do người dùng tự phối ghép bao gồm danh sách các item thời trang, tọa độ hiển thị 2D trên canvas và thứ tự hiển thị layer của từng item.
- **Request Body:**
    ```json
    {
      "name": "Set đồ đi làm",
      "description": "...",
      "coverImageUrl": "https://...",
      "coverPublicId": "...",
      "items": [
        {
          "fashionItemId": "uuid",
          "itemContext": "system_catalog",
          "positionX": 0,
          "positionY": 0,
          "scale": 1,
          "layerOrder": 1
        }
      ]
    }
    ```
    -   `itemContext` (optional, **mới 033**): `user_wardrobe | brand_item | system_catalog`.
        Bỏ trống ⇒ auto-resolve theo thứ tự **wardrobe → brand item hợp lệ → system catalog**.
    -   Món `system_catalog` phải là item hệ thống còn hợp lệ. Nếu user **chưa sở hữu** fashion item,
        backend tạo một row `wardrobe_items` mới (`item_type = UserItem`, copy giá) trong **cùng
        transaction** với outfit; outfit item lưu cùng `fashion_item_id` với
        `item_context = user_wardrobe`.
    -   Nếu user **đã sở hữu** `fashionItemId` (tồn tại row `wardrobe_items` chưa xoá, **bất kỳ status**),
        backend tái sử dụng row đó, không clone thêm và không tính thêm hạn mức. Ngoại lệ: row đã bán
        (`Sold`) bị từ chối với `OUTFIT_ITEM_SOLD`; row bị khoá do vượt hạn mức bị từ chối với
        `ITEM_LOCKED_DUE_TO_LIMIT`.
- **Lỗi (033):**
    -   `400 CATALOG_ITEM_UNAVAILABLE`: món hệ thống đã bị gỡ/không còn `InWardrobe`/không có fashion item.
    -   `400 WARDROBE_LIMIT_EXCEEDED_FOR_OUTFIT`: clone làm vượt `MaxWardrobeItems`; toàn bộ outfit bị từ
        chối và message nêu mã các món gây vượt.
    -   `400 DUPLICATE_OUTFIT_ITEM`: cùng `fashionItemId` xuất hiện hai lần.
    -   `400 OUTFIT_ITEM_INVALID_OR_FORBIDDEN`: món không thuộc tủ/brand không hợp lệ.
    -   `400 OUTFIT_ITEM_SOLD`: món đã bán.
    -   `403 ITEM_LOCKED_DUE_TO_LIMIT`: món bị khoá do vượt hạn mức.
- **Trước → sau:** trước 033, `itemContext` không có trong request và backend không nhận món từ tủ hệ
    thống; nay gửi `system_catalog` sẽ clone trong cùng transaction. Client cũ không gửi field vẫn chạy
    như trước.

### 3. Cập nhật thông tin bộ phối đồ

- **Endpoint:** `PUT /api/v1/outfits/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Cập nhật thông tin outfit của current user; có thể clone thêm `wardrobe_items` khi thêm món hệ thống.
- **Mô tả:** Dùng chung logic resolve/clone với tạo mới. Chỉ các item **mới thêm** được touch
  `last_used_at` và phát signal học gu; cập nhật metadata không làm thay đổi `last_used_at`.
- **Trước → sau:** trước 033 chỉ cập nhật metadata và item hiện có; nay `items` cũng nhận `itemContext`
  (optional) và có thể clone món hệ thống như bước tạo mới.

### 4. Lấy danh sách các bộ phối đồ của tôi

- **Endpoint:** `GET /api/v1/me/outfits`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc danh sách outfits của current user.

### 5. Xem thông tin chi tiết một bộ phối đồ

- **Endpoint:** `GET /api/v1/outfits/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc chi tiết một bản ghi outfit của current user.
- **Mô tả:** Trả về thông tin chi tiết của outfit và danh sách các item thời trang đi kèm để hiển thị vẽ lại trên canvas của app.

### 6. Xóa bộ phối đồ khỏi danh sách

- **Endpoint:** `DELETE /api/v1/outfits/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Xóa mềm bản ghi outfit của current user.

---

## B2B2C Model Notes

- Món đồ cá nhân trong tủ đồ (`wardrobe_items`) và sản phẩm của nhãn hàng (`brand_items`) đều sử dụng chung một cơ chế tham chiếu thông tin dữ liệu thời trang qua bảng trung gian `fashion_items`.
- Danh sách chi tiết bộ phối đồ (`outfit_items`) sử dụng khóa ngoại `fashion_item_id` kết hợp với trường bối cảnh `item_context` để phân biệt rõ món đồ đó được lấy từ tủ đồ cá nhân hay từ Digital Sample Lab của nhãn hàng đối tác. Giá trị trường bối cảnh `item_context` tham chiếu chi tiết tại [constants.md](constants.md).
- Khi lưu trữ hoặc xem chi tiết bộ phối đồ có chứa sản phẩm nhãn hàng (brand items), hệ thống backend bắt buộc phải xác thực sản phẩm đó thuộc nhãn hàng hợp lệ và tài khoản người dùng có quyền tiếp cận theo đúng các chính sách loyalty/benefit đang áp dụng.
