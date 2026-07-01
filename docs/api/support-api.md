# Support & Brand Chat API Specs

Tài liệu thiết kế các API liên quan đến kênh chat trực tuyến hỗ trợ khách hàng giữa Người dùng (Khách hàng) và Đội ngũ hỗ trợ của Nhãn hàng (Brand Staff) trong mô hình B2B2C.

---

## Flow 1: Khách hàng trao đổi (chat) với thương hiệu (Customer)

### 1. Lấy thông tin phòng chat hiện tại với nhãn hàng (kèm tin nhắn phân trang)
*   **Endpoint:** `GET /api/v1/brands/:brandId/conversation`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc thông tin hoặc tự động khởi tạo bản ghi cuộc hội thoại tùy thuộc vào use case backend.
*   **Query Params:**
    *   `page` (optional): Trang hiện tại, mặc định `1`.
    *   `limit` (optional): Số tin nhắn mỗi trang, mặc định `20`.
*   **Mô tả:** Lấy thông tin phòng chat của người dùng hiện tại (current user) với nhãn hàng cụ thể, kèm danh sách tin nhắn phân trang. Khách hàng offline chưa liên kết tài khoản Closy sẽ không thể truy cập giao diện chat công khai này. Trạng thái `status` tham chiếu chi tiết tại [constants/brand.md:BrandConversationStatus](constants/brand.md).
*   **Response:**
    *   `200 OK`: Trả về thông tin phòng chat `BrandConversationDetailRes` bao gồm thông tin hội thoại (`BrandConversationRes`) + danh sách tin nhắn phân trang (`messages` + `metadata`).

### 2. Khách hàng gửi tin nhắn mới đến thương hiệu
*   **Endpoint:** `POST /api/v1/brands/:brandId/conversation/messages`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi tin nhắn mới `brand_conversation_messages`; tự động mở lại phòng chat nếu đang ở trạng thái đóng.
*   **Mô tả:** Gửi tin nhắn đến nhãn hàng. Nếu cuộc hội thoại hiện tại đang ở trạng thái closed (đã đóng), quy định của MVP cho phép tự động mở lại phòng chat khi khách gửi tin nhắn mới.
*   **Request Body:**
    ```json
    {
      "message": "Shop tư vấn giúp mình mẫu đầm này size M nhé"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin tin nhắn vừa tạo `BrandConversationMessageRes`.

### 3. Khách hàng đánh dấu đã đọc hội thoại
*   **Endpoint:** `POST /api/v1/brands/:brandId/conversation/read`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Cập nhật thông tin thời gian đọc cuộc hội thoại phía khách hàng.
*   **Mô tả:** Đánh dấu toàn bộ tin nhắn trong cuộc hội thoại với nhãn hàng đã được người dùng đọc.
*   **Response:**
    *   `200 OK`: Trả về thông tin phòng chat `BrandConversationRes`.

---

## Flow 2: Đội ngũ hỗ trợ của nhãn hàng xử lý hội thoại (Brand Portal Staff)

### 1. Lấy danh sách các cuộc hội thoại chat của thương hiệu
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/conversations`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Đọc danh sách các bản ghi cuộc hội thoại `brand_conversations` của nhãn hàng.
*   **Mô tả:** Nhân viên hỗ trợ chỉ có quyền xem các cuộc hội thoại chat thuộc thương hiệu mà mình được phân quyền quản lý.
*   **Response:**
    *   `200 OK`: Trả về mảng danh sách phòng chat `BrandConversationRes`.

### 2. Xem chi tiết lịch sử tin nhắn trong phòng chat cụ thể (phân trang)
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/conversations/:conversationId/messages`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Đọc danh sách tin nhắn `brand_conversation_messages`.
*   **Query Params:**
    *   `page` (optional): Trang hiện tại, mặc định `1`.
    *   `limit` (optional): Số tin nhắn mỗi trang, mặc định `20`.
*   **Mô tả:** Lấy lịch sử trò chuyện phân trang của một phòng chat thuộc nhãn hàng.
*   **Response:**
    *   `200 OK`: Trả về danh sách tin nhắn phân trang `BrandConversationMessageListRes` (`items` + `metadata`).

### 3. Nhân viên gửi tin nhắn phản hồi cho khách hàng
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/conversations/:conversationId/messages`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi tin nhắn mới `brand_conversation_messages`; cập nhật thời gian tin nhắn cuối cùng `lastMessageAt`.
*   **Mô tả:** Gửi tin nhắn trả lời khách hàng trong cuộc hội thoại thuộc thương hiệu quản lý.
*   **Request Body:**
    ```json
    {
      "message": "Chào bạn, mẫu đầm này size M hiện vẫn còn hàng sẵn tại cửa hàng ạ."
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin tin nhắn phản hồi `BrandConversationMessageRes`.

### 4. Nhân viên đánh dấu đã đọc hội thoại
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/conversations/:conversationId/read`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Cập nhật thông tin thời gian đọc cuộc hội thoại phía nhân viên.
*   **Mô tả:** Đánh dấu toàn bộ tin nhắn trong cuộc hội thoại đã được nhân viên đọc.
*   **Response:**
    *   `200 OK`: Trả về thông tin phòng chat `BrandConversationRes`.

### 5. Nhân viên đóng hội thoại
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/conversations/:conversationId/close`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái cuộc hội thoại `brand_conversations.status` sang `closed`.
*   **Mô tả:** Đóng cuộc hội thoại khi đã xử lý xong yêu cầu của khách hàng. Khách hàng vẫn có thể gửi tin nhắn mới để tự động mở lại.
*   **Response:**
    *   `200 OK`: Trả về thông tin phòng chat `BrandConversationRes`.

### 6. Nhân viên mở lại hội thoại
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/conversations/:conversationId/reopen`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái cuộc hội thoại `brand_conversations.status` sang `open`.
*   **Mô tả:** Mở lại cuộc hội thoại đã đóng để tiếp tục trao đổi.
*   **Response:**
    *   `200 OK`: Trả về thông tin phòng chat `BrandConversationRes`.

---

## Privacy Rules

*   Nhân viên hỗ trợ nhãn hàng tuyệt đối chỉ xem được các cuộc hội thoại và tin nhắn thuộc thương hiệu mà mình được phân quyền hoạt động.
*   Cuộc hội thoại chat bắt buộc phải liên kết với một tài khoản người dùng Closy đã đăng nhập; khách hàng offline chưa liên kết tài khoản không hỗ trợ mở kênh chat trực tuyến công khai.
*   Trường nội dung tin nhắn trong phản hồi bắt buộc sử dụng thuộc tính `message`; vai trò người gửi `senderRole` tham chiếu tại [constants/brand.md:SenderRole](constants/brand.md).
*   Hành động đọc (`read`) chỉ cập nhật thời gian đọc mới nhất, không thay đổi trạng thái cuộc hội thoại.
*   Hành động đóng (`close`) và mở lại (`reopen`) chỉ thay đổi trạng thái cuộc hội thoại, không ảnh hưởng đến lịch sử tin nhắn.