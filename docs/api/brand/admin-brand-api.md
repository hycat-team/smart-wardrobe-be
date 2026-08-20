# Admin Brand API Specs

Tài liệu thiết kế API quản trị (Admin) liên quan đến quản lý thương hiệu: duyệt/thay đổi trạng thái, tạo thương hiệu trực tiếp, quản lý danh sách và thành viên nhãn hàng.

---

## 1. Admin cập nhật trạng thái của thương hiệu

- **Endpoint:** `PATCH /api/v1/admin/brands/:brandId/status`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Cập nhật trường trạng thái thương hiệu `brands.status`.
- **Mô tả:** Duyệt thương hiệu hoạt động bằng cách đổi trạng thái sang `active`, hoặc tạm ngưng hoạt động sang `suspended`. Trạng thái `status` tham chiếu [constants.md:BrandStatus](constants.md#1-trang-thai-cua-brand-brandstatus).
- **Request Body:**
    ```json
    {
        "status": "active"
    }
    ```
- **Response:**
    - `200 OK`: Trả về thông tin thương hiệu sau cập nhật `BrandRes`.

---

## 2. Admin tạo trực tiếp thương hiệu ở trạng thái active

- **Endpoint:** `POST /api/v1/admin/brands`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Tạo bản ghi thương hiệu `brands` ở trạng thái active trực tiếp.
- **Mô tả:** Phục vụ cho mục đích vận hành nội bộ, seed dữ liệu demo, hoặc đối tác đặc biệt đã phê duyệt trực tiếp.
- **Request Body:** Tương tự cấu trúc `POST /api/v1/brand-portal/brands`.
- **Response:**
    - `201 Created`: Trả về thông tin thương hiệu `BrandRes`.

---

## 3. Admin lấy danh sách thương hiệu

- **Endpoint:** `GET /api/v1/admin/brands`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Đọc danh sách thương hiệu (`brands`).
- **Mô tả:** Cho phép admin lấy danh sách thương hiệu phân trang, hỗ trợ lọc theo trạng thái `status` (`pending_review`, `active`, `suspended`, `archived`) và tìm kiếm theo tên hoặc slug nhãn hàng qua query param `q`.
- **Query Parameters:**
    - `page` (int, optional): Số trang cần lấy, mặc định `1`.
    - `limit` (int, optional): Số lượng phần tử mỗi trang, mặc định `20`.
    - `status` (string, optional): Trạng thái của nhãn hàng.
    - `q` (string, optional): Từ khóa tìm kiếm theo tên hoặc slug.
- **Response:**
    - `200 OK`: Trả về danh sách thương hiệu phân trang `AdminBrandListRes`.
        ```json
        {
            "success": true,
            "data": {
                "items": [
                    {
                        "id": "787c9f80-0a15-4be4-8a48-f68cdbf5f154",
                        "slug": "local-brand-a",
                        "name": "Local Brand A",
                        "description": "Thương hiệu thời trang nội địa",
                        "logoUrl": "https://res.cloudinary.com/.../logo.png",
                        "logoPublicId": "brands/local-brand-a/logo",
                        "backgroundUrl": null,
                        "backgroundPublicId": null,
                        "status": "active",
                        "totalCustomer": 150,
                        "createdByUserId": "2c9164cb-1c61-44d1-b82e-4efbb5f4b111",
                        "approvedByUserId": "3d9164cb-1c61-44d1-b82e-4efbb5f4b222",
                        "approvedAt": "2026-06-29T10:00:00Z",
                        "createdAt": "2026-06-29T09:00:00Z",
                        "updatedAt": "2026-06-29T10:00:00Z"
                    }
                ],
                "metadata": {
                    "page": 1,
                    "limit": 20,
                    "totalItems": 1,
                    "totalPages": 1
                }
            }
        }
        ```

---

## 4. Quản lý thành viên nhãn hàng (Admin)

### 4.1. Danh sách thành viên nhãn hàng

- **Endpoint:** `GET /api/v1/admin/brands/:brandId/members`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Đọc danh sách thành viên `brand_members` của nhãn hàng.
- **Mô tả:** Quản trị viên xem danh sách tất cả thành viên của nhãn hàng.
- **Response:**
    - `200 OK`: Trả về mảng danh sách thành viên `BrandMemberRes`.

### 4.2. Thêm nhân viên cho nhãn hàng

- **Endpoint:** `POST /api/v1/admin/brands/:brandId/members`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Tạo mới bản ghi thành viên `brand_members`.
- **Mô tả:** Quản trị viên gán người dùng làm nhân viên (staff) của nhãn hàng.
- **Request Body:**
    ```json
    {
        "userId": "2c9164cb-1c61-44d1-b82e-4efbb5f4b111"
    }
    ```
- **Validation chính:**
    - `userId`: bắt buộc, phải là UUID hợp lệ của người dùng đã tồn tại trong hệ thống.
- **Response:**
    - `200 OK`: Trả về thông tin thành viên vừa tạo `BrandMemberRes`.

### 4.3. Cập nhật vai trò/trạng thái thành viên

- **Endpoint:** `PATCH /api/v1/admin/brands/:brandId/members/:memberId`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Cập nhật bản ghi thành viên `brand_members`.
- **Mô tả:** Quản trị viên cập nhật vai trò hoặc trạng thái hoạt động của thành viên nhãn hàng. Tất cả các trường đều là tùy chọn, chỉ gửi những trường cần thay đổi.
- **Request Body:**
    ```json
    {
        "role": "staff",
        "status": "active"
    }
    ```
- **Ghi chú:**
    - `role`: Tham chiếu [constants.md:BrandMemberRole](constants.md#2-vai-tro-thanh-vien-trong-brand-brandmemberrole).
    - `status`: Tham chiếu [constants.md:BrandMemberStatus](constants.md#3-trang-thai-thanh-vien-trong-brand-brandmemberstatus).
- **Response:**
    - `200 OK`: Trả về thông tin thành viên sau cập nhật `BrandMemberRes`.

### 4.4. Gỡ bỏ thành viên khỏi nhãn hàng

- **Endpoint:** `DELETE /api/v1/admin/brands/:brandId/members/:memberId`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Xóa bản ghi thành viên `brand_members`.
- **Mô tả:** Quản trị viên xóa thành viên khỏi nhãn hàng (bảo vệ chủ sở hữu duy nhất).
- **Response:**
    - `200 OK`: Thành công.

### 4.5. Chuyển giao quyền sở hữu nhãn hàng

- **Endpoint:** `POST /api/v1/admin/brands/:brandId/transfer-ownership`
- **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
- **Đối tượng ảnh hưởng:** Chuyển quyền chủ sở hữu (`owner`) cho người dùng khác trong nhãn hàng.
- **Mô tả:** Quản trị viên chuyển quyền Owner nhãn hàng cho người dùng khác trong nhãn hàng, đồng thời chọn hạ chủ cũ thành staff hoặc gỡ hoàn toàn.
- **Request Body:**
    ```json
    {
        "newOwnerUserId": "2c9164cb-1c61-44d1-b82e-4efbb5f4b111",
        "oldOwnerHandling": "staff"
    }
    ```
- **Validation chính:**
    - `newOwnerUserId`: bắt buộc, phải là UUID hợp lệ.
    - `oldOwnerHandling`: bắt buộc, chỉ nhận `staff` (hạ xuống staff) hoặc `remove` (gỡ khỏi brand).
- **Response:**
    - `200 OK`: Thành công.
