# Loyalty & Membership API Specs

Tài liệu thiết kế các API liên quan đến chương trình loyalty, tích điểm thành viên, đặc quyền/voucher và cơ chế feature access trong mô hình B2B2C. Tất cả các giá trị hằng số sử dụng trong request/response tham chiếu chi tiết tại [constants/brand.md](constants/brand.md).

---

## Flow 1: Gia nhập và tích lũy điểm thành viên

### 1. Khách hàng đăng ký tham gia chương trình loyalty của nhãn hàng

- **Endpoint:** `POST /api/v1/brands/:brandId/join-loyalty`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Tạo bản ghi khách hàng nhãn hàng `brand_customers` và tài khoản tích điểm loyalty tương ứng cho người dùng hiện tại (current user) trong nhãn hàng.
- **Mô tả:** Đăng ký người dùng hiện tại trở thành khách hàng thân thiết của thương hiệu. Nguồn gốc tham gia `joinedSource` mặc định được hệ thống gán là `self_join` (tự tham gia).
- **Response:**
    - `201 Created`: Trả về thông tin khách hàng nhãn hàng `BrandCustomerRes`.

### 2. Nhân viên thực hiện cộng hoặc trừ điểm tích lũy cho khách hàng

- **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/loyalty/points`
- **Tác nhân (Actor):** Nhân viên hỗ trợ của nhãn hàng (Brand staff).
- **Đối tượng ảnh hưởng:** Tạo bản ghi lịch sử giao dịch điểm `loyalty_point_transactions`, cập nhật số dư điểm trong tài khoản loyalty và các lô điểm sử dụng (point lots).
- **Mô tả:** API hợp nhất cho phép nhân viên POS ghi nhận tích điểm dựa trên mã người dùng `userId`, số điện thoại `phone`, hoặc mã khách hàng bên ngoài `externalCustomerCode`. Nếu chỉ cung cấp số điện thoại hoặc mã bên ngoài, backend sẽ tự động tạo hoặc sử dụng lại hồ sơ offline `brand_customers` mà không tạo tài khoản `users` mới. Loại giao dịch `transactionType` tham chiếu [constants/brand.md:LoyaltyTransactionType](constants/brand.md).
- **Request Body:**
    ```json
    {
        "userId": "2c9164cb-1c61-44d1-b82e-4efbb5f4b111",
        "phone": "+84999999999",
        "customerName": "Nguyễn Văn A",
        "externalCustomerCode": "POS-CUS-001",
        "purchaseAmount": 1200000,
        "pointsDelta": 120,
        "transactionType": "earn",
        "reason": "Hóa đơn mua sắm HD12345",
        "referenceType": "pos_invoice",
        "referenceId": "e07f29f1-54b2-411b-8df5-0169c38b0111",
        "idempotencyKey": "pos-HD12345"
    }
    ```
- **Response:**
    - `201 Created`: Trả về kết quả giao dịch điểm thành viên `LoyaltyPointsTransactionRes`.

---

## Flow 2: Giao diện quản lý thẻ Loyalty của Khách hàng (Customer)

### 1. Xem danh sách tất cả thẻ thành viên của tôi

- **Endpoint:** `GET /api/v1/me/brand-loyalties`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc thông tin các tài khoản loyalty của người dùng hiện tại.
- **Mô tả:** Trả về danh sách tất cả các thẻ thành viên nhãn hàng mà người dùng đang tham gia, bao gồm số điểm hiện tại, điểm tích lũy trọn đời (lifetime points), tổng chi tiêu (total spend), hạng thành viên hiện tại (tier name) và lô điểm sắp hết hạn gần nhất.
- **Response:**
    - `200 OK`: Trả về mảng danh sách thẻ thành viên `BrandLoyaltyRes`.

### 2. Xem chi tiết thông tin thẻ thành viên theo thương hiệu

- **Endpoint:** `GET /api/v1/me/brand-loyalties/:brandId`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc tài khoản loyalty của người dùng hiện tại đối với nhãn hàng được chỉ định.
- **Mô tả:** Trả về thông tin thẻ chi tiết của current user tại một nhãn hàng cụ thể.
- **Response:**
    - `200 OK`: Trả về thông tin chi tiết thẻ `BrandLoyaltyRes`.

### 3. Xem lịch sử biến động điểm theo thương hiệu

- **Endpoint:** `GET /api/v1/me/brand-loyalties/:brandId/transactions`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc danh sách lịch sử giao dịch điểm `loyalty_point_transactions` của current user.
- **Mô tả:** Trả về danh sách lịch sử cộng/trừ điểm của người dùng tại nhãn hàng, bao gồm lượng điểm thay đổi `pointsDelta`, số dư sau giao dịch `balanceAfter`, loại giao dịch `transactionType`, lý do biến động `reason`, số tiền chi tiêu liên quan `spendAmount`, và ngày hết hạn của lô điểm `expiresAt`.
- **Response:**
    - `200 OK`: Trả về mảng lịch sử giao dịch điểm `LoyaltyPointTransactionDetailRes`.

---

## Flow 3: Quản trị chương trình Loyalty trong Brand Portal

### 1. Lấy thông tin cấu hình chương trình loyalty đang hoạt động

- **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/loyalty/program`
- **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
- **Đối tượng ảnh hưởng:** Đọc bảng dữ liệu chương trình loyalty `loyalty_programs`.
- **Mô tả:** Trả về cấu hình thiết lập quy tắc tích điểm hiện tại của nhãn hàng, bao gồm tỷ lệ đổi tiền sang điểm `amountPerPoint`, thời hạn hết hạn của điểm `pointExpiryDays`, cơ chế làm tròn `roundingMode`, và trạng thái hoạt động `isActive`.
- **Response:**
    - `200 OK`: Trả về cấu hình chương trình `LoyaltyProgramRes`.

### 2. Tạo và cập nhật cấu hình chương trình loyalty

- **Endpoint:** `PUT /api/v1/brand-portal/brands/:brandId/loyalty/program`
- **Tác nhân (Actor):** Chủ thương hiệu (Brand owner).
- **Đối tượng ảnh hưởng:** Tạo mới hoặc cập nhật bản ghi trong bảng dữ liệu chương trình loyalty `loyalty_programs`.
- **Mô tả:** API cho phép tạo hoặc cập nhật cấu hình thiết lập quy tắc tích điểm của nhãn hàng. Nếu chưa có chương trình loyalty, hệ thống sẽ tạo mới. Nếu đã có, hệ thống sẽ cập nhật thông tin hiện tại. `pointExpiryDays` là optional, nếu không truyền thì xem như điểm sẽ không hết hạn.
- **Request Body:**
    ```json
    {
        "name": "Chương trình thành viên thân thiết",
        "amountPerPoint": 10000,
        "pointExpiryDays": 365,
        "roundingMode": "floor",
        "isActive": true
    }
    ```
- **Response:**
    - `200 OK`: Trả về cấu hình chương trình `LoyaltyProgramRes`.

### 3. Lấy danh sách thiết lập các hạng thành viên của nhãn hàng

- **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/loyalty/tiers`
- **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
- **Đối tượng ảnh hưởng:** Đọc bảng thiết lập các hạng thành viên `loyalty_tiers`.
- **Mô tả:** Trả về thông tin các hạng thành viên đã cấu hình của nhãn hàng, bao gồm thứ tự xếp hạng `rank`, số tiền chi tiêu tích lũy tối thiểu yêu cầu `minTotalSpend`, và mô tả đặc quyền của hạng `description`.
- **Response:**
    - `200 OK`: Trả về mảng danh sách hạng thành viên `LoyaltyTierRes`.

### 4. Tạo hạng thành viên mới (Brand Owner)

- **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/loyalty/tiers`
- **Tác nhân (Actor):** Chủ thương hiệu (Brand owner).
- **Đối tượng ảnh hưởng:** Tạo bản ghi mới trong bảng `loyalty_tiers`.
- **Mô tả:** Tạo một hạng thành viên mới cho nhãn hàng. Tên hạng (`name`) và thứ tự hạng (`rank`) phải là duy nhất trong cùng một nhãn hàng. Hệ thống tự kiểm tra trùng lặp và trả về lỗi nếu đã tồn tại.
- **Request Body:**
    ```json
    {
        "name": "Gold",
        "rank": 3,
        "minTotalSpend": 5000000,
        "description": "Hạng Gold - Chi tiêu từ 5 triệu"
    }
    ```
- **Response:**
    - `201 Created`: Trả về thông tin hạng thành viên vừa tạo `LoyaltyTierRes`.

### 5. Cập nhật hạng thành viên (Brand Owner)

- **Endpoint:** `PUT /api/v1/brand-portal/brands/:brandId/loyalty/tiers/:tierId`
- **Tác nhân (Actor):** Chủ thương hiệu (Brand owner).
- **Đối tượng ảnh hưởng:** Cập nhật bản ghi trong bảng `loyalty_tiers`.
- **Mô tả:** Cập nhật thông tin hạng thành viên. Tất cả các trường đều là tùy chọn (optional), chỉ gửi những trường cần thay đổi. Hệ thống tự kiểm tra trùng lặp tên và thứ tự hạng (loại trừ chính hạng đang cập nhật).
- **Request Body:**
    ```json
    {
        "name": "Gold Plus",
        "minTotalSpend": 6000000
    }
    ```
- **Response:**
    - `200 OK`: Trả về thông tin hạng thành viên sau khi cập nhật `LoyaltyTierRes`.

### 6. Lấy lịch sử điểm của một tài khoản loyalty bất kỳ (phục vụ đối soát)

- **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/loyalty/accounts/:accountId/transactions`
- **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
- **Đối tượng ảnh hưởng:** Đọc các giao dịch điểm `loyalty_point_transactions` của nhãn hàng.
- **Mô tả:** Trả về toàn bộ lịch sử biến động điểm của một tài khoản loyalty cụ thể phục vụ tra cứu thông tin khi hỗ trợ khách hàng tại quầy POS.
- **Response:**
    - `200 OK`: Trả về mảng lịch sử giao dịch điểm `LoyaltyPointTransactionDetailRes`.

---

## Flow 4: Đặc quyền (Benefits), voucher và feature access

### 1. Khách hàng xem danh sách các đặc quyền đang hoạt động của nhãn hàng

- **Endpoint:** `GET /api/v1/brands/:brandId/benefits`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc danh sách các đặc quyền `brand_benefits` đang ở trạng thái active.
- **Mô tả:** Trả về danh sách các ưu đãi hoặc đặc quyền mở để người dùng đổi điểm. Loại đặc quyền `benefitType`, loại mở khóa `unlockType`, mã đặc quyền hệ thống `featureCode` và trạng thái `status` tham chiếu chi tiết tại [constants/brand.md](constants/brand.md).
- **Response:**
    - `200 OK`: Trả về mảng danh sách đặc quyền `BrandBenefitRes`.

### 2. Khách hàng xem thông tin chi tiết một đặc quyền đang active

- **Endpoint:** `GET /api/v1/brand-benefits/:benefitId`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc một bản ghi đặc quyền `brand_benefits` đang active.
- **Mô tả:** Trả về thông tin chi tiết và điều kiện áp dụng của một đặc quyền cụ thể theo `benefitId`. Backend tự xác định brand từ bản ghi `brand_benefits`, chỉ trả về khi user là khách hàng loyalty active của brand đó và đặc quyền đang ở trạng thái `active`.
- **Response:**
    - `200 OK`: Trả về thông tin đặc quyền `BrandBenefitRes`.

### 3. Khách hàng thực hiện quy đổi đặc quyền

- **Endpoint:** `POST /api/v1/brand-benefits/:benefitId/redeem`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Tạo bản ghi lượt đổi quà `benefit_redemptions`, thực hiện trừ số dư điểm trong tài khoản loyalty nếu đặc quyền yêu cầu điểm để quy đổi.
- **Mô tả:** Đổi mã giảm giá voucher hoặc kích hoạt đặc quyền truy cập (feature access) theo `benefitId`. Backend tự xác định brand từ bản ghi `brand_benefits`, kiểm tra user là khách hàng loyalty active của brand đó, rồi áp dụng điều kiện đổi bằng điểm tích lũy hoặc quyền lợi hạng thẻ thành viên (tier privilege). Trạng thái của lượt đổi `status` tham chiếu tại [constants/brand.md:BenefitRedemptionStatus](constants/brand.md#9-trang-thai-quy-doi-dac-quyen-benefitredemptionstatus).
- **Response:**
    - `201 Created`: Trả về thông tin kết quả quy đổi `BenefitRedemptionRes`.

### 4. Khách hàng xem danh sách đặc quyền / voucher đã quy đổi thành công

- **Endpoint:** `GET /api/v1/me/benefit-redemptions`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc danh sách lượt đổi đặc quyền `benefit_redemptions` của người dùng hiện tại.
- **Mô tả:** Trả về danh sách tất cả các vouchers, quà tặng hoặc đặc quyền mở khóa hệ thống mà người dùng hiện tại đang sở hữu.
- **Response:**
    - `200 OK`: Trả về mảng danh sách quà tặng đã đổi `BenefitRedemptionRes`.

### 5. Quản lý nhãn hàng tạo mới một đặc quyền / voucher

- **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/benefits`
- **Tác nhân (Actor):** Chủ thương hiệu hoặc quản lý nhãn hàng (Brand owner/staff).
- **Đối tượng ảnh hưởng:** Tạo bản ghi đặc quyền mới `brand_benefits`.
- **Mô tả:** Thiết lập đặc quyền, ưu đãi voucher hoặc mở khóa tính năng ảo trên Closy dành cho các hạng thẻ khách hàng của thương hiệu.
- **Request Body:**
    - Ví dụ benefit theo hạng thành viên (`unlockType = tier_privilege`):
    ```json
    {
        "name": "Quyền thử đồ mẫu cho hạng Gold",
        "description": "Cho phép khách hàng Gold trở lên thử phối đồ với sample của brand",
        "benefitType": "feature_access",
        "unlockType": "tier_privilege",
        "requiredTierId": "a98c9f80-0a15-4be4-8a48-f68cdbf5f111",
        "featureCode": "sample_mix_access",
        "featureConfig": {
            "validDurationDays": 30
        }
    }
    ```

    - Ví dụ benefit đổi bằng điểm (`unlockType = point_redemption`):
    ```json
    {
        "name": "Voucher 50k",
        "description": "Mã giảm giá trực tiếp cho khách hàng đổi bằng điểm loyalty",
        "benefitType": "voucher",
        "unlockType": "point_redemption",
        "requiredPoints": 100
    }
    ```
- **Ghi chú field:**
    - `unlockType = tier_privilege`: FE hiển thị và gửi `requiredTierId`; không hiển thị / không gửi `requiredPoints`. User có quyền khi tier hiện tại có `rank` lớn hơn hoặc bằng tier yêu cầu.
    - `unlockType = point_redemption`: FE hiển thị `requiredPoints`; không hiển thị / không gửi `requiredTierId`. Nếu gửi `requiredPoints = null`, `0` hoặc số âm thì backend lưu là `null` và benefit không yêu cầu điểm khi redeem.
    - `unlockType = manual_grant`: FE không cần hiển thị `requiredPoints` hoặc `requiredTierId`; benefit được brand trao thủ công.
    - `featureCode`: bắt buộc khi `benefitType = feature_access` và phải nằm trong [constants/brand.md:BenefitFeatureCode](constants/brand.md#8-ma-dac-quyen-he-thong-benefitfeaturecode).
    - `featureConfig`: JSON tuỳ chọn. Backend hiện chỉ đọc `validDurationDays` để tính `expiresAt` của lượt redeem; các key khác chỉ được lưu lại và trả về, chưa dùng để check quyền.
- **Response:**
    - `201 Created`: Trả về thông tin đặc quyền vừa tạo `BrandBenefitRes`.

### 6. Quản trị viên xem tất cả danh sách đặc quyền của nhãn hàng

- **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/benefits`
- **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
- **Đối tượng ảnh hưởng:** Đọc toàn bộ các đặc quyền `brand_benefits` thuộc nhãn hàng.
- **Mô tả:** Trả về danh sách toàn bộ các ưu đãi đã thiết kế bao gồm cả các bản nháp hoặc đặc quyền đã hết hạn/lưu trữ.
- **Response:**
    - `200 OK`: Trả về mảng danh sách đặc quyền `BrandBenefitRes`.

### 7. Cập nhật trạng thái hoạt động của một đặc quyền

- **Endpoint:** `PATCH /api/v1/brand-portal/brands/:brandId/benefits/:benefitId/status`
- **Tác nhân (Actor):** Chủ thương hiệu hoặc quản lý nhãn hàng (Brand owner/staff).
- **Đối tượng ảnh hưởng:** Cập nhật trạng thái hoạt động của đặc quyền `brand_benefits.status`.
- **Mô tả:** Kích hoạt hiển thị (active), tạm đóng (inactive) hoặc lưu trữ thu hồi (archived) một chương trình ưu đãi. Trạng thái `status` tham chiếu chi tiết tại [constants/brand.md:BenefitStatus](constants/brand.md).
- **Request Body:**
    ```json
    {
        "status": "active"
    }
    ```
- **Response:**
    - `200 OK`: Trả về thông tin đặc quyền sau khi đổi trạng thái `BrandBenefitRes`.

---

## Cập nhật MVP: API cộng điểm loyalty

Endpoint `POST /api/v1/brand-portal/brands/:brandId/loyalty/points` là API unified cho staff/POS ghi nhận điểm loyalty. Endpoint này hỗ trợ hai mode:

### 1. Purchase earn mode

Frontend/POS truyền `purchaseAmount`; backend tự đổi số tiền mua hàng thành điểm dựa trên loyalty program active của brand:

- `amountPerPoint`: số tiền tương ứng với một điểm.
- `roundingMode`: cách làm tròn điểm.
- `pointExpiryDays`: hạn dùng của lô điểm được tạo.

Ví dụ chỉ truyền số điện thoại và số tiền mua hàng:

```json
{
    "phone": "+84999999999",
    "customerName": "Nguyễn Văn A",
    "purchaseAmount": 1200000,
    "transactionType": "earn",
    "reason": "Hóa đơn mua sắm HD12345",
    "referenceType": "pos_invoice",
    "idempotencyKey": "pos-HD12345"
}
```

### 2. Manual adjustment mode

Frontend/POS truyền `pointsDelta`; backend dùng trực tiếp số điểm này. Mode này dành cho điều chỉnh thủ công, import dữ liệu, hoặc hệ thống POS đã tự tính điểm.

Ví dụ dùng mã khách hàng ngoài và điểm đã tính sẵn:

```json
{
    "externalCustomerCode": "POS-CUS-001",
    "pointsDelta": 120,
    "transactionType": "earn",
    "reason": "Import điểm từ POS",
    "idempotencyKey": "import-POS-CUS-001-001"
}
```

### Quy tắc request

- MVP hiện tại chỉ hỗ trợ `transactionType = "earn"` qua endpoint này; redeem và expire dùng flow riêng.
- Request phải có ít nhất một customer identifier: `userId`, `phone`, hoặc `externalCustomerCode`.
- Request có thể chỉ truyền `phone` làm customer identifier. Khi đó backend tìm khách theo `brand_id + phone_hash`; nếu chưa có thì tự tạo offline `brand_customers`, không tạo tài khoản `users`.
- Request phải có `purchaseAmount` hoặc `pointsDelta`.
- Nếu truyền cả `purchaseAmount` và `pointsDelta`, backend ưu tiên `pointsDelta`; frontend/POS nên chỉ truyền một trong hai để tránh hiểu nhầm.
- Nếu truyền nhiều customer identifier, backend ưu tiên theo thứ tự: `userId` > `phone` > `externalCustomerCode`.
