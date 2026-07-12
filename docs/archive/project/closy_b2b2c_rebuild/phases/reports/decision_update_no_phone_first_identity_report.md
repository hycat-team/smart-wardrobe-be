# Báo cáo cập nhật quyết định - Không dùng phone-first identity trong MVP

## Files changed

- `docs/project/closy_b2b2c_rebuild/reports/closy_rebuild_analysis_report.md`
- `docs/project/closy_b2b2c_rebuild/repo_audit_before_rebuild.md`
- `docs/project/closy_b2b2c_rebuild/phases/README.md`
- `docs/project/closy_b2b2c_rebuild/phases/shared_rules/global_constraints.md`
- `docs/project/closy_b2b2c_rebuild/phases/shared_rules/module_boundaries.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_01_identity_phone_first/01a_users_schema.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_01_identity_phone_first/01b_auth_phone_first.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_01_identity_phone_first/01c_offline_unverified_user_contract.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_05_brand_module/05a_brand_core.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_05_brand_module/05b_loyalty_schema.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_05_brand_module/05c_loyalty_points_usecase.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_05_brand_module/05d_benefits_feature_access.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_05_brand_module/05e_brand_chat.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_05_brand_module/05f_brand_privacy_visibility.md`
- `docs/project/closy_b2b2c_rebuild/phases/phase_08_seed_demo_and_final_validation.md`
- `docs/project/closy_b2b2c_rebuild/phases/reports/decision_update_no_phone_first_identity_report.md`

## Migrations added

- Không có.

## APIs added/changed

- Không có API thật nào được sửa.
- Tài liệu plan đã cập nhật API loyalty points thống nhất: `POST /api/v1/brand-portal/brands/:brandId/loyalty/points`.
- Tài liệu plan đã cập nhật claim/link flow bằng claim code/QR, không dùng phone OTP trong MVP.

## Tests added/updated

- Không có test code.
- Các checklist/test case trong phase docs đã được cập nhật theo quyết định mới.

## Backward compatibility notes

- Auth hiện tại của Closy được giữ nguyên.
- Không chuyển sang phone-first identity trong MVP.
- Không tạo `users UNVERIFIED` từ phone offline.
- Offline customer được lưu ở `brand_customers` với `user_id NULL`.
- Loyalty account dùng `brand_customer_id` làm identity chính, `user_id` nullable.
- `joined_source` chỉ còn mô tả nguồn/trigger như `SELF_JOIN` và `OFFLINE_PURCHASE`; không dùng `STAFF_CREATED`.

## Manual verification steps

- Đọc file đính kèm quyết định mới.
- Quét các tài liệu rebuild để tìm các cụm phone-first, offline UNVERIFIED user, `BRAND_CREATED`, `STAFF_CREATED`, `remaining_points`.
- Cập nhật report rebuild chính thành bản self-contained theo hướng mới.
- Cập nhật shared rules và phase docs liên quan identity, brand core, loyalty schema, loyalty points usecase, benefits, privacy, chat và seed validation.
- Xác nhận không có code/migration/API production nào được sửa.

## Known limitations

- Chưa cập nhật các tài liệu ngoài `phases/` và report chính nếu chúng là bản kế hoạch cũ hoặc prompt lịch sử.
- Folder Phase 01 vẫn giữ tên cũ `phase_01_identity_phone_first` để tránh đổi path rộng; nội dung bên trong đã được cập nhật thành giữ current auth.
- Chưa code, chưa tạo migration, chưa chạy test/build vì đây là cập nhật tài liệu plan/report.

