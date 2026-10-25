# Loyalty & Membership API Specs

Tài liệu thiết kế các API liên quan đến chương trình loyalty, tích điểm thành viên, đặc quyền/voucher và cơ chế feature access trong mô hình B2B2C. Tất cả các giá trị hằng số sử dụng trong request/response tham chiếu chi tiết tại [constants.md](constants.md). Ma trận phân quyền owner/staff của brand portal tham chiếu tại [Brand Permission Matrix](../../../system-design/brand-permission-matrix.md).

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
- **Mô tả:** API hợp nhất cho phép nhân viên POS ghi nhận tích điểm dựa trên mã người dùng `userId`, số điện thoại `phone`, hoặc mã khách hàng bên ngoài `externalCustomerCode`. Nếu chỉ cung cấp số điện thoại hoặc mã bên ngoài, backend sẽ tự động tạo hoặc sử dụng lại hồ sơ offline `brand_customers` mà không tạo tài khoản `users` mới. Loại giao dịch `transactionType` tham chiếu [constants.md:LoyaltyTransactionType](constants.md#17-loai-giao-dich-diem-loyalty-loyaltytransactiontype).
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

### 6. Quản trị viên xem chi tiết hạng thành viên kèm danh sách đặc quyền

- **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/loyalty/tiers/:tierId`
- **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
- **Đối tượng ảnh hưởng:** Đọc thông tin hạng thành viên `loyalty_tiers` và các đặc quyền `brand_benefits` gắn với hạng đó.
- **Mô tả:** Trả về thông tin chi tiết của một hạng thành viên kèm danh sách các đặc quyền đang active mà yêu cầu hạng này (`unlockType = tier_privilege` và `requiredTierId` trùng với `tierId`).
- **Response:**
    - `200 OK`: Trả về thông tin hạng `LoyaltyTierDetailRes`.
    ```json
    {
        "data": {
            "id": "...",
            "brandId": "...",
            "name": "Gold",
            "rank": 3,
            "minTotalSpend": 5000000,
            "description": "Hạng Gold - Chi tiêu từ 5 triệu",
            "createdAt": "2026-08-01T00:00:00Z",
            "updatedAt": "2026-08-01T00:00:00Z",
            "benefits": [
                {
                    "id": "...",
                    "name": "Quyền thử đồ mẫu cho hạng Gold",
                    "benefitType": "feature_access",
                    "unlockType": "tier_privilege",
                    "requiredPoints": null,
                    "requiredTierId": "...",
                    "requiredTierName": "Gold",
                    "featureCode": "sample_mix_access",
                    "featureConfig": {"validDurationDays": 30},
                    "status": "active"
                }
            ]
        }
    }
    ```

### 7. Khách hàng xem chi tiết hạng thành viên kèm danh sách đặc quyền

- **Endpoint:** `GET /api/v1/brands/:brandId/loyalty/tiers/:tierId`
- **Tác nhân (Actor):** Khách hàng (Customer) hoặc công khai (Public).
- **Đối tượng ảnh hưởng:** Đọc thông tin hạng thành viên `loyalty_tiers` và các đặc quyền `brand_benefits` gắn với hạng đó.
- **Mô tả:** Tương tự endpoint staff (Item 6), nhưng không yêu cầu quyền brand portal. Kiểm tra brand đang active trước khi trả về.
- **Response:**
    - `200 OK`: Trả về thông tin hạng `LoyaltyTierDetailRes` (cùng format với Item 6).

### 8. Lấy lịch sử điểm của một tài khoản loyalty bất kỳ (phục vụ đối soát)

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
- **Mô tả:** Trả về danh sách các ưu đãi hoặc đặc quyền mở để người dùng đổi điểm. Hỗ trợ phân trang và lọc theo `unlockType` hoặc `benefitType`. Loại đặc quyền `benefitType`, loại mở khóa `unlockType`, mã đặc quyền hệ thống `featureCode` và trạng thái `status` tham chiếu chi tiết tại [constants.md](constants.md).
- **Query Params:**

    | Param | Type | Bắt buộc | Mặc định | Mô tả |
    |---|---|---|---|---|
    | `page` | int | Không | `1` | Số trang |
    | `limit` | int | Không | `20` | Số item mỗi trang (tối đa 100) |
    | `unlockType` | string | Không | - | Lọc theo loại mở khóa: `tier_privilege`, `point_redemption`, `manual_grant` |
    | `benefitType` | string | Không | - | Lọc theo loại quyền lợi: `voucher`, `discount`, `gift`, `free_shipping`, `early_access`, `feature_access` |

- **Response:**
    - `200 OK`: Trả về danh sách đặc quyền phân trang `BrandBenefitListRes`.
    ```json
    {
        "data": {
            "items": [
                {
                    "id": "...",
                    "brandId": "...",
                    "name": "Giảm 10%",
                    "description": "Giảm 10% cho đơn hàng tiếp theo",
                    "benefitType": "voucher",
                    "unlockType": "point_redemption",
                    "requiredPoints": 500,
                    "requiredTierId": null,
                    "requiredTierName": null,
                    "featureCode": null,
                    "featureConfig": null,
                    "status": "active",
                    "createdAt": "2026-08-01T00:00:00Z",
                    "updatedAt": "2026-08-01T00:00:00Z"
                }
            ],
            "metadata": {
                "page": 1,
                "limit": 20,
                "totalItems": 5,
                "totalPages": 1
            }
        }
    }
    ```

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
- **Mô tả:** Đổi mã giảm giá voucher hoặc kích hoạt đặc quyền truy cập (feature access) theo `benefitId`. Backend tự xác định brand từ bản ghi `brand_benefits`, kiểm tra user là khách hàng loyalty active của brand đó, rồi áp dụng điều kiện đổi bằng điểm tích lũy hoặc quyền lợi hạng thẻ thành viên (tier privilege). Trạng thái của lượt đổi `status` tham chiếu tại [constants.md:BenefitRedemptionStatus](constants.md#9-trang-thai-quy-doi-dac-quyen-benefitredemptionstatus).
- **Response:**
    - `201 Created`: Trả về thông tin kết quả quy đổi `BenefitRedemptionRes`.

### 4. Khách hàng xem danh sách đặc quyền / voucher đã quy đổi thành công

- **Endpoint:** `GET /api/v1/me/benefit-redemptions`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc danh sách lượt đổi đặc quyền `benefit_redemptions` của người dùng hiện tại.
- **Mô tả:** Trả về danh sách tất cả các vouchers, quà tặng hoặc đặc quyền mở khóa hệ thống mà người dùng hiện tại đang sở hữu.
- **Response:**
    - `200 OK`: Trả về mảng danh sách quà tặng đã đổi `BenefitRedemptionRes`.

### 5. Khách hàng xem danh sách quyền lợi của tôi kèm trạng thái

- **Endpoint:** `GET /api/v1/me/brand-loyalties/:brandId/benefits`
- **Tác nhân (Actor):** Khách hàng (Customer).
- **Đối tượng ảnh hưởng:** Đọc các đặc quyền `brand_benefits` đang active của brand, hạng hiện tại của khách hàng và các lượt đổi `benefit_redemptions` của người dùng hiện tại.
- **Mô tả:** Trả về toàn bộ quyền lợi đang active của brand kèm trạng thái mở khoá theo hạng thành viên hoặc điểm hiện có, cùng lượt đổi gần nhất của người dùng. Đây là endpoint dành cho tab "Quyền lợi của tôi" ở phía FE; `GET /api/v1/me/benefit-redemptions` chỉ trả về lịch sử đã đổi.
- **Query Params:**

    | Param | Type | Bắt buộc | Mặc định | Mô tả |
    |---|---|---|---|---|
    | `page` | int | Không | `1` | Số trang |
    | `limit` | int | Không | `20` | Số item mỗi trang (tối đa 100) |
    | `unlockType` | string | Không | - | Lọc theo loại mở khoá: `tier_privilege`, `point_redemption`, `manual_grant` |
    | `benefitType` | string | Không | - | Lọc theo loại quyền lợi: `voucher`, `discount`, `gift`, `free_shipping`, `early_access`, `feature_access` |

- **Response:**
    - `200 OK`: Trả về `MyBrandBenefitListRes`.
    ```json
    {
        "data": {
            "currentTier": {
                "id": "fbf10001-0000-0000-0000-000000000002",
                "brandId": "e7b10001-0000-0000-0000-000000000001",
                "name": "Gold",
                "rank": 2,
                "minTotalSpend": 2000000,
                "description": "Hạng Gold cho khách hàng chi tiêu từ 2 triệu.",
                "createdAt": "2026-07-24T06:34:45Z",
                "updatedAt": "2026-07-24T06:38:15Z"
            },
            "currentPoints": 500,
            "items": [
                {
                    "id": "feed0009-0000-0000-0000-000000000001",
                    "brandId": "e7b10001-0000-0000-0000-000000000001",
                    "name": "Sample Mix Access",
                    "description": "Allows user to try digital samples in AI outfit recommendations.",
                    "benefitType": "feature_access",
                    "unlockType": "tier_privilege",
                    "requiredPoints": null,
                    "requiredTierId": "fbf10001-0000-0000-0000-000000000002",
                    "requiredTierName": "Gold",
                    "featureCode": "sample_mix_access",
                    "featureConfig": {"source": "feedback_seed"},
                    "status": "active",
                    "createdAt": "2026-07-24T06:34:45Z",
                    "updatedAt": "2026-07-24T06:38:15Z",
                    "unlocked": true,
                    "redemption": null
                }
            ],
            "metadata": {
                "page": 1,
                "limit": 20,
                "totalItems": 1,
                "totalPages": 1
            }
        }
    }
    ```
- **Quy tắc `unlocked`:**
    - `tier_privilege`: `true` khi `currentTier.rank >= requiredTierId.rank`; `requiredTierId = null` thì luôn `false`.
    - `point_redemption`: `true` khi `requiredPoints` rỗng/`<= 0` hoặc `currentPoints >= requiredPoints`.
    - `manual_grant`: luôn `false` (chỉ nhận được khi brand trao tặng; kiểm tra qua `redemption`).
- **Lỗi:**
    - `404`: Người dùng chưa là khách hàng loyalty active của brand (`ErrCustomerNotFound`).
    - `404`: Khách hàng chưa có tài khoản loyalty (`ErrActiveLoyaltyProgramRequired`).
    - `403`: Brand không ở trạng thái active (`ErrBrandNotActive`).

### 6. Quản lý nhãn hàng tạo mới một đặc quyền / voucher

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
    - `featureCode`: bắt buộc khi `benefitType = feature_access` và phải nằm trong [constants.md:BenefitFeatureCode](constants.md#8-ma-dac-quyen-he-thong-benefitfeaturecode).
    - `featureConfig`: JSON tuỳ chọn. Backend hiện chỉ đọc `validDurationDays` để tính `expiresAt` của lượt redeem; các key khác chỉ được lưu lại và trả về, chưa dùng để check quyền.
- **Response:**
    - `201 Created`: Trả về thông tin đặc quyền vừa tạo `BrandBenefitRes`.

### 7. Quản trị viên xem danh sách đặc quyền của nhãn hàng

- **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/benefits`
- **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
- **Đối tượng ảnh hưởng:** Đọc các đặc quyền `brand_benefits` đang active thuộc nhãn hàng.
- **Mô tả:** Trả về danh sách các ưu đãi đang hoạt động. Hỗ trợ phân trang và lọc theo `unlockType` hoặc `benefitType`.
- **Query Params:**

    | Param | Type | Bắt buộc | Mặc định | Mô tả |
    |---|---|---|---|---|
    | `page` | int | Không | `1` | Số trang |
    | `limit` | int | Không | `20` | Số item mỗi trang (tối đa 100) |
    | `unlockType` | string | Không | - | Lọc theo loại mở khóa: `tier_privilege`, `point_redemption`, `manual_grant` |
    | `benefitType` | string | Không | - | Lọc theo loại quyền lợi: `voucher`, `discount`, `gift`, `free_shipping`, `early_access`, `feature_access` |

- **Response:**
    - `200 OK`: Trả về danh sách đặc quyền phân trang `BrandBenefitListRes` (cùng format với Item 1).

### 8. Cập nhật trạng thái hoạt động của một đặc quyền

- **Endpoint:** `PATCH /api/v1/brand-portal/brands/:brandId/benefits/:benefitId/status`
- **Tác nhân (Actor):** Chủ thương hiệu hoặc quản lý nhãn hàng (Brand owner/staff).
- **Đối tượng ảnh hưởng:** Cập nhật trạng thái hoạt động của đặc quyền `brand_benefits.status`.
- **Mô tả:** Kích hoạt hiển thị (active), tạm đóng (inactive) hoặc lưu trữ thu hồi (archived) một chương trình ưu đãi. Trạng thái `status` tham chiếu chi tiết tại [constants.md:BenefitStatus](constants.md#13-trang-thai-cua-dac-quyen-benefitstatus).
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
