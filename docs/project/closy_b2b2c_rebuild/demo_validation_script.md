# Kịch Bản Kiểm Thử & Xác Minh Nghiệp Vụ Demo B2B2C Trên Swagger UI

Tài liệu này hướng dẫn cách thực hiện và xác minh toàn diện 7 kịch bản kiểm thử nghiệp vụ (E2E Validation Cases) của Phase 08 trực tiếp trên giao diện **Swagger UI** (`http://localhost:8080/swagger`).

---

## 1. Chuẩn bị Cơ sở dữ liệu (Database Setup)

Trước khi bắt đầu, hãy đảm bảo rằng bạn đã khởi tạo lại database và áp dụng đầy đủ dữ liệu seed demo B2B2C:

```bash
# Thực hiện rollback toàn bộ nếu có database cũ
make migration-down

# Thực hiện migrate up toàn bộ schema và dữ liệu seed
make migration-up
```

### Thông tin tài khoản và dữ liệu mẫu (Seed Data)
*   **Tài khoản Staff (Brand Owner/Manager)**:
    *   Username: `brandmanager` | Password: `123456`
    *   ID: `11111111-1111-1111-1111-111111111112`
*   **Tài khoản người dùng (B2C Users)**:
    *   Hạng Bronze (User 1): `bronzeuser` | Password: `123456` | ID: `22222222-2222-2222-2222-222222222221` (Spend: 0đ, đã là khách hàng của Closy Brand)
    *   Hạng Gold (User 2): `golduser` | Password: `123456` | ID: `22222222-2222-2222-2222-222222222222` (Spend: 5.000.000đ, đã được mở khóa quyền `SAMPLE_MIX_ACCESS`, đã là khách hàng của Closy Brand)
    *   **Claim User (User 3)**: `claimuser` | Password: `123456` | ID: `22222222-2222-2222-2222-222222222223` (User mới, **CHƯA** là khách hàng của brand → Dùng để test kịch bản claim)
*   **Thương hiệu (Brand Demo)**:
    *   Closy Brand | ID: `33333333-3333-3333-3333-333333333333`
*   **Khách hàng Offline (Chưa liên kết)**:
    *   Name: `Offline Client` | ID: `55555555-5555-5555-5555-555555555553` | Phone: `+84999999999` (chưa có `user_id`)

> **Lưu ý quan trọng**: `bronzeuser` và `golduser` đã được seed sẵn với tư cách là khách hàng của Closy Brand, nếu dùng 2 user này để claim tài khoản offline sẽ bị lỗi trùng unique constraint. Hãy dùng **`claimuser`** (User 3) cho kịch bản Case 2.

---

## 2. Lưu Ý Về Xác Thực Trên Swagger UI

Hệ thống sử dụng cơ chế **HttpOnly Cookie** (`accessToken`). Vì vậy:
1. Bạn chỉ cần gọi API đăng nhập (`POST /api/v1/auth/login`) trên giao diện Swagger UI.
2. Trình duyệt sẽ tự động nhận và lưu trữ Cookie này, sau đó tự động đính kèm vào mọi request gửi đi tiếp theo từ giao diện Swagger.
3. Không cần copy-paste hay điền bất cứ Token/Header thủ công nào trong Swagger UI.

---

## 3. Các Bước Thực Hiện Kiểm Thử Chi Tiết

### Đăng nhập tài khoản Brand Manager
Trước khi bắt đầu các case cần quyền Staff (Case 1, Case 2):
1. Tìm nhóm **Auth** -> Endpoint `POST /api/v1/auth/login`.
2. Click **Try it out**, nhập Body:
   ```json
   {
     "username": "brandmanager",
     "password": "123456"
   }
   ```
3. Click **Execute**. Xác nhận trả về `200 OK`.

---

### Case 1: Tích lũy điểm offline (Offline Loyalty Acquisition)
*   **Mục tiêu**: Staff ghi nhận hóa đơn cho một số điện thoại khách hàng chưa đăng ký app, hệ thống tự tạo tài khoản loyalty offline mà không tạo thêm user mới.

**Thực hiện trên Swagger**:
1. Đảm bảo đang đăng nhập với quyền `brandmanager`.
2. Tìm nhóm **Brand Portal** -> Endpoint `POST /api/v1/brand-portal/brands/{brandId}/loyalty/points`.
3. Click **Try it out** và điền các tham số:
   *   `brandId`: `33333333-3333-3333-3333-333333333333`
   *   Request Body:
       ```json
       {
         "phone": "+84999999999",
         "customerName": "Offline Client",
         "purchaseAmount": 200000.00,
         "transactionType": "EARN",
         "reason": "Mua hàng trực tiếp tại quầy"
       }
       ```
4. Click **Execute**.

**Kết quả mong đợi (Response)**: `200 OK`
```json
{
  "message": "Ghi nhận giao dịch điểm loyalty thành công",
  "data": {
    "pointsDelta": 20,
    "balanceAfter": 20,
    "transactionType": "EARN"
  }
}
```

---

### Case 2: User claim offline account
*   **Mục tiêu**: Người dùng online dùng mã claim do staff cấp để liên kết hồ sơ loyalty offline về tài khoản của mình.

**Bước 2.1: Staff tạo mã claim cho khách hàng offline**
1. Đảm bảo đang đăng nhập với quyền `brandmanager`.
2. Tìm nhóm **Brand Portal** -> Endpoint `POST /api/v1/brand-portal/brands/{brandId}/customers/{customerId}/claim-token`.
3. Click **Try it out** và điền tham số:
   *   `brandId`: `33333333-3333-3333-3333-333333333333`
   *   `customerId`: `55555555-5555-5555-5555-555555555553` (ID khách hàng offline)
4. Click **Execute**.
5. **Sao chép mã `claimToken`** nhận được ở response (Ví dụ: `d8b5c92f-b4bb-4e89-be4d-616900f40cf9`).

**Bước 2.2: Đăng xuất Staff và Đăng nhập User mới (claimuser)**

> ⚠ï¸ **Lưu ý**: Hãy dùng tài khoản **`claimuser`** — đây là user CHƯA có hồ sơ khách hàng tại Closy Brand. Nếu dùng `bronzeuser` hoặc `golduser` sẽ bị lỗi conflict vì 2 user này đã được seed sẵn là khách hàng.

1. Tìm nhóm **Auth** -> Endpoint `POST /api/v1/auth/logout`. Click **Try it out** -> **Execute** để xóa cookie Staff.
2. Tìm nhóm **Auth** -> Endpoint `POST /api/v1/auth/login`. Click **Try it out** và nhập thông tin `claimuser`:
   ```json
   {
     "username": "claimuser",
     "password": "123456"
   }
   ```
3. Click **Execute** để nhận cookie của `claimuser`.

**Bước 2.3: User gửi mã claim để thực hiện liên kết**
1. Tìm nhóm **Brand Customer** -> Endpoint `POST /api/v1/brands/claim`.
2. Click **Try it out** và nhập Body (dán mã token đã lấy ở Bước 2.1):
   ```json
   {
     "claimToken": "d8b5c92f-b4bb-4e89-be4d-616900f40cf9"
   }
   ```
3. Click **Execute**.

**Kết quả mong đợi (Response)**: `200 OK`
```json
{
  "message": "Liên kết tài khoản loyalty thành công",
  "data": {
    "id": "55555555-5555-5555-5555-555555555553",
    "brandId": "33333333-3333-3333-3333-333333333333",
    "userId": "22222222-2222-2222-2222-222222222223",
    "customerName": "Offline Client",
    "status": "ACTIVE"
  }
}
```

---

### Case 3: AI Phối đồ chỉ dùng tủ đồ cá nhân (Wardrobe-only AI Recommendation)
*   **Mục tiêu**: Gọi phối đồ không đính kèm sản phẩm của brand, đảm bảo phản hồi chỉ có đồ cá nhân.

**Thực hiện trên Swagger**:
1. Đảm bảo đang đăng nhập với quyền `golduser`.
2. Tìm nhóm **Wardrobe AI** -> Endpoint `POST /api/v1/ai/outfit-recommendations`.
3. Click **Try it out** và nhập Body:
   ```json
   {
     "include_brand_items": false,
     "styleTarget": "Casual",
     "occasion": "Dạo phố cuối tuần"
   }
   ```
4. Click **Execute**.

**Kết quả mong đợi (Response)**: Phản hồi chứa các nhóm phối đồ có `"itemContext": "USER_WARDROBE"`. Không được có bất kỳ `"itemContext": "BRAND_ITEM"` nào.

---

### Case 4: AI Phối đồ trộn sản phẩm Brand (Brand Item AI Recommendation)
*   **Mục tiêu**: Người dùng hạng Gold sẽ nhận được gợi ý phối đồ trộn tối đa 30% sản phẩm của Brand (bao gồm cả sản phẩm thực tế `PRODUCT` và mẫu thử `SAMPLE`).

**Thực hiện trên Swagger**:
1. Đảm bảo đang đăng nhập với quyền `golduser`.
2. Tìm nhóm **Wardrobe AI** -> Endpoint `POST /api/v1/ai/outfit-recommendations`.
3. Click **Try it out** và nhập Body:
   ```json
   {
     "include_brand_items": true,
     "styleTarget": "Casual",
     "occasion": "Dạo phố cuối tuần"
   }
   ```
4. Click **Execute**.

**Kết quả mong đợi (Response)**:
*   Trong nhóm phối đồ xuất hiện các item có `"itemContext": "BRAND_ITEM"`.
*   Có đi kèm object `"brandItem"` hiển thị đầy đủ thông tin mẫu thử hoặc sản phẩm (ví dụ: `Mẫu thử Áo thun vàng Closy` - `SAMPLE`).

---

### Case 5: Nhận quyền lợi hạng thành viên (Benefit Redemption)
*   **Mục tiêu**: Thành viên hạng Gold redeem quyền lợi `SAMPLE_MIX_ACCESS` theo tier privilege của brand. Quyền lợi này không trừ điểm, nên `pointsSpent = 0`.

**Thực hiện trên Swagger**:
1. Đảm bảo đang đăng nhập với quyền `golduser` (đang ở hạng Gold và có sẵn 500 điểm seed để phục vụ các luồng loyalty khác).
2. Tìm nhóm **Brand Customer** (hoặc **Brand**) -> Endpoint `POST /api/v1/brands/{brandId}/benefits/{benefitId}/redeem`.
3. Click **Try it out** và điền tham số:
   *   `brandId`: `33333333-3333-3333-3333-333333333333`
   *   `benefitId`: `99999999-9999-9999-9999-999999999991` (ID quyền lợi Gold)
4. Click **Execute**.

**Kết quả mong đợi (Response)**: `200 OK` (hoặc `201 Created`)
```json
{
  "message": "Đổi quyền lợi thành công",
  "data": {
    "benefitId": "99999999-9999-9999-9999-999999999991",
    "pointsSpent": 0,
    "status": "REDEEMED"
  }
}
```

---

### Case 6: Nhắn tin thương hiệu (Brand Chat)
*   **Mục tiêu**: Người dùng nhắn tin cho Brand, Staff của brand nhận được tin nhắn và phản hồi lại.

**B1: Người dùng (Gold User) nhắn tin cho Closy Brand**
1. Đảm bảo đang đăng nhập với quyền `golduser`.
2. Tìm nhóm **Brand Customer** -> Endpoint `POST /api/v1/brands/{brandId}/conversation/messages`.
3. Click **Try it out** và điền tham số:
   *   `brandId`: `33333333-3333-3333-3333-333333333333`
   *   Request Body:
       ```json
       {
         "message": "Tôi muốn hỏi về mẫu thử áo thun vàng"
       }
       ```
4. Click **Execute**.

**B2: Staff của Brand xem danh sách hội thoại và trả lời tin nhắn**
1. Đăng xuất và đăng nhập lại bằng tài khoản `brandmanager` (Lặp lại Bước 4 & 5 ở trên).
2. Tìm nhóm **Brand Portal** -> Endpoint `GET /api/v1/brand-portal/brands/{brandId}/conversations`. Điền `brandId`: `33333333-3333-3333-3333-333333333333`. Click **Execute** để lấy danh sách hội thoại.
3. Tìm nhóm **Brand Portal** -> Endpoint `POST /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/messages`.
4. Click **Try it out** và điền:
   *   `brandId`: `33333333-3333-3333-3333-333333333333`
   *   `conversationId`: `dddddddd-dddd-dddd-dddd-dddddddddddd` (ID cuộc hội thoại mẫu)
   *   Request Body:
       ```json
       {
         "message": "Chào bạn, mẫu thử đó hiện khả dụng để bạn phối đồ thử rồi nhé!"
       }
       ```
5. Click **Execute**. Xác nhận tin nhắn được thêm vào và hiển thị đúng vai trò của staff.

---

### Case 7: Bảo mật dữ liệu (Privacy Validation)
*   **Mục tiêu**: Ngăn chặn rò rỉ dữ liệu tủ đồ cá nhân thô hoặc lịch sử chat riêng tư của người dùng cho nhân viên Brand.

**Thực hiện trên Swagger**:
1. Đảm bảo đang đăng nhập với quyền `brandmanager`.
2. Thử truy cập một endpoint không được thiết kế cho staff hoặc vi phạm quyền (ví dụ: cố lấy tủ đồ cá nhân thô của khách hàng):
   *   Sử dụng bất kỳ API xem tủ đồ cá nhân hoặc gọi API không có trong Portal.
3. Click **Execute**.
4. **Kết quả mong đợi**: Mã lỗi `403 Forbidden` hoặc `404 Not Found`. Hệ thống chặn hoàn toàn mọi truy cập trái phép.
