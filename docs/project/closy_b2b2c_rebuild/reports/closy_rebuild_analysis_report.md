# BÁO CÁO PHÂN TÍCH VÀ ĐÁNH GIÁ REBUILD CLOSY THEO MÔ HÌNH B2B2C (CẬP NHẬT CHUYỂN ĐỔI PHONE-FIRST IDENTITY)

Tài liệu này đánh giá hiện trạng hệ thống Closy (backend `smart-wardrobe-be`) và lập kế hoạch chi tiết để chuyển đổi sang mô hình B2B2C mới với mô hình cơ sở dữ liệu tối giản và chuyển đổi hoàn toàn phương thức xác thực danh tính sang số điện thoại (Phone-First Identity).

---

## 1. Tóm tắt quyết định kiến trúc mới
*   **Mô hình kinh doanh**: Chuyển từ B2C wardrobe app kết hợp mạng xã hội tự do/resale C2C sang mô hình **B2B2C Fashion Loyalty & Co-creation Platform**.
*   **Định vị công nghệ**: Sử dụng dữ liệu tủ đồ số và AI styling làm lớp tạo tương tác (B2C engagement), đồng thời tạo nguồn thu chính B2B từ các dịch vụ Loyalty, Campaigns, Customer Service và Digital Sample Lab cho nhãn hàng thời trang.
*   **Xác thực danh tính số điện thoại làm lõi (Phone-First Identity)**: Loại bỏ phương thức xác thực bằng email làm định danh chính. Số điện thoại chuẩn hóa E.164 (`phone_e164`) sẽ là trường định danh chính cho tất cả các luồng đăng ký, đăng nhập, và liên kết thẻ thành viên nhãn hàng (Loyalty) trên toàn hệ thống. Email trở thành tùy chọn (nullable).
*   **Cấu trúc Package**: Tổ chức theo cấu trúc **Modular Monolith** tối giản hóa với **5 runtime modules**:
    *   `identity`: Quản lý auth/user (đăng nhập/đăng ký bằng số điện thoại, xác minh mã OTP gửi qua SMS/Zalo, khôi phục tài khoản, xác thực email bổ sung).
    *   `subscription`: Quản lý premium/quota/payment.
    *   `wardrobe` (Unified Module): Gộp toàn bộ nghiệp vụ tủ đồ và Digital Sample Lab bao gồm categories, fashion_items, wardrobe_items, brand_items, outfits, outfit_items, digital_sample_responses.
    *   `styling`: Quản lý AI outfit recommendation, AI chat, prompt orchestration, RAG/retrieval/reranking.
    *   `brand`: Quản lý brand CRM, brand members, brand customers, loyalty, campaign, benefit, customer service.

---

## 2. Thiết kế nghiệp vụ Nhãn hàng & Identity (Phone-First)

### 2.1. Đăng ký, Đăng nhập & Cập nhật Thông tin
*   Số điện thoại E.164 (`phone_e164`) là thông tin bắt buộc và duy nhất dùng để tạo tài khoản mới đối với khách hàng tự đăng ký (Self-signup) hoặc được nhãn hàng tạo offline.
*   **Luồng cập nhật Email (yêu cầu xác thực OTP)**:
    *   Chỉ áp dụng sau khi người dùng đã có tài khoản hoạt động (`ACTIVE`).
    *   Khi người dùng cập nhật Email qua app, hệ thống sẽ gửi một mã OTP xác thực đến địa chỉ email mới đó.
    *   Email chỉ được cập nhật chính thức vào profile và set cờ `email_verified_at` khi người dùng nhập đúng mã OTP xác thực email qua endpoint kiểm tra.
*   **Mở rộng luồng khôi phục mật khẩu hiện có (Account Recovery)**:
    *   Thay vì tạo api mới, chúng ta **mở rộng trực tiếp các API quên mật khẩu có sẵn** (`POST /api/v1/auth/forgot-password`) để nhận diện và hỗ trợ gửi OTP khôi phục qua Số điện thoại (`phone_e164`) hoặc Email.
    *   **Điều kiện bảo mật bắt buộc**: Hệ thống phải kiểm tra xem thông tin định danh gửi yêu cầu (Số điện thoại/Email) đã được xác thực trước đó hay chưa (kiểm tra cờ `phone_verified_at IS NOT NULL` đối với SĐT, hoặc `email_verified_at IS NOT NULL` đối với Email). Nếu định danh chưa từng được verify, hệ thống từ chối cho phép khôi phục tài khoản.

### 2.2. Quy trình Tích điểm Loyalty Offline cho Khách hàng mới
Để hỗ trợ Brand thu hút khách hàng mới tại cửa hàng thực tế (offline), Closy hỗ trợ luồng đăng ký hội viên qua số điện thoại:
1.  Nhân viên nhãn hàng (Brand Staff) tại quầy nhập số điện thoại (`phone`), tên khách hàng (`customer_name`) và giá trị đơn hàng thanh toán (`purchase_amount`) trên Brand Portal.
2.  Hệ thống chuẩn hóa số điện thoại về định dạng **E.164** (`phone_e164`).
3.  Tìm kiếm người dùng trong hệ thống theo `phone_e164`:
    *   **Nếu chưa tồn tại**: Hệ thống tự động tạo tài khoản `users` mới với trạng thái `status = 'UNVERIFIED'`, nguồn đăng ký `registration_source = 'BRAND_CREATED'`. Tài khoản này chưa thể đăng nhập hoặc chat cho đến khi người dùng kích hoạt.
    *   **Nếu đã tồn tại**: Sử dụng tài khoản `user_id` hiện có.
4.  Tự động liên kết hoặc tạo bản ghi `brand_customers` với `joined_source = 'OFFLINE_PURCHASE'` (hoặc `STAFF_CREATED`).
5.  Khởi tạo `loyalty_accounts` của user tại brand này nếu chưa có.
6.  Đọc `loyalty_programs.amount_per_point` của nhãn hàng để tính toán số điểm tích lũy (`earned_points`).
7.  Cập nhật `loyalty_accounts` của user: cộng điểm tích lũy (`current_points`, `lifetime_points`), cộng doanh thu chi tiêu (`total_spend`), và cập nhật cấp hạng thành viên (`current_tier_id`) tương ứng theo mức chi tiêu mới.
8.  Tạo bản ghi lịch sử `loyalty_point_transactions` loại `EARN` (lưu `expires_at` và `remaining_points`).
9.  **Kích hoạt tài khoản (App Activation)**: Khi người dùng tải app Closy về máy và đăng ký/xác minh OTP số điện thoại qua luồng OTP hiện có, tài khoản `users` sẽ chuyển sang trạng thái `status = 'ACTIVE'`, thiết lập `phone_verified_at`. Lúc này, toàn bộ điểm số và thứ hạng thành viên đã tích lũy offline trước đó sẽ tự động hiển thị trong ứng dụng.
10. **Giới hạn tin nhắn**: Tính năng Chat hỗ trợ (`brand_conversations`) chỉ hiển thị và hoạt động đầy đủ khi tài khoản người dùng đã chuyển sang trạng thái `ACTIVE`. Đối với tài khoản `UNVERIFIED` do Brand tạo offline, hệ thống chỉ duy trì thông tin tích điểm và hạng thành viên.

### 2.3. Quy tắc di chuyển tài khoản Email cũ (Backward Compatibility)
*   Để không làm đứt gãy trải nghiệm của người dùng cũ đã đăng ký bằng Email:
    *   Trong bảng `users`, trường `phone_e164` vẫn cấu hình `nullable` ở mức Database để tương thích ngược.
    *   Khi người dùng cũ đăng nhập bằng Email, hệ thống sẽ yêu cầu (prompt) bổ sung và xác minh Số điện thoại trước khi cho phép sử dụng các tính năng mới (phối đồ đối tác, tích điểm loyalty). 
    *   Mọi tài khoản mới tạo từ thời điểm Rebuild bắt buộc phải cung cấp số điện thoại.

---

## 3. Phân loại field hiện tại của `wardrobe_items` sang `fashion_items`

| Field hiện tại | Hướng xử lý | Module sở hữu mới |
| :--- | :--- | :--- |
| `id` | Giữ ở `wardrobe_items.id` làm ID user-facing. | `wardrobe` |
| `user_id` | Giữ ở `wardrobe_items.user_id`. | `wardrobe` |
| `category_id` | **Chuyển sang** `fashion_items.category_id`. | `wardrobe` (nội bộ) |
| `image_url` & `image_public_id` | **Chuyển sang** `fashion_items`. | `wardrobe` (nội bộ) |
| `color`, `color_hex`, `color_hue`, `color_saturation`, `color_lightness` | **Chuyển sang** `fashion_items` (color metadata). | `wardrobe` (nội bộ) |
| `style`, `material`, `pattern`, `fit`, `seasonality`, `description` | **Chuyển sang** `fashion_items` (fashion metadata). | `wardrobe` (nội bộ) |
| `price` | **Đổi tên thành** `purchase_price` trên bảng `wardrobe_items` để lưu giá mua của user. | `wardrobe` |
| `status` | Giữ ở `wardrobe_items.status` (trạng thái trong tủ đồ). | `wardrobe` |
| `item_type` | Giữ ở `wardrobe_items.item_type` (định danh phân loại). | `wardrobe` |
| `embedding` | **Chuyển sang** `fashion_items.embedding` (Vector 768). | `wardrobe` (nội bộ) |
| `last_used_at` & `is_deleted` | Giữ ở `wardrobe_items`. | `wardrobe` |
| `processing_*` (Các trường AI processing) | **Chuyển sang** `fashion_items`. | `wardrobe` (nội bộ) |
| `review_reason` | **Chuyển sang** `fashion_items.review_reason`. | `wardrobe` (nội bộ) |

---

## 4. Thiết kế database cuối cùng (Self-contained)

### Bảng giữ nguyên trạng hoặc chỉnh sửa nhẹ

#### `users` (Mở rộng xác thực số điện thoại làm định danh chính)
*   `id` (UUID, PK)
*   `phone_e164` (VARCHAR(50), NULLABLE, UNIQUE) - Số điện thoại E.164 làm khóa đăng nhập và tích điểm. Bắt buộc cho tài khoản mới.
*   `phone_verified_at` (TIMESTAMP, NULLABLE) - Thời điểm xác thực số điện thoại thành công.
*   `email` (VARCHAR(255), NULLABLE, UNIQUE) - Email chuyển thành tùy chọn.
*   `email_verified_at` (TIMESTAMP, NULLABLE) - Thời điểm xác thực email thành công.
*   `display_name` (VARCHAR(255), NULLABLE)
*   `status` (VARCHAR(50)) - Giá trị: `UNVERIFIED`, `ACTIVE`, `SUSPENDED`, `DELETED`
*   `registration_source` (VARCHAR(50)) - Giá trị: `SELF_SIGNUP`, `BRAND_CREATED`, `ADMIN_CREATED`
*   `created_at`, `updated_at`

*(Các bảng categories, user_style_profiles, refresh_tokens, và các bảng quota/subscription giữ nguyên).*

### Bảng sửa đổi cấu trúc

#### `fashion_items` (Bảng mới - Item lõi chứa thuộc tính thời trang chung)
*   `id` (UUID, PK)
*   `category_id` (UUID, FK `categories.id`)
*   `image_url` (VARCHAR(500)), `image_public_id` (VARCHAR(255))
*   `color` (VARCHAR(50)), `color_hex` (VARCHAR(7)), `color_hue` (DOUBLE PRECISION), `color_saturation` (DOUBLE PRECISION), `color_lightness` (DOUBLE PRECISION)
*   `style` (VARCHAR(100)), `material` (VARCHAR(100)), `pattern` (VARCHAR(100)), `fit` (VARCHAR(50)), `seasonality` (VARCHAR(100))
*   `description` (TEXT)
*   `embedding` (Vector/HNSW index 768)
*   `processing_retry_count` (INT), `processing_version` (INT), `processing_started_at` (TIMESTAMP), `last_processing_attempt_at` (TIMESTAMP), `processing_error_reason` (TEXT), `review_reason` (VARCHAR(100))
*   `created_at`, `updated_at`

#### `wardrobe_items` (Wrapper tủ đồ user)
*   `id` (UUID, PK)
*   `user_id` (UUID, FK `users.id`)
*   `fashion_item_id` (UUID, FK `fashion_items.id`)
*   `purchase_price` (DECIMAL(12,2), NULLABLE)
*   `status` (SMALLINT)
*   `item_type` (SMALLINT)
*   `last_used_at` (TIMESTAMP)
*   `is_deleted` (BOOLEAN)
*   `created_at`, `updated_at`

#### `brand_items` (Bảng mới - Wrapper danh mục sản phẩm & mẫu thử của Brand)
*   `id` (UUID, PK)
*   `brand_id` (UUID)
*   `fashion_item_id` (UUID, FK `fashion_items.id`, UNIQUE INDEX)
*   `product_code` (VARCHAR(100))
*   `name` (VARCHAR(255))
*   `description` (TEXT, NULLABLE)
*   `price` (DECIMAL(12,2), NULLABLE) - Lưu giá bán thực tế cho PRODUCT, hoặc giá dự kiến cho SAMPLE.
*   `item_type` (VARCHAR(50)) - Giá trị: `PRODUCT`, `SAMPLE`
*   `status` (VARCHAR(50)) - Giá trị: `DRAFT`, `ACTIVE`, `ARCHIVED`
*   `created_at`, `updated_at`

#### `digital_sample_responses` (Bình chọn & Phản hồi tập trung)
*   `id` (UUID, PK)
*   `brand_item_id` (UUID, FK `brand_items.id` với điều kiện `item_type = 'SAMPLE'`)
*   `user_id` (UUID, FK `users.id`)
*   `outfit_id` (UUID, FK `outfits.id`, NULLABLE)
*   `vote_type` (VARCHAR(50), NULLABLE) - Giá trị: `LIKE`, `DISLIKE`, `WOULD_BUY`, `NOT_INTERESTED`
*   `rating` (INT, NULLABLE)
*   `feedback_text` (TEXT, NULLABLE)
*   `created_at` (TIMESTAMP)

#### `outfits`
*   `id` (UUID, PK), `user_id` (UUID, FK `users.id`), `name`, `description`, `cover_image_url`, `cover_public_id`, `outfit_source`, `status`, `is_deleted`, `created_at`, `updated_at`

#### `outfit_items`
*   `outfit_id` (UUID, FK `outfits.id`, Composite PK)
*   `fashion_item_id` (UUID, FK `fashion_items.id`, Composite PK)
*   `item_context` (VARCHAR(50)) - Giá trị: `USER_WARDROBE`, `BRAND_ITEM`
*   `position_x` (DOUBLE PRECISION), `position_y` (DOUBLE PRECISION), `scale` (DOUBLE PRECISION), `layer_order` (SMALLINT)
*   `created_at`, `updated_at`

### Bảng mới đề xuất (Module `brand` quản lý)

#### `brands` & `brand_members` & `brand_customers`
*(Cơ cấu giữ nguyên thiết kế)*.

#### `brand_customers` (Khách hàng thành viên)
*   `id` (UUID, PK), `brand_id` (UUID, FK `brands.id`), `user_id` (UUID, FK `users.id`), `customer_name` (VARCHAR(255), NULLABLE), `external_customer_code` (VARCHAR(100), NULLABLE), `joined_source`, `status`, `joined_at`, `created_by_member_id` (UUID, NULLABLE), `created_at`, `updated_at`
*   *Unique Constraint*: `(brand_id, user_id)`

#### `loyalty_programs` & `loyalty_tiers` & `loyalty_accounts` & `loyalty_point_transactions`
*(Cơ cấu giữ nguyên thiết kế)*.

#### `brand_benefits` & `benefit_redemptions`
*(Cơ cấu giữ nguyên thiết kế)*.

#### `brand_conversations` & `brand_conversation_messages`
*(Cơ cấu giữ nguyên thiết kế)*.

---

## 5. Thiết kế contract giữa các module

### `wardrobe/contract`
*   `CreateFashionItem(input) -> FashionItemDTO`
*   `GetFashionItem(id) -> FashionItemDTO`
*   `ListUserWardrobeItemsForStyling(userID, filter) -> []WardrobeItemStylingDTO`
*   `GetUserStyleProfile(userID) -> StyleProfileDTO`

### `styling/contract`
*   `RecommendOutfit(input) -> RecommendationResultDTO`

### `subscription/contract`
*   `CanUseAI(userID, operation) -> bool`
*   `ReserveAIUsage(userID, operation) -> (reservationID, error)`
*   `FinalizeAIUsage(reservationID) -> error`
*   `RefundAIUsage(reservationID) -> error`

### `brand/contract`
*   `CheckBrandMemberRole(userID, brandID) -> string`
*   `GrantLoyaltyPoints(userID, brandID, points, reason) -> error`
*   `RecordCampaignInteraction(userID, campaignID, interactionType) -> error`
*   `ListEligibleBrandItemsForStyling(userID, filter) -> []BrandItemStylingDTO`
*   `CheckBrandFeatureAccess(userID, brandID, featureCode) -> bool`

---

## 6. API mới/mở rộng bổ sung

### 6.1. Auth & Me Routes (Mở rộng từ Router có sẵn)
*   **Mở rộng Router `/auth` hiện tại**:
    *   `POST /api/v1/auth/forgot-password`: Mở rộng để nhận diện và gửi mã OTP khôi phục qua cả SĐT (`phone_e164`) hoặc Email. Kiểm tra cờ verified trước khi gửi.
    *   `POST /api/v1/auth/forgot-password/confirm-otp`: Xác thực OTP khôi phục.
    *   `POST /api/v1/auth/forgot-password/resend-otp`: Gửi lại OTP khôi phục.
    *   `POST /api/v1/auth/reset-password`: Reset mật khẩu mới.
*   **Thêm mới dưới Router `/me` hiện tại**:
    *   `POST /api/v1/me/email/request-update`: Gửi mã OTP xác nhận tới Email mới yêu cầu cập nhật.
    *   `POST /api/v1/me/email/verify-update`: Xác thực mã OTP email để chính thức cập nhật địa chỉ email vào profile và set `email_verified_at`.

### 6.2. Brand Portal API
*   `GET /api/v1/brand-portal/brands/:brandId/brand-items`
*   `POST /api/v1/brand-portal/brands/:brandId/brand-items`
*   `PATCH /api/v1/brand-portal/brand-items/:itemId`
*   `GET /api/v1/brand-portal/brands/:brandId/customers`
*   `POST /api/v1/brand-portal/brands/:brandId/customers/:userId/points`
*   `GET /api/v1/brand-portal/brands/:brandId/customers/:userId/loyalty`
*   `POST /api/v1/brand-portal/brands/:brandId/loyalty-programs`
*   `POST /api/v1/brand-portal/brands/:brandId/loyalty-tiers`
*   `POST /api/v1/brand-portal/brands/:brandId/benefits`
*   `GET /api/v1/brand-portal/brands/:brandId/conversations`
*   `GET /api/v1/brand-portal/conversations/:conversationId/messages`
*   `POST /api/v1/brand-portal/conversations/:conversationId/messages`

### 6.3. User-facing API (B2C)
*   `POST /api/v1/brand-items/:itemId/responses` (Vote/feedback của user đối với mẫu thử `SAMPLE`).
*   `POST /api/v1/ai/outfit-recommendations` (Bổ sung `include_brand_items` BOOLEAN).
*   `GET /api/v1/brands`
*   `POST /api/v1/brands/:brandId/join-loyalty`
*   `GET /api/v1/me/loyalties`
*   `GET /api/v1/me/loyalties/:brandId`
*   `GET /api/v1/brands/:brandId/benefits`
*   `POST /api/v1/brands/:brandId/benefits/:benefitId/redeem`
*   `GET /api/v1/brands/:brandId/conversation`
*   `POST /api/v1/brands/:brandId/conversation/messages`

---

## 7. Các phần không thực hiện ở MVP (Tránh over-engineering)
*   `phone_otp_challenges` & `user_brand_consents`: Sử dụng Redis OTP hiện tại.
*   `loyalty_point_lots`: Tích hợp trực tiếp hạn dùng trên transaction ledger.
*   `brand_orders` & `brand_order_items`: Không lưu vết đơn hàng vật lý của brand.
*   `support_tickets` & `return_exchange_requests`: Chat hỗ trợ trực tiếp.

---

## 8. Kịch bản triển khai & Danh sách task (Phases)
1.  **Phase 0**: Duyệt báo cáo thiết kế (Báo cáo này).
2.  **Phase 1**: Drop các bảng community/resale cũ, archive module community.
3.  **Phase 2**: Tạo bảng `fashion_items` trong module `wardrobe`, di chuyển dữ liệu cũ từ `wardrobe_items` sang `fashion_items`.
4.  **Phase 3**: Cập nhật `outfit_items.item_id` trỏ sang `fashion_items.id`, thêm trường `item_context`.
5.  **Phase 4**: Tách module `styling` ra khỏi wardrobe, thiết lập `styling/contract`.
6.  **Phase 5**: Xây dựng module `brand` (CRM, Loyalty, Campaigns, Chat, Offline Acquisition).
7.  **Phase 6**: Thêm bảng `brand_items`, `digital_sample_responses` và kịch bản phối đồ AI tích hợp flag đối tác.
8.  **Phase 7**: Seed dữ liệu và demo.
