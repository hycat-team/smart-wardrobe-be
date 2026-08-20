# Subscription API Specs

Tài liệu thiết kế các API liên quan đến gói dịch vụ đăng ký thành viên (subscription) và hạn mức sử dụng AI (quota AI). Tất cả các giá trị hằng số sử dụng tham chiếu tại [constants.md](constants.md).

---

## Flow 1: Xem gói subscription và quản lý hạn mức AI (Quota)

Người dùng tham khảo các gói dịch vụ Premium hiện có và theo dõi mức độ tiêu thụ lượng gọi phối đồ AI của cá nhân trong ngày.

### 1. Lấy danh sách các gói Premium hiện có trên Closy
*   **Endpoint:** `GET /api/v1/subscriptions/plans`
*   **Tác nhân (Actor):** Khách hàng / Khách vãng lai (Public/Customer).
*   **Đối tượng ảnh hưởng:** Đọc danh sách gói dịch vụ `subscription_plans`.
*   **Mô tả:** Trả về danh sách các gói cước đang được cấu hình hoạt động trên hệ thống. Loại gói cước `planKind` tham chiếu tại [constants.md:PlanKind](constants.md#3-phan-loai-goi-cuoc-plankind).
*   **Response:**
    *   `200 OK`: Trả về danh sách các gói cước.

### 2. Xem tổng quan tình trạng gói cước của cá nhân
*   **Endpoint:** `GET /api/v1/subscriptions/me`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc bản ghi gói cước đăng ký đang hoạt động của người dùng hiện tại (current user).
*   **Mô tả:** Trả về thông tin chi tiết gói cước người dùng đang sử dụng, thời hạn hết hạn gói, trạng thái tự động gia hạn (auto-renew) và các thông số giới hạn quota kèm theo.
*   **Response:**
    *   `200 OK`: Trả về thông tin tổng quan gói cước.

### 3. Lấy thông tin hạn mức gọi AI hàng ngày
*   **Endpoint:** `GET /api/v1/subscriptions/me/daily-quota`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc hạn mức sử dụng AI hàng ngày của current user.
*   **Mô tả:** Trả về giới hạn số lần gọi AI gợi ý phối đồ tối đa được cấp trong ngày và lượng đã tiêu dùng thực tế của người dùng tính đến thời điểm hiện tại.
*   **Response:**
    *   `200 OK`: Trả về thông tin quota trong ngày.

### 4. Bật hoặc tắt tính năng tự động gia hạn gói cước
*   **Endpoint:** `PUT /api/v1/subscriptions/me/auto-renew`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái tự động gia hạn của bản ghi đăng ký gói của current user.
*   **Mô tả:** Thiết lập bật hoặc tắt cơ chế tự động trừ tiền từ ví nội bộ để duy trì gói Premium khi đến ngày hết hạn chu kỳ.
*   **Response:**
    *   `200 OK`: Trả về thông tin đăng ký gói sau cập nhật.
