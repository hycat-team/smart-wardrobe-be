# Admin Subscription API Specs

Tài liệu thiết kế các API quản trị viên liên quan đến quản lý gói cước và đăng ký hội viên người dùng. Tất cả các giá trị hằng số sử dụng tham chiếu tại [constants.md](constants.md).

---

## Danh sách đăng ký gói của người dùng

### 1. Lấy danh sách đăng ký gói hội viên của người dùng
*   **Endpoint:** `GET /api/v1/admin/subscriptions`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách bản ghi đăng ký gói hội viên `user_subscriptions`.
*   **Mô tả:** Lấy danh sách đăng ký gói hội viên của người dùng, hỗ trợ lọc theo `planCode` và `status`.
*   **Query Parameters:**
    *   `planCode` (string, optional): Mã slug gói.
    *   `status` (string, optional): Trạng thái (`active`, `expired`).
    *   `page` (integer, optional): Trang.
    *   `limit` (integer, optional): Số bản ghi/trang.
*   **Response:**
    *   `200 OK`: Trả về danh sách đăng ký gói của người dùng (phân trang).

---

## Quản lý gói cước (Plans)

### 2. Lấy danh sách tất cả gói cước
*   **Endpoint:** `GET /api/v1/admin/subscriptions/plans`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc danh sách gói dịch vụ `subscription_plans`.
*   **Mô tả:** Lấy toàn bộ danh sách gói cước trong hệ thống (bao gồm cả gói đang tắt).
*   **Response:**
    *   `200 OK`: Trả về danh sách tất cả gói cước.

### 3. Tạo gói cước mới
*   **Endpoint:** `POST /api/v1/admin/subscriptions/plans`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Tạo mới bản ghi gói dịch vụ `subscription_plans`.
*   **Mô tả:** Tạo một gói cước mới trong hệ thống.
*   **Request Body:**
    ```json
    {
      "name": "Premium Yearly",
      "planCode": "premium_yearly",
      "planKind": "premium",
      "price": 990000,
      "currency": "VND",
      "billingPeriodMonths": 12,
      "dailyAiQuota": 50,
      "aiEnforcementMode": "strict",
      "wardrobeLimit": 500,
      "description": "Gói Premium trả phí hàng năm"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin gói cước vừa tạo.

### 4. Cập nhật thông tin gói cước
*   **Endpoint:** `PUT /api/v1/admin/subscriptions/plans/{id}`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật bản ghi gói dịch vụ `subscription_plans`.
*   **Mô tả:** Cập nhật thông tin chi tiết một gói cước theo ID.
*   **Path Parameters:**
    *   `id` (string, required): ID gói cước.
*   **Request Body:**
    ```json
    {
      "name": "Premium Yearly",
      "price": 899000,
      "dailyAiQuota": 60,
      "wardrobeLimit": 600,
      "description": "Gói Premium trả phí hàng năm - cập nhật"
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin gói cước sau cập nhật.

### 5. Bật hoặc tắt trạng thái hoạt động gói cước
*   **Endpoint:** `PATCH /api/v1/admin/subscriptions/plans/{id}/status`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật trường `isActive` của bản ghi gói dịch vụ `subscription_plans`.
*   **Mô tả:** Bật hoặc tắt trạng thái isActive của gói cước.
*   **Path Parameters:**
    *   `id` (string, required): ID gói cước.
*   **Request Body:**
    ```json
    {
      "isActive": false
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin gói cước sau cập nhật.

---

## Quản lý đăng ký hội viên người dùng

### 6. Hủy gói hội viên của người dùng
*   **Endpoint:** `POST /api/v1/admin/subscriptions/{userId}/cancel`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Cập nhật bản ghi đăng ký gói `user_subscriptions`, chuyển trạng thái sang hủy.
*   **Mô tả:** Hủy gói hội viên của người dùng và chuyển về gói miễn phí (Free).
*   **Path Parameters:**
    *   `userId` (string, required): User ID.
*   **Response:**
    *   `200 OK`: Trả về thông tin đăng ký gói sau khi hủy.

### 7. Cấp gói hội viên thủ công cho người dùng
*   **Endpoint:** `POST /api/v1/admin/subscriptions/{userId}/grant`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Tạo mới hoặc cập nhật bản ghi đăng ký gói `user_subscriptions`.
*   **Mô tả:** Gán một gói hội viên cụ thể cho người dùng.
*   **Path Parameters:**
    *   `userId` (string, required): User ID.
*   **Request Body:**
    ```json
    {
      "planId": "38a68d7f-5f47-46c4-8ad4-b98f28130222"
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin đăng ký gói vừa cấp.
