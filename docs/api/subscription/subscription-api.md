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
    *   **Mới 033:** mỗi gói có thêm `personalizationEnabled` (bool) — quyền lợi cá nhân hoá gu phong cách.

### 2. Xem tổng quan tình trạng gói cước của cá nhân
*   **Endpoint:** `GET /api/v1/subscriptions/me`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc bản ghi gói cước đăng ký đang hoạt động của người dùng hiện tại (current user).
*   **Mô tả:** Trả về thông tin chi tiết gói cước người dùng đang sử dụng, thời hạn hết hạn gói, trạng thái tự động gia hạn (auto-renew) và các thông số giới hạn quota kèm theo.
*   **Response:**
    *   `200 OK`: Trả về thông tin tổng quan gói cước.
    *   **Mới 033:** thêm `personalizationEnabled` (bool). Ví dụ:

        ```json
        {
          "planId": "uuid",
          "planName": "Premium Plan",
          "planSlug": "premium-monthly",
          "expiresAt": "2026-11-26T00:00:00Z",
          "isAutoRenewEnabled": true,
          "maxWardrobeItems": 300,
          "maxOutfits": 300,
          "aiOutfitDailyQuota": 30,
          "aiChatDailyQuota": 10,
          "personalizationEnabled": true
        }
        ```
        Giá trị đọc từ **plan row hiện tại** qua overview cache (TTL 60 giây), không đọc lại snapshot quyền
        lợi cũ. Admin bật/tắt có thể mất tối đa 60 giây mới thấy thay đổi; cache cũ thiếu field được đọc là
        `false`.

> **Trước → sau (033):** response `GET /subscriptions/me` và danh sách gói trước đây không có
> `personalizationEnabled`; nay có thêm field additive, client cũ bỏ qua được. Không đổi tên/kiểu field cũ.

### 3. Lấy thông tin hạn mức gọi AI hàng ngày
*   **Endpoint:** `GET /api/v1/subscriptions/me/daily-quota`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc hạn mức sử dụng AI hàng ngày của current user.
*   **Mô tả:** Trả về giới hạn số lần gọi AI gợi ý phối đồ tối đa được cấp trong ngày, lượng đã tiêu dùng thực tế tính đến thời điểm hiện tại, và hạn mức tối đa của gói hiện hành (`maxWardrobeItems`, `maxOutfits`) để client hiển thị song song với số lượng hiện có.
*   **Response:**
    *   `200 OK`: Trả về thông tin quota trong ngày, bao gồm:
        *   `maxWardrobeItems`: Số lượng món đồ tối đa của gói hiện hành.
        *   `maxOutfits`: Số lượng bộ phối đồ tối đa của gói hiện hành.
        *   `aiOutfitDailyQuota` / `outfitRecommendCount`: Hạn mức và lượng đã dùng AI phối đồ trong ngày.
        *   `aiChatDailyQuota` / `aiUsageCount`: Hạn mức và lượng đã dùng AI chat trong ngày.
        *   `lastResetDate`: Mốc thời gian reset hạn mức gần nhất.

> **Ghi chú:** Hai trường `maxWardrobeItems`/`maxOutfits` được bổ sung vào `daily-quota` để frontend đọc hạn mức tối đa từ một nguồn duy nhất. Số lượng **hiện tại** đọc từ `GET /api/v1/me/wardrobe-items/stats`. Thay đổi này đảo ngược quyết định trim ở spec `013-briefres-audit` (T057/T081) với lý do nêu trên.

### 4. Bật hoặc tắt tính năng tự động gia hạn gói cước
*   **Endpoint:** `PUT /api/v1/subscriptions/me/auto-renew`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái tự động gia hạn của bản ghi đăng ký gói của current user.
*   **Mô tả:** Thiết lập bật hoặc tắt cơ chế tự động trừ tiền từ ví nội bộ để duy trì gói Premium khi đến ngày hết hạn chu kỳ.
*   **Response:**
    *   `200 OK`: Trả về thông tin đăng ký gói sau cập nhật.
