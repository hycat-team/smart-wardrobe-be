# Admin AI API Specs

Tài liệu đặc tả API Quản trị AI (Admin AI) — quản lý giá model, hạn mức sử dụng AI và theo dõi lịch sử sử dụng. Xem thêm hằng số tại [constants.md](constants.md).

---

## Flow 1: Quản lý cấu hình giá AI Model

### 1. Lấy danh sách cấu hình giá AI model

- **Endpoint:** `GET /api/v1/admin/ai/pricing`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Xem danh sách các cấu hình giá model AI hiện có trong hệ thống.
- **Response:** `200 OK` — Danh sách cấu hình giá.

### 2. Cập nhật cấu hình giá AI model

- **Endpoint:** `PUT /api/v1/admin/ai/pricing/{id}`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Cập nhật thông tin giá của một model AI.
- **Path Parameters:**
    - `id` (string, **required**) — UUID cấu hình giá.
- **Response:** `200 OK` — Cấu hình giá sau cập nhật.

---

## Flow 2: Quản lý hạn mức chính sách AI

### 1. Lấy danh sách hạn mức chính sách AI của người dùng

- **Endpoint:** `GET /api/v1/admin/ai/grants`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Xem các lượt grant chính sách AI đã cấp cho người dùng.
- **Query Parameters:**
    - `userId` (string, **required**) — UUID người dùng.
- **Response:** `200 OK` — Danh sách chính sách AI đã cấp.

### 2. Cấp chính sách hạn mức AI cho người dùng

- **Endpoint:** `POST /api/v1/admin/ai/grants`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Cấp trực tiếp chính sách hạn mức AI mới cho người dùng.
- **Request Body:**
    ```json
    {
      "userId": "uuid",
      "aiCostPolicyId": "uuid"
    }
    ```
- **Response:** `201 Created` — Thông tin chính sách AI vừa cấp.

---

## Flow 3: Theo dõi lịch sử sử dụng AI

### 1. Lấy lịch sử sử dụng AI của người dùng

- **Endpoint:** `GET /api/v1/admin/ai/usage`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Xem nhật ký sử dụng và chi phí AI của một người dùng.
- **Query Parameters:**
    - `userId` (string, **required**) — UUID người dùng.
    - `fromDate` (string, optional) — Từ ngày, định dạng `YYYY-MM-DD`.
    - `toDate` (string, optional) — Đến ngày, định dạng `YYYY-MM-DD`.
    - `page` (int, optional) — Số trang, mặc định `1`.
    - `limit` (int, optional) — Số bản ghi mỗi trang, mặc định `20`.
- **Response:** `200 OK` — Phân trang lịch sử sử dụng AI.
