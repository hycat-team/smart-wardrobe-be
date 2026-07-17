# Kết quả Nghiên cứu: Chuẩn hoá các Hằng số Kỹ thuật của AI Executor

## Quyết định Kỹ thuật

### 1. Loại bỏ kiểu custom `aimodel.Model` và `aiprovider.Provider`
- **Quyết định**: Xóa bỏ hoàn toàn các package `internal/modules/aiexecutor/domain/constants/aimodel` và `internal/modules/aiexecutor/domain/constants/aiprovider`. Tất cả các khai báo sử dụng kiểu dữ liệu này sẽ được chuyển thành `string` thuần của Go.
- **Lý do lựa chọn**: Danh sách các AI model và AI provider thay đổi liên tục theo thời gian thực (ví dụ Google ra mắt các dòng Gemini mới, hoặc tích hợp thêm OpenAI, Anthropic...). Việc cứng hóa các giá trị này dưới dạng enum trong code Go khiến việc mở rộng bắt buộc phải sửa code và deploy lại. Dùng string thuần giúp Admin dễ dàng cập nhật qua cơ sở dữ liệu.
- **Các giải pháp thay thế đã bị loại bỏ**:
  - *Giải pháp giữ nguyên kiểu tự định nghĩa nhưng cho phép gán chuỗi*: Không giải quyết được triệt để nợ kỹ thuật và gây nhầm lẫn khi biên dịch vì hằng số vẫn được định nghĩa cố định trong code.

### 2. Tạo các kiểu dữ liệu Domain mới `aioperation.AIOperation` và `airoute.AIRoute`
- **Quyết định**: Tạo 2 package mới tại `internal/modules/aiexecutor/domain/constants/aioperation` và `internal/modules/aiexecutor/domain/constants/airoute`. 
  - Kiểu `AIOperation` (kiểu gốc `string`) lưu trữ các hoạt động AI: `chat`, `outfit`, `summary`, `rewriter`, `vision`, `embedding`.
  - Kiểu `AIRoute` (kiểu gốc `string`) lưu trữ các tuyến định tuyến AI: `paid`, `free`, `local`.
- **Lý do lựa chọn**: Khác với Model và Provider (vốn động và do Admin kiểm soát), AI Operation và AI Route là các thực thể nghiệp vụ cốt lõi của hệ thống (hệ thống có các luồng logic riêng biệt xử lý cho từng operation/route cụ thể). Do đó, chúng cần được kiểm soát chặt chẽ ở mức biên dịch (type-safe) để tránh lập trình viên gõ sai chính tả chuỗi (ví dụ `"Chat"` thay vì `"chat"`).

## Đánh giá Tương thích ngược

- **Cơ sở dữ liệu (PostgreSQL)**: GORM tự động map các kiểu dữ liệu Go tự định nghĩa có kiểu gốc là `string` sang kiểu dữ liệu `varchar`/`text` của Postgres. Do đó, không cần chạy migration thay đổi schema database.
- **Giao tiếp API (JSON)**: Định dạng JSON của `aioperation.AIOperation` và `airoute.AIRoute` khi serialize/deserialize vẫn là dạng string thuần, đảm bảo API không bị thay đổi định dạng đối với client.
