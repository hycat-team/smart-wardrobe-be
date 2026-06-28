# Kịch Bản Kiểm Thử & Xác Minh Nghiệp Vụ Demo B2B2C

Tài liệu này hướng dẫn cách chuẩn bị cơ sở dữ liệu và gọi tuần tự các API bằng công cụ như Postman, cURL, hoặc REST Client để kiểm thử và xác minh toàn diện 7 test cases nghiệp vụ cốt lõi của Phase 08.

---

## 1. Chuẩn bị Cơ sở dữ liệu (Database Setup)

Trước khi bắt đầu, hãy đảm bảo rằng bạn đã chạy các lệnh sau để khởi tạo database trống và áp dụng toàn bộ các migrations bao gồm cả dữ liệu seed demo B2B2C:

```bash
# Thực hiện rollback toàn bộ nếu có database cũ
make migration-down

# Thực hiện migrate up toàn bộ schema và dữ liệu seed
make migration-up
```

Dữ liệu seed chính bao gồm:
* **Tài khoản Staff (Brand Owner/Manager)**:
  * Username: `brandmanager` | Password: `123456`
  * ID: `11111111-1111-1111-1111-111111111112`
* **Tài khoản người dùng (B2C Users)**:
  * Hạng Bronze (B2C User 1): `bronzeuser` | Password: `123456` | ID: `22222222-2222-2222-2222-222222222221` (Spend: 0đ)
  * Hạng Gold (B2C User 2): `golduser` | Password: `123456` | ID: `22222222-2222-2222-2222-222222222222` (Spend: 5.000.000đ, có quyền `SAMPLE_MIX_ACCESS`)
* **Brand Demo**:
  * Closy Brand | ID: `33333333-3333-3333-3333-333333333333`
* **Khách hàng Offline (Chưa liên kết)**:
  * Name: `Offline Client` | ID: `55555555-5555-5555-5555-555555555553` | Phone: `+84999999999` (chưa có `user_id`)

---

## 2. Các Kịch Bản Xác Minh Chi Tiết (E2E Validation Cases)

Do hệ thống sử dụng cơ chế HttpOnly Cookie để lưu trữ token, đối với các request cURL dưới đây, chúng ta sẽ đính kèm header `Cookie: accessToken=<jwt_token>`.

### Đăng nhập để lấy accessToken
Trước mỗi kịch bản, bạn cần đăng nhập tài khoản tương ứng để nhận được `accessToken` trong cookie hoặc response:
```bash
# Đăng nhập tài khoản Brand Manager
curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username": "brandmanager", "password": "123456"}'
```

---

### Case 1: Tích lũy điểm offline (Offline Loyalty Acquisition)

**Mục tiêu**: Staff ghi nhận hóa đơn cho một số điện thoại khách hàng chưa đăng ký app, hệ thống tự tạo tài khoản loyalty offline mà không tạo thêm user mới.

**B1: Staff của Brand ghi nhận điểm cho khách hàng offline `+84999999999`**
```bash
# Sử dụng accessToken của brandmanager
curl -X POST http://localhost:8080/api/v1/brand-portal/brands/33333333-3333-3333-3333-333333333333/loyalty/points \
     -H "Cookie: accessToken=<manager_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "phoneE164": "+84999999999",
       "customerName": "Offline Client",
       "purchaseAmount": 200000.00,
       "reason": "Mua hàng trực tiếp tại quầy"
     }'
```
* **Expected Response**: Mã 200 OK.
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
* **Verify database**:
  * Kiểm tra bảng `brand_customers`: Bản ghi của `Offline Client` vẫn có `user_id IS NULL`.
  * Kiểm tra bảng `loyalty_accounts`: Số dư điểm tăng thêm 20, `user_id IS NULL`.
  * Không có user nào mới được tạo trong bảng `users`.

---

### Case 2: User liên kết tài khoản offline (User Claim Offline Account)

**Mục tiêu**: Người dùng online dùng mã claim do staff cấp để thu hồi/liên kết hồ sơ loyalty offline về tài khoản của mình.

**B1: Staff sinh mã claim cho khách hàng offline `Offline Client`**
```bash
# Sử dụng accessToken của brandmanager
curl -X POST http://localhost:8080/api/v1/brand-portal/brands/33333333-3333-3333-3333-333333333333/customers/55555555-5555-5555-5555-555555555553/claim-token \
     -H "Cookie: accessToken=<manager_token>"
```
* **Expected Response**: Mã 200 OK chứa raw token (ví dụ: `d8b5c92f-b4bb-4e89-be4d-616900f40cf9`).
```json
{
  "message": "Tạo mã claim liên kết tài khoản thành công",
  "data": {
    "claimToken": "d8b5c92f-b4bb-4e89-be4d-616900f40cf9",
    "expiresAt": "2026-06-30T00:00:00Z"
  }
}
```

**B2: Người dùng đăng nhập `bronzeuser` và gửi mã claim để liên kết**
```bash
# Đăng nhập bằng bronzeuser để lấy token
# Thực hiện gọi API Claim
curl -X POST http://localhost:8080/api/v1/brands/claim \
     -H "Cookie: accessToken=<bronzeuser_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "claimToken": "d8b5c92f-b4bb-4e89-be4d-616900f40cf9"
     }'
```
* **Expected Response**: Mã 200 OK.
```json
{
  "message": "Liên kết tài khoản loyalty thành công",
  "data": {
    "id": "55555555-5555-5555-5555-555555555553",
    "brandId": "33333333-3333-3333-3333-333333333333",
    "userId": "22222222-2222-2222-2222-222222222221",
    "customerName": "Offline Client",
    "status": "ACTIVE"
  }
}
```
* **Verify database & logic**:
  * Kiểm tra bảng `brand_customers`: Bản ghi `55555555-...` có `user_id` đã được cập nhật thành `22222222-2222-2222-2222-222222222221` (ID của `bronzeuser`), trường `claimed_at` được set thời gian hiện tại.
  * Kiểm tra bảng `loyalty_accounts`: Trường `user_id` được liên kết thành công với `bronzeuser`.
  * Kiểm tra bảng `brand_customer_claims`: Cột `consumed_at` được điền mốc thời gian hiện tại.

---

### Case 3: AI Phối đồ chỉ dùng tủ đồ cá nhân (Wardrobe-only AI Recommendation)

**Mục tiêu**: Gọi phối đồ không đính kèm sản phẩm của brand, đảm bảo phản hồi chỉ có đồ cá nhân.

```bash
# Đăng nhập bằng golduser
curl -X POST http://localhost:8080/api/v1/fashion/recommend \
     -H "Cookie: accessToken=<golduser_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "include_brand_items": false,
       "style_preference": "Casual",
       "event_context": "Dạo phố cuối tuần"
     }'
```
* **Expected Response**: Phản hồi chứa các phối đồ chỉ thuộc `USER_WARDROBE`.
```json
{
  "data": {
    "primary": {
      "items": [
        {
          "id": "ca7ca7ca-ca7c-ca7c-ca7c-ca7ca7ca7c02",
          "itemContext": "USER_WARDROBE"
        }
      ]
    }
  }
}
```

---

### Case 4: AI Phối đồ trộn sản phẩm Brand (Brand Item AI Recommendation)

**Mục tiêu**: Người dùng có quyền `SAMPLE_MIX_ACCESS` (Hạng Gold) sẽ nhận được gợi ý phối đồ trộn tối đa 30% sản phẩm của Brand (bao gồm cả sản phẩm thực tế `PRODUCT` và mẫu thử `SAMPLE`).

```bash
# Sử dụng accessToken của golduser (đã được seed hạng Gold và có quyền SAMPLE_MIX_ACCESS)
curl -X POST http://localhost:8080/api/v1/fashion/recommend \
     -H "Cookie: accessToken=<golduser_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "include_brand_items": true,
       "style_preference": "Casual",
       "event_context": "Dạo phố cuối tuần"
     }'
```
* **Expected Response**: Phản hồi chứa cả quần áo trong tủ đồ và sản phẩm brand.
* Kiểm tra thuộc tính của mỗi item:
  * Đồ cá nhân: `"itemContext": "USER_WARDROBE"`
  * Đồ thương hiệu: `"itemContext": "BRAND_ITEM"` và có object `"brandItem"` chứa `{ "id": "...", "brandName": "Closy Brand", "itemType": "SAMPLE", "name": "Mẫu thử Áo thun vàng Closy" }`.
* Đối với user `bronzeuser` (chưa có quyền `SAMPLE_MIX_ACCESS`), nếu gọi API tương tự, trong danh sách kết quả tuyệt đối sẽ không xuất hiện các món đồ có `"itemType": "SAMPLE"`.

---

### Case 5: Đổi điểm nhận quà (Benefit Redemption)

**Mục tiêu**: Thành viên dùng điểm loyalty tích lũy được để quy đổi quyền lợi ưu đãi tại brand.

**B1: Người dùng hạng Gold (`golduser`) đổi điểm nhận quà**
```bash
# Sử dụng accessToken của golduser (có sẵn 500 điểm)
curl -X POST http://localhost:8080/api/v1/brands/33333333-3333-3333-3333-333333333333/benefits/99999999-9999-9999-9999-999999999991/redeem \
     -H "Cookie: accessToken=<golduser_token>"
```
* **Expected Response**: Mã 200 OK (hoặc 201 Created).
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
* **Verify database**:
  * Kiểm tra số dư điểm trong `loyalty_accounts` của `golduser`: Điểm hiện tại cập nhật chính xác.
  * Bản ghi giao dịch điểm dạng `REDEEM` được chèn vào bảng `loyalty_point_transactions`.

---

### Case 6: Nhắn tin thương hiệu (Brand Chat)

**Mục tiêu**: Người dùng nhắn tin cho Brand, Staff của brand nhận được tin nhắn và phản hồi lại.

**B1: Người dùng gửi tin nhắn cho Closy Brand**
```bash
# Sử dụng accessToken của golduser
curl -X POST http://localhost:8080/api/v1/brands/33333333-3333-3333-3333-333333333333/conversation/messages \
     -H "Cookie: accessToken=<golduser_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "message": "Tôi muốn hỏi về mẫu thử áo thun vàng"
     }'
```

**B2: Staff của Brand xem danh sách hội thoại và trả lời tin nhắn**
```bash
# Sử dụng accessToken của brandmanager
# Lấy danh sách hội thoại của Brand
curl -X GET http://localhost:8080/api/v1/brand-portal/brands/33333333-3333-3333-3333-333333333333/conversations \
     -H "Cookie: accessToken=<manager_token>"

# Staff gửi tin nhắn trả lời vào hội thoại (ID hội thoại dddddddd-dddd-dddd-dddd-dddddddddddd)
curl -X POST http://localhost:8080/api/v1/brand-portal/brands/33333333-3333-3333-3333-333333333333/conversations/dddddddd-dddd-dddd-dddd-dddddddddddd/messages \
     -H "Cookie: accessToken=<manager_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "message": "Chào bạn, mẫu thử đó hiện khả dụng để bạn phối đồ thử rồi nhé!"
     }'
```
* **Expected**: Cả 2 phía đều xem được toàn bộ nội dung tin nhắn đã cập nhật, trường `last_message_at` của cuộc hội thoại được cập nhật.

---

### Case 7: Bảo mật dữ liệu (Privacy Validation)

**Mục tiêu**: Ngăn chặn rò rỉ dữ liệu nhạy cảm của người dùng (tủ đồ cá nhân, lịch sử chat riêng tư) cho nhân viên của Brand thông qua việc kiểm tra phân quyền chặt chẽ.

**B1: Gọi API Portal của Brand khác hoặc xem tủ đồ cá nhân thô của khách hàng**
```bash
# Staff cố tình lấy thông tin tủ đồ cá nhân thô của khách hàng (API không được cung cấp hoặc chặn quyền)
curl -X GET http://localhost:8080/api/v1/brand-portal/brands/33333333-3333-3333-3333-333333333333/customers/55555555-5555-5555-5555-555555555552/wardrobe \
     -H "Cookie: accessToken=<manager_token>"
```
* **Expected Response**: Mã 404 Not Found hoặc 403 Forbidden. Nhân viên thương hiệu không có quyền xem thông tin cá nhân thô nếu không qua luồng chia sẻ/phối đồ của AI được che giấu.
