# Quickstart: Hướng dẫn xác thực định tuyến động AI Routing Policy

Tài liệu này hướng dẫn cách chạy thử nghiệm và xác thực tính năng định tuyến động kết hợp fallback cục bộ.

## 1. Chuẩn bị môi trường (Prerequisites)

1. Khởi động cơ sở dữ liệu và Redis:
   ```bash
   make docker-dev-up
   ```
2. Thực thi migration mới nhất:
   ```bash
   make migration-up
   ```

## 2. Kịch bản kiểm thử (Validation Scenarios)

### Kịch bản 1: Định tuyến Paid (Mặc định khi còn hạn mức)
1. **Thiết lập**: Cấp quyền hạn mức sử dụng AI cho người dùng với số dư ví dồi dào.
2. **Thực thi**: Gửi yêu cầu Chat hoặc Gợi ý trang phục.
3. **Kết quả kỳ vọng**: 
   - Ứng dụng thực hiện gọi mô hình Premium (`gemini-3.1-flash`).
   - Log ghi nhận cuộc gọi với route là `"paid"`.
   - Một bản ghi `ai_usage_events` được tạo ra trong DB lưu `logical_route = 'paid'` và liên kết với config ID của mô hình Premium.

### Kịch bản 2: Định tuyến Free (Khi vượt quá hạn mức thanh toán nhưng còn hạn mức phụ)
1. **Thiết lập**: Điều chỉnh số dư ví của người dùng về mức gần bằng 0 hoặc điều chỉnh chính sách của User về `free_only`.
2. **Thực thi**: Gửi yêu cầu Chat hoặc Gợi ý trang phục.
3. **Kết quả kỳ vọng**:
   - Hệ thống tự động chuyển hướng gọi mô hình Free (`gemma-4-31b-it` hoặc `gemini-2.5-flash-lite` tuỳ cấu hình env).
   - Log ghi nhận dòng chữ `"attempting free fallback"`.
   - Bản ghi trong DB ghi nhận `logical_route = 'free'` và liên kết với config ID của mô hình Free.

### Kịch bản 3: Định tuyến Local Fallback (Khi hết toàn bộ hạn mức hoặc offline)
1. **Thiết lập**: Chặn hoàn toàn quyền truy cập AI (ví dụ: đặt hạn mức tối đa = 0 hoặc khóa trạng thái Grant).
2. **Thực thi**: Gửi yêu cầu gợi ý trang phục.
3. **Kết quả kỳ vọng**:
   - Orchestrator trả về lỗi fallback ngay lập tức mà không gọi API ngoại vi.
   - Hệ thống tự động chuyển sang chạy thuật toán phối màu HSL cục bộ [RunLocalHSLMatching].
   - Kết quả trả về cho client có cờ `IsFallback: true`.
