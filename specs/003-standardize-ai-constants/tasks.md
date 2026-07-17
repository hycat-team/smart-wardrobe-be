# Danh sách Công việc (Tasks): Chuẩn hoá các Hằng số Kỹ thuật của AI Executor

**Đầu vào**: Các tài liệu thiết kế từ `/specs/003-standardize-ai-constants/`

**Điều kiện tiên quyết**: plan.md (bắt buộc), spec.md (bắt buộc), research.md, data-model.md

**Tổ chức công việc**: Các công việc được nhóm theo từng kịch bản người dùng (User Story) để đảm bảo việc triển khai và kiểm thử diễn ra độc lập.

## Định dạng: `- [ ] [ID] [P?] [Story] Mô tả công việc với đường dẫn file cụ thể`

- **[P]**: Có thể chạy song song (các file khác nhau, không có quan hệ phụ thuộc lẫn nhau).
- **[Story]**: Kịch bản người dùng tương ứng (ví dụ: US1, US2).

---

## Giai đoạn 1: Thiết lập (Setup - Hạ tầng dùng chung)

**Mục đích**: Khởi tạo cấu trúc folder/file mới và chuẩn bị file migration cơ sở dữ liệu.

- [x] T001 [P] Khởi tạo folder và file định nghĩa kiểu dữ liệu `AIOperation` tại [operation.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aioperation/operation.go)
- [x] T002 [P] Khởi tạo folder và file định nghĩa kiểu dữ liệu `AIRoute` tại [route.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/airoute/route.go)
- [x] T003 [P] Hợp nhất toàn bộ logic vào tệp migration duy nhất tại [20260704062159_init_ai_executor_module.sql](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/database/migrations/20260704062159_init_ai_executor_module.sql), đồng thời xoá bỏ 2 tệp cũ `20260705104500_make_ai_usage_events_ids_nullable.sql` và `20260709034434_add_task_id_to_items.sql`.


---

## Giai đoạn 2: Nền tảng (Foundational - Chuyển đổi thực thể)

**Mục đích**: Cập nhật các thực thể (entities) cốt lõi của domain để chuyển sang kiểu dữ liệu mới.

- [x] T004 Xóa bỏ hoặc dọn dẹp các tệp định nghĩa enum model cũ tại [model.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel/model.go) và [provider.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider/provider.go)
- [x] T005 Đổi tên file [pricing.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/entities/pricing.go) thành `model_config.go` và định nghĩa struct `AIModelConfig` sử dụng kiểu dữ liệu `string` cho Model/Provider.


**Cột mốc**: Hoàn thành tái cấu trúc tầng Domain Entities.

---

## Giai đoạn 3: Kịch bản Người dùng 1 - Cấu hình động AI Model & Provider (US1) 🎯 MVP

**Mục tiêu**: Chuyển đổi toàn bộ logic xử lý AI Model và AI Provider từ kiểu enum sang string thuần để hỗ trợ cấu hình động.

**Kiểm thử Độc lập**: Build ứng dụng thành công và các truy vấn cấu hình model qua DB/Cache trả về kiểu string.

### Triển khai cho US1

- [x] T006 [P] [US1] Cập nhật các DTO trong [cost.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/dto/cost.go) để sử dụng `string` cho Model và Provider.
- [x] T007 [P] [US1] Đổi tên `pricing.go` ở tầng DTO thành [model_config.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/dto/model_config.go) và định nghĩa struct `AIModelConfigDTO` dùng kiểu string.
- [x] T008 [US1] Cập nhật kiểu tham số của interface Repository [pricing_repo.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories/pricing_repo.go) và file implement [pricing_repo_impl.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/infrastructure/persistence/pricing_repo_impl.go) sang kiểu string.
- [x] T009 [US1] Cập nhật interface/implement của Redis pricing cache sang kiểu string tại [pricing_cache.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/interface/caching/pricing_cache.go) và [redis_pricing_cache.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/infrastructure/caching/redis_pricing_cache.go).
- [x] T010 [US1] Cập nhật usecase [pricing/usecase.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/pricing/usecase.go) và các interface tương ứng để dùng kiểu string.


---

## Giai đoạn 4: Kịch bản Người dùng 2 - Kiểu dữ liệu an toàn cho AI Operation & Route (US2)

**Mục tiêu**: Đóng gói các hằng số operation/route và cập nhật toàn bộ logic kiểm tra chuỗi literal sang kiểu type-safe domain constants.

**Kiểm thử Độc lập**: Trình biên dịch báo lỗi nếu truyền chuỗi string thuần vào các hàm yêu cầu `AIOperation` hoặc `AIRoute`.

### Triển khai cho US2

- [x] T011 [US2] Cập nhật thực thể [policy.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/entities/policy.go) sử dụng `aioperation.AIOperation` và `airoute.AIRoute`.
- [x] T012 [US2] Cập nhật thực thể [usage.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/entities/usage.go) sử dụng `aioperation.AIOperation`, `airoute.AIRoute` và trường Model/Provider dạng pointer string.
- [x] T013 [US2] Cập nhật logic Orchestrator resolve model/provider tại [plan.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/orchestrator/plan.go) sử dụng các constant và kiểu dữ liệu mới.
- [x] T014 [US2] Cập nhật logic kiểm tra hạn mức và định tuyến trong [cost/usecase.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/cost/usecase.go) sử dụng kiểu dữ liệu mới.
- [x] T015 [US2] Cập nhật các câu lệnh SQL/GORM trong [ai_cost_repo_impl.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/infrastructure/persistence/ai_cost_repo_impl.go) để sử dụng hằng số `airoute.Paid` thay cho chuỗi string thuần `"paid"`.


---

## Giai đoạn 5: Hoàn thiện & Kiểm thử (Polish)

**Mục đích**: Chạy kiểm thử tích hợp, tối ưu hóa code và chạy xác thực.

- [x] T016 Thực thi chạy migration trên môi trường local.
- [x] T017 Thực hiện biên dịch dự án (`go build ./...`) để phát hiện lỗi compile sót.
- [x] T018 Chạy toàn bộ unit test (`go test ./...`) để đảm bảo không phát sinh lỗi logic.
- [x] T019 Xác thực kịch bản bằng tay theo [quickstart.md](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/specs/003-standardize-ai-constants/quickstart.md).


---

## Trình tự và Sự phụ thuộc giữa các Giai đoạn

1. **Setup & Foundational (Giai đoạn 1 & 2)**: Phải chạy trước tiên để có đủ các kiểu dữ liệu và thực thể mới làm nền tảng.
2. **User Story 1 (Giai đoạn 3)**: Có thể triển khai tiếp theo để hoàn thiện tính năng động hóa model & provider.
3. **User Story 2 (Giai đoạn 4)**: Cần thực hiện sau hoặc song song với US1 để hoàn tất chuẩn hoá các hằng số.
4. **Polish (Giai đoạn 5)**: Chạy cuối cùng sau khi hoàn tất toàn bộ code để đảm bảo chất lượng hệ thống.

---

## Chiến lược Triển khai

1. Triển khai hoàn tất Giai đoạn 1 và 2.
2. Cập nhật các phần liên quan đến Model/Provider sang string (Giai đoạn 3).
3. Đóng gói và cập nhật route/operation sang kiểu domain mới (Giai đoạn 4).
4. Viết file migration đổi tên bảng, chạy test suite để xác thực.
