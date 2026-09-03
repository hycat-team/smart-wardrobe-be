# Kế Hoạch Kỹ Thuật (Implementation Plan): Tách Biệt Contract DTO Và Chuẩn Hóa Ranh Giới Dữ Liệu Giữa Các Module

**Branch**: `018-contract-dto-refactor` | **Date**: 2026-09-03 | **Spec**: [`specs/018-contract-dto-refactor/spec.md`](spec.md)

---

## Tóm Tắt (Summary)

Đợt refactor này giải quyết dứt điểm vấn đề **Over-sharing DTO** giữa tầng API Client Response và tầng Internal Contract. Trọng tâm ưu tiên cao nhất (Phase 1) là tách bỏ hoàn toàn các trường lồng `FashionItem` và `BrandItem` rỗng khỏi các DTO của `IWardrobeContract` và `IBrandContract`, thay thế bằng các struct thuần túy `WardrobeItemContractDTO` và `BrandItemContractDTO` đặt trong `contract_dto.go`. Phase 2 sẽ chuẩn hóa tổ chức file `contract_dto.go` cho tất cả các module còn lại (`identity`, `subscription`, `fashion`, `admin`, `notification`), bảo đảm sự thống nhất 100% trong toàn bộ codebase mà không gây ra bất kỳ lỗi hồi quy nào cho các API công khai của Web/Mobile Client.

---

## Bối Cảnh Kỹ Thuật (Technical Context)

**Ngôn ngữ/Phiên bản**: Go 1.23+  
**Kiến trúc chính**: Modular Monolith, Clean Architecture (Presentation, Application, Domain, Infrastructure), In-Process Contracts (`presentation/contract/`), Wire DI.  
**Lưu trữ**: PostgreSQL 16 (GORM), Redis (Caching) — *Refactor này hoàn toàn không thay đổi Database Schema hay Persistence layer.*  
**Testing**: Go standard testing suite, test race detector (`make test-race`), GolangCI-Lint (`make lint`).  
**Nền tảng mục tiêu**: Linux / Docker container, Windows dev environment.  
**Mục tiêu hiệu năng**: Truy xuất dữ liệu chéo module ở phía Caller giữ nguyên độ phức tạp $O(1)$ nhờ `sliceutils.IndexBy` / `GroupBy`. Không phát sinh truy vấn N+1.  
**Ràng buộc kỹ thuật**:
- Tuân thủ nghiêm ngặt Hiến pháp dự án ([`constitution.md`](../../.specify/memory/constitution.md)) và [Contract Purity Pattern](../../docs/system-design/contract-purity-pattern.md).
- Zero API breaking changes: Cấu trúc JSON trả về cho Client của các public endpoints (`WardrobeItemRes`, `BrandItemRes`) giữ nguyên vẹn 100%.

---

## Kiểm Tra Hiến Pháp (Constitution Check)

*GATE: Phải vượt qua trước và sau khi hoàn thiện thiết kế.*

| Nguyên tắc Hiến pháp | Đánh giá tuân thủ | Chi tiết xác thực |
|:---|:---:|:---|
| **§I. Pragmatic Modular Monolith** | **PASS** | Không vi phạm ranh giới module, không preload chéo database, giữ nguyên tính độc lập của các domain entities. |
| **§II. Strict Module Boundaries & DTO Contracts** | **PASS (CHÍNH)** | Refactor này trực tiếp củng cố §II: Loại bỏ hoàn toàn các trường lồng liên module rỗng (`FashionItem == nil`) trong Contract DTO. Caller tự làm giàu dữ liệu bằng `sliceutils.IndexBy`/`GroupBy`. |
| **§II. Pure Contract Usecase Principle** | **PASS** | `WardrobeContractUseCase` và `BrandItemContractUseCase` giữ trạng thái thuần khiết (Pure), map trực tiếp entity sang `ContractDTO`. |
| **§IV. Clean Architecture & Strict Layer Dependency** | **PASS** | `contract_dto.go` nằm ở `application/dto`, tầng `presentation/contract` phụ thuộc vào `application/dto` là đúng chiều phụ thuộc (`presentation -> application`). |
| **Allowed cross-module imports** | **PASS** | Caller import `presentation/contract/` và `application/dto/contract_dto.go` là hoàn toàn hợp lệ theo bảng allowed imports của Hiến pháp §II. |

*Kết luận Gate*: **PASSED 100%** — Thiết kế hoàn toàn tương thích và tăng cường sự tuân thủ Hiến pháp.

---

## Cấu Trúc Dự Án (Project Structure)

### Tài liệu thiết kế (Specs & Design Artifacts)

```text
specs/018-contract-dto-refactor/
├── spec.md                     # Đặc tả yêu cầu và kịch bản người dùng
├── plan.md                     # Bản kế hoạch kỹ thuật chi tiết này
├── research.md                 # Phân tích kỹ thuật và ma trận audit tác động
├── data-model.md               # Thiết kế chi tiết cấu trúc các Contract DTO
├── quickstart.md               # Kịch bản kiểm thử biên dịch và xác thực regression
├── contracts/
│   ├── wardrobe_contract.md    # Đặc tả interface IWardrobeContract và di chuyển DTO
│   ├── brand_contract.md       # Đặc tả interface IBrandContract và di chuyển DTO
│   └── other_contracts.md      # Đặc tả chuẩn hóa Phase 2 cho các module còn lại
└── checklists/
    └── requirements.md         # Checklist chất lượng yêu cầu (16/16 pass)
```

### Cấu trúc mã nguồn liên quan (Source Code Layout)

```text
internal/modules/
├── wardrobe/
│   ├── application/
│   │   ├── dto/
│   │   │   ├── contract_dto.go         # [NEW] Chứa WardrobeItemContractDTO
│   │   │   └── wardrobe.go             # [MODIFY] Giữ WardrobeItemRes phục vụ API Handlers
│   │   ├── mapper/
│   │   │   └── wardrobe.go             # [MODIFY] Thêm mapper MapToWardrobeItemContractDTO
│   │   └── usecase/wardrobe/
│   │       ├── contractuc/             # [MODIFY] Trả về WardrobeItemContractDTO
│   │       └── item/                   # [MODIFY] Stitch dữ liệu từ BrandItemContractDTO
│   └── presentation/contract/
│       ├── wardrobe_contract.go        # [MODIFY] Signature trả về WardrobeItemContractDTO
│       └── wardrobe_contract_impl.go   # [MODIFY] Implementation trả về WardrobeItemContractDTO
├── brand/
│   ├── application/
│   │   ├── dto/
│   │   │   ├── contract_dto.go         # [NEW] Chứa BrandItemContractDTO
│   │   │   └── brand_item.go           # [MODIFY] Giữ BrandItemRes phục vụ API Handlers
│   │   ├── mapper/
│   │   │   └── brand_item_mapper.go    # [MODIFY] Thêm mapper MapToBrandItemContractDTO
│   │   └── usecase/item/contractuc/    # [MODIFY] Trả về BrandItemContractDTO
│   └── presentation/contract/
│       ├── brand_contract.go           # [MODIFY] Signature trả về BrandItemContractDTO
│       └── brand_contract_impl.go      # [MODIFY] Implementation trả về BrandItemContractDTO
├── fashion/                            # [CONSUMER / CALLER]
│   ├── application/
│   │   ├── dto/
│   │   │   └── contract_dto.go         # [NEW / PHASE 2] Chuẩn hóa Fashion Contract DTOs
│   │   └── usecase/
│   │       ├── ai/recommendation/      # [MODIFY] Cập nhật nhận ContractDTO và stitch
│   │       ├── ai/chat/                # [MODIFY] Cập nhật nhận ContractDTO và stitch
│   │       └── worker/                 # [MODIFY] Cleanup & Recover stale worker
├── dashboard/                          # [CONSUMER / CALLER]
│   └── application/usecase/            # [MODIFY] Cập nhật nhận ContractDTO cho analytics
├── identity/
│   └── application/dto/
│       └── contract_dto.go             # [NEW / PHASE 2] Chuẩn hóa UserBriefContractDTO
├── subscription/
│   └── application/dto/
│       └── contract_dto.go             # [NEW / PHASE 2] Chuẩn hóa Subscription Contract DTOs
├── admin/
│   └── application/dto/
│       └── contract_dto.go             # [NEW / PHASE 2] Chuẩn hóa Audit Log Contract DTOs
└── notification/
    └── application/dto/
        └── contract_dto.go             # [NEW / PHASE 2] Chuẩn hóa Email/Notification Contract DTOs
```

---

## Phân Kỳ Triển Khai (Phasing & Execution Strategy)

### Phase 1: Trọng tâm cốt lõi (Wardrobe & Brand) — Độ ưu tiên P1
1. Tạo `wardrobe/application/dto/contract_dto.go` và `brand/application/dto/contract_dto.go`.
2. Bổ sung các hàm mapper chuyển đổi từ Domain Entity sang Contract DTO.
3. Cập nhật chữ ký interface `IWardrobeContract` và `IBrandContract` cùng file triển khai `*_impl.go`.
4. Cập nhật các Callers trong module `fashion` (Recommendation, Chat, Cleanup workers) và `dashboard`.
5. Cập nhật các Callers trong module `wardrobe` gọi `brandContract`.
6. Cập nhật toàn bộ Mock và Unit Test tương ứng (`worker_uc_test.go`, `workflows_test.go`, `dashboard_*_test.go`).
7. Chạy `make wire && make lint && make test-race` xác thực Phase 1 hoàn tất không lỗi.

### Phase 2: Chuẩn hóa toàn hệ thống — Độ ưu tiên P2
1. Tạo `contract_dto.go` trong `application/dto/` của các module: `identity`, `subscription`, `fashion`, `admin`, `notification`.
2. Di chuyển các DTO phục vụ contract vào file này (giữ nguyên alias nếu cần để tránh breaking change không đáng có).
3. Chạy `make lint && make test-race` để xác thực toàn bộ hệ thống đồng bộ 100%.

---

## Theo Dõi Độ Phức Tạp (Complexity Tracking)

> **Không có vi phạm Hiến pháp nào cần biện hộ.** Kiến trúc hoàn toàn tinh gọn và tuân thủ các nguyên tắc thiết kế của dự án.
