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
      "personalizationEnabled": true,
      "description": "Gói Premium trả phí hàng năm"
    }
    ```
    -   `personalizationEnabled` (**mới 033**, optional bool): quyền cá nhân hoá gu phong cách. **Bỏ trống ⇒
        `false`** (mặc định tắt); gói giá 0 cũng mặc định tắt cho tới khi admin bật.
*   **Response:**
    *   `201 Created`: Trả về thông tin gói cước vừa tạo, kèm `personalizationEnabled`.

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
      "personalizationEnabled": true,
      "description": "Gói Premium trả phí hàng năm - cập nhật"
    }
    ```
    -   `personalizationEnabled` (**mới 033**, optional): `null`/bỏ trống = **giữ nguyên** giá trị hiện tại;
        gửi `false` để thu hồi quyền lợi.
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

---

## Ghi chú 033 — `personalizationEnabled`

-   **Trước → sau:** trước 033, plan DTO (public + admin) **không** có `personalizationEnabled`; nay field
    additive xuất hiện ở cả `GET /subscriptions/plans`, `GET /subscriptions/me` và các API admin plan.
-   **Nguồn sự thật:** cột `subscription.subscription_plans.personalization_enabled`. Migration
    `20261027092637_add_plan_personalization.sql` bật `true` một lần cho plan `is_active = true AND price > 0`;
    gói giá 0 và gói tạo sau triển khai mặc định **tắt**.
-   **Hiệu lực:** overview subscription được cache 60 giây, nên bật/tắt quyền lợi có thể mất tối đa 60 giây
    để Commerce thấy. Không xoá dữ liệu gu; tắt quyền lợi chỉ dừng áp dụng.
-   **Tắt khẩn cấp toàn hệ thống:** dùng kill switch `personalization.enabled = false` trong config (không
    phải API này).
