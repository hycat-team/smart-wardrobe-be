# Contract Purity Pattern & Pure Contract DTO Specification

## 1. Overview
Quy định chuẩn hóa về Contract DTO giữa các module trong hệ thống Closy Smart Wardrobe Backend per Constitution §II.

## 2. Core Rules
1. **Thư mục lưu trữ**: Mỗi module phục vụ giao tiếp liên module (cross-module presentation contract) MUST khai báo các DTO dành riêng cho contract tại `internal/modules/<module_name>/application/dto/contract_dto.go`.
2. **Quy tắc đặt tên**: Struct DTO dành cho contract MUST mang hậu tố `ContractDTO` (ví dụ: `WardrobeItemContractDTO`, `BrandItemContractDTO`, `UserBriefContractDTO`).
3. **Pure Domain Boundary (Không rò rỉ quan hệ cross-module)**:
   - Contract DTO **CHỈ** chứa các trường thuộc tính nội tại của domain (domain-native primitives/enums) và các khóa ngoại ID dạng `uuid.UUID` / `*uuid.UUID`.
   - Contract DTO **TUYỆT ĐỐI KHÔNG** chứa con trỏ tới entity/DTO của module khác (ví dụ: `WardrobeItemContractDTO` KHÔNG được chứa `FashionItem *FashionItemRes` hay `BrandItem *BrandItemRes`).
4. **Self-Enrichment & Stitching (Bên gọi tự làm giàu dữ liệu)**:
   - Consumer Module (như `fashion`, `dashboard`, `wardrobe`) khi gọi contract nhận về `ContractDTO` phải sử dụng `sliceutils.IndexBy` hoặc `GroupBy` để tự kết hợp dữ liệu với độ phức tạp $O(1)$ per Constitution §II.
5. **Tách biệt với HTTP API Response DTO**:
   - Contract DTO dành riêng cho gọi qua Go interface giữa các module backend.
   - HTTP API Response DTO (`WardrobeItemRes`, `BrandItemRes`, v.v.) nằm tại các file DTO thông thường và phục vụ API client công khai, giữ nguyên vẹn 100% cấu trúc JSON response.

## 3. Standard Structure Matrix

| Module | Thư Mục DTO | File Contract DTO | Contract DTO Structs |
|---|---|---|---|
| `wardrobe` | `internal/modules/wardrobe/application/dto/` | `contract_dto.go` | `WardrobeItemContractDTO` |
| `brand` | `internal/modules/brand/application/dto/` | `contract_dto.go` | `BrandItemContractDTO` |
| `identity` | `internal/modules/identity/application/dto/` | `contract_dto.go` | `UserBriefContractDTO` |
| `subscription` | `internal/modules/subscription/application/dto/` | `contract_dto.go` | `UserSubscriptionContractDTO`, `UserQuotaContractDTO` |
| `fashion` | `internal/modules/fashion/application/dto/` | `contract_dto.go` | `FashionItemContractDTO`, `CategoryContractDTO` |
| `admin` | `internal/modules/admin/application/dto/` | `contract_dto.go` | `AuditLogContractDTO` |
| `notification` | `internal/modules/notification/application/dto/` | `contract_dto.go` | `NotificationContractDTO` |
