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
- **Đối tượng ảnh hưởng:** Tạo bản ghi outfit và danh sách chi tiết các item thuộc bộ phối đồ (`outfit_items`).
- **Mô tả:** Lưu trữ thông tin bộ trang phục do người dùng tự phối ghép bao gồm danh sách các item thời trang, tọa độ hiển thị 2D trên canvas và thứ tự hiển thị layer của từng item.

### 3. Cập nhật thông tin bộ phối đồ

- **Endpoint:** `PUT /api/v1/outfits/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Cập nhật thông tin outfit của current user.

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
