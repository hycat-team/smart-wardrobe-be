# Closy B2B2C Rebuild Phases

Tài liệu này là kế hoạch code theo phase cho agent. Mục tiêu là chuyển Closy từ B2C wardrobe app sang B2B2C Fashion Loyalty & Co-creation Platform nhưng vẫn giữ backend theo hướng modular monolith, tránh over-engineering và tránh làm sai business đã chốt.

## Cách dùng

Agent phải đọc theo thứ tự:

0. Tuân theo rule của dự án nằm ở `.agentrules` ngoài root dự án
1. `shared_rules/global_constraints.md`
2. `shared_rules/module_boundaries.md`
3. `shared_rules/migration_and_testing_rules.md`
4. `phase_00_alignment.md`
5. Các phase còn lại theo thứ tự từ phase 01 đến phase 08.

Không được nhảy phase nếu phase trước chưa compile, migration chưa chạy được, hoặc acceptance checklist chưa pass.

## Nguồn quyết định cuối

Khi có mâu thuẫn giữa report cũ và tài liệu phases này, ưu tiên tài liệu phases này.

Các quyết định cuối đã chốt:

- Không tạo module `garment`, `samplelab`, `campaign`, `loyalty`, `chat`, `brand_subscription`.
- Chỉ có 5 runtime modules: `identity`, `subscription`, `wardrobe`, `styling`, `brand`.
- Phone-first identity, nhưng không tạo bảng OTP vì OTP đang xử lý qua Redis hiện có.
- Không dùng `user_brand_consents` trong MVP.
- Không dùng `loyalty_point_lots`.
- Không dùng `remaining_points` trong `loyalty_point_transactions`.
- `loyalty_point_transactions` là append-only ledger.
- Campaign out of scope trong MVP hiện tại.
- Brand subscription/B2B billing out of scope trong MVP hiện tại.
- Brand không đăng nhập bằng account riêng; brand staff là `users` được map qua `brand_members`.
- User offline do brand tạo bằng phone có `status = UNVERIFIED`, `registration_source = BRAND_CREATED`.
- User chỉ active khi verify phone bằng OTP flow hiện có.
- Brand không được xem raw wardrobe cá nhân của user.
- AI outfit recommendation chỉ có `include_brand_items`; không có `required_brand_item_id`.
- Hệ thống tự tìm brand items hợp lệ khi `include_brand_items = true`.
- Tier loyalty dựa trên `total_spend`, không dựa trên `current_points`.
- Một API loyalty points thống nhất nhận `user_id` hoặc `phone`.

## Cấu trúc phase

```text
phases/
├── README.md
├── shared_rules/
│   ├── global_constraints.md
│   ├── migration_and_testing_rules.md
│   └── module_boundaries.md
├── phase_00_alignment.md
├── phase_01_identity_phone_first/
│   ├── 01a_users_schema.md
│   ├── 01b_auth_phone_first.md
│   └── 01c_offline_unverified_user_contract.md
├── phase_02_archive_legacy_community.md
├── phase_03_fashion_items_migration/
│   ├── 03a_schema_and_backfill.md
│   ├── 03b_code_refactor.md
│   └── 03c_search_embedding_sync.md
├── phase_04_outfit_items_context.md
├── phase_05_brand_module/
│   ├── 05a_brand_core.md
│   ├── 05b_loyalty_schema.md
│   ├── 05c_loyalty_points_usecase.md
│   ├── 05d_benefits_feature_access.md
│   ├── 05e_brand_chat.md
│   └── 05f_brand_privacy_visibility.md
├── phase_06_brand_items_and_sample_feedback.md
├── phase_07_styling_brand_integration.md
└── phase_08_seed_demo_and_final_validation.md
```

## Stop gates

Agent phải dừng và báo lại nếu gặp một trong các tình huống sau:

- Schema hiện tại khác đáng kể so với assumption trong phase.
- Có migration phá dữ liệu cũ nhưng chưa có backfill hoặc rollback plan.
- Không xác định được current auth flow dùng password hay OTP-only.
- Không tìm thấy nơi lưu OTP Redis hiện có nhưng phase yêu cầu mở rộng auth.
- Không xác định được current route/handler của AI outfit recommendation.
- Không xác định được current query search/vector/Elastic sync đang đọc `wardrobe_items` như thế nào.
- Muốn thêm table/module/API ngoài scope đã chốt.

## Quy tắc output sau mỗi phase

Sau mỗi phase, agent phải báo lại các thông tin bên dưới vào folder /phases/reports, mỗi phase hoặc phase con đều phải có 1 bản report kết quả:

```text
- Files changed
- Migrations added
- APIs added/changed
- Tests added/updated
- Backward compatibility notes
- Manual verification steps
- Known limitations
```
