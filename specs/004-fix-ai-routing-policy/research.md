# Research: AI Executor Routing Policy & Fallback Implementation

Tài liệu này nghiên cứu và làm rõ cách giải quyết nợ kỹ thuật định tuyến động AI dựa trên kết quả phân giải `AIRoute` thực tế.

## 1. Cơ chế hoạt động của `Prepare` và `Orchestrator`

Hiện tại, quá trình diễn ra theo các bước sau:

1. **Giai đoạn chuẩn bị (Tentative Resolution)**:
   - Orchestrator gọi `resolveExecutionPlan` để tạo ra một kế hoạch thực thi tạm tính (`plan`). Kế hoạch này mặc định lấy model Premium (`paid`) tương ứng với tính năng và chính sách `op.NormalRoute` hoặc `op.FreeRoute`.
2. **Kiểm tra và đặt trước chi phí (Cost Reservation)**:
   - Orchestrator gọi `CostUseCase.Prepare(plan)` để kiểm tra số dư và thực hiện đặt trước chi phí (`Reserve` trong DB).
   - Nếu số dư khả dụng và thỏa mãn chính sách, hệ thống giữ nguyên định tuyến `"paid"`.
   - Nếu vượt quá hạn mức hoặc không đủ tiền, `Prepare` tự động chuyển đổi định tuyến trả về thành `"free"` (thông qua `PrepareFree`).
   - Nếu có lỗi hệ thống hoặc bị chặn hoàn toàn, hệ thống trả về `"local"`.
3. **Thực thi định tuyến động**:
   - **Với định tuyến `"paid"`**: Giữ nguyên model Premium ban đầu và gọi API AI.
   - **Với định tuyến `"free"`**: Hệ thống chuyển đổi thông số gọi model sang mô hình miễn phí/giá rẻ được cấu hình qua biến `FREE_TEXT_PRIMARY_PROVIDER` và `FREE_TEXT_PRIMARY_MODEL` (ví dụ: `gemini-2.5-flash-lite`).
   - **Với định tuyến `"local"`**: Bỏ qua toàn bộ cuộc gọi API AI, hoàn trả ngay lập tức lỗi định danh (ví dụ: `ErrAILocalRouteFallback`) để lớp ứng dụng (như module `fashion`) tự động chạy thuật toán fallback cục bộ bằng code Go thuần (như `RunLocalHSLMatching` hoặc `LocalRewriter`).

---

## 2. Giải pháp Ánh xạ Model cho Định tuyến Free

Thay vì sửa đổi cấu trúc Config phức tạp, chúng ta tận dụng cấu hình `FreeTextPrimary` đã có sẵn trong `config.Config` (được nạp từ các biến môi trường `FREE_TEXT_PRIMARY_...`):

- **Paid**: `cfg.AI.ChatTextPrimary` (cho Chat) / `cfg.AI.RecommendationTextPrimary` (cho Outfit)
- **Free**: `cfg.AI.FreeTextPrimary` (áp dụng chung cho mọi tác vụ được chuyển sang route miễn phí)

Khi `decision.Route == airoute.Free`, Orchestrator sẽ cập nhật lại thông tin `Provider` và `Model` của `plan` bằng `cfg.AI.FreeTextPrimary.Provider` và `cfg.AI.FreeTextPrimary.Model` trước khi gọi xuống `aiService`.

---

## 3. Quản lý luồng Local Fallback

Khi `decision.Route == airoute.Local`:
- Trả về lỗi `errors.New("ai route resolved to local fallback: bypassing remote api call")` (hoặc định nghĩa mã lỗi trong `apperror`).
- Luồng nghiệp vụ phía trên (ví dụ: `fashion/application/usecase/ai/recommendation/workflow.go`) khi nhận được lỗi từ AI Service sẽ tự động chuyển tiếp sang thuật toán phối màu HSL cục bộ.
