# Phase 05d Report - Benefits and Feature Access

Báo cáo kết quả triển khai hệ thống Quyền lợi thương hiệu (Brand Benefits) và quản lý phân quyền tính năng (Feature Access).

## Files Changed

- **Database Migration**:
  - `migrations/20260628142315_create_brand_benefits.sql`
- **Domain & Constants**:
  - `internal/shared/domain/entities/brand_entities.go` (Thêm entity `BrandBenefit`, `BenefitRedemption`)
  - `internal/shared/domain/entities/table_name.go` (Đăng ký TableName)
  - `internal/shared/domain/constants/benefit/...` (Định nghĩa các hằng số trạng thái, unlock types, và feature codes)
- **Repositories**:
  - `internal/modules/brand/domain/repositories/interfaces.go`
  - `internal/modules/brand/infrastructure/persistence/benefit_repo.go`
  - `internal/modules/brand/provider.go` (Đăng ký Dependency Injection)
- **Application (Use Cases & Contracts)**:
  - `internal/modules/brand/application/dto/brand.go` (Định nghĩa DTO)
  - `internal/modules/brand/application/usecase/brand_core_uc.go` (Triển khai CRUD, RedeemBenefit, CheckBrandFeatureAccess)
  - `internal/modules/brand/contract/service.go`
- **Presentation (API)**:
  - `internal/modules/brand/presentation/handler/brand_handler.go`
  - `internal/api/routes/brand/router.go`

## Migrations Added

- `20260628142315_create_brand_benefits.sql`: Tạo bảng `brand_benefits` và `benefit_redemptions`.

## APIs Added/Changed

- `POST /api/v1/brand-portal/brands/:brandId/benefits` (Tạo Benefit mới - Staff)
- `GET /api/v1/brand-portal/brands/:brandId/benefits` (Xem danh sách Benefit - Staff)
- `PATCH /api/v1/brand-portal/brands/:brandId/benefits/:benefitId/status` (Cập nhật trạng thái Benefit - Staff)
- `GET /api/v1/brands/:brandId/benefits` (Xem các active benefits - Khách hàng)
- `POST /api/v1/brands/:brandId/benefits/:benefitId/redeem` (Đổi điểm nhận quyền lợi - Khách hàng)

## Tests Added/Updated

- `internal/modules/brand/application/usecase/brand_core_uc_benefit_test.go`:
  - `TestCreateBrandBenefit`: Xác thực quyền staff tạo benefit.
  - `TestRedeemBenefit_Success`: Khách hàng đổi điểm thành công (giảm trừ điểm và lưu ledger).
  - `TestRedeemBenefit_InsufficientPoints`: Từ chối giao dịch khi số dư điểm không đủ.

## Backward Compatibility Notes

- Không có xung đột. Các thực thể và bảng cơ sở dữ liệu được tạo mới hoàn toàn độc lập.

## Manual Verification Steps

1. Chạy `make migration-up` để cập nhật database schema.
2. Kiểm tra Swagger tại `/swagger/index.html` tìm các API có Tag `Brand` và `Brand Portal` liên quan đến `benefits`.

## Known Limitations

- Chưa tự động áp dụng feature config khi recommendation (sẽ được tích hợp trong Phase 07).
