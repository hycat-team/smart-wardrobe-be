# Đặc tả Tính năng: Chuẩn hoá các Hằng số Kỹ thuật của AI Executor

**Feature Branch**: `003-standardize-ai-constants`

**Created**: 2026-07-17

**Status**: Draft

**Input**: User description: "/speckit-specify chuẩn hoá lại các const sai về mặt kỹ thuật
- internal/modules/aiexecutor/application/dto/cost.go : những const nên tạo thành 2 loại và để vào internal/modules/aiexecutor/domain/constants, và cho entity dùng trực tiếp chứ không để string thuần
- internal/modules/aiexecutor/domain/constants/aimodel và internal/modules/aiexecutor/domain/constants/aiprovider : nên là string thuần chứ không nên là cố định do admin có thể chỉnh lại sau"

## User Scenarios & Testing *(mandatory)*

### Kịch bản Người dùng 1 - Cấu hình động AI Provider và AI Model qua Model Config (Độ ưu tiên: P1)

Với vai trò là Quản trị viên (Admin), tôi muốn hệ thống chấp nhận tên AI Provider và AI Model dưới dạng các chuỗi ký tự (string) tự do trong bảng Model Config, để tôi có thể đăng ký thêm model mới hoặc thay đổi cấu hình mặc định mà không cần chỉnh sửa code hay triển khai lại backend (ngoại trừ restart app để load lại cache/config nếu cần).

**Lý do ưu tiên**: Quan trọng cho vận hành. Ngăn chặn việc hardcode danh sách model, giúp hệ thống nhanh chóng cập nhật các model mới ra mắt (như Gemini 3.5, Gemini 4, v.v.) qua DB/admin config.

**Kiểm thử Độc lập**: Cấu hình tên model và provider có thể được thêm vào database trong bảng `ai_model_config`, và bộ định tuyến (orchestrator) vẫn hoạt động bình thường với các chuỗi cấu hình mới.

**Kịch bản Nghiệm thu**:

1. **Cho** cơ sở dữ liệu đã cấu hình một model mới dạng string `"gemini-3.5-pro"`, **Khi** có yêu cầu thực hiện AI operation, **Thì** orchestrator khởi tạo request thành công bằng việc sử dụng `"gemini-3.5-pro"` dưới dạng string.

---

### Kịch bản Người dùng 2 - Định nghĩa kiểu dữ liệu an toàn cho AI Operation và AI Route (Độ ưu tiên: P2)

Với vai trò là Lập trình viên, tôi muốn các AI Operation và AI Route được định nghĩa dưới dạng các hằng số có kiểu dữ liệu an toàn (type-safe constants) ở tầng domain, để các thực thể domain (entities) và usecase có thể sử dụng trực tiếp mà không cần dùng chuỗi string thuần dễ gõ sai.

**Lý do ưu tiên**: Giảm nợ kỹ thuật. Loại bỏ các bug phát sinh do gõ sai các giá trị chuỗi như `"chat"` so với `"Chat"`, hay `"paid"` so với `"Paid"`.

**Kiểm thử Độc lập**: Usecase và các hoạt động ở domain sẽ bị lỗi compile nếu truyền sai kiểu dữ liệu hoặc truyền sai giá trị route.

**Kịch bản Nghiệm thu**:

1. **Cho** thực thể cấu hình chi phí cần truyền một AI Route, **Khi** lập trình viên gán hằng số `airoute.Paid`, **Thì** chương trình biên dịch thành công.
2. **Cho** thực thể cấu hình chi phí cần truyền một AI Route, **Khi** lập trình viên gán chuỗi string thuần `"paid"`, **Thì** trình biên dịch báo lỗi và từ chối.

---

### Các Trường hợp Biên (Edge Cases)

- **Tương thích ngược dữ liệu cũ trong DB**: Các cột dữ liệu hiện tại trong DB lưu operation (ví dụ "chat", "outfit") hoặc route ("paid", "free"). Kiểu dữ liệu domain mới phải có khả năng serialize/deserialize tự động từ/sang chuỗi lưu trong DB mà không làm mất mát dữ liệu hoặc lỗi GORM.
- **Xử lý giá trị cấu hình sai**: Khi model/provider chuyển thành string thuần, nếu Admin cấu hình sai chính tả tên model trong DB, hệ thống cần phát hiện và trả về lỗi hợp lệ ở runtime thay vì crash ứng dụng.

## Requirements *(mandatory)*

### Yêu cầu Chức năng

- **FR-001**: Di chuyển các hằng số `AIOperation` (Chat, Outfit, Summary, Rewriter, Vision, Embedding) từ [cost.go](../../internal/modules/aiexecutor/application/dto/cost.go) vào một package domain riêng biệt là [aioperation](../../internal/modules/aiexecutor/domain/constants/aioperation/operation.go) dưới dạng kiểu dữ liệu tự định nghĩa `AIOperation string`.
- **FR-002**: Di chuyển các hằng số `AIRoute` (Paid, Free, Local) từ [cost.go](../../internal/modules/aiexecutor/application/dto/cost.go) vào một package domain riêng biệt là [airoute](../../internal/modules/aiexecutor/domain/constants/airoute/route.go) dưới dạng kiểu dữ liệu tự định nghĩa `AIRoute string`.
- **FR-003**: Các thực thể domain [AICostPolicyOperation](../../internal/modules/aiexecutor/domain/entities/policy.go) và [AIUsageEvent](../../internal/modules/aiexecutor/domain/entities/usage.go) PHẢI sử dụng trực tiếp kiểu dữ liệu `aioperation.AIOperation` và `airoute.AIRoute` thay vì dùng string thuần.
- **FR-004**: Hợp nhất cấu trúc bảng cấu hình AI từ `AIPricingConfig` (bảng `ai_pricing_configs`) thành `AIModelConfig` (bảng `ai_model_configs`).
- **FR-005**: Thay đổi kiểu dữ liệu của `Model` và `Provider` trong các entities (bao gồm `AIModelConfig`), repositories, use cases, mappers và config từ các kiểu enum cũ (`aimodel.Model`, `aiprovider.Provider`) thành kiểu `string` chuẩn của Go để hỗ trợ cấu hình động. Loại bỏ các file định nghĩa enum cũ nếu không còn được sử dụng.
- **FR-006**: Thực hiện gộp 3 tệp migration cũ (`20260704062159_init_ai_executor_module.sql`, `20260705104500_make_ai_usage_events_ids_nullable.sql`, `20260709034434_add_task_id_to_items.sql`) kết hợp với việc đặt tên bảng `ai_model_configs` thành một tệp migration duy nhất tại [20260704062159_init_ai_executor_module.sql](../../database/migrations/20260704062159_init_ai_executor_module.sql) để tối ưu lịch sử migration của Goose SQL.

## Success Criteria *(mandatory)*

### Tiêu chí Đo lường được

- **SC-001**: Không có bất kỳ lỗi biên dịch (compile error) nào liên quan đến kiểu dữ liệu constants trong module `aiexecutor` sau khi refactor.
- **SC-002**: Hệ thống build thành công (`go build ./...`) và toàn bộ test cases hiện tại chạy thành công (`go test ./...`).
- **SC-003**: Đảm bảo tương thích 100% với dữ liệu hiện tại trong DB. Tệp migration duy nhất được áp dụng thành công mà không gây xung đột cấu trúc.

## Assumptions

- Chúng ta giả định việc chuyển đổi `aimodel.Model` và `aiprovider.Provider` thành `string` sẽ không làm thay đổi định dạng JSON đầu ra khi trả về cho client.
- Gộp các tệp migration được thực hiện khi dọn dẹp các tệp migration chưa được đẩy/chạy trên môi trường production, đảm bảo an toàn cho lịch sử DB local/dev.

## Đánh giá Rủi ro (Risk Assessment)

1. **Tác động chỉnh sửa rộng (High Code Churn)**:
   - Các kiểu cũ như `aimodel.Model` và `aiprovider.Provider` xuất hiện ở rất nhiều nơi (Repository, Cache, Orchestrator usecase, Config). Việc chuyển đổi sang `string` sẽ làm lỗi compile hàng loạt file.
   - *Biện pháp giảm thiểu*: Tiến hành refactor từng bước, bắt đầu từ tầng domain, sau đó tới các interface repository, cơ chế cache, các cấu hình config, và cuối cùng là tầng application/usecase.

2. **Lỗi liên kết dữ liệu API (DTO Binding)**:
   - Các API nhận request có thể đang validate dựa trên các tag binding của Gin. Nếu thay đổi kiểu dữ liệu hoặc hằng số định nghĩa, cần kiểm tra lại các tag `binding` trên DTO.
   - *Biện pháp giảm thiểu*: Rà soát lại tất cả các struct DTO có trường `Model` hoặc `Provider` xem có sử dụng các tag ràng buộc enum cũ hay không.

3. **Mất an toàn Compile-time đối với Model/Provider**:
   - Khi chuyển sang `string`, lập trình viên có thể viết nhầm tên model mà trình biên dịch không phát hiện ra.
   - *Biện pháp giảm thiểu*: Bổ sung kiểm tra cấu hình tại thời điểm chạy (runtime validation) hoặc thêm unit test đảm bảo các model được cấu hình đúng.
