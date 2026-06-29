# Digital Sample Lab & Brand Items API Specs

Tài liệu thiết kế các API liên quan đến sản phẩm/mẫu thử của nhãn hàng (Brand Items): bao gồm sản phẩm thương mại thực tế, các mẫu thử thiết kế kỹ thuật số (3D/phác thảo), luồng thu nhận ý kiến khảo sát/bình chọn (feedback/vote) của khách hàng và quy trình quản trị sản phẩm trên cổng Brand Portal. Tất cả các giá trị hằng số sử dụng tham chiếu tại [constants/brand.md](constants/brand.md).

---

## Flow 1: Khách hàng trải nghiệm sản phẩm/mẫu thử và gửi phản hồi (Feedback)

Người dùng duyệt xem các mẫu thiết kế mới của nhãn hàng, có thể ướm thử ảo lên canvas phối đồ và gửi đánh giá đóng góp ý kiến về việc có nên đưa thiết kế vào sản xuất thực tế hay không.

### 1. Lấy danh sách sản phẩm hoặc mẫu thử của thương hiệu
*   **Endpoint:** `GET /api/v1/brands/:brandId/items`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc danh sách các sản phẩm/mẫu thử `brand_items` đang hoạt động của brand.
*   **Mô tả:** Trả về danh sách sản phẩm thương mại hoặc mẫu thiết kế ảo đang mở cho người dùng xem và đánh giá. Bộ lọc loại vật phẩm `itemType` và trạng thái `status` tham chiếu chi tiết tại [constants/brand.md:BrandItemType](constants/brand.md#11-phan-loai-vat-pham-trong-digital-sample-lab-branditemtype) và [constants/brand.md:BrandItemStatus](constants/brand.md#10-trang-thai-vat-pham-trong-digital-sample-lab-branditemstatus).
*   **Response:**
    *   `200 OK`: Trả về mảng danh sách vật phẩm `BrandItemRes`.

### 2. Xem thông tin chi tiết một sản phẩm/mẫu thử đang active
*   **Endpoint:** `GET /api/v1/brand-items/:itemId`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Đọc thông tin một bản ghi sản phẩm `brand_items` đang active.
*   **Mô tả:** Trả về toàn bộ hồ sơ chi tiết của vật phẩm theo `itemId`. Backend tự xác định brand từ bản ghi `brand_items`, chỉ trả về item khi brand đang hoạt động và item ở trạng thái `active`.
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết `BrandItemRes`.

### 3. Gửi bình chọn (vote) và ý kiến nhận xét đóng góp cho mẫu thiết kế ảo
*   **Endpoint:** `POST /api/v1/brand-items/:itemId/feedbacks`
*   **Tác nhân (Actor):** Khách hàng (Customer).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi phản hồi mẫu thử `digital_sample_responses`, có thể ghi nhận liên kết với bộ trang phục phối đồ nếu request cung cấp mã `outfitId`.
*   **Mô tả:** Người dùng thực hiện bình chọn cho mẫu thử theo `itemId`. Backend tự xác định brand từ bản ghi `brand_items`, từ chối request nếu brand không hoạt động hoặc item không ở trạng thái `active`. Loại bình chọn `voteType` chỉ nhận `like`, `dislike`, `would_buy`, `not_interested` và tham chiếu chi tiết tại [constants/brand.md:VoteType](constants/brand.md#12-loai-vote-san-pham-mau-votetype). Nếu liên kết với outfit của người dùng (`outfitId`), hệ thống sẽ thực hiện kiểm tra xác thực quyền sở hữu outfit của user hiện tại và tính hợp lệ của việc kết hợp với brand item đó.
*   **Request Body:**
    ```json
    {
      "outfitId": "2a9de8d4-0d1c-459e-bf22-9479a9320111",
      "voteType": "would_buy",
      "rating": 5,
      "feedbackText": "Form dáng đầm rất đẹp, nếu sản xuất màu này mình chắc chắn sẽ mua."
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin phản hồi vừa lưu `DigitalSampleResponseRes`.

---

## Flow 2: Đăng tải và quản lý vòng đời Mẫu thiết kế (Brand Portal Staff)

Đội ngũ thiết kế hoặc quản lý nhãn hàng đăng tải các ý tưởng thiết kế mới dạng ảnh phác thảo/3D để tiến hành trưng bày khảo sát ý kiến khách hàng.

### 1. Lấy chữ ký tải lên hình ảnh sản phẩm/mẫu thử
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/items/upload-signature`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Không thay đổi dữ liệu nghiệp vụ.
*   **Mô tả:** Lấy mã chữ ký Cloudinary upload signature để phía client có thể tải ảnh thiết kế trực tiếp lên Cloudinary. Sau khi tải lên thành công, client sử dụng hai trường `imageUrl` và `imagePublicId` để gọi tiếp API tạo sản phẩm.
*   **Response:**
    *   `200 OK`: Trả về kết quả chữ ký `UploadSignatureResult`.

### 2. Tạo mới một sản phẩm hoặc mẫu thiết kế ảo trong hệ thống
*   **Endpoint:** `POST /api/v1/brand-portal/brands/:brandId/items`
*   **Tác nhân (Actor):** Chủ thương hiệu hoặc quản lý nhãn hàng (Brand owner/manager).
*   **Đối tượng ảnh hưởng:** Tạo bản ghi sản phẩm nhãn hàng `brand_items` và liên kết thông tin dữ liệu thời trang dùng chung `fashion_items` tương ứng theo use case backend.
*   **Mô tả:** Đăng tải một mẫu sản phẩm mới vào Sample Lab. Trạng thái mặc định ban đầu là `draft` (bản nháp). Loại vật phẩm `itemType` và trạng thái `status` tham chiếu chi tiết tại [constants/brand.md](constants/brand.md).
*   **Request Body:**
    ```json
    {
      "categoryId": "48a68d7f-5f47-46c4-8ad4-b98f28130111",
      "imageUrl": "https://res.cloudinary.com/.../jacket.png",
      "imagePublicId": "brands/local-brand-a/items/jacket",
      "productCode": "JACKET-001",
      "name": "Áo khoác gió thu đông",
      "description": "Mẫu áo khoác gió chất liệu cản gió chống thấm nước nhẹ",
      "price": 590000,
      "itemType": "product",
      "status": "draft"
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về thông tin sản phẩm vừa tạo `BrandItemRes`.

### 3. Nhân viên lấy danh sách toàn bộ sản phẩm của thương hiệu
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/items`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Đọc danh sách bản ghi sản phẩm `brand_items` thuộc nhãn hàng.
*   **Mô tả:** Trả về danh sách đầy đủ các sản phẩm/mẫu thử của brand để phục vụ việc quản trị nội bộ (bao gồm cả các sản phẩm đang ẩn hoặc nháp).
*   **Response:**
    *   `200 OK`: Trả về mảng danh sách sản phẩm `BrandItemRes`.

### 4. Nhân viên xem thông tin chi tiết một sản phẩm quản trị
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/items/:itemId`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Đọc một bản ghi sản phẩm `brand_items` thuộc nhãn hàng.
*   **Mô tả:** Trả về thông tin chi tiết sản phẩm bao gồm cả metadata thời trang liên kết `fashionItem` nếu backend có hỗ trợ trả về.
*   **Response:**
    *   `200 OK`: Trả về thông tin chi tiết `BrandItemRes`.

### 5. Cập nhật thông tin chi tiết sản phẩm / mẫu thiết kế
*   **Endpoint:** `PUT /api/v1/brand-portal/brands/:brandId/items/:itemId`
*   **Tác nhân (Actor):** Chủ thương hiệu hoặc quản lý nhãn hàng (Brand owner/manager).
*   **Đối tượng ảnh hưởng:** Cập nhật thông tin bản ghi sản phẩm `brand_items`.
*   **Mô tả:** Cập nhật các trường thông tin cơ bản của sản phẩm hoặc mẫu thử như tên gọi, mô tả, giá dự kiến hoặc trạng thái hiển thị.
*   **Request Body:**
    ```json
    {
      "name": "Áo khoác gió thu đông 2026",
      "description": "Mẫu áo gió chất liệu chống thấm nước cao cấp",
      "price": 620000,
      "status": "active"
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin sản phẩm sau khi cập nhật `BrandItemRes`.

### 6. Cập nhật nhanh trạng thái hiển thị / khảo sát của vật phẩm
*   **Endpoint:** `PATCH /api/v1/brand-portal/brands/:brandId/items/:itemId/status`
*   **Tác nhân (Actor):** Chủ thương hiệu hoặc quản lý nhãn hàng (Brand owner/manager).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái hiển thị của vật phẩm `brand_items.status`.
*   **Mô tả:** Thay đổi nhanh trạng thái để mở khảo sát (active), lưu kho hoặc đóng khảo sát (archived). Trạng thái `status` tham chiếu chi tiết tại [constants/brand.md:BrandItemStatus](constants/brand.md#10-trang-thai-vat-pham-trong-digital-sample-lab-branditemstatus).
*   **Request Body:**
    ```json
    {
      "status": "active"
    }
    ```
*   **Response:**
    *   `200 OK`: Trả về thông tin sản phẩm sau cập nhật `BrandItemRes`.

### 7. Xem danh sách ý kiến đóng góp (Feedbacks) của khách hàng cho mẫu thử
*   **Endpoint:** `GET /api/v1/brand-portal/brands/:brandId/items/:itemId/feedbacks`
*   **Tác nhân (Actor):** Nhân viên hỗ trợ hoặc quản lý nhãn hàng (Brand staff).
*   **Đối tượng ảnh hưởng:** Đọc danh sách ý kiến khảo sát `digital_sample_responses` của mẫu thử.
*   **Mô tả:** Trả về danh sách các bình chọn (vote), điểm đánh giá (rating), và nhận xét thô dạng chữ của khách hàng đã gửi cho mẫu thử để nhãn hàng phân tích thống kê thị hiếu người dùng.
*   **Response:**
    *   `200 OK`: Trả về mảng danh sách phản hồi của người dùng `DigitalSampleResponseRes`.

---

## B2B2C Model Notes

*   Mỗi bản ghi sản phẩm của nhãn hàng (`brand_items`) đều liên kết chặt chẽ đến một bản ghi metadata thời trang dùng chung `fashion_items` qua khóa ngoại `fashionItemId` để hệ thống AI hoặc tủ đồ có thể dùng chung dữ liệu thuộc tính thời trang.
*   Các sản phẩm thương mại (`product`) của nhãn hàng đang active và có đủ điều kiện có thể được hệ thống AI ưu tiên lựa chọn đưa vào danh sách ứng viên đề xuất phối đồ (AI outfit recommendations) cho người dùng.
*   Các mẫu thử kỹ thuật số (`sample`) có thể thiết lập các chính sách đặc quyền bảo mật như `sample_mix_access` (chỉ cho phép khách hàng đạt hạng thẻ thành viên quy định được thử đồ). Các mã đặc quyền hệ thống `featureCode` tham chiếu chi tiết tại [constants/brand.md:BenefitFeatureCode](constants/brand.md#8-ma-dac-quyen-he-thong-benefitfeaturecode).
