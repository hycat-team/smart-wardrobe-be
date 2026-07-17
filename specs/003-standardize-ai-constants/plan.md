# Kế hoạch Triển khai: Chuẩn hoá các Hằng số Kỹ thuật của AI Executor

**Branch**: `003-standardize-ai-constants` | **Ngày**: 2026-07-17 | **Spec**: [spec.md](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/specs/003-standardize-ai-constants/spec.md)

**Đầu vào**: Đặc tả tính năng từ `/specs/003-standardize-ai-constants/spec.md`

## Tóm tắt

Tính năng này tập trung vào việc tái cấu trúc các hằng số kỹ thuật trong module `aiexecutor`:
1. Phân tách và đóng gói các hằng số từ `dto/cost.go` (Operation và Route) vào các package domain tương ứng (`domain/constants/aioperation` và `domain/constants/airoute`), đồng thời thay thế string thuần trong các thực thể domain bằng các kiểu dữ liệu mới này.
2. Thiết lập cấu hình AI Model bằng thực thể `AIModelConfig` (bảng `ai_model_configs` trực tiếp từ đầu).
3. Chuyển đổi các kiểu dữ liệu enum của `aimodel.Model` và `aiprovider.Provider` thành `string` thuần ở mọi nơi để đảm bảo tính năng cấu hình động cho Admin mà không cần sửa code.
4. Gộp 3 tệp migration cũ (`20260704062159_init_ai_executor_module.sql`, `20260705104500_make_ai_usage_events_ids_nullable.sql`, `20260709034434_add_task_id_to_items.sql`) thành một tệp migration duy nhất tại [20260704062159_init_ai_executor_module.sql](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/database/migrations/20260704062159_init_ai_executor_module.sql).

## Bối cảnh Kỹ thuật

**Ngôn ngữ/Phiên bản**: Go v1.25.+

**Thư viện Phụ thuộc**: GORM (v2), Gin, Wire, Goose (cho database migration)

**Cơ sở Dữ liệu**: PostgreSQL (với pgvector) và Redis cho model config cache.

**Công cụ Kiểm thử**: go test

**Hệ điều hành đích**: Linux / Windows Server

**Loại dự án**: Web service

**Mục tiêu Hiệu năng**: Không bị ảnh hưởng hoặc suy giảm hiệu năng do thay đổi này chỉ là kiểu biên dịch và ép kiểu ở Go runtime.

**Ràng buộc**: Đảm bảo tương thích ngược 100% về mặt dữ liệu. Sử dụng file migration Goose SQL duy nhất được gộp lại.

**Quy mô/Phạm vi**: Chỉ giới hạn trong module `internal/modules/aiexecutor` và tầng migration của toàn hệ thống.

## Kiểm tra Hiến pháp (Constitution Check)

*GATE: Phải vượt qua trước khi thực hiện Nghiên cứu Phase 0. Kiểm tra lại sau Thiết kế Phase 1.*

- **Nguyên tắc I (Pragmatic Modular Monolith)**: Việc thay đổi không phá vỡ ranh giới module. Các constant mới vẫn nằm trọn trong `internal/modules/aiexecutor`. -> **ĐẠT**
- **Nguyên tắc II (Strict Module Boundaries & DTO Contracts)**: Không có việc import chéo các entity của module khác. Các thay đổi kiểu dữ liệu trong DTO ở `application/dto` tương thích hoàn toàn. -> **ĐẠT**
- **Nguyên tắc VI (AI Pipeline, Cost Control & Local Fallbacks)**: Các trường AI model và provider chuyển sang string thuần giúp tăng tính linh hoạt cho việc cấu hình local fallbacks và pricing policies ở cơ sở dữ liệu. -> **ĐẠT**
- **Ngôn ngữ (Language Boundaries)**: Các định nghĩa code và tên biến/hằng số được viết hoàn toàn bằng tiếng Anh. Tài liệu đặc tả và kế hoạch được viết bằng tiếng Việt. -> **ĐẠT**
- **Goose Database Migrations**: Mọi thay đổi schema bắt buộc sử dụng file migration Goose SQL. -> **ĐẠT**

Không phát hiện vi phạm Hiến pháp nào.

## Cấu trúc Dự án

### Tài liệu (Feature này)

```text
specs/003-standardize-ai-constants/
├── plan.md              # File này (Kế hoạch triển khai)
├── research.md          # Kết quả nghiên cứu Phase 0
├── data-model.md        # Kết quả thiết kế data model Phase 1
├── quickstart.md        # Hướng dẫn chạy và xác thực Phase 1
└── checklists/
    └── requirements.md  # Checklist chất lượng
```

### Cấu trúc mã nguồn (Source Code)

```text
internal/modules/aiexecutor/
├── domain/
│   ├── constants/
│   │   ├── aioperation/
│   │   │   └── operation.go    # [NEW] Định nghĩa kiểu AIOperation string
│   │   ├── airoute/
│   │   │   └── route.go        # [NEW] Định nghĩa kiểu AIRoute string
│   │   ├── aimodel/
│   │   │   └── model.go        # [DELETE/CLEANUP] Chuyển đổi thành string thuần
│   │   └── aiprovider/
│   │       └── provider.go     # [DELETE/CLEANUP] Chuyển đổi thành string thuần
│   └── entities/
│       ├── policy.go           # [MODIFY] Đổi Operation string -> aioperation.AIOperation, routes -> airoute.AIRoute
│       ├── usage.go            # [MODIFY] Đổi Model/Provider thành string thuần, Operation/Route thành kiểu mới
│       └── model_config.go     # [NEW] Đổi tên từ pricing.go, struct AIModelConfig, Model/Provider thành string thuần
├── application/
│   └── dto/
│       ├── cost.go             # [MODIFY] Loại bỏ các constant cũ, cập nhật Model/Provider sang string
│       └── model_config.go     # [NEW] Đổi tên từ pricing.go, DTO AIModelConfigDTO, Model/Provider sang string
database/
└── migrations/
    ├── 20260704062159_init_ai_executor_module.sql # [MODIFY] Chứa toàn bộ logic migration gộp
    ├── 20260705104500_make_ai_usage_events_ids_nullable.sql # [DELETE] Xoá bỏ
    └── 20260709034434_add_task_id_to_items.sql # [DELETE] Xoá bỏ
```

**Quyết định Cấu trúc**: Tuân thủ cấu trúc thư mục hiện tại của module `aiexecutor` và dọn dẹp lịch sử migration thừa.

## Theo dõi độ phức tạp (Complexity Tracking)

> *Chỉ điền nếu có vi phạm Hiến pháp cần giải trình. Hiện tại không có.*

| Vi phạm | Lý do cần thiết | Giải pháp thay thế đơn giản hơn bị loại bỏ vì |
|---------|-----------------|---------------------------------------------|
| Không   | N/A             | N/A                                         |
