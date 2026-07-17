# Data Model: AI Executor Routing Policy

Tính năng này không yêu cầu thay đổi lược đồ cơ sở dữ liệu (Database Schema) mới. Hệ thống tái sử dụng cấu trúc bảng hiện tại đã được chuẩn hoá trong Migration [20260704062159_init_ai_executor_module.sql](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/database/migrations/20260704062159_init_ai_executor_module.sql).

## Các thực thể liên quan (Key Entities)

### 1. `ai_model_configs` (AIModelConfig)
Lưu cấu hình chi phí và model của nhà cung cấp. Khi định tuyến động sang `free`, hệ thống sẽ truy vấn cấu hình active của model free tương ứng.

### 2. `ai_policy_grants` (AIPolicyGrant) và `ai_cost_policies` (AICostPolicy)
Dùng để xác định hạn mức, chính sách định tuyến gốc (`op.NormalRoute`, `op.FreeRoute`).

### 3. `ai_usage_events` (AIUsageEvent)
Ghi nhận lịch sử cuộc gọi AI. Trường `logical_route` lưu trữ route thực tế (`paid`, `free`, `local`) được áp dụng cho request.
- Nếu định tuyến là `free` $\rightarrow$ ghi nhận cuộc gọi với đơn giá của model free.
- Nếu định tuyến là `local` $\rightarrow$ ghi nhận trạng thái cục bộ.
