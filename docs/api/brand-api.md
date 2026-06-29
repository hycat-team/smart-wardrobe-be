# Brand Management API Specs

Tài liệu thiết kế API liên quan đến hồ sơ nhãn hàng (brand profile), cổng đối tác (Brand Portal), thành viên, xét duyệt của Admin (Admin approval) và claim tài khoản khách offline trong mô hình B2B2C. Tất cả các trường dữ liệu JSON sử dụng camelCase theo đúng [API Guidelines](api-guidelines.md).

---

## Flow 1: Tìm kiếm và xem thông tin Brand (Khách hàng / Customer)

### 1. Lấy danh sách brand đang hoạt động (active)
*   **Endpoint:** `GET /api/v1/brands`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc danh sách dữ liệu các nhãn hàng (`brands`) có trạng thái active.
*   **Mô tả:** Trả về các thương hiệu công khai (brand public) đang hoạt động để người dùng có thể duyệt xem, đăng ký tham gia chương trình loyalty, xem các đặc quyền (benefits) và danh sách sản phẩm. Trạng thái `status` tham chiếu [constants/brand.md:BrandStatus](constants/brand.md#1-trang-thai-cua-brand-brandstatus).
*   **Response:**
    *   `200 OK`:
        ```json
        {
          "success": true,
          "data": [
            {
              "id": "787c9f80-0a15-4be4-8a48-f68cdbf5f154",
              "slug": "local-brand-a",
              "name": "Local Brand A",
              "description": "Thương hiệu thời trang nội địa",
              "logoUrl": "https://res.cloudinary.com/.../logo.png",
              "logoPublicId": "brands/local-brand-a/logo",
              "status": "active"
            }
          ]
        }
        ```

### 2. Xem chi tiết brand hoạt động (active)
*   **Endpoint:** `GET /api/v1/brands/:brandId`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc thông tin của một nhãn hàng (`brands`) đang active.
*   **Mô tả:** Trả về hồ sơ công khai (profile public) của thương hiệu. Endpoint chỉ trả về thông tin đối với các brand đang active.
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết nhãn hàng `BrandRes`.

---

## Flow 2: Đăng ký và xét duyệt đối tác nhãn hàng (Brand Portal & Admin)

### 1. Gửi yêu cầu đăng ký tạo brand mới
*   **Endpoint:** `POST /api/v1/brand-portal/brands`
*   **Tác nhân (Actor):** Người dùng muốn trở thành chủ sở hữu nhãn hàng (brand owner).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi thương hiệu (`brands`) ở trạng thái chờ duyệt và gán quyền chủ thương hiệu (brand member owner) tương ứng theo use case backend.
*   **Mô tả:** Gửi hồ sơ đăng ký nhãn hàng để Admin xem xét phê duyệt. Trạng thái `status` tham chiếu [constants/brand.md:BrandStatus](constants/brand.md#1-trang-thai-cua-brand-brandstatus).
*   **Request Body:**
    ```json
    {
      "slug": "local-brand-b",
      "name": "Local Brand B",
      "description": "Chuyên trang phục streetwear nam nữ",
      "logoUrl": "https://res.cloudinary.com/.../logo.png",
      "logoPublicId": "brands/local-brand-b/logo"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về kết quả khởi tạo nhãn hàng `BrandRes`.

### 2. Lấy danh sách thương hiệu của người dùng hiện tại trong Portal
*   **Endpoint:** `GET /api/v1/brand-portal/me/brands`
*   **Tác nhân (Actor):** Người dùng của Brand Portal.
*   **Đối tượng ảnh hưởng:** Đọc danh sách thương hiệu mà người dùng hiện tại (current user) đang là thành viên hoạt động (active member).
*   **Mô tả:** Sử dụng cho giao diện chuyển đổi thương hiệu (brand switcher) trên portal.
*   **Response:**
    *   `200 OK`: Trả về mảng danh sách thương hiệu `BrandRes`.

### 3. Admin cập nhật trạng thái của thương hiệu
*   **Endpoint:** `PATCH /api/v1/admin/brands/:brandId/status`
*   **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
*   **Đối tượng ảnh hưởng:** Cập nhật trường trạng thái thương hiệu `brands.status`.
*   **Mô tả:** Duyệt thương hiệu hoạt động bằng cách đổi trạng thái sang `active`, hoặc tạm ngưng hoạt động sang `suspended`. Trạng thái `status` tham chiếu [constants/brand.md:BrandStatus](constants/brand.md#1-trang-thai-cua-brand-brandstatus).
*   **Request Body:**
    ```json
    {
      "status": "active"
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin thương hiệu sau cập nhật `BrandRes`.

### 4. Admin tạo trực tiếp thương hiệu ở trạng thái active
*   **Endpoint:** `POST /api/v1/admin/brands`
*   **Tác nhân (Actor):** Quản trị viên hệ thống (Admin Closy).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi thương hiệu `brands` ở trạng thái active trực tiếp.
*   **Mô tả:** Phục vụ cho mục đích vận hành nội bộ, seed dữ liệu demo, hoặc đối tác đặc biệt đã phê duyệt trực tiếp.
*   **Request Body:** Tương tự cấu trúc `POST /api/v1/brand-portal/brands`.
*   **Response:**
    *   `201 Created`: Trả về thông tin thương hiệu `BrandRes`.

---

## Flow 3: Quản lý hồ sơ và thành viên thương hiệu (Brand Portal)

### 1. Xem thông tin chi tiết nhãn hàng trong Portal
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId`
*   **Tác nhân (Actor):** Thành viên nhãn hàng (Brand member) hoặc Admin Closy.
*   **Đối tượng ảnh hưởng:** Đọc thông tin hồ sơ của brand trong phạm vi quyền quản trị.
*   **Mô tả:** Trả về thông tin chi tiết của brand phục vụ trang Dashboard quản trị. Chỉ có thành viên active của nhãn hàng hoặc Admin mới có quyền truy cập.
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết `BrandRes`.

### 2. Lấy chữ ký tải lên ảnh logo của thương hiệu
*   **Endpoint:** `GET /api/v1/brand-portal/brands/logo-upload-signature`
*   **Tác nhân (Actor):** Người dùng cổng Brand Portal.
*   **Đối tượng ảnh hưởng:** Không thay đổi dữ liệu nghiệp vụ.
*   **Mô tả:** Lấy mã chữ ký Cloudinary upload signature để phía client có thể tải logo trực tiếp lên Cloudinary.
*   **Response:**
    *   `200 OK`: Trả về kết quả chữ ký `UploadSignatureResult`.

### 3. Cập nhật logo mới của thương hiệu
*   **Endpoint:** `PATCH /api/v1/brand-portal/brands/:brandId/logo`
*   **Tác nhân (Actor):** Chủ nhãn hàng hoặc quản lý nhãn hàng (Brand owner/manager).
*   **Đối tượng ảnh hưởng:** Cập nhật đường dẫn logo của brand.
*   **Mô tả:** Lưu lại đường dẫn URL và mã nhận diện public ID của logo mới đã tải lên thành công.
*   **Request Body:**
    ```json
    {
      "logoUrl": "https://res.cloudinary.com/.../new-logo.png",
      "logoPublicId": "brands/local-brand-a/new-logo"
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin thương hiệu sau cập nhật `BrandRes`.

### 4. Thêm thành viên mới vào thương hiệu
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/members`
*   **Tác nhân (Actor):** Chủ thương hiệu (Brand owner).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi thành viên `brand_members`.
*   **Mô tả:** Thêm một tài khoản user đã tồn tại vào danh sách thành viên quản lý thương hiệu với vai trò cụ thể. Vai trò `role` tham chiếu [constants/brand.md:BrandMemberRole](constants/brand.md#2-vai-tro-thanh-vien-trong-brand-brandmemberrole).
*   **Request Body:**
    ```json
    {
      "userId": "2c9164cb-1c61-44d1-b82e-4efbb5f4b111",
      "role": "support_staff"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin thành viên vừa thêm `BrandMemberRes`.

### 5. Lấy danh sách thành viên quản lý thương hiệu
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/members`
*   **Tác nhân (Actor):** Chủ thương hiệu hoặc quản lý nhãn hàng (Brand owner/manager).
*   **Đối tượng ảnh hưởng:** Đọc danh sách bản ghi thành viên `brand_members`.
*   **Mô tả:** Trả về danh sách tất cả các thành viên quản trị của nhãn hàng. Vai trò `role` và trạng thái `status` tham chiếu [constants/brand.md](constants/brand.md).
*   **Response:**
    *   `200 OK`: Trả về mảng danh sách thành viên `BrandMemberRes`.

---

## Flow 4: Khách hàng Offline và quy trình Claim tài khoản (B2B2C)

Khách hàng mua hàng trực tiếp tại cửa hàng được nhân viên POS ghi nhận thông tin vào hệ thống `brand_customers` mà không tạo tài khoản người dùng Closy (`users`). Khi khách hàng chủ động cài đặt và đăng nhập app Closy, họ có thể liên kết (link) với hồ sơ lịch sử offline cũ thông qua claim token hoặc quét mã QR.

### 1. Tạo hồ sơ khách hàng offline từ giao dịch tại POS cửa hàng
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/customers/offline-purchase`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Tạo mới hoặc sử dụng lại bản ghi khách hàng thương hiệu `brand_customers`; khởi tạo tài khoản điểm loyalty tương ứng.
*   **Mô tả:** Ghi nhận thông tin khách hàng mua sắm offline để tích điểm mà không tạo mới tài khoản `users`. Khách offline ban đầu sẽ có trường `userId = null` cho đến khi họ thực hiện liên kết tài khoản thành công. Trường `joinedSource` và trạng thái `status` tham chiếu [constants/brand.md](constants/brand.md).
*   **Request Body:**
    ```json
    {
      "customerName": "Nguyễn Văn A",
      "phoneE164": "+84999999999",
      "externalCustomerCode": "POS-CUS-001"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin khách hàng nhãn hàng `BrandCustomerRes`.

### 2. Lấy danh sách khách hàng của nhãn hàng
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/customers`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Đọc danh sách bản ghi khách hàng `brand_customers`.
*   **Mô tả:** Trả về toàn bộ danh sách khách hàng bao gồm cả khách đã liên kết tài khoản và khách hàng offline chưa liên kết. Nhân viên chỉ xem được danh sách khách hàng thuộc thương hiệu mình được phân quyền.
*   **Response:**
    *   `200 OK`: Trả về mảng danh sách khách hàng `BrandCustomerRes`.

### 3. Lấy thông tin chi tiết một khách hàng của nhãn hàng
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/customers/:customerId`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Đọc một bản ghi khách hàng `brand_customer`.
*   **Mô tả:** Trả về chi tiết thông tin và hạng thẻ thành viên của khách hàng trong phạm vi nhãn hàng.
*   **Response:**
    *   `200 OK`: Trả về chi tiết `BrandCustomerRes`.

### 4. Tạo mã liên kết (claim token) cho khách hàng offline
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/customers/:customerId/claim-token`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi mã liên kết `brand_customer_claims`.
*   **Mô tả:** Tạo mã claim token dạng thô cho khách offline để in hóa đơn hoặc gửi SMS. Phía backend chỉ lưu chuỗi băm (hash token) để bảo mật; mã thô chỉ trả về duy nhất một lần ở response. Thời gian hết hạn mặc định là 24 giờ.
*   **Response:**
    *   `200 OK`:
        ```json
        {
          "success": true,
          "data": {
            "claimToken": "raw-claim-token-returned-once",
            "expiresAt": "2026-06-30T10:00:00Z"
          }
        }
        ```

### 5. Khách hàng thực hiện liên kết (claim) hồ sơ offline vào tài khoản Closy
*   **Endpoint:** `POST /api/v1/brands/claim`
*   **Tác nhân (Actor):** Khách hàng đã đăng nhập app Closy (Customer).
*   **Đối tượng ảnh hưởng:** Cập nhật liên kết các trường `brand_customers.user_id`, `brand_customers.claimed_at`, `loyalty_accounts.user_id` và cập nhật mã claim đã dùng `brand_customer_claims.consumed_at`.
*   **Mô tả:** Người dùng nhập mã claim hoặc quét QR từ hóa đơn. Hệ thống sẽ băm mã token, tìm bản ghi claim hợp lệ, chưa hết hạn và chưa sử dụng, sau đó liên kết toàn bộ lịch sử điểm offline vào tài khoản Closy của người dùng hiện tại trong một transaction an toàn.
*   **Request Body:**
    ```json
    {
      "claimToken": "token-from-brand-staff"
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin khách hàng thương hiệu sau liên kết `BrandCustomerRes`.

### Quy tắc bắt buộc
*   Tuyệt đối không tự động tạo tài khoản user hệ thống (`users`) đối với khách hàng mua offline.
*   Khách hàng offline chưa thực hiện liên kết bắt buộc phải trả về giá trị `userId = null` trong các DTO.
*   Mọi luồng liên kết tài khoản offline chỉ hỗ trợ qua mã claim token hoặc mã QR trên hóa đơn, phiên bản MVP không sử dụng OTP gửi qua số điện thoại để link tự động.
*   Nhân viên của nhãn hàng chỉ được quyền truy vấn thông tin khách hàng và tạo mã claim thuộc nhãn hàng của mình quản lý.
